// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/event-bus/internal/api"
	"github.com/zynax-io/zynax/services/event-bus/internal/domain"
)

// subscribeEventBus is an in-memory test double that supports Subscribe.
// It accepts a pre-configured channel of events to send, and optional errors.
type subscribeEventBus struct {
	fakeEventBus
	subscribeErr error
	events       []domain.CloudEvent
}

func (s *subscribeEventBus) Subscribe(_ context.Context, _ domain.SubscribeRequest) (<-chan domain.CloudEvent, error) {
	if s.subscribeErr != nil {
		return nil, s.subscribeErr
	}
	ch := make(chan domain.CloudEvent, len(s.events)+1)
	for _, e := range s.events {
		ch <- e
	}
	close(ch)
	return ch, nil
}

// fakeSubscribeStream is a mock of grpc.ServerStreamingServer[SubscribeResponse].
type fakeSubscribeStream struct {
	ctx     context.Context
	sent    []*zynaxv1.SubscribeResponse
	sendErr error
}

func (f *fakeSubscribeStream) Send(resp *zynaxv1.SubscribeResponse) error {
	if f.sendErr != nil {
		return f.sendErr
	}
	f.sent = append(f.sent, resp)
	return nil
}

func (f *fakeSubscribeStream) Context() context.Context { return f.ctx }

// grpc.ServerStream boilerplate (unused but required by interface).
func (f *fakeSubscribeStream) SetHeader(metadata.MD) error  { return nil }
func (f *fakeSubscribeStream) SendHeader(metadata.MD) error { return nil }
func (f *fakeSubscribeStream) SetTrailer(metadata.MD)       {}
func (f *fakeSubscribeStream) RecvMsg(any) error            { return io.EOF }
func (f *fakeSubscribeStream) SendMsg(any) error            { return nil }

var _ grpc.ServerStreamingServer[zynaxv1.SubscribeResponse] = (*fakeSubscribeStream)(nil)

func TestSubscribe_EmptySubscriberID(t *testing.T) {
	h := api.NewHandler(&subscribeEventBus{})
	stream := &fakeSubscribeStream{ctx: context.Background()}
	err := h.Subscribe(&zynaxv1.SubscribeRequest{
		SubscriberId: "",
		TypePattern:  "zynax.v1.*",
	}, stream)
	requireCode(t, err, codes.InvalidArgument)
}

func TestSubscribe_EmptyTypePattern(t *testing.T) {
	h := api.NewHandler(&subscribeEventBus{})
	stream := &fakeSubscribeStream{ctx: context.Background()}
	err := h.Subscribe(&zynaxv1.SubscribeRequest{
		SubscriberId: "sub-1",
		TypePattern:  "",
	}, stream)
	requireCode(t, err, codes.InvalidArgument)
}

func TestSubscribe_DomainError(t *testing.T) {
	fake := &subscribeEventBus{subscribeErr: errors.New("nats unavailable")}
	h := api.NewHandler(fake)
	stream := &fakeSubscribeStream{ctx: context.Background()}
	err := h.Subscribe(&zynaxv1.SubscribeRequest{
		SubscriberId: "sub-1",
		TypePattern:  "zynax.v1.*",
	}, stream)
	requireCode(t, err, codes.Internal)
}

func TestSubscribe_HappyPath_DeliversEvent(t *testing.T) {
	event := domain.CloudEvent{
		ID:          "evt-001",
		Source:      "/zynax/engine-adapter",
		SpecVersion: "1.0",
		Type:        "zynax.v1.workflow.completed",
		Data:        []byte(`{"status":"ok"}`),
		WorkflowID:  "wf-42",
	}
	fake := &subscribeEventBus{events: []domain.CloudEvent{event}}
	h := api.NewHandler(fake)
	stream := &fakeSubscribeStream{ctx: context.Background()}

	err := h.Subscribe(&zynaxv1.SubscribeRequest{
		SubscriberId: "sub-1",
		TypePattern:  "zynax.v1.workflow.*",
	}, stream)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stream.sent) != 1 {
		t.Fatalf("expected 1 sent response, got %d", len(stream.sent))
	}
	got := stream.sent[0]
	if got.GetSubscriberId() != "sub-1" {
		t.Errorf("subscriber_id: got %q, want sub-1", got.GetSubscriberId())
	}
	if got.GetEvent().GetId() != "evt-001" {
		t.Errorf("event.id: got %q, want evt-001", got.GetEvent().GetId())
	}
	if got.GetEvent().GetType() != "zynax.v1.workflow.completed" {
		t.Errorf("event.type: got %q, want zynax.v1.workflow.completed", got.GetEvent().GetType())
	}
	if got.GetEvent().GetWorkflowId() != "wf-42" {
		t.Errorf("event.workflow_id: got %q, want wf-42", got.GetEvent().GetWorkflowId())
	}
}

