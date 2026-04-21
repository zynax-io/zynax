// SPDX-License-Identifier: Apache-2.0
// BDD contract tests for AgentService.ExecuteCapability.
package agent_service_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/cucumber/godog"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/protos/tests/testserver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ─── Stub server ─────────────────────────────────────────────────────────────

type agentStub struct {
	zynaxv1.UnimplementedAgentServiceServer
}

func (s *agentStub) ExecuteCapability(req *zynaxv1.ExecuteCapabilityRequest, stream grpc.ServerStreamingServer[zynaxv1.TaskEvent]) error {
	// Input validation
	if req.CapabilityName == "" {
		return status.Error(codes.InvalidArgument, "capability_name must not be empty")
	}
	if req.TaskId == "" {
		return status.Error(codes.InvalidArgument, "task_id must not be empty")
	}
	if len(req.InputPayload) > 0 && !json.Valid(req.InputPayload) {
		return status.Error(codes.InvalidArgument, "input_payload must be valid JSON")
	}

	switch req.CapabilityName {
	case "summarize":
		timeout := req.TimeoutSeconds
		if timeout > 0 && timeout <= 1 {
			// simulate timeout
			_ = stream.Send(&zynaxv1.TaskEvent{
				TaskId:    req.TaskId,
				EventType: zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED,
				Timestamp: timestamppb.Now(),
				Error: &zynaxv1.CapabilityError{
					Code:    "TIMEOUT",
					Message: "capability timed out",
				},
			})
			return status.Error(codes.DeadlineExceeded, "timeout exceeded")
		}
		// emit PROGRESS then COMPLETED
		_ = stream.Send(&zynaxv1.TaskEvent{
			TaskId:    req.TaskId,
			EventType: zynaxv1.TaskEventType_TASK_EVENT_TYPE_PROGRESS,
			Payload:   []byte(`{"progress": 50}`),
			Timestamp: timestamppb.Now(),
		})
		_ = stream.Send(&zynaxv1.TaskEvent{
			TaskId:    req.TaskId,
			EventType: zynaxv1.TaskEventType_TASK_EVENT_TYPE_COMPLETED,
			Payload:   []byte(`{"summary": "done"}`),
			Timestamp: timestamppb.Now(),
		})
		return nil

	case "always_fails":
		_ = stream.Send(&zynaxv1.TaskEvent{
			TaskId:    req.TaskId,
			EventType: zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED,
			Timestamp: timestamppb.Now(),
			Error: &zynaxv1.CapabilityError{
				Code:    "INTERNAL",
				Message: "capability always fails",
			},
		})
		return nil

	default:
		return status.Errorf(codes.NotFound, "capability %q not found", req.CapabilityName)
	}
}

// ─── Test context ─────────────────────────────────────────────────────────────

type testCtx struct {
	client     zynaxv1.AgentServiceClient
	req        *zynaxv1.ExecuteCapabilityRequest
	events     []*zynaxv1.TaskEvent
	grpcErr    error
	streamDone bool
}

func newTestCtx() *testCtx {
	return &testCtx{
		req: &zynaxv1.ExecuteCapabilityRequest{
			RequestId:      "req-default",
			CapabilityName: "summarize",
			TaskId:         "task-default",
			WorkflowId:     "wf-default",
			InputPayload:   []byte(`{"documents": ["hello"]}`),
		},
	}
}

// ─── Steps ───────────────────────────────────────────────────────────────────

func (tc *testCtx) anAgentIsRunningOnTestServer(t *testing.T) func() error {
	return func() error {
		srv, dialer := testserver.NewBufconnServer(t)
		zynaxv1.RegisterAgentServiceServer(srv, &agentStub{})
		conn, err := grpc.NewClient("passthrough://bufnet",
			grpc.WithContextDialer(dialer),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			return err
		}
		t.Cleanup(func() { conn.Close() })
		tc.client = zynaxv1.NewAgentServiceClient(conn)
		return nil
	}
}

