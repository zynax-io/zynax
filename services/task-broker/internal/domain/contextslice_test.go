// SPDX-License-Identifier: Apache-2.0

// Context-slice injection binding tests (EPIC #881 O5, ADR-028). Acceptance:
// (1) an expert invocation receives only its slice files (bounded max_tokens);
// (2) isolation — no cross-expert context leakage, even via a smuggled slice;
// (3) fan-out is durable across a simulated restart (repository-backed state).
package domain_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/zynax-io/zynax/services/task-broker/internal/domain"
)

// recordingExecutor captures the exact input payload delivered to each agent.
type recordingExecutor struct {
	mu    sync.Mutex
	calls map[string][]byte // agent ID -> input payload as executed
}

func newRecordingExecutor() *recordingExecutor {
	return &recordingExecutor{calls: make(map[string][]byte)}
}

func (e *recordingExecutor) Execute(_ context.Context, agent domain.AgentInfo, task *domain.Task) ([]byte, *domain.TaskError, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.calls[agent.AgentID] = append([]byte(nil), task.InputPayload...)
	return []byte(`{"ok":true}`), nil, nil
}

func (e *recordingExecutor) payloadFor(t *testing.T, agentID string) map[string]json.RawMessage {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	raw, ok := e.calls[agentID]
	if !ok {
		t.Fatalf("agent %q was never invoked", agentID)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatalf("payload for %q is not a JSON object: %v", agentID, err)
	}
	return fields
}

// fakePublisher records every published task lifecycle event as "<id>:<status>".
type fakePublisher struct {
	mu     sync.Mutex
	events []string
}

func (p *fakePublisher) PublishTaskEvent(_ context.Context, task *domain.Task) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, task.TaskID+":"+task.Status.String())
}

func (p *fakePublisher) snapshot() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.events...)
}

// expertSchema builds a registered capability input_schema declaring a context
// slice — the shape the automation/workflows/experts/ AgentDefs register.
func expertSchema(t *testing.T, files []string, maxTokens int) []byte {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"trigger": map[string]any{"type": "string"},
			"context_slice": map[string]any{
				"type":     "object",
				"required": []string{"files", "max_tokens"},
				"properties": map[string]any{
					"files":      map[string]any{"type": "array", "default": files},
					"max_tokens": map[string]any{"type": "integer", "default": maxTokens},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal schema: %v", err)
	}
	return raw
}

var (
	archSliceFiles = []string{"AGENTS.md", "docs/adr/INDEX.md", "docs/adr/*.md"}
	qaSliceFiles   = []string{"protos/tests/**/*.feature", "coverage/**"}
)

// reviewExperts returns two expert agents both providing "review", each with
// its own declared context slice — the isolation test pair.
func reviewExperts(t *testing.T) map[string][]domain.AgentInfo {
	t.Helper()
	return map[string][]domain.AgentInfo{"review": {
		{AgentID: "agent-arch", Name: "arch-adr", Endpoint: "localhost:9001",
			InputSchema: expertSchema(t, archSliceFiles, 4000)},
		{AgentID: "agent-qa", Name: "qa-bdd", Endpoint: "localhost:9002",
			InputSchema: expertSchema(t, qaSliceFiles, 3000)},
	}}
}

func reviewTask(payload string) *domain.Task {
	return &domain.Task{WorkflowID: "wf-orch", CapabilityName: "review", InputPayload: []byte(payload)}
}

func sliceOf(t *testing.T, fields map[string]json.RawMessage) domain.ContextSlice {
	t.Helper()
	raw, ok := fields["context_slice"]
	if !ok {
		t.Fatalf("payload has no context_slice: %s", fields)
	}
	var slice domain.ContextSlice
	if err := json.Unmarshal(raw, &slice); err != nil {
		t.Fatalf("unmarshal context_slice: %v", err)
	}
	return slice
}

func assertFiles(t *testing.T, got, want []string) {
	t.Helper()
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("slice files = %v, want %v", got, want)
	}
}

// ── AC1: expert invocation receives only its declared slice ───────────────

