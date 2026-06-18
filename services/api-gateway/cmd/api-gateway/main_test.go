// SPDX-License-Identifier: Apache-2.0

package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// flushableRecorder is a ResponseWriter that records whether Flush was called,
// so we can assert the statusRecorder wrapper forwards Flush to the real writer.
type flushableRecorder struct {
	*httptest.ResponseRecorder
	flushed bool
}

func (f *flushableRecorder) Flush() { f.flushed = true }

func TestValidateConfig_EmptyKey_ProductionMode_Fails(t *testing.T) {
	err := validateConfig(config{})
	if err == nil {
		t.Fatal("expected error for empty API key in production mode, got nil")
	}
}

func TestValidateConfig_EmptyKey_DevInsecure_OK(t *testing.T) {
	err := validateConfig(config{DevInsecure: true})
	if err != nil {
		t.Fatalf("expected no error in dev-insecure mode, got: %v", err)
	}
}

func TestValidateConfig_NonEmptyKey_OK(t *testing.T) {
	err := validateConfig(config{APIKey: "test-secret"})
	if err != nil {
		t.Fatalf("expected no error with API key set, got: %v", err)
	}
}

// TestStatusRecorder_ForwardsFlush guards the fix for #1373: the workRecord
// statusRecorder must implement http.Flusher and forward Flush to the wrapped
// writer, otherwise the SSE logs endpoint 500s with "streaming not supported".
func TestStatusRecorder_ForwardsFlush(t *testing.T) {
	inner := &flushableRecorder{ResponseRecorder: httptest.NewRecorder()}
	rec := &statusRecorder{ResponseWriter: inner, code: http.StatusOK}

	f, ok := any(rec).(http.Flusher)
	if !ok {
		t.Fatal("statusRecorder does not implement http.Flusher")
	}
	f.Flush()
	if !inner.flushed {
		t.Error("Flush was not forwarded to the underlying ResponseWriter")
	}
}

// TestStatusRecorder_UnwrapExposesWriter ensures http.ResponseController can
// reach the real writer (needed to clear the write deadline on long streams).
func TestStatusRecorder_UnwrapExposesWriter(t *testing.T) {
	inner := httptest.NewRecorder()
	rec := &statusRecorder{ResponseWriter: inner, code: http.StatusOK}

	if got := rec.Unwrap(); got != inner {
		t.Errorf("Unwrap returned %v, want the wrapped writer", got)
	}
}