func TestSubscribe_TwoConsumerGroupsReceiveSameEvent(t *testing.T) {
	// Two distinct subscriber_ids both receive the same published event.
	event := domain.CloudEvent{
		ID:   "evt-shared",
		Type: "zynax.v1.workflow.completed",
	}

	fake1 := &subscribeEventBus{events: []domain.CloudEvent{event}}
	fake2 := &subscribeEventBus{events: []domain.CloudEvent{event}}

	h1 := api.NewHandler(fake1)
	h2 := api.NewHandler(fake2)

	stream1 := &fakeSubscribeStream{ctx: context.Background()}
	stream2 := &fakeSubscribeStream{ctx: context.Background()}

	req1 := &zynaxv1.SubscribeRequest{SubscriberId: "group-A", TypePattern: "zynax.v1.workflow.*"}
	req2 := &zynaxv1.SubscribeRequest{SubscriberId: "group-B", TypePattern: "zynax.v1.workflow.*"}

	if err := h1.Subscribe(req1, stream1); err != nil {
		t.Fatalf("group-A subscribe error: %v", err)
	}
	if err := h2.Subscribe(req2, stream2); err != nil {
		t.Fatalf("group-B subscribe error: %v", err)
	}

	if len(stream1.sent) != 1 {
		t.Errorf("group-A: expected 1 event, got %d", len(stream1.sent))
	}
	if len(stream2.sent) != 1 {
		t.Errorf("group-B: expected 1 event, got %d", len(stream2.sent))
	}
	if stream1.sent[0].GetEvent().GetId() != "evt-shared" {
		t.Errorf("group-A event ID: got %q, want evt-shared", stream1.sent[0].GetEvent().GetId())
	}
	if stream2.sent[0].GetEvent().GetId() != "evt-shared" {
		t.Errorf("group-B event ID: got %q, want evt-shared", stream2.sent[0].GetEvent().GetId())
	}
}

func TestSubscribe_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	blockingBus := &blockingSubscribeEventBus{}
	h := api.NewHandler(blockingBus)
	stream := &fakeSubscribeStream{ctx: ctx}

	err := h.Subscribe(&zynaxv1.SubscribeRequest{
		SubscriberId: "sub-timeout",
		TypePattern:  "zynax.v1.*",
	}, stream)

	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
}

// blockingSubscribeEventBus returns a channel that never closes (simulates no events).
type blockingSubscribeEventBus struct {
	fakeEventBus
}

func (b *blockingSubscribeEventBus) Subscribe(ctx context.Context, _ domain.SubscribeRequest) (<-chan domain.CloudEvent, error) {
	ch := make(chan domain.CloudEvent)
	go func() {
		<-ctx.Done()
		close(ch)
	}()
	return ch, nil
}

func TestSubscribe_TopicIsolation(t *testing.T) {
	// Bus delivers zero events when topic doesn't match (infra layer filters).
	fake := &subscribeEventBus{events: nil}
	h := api.NewHandler(fake)
	stream := &fakeSubscribeStream{ctx: context.Background()}

	err := h.Subscribe(&zynaxv1.SubscribeRequest{
		SubscriberId: "sub-isolated",
		TypePattern:  "zynax.v1.task.*",
	}, stream)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stream.sent) != 0 {
		t.Errorf("expected 0 events for isolated topic, got %d", len(stream.sent))
	}
}

func TestSubscribe_ExtensionFieldsPropagated(t *testing.T) {
	event := domain.CloudEvent{
		ID:              "evt-ext",
		Source:          "/zynax/engine-adapter",
		SpecVersion:     "1.0",
		Type:            "zynax.v1.workflow.completed",
		DataContentType: testDataContentType,
		Data:            []byte(`{}`),
		WorkflowID:      testWorkflowID,
		RunID:           testRunID,
		Namespace:       testNamespace,
		CapabilityName:  testCapabilityName,
		Time:            time.Now().UTC(),
	}
	fake := &subscribeEventBus{events: []domain.CloudEvent{event}}
	h := api.NewHandler(fake)
	stream := &fakeSubscribeStream{ctx: context.Background()}

	if err := h.Subscribe(&zynaxv1.SubscribeRequest{
		SubscriberId: "sub-ext",
		TypePattern:  "zynax.v1.*",
	}, stream); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stream.sent) != 1 {
		t.Fatalf("expected 1 event, got %d", len(stream.sent))
	}
	pbEv := stream.sent[0].GetEvent()
	if pbEv.GetWorkflowId() != testWorkflowID {
		t.Errorf("workflow_id: got %q", pbEv.GetWorkflowId())
	}
	if pbEv.GetRunId() != testRunID {
		t.Errorf("run_id: got %q", pbEv.GetRunId())
	}
	if pbEv.GetNamespace() != testNamespace {
		t.Errorf("namespace: got %q", pbEv.GetNamespace())
	}
	if pbEv.GetCapabilityName() != testCapabilityName {
		t.Errorf("capability_name: got %q", pbEv.GetCapabilityName())
	}
	if pbEv.GetDatacontenttype() != testDataContentType {
		t.Errorf("datacontenttype: got %q", pbEv.GetDatacontenttype())
	}
	if pbEv.GetTime() == nil {
		t.Error("time must not be nil when event.Time is set")
	}
}