func TestDispatchTask_BindsDeclaredContextSlice(t *testing.T) {
	repo := newFakeRepo()
	exec := newRecordingExecutor()
	svc := domain.NewTaskService(repo, &fakeFinder{agents: reviewExperts(t)}, exec)

	taskID, _, err := svc.DispatchTask(context.Background(),
		reviewTask(`{"expert":"arch-adr","trigger":"pull_request"}`))
	if err != nil {
		t.Fatalf("DispatchTask: %v", err)
	}
	svc.WaitBackground()

	payload := exec.payloadFor(t, "agent-arch")
	slice := sliceOf(t, payload)
	assertFiles(t, slice.Files, archSliceFiles)
	if slice.MaxTokens != 4000 {
		t.Errorf("max_tokens = %d, want 4000 (bounded by declaration)", slice.MaxTokens)
	}
	var trigger string
	_ = json.Unmarshal(payload["trigger"], &trigger)
	if trigger != "pull_request" {
		t.Errorf("trigger = %q, want pull_request (non-slice fields untouched)", trigger)
	}

	// The bound payload is what was persisted: restart recovery re-dispatches
	// the same slice (durability of the binding) — and targeting was strict.
	stored, _ := repo.GetByID(context.Background(), taskID)
	var storedFields map[string]json.RawMessage
	_ = json.Unmarshal(stored.InputPayload, &storedFields)
	assertFiles(t, sliceOf(t, storedFields).Files, archSliceFiles)
	if stored.DispatchedTo != "agent-arch" {
		t.Errorf("dispatched_to = %q, want agent-arch (strict expert targeting)", stored.DispatchedTo)
	}
}

// ── AC2: strict isolation — no cross-expert context leakage ───────────────

func TestDispatchTask_StrictIsolation_NoCrossExpertLeakage(t *testing.T) {
	exec := newRecordingExecutor()
	svc := domain.NewTaskService(newFakeRepo(), &fakeFinder{agents: reviewExperts(t)}, exec)

	// A caller tries to smuggle qa-bdd's slice into the arch-adr invocation.
	smuggled := fmt.Sprintf(`{"expert":"arch-adr","trigger":"pull_request","context_slice":{"files":[%q],"max_tokens":99999}}`,
		qaSliceFiles[0])
	if _, _, err := svc.DispatchTask(context.Background(), reviewTask(smuggled)); err != nil {
		t.Fatalf("DispatchTask arch-adr: %v", err)
	}
	if _, _, err := svc.DispatchTask(context.Background(),
		reviewTask(`{"expert":"qa-bdd","trigger":"pull_request"}`)); err != nil {
		t.Fatalf("DispatchTask qa-bdd: %v", err)
	}
	svc.WaitBackground()

	archPayload := exec.payloadFor(t, "agent-arch")
	qaPayload := exec.payloadFor(t, "agent-qa")

	// Each expert sees exactly its own declared slice (smuggled budget gone)…
	assertFiles(t, sliceOf(t, archPayload).Files, archSliceFiles)
	assertFiles(t, sliceOf(t, qaPayload).Files, qaSliceFiles)
	if sliceOf(t, archPayload).MaxTokens != 4000 {
		t.Errorf("smuggled max_tokens survived: %d", sliceOf(t, archPayload).MaxTokens)
	}
	// …and nothing of the other expert's slice appears anywhere in the
	// payload (leak check over the raw bytes, both directions).
	rawArch, _ := json.Marshal(archPayload)
	rawQA, _ := json.Marshal(qaPayload)
	for _, f := range qaSliceFiles {
		if strings.Contains(string(rawArch), f) {
			t.Errorf("arch-adr payload leaks qa-bdd slice file %q", f)
		}
	}
	for _, f := range archSliceFiles {
		if strings.Contains(string(rawQA), f) {
			t.Errorf("qa-bdd payload leaks arch-adr slice file %q", f)
		}
	}
}

func TestDispatchTask_UnknownExpert_NeverFallsBack(t *testing.T) {
	svc := domain.NewTaskService(newFakeRepo(), &fakeFinder{agents: reviewExperts(t)}, newRecordingExecutor())
	_, _, err := svc.DispatchTask(context.Background(),
		reviewTask(`{"expert":"security-supply-chain","trigger":"pull_request"}`))
	if !errors.Is(err, domain.ErrNoEligibleAgent) {
		t.Errorf("err = %v, want ErrNoEligibleAgent (strict targeting, no fallback)", err)
	}
}

