// SPDX-License-Identifier: Apache-2.0

package adapter_test

// Tests for the requestReview capability handler and progressEvent helper.
// Closes #715 — part of the git-adapter coverage epic (#713).

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
)

const errCodeInvalidInput = "INVALID_INPUT"

// ── requestReview input validation ───────────────────────────────────────────

func TestRequestReview_InvalidPRNumber(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, "")
	stream := &stubStream{ctx: context.Background()}
	payload, _ := json.Marshal(map[string]interface{}{
		"pr_number": 0,
		"reviewers": []string{"alice"},
	})
	req := &zynaxv1.ExecuteCapabilityRequest{
		TaskId:         "t-rr-zero",
		CapabilityName: "request_review",
		InputPayload:   payload,
	}
	if err := srv.ExecuteCapability(req, stream); err != nil {
		t.Fatalf("unexpected gRPC error: %v", err)
	}
	if len(stream.events) == 0 {
		t.Fatal("expected at least one event")
	}
	last := stream.last()
	if last.EventType != zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED {
		t.Fatalf("expected FAILED, got %v", last.EventType)
	}
	if last.Error.Code != errCodeInvalidInput {
		t.Errorf("expected INVALID_INPUT, got %q", last.Error.Code)
	}
}

func TestRequestReview_NegativePRNumber(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, "")
	stream := &stubStream{ctx: context.Background()}
	payload, _ := json.Marshal(map[string]interface{}{
		"pr_number": -5,
		"reviewers": []string{"alice"},
	})
	req := &zynaxv1.ExecuteCapabilityRequest{
		TaskId:         "t-rr-neg",
		CapabilityName: "request_review",
		InputPayload:   payload,
	}
	if err := srv.ExecuteCapability(req, stream); err != nil {
		t.Fatalf("unexpected gRPC error: %v", err)
	}
	last := stream.last()
	if last.Error.Code != errCodeInvalidInput {
		t.Errorf("expected INVALID_INPUT, got %q", last.Error.Code)
	}
}

func TestRequestReview_EmptyReviewers(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, "")
	stream := &stubStream{ctx: context.Background()}
	payload, _ := json.Marshal(map[string]interface{}{
		"pr_number": 5,
		"reviewers": []string{},
	})
	req := &zynaxv1.ExecuteCapabilityRequest{
		TaskId:         "t-rr-noreviewers",
		CapabilityName: "request_review",
		InputPayload:   payload,
	}
	if err := srv.ExecuteCapability(req, stream); err != nil {
		t.Fatalf("unexpected gRPC error: %v", err)
	}
	last := stream.last()
	if last.EventType != zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED {
		t.Fatalf("expected FAILED, got %v", last.EventType)
	}
	if last.Error.Code != errCodeInvalidInput {
		t.Errorf("expected INVALID_INPUT, got %q", last.Error.Code)
	}
}

// ── requestReview GitHub API error ────────────────────────────────────────────

func TestRequestReview_APIError_404(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/test-owner/test-repo/pulls/3/requested_reviewers",
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "Not Found"})
		})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	srv := newTestServer(t, ts.URL)
	stream := &stubStream{ctx: context.Background()}
	payload, _ := json.Marshal(map[string]interface{}{
		"pr_number": 3,
		"reviewers": []string{"alice"},
	})
	req := &zynaxv1.ExecuteCapabilityRequest{
		TaskId:         "t-rr-404",
		CapabilityName: "request_review",
		InputPayload:   payload,
	}
	_ = srv.ExecuteCapability(req, stream)
	last := stream.last()
	if last.EventType != zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED {
		t.Fatalf("expected FAILED, got %v", last.EventType)
	}
	if last.Error.Code != "NOT_FOUND" {
		t.Errorf("expected NOT_FOUND, got %q", last.Error.Code)
	}
}

// ── requestReview poll loop: progressEvent + context cancellation ─────────────

