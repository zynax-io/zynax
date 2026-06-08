// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/event-bus/internal/api"
	"github.com/zynax-io/zynax/services/event-bus/internal/domain"
)

// Shared test constants to satisfy the goconst linter threshold.
const (
	testWorkflowID      = "wf-99"
	testRunID           = "run-5"
	testNamespace       = "prod"
	testCapabilityName  = "echo"
	testDataContentType = "application/json"
)

// fakeEventBus is an in-memory test double for domain.EventBus.
// It records published events and allows callers to inject errors.
type fakeEventBus struct {
	publishErr error
	published  []domain.CloudEvent
	returnID   string
}

func (f *fakeEventBus) Publish(_ context.Context, event domain.CloudEvent) (string, error) {
	if f.publishErr != nil {
		return "", f.publishErr
	}
	f.published = append(f.published, event)
	if f.returnID != "" {
		return f.returnID, nil
	}
	return "STREAM:42", nil
}

func (f *fakeEventBus) Subscribe(_ context.Context, _ domain.SubscribeRequest) (<-chan domain.CloudEvent, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeEventBus) Unsubscribe(_ context.Context, _ string) error {
	return errors.New("not implemented")
}

func validEvent() *zynaxv1.CloudEvent {
	return &zynaxv1.CloudEvent{
		Id:          "evt-001",
		Source:      "/zynax/wf-42/engine-adapter",
		Specversion: "1.0",
		Type:        "zynax.v1.engine-adapter.workflow.completed",
		Time:        timestamppb.New(time.Now().UTC()),
		Data:        []byte(`{"status":"ok"}`),
	}
}

func TestPublish_NilEvent(t *testing.T) {
	h := api.NewHandler(&fakeEventBus{})
	_, err := h.Publish(context.Background(), &zynaxv1.PublishRequest{Event: nil})
	requireCode(t, err, codes.InvalidArgument)
}

func TestPublish_EmptyID(t *testing.T) {
	ev := validEvent()
	ev.Id = ""
	h := api.NewHandler(&fakeEventBus{})
	_, err := h.Publish(context.Background(), &zynaxv1.PublishRequest{Event: ev})
	requireCode(t, err, codes.InvalidArgument)
}

func TestPublish_EmptySource(t *testing.T) {
	ev := validEvent()
	ev.Source = ""
	h := api.NewHandler(&fakeEventBus{})
	_, err := h.Publish(context.Background(), &zynaxv1.PublishRequest{Event: ev})
	requireCode(t, err, codes.InvalidArgument)
}

func TestPublish_EmptyType(t *testing.T) {
	ev := validEvent()
	ev.Type = ""
	h := api.NewHandler(&fakeEventBus{})
	_, err := h.Publish(context.Background(), &zynaxv1.PublishRequest{Event: ev})
	requireCode(t, err, codes.InvalidArgument)
}

func TestPublish_HappyPath(t *testing.T) {
	fake := &fakeEventBus{returnID: "MYSTREAM:7"}
	h := api.NewHandler(fake)
	ev := validEvent()

	resp, err := h.Publish(context.Background(), &zynaxv1.PublishRequest{Event: ev})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetEventId() != "MYSTREAM:7" {
		t.Errorf("EventId: got %q, want %q", resp.GetEventId(), "MYSTREAM:7")
	}
	if resp.GetAcceptedAt() == nil {
		t.Error("AcceptedAt must not be nil")
	}
	if len(fake.published) != 1 {
		t.Fatalf("expected 1 published event, got %d", len(fake.published))
	}
	got := fake.published[0]
	if got.ID != ev.Id {
		t.Errorf("domain event ID: got %q, want %q", got.ID, ev.Id)
	}
	if got.Source != ev.Source {
		t.Errorf("domain event Source: got %q, want %q", got.Source, ev.Source)
	}
	if got.Type != ev.Type {
		t.Errorf("domain event Type: got %q, want %q", got.Type, ev.Type)
	}
}

func TestPublish_DomainError(t *testing.T) {
	fake := &fakeEventBus{publishErr: errors.New("nats timeout")}
	h := api.NewHandler(fake)

	_, err := h.Publish(context.Background(), &zynaxv1.PublishRequest{Event: validEvent()})
	requireCode(t, err, codes.Internal)
}

func TestPublish_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	h := api.NewHandler(&fakeEventBus{})
	_, err := h.Publish(ctx, &zynaxv1.PublishRequest{Event: validEvent()})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestPublish_ExtensionFieldsPropagated(t *testing.T) {
	fake := &fakeEventBus{}
	h := api.NewHandler(fake)
	ev := validEvent()
	ev.WorkflowId = testWorkflowID
	ev.RunId = testRunID
	ev.Namespace = testNamespace
	ev.CapabilityName = testCapabilityName
	ev.Datacontenttype = testDataContentType

	_, err := h.Publish(context.Background(), &zynaxv1.PublishRequest{Event: ev})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fake.published) != 1 {
		t.Fatalf("expected 1 published event, got %d", len(fake.published))
	}
	got := fake.published[0]
	if got.WorkflowID != testWorkflowID {
		t.Errorf("WorkflowID: got %q, want %q", got.WorkflowID, testWorkflowID)
	}
	if got.RunID != testRunID {
		t.Errorf("RunID: got %q, want %q", got.RunID, testRunID)
	}
	if got.Namespace != testNamespace {
		t.Errorf("Namespace: got %q, want %q", got.Namespace, testNamespace)
	}
	if got.CapabilityName != testCapabilityName {
		t.Errorf("CapabilityName: got %q, want %q", got.CapabilityName, testCapabilityName)
	}
	if got.DataContentType != testDataContentType {
		t.Errorf("DataContentType: got %q, want %q", got.DataContentType, testDataContentType)
	}
}

// requireCode asserts that err is a gRPC status error with the expected code.
func requireCode(t *testing.T, err error, want codes.Code) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error with code %s, got nil", want)
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("error %v is not a gRPC status error", err)
	}
	if st.Code() != want {
		t.Errorf("code: got %s, want %s (message: %s)", st.Code(), want, st.Message())
	}
}
