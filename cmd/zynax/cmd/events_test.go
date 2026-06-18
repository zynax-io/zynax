// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseDataFlags(t *testing.T) {
	tests := []struct {
		name    string
		input   []string
		want    map[string]string
		wantErr bool
	}{
		{"nil when empty", nil, nil, false},
		{"single pair", []string{"by=alice"}, map[string]string{"by": "alice"}, false},
		{"multiple pairs", []string{"a=1", "b=2"}, map[string]string{"a": "1", "b": "2"}, false},
		{"empty value allowed", []string{"k="}, map[string]string{"k": ""}, false},
		{"value with equals", []string{"url=a=b"}, map[string]string{"url": "a=b"}, false},
		{"missing equals", []string{"oops"}, nil, true},
		{"empty key", []string{"=v"}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDataFlags(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v; wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %v; want %v", got, tt.want)
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("got[%q] = %q; want %q", k, got[k], v)
				}
			}
		})
	}
}

func TestEventsPublishCmd_PostsEventAndSurfacesID(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody struct {
		EventType string            `json:"event_type"`
		Data      map[string]string `json:"data"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]string{"event_id": "evt-9"})
	}))
	defer srv.Close()
	apiURL = srv.URL
	eventData = []string{"by=alice"}
	defer func() { eventData = nil }()

	var out bytes.Buffer
	eventsPublishCmd.SetOut(&out)
	eventsPublishCmd.SetContext(context.Background())
	if err := eventsPublishCmd.RunE(eventsPublishCmd, []string{"run-7", "review.approved"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q; want POST", gotMethod)
	}
	if gotPath != "/api/v1/workflows/run-7/events" {
		t.Errorf("path = %q; want /api/v1/workflows/run-7/events", gotPath)
	}
	if gotBody.EventType != "review.approved" {
		t.Errorf("event_type = %q; want review.approved", gotBody.EventType)
	}
	if gotBody.Data["by"] != "alice" {
		t.Errorf("data[by] = %q; want alice", gotBody.Data["by"])
	}
	if !bytes.Contains(out.Bytes(), []byte("event_id: evt-9")) {
		t.Errorf("expected event_id in output, got:\n%s", out.String())
	}
}

func TestEventsPublishCmd_InvalidDataFlag(t *testing.T) {
	eventData = []string{"nope"}
	defer func() { eventData = nil }()
	eventsPublishCmd.SetContext(context.Background())
	if err := eventsPublishCmd.RunE(eventsPublishCmd, []string{"run-7", "review.approved"}); err == nil {
		t.Fatal("expected error for malformed --data, got nil")
	}
}
