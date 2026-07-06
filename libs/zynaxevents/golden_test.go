// SPDX-License-Identifier: Apache-2.0

package zynaxevents

// Golden byte-compat gate (M8.H, ADR-046). The fixtures under testdata/golden/
// pin the wire shape and naming conventions the retired event-bus facade
// produces — this library must reproduce them EXACTLY. The facade asserts the
// same fixtures (services/event-bus golden_compat_test.go) until M9 removes it.

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func loadGolden(t *testing.T, name string, v any) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "golden", name)) //nolint:gosec // fixture path built from constants
	if err != nil {
		t.Fatalf("reading golden %s: %v", name, err)
	}
	if err := json.Unmarshal(raw, v); err != nil {
		t.Fatalf("parsing golden %s: %v", name, err)
	}
}

func TestGolden_StreamDerivation(t *testing.T) {
	var g struct {
		Cases []struct {
			EventType      string   `json:"eventType"`
			StreamName     string   `json:"streamName"`
			SubjectFilter  string   `json:"subjectFilter"`
			StreamSubjects []string `json:"streamSubjects"`
		} `json:"cases"`
		PatternSubjects []struct {
			Pattern         string `json:"pattern"`
			ConcreteSubject string `json:"concreteSubject"`
		} `json:"patternSubjects"`
	}
	loadGolden(t, "stream_derivation.json", &g)
	if len(g.Cases) == 0 {
		t.Fatal("no golden cases loaded")
	}

	for _, c := range g.Cases {
		if got := StreamName(c.EventType); got != c.StreamName {
			t.Errorf("StreamName(%q) = %q, golden %q", c.EventType, got, c.StreamName)
		}
		if got := SubjectFilter(c.EventType); got != c.SubjectFilter {
			t.Errorf("SubjectFilter(%q) = %q, golden %q", c.EventType, got, c.SubjectFilter)
		}
		got := streamSubjects(c.EventType)
		if len(got) != len(c.StreamSubjects) {
			t.Errorf("streamSubjects(%q) = %v, golden %v", c.EventType, got, c.StreamSubjects)
			continue
		}
		for i := range got {
			if got[i] != c.StreamSubjects[i] {
				t.Errorf("streamSubjects(%q)[%d] = %q, golden %q", c.EventType, i, got[i], c.StreamSubjects[i])
			}
		}
	}

	for _, p := range g.PatternSubjects {
		if got := StreamSubjectFromPattern(p.Pattern); got != p.ConcreteSubject {
			t.Errorf("StreamSubjectFromPattern(%q) = %q, golden %q", p.Pattern, got, p.ConcreteSubject)
		}
	}
}

func TestGolden_CloudEventEnvelope(t *testing.T) {
	var g struct {
		Cases []struct {
			Name  string `json:"name"`
			Event struct {
				ID              string `json:"id"`
				Source          string `json:"source"`
				SpecVersion     string `json:"specVersion"`
				Type            string `json:"type"`
				DataContentType string `json:"dataContentType"`
				WorkflowID      string `json:"workflowID"`
				RunID           string `json:"runID"`
				Namespace       string `json:"namespace"`
				CapabilityName  string `json:"capabilityName"`
				DataBase64      string `json:"dataBase64"`
			} `json:"event"`
			WantJSON string `json:"wantJSON"`
		} `json:"cases"`
	}
	loadGolden(t, "cloudevent_envelope.json", &g)
	if len(g.Cases) == 0 {
		t.Fatal("no golden cases loaded")
	}

	for _, c := range g.Cases {
		var data []byte
		if c.Event.DataBase64 != "" {
			var err error
			data, err = base64.StdEncoding.DecodeString(c.Event.DataBase64)
			if err != nil {
				t.Fatalf("%s: bad dataBase64: %v", c.Name, err)
			}
		}
		env := cloudEventEnvelope{
			SpecVersion:     c.Event.SpecVersion,
			ID:              c.Event.ID,
			Source:          c.Event.Source,
			Type:            c.Event.Type,
			DataContentType: c.Event.DataContentType,
			WorkflowID:      c.Event.WorkflowID,
			RunID:           c.Event.RunID,
			Namespace:       c.Event.Namespace,
			CapabilityName:  c.Event.CapabilityName,
			Data:            data,
		}
		payload, err := json.Marshal(env)
		if err != nil {
			t.Fatalf("%s: marshal: %v", c.Name, err)
		}
		if string(payload) != c.WantJSON {
			t.Errorf("%s: envelope bytes drifted from golden\n got: %s\nwant: %s", c.Name, payload, c.WantJSON)
		}
	}
}