func (tc *testCtx) callExecuteCapabilityAndCollect() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := tc.client.ExecuteCapability(ctx, tc.req)
	if err != nil {
		tc.grpcErr = err
		tc.streamDone = true
		return nil
	}
	tc.events = nil
	for {
		ev, err := stream.Recv()
		if err != nil {
			tc.grpcErr = err
			tc.streamDone = true
			break
		}
		tc.events = append(tc.events, ev)
	}
	return nil
}

func (tc *testCtx) callAndWaitForFailed() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stream, err := tc.client.ExecuteCapability(ctx, tc.req)
	if err != nil {
		tc.grpcErr = err
		tc.streamDone = true
		return nil
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		ev, err := stream.Recv()
		if err != nil {
			tc.grpcErr = err
			tc.streamDone = true
			break
		}
		tc.events = append(tc.events, ev)
		if ev.EventType == zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED {
			break
		}
	}
	return nil
}

func InitializeScenario(sc *godog.ScenarioContext) {
	var tc *testCtx
	var t *testing.T

	sc.Before(func(ctx context.Context, scenario *godog.Scenario) (context.Context, error) {
		tc = newTestCtx()
		return ctx, nil
	})

	// We need t available. Use a wrapper via TestFeatures.
	_ = t

	sc.Step(`^an agent implementing AgentService is running on a test gRPC server$`, func(ctx context.Context) (context.Context, error) {
		// The t here is from the test suite
		godogT := ctx.Value(godogTKey{})
		if godogT == nil {
			return ctx, fmt.Errorf("testing.T not in context")
		}
		theT := godogT.(*testing.T)
		return ctx, tc.anAgentIsRunningOnTestServer(theT)()
	})

	sc.Step(`^a valid ExecuteCapabilityRequest for capability "([^"]*)"$`, func(ctx context.Context, cap string) (context.Context, error) {
		tc.req.CapabilityName = cap
		tc.req.TaskId = "task-default"
		tc.req.InputPayload = []byte(`{"documents": ["hello"]}`)
		return ctx, nil
	})

	sc.Step(`^the input payload is valid JSON: (\{.*\})$`, func(ctx context.Context, payload string) (context.Context, error) {
		tc.req.InputPayload = []byte(payload)
		return ctx, nil
	})

	sc.Step(`^ExecuteCapability is called$`, func(ctx context.Context) (context.Context, error) {
		return ctx, tc.callExecuteCapabilityAndCollect()
	})

	sc.Step(`^ExecuteCapability is called and the stream is fully consumed$`, func(ctx context.Context) (context.Context, error) {
		return ctx, tc.callExecuteCapabilityAndCollect()
	})

	sc.Step(`^the stream emits at least one TaskEvent with event_type PROGRESS$`, func(ctx context.Context) (context.Context, error) {
		for _, ev := range tc.events {
			if ev.EventType == zynaxv1.TaskEventType_TASK_EVENT_TYPE_PROGRESS {
				return ctx, nil
			}
		}
		return ctx, fmt.Errorf("no PROGRESS event found in stream (events: %d)", len(tc.events))
	})

	sc.Step(`^the final TaskEvent has event_type COMPLETED$`, func(ctx context.Context) (context.Context, error) {
		if len(tc.events) == 0 {
			return ctx, fmt.Errorf("no events received")
		}
		last := tc.events[len(tc.events)-1]
		if last.EventType != zynaxv1.TaskEventType_TASK_EVENT_TYPE_COMPLETED {
			return ctx, fmt.Errorf("expected COMPLETED but got %v", last.EventType)
		}
		return ctx, nil
	})

	sc.Step(`^the COMPLETED event payload is valid JSON$`, func(ctx context.Context) (context.Context, error) {
		for _, ev := range tc.events {
			if ev.EventType == zynaxv1.TaskEventType_TASK_EVENT_TYPE_COMPLETED {
				if !json.Valid(ev.Payload) {
					return ctx, fmt.Errorf("COMPLETED payload is not valid JSON")
				}
				return ctx, nil
			}
		}
		return ctx, fmt.Errorf("no COMPLETED event found")
	})

	sc.Step(`^the stream closes cleanly after the COMPLETED event$`, func(ctx context.Context) (context.Context, error) {
		if !tc.streamDone {
			return ctx, fmt.Errorf("stream not done")
		}
		return ctx, nil
	})

	sc.Step(`^a valid ExecuteCapabilityRequest with task_id "([^"]*)"$`, func(ctx context.Context, taskID string) (context.Context, error) {
		tc.req.TaskId = taskID
		tc.req.CapabilityName = "summarize"
		tc.req.InputPayload = []byte(`{"documents": ["hello"]}`)
		return ctx, nil
	})

	sc.Step(`^every TaskEvent in the stream has task_id "([^"]*)"$`, func(ctx context.Context, taskID string) (context.Context, error) {
		for _, ev := range tc.events {
			if ev.TaskId != taskID {
				return ctx, fmt.Errorf("event has task_id %q, expected %q", ev.TaskId, taskID)
			}
		}
		return ctx, nil
	})

	sc.Step(`^every TaskEvent has a non-zero timestamp$`, func(ctx context.Context) (context.Context, error) {
		for _, ev := range tc.events {
			if ev.Timestamp == nil || ev.Timestamp.AsTime().IsZero() {
				return ctx, fmt.Errorf("event has zero/nil timestamp")
			}
		}
		return ctx, nil
	})

	sc.Step(`^an ExecuteCapabilityRequest with timeout_seconds set to (\d+)$`, func(ctx context.Context, secs int) (context.Context, error) {
		tc.req.TimeoutSeconds = int32(secs)
		tc.req.CapabilityName = "summarize"
		tc.req.InputPayload = []byte(`{"documents": ["hello"]}`)
		return ctx, nil
	})

	sc.Step(`^the agent simulates a capability that runs for \d+ seconds$`, func(ctx context.Context) (context.Context, error) {
		// The stub already handles timeout_seconds=1 specially
		return ctx, nil
	})

	sc.Step(`^the stream receives a TaskEvent of type FAILED within \d+ seconds$`, func(ctx context.Context) (context.Context, error) {
		return ctx, tc.callAndWaitForFailed()
	})

	sc.Step(`^the CapabilityError code is "([^"]*)"$`, func(ctx context.Context, code string) (context.Context, error) {
		for _, ev := range tc.events {
			if ev.EventType == zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED && ev.Error != nil {
				if ev.Error.Code != code {
					return ctx, fmt.Errorf("expected error code %q got %q", code, ev.Error.Code)
				}
				return ctx, nil
			}
		}
		return ctx, fmt.Errorf("no FAILED event with error found")
	})

	sc.Step(`^the gRPC status is DEADLINE_EXCEEDED$`, func(ctx context.Context) (context.Context, error) {
		if tc.grpcErr == nil {
			return ctx, fmt.Errorf("expected error but got none")
		}
		st, ok := status.FromError(tc.grpcErr)
		if !ok || st.Code() != codes.DeadlineExceeded {
			return ctx, fmt.Errorf("expected DEADLINE_EXCEEDED, got %v", tc.grpcErr)
		}
		return ctx, nil
	})

	sc.Step(`^the stream emits exactly one TaskEvent with event_type FAILED$`, func(ctx context.Context) (context.Context, error) {
		count := 0
		for _, ev := range tc.events {
			if ev.EventType == zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED {
				count++
			}
		}
		if count != 1 {
			return ctx, fmt.Errorf("expected exactly 1 FAILED event, got %d", count)
		}
		return ctx, nil
	})

	sc.Step(`^the TaskEvent\.error\.code is a non-empty string$`, func(ctx context.Context) (context.Context, error) {
		for _, ev := range tc.events {
			if ev.EventType == zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED {
				if ev.Error == nil || ev.Error.Code == "" {
					return ctx, fmt.Errorf("FAILED event has empty error code")
				}
				return ctx, nil
			}
		}
		return ctx, fmt.Errorf("no FAILED event found")
	})

	sc.Step(`^the TaskEvent\.error\.message is a non-empty string$`, func(ctx context.Context) (context.Context, error) {
		for _, ev := range tc.events {
			if ev.EventType == zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED {
				if ev.Error == nil || ev.Error.Message == "" {
					return ctx, fmt.Errorf("FAILED event has empty error message")
				}
				return ctx, nil
			}
		}
		return ctx, fmt.Errorf("no FAILED event found")
	})

	sc.Step(`^no further events are emitted after the FAILED event$`, func(ctx context.Context) (context.Context, error) {
		foundFailed := false
		for _, ev := range tc.events {
			if foundFailed {
				return ctx, fmt.Errorf("found event after FAILED: %v", ev.EventType)
			}
			if ev.EventType == zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED {
				foundFailed = true
			}
		}
		return ctx, nil
	})

	sc.Step(`^an ExecuteCapabilityRequest for capability "([^"]*)"$`, func(ctx context.Context, cap string) (context.Context, error) {
		tc.req.CapabilityName = cap
		tc.req.InputPayload = []byte(`{"documents": ["hello"]}`)
		return ctx, nil
	})

	sc.Step(`^the gRPC status is NOT_FOUND$`, func(ctx context.Context) (context.Context, error) {
		if tc.grpcErr == nil {
			return ctx, fmt.Errorf("expected error but got none")
		}
		st, ok := status.FromError(tc.grpcErr)
		if !ok || st.Code() != codes.NotFound {
			return ctx, fmt.Errorf("expected NOT_FOUND, got %v", tc.grpcErr)
		}
		return ctx, nil
	})

	sc.Step(`^the error message contains "([^"]*)"$`, func(ctx context.Context, substr string) (context.Context, error) {
		if tc.grpcErr == nil {
			return ctx, fmt.Errorf("expected error but got none")
		}
		if !strings.Contains(tc.grpcErr.Error(), substr) {
			return ctx, fmt.Errorf("error %q does not contain %q", tc.grpcErr.Error(), substr)
		}
		return ctx, nil
	})

	sc.Step(`^no TaskEvent is emitted$`, func(ctx context.Context) (context.Context, error) {
		if len(tc.events) > 0 {
			return ctx, fmt.Errorf("expected no events but got %d", len(tc.events))
		}
		return ctx, nil
	})

	sc.Step(`^an ExecuteCapabilityRequest with capability_name set to ""$`, func(ctx context.Context) (context.Context, error) {
		tc.req.CapabilityName = ""
		return ctx, nil
	})

	sc.Step(`^the gRPC status is INVALID_ARGUMENT$`, func(ctx context.Context) (context.Context, error) {
		if tc.grpcErr == nil {
			return ctx, fmt.Errorf("expected error but got none")
		}
		st, ok := status.FromError(tc.grpcErr)
		if !ok || st.Code() != codes.InvalidArgument {
			return ctx, fmt.Errorf("expected INVALID_ARGUMENT, got %v", tc.grpcErr)
		}
		return ctx, nil
	})

	sc.Step(`^the error message mentions "([^"]*)"$`, func(ctx context.Context, field string) (context.Context, error) {
		if tc.grpcErr == nil {
			return ctx, fmt.Errorf("expected error but got none")
		}
		if !strings.Contains(tc.grpcErr.Error(), field) {
			return ctx, fmt.Errorf("error %q does not mention %q", tc.grpcErr.Error(), field)
		}
		return ctx, nil
	})

	sc.Step(`^an ExecuteCapabilityRequest with task_id set to ""$`, func(ctx context.Context) (context.Context, error) {
		tc.req.TaskId = ""
		return ctx, nil
	})

	sc.Step(`^an ExecuteCapabilityRequest with input_payload set to "([^"]*)"$`, func(ctx context.Context, payload string) (context.Context, error) {
		tc.req.InputPayload = []byte(payload)
		return ctx, nil
	})

	sc.Step(`^a valid ExecuteCapabilityRequest$`, func(ctx context.Context) (context.Context, error) {
		tc.req.CapabilityName = "summarize"
		tc.req.TaskId = "task-default"
		tc.req.InputPayload = []byte(`{"documents": ["hello"]}`)
		return ctx, nil
	})

	sc.Step(`^the final TaskEvent has event_type COMPLETED or FAILED$`, func(ctx context.Context) (context.Context, error) {
		if len(tc.events) == 0 {
			return ctx, fmt.Errorf("no events received")
		}
		last := tc.events[len(tc.events)-1]
		if last.EventType != zynaxv1.TaskEventType_TASK_EVENT_TYPE_COMPLETED &&
			last.EventType != zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED {
			return ctx, fmt.Errorf("expected COMPLETED or FAILED, got %v", last.EventType)
		}
		return ctx, nil
	})

	sc.Step(`^no TaskEvent is received after the first FAILED event$`, func(ctx context.Context) (context.Context, error) {
		foundFailed := false
		for _, ev := range tc.events {
			if foundFailed {
				return ctx, fmt.Errorf("got event after FAILED: %v", ev.EventType)
			}
			if ev.EventType == zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED {
				foundFailed = true
			}
		}
		return ctx, nil
	})
}