// TestRequestReview_ContextCancelledDuringPoll exercises the progressEvent helper
// and the ctx.Done() branch inside the poll loop.
// A 100ms context timeout lets the mock RequestReviewers call succeed but fires
// ctx.Done before pollInterval (3s) elapses.
func TestRequestReview_ContextCancelledDuringPoll(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	// RequestReviewers succeeds immediately.
	mux.HandleFunc("/repos/test-owner/test-repo/pulls/4/requested_reviewers",
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"number": 4})
		})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// 100ms: enough for the mock to respond but short enough to fire ctx.Done
	// before pollInterval (3s), exercising the TIMEOUT branch.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	srv := newTestServer(t, ts.URL)
	stream := &stubStream{ctx: ctx}
	payload, _ := json.Marshal(map[string]interface{}{
		"pr_number": 4,
		"reviewers": []string{"alice"},
	})
	req := &zynaxv1.ExecuteCapabilityRequest{
		TaskId:         "t-rr-ctx-cancel",
		CapabilityName: "request_review",
		InputPayload:   payload,
	}
	_ = srv.ExecuteCapability(req, stream)

	if len(stream.events) == 0 {
		t.Fatal("expected at least one event")
	}
	// First event is a progress event (from the poll loop entry before ctx.Done fires).
	first := stream.events[0]
	if first.EventType != zynaxv1.TaskEventType_TASK_EVENT_TYPE_PROGRESS {
		t.Errorf("expected first event to be PROGRESS (progressEvent helper), got %v", first.EventType)
	}
	if first.TaskId != "t-rr-ctx-cancel" {
		t.Errorf("expected task_id 't-rr-ctx-cancel', got %q", first.TaskId)
	}
	if first.Timestamp == nil {
		t.Error("progressEvent must set a non-nil Timestamp")
	}

	last := stream.last()
	if last.EventType != zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED {
		t.Fatalf("expected FAILED after timeout, got %v", last.EventType)
	}
	if last.Error.Code != "TIMEOUT" {
		t.Errorf("expected TIMEOUT code, got %q", last.Error.Code)
	}
}

// ── requestReview happy path: reviewer confirmed on first poll ─────────────────

// TestRequestReview_HappyPath verifies that the poll loop completes with COMPLETED
// once the reviewer appears in PR.RequestedReviewers.
// NOTE: this test intentionally waits ~3s (one pollInterval) for the poll tick.
func TestRequestReview_HappyPath(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	// POST requested_reviewers: success.
	mux.HandleFunc("/repos/test-owner/test-repo/pulls/6/requested_reviewers",
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"number": 6,
				"requested_reviewers": []map[string]interface{}{
					{"login": "alice", "id": 1},
				},
			})
		})
	// GET pull: returns PR with alice as requested reviewer.
	mux.HandleFunc("/repos/test-owner/test-repo/pulls/6",
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"number": 6,
				"requested_reviewers": []map[string]interface{}{
					{"login": "alice", "id": 1},
				},
			})
		})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// Allow 10 seconds: enough for one poll cycle (3s) plus margin.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	srv := newTestServer(t, ts.URL)
	stream := &stubStream{ctx: ctx}
	payload, _ := json.Marshal(map[string]interface{}{
		"pr_number": 6,
		"reviewers": []string{"alice"},
	})
	req := &zynaxv1.ExecuteCapabilityRequest{
		TaskId:         "t-rr-happy",
		CapabilityName: "request_review",
		InputPayload:   payload,
	}
	if err := srv.ExecuteCapability(req, stream); err != nil {
		t.Fatalf("unexpected gRPC error: %v", err)
	}

	last := stream.last()
	if last.EventType != zynaxv1.TaskEventType_TASK_EVENT_TYPE_COMPLETED {
		t.Fatalf("expected COMPLETED, got %v — error: %v", last.EventType, last.Error)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(last.Payload, &out); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if out["requested"] != true {
		t.Errorf("expected requested=true, got %v", out["requested"])
	}
}
