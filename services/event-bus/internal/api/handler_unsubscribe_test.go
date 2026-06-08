// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/event-bus/internal/api"
	"github.com/zynax-io/zynax/services/event-bus/internal/domain"
)

// unsubscribeFakeEventBus is an in-memory test double that controls Unsubscribe behaviour.
type unsubscribeFakeEventBus struct {
	fakeEventBus
	unsubscribeErr    error
	unsubscribeCalled bool
	lastSubscriberID  string
}

func (f *unsubscribeFakeEventBus) Unsubscribe(_ context.Context, subscriberID string) error {
	f.unsubscribeCalled = true
	f.lastSubscriberID = subscriberID
	return f.unsubscribeErr
}

func TestUnsubscribe_EmptySubscriberID(t *testing.T) {
	h := api.NewHandler(&unsubscribeFakeEventBus{})
	_, err := h.Unsubscribe(context.Background(), &zynaxv1.UnsubscribeRequest{SubscriberId: ""})
	requireCode(t, err, codes.InvalidArgument)
}

func TestUnsubscribe_HappyPath(t *testing.T) {
	fake := &unsubscribeFakeEventBus{}
	h := api.NewHandler(fake)

	resp, err := h.Unsubscribe(context.Background(), &zynaxv1.UnsubscribeRequest{SubscriberId: "sub-abc"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("response must not be nil")
	}
	if !fake.unsubscribeCalled {
		t.Error("expected Unsubscribe to be called on the domain bus")
	}
	if fake.lastSubscriberID != "sub-abc" {
		t.Errorf("subscriber_id: got %q, want %q", fake.lastSubscriberID, "sub-abc")
	}
}

func TestUnsubscribe_NotFound(t *testing.T) {
	fake := &unsubscribeFakeEventBus{unsubscribeErr: domain.ErrSubscriberNotFound}
	h := api.NewHandler(fake)

	_, err := h.Unsubscribe(context.Background(), &zynaxv1.UnsubscribeRequest{SubscriberId: "ghost-sub"})
	requireCode(t, err, codes.NotFound)
}

func TestUnsubscribe_InternalError(t *testing.T) {
	fake := &unsubscribeFakeEventBus{unsubscribeErr: domain.ErrDeadLetter}
	h := api.NewHandler(fake)

	_, err := h.Unsubscribe(context.Background(), &zynaxv1.UnsubscribeRequest{SubscriberId: "sub-err"})
	requireCode(t, err, codes.Internal)
}

func TestUnsubscribe_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	h := api.NewHandler(&unsubscribeFakeEventBus{})
	_, err := h.Unsubscribe(ctx, &zynaxv1.UnsubscribeRequest{SubscriberId: "sub-ctx"})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}
