// SPDX-License-Identifier: Apache-2.0

package adapter

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/zynax-io/zynax/agents/adapters/http/internal/config"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	maxResponseBytes = 10 * 1024 * 1024 // 10 MB hard cap on response body
	maxErrMsgLen     = 512
	tickInterval     = 2 * time.Second
)

type httpHandler struct {
	client *http.Client
}

func newHTTPHandler() *httpHandler {
	return &httpHandler{client: &http.Client{}}
}

// executeStream is the minimal surface of AgentService_ExecuteCapabilityServer used by the handler.
type executeStream interface {
	Send(*zynaxv1.TaskEvent) error
	Context() context.Context
}

func (h *httpHandler) execute(
	ctx context.Context,
	route config.RouteConfig,
	taskID string,
	payload []byte,
	stream executeStream,
) error {
	var body io.Reader
	method := strings.ToUpper(route.Method)
	if len(payload) > 0 && (method == "POST" || method == "PUT" || method == "PATCH") {
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, route.URL, body)
	if err != nil {
		return sendFailed(stream, taskID, "UPSTREAM_ERROR", sanitise(err.Error()))
	}
	for k, v := range route.Headers {
		req.Header.Set(k, v)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	type result struct {
		resp *http.Response
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		resp, err := h.client.Do(req)
		ch <- result{resp, err}
	}()

	tick := time.NewTicker(tickInterval)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			if err := stream.Send(progressEvent(taskID)); err != nil {
				return err
			}
		case res := <-ch:
			if res.err != nil {
				if ctx.Err() != nil {
					return sendFailed(stream, taskID, "TIMEOUT", "request exceeded timeout_seconds")
				}
				return sendFailed(stream, taskID, "UPSTREAM_ERROR", sanitise(res.err.Error()))
			}
			defer res.resp.Body.Close()
			respBody, err := io.ReadAll(io.LimitReader(res.resp.Body, maxResponseBytes+1))
			if err != nil {
				return sendFailed(stream, taskID, "UPSTREAM_ERROR", "failed to read response body")
			}
			if int64(len(respBody)) > maxResponseBytes {
				return sendFailed(stream, taskID, "UPSTREAM_ERROR", "response body exceeds 10 MB limit")
			}
			if res.resp.StatusCode >= 200 && res.resp.StatusCode < 300 {
				return stream.Send(completedEvent(taskID, respBody))
			}
			return sendFailed(stream, taskID, "UPSTREAM_ERROR",
				fmt.Sprintf("upstream returned HTTP %d", res.resp.StatusCode))
		case <-ctx.Done():
			return sendFailed(stream, taskID, "TIMEOUT", "request exceeded timeout_seconds")
		}
	}
}

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
	return stream.Send(&zynaxv1.TaskEvent{
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
