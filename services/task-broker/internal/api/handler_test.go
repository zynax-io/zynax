// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/task-broker/internal/api"
	"github.com/zynax-io/zynax/services/task-broker/internal/domain"
)

const bufSize = 1024 * 1024

// ── fakes ──────────────────────────────────────────────────────────────────

type fakeRepo struct {
	mu    sync.Mutex
	tasks map[string]*domain.Task
}

func newFakeRepo() *fakeRepo { return &fakeRepo{tasks: make(map[string]*domain.Task)} }

func (r *fakeRepo) Save(_ context.Context, t *domain.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	c := *t
	r.tasks[c.TaskID] = &c
	return nil
}

func (r *fakeRepo) GetByID(_ context.Context, id string) (*domain.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.tasks[id]
	if !ok {
		return nil, fmt.Errorf("%w: %q", domain.ErrTaskNotFound, id)
	}
	c := *t
	return &c, nil
}

func (r *fakeRepo) Update(_ context.Context, t *domain.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	c := *t
	r.tasks[t.TaskID] = &c
	return nil
}

func (r *fakeRepo) List(_ context.Context, _ domain.ListFilter) (domain.ListResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*domain.Task
	for _, t := range r.tasks {
		c := *t
		out = append(out, &c)
	}
	return domain.ListResult{Tasks: out}, nil
}

type fakeFinder struct{ agents map[string][]domain.AgentInfo }

// Select implements domain.AgentSelector with the reference semantics of the
// CRD-native scheduler: capability lookup, strict expert filter (no
// fallback), first eligible agent deterministically.
func (f *fakeFinder) Select(_ context.Context, capName, expertTarget string) (domain.AgentInfo, error) {
	agents := f.agents[capName]
	if len(agents) == 0 {
		return domain.AgentInfo{}, fmt.Errorf("%w: no agent declares capability %q", domain.ErrNoEligibleAgent, capName)
	}
	if expertTarget != "" {
		for _, a := range agents {
			if a.Name == expertTarget || a.AgentID == expertTarget {
				return a, nil
			}
		}
		return domain.AgentInfo{}, fmt.Errorf("%w: no agent %q", domain.ErrNoEligibleAgent, expertTarget)
	}
	return agents[0], nil
}

type errorFinder struct{ err error }

func (f *errorFinder) Select(_ context.Context, _, _ string) (domain.AgentInfo, error) {
	return domain.AgentInfo{}, f.err
}

type fakeExecutor struct{}

func (e *fakeExecutor) Execute(_ context.Context, _ domain.AgentInfo, _ *domain.Task) ([]byte, *domain.TaskError, error) {
	return []byte(`{"result":"ok"}`), nil, nil
}

// blockingExecutor blocks in Execute until released, letting tests control task lifecycle.
type blockingExecutor struct {
	ready   chan struct{}
	release chan struct{}
	once    sync.Once
}

func newBlockingExecutor() *blockingExecutor {
	return &blockingExecutor{ready: make(chan struct{}), release: make(chan struct{})}
}

func (e *blockingExecutor) Execute(_ context.Context, _ domain.AgentInfo, _ *domain.Task) ([]byte, *domain.TaskError, error) {
	e.once.Do(func() { close(e.ready) })
	<-e.release
	return []byte(`{"result":"ok"}`), nil, nil
}

// ── server helper ───────────────────────────────────────────────────────────

type testEnv struct {
	client zynaxv1.TaskBrokerServiceClient
	svc    *domain.TaskService
}

