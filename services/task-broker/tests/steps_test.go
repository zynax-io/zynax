// SPDX-License-Identifier: Apache-2.0

// Package task_broker_bdd_test contains service-level BDD tests for the task-broker.
// Tests exercise the real TaskService + memoryRepo over an in-process gRPC connection
// (bufconn) using fake AgentFinder and CapabilityExecutor port adapters.
package task_broker_bdd_test

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/cucumber/godog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/task-broker/internal/api"
	"github.com/zynax-io/zynax/services/task-broker/internal/domain"
	"github.com/zynax-io/zynax/services/task-broker/internal/infrastructure"
)

// ─── holdingRepo ──────────────────────────────────────────────────────────────
// Wraps a TaskRepository and optionally blocks Update() until released.
// Freezes the task in PENDING state while the test checks status after DispatchTask:
// the goroutine launched by DispatchTask sets DISPATCHED via Update() before calling
// Execute(), so holding Update() is the only way to observe PENDING reliably.

type holdingRepo struct {
	domain.TaskRepository
	mu   sync.Mutex
	hold chan struct{}
}

func newHoldingRepo(base domain.TaskRepository) *holdingRepo {
	return &holdingRepo{TaskRepository: base}
}

func (r *holdingRepo) startHold() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hold = make(chan struct{})
}

func (r *holdingRepo) release() {
	r.mu.Lock()
	h := r.hold
	r.hold = nil
	r.mu.Unlock()
	if h != nil {
		close(h)
	}
}

func (r *holdingRepo) Update(ctx context.Context, task *domain.Task) error {
	r.mu.Lock()
	h := r.hold
	r.mu.Unlock()
	if h != nil {
		select {
		case <-h:
		case <-ctx.Done():
			return ctx.Err() //nolint:wrapcheck
		}
	}
	return r.TaskRepository.Update(ctx, task) //nolint:wrapcheck
}

// ─── fakeAgentFinder ─────────────────────────────────────────────────────────

type fakeAgentFinder struct {
	mu     sync.Mutex
	agents map[string][]domain.AgentInfo
}

func newFakeAgentFinder() *fakeAgentFinder {
	return &fakeAgentFinder{agents: make(map[string][]domain.AgentInfo)}
}

func (f *fakeAgentFinder) add(capability, agentID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.agents[capability] = append(f.agents[capability], domain.AgentInfo{AgentID: agentID, Endpoint: "fake:///agent"})
}