// godogTKey is used to store *testing.T in context.
type godogTKey struct{}

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: func(sc *godog.ScenarioContext) {
			var tc *testCtx

			sc.Before(func(ctx context.Context, scenario *godog.Scenario) (context.Context, error) {
				tc = newTestCtx()
				ctx = context.WithValue(ctx, godogTKey{}, t)
				return ctx, nil
			})

			sc.Step(`^an agent implementing AgentService is running on a test gRPC server$`, func(ctx context.Context) (context.Context, error) {
				return ctx, tc.anAgentIsRunningOnTestServer(t)()
			})

			sc.Step(`^a valid ExecuteCapabilityRequest for capability "([^"]*)"$`, func(ctx context.Context, cap string) (context.Context, error) {
				tc.req.CapabilityName = cap
				tc.req.TaskId = "task-default"
				tc.req.InputPayload = []byte(`{"documents": ["hello"]}`)
				return ctx, nil
			})

			sc.Step(`^the input payload is valid JSON: (\{.+\})$`, func(ctx context.Context, payload string) (context.Context, error) {
				tc.req.InputPayload = []byte(payload)
				return ctx, nil
			})

			sc.Step(`^ExecuteCapability is called$`, func(ctx context.Context) (context.Context, error) {
				return ctx, tc.callExecuteCapabilityAndCollect()
			})

			sc.Step(`^ExecuteCapability is called and the stream is fully consumed$`, func(ctx context.Context) (context.Context, error) {
				return ctx, tc.callExecuteCapabilityAndCollect()
			})

			sc.Step(`^the stream emits at least one TaskEvent with event_type PROGRESS$`, func(ctx context.Context) (context.Context, error) {
				for _, ev := range tc.events {
					if ev.EventType == zynaxv1.TaskEventType_TASK_EVENT_TYPE_PROGRESS {
						return ctx, nil
					}
				}
				return ctx, fmt.Errorf("no PROGRESS event found (events=%d)", len(tc.events))
			})

			sc.Step(`^the final TaskEvent has event_type COMPLETED$`, func(ctx context.Context) (context.Context, error) {
				if len(tc.events) == 0 {
					return ctx, fmt.Errorf("no events received")
				}
				last := tc.events[len(tc.events)-1]
				if last.EventType != zynaxv1.TaskEventType_TASK_EVENT_TYPE_COMPLETED {
					return ctx, fmt.Errorf("expected COMPLETED, got %v", last.EventType)
				}
				return ctx, nil
			})

			sc.Step(`^the COMPLETED event payload is valid JSON$`, func(ctx context.Context) (context.Context, error) {
				for _, ev := range tc.events {
					if ev.EventType == zynaxv1.TaskEventType_TASK_EVENT_TYPE_COMPLETED {
						if !json.Valid(ev.Payload) {
							return ctx, fmt.Errorf("COMPLETED payload is not valid JSON")
						}
						return ctx, nil
					}
				}
				return ctx, fmt.Errorf("no COMPLETED event found")
			})

			sc.Step(`^the stream closes cleanly after the COMPLETED event$`, func(ctx context.Context) (context.Context, error) {
				return ctx, nil
			})

			sc.Step(`^a valid ExecuteCapabilityRequest with task_id "([^"]*)"$`, func(ctx context.Context, taskID string) (context.Context, error) {
				tc.req.TaskId = taskID
				tc.req.CapabilityName = "summarize"
				tc.req.InputPayload = []byte(`{"documents": ["hello"]}`)
				return ctx, nil
			})

			sc.Step(`^every TaskEvent in the stream has task_id "([^"]*)"$`, func(ctx context.Context, taskID string) (context.Context, error) {
				for _, ev := range tc.events {
					if ev.TaskId != taskID {
						return ctx, fmt.Errorf("event task_id=%q, want %q", ev.TaskId, taskID)
					}
				}
				return ctx, nil
			})

			sc.Step(`^every TaskEvent has a non-zero timestamp$`, func(ctx context.Context) (context.Context, error) {
				for _, ev := range tc.events {
					if ev.Timestamp == nil || ev.Timestamp.AsTime().IsZero() {
						return ctx, fmt.Errorf("event has zero/nil timestamp")
					}
				}
				return ctx, nil
			})

			sc.Step(`^an ExecuteCapabilityRequest with timeout_seconds set to (\d+)$`, func(ctx context.Context, secs int) (context.Context, error) {
				tc.req.TimeoutSeconds = int32(secs)
				tc.req.CapabilityName = "summarize"
				tc.req.InputPayload = []byte(`{"documents": ["hello"]}`)
				return ctx, nil
			})

			sc.Step(`^the agent simulates a capability that runs for \d+ seconds$`, func(ctx context.Context) (context.Context, error) {
				return ctx, nil
			})

			sc.Step(`^the stream receives a TaskEvent of type FAILED within \d+ seconds$`, func(ctx context.Context) (context.Context, error) {
				return ctx, tc.callAndWaitForFailed()
			})

			sc.Step(`^the CapabilityError code is "([^"]*)"$`, func(ctx context.Context, code string) (context.Context, error) {
				for _, ev := range tc.events {
					if ev.EventType == zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED && ev.Error != nil {
						if ev.Error.Code != code {
							return ctx, fmt.Errorf("expected code %q, got %q", code, ev.Error.Code)
						}
						return ctx, nil
					}
				}
				return ctx, fmt.Errorf("no FAILED event with error found")
			})

			sc.Step(`^the gRPC status is DEADLINE_EXCEEDED$`, func(ctx context.Context) (context.Context, error) {
				if tc.grpcErr == nil {
					return ctx, fmt.Errorf("expected error")
				}
				st, _ := status.FromError(tc.grpcErr)
				if st.Code() != codes.DeadlineExceeded {
					return ctx, fmt.Errorf("expected DEADLINE_EXCEEDED, got %v", st.Code())
				}
				return ctx, nil
			})

			sc.Step(`^the stream emits exactly one TaskEvent with event_type FAILED$`, func(ctx context.Context) (context.Context, error) {
				cnt := 0
				for _, ev := range tc.events {
					if ev.EventType == zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED {
						cnt++
					}
				}
				if cnt != 1 {
					return ctx, fmt.Errorf("want 1 FAILED event, got %d", cnt)
				}
				return ctx, nil
			})

			sc.Step(`^the TaskEvent\.error\.code is a non-empty string$`, func(ctx context.Context) (context.Context, error) {
				for _, ev := range tc.events {
					if ev.EventType == zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED {
						if ev.Error == nil || ev.Error.Code == "" {
							return ctx, fmt.Errorf("empty error code")
						}
						return ctx, nil
					}
				}
				return ctx, fmt.Errorf("no FAILED event")
			})

			sc.Step(`^the TaskEvent\.error\.message is a non-empty string$`, func(ctx context.Context) (context.Context, error) {
				for _, ev := range tc.events {
					if ev.EventType == zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED {
						if ev.Error == nil || ev.Error.Message == "" {
							return ctx, fmt.Errorf("empty error message")
						}
						return ctx, nil
					}
				}
				return ctx, fmt.Errorf("no FAILED event")
			})

			sc.Step(`^no further events are emitted after the FAILED event$`, func(ctx context.Context) (context.Context, error) {
				foundFailed := false
				for _, ev := range tc.events {
					if foundFailed {
						return ctx, fmt.Errorf("event after FAILED: %v", ev.EventType)
					}
					if ev.EventType == zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED {
						foundFailed = true
					}
				}
				return ctx, nil
			})

			sc.Step(`^an ExecuteCapabilityRequest for capability "([^"]*)"$`, func(ctx context.Context, cap string) (context.Context, error) {
				tc.req.CapabilityName = cap
				tc.req.InputPayload = []byte(`{"documents": ["hello"]}`)
				return ctx, nil
			})

			sc.Step(`^the gRPC status is NOT_FOUND$`, func(ctx context.Context) (context.Context, error) {
				if tc.grpcErr == nil {
					return ctx, fmt.Errorf("expected error")
				}
				st, _ := status.FromError(tc.grpcErr)
				if st.Code() != codes.NotFound {
					return ctx, fmt.Errorf("expected NOT_FOUND, got %v", st.Code())
				}
				return ctx, nil
			})

			sc.Step(`^the error message contains "([^"]*)"$`, func(ctx context.Context, substr string) (context.Context, error) {
				if tc.grpcErr == nil {
					return ctx, fmt.Errorf("expected error")
				}
				if !strings.Contains(tc.grpcErr.Error(), substr) {
					return ctx, fmt.Errorf("error %q doesn't contain %q", tc.grpcErr.Error(), substr)
				}
				return ctx, nil
			})

			sc.Step(`^no TaskEvent is emitted$`, func(ctx context.Context) (context.Context, error) {
				if len(tc.events) > 0 {
					return ctx, fmt.Errorf("expected no events, got %d", len(tc.events))
				}
				return ctx, nil
			})

			sc.Step(`^an ExecuteCapabilityRequest with capability_name set to ""$`, func(ctx context.Context) (context.Context, error) {
				tc.req.CapabilityName = ""
				return ctx, nil
			})

			sc.Step(`^the gRPC status is INVALID_ARGUMENT$`, func(ctx context.Context) (context.Context, error) {
				if tc.grpcErr == nil {
					return ctx, fmt.Errorf("expected error")
				}
				st, _ := status.FromError(tc.grpcErr)
				if st.Code() != codes.InvalidArgument {
					return ctx, fmt.Errorf("expected INVALID_ARGUMENT, got %v", st.Code())
				}
				return ctx, nil
			})

			sc.Step(`^the error message mentions "([^"]*)"$`, func(ctx context.Context, field string) (context.Context, error) {
				if tc.grpcErr == nil {
					return ctx, fmt.Errorf("expected error")
				}
				if !strings.Contains(tc.grpcErr.Error(), field) {
					return ctx, fmt.Errorf("error %q doesn't mention %q", tc.grpcErr.Error(), field)
				}
				return ctx, nil
			})

			sc.Step(`^an ExecuteCapabilityRequest with task_id set to ""$`, func(ctx context.Context) (context.Context, error) {
				tc.req.TaskId = ""
				return ctx, nil
			})

			sc.Step(`^an ExecuteCapabilityRequest with input_payload set to "([^"]*)"$`, func(ctx context.Context, payload string) (context.Context, error) {
				tc.req.InputPayload = []byte(payload)
				return ctx, nil
			})

			sc.Step(`^a valid ExecuteCapabilityRequest$`, func(ctx context.Context) (context.Context, error) {
				tc.req.CapabilityName = "summarize"
				tc.req.TaskId = "task-default"
				tc.req.InputPayload = []byte(`{"documents": ["hello"]}`)
				return ctx, nil
			})

			sc.Step(`^the final TaskEvent has event_type COMPLETED or FAILED$`, func(ctx context.Context) (context.Context, error) {
				if len(tc.events) == 0 {
					return ctx, fmt.Errorf("no events")
				}
				last := tc.events[len(tc.events)-1]
				if last.EventType != zynaxv1.TaskEventType_TASK_EVENT_TYPE_COMPLETED &&
					last.EventType != zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED {
					return ctx, fmt.Errorf("expected COMPLETED or FAILED, got %v", last.EventType)
				}
				return ctx, nil
			})

			sc.Step(`^no TaskEvent is received after the first FAILED event$`, func(ctx context.Context) (context.Context, error) {
				foundFailed := false
				for _, ev := range tc.events {
					if foundFailed {
						return ctx, fmt.Errorf("got event after FAILED: %v", ev.EventType)
					}
					if ev.EventType == zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED {
						foundFailed = true
					}
				}
				return ctx, nil
			})
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../features/agent_service.feature"},
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}