func TestGolden_DurableNames(t *testing.T) {
	var g struct {
		Cases []struct {
			SubscriberID string `json:"subscriberID"`
			Durable      string `json:"durable"`
		} `json:"cases"`
	}
	loadGolden(t, "durable_names.json", &g)
	if len(g.Cases) == 0 {
		t.Fatal("no golden cases loaded")
	}
	for _, c := range g.Cases {
		if got := DurableConsumerName(c.SubscriberID); got != c.Durable {
			t.Errorf("DurableConsumerName(%q) = %q, golden %q", c.SubscriberID, got, c.Durable)
		}
	}
}

func TestGolden_GlobMatching(t *testing.T) {
	var g struct {
		Cases []struct {
			Pattern   string `json:"pattern"`
			EventType string `json:"eventType"`
			Matches   bool   `json:"matches"`
		} `json:"cases"`
	}
	loadGolden(t, "glob_matching.json", &g)
	if len(g.Cases) == 0 {
		t.Fatal("no golden cases loaded")
	}
	for _, c := range g.Cases {
		if got := MatchesGlob(c.Pattern, c.EventType); got != c.Matches {
			t.Errorf("MatchesGlob(%q, %q) = %v, golden %v", c.Pattern, c.EventType, got, c.Matches)
		}
	}
}

func TestGolden_RetryPolicy(t *testing.T) {
	var g struct {
		MaxDeliver         int      `json:"maxDeliver"`
		Backoff            []string `json:"backoff"`
		DurableNameByteCap int      `json:"durableNameByteCap"`
	}
	loadGolden(t, "retry_policy.json", &g)

	if len(RetryBackoff) != g.MaxDeliver {
		t.Errorf("len(RetryBackoff) = %d, golden maxDeliver %d", len(RetryBackoff), g.MaxDeliver)
	}
	if len(RetryBackoff) != len(g.Backoff) {
		t.Fatalf("len(RetryBackoff) = %d, golden backoff has %d entries", len(RetryBackoff), len(g.Backoff))
	}
	for i, want := range g.Backoff {
		if got := RetryBackoff[i].String(); got != want {
			t.Errorf("RetryBackoff[%d] = %s, golden %s", i, got, want)
		}
	}
	long := make([]byte, 300)
	for i := range long {
		long[i] = 'a'
	}
	if got := DurableConsumerName(string(long)); len(got) != g.DurableNameByteCap {
		t.Errorf("durable name cap = %d bytes, golden %d", len(got), g.DurableNameByteCap)
	}
}

func TestIsTerminalEventType(t *testing.T) {
	terminal := []string{
		"zynax.v1.engine-adapter.workflow.completed",
		"zynax.v1.engine-adapter.workflow.failed",
		"zynax.v1.engine-adapter.workflow.terminated",
		"zynax.v1.engine-adapter.workflow.canceled",
		"zynax.v1.engine-adapter.workflow.timed_out",
		"workflow.completed",
	}
	for _, et := range terminal {
		if !IsTerminalEventType(et) {
			t.Errorf("IsTerminalEventType(%q) = false, want true", et)
		}
	}
	nonTerminal := []string{
		"zynax.v1.engine-adapter.workflow.state.entered",
		"zynax.v1.engine-adapter.workflow.started",
		"zynax.v1.task-broker.task.dispatched",
		"workflowcompleted",
	}
	for _, et := range nonTerminal {
		if IsTerminalEventType(et) {
			t.Errorf("IsTerminalEventType(%q) = true, want false", et)
		}
	}
}

// TestGolden_DLQNaming pins the dead-letter naming conventions.
func TestGolden_DLQNaming(t *testing.T) {
	var g struct {
		Cases []struct {
			EventType         string `json:"eventType"`
			DLQStreamName     string `json:"dlqStreamName"`
			DLQDeliverSubject string `json:"dlqDeliverSubject"`
		} `json:"cases"`
	}
	loadGolden(t, "stream_derivation.json", &g)
	if len(g.Cases) == 0 {
		t.Fatal("no golden cases loaded")
	}
	for _, c := range g.Cases {
		if got := dlqStreamName(StreamName(c.EventType)); got != c.DLQStreamName {
			t.Errorf("dlqStreamName(%q) = %q, golden %q", c.EventType, got, c.DLQStreamName)
		}
		if got := dlqDeliverSubject(c.EventType); got != c.DLQDeliverSubject {
			t.Errorf("dlqDeliverSubject(%q) = %q, golden %q", c.EventType, got, c.DLQDeliverSubject)
		}
	}
}