func newTestEnv(t *testing.T, finder domain.AgentSelector, exec domain.CapabilityExecutor) *testEnv {
	t.Helper()
	repo := newFakeRepo()
	svc := domain.NewTaskService(repo, finder, exec)
	h := api.NewHandler(svc)

	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer()
	zynaxv1.RegisterTaskBrokerServiceServer(srv, h)
	t.Cleanup(func() { srv.GracefulStop() })
	go func() { _ = srv.Serve(lis) }()

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("bufconn dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	return &testEnv{client: zynaxv1.NewTaskBrokerServiceClient(conn), svc: svc}
}

func agentsWith(capability string) domain.AgentSelector {
	return &fakeFinder{agents: map[string][]domain.AgentInfo{
		capability: {{AgentID: "agent-1", Endpoint: "localhost:9000"}},
	}}
}

func validDispatchReq(workflowID, capName string) *zynaxv1.DispatchTaskRequest {
	return &zynaxv1.DispatchTaskRequest{
		Task: &zynaxv1.WorkflowTask{
			WorkflowId:     workflowID,
			CapabilityName: capName,
			InputPayload:   []byte(`{}`),
		},
	}
}

// ── DispatchTask ────────────────────────────────────────────────────────────

func TestDispatchTask_HappyPath(t *testing.T) {
	env := newTestEnv(t, agentsWith("summarize"), &fakeExecutor{})

	resp, err := env.client.DispatchTask(context.Background(), validDispatchReq("wf-1", "summarize"))
	if err != nil {
		t.Fatalf("DispatchTask: %v", err)
	}
	if resp.TaskId == "" {
		t.Error("want non-empty task_id")
	}
	if resp.CreatedAt == nil {
		t.Error("want non-nil created_at")
	}
}

func TestDispatchTask_MissingWorkflowID_InvalidArgument(t *testing.T) {
	env := newTestEnv(t, agentsWith("summarize"), &fakeExecutor{})

	_, err := env.client.DispatchTask(context.Background(), &zynaxv1.DispatchTaskRequest{
		Task: &zynaxv1.WorkflowTask{CapabilityName: "summarize", InputPayload: []byte(`{}`)},
	})
	if code := status.Code(err); code != codes.InvalidArgument {
		t.Errorf("want InvalidArgument, got %v", code)
	}
}

func TestDispatchTask_NoEligibleAgent_NotFound(t *testing.T) {
	env := newTestEnv(t, &fakeFinder{agents: map[string][]domain.AgentInfo{}}, &fakeExecutor{})

	_, err := env.client.DispatchTask(context.Background(), validDispatchReq("wf-2", "unknown-cap"))
	if code := status.Code(err); code != codes.NotFound {
		t.Errorf("want NotFound, got %v", code)
	}
}

func TestDispatchTask_FinderError_Internal(t *testing.T) {
	env := newTestEnv(t, &errorFinder{err: errors.New("db connection refused")}, &fakeExecutor{})

	_, err := env.client.DispatchTask(context.Background(), validDispatchReq("wf-3", "any-cap"))
	if code := status.Code(err); code != codes.Internal {
		t.Errorf("want Internal, got %v", code)
	}
}

// ── AcknowledgeTask ─────────────────────────────────────────────────────────

func TestAcknowledgeTask_WithTaskError(t *testing.T) {
	blocker := newBlockingExecutor()
	t.Cleanup(func() { close(blocker.release) })

	env := newTestEnv(t, agentsWith("summarize"), blocker)

	disp, err := env.client.DispatchTask(context.Background(), validDispatchReq("wf-ack-err", "summarize"))
	if err != nil {
		t.Fatalf("DispatchTask: %v", err)
	}
	<-blocker.ready

	resp, err := env.client.AcknowledgeTask(context.Background(), &zynaxv1.AcknowledgeTaskRequest{
		TaskId: disp.TaskId,
		Status: zynaxv1.TaskStatus_TASK_STATUS_FAILED,
		Error: &zynaxv1.TaskError{
			Code:    "EXEC_ERROR",
			Message: "something went wrong",
		},
	})
	if err != nil {
		t.Fatalf("AcknowledgeTask: %v", err)
	}
	if resp.ResultingStatus != zynaxv1.TaskStatus_TASK_STATUS_FAILED {
		t.Errorf("want FAILED, got %v", resp.ResultingStatus)
	}
}

func TestAcknowledgeTask_HappyPath(t *testing.T) {
	blocker := newBlockingExecutor()
	t.Cleanup(func() { close(blocker.release) })

	env := newTestEnv(t, agentsWith("summarize"), blocker)

	disp, err := env.client.DispatchTask(context.Background(), validDispatchReq("wf-ack", "summarize"))
	if err != nil {
		t.Fatalf("DispatchTask: %v", err)
	}
	<-blocker.ready // task is DISPATCHED; goroutine is blocked in Execute

	resp, err := env.client.AcknowledgeTask(context.Background(), &zynaxv1.AcknowledgeTaskRequest{
		TaskId:        disp.TaskId,
		Status:        zynaxv1.TaskStatus_TASK_STATUS_COMPLETED,
		ResultPayload: []byte(`{"ok":true}`),
	})
	if err != nil {
		t.Fatalf("AcknowledgeTask: %v", err)
	}
	if resp.ResultingStatus != zynaxv1.TaskStatus_TASK_STATUS_COMPLETED {
		t.Errorf("want COMPLETED, got %v", resp.ResultingStatus)
	}
}

// ── GetTask ──────────────────────────────────────────────────────────────────

func TestGetTask_HappyPath(t *testing.T) {
	env := newTestEnv(t, agentsWith("summarize"), &fakeExecutor{})

	disp, err := env.client.DispatchTask(context.Background(), validDispatchReq("wf-get", "summarize"))
	if err != nil {
		t.Fatalf("DispatchTask: %v", err)
	}

	got, err := env.client.GetTask(context.Background(), &zynaxv1.GetTaskRequest{TaskId: disp.TaskId})
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if got.TaskId != disp.TaskId {
		t.Errorf("task_id mismatch: want %q, got %q", disp.TaskId, got.TaskId)
	}
}

func TestGetTask_UnknownID_NotFound(t *testing.T) {
	env := newTestEnv(t, agentsWith("summarize"), &fakeExecutor{})

	_, err := env.client.GetTask(context.Background(), &zynaxv1.GetTaskRequest{TaskId: "task-does-not-exist"})
	if code := status.Code(err); code != codes.NotFound {
		t.Errorf("want NotFound, got %v", code)
	}
}

// ── ListTasks ────────────────────────────────────────────────────────────────

func TestListTasks_EmptyReturnsOK(t *testing.T) {
	env := newTestEnv(t, agentsWith("summarize"), &fakeExecutor{})

	resp, err := env.client.ListTasks(context.Background(), &zynaxv1.ListTasksRequest{})
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(resp.Tasks) != 0 {
		t.Errorf("want empty list, got %d tasks", len(resp.Tasks))
	}
}

// ── CancelTask ────────────────────────────────────────────────────────────────

func TestCancelTask_HappyPath(t *testing.T) {
	blocker := newBlockingExecutor()
	t.Cleanup(func() { close(blocker.release) })

	env := newTestEnv(t, agentsWith("summarize"), blocker)

	disp, err := env.client.DispatchTask(context.Background(), validDispatchReq("wf-cancel", "summarize"))
	if err != nil {
		t.Fatalf("DispatchTask: %v", err)
	}
	<-blocker.ready // task is DISPATCHED; goroutine is blocked

	resp, err := env.client.CancelTask(context.Background(), &zynaxv1.CancelTaskRequest{TaskId: disp.TaskId})
	if err != nil {
		t.Fatalf("CancelTask: %v", err)
	}
	if resp.CancelledAt == nil {
		t.Error("want non-nil cancelled_at")
	}
}

func TestCancelTask_TerminalTask_FailedPrecondition(t *testing.T) {
	env := newTestEnv(t, agentsWith("summarize"), &fakeExecutor{})

	disp, err := env.client.DispatchTask(context.Background(), validDispatchReq("wf-cancel-terminal", "summarize"))
	if err != nil {
		t.Fatalf("DispatchTask: %v", err)
	}
	// Wait for background execution to finish → task is COMPLETED (terminal).
	env.svc.WaitBackground()

	_, err = env.client.CancelTask(context.Background(), &zynaxv1.CancelTaskRequest{TaskId: disp.TaskId})
	if code := status.Code(err); code != codes.FailedPrecondition {
		t.Errorf("want FailedPrecondition, got %v", code)
	}
}