// Select implements domain.AgentSelector with the reference semantics of the
// CRD-native scheduler: capability lookup, strict expert filter (no
// fallback), first eligible agent deterministically.
func (f *fakeAgentFinder) Select(_ context.Context, capName, expertTarget string) (domain.AgentInfo, error) {
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

// ─── fakeCapabilityExecutor ──────────────────────────────────────────────────

type fakeCapabilityExecutor struct{}

func (*fakeCapabilityExecutor) Execute(_ context.Context, _ domain.AgentInfo, _ *domain.Task) ([]byte, *domain.TaskError, error) {
	return []byte(`{"result":"ok"}`), nil, nil
}

// ─── testEnv ─────────────────────────────────────────────────────────────────

type testEnv struct {
	holdRepo   *holdingRepo
	finder     *fakeAgentFinder
	svc        *domain.TaskService
	srv        *grpc.Server
	lis        *bufconn.Listener
	conn       *grpc.ClientConn
	client     zynaxv1.TaskBrokerServiceClient
	lastTaskID string
	lastErr    error
}

func (e *testEnv) setup() error {
	e.holdRepo = newHoldingRepo(infrastructure.NewMemoryRepo())
	e.finder = newFakeAgentFinder()
	e.svc = domain.NewTaskService(e.holdRepo, e.finder, &fakeCapabilityExecutor{})

	const bufSize = 1 << 20
	e.lis = bufconn.Listen(bufSize)
	e.srv = grpc.NewServer()
	zynaxv1.RegisterTaskBrokerServiceServer(e.srv, api.NewHandler(e.svc))
	go func() { _ = e.srv.Serve(e.lis) }()

	conn, err := grpc.NewClient(
		"passthrough://bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return e.lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return err //nolint:wrapcheck
	}
	e.conn = conn
	e.client = zynaxv1.NewTaskBrokerServiceClient(conn)
	return nil
}

func (e *testEnv) stop() {
	e.holdRepo.release()
	e.svc.WaitBackground()
	e.srv.GracefulStop()
	_ = e.conn.Close() //nolint:errcheck
	_ = e.lis.Close()  //nolint:errcheck
}

func (e *testEnv) insertTask(taskID, workflowID string, s domain.TaskStatus, maxRetries int32) error {
	return e.holdRepo.Save(context.Background(), &domain.Task{ //nolint:wrapcheck
		TaskID:         taskID,
		WorkflowID:     workflowID,
		CapabilityName: "summarize",
		InputPayload:   []byte(`{}`),
		Status:         s,
		MaxRetries:     maxRetries,
		CreatedAt:      time.Now(),
	})
}

func (e *testEnv) expectStatus(ctx context.Context, taskID string, expected zynaxv1.TaskStatus) error {
	callCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	task, err := e.client.GetTask(callCtx, &zynaxv1.GetTaskRequest{TaskId: taskID})
	if err != nil {
		return fmt.Errorf("GetTask(%q): %w", taskID, err)
	}
	if task.Status != expected {
		return fmt.Errorf("expected %v, got %v", expected, task.Status)
	}
	return nil
}

// ─── TestFeatures ─────────────────────────────────────────────────────────────

//nolint:cyclop,funlen
func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: func(sc *godog.ScenarioContext) {
			var env *testEnv

			sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
				env = &testEnv{}
				return ctx, nil
			})
			sc.After(func(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
				if env.svc != nil {
					env.stop()
				}
				return ctx, nil
			})

			sc.Step(`^the task broker service is running$`, func(ctx context.Context) (context.Context, error) {
				return ctx, env.setup()
			})
			sc.Step(`^agent "([^"]*)" handles capability "([^"]*)"$`, func(ctx context.Context, agentID, capName string) (context.Context, error) {
				env.finder.add(capName, agentID)
				return ctx, nil
			})
			sc.Step(`^the repo holds updates until released$`, func(ctx context.Context) (context.Context, error) {
				env.holdRepo.startHold()
				return ctx, nil
			})
			sc.Step(`^I dispatch a task with capability "([^"]*)" for workflow "([^"]*)"$`, func(ctx context.Context, capName, wfID string) (context.Context, error) {
				callCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
				defer cancel()
				resp, err := env.client.DispatchTask(callCtx, &zynaxv1.DispatchTaskRequest{
					Task: &zynaxv1.WorkflowTask{WorkflowId: wfID, CapabilityName: capName, InputPayload: []byte(`{}`)},
				})
				env.lastErr = err
				if err == nil {
					env.lastTaskID = resp.TaskId
				}
				return ctx, nil
			})
			sc.Step(`^the response contains a non-empty task_id$`, func(ctx context.Context) (context.Context, error) {
				if env.lastErr != nil {
					return ctx, fmt.Errorf("DispatchTask: %w", env.lastErr)
				}
				if env.lastTaskID == "" {
					return ctx, fmt.Errorf("task_id is empty")
				}
				return ctx, nil
			})
			sc.Step(`^GetTask returns status PENDING$`, func(ctx context.Context) (context.Context, error) {
				return ctx, env.expectStatus(ctx, env.lastTaskID, zynaxv1.TaskStatus_TASK_STATUS_PENDING)
			})
			sc.Step(`^a task "([^"]*)" in DISPATCHED state for workflow "([^"]*)"$`, func(ctx context.Context, taskID, wfID string) (context.Context, error) {
				return ctx, env.insertTask(taskID, wfID, domain.TaskStatusDispatched, 0)
			})
			sc.Step(`^a task "([^"]*)" in DISPATCHED state for workflow "([^"]*)" with max_retries (\d+)$`, func(ctx context.Context, taskID, wfID string, maxRetries int) (context.Context, error) {
				return ctx, env.insertTask(taskID, wfID, domain.TaskStatusDispatched, int32(maxRetries)) //nolint:gosec
			})
			sc.Step(`^a task "([^"]*)" in PENDING state for workflow "([^"]*)"$`, func(ctx context.Context, taskID, wfID string) (context.Context, error) {
				return ctx, env.insertTask(taskID, wfID, domain.TaskStatusPending, 0)
			})
			sc.Step(`^AcknowledgeTask is called with status COMPLETED and a valid result for task "([^"]*)"$`, func(ctx context.Context, taskID string) (context.Context, error) {
				callCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
				defer cancel()
				_, err := env.client.AcknowledgeTask(callCtx, &zynaxv1.AcknowledgeTaskRequest{
					TaskId:        taskID,
					Status:        zynaxv1.TaskStatus_TASK_STATUS_COMPLETED,
					ResultPayload: []byte(`{"result":"ok"}`),
				})
				env.lastErr = err
				return ctx, nil
			})
			sc.Step(`^AcknowledgeTask is called with status FAILED for task "([^"]*)"$`, func(ctx context.Context, taskID string) (context.Context, error) {
				callCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
				defer cancel()
				_, err := env.client.AcknowledgeTask(callCtx, &zynaxv1.AcknowledgeTaskRequest{
					TaskId: taskID,
					Status: zynaxv1.TaskStatus_TASK_STATUS_FAILED,
					Error:  &zynaxv1.TaskError{Code: "ERR_INTERNAL", Message: "simulated failure"},
				})
				env.lastErr = err
				return ctx, nil
			})
			sc.Step(`^GetTask for "([^"]*)" returns status COMPLETED$`, func(ctx context.Context, taskID string) (context.Context, error) {
				return ctx, env.expectStatus(ctx, taskID, zynaxv1.TaskStatus_TASK_STATUS_COMPLETED)
			})
			sc.Step(`^GetTask for "([^"]*)" returns status RETRYING$`, func(ctx context.Context, taskID string) (context.Context, error) {
				return ctx, env.expectStatus(ctx, taskID, zynaxv1.TaskStatus_TASK_STATUS_RETRYING)
			})
			sc.Step(`^GetTask for "([^"]*)" returns status CANCELLED$`, func(ctx context.Context, taskID string) (context.Context, error) {
				return ctx, env.expectStatus(ctx, taskID, zynaxv1.TaskStatus_TASK_STATUS_CANCELLED)
			})
			sc.Step(`^CancelTask is called for task "([^"]*)"$`, func(ctx context.Context, taskID string) (context.Context, error) {
				callCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
				defer cancel()
				_, err := env.client.CancelTask(callCtx, &zynaxv1.CancelTaskRequest{TaskId: taskID})
				env.lastErr = err
				return ctx, nil
			})
			sc.Step(`^GetTask is called for task_id "([^"]*)"$`, func(ctx context.Context, taskID string) (context.Context, error) {
				callCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
				defer cancel()
				_, err := env.client.GetTask(callCtx, &zynaxv1.GetTaskRequest{TaskId: taskID})
				env.lastErr = err
				return ctx, nil
			})
			sc.Step(`^the error code is NOT_FOUND$`, func(ctx context.Context) (context.Context, error) {
				if env.lastErr == nil {
					return ctx, fmt.Errorf("expected an error, got nil")
				}
				st, _ := status.FromError(env.lastErr)
				if st.Code() != codes.NotFound {
					return ctx, fmt.Errorf("expected NOT_FOUND, got %v", st.Code())
				}
				return ctx, nil
			})
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"features"},
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}
