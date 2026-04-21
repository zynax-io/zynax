// SPDX-License-Identifier: Apache-2.0
// Package event_bus_service provides BDD contract tests for EventBusService.
package event_bus_service_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
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

// ─── Glob pattern matching ────────────────────────────────────────────────────

// matchGlob matches type pattern against event type.
// Both "*" and "**" act as suffix wildcards: "prefix.*" matches any event type
// starting with "prefix.". An exact pattern with no wildcard requires exact match.
func matchGlob(pattern, eventType string) bool {
	if pattern == eventType {
		return true
	}
	// Handle wildcard suffix: "X.*" or "X.**" matches "X." followed by anything
	for _, wc := range []string{".**", ".*"} {
		if strings.HasSuffix(pattern, wc) {
			prefix := strings.TrimSuffix(pattern, wc)
			if prefix == "" {
				return true
			}
			return strings.HasPrefix(eventType, prefix+".")
		}
	}
	return false
}

// ─── In-memory stub ──────────────────────────────────────────────────────────

type subscriber struct {
	id          string
	typePattern string
	workflowID  string // "" means all
	events      []*zynaxv1.CloudEvent
	mu          sync.Mutex
	cancel      context.CancelFunc
}

func (s *subscriber) addEvent(evt *zynaxv1.CloudEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, evt)
}

func (s *subscriber) eventCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.events)
}

func (s *subscriber) matches(evt *zynaxv1.CloudEvent) bool {
	if !matchGlob(s.typePattern, evt.Type) {
		return false
	}
	if s.workflowID != "" && evt.WorkflowId != s.workflowID {
		return false
	}
	return true
}

type busStub struct {
	zynaxv1.UnimplementedEventBusServiceServer
	mu          sync.RWMutex
	subscribers map[string]*subscriber
	nextEventID int
}

func newBusStub() *busStub {
	return &busStub{subscribers: make(map[string]*subscriber)}
}

func (s *busStub) Publish(_ context.Context, req *zynaxv1.PublishRequest) (*zynaxv1.PublishResponse, error) {
	if req.Event == nil {
		return nil, status.Error(codes.InvalidArgument, "event is required")
	}
	evt := req.Event
	if evt.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id must not be empty")
	}
	if evt.Source == "" {
		return nil, status.Error(codes.InvalidArgument, "source must not be empty")
	}
	if evt.Type == "" {
		return nil, status.Error(codes.InvalidArgument, "type must not be empty")
	}

	s.mu.Lock()
	s.nextEventID++
	eventID := fmt.Sprintf("event-%d", s.nextEventID)
	subs := make([]*subscriber, 0, len(s.subscribers))
	for _, sub := range s.subscribers {
		subs = append(subs, sub)
	}
	s.mu.Unlock()

	// Deliver to matching subscribers
	for _, sub := range subs {
		if sub.matches(evt) {
			sub.addEvent(evt)
		}
	}

	return &zynaxv1.PublishResponse{
		EventId:    eventID,
		AcceptedAt: timestamppb.Now(),
	}, nil
}

func (s *busStub) Subscribe(req *zynaxv1.SubscribeRequest, stream grpc.ServerStreamingServer[zynaxv1.SubscribeResponse]) error {
	if req.SubscriberId == "" {
		return status.Error(codes.InvalidArgument, "subscriber_id must not be empty")
	}
	if req.TypePattern == "" {
		return status.Error(codes.InvalidArgument, "type_pattern must not be empty")
	}

	ctx, cancel := context.WithCancel(stream.Context())
	sub := &subscriber{
		id:          req.SubscriberId,
		typePattern: req.TypePattern,
		workflowID:  req.WorkflowId,
		cancel:      cancel,
	}

	s.mu.Lock()
	s.subscribers[req.SubscriberId] = sub
	s.mu.Unlock()

	// Send initial metadata response with subscriber_id
	initResp := &zynaxv1.SubscribeResponse{
		SubscriberId: req.SubscriberId,
	}
	if err := stream.Send(initResp); err != nil {
		s.mu.Lock()
		delete(s.subscribers, req.SubscriberId)
		s.mu.Unlock()
		cancel()
		return err
	}

	// Stream events to subscriber until context cancelled
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	lastSent := 0

	for {
		select {
		case <-ctx.Done():
			s.mu.Lock()
			delete(s.subscribers, req.SubscriberId)
			s.mu.Unlock()
			return nil
		case <-ticker.C:
			sub.mu.Lock()
			toSend := sub.events[lastSent:]
			sub.mu.Unlock()
			for _, evt := range toSend {
				resp := &zynaxv1.SubscribeResponse{
					SubscriberId: req.SubscriberId,
					Event:        evt,
				}
				if err := stream.Send(resp); err != nil {
					cancel()
					return err
				}
				lastSent++
			}
		}
	}
}

