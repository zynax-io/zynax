// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/zynax-io/zynax/agents/adapters/llm/internal/config"
)

// fakeBedrockStream is an in-memory bedrockEventStream for tests.
type fakeBedrockStream struct {
	events chan types.ConverseStreamOutput
	err    error
}

func (f *fakeBedrockStream) Events() <-chan types.ConverseStreamOutput { return f.events }
func (f *fakeBedrockStream) Err() error                                { return f.err }
func (f *fakeBedrockStream) Close() error                              { return nil }

// textEvent builds a contentBlockDelta event carrying text.
func textEvent(s string) types.ConverseStreamOutput {
	return &types.ConverseStreamOutputMemberContentBlockDelta{
		Value: types.ContentBlockDeltaEvent{
			Delta: &types.ContentBlockDeltaMemberText{Value: s},
		},
	}
}

// fakeConverseClient is a bedrockConverseStreamAPI that fails on call.
type fakeConverseClient struct{ err error }

func (f *fakeConverseClient) ConverseStream(
	_ context.Context,
	_ *bedrockruntime.ConverseStreamInput,
	_ ...func(*bedrockruntime.Options),
) (*bedrockruntime.ConverseStreamOutput, error) {
	return nil, f.err
}

func TestBedrockRunSuccess(t *testing.T) {
	t.Parallel()
	stream := &fakeBedrockStream{events: make(chan types.ConverseStreamOutput, 3)}
	stream.events <- textEvent("Hel")
	stream.events <- textEvent("lo")
	stream.events <- &types.ConverseStreamOutputMemberMetadata{} // ignored event type
	close(stream.events)

	p := &bedrockProvider{model: "m", maxTokens: 16}
	out := make(chan Chunk)
	go p.run(context.Background(), stream, out)

	texts, streamErr := collect(t, out)
	if streamErr != nil {
		t.Fatalf("unexpected stream error: %v", streamErr)
	}
	if got := strings.Join(texts, ""); got != wantJoined {
		t.Fatalf("joined chunks = %q, want %q", got, wantJoined)
	}
}

func TestBedrockRunStreamError(t *testing.T) {
	t.Parallel()
	stream := &fakeBedrockStream{events: make(chan types.ConverseStreamOutput), err: errors.New("throttled: secret-arn")}
	close(stream.events)

	p := &bedrockProvider{model: "m", maxTokens: 16}
	out := make(chan Chunk)
	go p.run(context.Background(), stream, out)

	_, streamErr := collect(t, out)
	if streamErr == nil {
		t.Fatal("want terminal error from stream.Err()")
	}
	if !strings.Contains(streamErr.Error(), "throttled") {
		t.Fatalf("error missing upstream detail: %v", streamErr)
	}
}

func TestBedrockStreamCallError(t *testing.T) {
	t.Parallel()
	p := &bedrockProvider{model: "m", maxTokens: 16, client: &fakeConverseClient{err: errors.New("access denied")}}
	_, err := p.Stream(context.Background(), "hi")
	if err == nil {
		t.Fatal("want error when ConverseStream fails")
	}
	if !strings.Contains(err.Error(), "bedrock:") {
		t.Fatalf("error not namespaced: %v", err)
	}
}

func TestBedrockRunRespectsCancel(t *testing.T) {
	t.Parallel()
	stream := &fakeBedrockStream{events: make(chan types.ConverseStreamOutput, 1)}
	stream.events <- textEvent("x")
	// channel left open; run should exit when ctx is cancelled and no reader drains.

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p := &bedrockProvider{model: "m", maxTokens: 16}
	out := make(chan Chunk) // no reader

	done := make(chan struct{})
	go func() {
		p.run(ctx, stream, out)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("run did not exit promptly on cancelled ctx")
	}
}

func TestNewBedrockProvider(t *testing.T) {
	t.Parallel()
	p, err := newBedrockProvider(config.ProviderConfig{Name: "bedrock", Model: "m", Region: "us-east-1"})
	if err != nil {
		t.Fatalf("newBedrockProvider: %v", err)
	}
	if p.client == nil {
		t.Fatal("bedrock client not constructed")
	}
}