func TestDispatchTask_ExpertWithoutDeclaredSlice_StripsCallerSlice(t *testing.T) {
	exec := newRecordingExecutor()
	agents := map[string][]domain.AgentInfo{"review": {
		{AgentID: "agent-plain", Name: "plain-expert", Endpoint: "localhost:9003",
			InputSchema: []byte(`{"type":"object","properties":{"trigger":{"type":"string"}}}`)},
	}}
	svc := domain.NewTaskService(newFakeRepo(), &fakeFinder{agents: agents}, exec)

	_, _, err := svc.DispatchTask(context.Background(),
		reviewTask(`{"expert":"plain-expert","context_slice":{"files":["secret.md"],"max_tokens":1}}`))
	if err != nil {
		t.Fatalf("DispatchTask: %v", err)
	}
	svc.WaitBackground()
	if _, ok := exec.payloadFor(t, "agent-plain")["context_slice"]; ok {
		t.Error("undeclared context_slice passed through")
	}
}

func TestDispatchTask_MalformedRegisteredSchema(t *testing.T) {
	for name, schema := range map[string][]byte{
		"invalid_json":     []byte(`{not json`),
		"slice_not_object": []byte(`{"properties":{"context_slice":"not-an-object"}}`),
	} {
		t.Run(name, func(t *testing.T) {
			agents := map[string][]domain.AgentInfo{"review": {
				{AgentID: "agent-bad", Name: "bad-expert", Endpoint: "localhost:9004", InputSchema: schema},
			}}
			svc := domain.NewTaskService(newFakeRepo(), &fakeFinder{agents: agents}, newRecordingExecutor())
			if _, _, err := svc.DispatchTask(context.Background(), reviewTask(`{"expert":"bad-expert"}`)); err == nil {
				t.Fatal("expected error for malformed registered input_schema")
			}
		})
	}
}

func TestDispatchTask_NonExpertPayloadUntouched(t *testing.T) {
	// No (string) "expert" field => not expert-targeted: the payload passes
	// through verbatim, even when it carries a context_slice.
	for name, payload := range map[string]string{
		"no_expert_field":   `{"text":"summarize me","context_slice":{"files":["a.md"],"max_tokens":10}}`,
		"non_string_expert": `{"expert":42,"context_slice":{"files":["a.md"],"max_tokens":10}}`,
	} {
		t.Run(name, func(t *testing.T) {
			exec := newRecordingExecutor()
			svc := domain.NewTaskService(newFakeRepo(), &fakeFinder{agents: oneAgent()}, exec)
			task := validTask()
			task.InputPayload = []byte(payload)
			if _, _, err := svc.DispatchTask(context.Background(), task); err != nil {
				t.Fatalf("DispatchTask: %v", err)
			}
			svc.WaitBackground()
			if _, ok := exec.payloadFor(t, "a1")["context_slice"]; !ok {
				t.Error("non-targeted payload lost its context_slice field")
			}
		})
	}
}

// ── AC3: fan-out durable across a simulated restart ───────────────────────

func TestRecoverInFlight_SimulatedRestart(t *testing.T) {
	repo := newFakeRepo()

	// A previous broker instance dispatched a fan-out and crashed: the repo
	// (Postgres-backed in production, #626) holds the bound non-terminal tasks.
	boundArch := fmt.Sprintf(`{"expert":"arch-adr","context_slice":{"files":["%s"],"max_tokens":4000}}`,
		strings.Join(archSliceFiles, `","`))
	boundQA := fmt.Sprintf(`{"expert":"qa-bdd","context_slice":{"files":["%s"],"max_tokens":3000}}`,
		strings.Join(qaSliceFiles, `","`))
	seed := func(id, payload string, status domain.TaskStatus) {
		storeTask(t, repo, &domain.Task{TaskID: id, WorkflowID: "wf", CapabilityName: "review",
			InputPayload: []byte(payload), Status: status, MaxRetries: 2, RetryCount: 1})
	}
	seed("t-pending", boundArch, domain.TaskStatusPending)
	seed("t-dispatched", boundQA, domain.TaskStatusDispatched)
	seed("t-retrying", boundArch, domain.TaskStatusRetrying)
	seed("t-done", `{}`, domain.TaskStatusCompleted)

	// The "restarted" broker instance shares only the repository.
	exec := newRecordingExecutor()
	restarted := domain.NewTaskService(repo, &fakeFinder{agents: reviewExperts(t)}, exec)

	n, err := restarted.RecoverInFlight(context.Background())
	if err != nil {
		t.Fatalf("RecoverInFlight: %v", err)
	}
	if n != 3 {
		t.Fatalf("recovered = %d, want 3 (terminal tasks excluded)", n)
	}
	restarted.WaitBackground()

	for _, id := range []string{"t-pending", "t-dispatched", "t-retrying"} {
		task, _ := repo.GetByID(context.Background(), id)
		if task.Status != domain.TaskStatusCompleted {
			t.Errorf("%s: status = %s, want COMPLETED after recovery", id, task.Status)
		}
	}

	// Strict targeting holds across the restart: the persisted bound slice
	// reached the right expert, untouched.
	assertFiles(t, sliceOf(t, exec.payloadFor(t, "agent-qa")).Files, qaSliceFiles)
	archTask, _ := repo.GetByID(context.Background(), "t-pending")
	if archTask.DispatchedTo != "agent-arch" {
		t.Errorf("t-pending dispatched_to = %q, want agent-arch", archTask.DispatchedTo)
	}
}