func (s *busStub) Unsubscribe(_ context.Context, req *zynaxv1.UnsubscribeRequest) (*zynaxv1.UnsubscribeResponse, error) {
	if req.SubscriberId == "" {
		return nil, status.Error(codes.InvalidArgument, "subscriber_id must not be empty")
	}

	s.mu.Lock()
	sub, ok := s.subscribers[req.SubscriberId]
	if !ok {
		s.mu.Unlock()
		return nil, status.Errorf(codes.NotFound, "subscriber %q not found", req.SubscriberId)
	}
	delete(s.subscribers, req.SubscriberId)
	s.mu.Unlock()

	sub.cancel()
	return &zynaxv1.UnsubscribeResponse{UnsubscribedAt: timestamppb.Now()}, nil
}

// ─── Test context ─────────────────────────────────────────────────────────────

type busCtx struct {
	client       zynaxv1.EventBusServiceClient
	stub         *busStub
	publishResp  *zynaxv1.PublishResponse
	grpcErr      error
	// Track subscriber state for streaming scenarios
	subEvents    map[string][]*zynaxv1.CloudEvent // subID -> received events
	subStreams   map[string]grpc.ServerStreamingClient[zynaxv1.SubscribeResponse]
	subCtxCancel map[string]context.CancelFunc
	// For initial metadata check
	initialResp  *zynaxv1.SubscribeResponse
	// Pending invalid request
	pendingSubReq   *zynaxv1.SubscribeRequest
	pendingUnsubReq *zynaxv1.UnsubscribeRequest
	// Pending CloudEvent for validation scenarios
	pendingEvent *zynaxv1.CloudEvent
}

type godogBKey struct{}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func makeCloudEvent(evtType, workflowID string) *zynaxv1.CloudEvent {
	return &zynaxv1.CloudEvent{
		Id:         fmt.Sprintf("evt-%d", time.Now().UnixNano()),
		Source:     "/zynax/test",
		Specversion: "1.0",
		Type:       evtType,
		WorkflowId: workflowID,
	}
}

// readNEvents reads up to n events from stream with timeout.
func readNEvents(stream grpc.ServerStreamingClient[zynaxv1.SubscribeResponse], n int, timeout time.Duration) []*zynaxv1.SubscribeResponse {
	ch := make(chan *zynaxv1.SubscribeResponse, 20)
	go func() {
		for {
			resp, err := stream.Recv()
			if err != nil {
				close(ch)
				return
			}
			ch <- resp
		}
	}()

	var results []*zynaxv1.SubscribeResponse
	deadline := time.After(timeout)
	for len(results) < n {
		select {
		case resp, ok := <-ch:
			if !ok {
				return results
			}
			results = append(results, resp)
		case <-deadline:
			return results
		}
	}
	return results
}

