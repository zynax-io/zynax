// SPDX-License-Identifier: Apache-2.0

package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/zynax-io/zynax/agents/adapters/ci/internal/config"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultBaseURL  = "https://api.github.com"
	maxBodyBytes    = 10 * 1024 * 1024 // 10 MB response body cap
	maxErrMsgLen    = 512
	githubAPIAccept = "application/vnd.github+json"
)

// executeStream is the minimal interface used by the handler (enables testing with mocks).
type executeStream interface {
	Send(*zynaxv1.TaskEvent) error
	Context() context.Context
}

// ciHandler performs GitHub Actions REST API calls for all capabilities.
// It is stateless — one instance is shared across all requests.
type ciHandler struct {
	token   string
	cfg     *config.CIConfig
	baseURL string
	client  *http.Client
}

func newCIHandler(token string, cfg *config.CIConfig) *ciHandler {
	return &ciHandler{
		token:   token,
		cfg:     cfg,
		baseURL: defaultBaseURL,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

// newCIHandlerWithURL creates a handler pointed at a custom base URL (for tests).
func newCIHandlerWithURL(token string, cfg *config.CIConfig, baseURL string) *ciHandler {
	h := newCIHandler(token, cfg)
	if baseURL != "" {
		h.baseURL = strings.TrimRight(baseURL, "/")
	}
	return h
}

// execute routes capability invocations by name.
func (h *ciHandler) execute(
	ctx context.Context,
	ccap config.CICapabilityConfig,
	taskID string,
	payload []byte,
	stream executeStream,
) error {
	if h.cfg.Provider == "jenkins-stub" {
		return sendFailed(stream, taskID, "INTERNAL", "not implemented: provider jenkins not supported in M5")
	}

	switch ccap.Name {
	case "trigger_workflow":
		return h.triggerWorkflow(ctx, ccap, taskID, payload, stream)
	case "get_run_status":
		return h.getRunStatus(ctx, ccap, taskID, payload, stream)
	default:
		return sendFailed(stream, taskID, "INVALID_INPUT",
			fmt.Sprintf("unknown capability: %s", ccap.Name))
	}
}

// ── trigger_workflow ──────────────────────────────────────────────────────────

type triggerWorkflowInput struct {
	Ref    string                 `json:"ref"`
	Inputs map[string]interface{} `json:"inputs,omitempty"`
}

// triggerWorkflow dispatches a workflow_dispatch event then polls the runs list for the new run ID.
func (h *ciHandler) triggerWorkflow(
	ctx context.Context,
	ccap config.CICapabilityConfig,
	taskID string,
	payload []byte,
	stream executeStream,
) error {
	var inp triggerWorkflowInput
	if err := parsePayload(payload, &inp); err != nil {
		return sendFailed(stream, taskID, "INVALID_INPUT", sanitise(err.Error()))
	}
	if inp.Ref == "" {
		return sendFailed(stream, taskID, "INVALID_INPUT", "ref is required")
	}

	triggerTime := time.Now().UTC()

	// POST workflow_dispatch event — returns 204 No Content on success.
	dispatchURL := fmt.Sprintf("%s/repos/%s/%s/actions/workflows/%s/dispatches",
		h.baseURL, ccap.Owner, ccap.Repo, ccap.WorkflowID)

	body, err := json.Marshal(map[string]interface{}{"ref": inp.Ref, "inputs": inp.Inputs})
	if err != nil {
		return sendFailed(stream, taskID, "INTERNAL", "failed to marshal dispatch request")
	}

	code, _, apiErr := h.doRequest(ctx, http.MethodPost, dispatchURL, body)
	if apiErr != nil {
		return sendFailed(stream, taskID, ciErrCode(code), sanitise(apiErr.Error()))
	}

	// Poll the runs list until the new run ID appears or the trigger timeout expires.
	triggerTimeout := time.Duration(h.cfg.TriggerPollTimeoutSeconds) * time.Second
	pollCtx, cancel := context.WithTimeout(ctx, triggerTimeout)
	defer cancel()

	runsURL := fmt.Sprintf("%s/repos/%s/%s/actions/runs?event=workflow_dispatch&branch=%s",
		h.baseURL, ccap.Owner, ccap.Repo, inp.Ref)

	for {
		select {
		case <-pollCtx.Done():
			return sendFailed(stream, taskID, "TIMEOUT",
				"trigger_workflow: run ID did not appear within trigger_poll_timeout_seconds")
		case <-time.After(time.Duration(h.cfg.PollIntervalSeconds) * time.Second):
		}

		runID, runURL, found, statusCode, reqErr := h.findNewRun(ctx, runsURL, triggerTime)
		if reqErr != nil {
			return sendFailed(stream, taskID, ciErrCode(statusCode), sanitise(reqErr.Error()))
		}
		if found {
			out, _ := json.Marshal(map[string]interface{}{
				"run_id":  runID,
				"run_url": runURL,
			})
			return stream.Send(completedEvent(taskID, out)) //nolint:wrapcheck
		}
		slog.Debug("trigger_workflow: waiting for run ID", "task_id", taskID)
	}
}

// findNewRun calls the runs list API and returns the first run created after triggerTime.
func (h *ciHandler) findNewRun(ctx context.Context, runsURL string, triggerTime time.Time) (int64, string, bool, int, error) {
	statusCode, data, err := h.doRequest(ctx, http.MethodGet, runsURL, nil)
	if err != nil {
		return 0, "", false, statusCode, err
	}

	var resp struct {
		WorkflowRuns []struct {
			ID        int64  `json:"id"`
			HTMLURL   string `json:"html_url"`
			CreatedAt string `json:"created_at"`
		} `json:"workflow_runs"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return 0, "", false, 0, fmt.Errorf("parse runs list: %w", err)
	}

	for _, run := range resp.WorkflowRuns {
		t, err := time.Parse(time.RFC3339, run.CreatedAt)
		if err != nil {
			continue
		}
		if t.After(triggerTime) || t.Equal(triggerTime) {
			return run.ID, run.HTMLURL, true, 0, nil
		}
	}
	return 0, "", false, 0, nil
}

// ── get_run_status ────────────────────────────────────────────────────────────

type getRunStatusInput struct {
	RunID int64 `json:"run_id"`
}

// terminalStatuses are GitHub Actions run statuses that indicate the run is done.
var terminalConclusions = map[string]bool{
	"success":         true,
	"failure":         true,
	"cancelled":       true,
	"skipped":         true,
	"timed_out":       true,
	"action_required": true,
	"neutral":         true,
	"stale":           true,
}

// getRunStatus polls the GitHub Actions run until it reaches a terminal state.
func (h *ciHandler) getRunStatus(
	ctx context.Context,
	ccap config.CICapabilityConfig,
	taskID string,
	payload []byte,
	stream executeStream,
) error {
	var inp getRunStatusInput
	if err := parsePayload(payload, &inp); err != nil {
		return sendFailed(stream, taskID, "INVALID_INPUT", sanitise(err.Error()))
	}
	if inp.RunID <= 0 {
		return sendFailed(stream, taskID, "INVALID_INPUT", "run_id must be a positive integer")
	}

	runURL := fmt.Sprintf("https://github.com/%s/%s/actions/runs/%d",
		ccap.Owner, ccap.Repo, inp.RunID)
	apiURL := fmt.Sprintf("%s/repos/%s/%s/actions/runs/%d",
		h.baseURL, ccap.Owner, ccap.Repo, inp.RunID)

	return h.pollLoop(ctx, taskID, runURL, apiURL, stream)
}

// pollLoop implements the exponential-backoff polling loop for run status.
// It emits PROGRESS events per cycle and COMPLETED/FAILED on terminal state or timeout.
func (h *ciHandler) pollLoop(
	ctx context.Context,
	taskID string,
	runURL string,
	apiURL string,
	stream executeStream,
) error {
	interval := time.Duration(h.cfg.PollIntervalSeconds) * time.Second
	maxInterval := time.Duration(h.cfg.MaxPollIntervalSeconds) * time.Second

	for {
		select {
		case <-ctx.Done():
			return sendFailed(stream, taskID, "TIMEOUT", "get_run_status: request exceeded timeout_seconds")
		case <-time.After(interval):
		}

		statusCode, data, err := h.doRequest(ctx, http.MethodGet, apiURL, nil)
		if err != nil {
			return sendFailed(stream, taskID, ciErrCode(statusCode), sanitise(err.Error()))
		}

		var run struct {
			Status     string `json:"status"`
			Conclusion string `json:"conclusion"`
		}
		if err := json.Unmarshal(data, &run); err != nil {
			return sendFailed(stream, taskID, "UPSTREAM_ERROR", "failed to parse run status response")
		}

		progPayload, _ := json.Marshal(map[string]string{
			"run_url": runURL,
			"status":  run.Status,
		})

		if run.Status == "completed" {
			out, _ := json.Marshal(map[string]string{
				"run_url":    runURL,
				"status":     run.Status,
				"conclusion": run.Conclusion,
			})
			return stream.Send(completedEvent(taskID, out)) //nolint:wrapcheck
		}

		// Also treat known terminal conclusions as completed (defensive: some API versions may set conclusion early).
		if terminalConclusions[run.Conclusion] {
			out, _ := json.Marshal(map[string]string{
				"run_url":    runURL,
				"status":     run.Status,
				"conclusion": run.Conclusion,
			})
			return stream.Send(completedEvent(taskID, out)) //nolint:wrapcheck
		}

		if err := stream.Send(progressEventWithPayload(taskID, progPayload)); err != nil {
			return err //nolint:wrapcheck
		}

		// Exponential backoff: double interval, cap at max.
		interval *= 2
		if interval > maxInterval {
			interval = maxInterval
		}
	}
}

// ── HTTP helpers ──────────────────────────────────────────────────────────────

// doRequest performs an authenticated GitHub API call and returns the HTTP status
// code and response body. Only the status code is returned on error so callers can
// map it to a CapabilityError code.
func (h *ciHandler) doRequest(ctx context.Context, method, url string, body []byte) (int, []byte, error) {
	var reqBody io.Reader
	if body != nil {
		reqBody = strings.NewReader(string(body))
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return 0, nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+h.token)
	req.Header.Set("Accept", githubAPIAccept)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("http request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	limited := io.LimitReader(resp.Body, maxBodyBytes)
	data, err := io.ReadAll(limited)
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return resp.StatusCode, nil, fmt.Errorf("GitHub API error: HTTP %d", resp.StatusCode)
	}

	return resp.StatusCode, data, nil
}

// ciErrCode maps GitHub API HTTP status codes to CapabilityError codes.
func ciErrCode(statusCode int) string {
	switch statusCode {
	case http.StatusTooManyRequests, http.StatusForbidden:
		return "RESOURCE_EXHAUSTED"
	case http.StatusNotFound:
		return "NOT_FOUND"
	case http.StatusUnprocessableEntity:
		return "INVALID_INPUT"
	}
	if statusCode == 0 {
		return "UPSTREAM_ERROR"
	}
	return "UPSTREAM_ERROR"
}

// ── event helpers ─────────────────────────────────────────────────────────────

func progressEventWithPayload(taskID string, payload []byte) *zynaxv1.TaskEvent {
	return &zynaxv1.TaskEvent{
		TaskId:    taskID,
		EventType: zynaxv1.TaskEventType_TASK_EVENT_TYPE_PROGRESS,
		Payload:   payload,
		Timestamp: timestamppb.Now(),
	}
}

func completedEvent(taskID string, payload []byte) *zynaxv1.TaskEvent {
	return &zynaxv1.TaskEvent{
		TaskId:    taskID,
		EventType: zynaxv1.TaskEventType_TASK_EVENT_TYPE_COMPLETED,
		Payload:   payload,
		Timestamp: timestamppb.Now(),
	}
}

func sendFailed(stream executeStream, taskID, code, msg string) error {
	return stream.Send(&zynaxv1.TaskEvent{ //nolint:wrapcheck
		TaskId:    taskID,
		EventType: zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED,
		Timestamp: timestamppb.Now(),
		Error:     &zynaxv1.CapabilityError{Code: code, Message: msg},
	})
}

func sanitise(s string) string {
	if len(s) > maxErrMsgLen {
		return s[:maxErrMsgLen]
	}
	return s
}

func parsePayload(payload []byte, v interface{}) error {
	if len(payload) == 0 {
		return fmt.Errorf("input payload is required")
	}
	if err := json.Unmarshal(payload, v); err != nil {
		return fmt.Errorf("invalid JSON payload: %w", err)
	}
	return nil
}