func TestRecoverInFlight_MissingExpertLeftForNextPass(t *testing.T) {
	repo := newFakeRepo()
	storeTask(t, repo, &domain.Task{TaskID: "t-orphan", WorkflowID: "wf", CapabilityName: "review",
		InputPayload: []byte(`{"expert":"arch-adr"}`), Status: domain.TaskStatusPending})

	svc := domain.NewTaskService(repo, &fakeFinder{agents: map[string][]domain.AgentInfo{}}, newRecordingExecutor())
	n, err := svc.RecoverInFlight(context.Background())
	if err != nil || n != 0 {
		t.Fatalf("RecoverInFlight: n=%d err=%v, want 0 recovered", n, err)
	}
	task, _ := repo.GetByID(context.Background(), "t-orphan")
	if task.Status != domain.TaskStatusPending {
		t.Errorf("orphan task status = %s, want PENDING (never re-routed)", task.Status)
	}
}

type listErrorRepo struct{ fakeRepo }

func (r *listErrorRepo) List(_ context.Context, _ domain.ListFilter) (domain.ListResult, error) {
	return domain.ListResult{}, fmt.Errorf("db unavailable")
}

func TestRecoverInFlight_RepositoryError(t *testing.T) {
	svc := domain.NewTaskService(&listErrorRepo{fakeRepo: *newFakeRepo()}, &fakeFinder{}, &fakeExecutor{})
	if n, err := svc.RecoverInFlight(context.Background()); err == nil || n != 0 {
		t.Fatalf("RecoverInFlight: n=%d err=%v, want repository error surfaced", n, err)
	}
}

// ── lifecycle events over the bus ──────────────────────────────────────────

func TestDispatchTask_PublishesLifecycleEvents(t *testing.T) {
	pub := &fakePublisher{}
	svc := domain.NewTaskService(newFakeRepo(), &fakeFinder{agents: reviewExperts(t)}, newRecordingExecutor()).
		WithEventPublisher(pub)

	taskID, _, err := svc.DispatchTask(context.Background(), reviewTask(`{"expert":"qa-bdd"}`))
	if err != nil {
		t.Fatalf("DispatchTask: %v", err)
	}
	svc.WaitBackground()

	want := fmt.Sprint([]string{taskID + ":DISPATCHED", taskID + ":COMPLETED"})
	if got := fmt.Sprint(pub.snapshot()); got != want {
		t.Errorf("events = %v, want %v", got, want)
	}
}

func TestAckAndCancel_PublishEvents(t *testing.T) {
	repo := newFakeRepo()
	pub := &fakePublisher{}
	svc := domain.NewTaskService(repo, &fakeFinder{}, &fakeExecutor{}).WithEventPublisher(pub)
	storeTask(t, repo, dispatched("ack-ev"))
	storeTask(t, repo, &domain.Task{TaskID: "c-ev", WorkflowID: "wf", CapabilityName: "cap",
		InputPayload: []byte(`{}`), Status: domain.TaskStatusPending})

	if _, err := svc.AcknowledgeTask(context.Background(), "ack-ev", domain.TaskStatusCompleted, []byte(`{"r":1}`), nil); err != nil {
		t.Fatalf("AcknowledgeTask: %v", err)
	}
	if _, err := svc.CancelTask(context.Background(), "c-ev"); err != nil {
		t.Fatalf("CancelTask: %v", err)
	}
	want := fmt.Sprint([]string{"ack-ev:COMPLETED", "c-ev:CANCELLED"})
	if got := fmt.Sprint(pub.snapshot()); got != want {
		t.Errorf("events = %v, want %v", got, want)
	}
}