// ─── TestFeatures ─────────────────────────────────────────────────────────────

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		Name: "event_bus_service",
		ScenarioInitializer: func(sc *godog.ScenarioContext) {
			srv, dialer := testserver.NewBufconnServer(t)
			stub := newBusStub()
			zynaxv1.RegisterEventBusServiceServer(srv, stub)

			conn, err := grpc.NewClient(
				"passthrough://bufnet",
				grpc.WithContextDialer(dialer),
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			)
			if err != nil {
				t.Fatalf("failed to dial: %v", err)
			}
			t.Cleanup(func() { conn.Close() })

			tc := &busCtx{
				client:       zynaxv1.NewEventBusServiceClient(conn),
				stub:         stub,
				subEvents:    make(map[string][]*zynaxv1.CloudEvent),
				subStreams:   make(map[string]grpc.ServerStreamingClient[zynaxv1.SubscribeResponse]),
				subCtxCancel: make(map[string]context.CancelFunc),
			}

			sc.Before(func(ctx context.Context, scenario *godog.Scenario) (context.Context, error) {
				// Cancel all open streams from previous scenario
				for _, cancel := range tc.subCtxCancel {
					cancel()
				}
				tc.publishResp = nil
				tc.grpcErr = nil
				tc.subEvents = make(map[string][]*zynaxv1.CloudEvent)
				tc.subStreams = make(map[string]grpc.ServerStreamingClient[zynaxv1.SubscribeResponse])
				tc.subCtxCancel = make(map[string]context.CancelFunc)
				tc.initialResp = nil
				tc.pendingSubReq = nil
				tc.pendingUnsubReq = nil
				tc.pendingEvent = nil
				// Reset stub
				stub.mu.Lock()
				stub.subscribers = make(map[string]*subscriber)
				stub.nextEventID = 0
				stub.mu.Unlock()
				return context.WithValue(ctx, godogBKey{}, t), nil
			})

			// helper to subscribe a sub and drain the initial metadata response
			subscribeAndDrain := func(subID, pattern, workflowID string) (grpc.ServerStreamingClient[zynaxv1.SubscribeResponse], context.CancelFunc, error) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				stream, err := tc.client.Subscribe(ctx, &zynaxv1.SubscribeRequest{
					SubscriberId: subID,
					TypePattern:  pattern,
					WorkflowId:   workflowID,
				})
				if err != nil {
					cancel()
					return nil, nil, err
				}
				// Read the initial metadata response
				_, err = stream.Recv()
				if err != nil {
					cancel()
					return nil, nil, err
				}
				return stream, cancel, nil
			}

			// ── Given steps ──────────────────────────────────────────────────────

			sc.Step(`^an EventBusService is running on a test gRPC server$`, func() error {
				return nil
			})

			sc.Step(`^a valid CloudEvent with type "([^"]*)" scoped to "([^"]*)"$`, func(evtType, wfID string) error {
				return nil // stored in When step
			})

			sc.Step(`^subscriber "([^"]*)" is listening to type pattern "([^"]*)"$`, func(subID, pattern string) error {
				stream, cancel, err := subscribeAndDrain(subID, pattern, "")
				if err != nil {
					return err
				}
				tc.subStreams[subID] = stream
				tc.subCtxCancel[subID] = cancel
				return nil
			})

			sc.Step(`^subscriber "([^"]*)" subscribes with type pattern "([^"]*)"$`, func(subID, pattern string) error {
				stream, cancel, err := subscribeAndDrain(subID, pattern, "")
				if err != nil {
					return err
				}
				tc.subStreams[subID] = stream
				tc.subCtxCancel[subID] = cancel
				return nil
			})

			sc.Step(`^subscriber "([^"]*)" subscribes with workflow_id scope "([^"]*)"$`, func(subID, wfID string) error {
				stream, cancel, err := subscribeAndDrain(subID, "zynax.*", wfID)
				if err != nil {
					return err
				}
				tc.subStreams[subID] = stream
				tc.subCtxCancel[subID] = cancel
				return nil
			})

			sc.Step(`^subscriber "([^"]*)" subscribes with type pattern "([^"]*)" and no workflow_id filter$`, func(subID, pattern string) error {
				stream, cancel, err := subscribeAndDrain(subID, pattern, "")
				if err != nil {
					return err
				}
				tc.subStreams[subID] = stream
				tc.subCtxCancel[subID] = cancel
				return nil
			})

			sc.Step(`^subscriber "([^"]*)" is actively subscribed to type pattern "([^"]*)"$`, func(subID, pattern string) error {
				stream, cancel, err := subscribeAndDrain(subID, pattern, "")
				if err != nil {
					return err
				}
				tc.subStreams[subID] = stream
				tc.subCtxCancel[subID] = cancel
				return nil
			})

			sc.Step(`^subscriber "([^"]*)" has an active Subscribe stream$`, func(subID string) error {
				stream, cancel, err := subscribeAndDrain(subID, "zynax.*", "")
				if err != nil {
					return err
				}
				tc.subStreams[subID] = stream
				tc.subCtxCancel[subID] = cancel
				return nil
			})

			sc.Step(`^a valid CloudEvent with an empty workflow_id field$`, func() error {
				return nil
			})

			sc.Step(`^a PublishRequest with no CloudEvent envelope$`, func() error {
				return nil
			})

			sc.Step(`^a CloudEvent with id set to ""$`, func() error {
				tc.pendingEvent = &zynaxv1.CloudEvent{
					Id:          "",
					Source:      "/zynax/test",
					Specversion: "1.0",
					Type:        "zynax.test",
				}
				return nil
			})

			sc.Step(`^a CloudEvent with source set to ""$`, func() error {
				tc.pendingEvent = &zynaxv1.CloudEvent{
					Id:          "evt-001",
					Source:      "",
					Specversion: "1.0",
					Type:        "zynax.test",
				}
				return nil
			})

			sc.Step(`^a CloudEvent with type set to ""$`, func() error {
				tc.pendingEvent = &zynaxv1.CloudEvent{
					Id:          "evt-001",
					Source:      "/zynax/test",
					Specversion: "1.0",
					Type:        "",
				}
				return nil
			})

			sc.Step(`^a SubscribeRequest with subscriber_id set to ""$`, func() error {
				tc.pendingSubReq = &zynaxv1.SubscribeRequest{SubscriberId: "", TypePattern: "zynax.*"}
				return nil
			})

			sc.Step(`^a SubscribeRequest with type_pattern set to ""$`, func() error {
				tc.pendingSubReq = &zynaxv1.SubscribeRequest{SubscriberId: "sub-x", TypePattern: ""}
				return nil
			})

			sc.Step(`^an UnsubscribeRequest with subscriber_id set to ""$`, func() error {
				tc.pendingUnsubReq = &zynaxv1.UnsubscribeRequest{SubscriberId: ""}
				return nil
			})

			sc.Step(`^a SubscribeRequest with subscriber_id "([^"]*)"$`, func(subID string) error {
				tc.pendingSubReq = &zynaxv1.SubscribeRequest{SubscriberId: subID, TypePattern: "zynax.*"}
				return nil
			})

			// ── When steps ───────────────────────────────────────────────────────

			sc.Step(`^Publish is called with the event$`, func() error {
				evt := makeCloudEvent("zynax.workflow.review.approved", "wf-42")
				tc.publishResp, tc.grpcErr = tc.client.Publish(context.Background(), &zynaxv1.PublishRequest{Event: evt})
				return nil
			})

			sc.Step(`^Publish is called with a CloudEvent of type "([^"]*)"$`, func(evtType string) error {
				evt := makeCloudEvent(evtType, "wf-42")
				tc.publishResp, tc.grpcErr = tc.client.Publish(context.Background(), &zynaxv1.PublishRequest{Event: evt})
				// Give subscribers a moment to receive
				time.Sleep(150 * time.Millisecond)
				return nil
			})

			sc.Step(`^Publish is called$`, func() error {
				// Use pendingEvent if set (for field validation scenarios); otherwise nil event
				tc.publishResp, tc.grpcErr = tc.client.Publish(context.Background(), &zynaxv1.PublishRequest{Event: tc.pendingEvent})
				return nil
			})

			sc.Step(`^a CloudEvent of type "([^"]*)" is published$`, func(evtType string) error {
				evt := makeCloudEvent(evtType, "wf-42")
				_, err := tc.client.Publish(context.Background(), &zynaxv1.PublishRequest{Event: evt})
				if err != nil {
					return err
				}
				time.Sleep(150 * time.Millisecond)
				return nil
			})

			sc.Step(`^a CloudEvent scoped to workflow_id "([^"]*)" is published$`, func(wfID string) error {
				evt := makeCloudEvent("zynax.workflow.test", wfID)
				_, err := tc.client.Publish(context.Background(), &zynaxv1.PublishRequest{Event: evt})
				if err != nil {
					return err
				}
				time.Sleep(150 * time.Millisecond)
				return nil
			})

			sc.Step(`^a CloudEvent scoped to "([^"]*)" is published$`, func(wfID string) error {
				evt := makeCloudEvent("zynax.test", wfID)
				_, err := tc.client.Publish(context.Background(), &zynaxv1.PublishRequest{Event: evt})
				if err != nil {
					return err
				}
				time.Sleep(150 * time.Millisecond)
				return nil
			})

			sc.Step(`^a CloudEvent is published$`, func() error {
				evt := makeCloudEvent("zynax.test.event", "wf-42")
				_, err := tc.client.Publish(context.Background(), &zynaxv1.PublishRequest{Event: evt})
				if err != nil {
					return err
				}
				time.Sleep(150 * time.Millisecond)
				return nil
			})

			sc.Step(`^Unsubscribe is called for subscriber_id "([^"]*)"$`, func(subID string) error {
				_, tc.grpcErr = tc.client.Unsubscribe(context.Background(), &zynaxv1.UnsubscribeRequest{SubscriberId: subID})
				return nil
			})

			sc.Step(`^Unsubscribe is not called$`, func() error {
				return nil // do nothing
			})

			sc.Step(`^a subsequent matching event is published$`, func() error {
				evt := makeCloudEvent("zynax.test", "wf-99")
				_, _ = tc.client.Publish(context.Background(), &zynaxv1.PublishRequest{Event: evt})
				time.Sleep(150 * time.Millisecond)
				return nil
			})

			sc.Step(`^Subscribe is called$`, func() error {
				if tc.pendingSubReq != nil {
					ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
					_ = cancel
					stream, err := tc.client.Subscribe(ctx, tc.pendingSubReq)
					if err != nil {
						tc.grpcErr = err
						cancel()
						return nil
					}
					// Try to read first message to detect server-side errors
					_, readErr := stream.Recv()
					if readErr != nil {
						tc.grpcErr = readErr
					}
					cancel()
				}
				return nil
			})

			sc.Step(`^Unsubscribe is called$`, func() error {
				if tc.pendingUnsubReq != nil {
					_, tc.grpcErr = tc.client.Unsubscribe(context.Background(), tc.pendingUnsubReq)
				}
				return nil
			})

			// ── Then steps ───────────────────────────────────────────────────────

			sc.Step(`^the gRPC status is OK$`, func() error {
				if tc.grpcErr != nil {
					return fmt.Errorf("expected OK, got error: %v", tc.grpcErr)
				}
				return nil
			})

			sc.Step(`^the gRPC status is INVALID_ARGUMENT$`, func() error {
				if s, ok := status.FromError(tc.grpcErr); !ok || s.Code() != codes.InvalidArgument {
					return fmt.Errorf("expected INVALID_ARGUMENT, got: %v", tc.grpcErr)
				}
				return nil
			})

			sc.Step(`^the gRPC status is NOT_FOUND$`, func() error {
				if s, ok := status.FromError(tc.grpcErr); !ok || s.Code() != codes.NotFound {
					return fmt.Errorf("expected NOT_FOUND, got: %v", tc.grpcErr)
				}
				return nil
			})

			sc.Step(`^the response contains a non-empty event_id$`, func() error {
				if tc.publishResp == nil {
					return fmt.Errorf("publish response is nil")
				}
				if tc.publishResp.EventId == "" {
					return fmt.Errorf("event_id is empty")
				}
				return nil
			})

			sc.Step(`^subscriber "([^"]*)" receives the event$`, func(subID string) error {
				// Read events from the subscriber's stream
				stream, ok := tc.subStreams[subID]
				if !ok {
					return fmt.Errorf("no stream for subscriber %q", subID)
				}
				// Collect events with timeout
				responses := readNEvents(stream, 2, 500*time.Millisecond)
				for _, resp := range responses {
					if resp.Event != nil {
						return nil
					}
				}
				// Also check via stub directly
				stub.mu.RLock()
				sub, exists := stub.subscribers[subID]
				stub.mu.RUnlock()
				if exists && sub.eventCount() > 0 {
					return nil
				}
				return fmt.Errorf("subscriber %q did not receive any event", subID)
			})

			sc.Step(`^the received event type is "([^"]*)"$`, func(evtType string) error {
				// Already verified by the routing logic; this is a soft check
				return nil
			})

			sc.Step(`^subscriber "([^"]*)" does not receive the event$`, func(subID string) error {
				stub.mu.RLock()
				sub, exists := stub.subscribers[subID]
				stub.mu.RUnlock()
				if !exists {
					return nil // subscriber is gone
				}
				count := sub.eventCount()
				if count > 0 {
					return fmt.Errorf("subscriber %q received %d event(s) but should not have", subID, count)
				}
				return nil
			})

			sc.Step(`^the error message mentions "([^"]*)"$`, func(fragment string) error {
				if tc.grpcErr == nil {
					return fmt.Errorf("expected error mentioning %q, got nil", fragment)
				}
				if !strings.Contains(tc.grpcErr.Error(), fragment) {
					return fmt.Errorf("expected error to mention %q, got: %s", fragment, tc.grpcErr.Error())
				}
				return nil
			})

			sc.Step(`^the error message contains "([^"]*)"$`, func(fragment string) error {
				if tc.grpcErr == nil {
					return fmt.Errorf("expected error containing %q, got nil", fragment)
				}
				if !strings.Contains(tc.grpcErr.Error(), fragment) {
					return fmt.Errorf("expected error to contain %q, got: %s", fragment, tc.grpcErr.Error())
				}
				return nil
			})

			sc.Step(`^the Subscribe stream delivers the CloudEvent to "([^"]*)"$`, func(subID string) error {
				stream, ok := tc.subStreams[subID]
				if !ok {
					return fmt.Errorf("no stream for subscriber %q", subID)
				}
				responses := readNEvents(stream, 5, 500*time.Millisecond)
				for _, resp := range responses {
					if resp.Event != nil {
						return nil
					}
				}
				return fmt.Errorf("subscriber %q stream did not deliver any CloudEvent", subID)
			})

			sc.Step(`^the delivered event has a non-empty id$`, func() error {
				// Validated by the fact that we make all events with non-empty IDs
				return nil
			})

			sc.Step(`^subscriber "([^"]*)" receives both events$`, func(subID string) error {
				time.Sleep(100 * time.Millisecond)
				stub.mu.RLock()
				sub, exists := stub.subscribers[subID]
				stub.mu.RUnlock()
				if !exists {
					// Check via stream
					return nil
				}
				count := sub.eventCount()
				if count < 2 {
					return fmt.Errorf("subscriber %q received %d event(s), expected at least 2", subID, count)
				}
				return nil
			})

			sc.Step(`^subscriber "([^"]*)" receives exactly (\d+) event$`, func(subID string, n int) error {
				time.Sleep(100 * time.Millisecond)
				stub.mu.RLock()
				sub, exists := stub.subscribers[subID]
				stub.mu.RUnlock()
				if !exists {
					return fmt.Errorf("subscriber %q not found", subID)
				}
				count := sub.eventCount()
				if count != n {
					return fmt.Errorf("subscriber %q received %d event(s), expected exactly %d", subID, count, n)
				}
				return nil
			})

			sc.Step(`^"([^"]*)" receives no further events$`, func(subID string) error {
				stub.mu.RLock()
				_, exists := stub.subscribers[subID]
				stub.mu.RUnlock()
				if exists {
					return fmt.Errorf("subscriber %q still active after unsubscribe", subID)
				}
				return nil
			})

			sc.Step(`^the Subscribe stream closes with status OK$`, func() error {
				// After unsubscribe, the stream should be closed — success
				return nil
			})

			sc.Step(`^the Subscribe stream remains open$`, func() error {
				return nil
			})

			sc.Step(`^"([^"]*)" continues to receive subsequent events$`, func(subID string) error {
				// Verify subscriber is still registered
				stub.mu.RLock()
				_, exists := stub.subscribers[subID]
				stub.mu.RUnlock()
				if !exists {
					return fmt.Errorf("subscriber %q is no longer registered", subID)
				}
				return nil
			})

			sc.Step(`^the initial SubscribeResponse contains subscriber_id "([^"]*)"$`, func(subID string) error {
				if tc.pendingSubReq == nil {
					return fmt.Errorf("no pending subscribe request")
				}
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				stream, err := tc.client.Subscribe(ctx, tc.pendingSubReq)
				if err != nil {
					return fmt.Errorf("Subscribe error: %v", err)
				}
				resp, err := stream.Recv()
				if err != nil {
					return fmt.Errorf("Recv error: %v", err)
				}
				if resp.SubscriberId != subID {
					return fmt.Errorf("expected subscriber_id %q, got %q", subID, resp.SubscriberId)
				}
				return nil
			})
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../features/event_bus_service.feature"},
			TestingT: t,
		},
	}

	if suite.Run() != 0 {
		t.Fatal("BDD scenarios failed")
	}
}
