// SPDX-License-Identifier: Apache-2.0

package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/go-github/v67/github"
	"github.com/zynax-io/zynax/agents/adapters/git/internal/config"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	maxDiffBytes  = 4 * 1024 * 1024 // 4 MB cap on diff response
	maxErrMsgLen  = 512
	tickInterval  = 2 * time.Second
	pollInterval  = 3 * time.Second
	maxPollCycles = 20
)

// executeStream is the minimal interface used by the handler (enables testing with mocks).
type executeStream interface {
	Send(*zynaxv1.TaskEvent) error
	Context() context.Context
}

type gitHandler struct {
	gh *github.Client
}

func newGitHandler(token string) *gitHandler {
	return &gitHandler{gh: newGitHubClient(token)}
}

// newGitHandlerWithURL creates a handler pointed at a custom base URL (for tests).
func newGitHandlerWithURL(token, baseURL string) *gitHandler {
	if baseURL == "" {
		return newGitHandler(token)
	}
	client := newGitHubClient(token)
	parsed, err := client.BaseURL.Parse(baseURL + "/")
	if err != nil {
		// Fall back to default if URL is malformed.
		return newGitHandler(token)
	}
	client.BaseURL = parsed
	return &gitHandler{gh: client}
}

func (h *gitHandler) execute(
	ctx context.Context,
	gcap config.GitCapabilityConfig,
	taskID string,
	payload []byte,
	stream executeStream,
) error {
	switch gcap.Name {
	case "open_pr":
		return h.openPR(ctx, gcap, taskID, payload, stream)
	case "request_review":
		return h.requestReview(ctx, gcap, taskID, payload, stream)
	case "get_diff":
		return h.getDiff(ctx, gcap, taskID, payload, stream)
	default:
		// Provider-specific capabilities added in future milestones.
		if gcap.Name == "gitlab" {
			return sendFailed(stream, taskID, "INTERNAL", "not implemented: provider gitlab not supported in M5")
		}
		return sendFailed(stream, taskID, "INVALID_INPUT", fmt.Sprintf("unknown capability: %s", gcap.Name))
	}
}

// ── open_pr ──────────────────────────────────────────────────────────────────

type openPRInput struct {
	Title string `json:"title"`
	Head  string `json:"head"`
	Base  string `json:"base"`
	Body  string `json:"body,omitempty"`
}

func (h *gitHandler) openPR(
	ctx context.Context,
	gcap config.GitCapabilityConfig,
	taskID string,
	payload []byte,
	stream executeStream,
) error {
	var inp openPRInput
	if err := parsePayload(payload, &inp); err != nil {
		return sendFailed(stream, taskID, "INVALID_INPUT", sanitise(err.Error()))
	}
	if inp.Title == "" || inp.Head == "" || inp.Base == "" {
		return sendFailed(stream, taskID, "INVALID_INPUT", "title, head, and base are required")
	}

	ch := make(chan prResult, 1)
	go func() {
		pr, _, err := h.gh.PullRequests.Create(ctx, gcap.Owner, gcap.Repo, &github.NewPullRequest{
			Title: github.String(inp.Title),
			Head:  github.String(inp.Head),
			Base:  github.String(inp.Base),
			Body:  github.String(inp.Body),
		})
		ch <- prResult{pr: pr, err: err}
	}()

	tick := time.NewTicker(tickInterval)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			if err := stream.Send(progressEvent(taskID)); err != nil {
				return err //nolint:wrapcheck
			}
		case res := <-ch:
			if res.err != nil {
				return sendFailed(stream, taskID, githubErrCode(res.err), sanitise(res.err.Error()))
			}
			out, err := marshalPayload(map[string]interface{}{
				"pr_url":    res.pr.GetHTMLURL(),
				"pr_number": res.pr.GetNumber(),
			})
			if err != nil {
				return sendFailed(stream, taskID, "UPSTREAM_ERROR", "failed to marshal response")
			}
			return stream.Send(completedEvent(taskID, out)) //nolint:wrapcheck
		case <-ctx.Done():
			return sendFailed(stream, taskID, "TIMEOUT", "request exceeded timeout_seconds")
		}
	}
}

type prResult struct {
	pr  *github.PullRequest
	err error
}

// ── request_review ────────────────────────────────────────────────────────────

type requestReviewInput struct {
	PRNumber  int      `json:"pr_number"`
	Reviewers []string `json:"reviewers"`
}

func (h *gitHandler) requestReview(
	ctx context.Context,
	gcap config.GitCapabilityConfig,
	taskID string,
	payload []byte,
	stream executeStream,
) error {
	var inp requestReviewInput
	if err := parsePayload(payload, &inp); err != nil {
		return sendFailed(stream, taskID, "INVALID_INPUT", sanitise(err.Error()))
	}
	if inp.PRNumber <= 0 {
		return sendFailed(stream, taskID, "INVALID_INPUT", "pr_number must be a positive integer")
	}
	if len(inp.Reviewers) == 0 {
		return sendFailed(stream, taskID, "INVALID_INPUT", "reviewers must be non-empty")
	}

	_, _, err := h.gh.PullRequests.RequestReviewers(ctx, gcap.Owner, gcap.Repo, inp.PRNumber,
		github.ReviewersRequest{Reviewers: inp.Reviewers})
	if err != nil {
		return sendFailed(stream, taskID, githubErrCode(err), sanitise(err.Error()))
	}

	// Poll for reviewer assignment confirmation (up to maxPollCycles).
	for range maxPollCycles {
		if err := stream.Send(progressEvent(taskID)); err != nil {
			return err //nolint:wrapcheck
		}
		select {
		case <-ctx.Done():
			return sendFailed(stream, taskID, "TIMEOUT", "request exceeded timeout_seconds")
		case <-time.After(pollInterval):
		}

		pr, _, err := h.gh.PullRequests.Get(ctx, gcap.Owner, gcap.Repo, inp.PRNumber)
		if err != nil {
			return sendFailed(stream, taskID, githubErrCode(err), sanitise(err.Error()))
		}
		for _, r := range pr.RequestedReviewers {
			for _, want := range inp.Reviewers {
				if r.GetLogin() == want {
					out, _ := marshalPayload(map[string]bool{"requested": true})
					return stream.Send(completedEvent(taskID, out)) //nolint:wrapcheck
				}
			}
		}
	}

	out, _ := marshalPayload(map[string]bool{"requested": true})
	return stream.Send(completedEvent(taskID, out)) //nolint:wrapcheck
}

// ── get_diff ──────────────────────────────────────────────────────────────────

type getDiffInput struct {
	PRNumber int `json:"pr_number"`
}

func (h *gitHandler) getDiff(
	ctx context.Context,
	gcap config.GitCapabilityConfig,
	taskID string,
	payload []byte,
	stream executeStream,
) error {
	var inp getDiffInput
	if err := parsePayload(payload, &inp); err != nil {
		return sendFailed(stream, taskID, "INVALID_INPUT", sanitise(err.Error()))
	}
	if inp.PRNumber <= 0 {
		return sendFailed(stream, taskID, "INVALID_INPUT", "pr_number must be a positive integer")
	}

	ch := make(chan diffResult, 1)
	go func() {
		// Use BareDo with a custom Accept header to get the raw diff media type.
		req, err := h.gh.NewRequest("GET",
			fmt.Sprintf("repos/%s/%s/pulls/%d", gcap.Owner, gcap.Repo, inp.PRNumber),
			nil)
		if err != nil {
			ch <- diffResult{err: err}
			return
		}
		req.Header.Set("Accept", "application/vnd.github.diff")

		resp, err := h.gh.BareDo(ctx, req)
		if err != nil {
			ch <- diffResult{err: err}
			return
		}
		defer func() { _ = resp.Body.Close() }()

		limited := io.LimitReader(resp.Body, maxDiffBytes+1)
		data, err := io.ReadAll(limited)
		if err != nil {
			ch <- diffResult{err: err}
			return
		}
		truncated := int64(len(data)) > maxDiffBytes
		if truncated {
			data = data[:maxDiffBytes]
		}
		ch <- diffResult{diff: string(data), truncated: truncated}
	}()

	tick := time.NewTicker(tickInterval)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			if err := stream.Send(progressEvent(taskID)); err != nil {
				return err //nolint:wrapcheck
			}
		case res := <-ch:
			if res.err != nil {
				return sendFailed(stream, taskID, githubErrCode(res.err), sanitise(res.err.Error()))
			}
			out, err := marshalPayload(map[string]interface{}{
				"diff":      res.diff,
				"truncated": res.truncated,
			})
			if err != nil {
				return sendFailed(stream, taskID, "UPSTREAM_ERROR", "failed to marshal diff response")
			}
			return stream.Send(completedEvent(taskID, out)) //nolint:wrapcheck
		case <-ctx.Done():
			return sendFailed(stream, taskID, "TIMEOUT", "request exceeded timeout_seconds")
		}
	}
}

type diffResult struct {
	diff      string
	truncated bool
	err       error
}

// ── helpers ───────────────────────────────────────────────────────────────────

func progressEvent(taskID string) *zynaxv1.TaskEvent {
	return &zynaxv1.TaskEvent{
		TaskId:    taskID,
		EventType: zynaxv1.TaskEventType_TASK_EVENT_TYPE_PROGRESS,
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

// githubErrCode maps GitHub API error responses to CapabilityError codes.
func githubErrCode(err error) string {
	if err == nil {
		return ""
	}
	var ghErr *github.ErrorResponse
	if errors.As(err, &ghErr) {
		switch ghErr.Response.StatusCode {
		case 429, 403:
			return "RESOURCE_EXHAUSTED"
		case 404:
			return "NOT_FOUND"
		case 422:
			return "INVALID_INPUT"
		}
	}
	return "UPSTREAM_ERROR"
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
