// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/zynax-io/zynax/agents/adapters/llm/internal/config"
)

// bedrockConverseStreamAPI is the minimal surface of the Bedrock Runtime client
// used by the provider. The real *bedrockruntime.Client satisfies it; tests
// inject a fake so no AWS network call is ever made.
type bedrockConverseStreamAPI interface {
	ConverseStream(
		ctx context.Context,
		in *bedrockruntime.ConverseStreamInput,
		optFns ...func(*bedrockruntime.Options),
	) (*bedrockruntime.ConverseStreamOutput, error)
}

// bedrockEventStream is the minimal surface of the Bedrock event stream reader
// used by run. The SDK's *bedrockruntime.ConverseStreamEventStream satisfies
// it; tests inject a fake to exercise drain/error paths without AWS.
type bedrockEventStream interface {
	Events() <-chan types.ConverseStreamOutput
	Err() error
	Close() error
}

// bedrockProvider streams token chunks from AWS Bedrock via the ConverseStream
// API. AWS credentials are resolved by the SDK's default chain (env, profile,
// IRSA) — never from this adapter's config or input_payload.
type bedrockProvider struct {
	model     string
	maxTokens int
	client    bedrockConverseStreamAPI
}

// newBedrockProvider builds the provider with a Bedrock Runtime client pinned
// to the configured region. AWS credentials are resolved lazily by the SDK's
// default credential chain (env vars, shared profile, IRSA) at request time —
// this adapter never reads or holds AWS credentials.
func newBedrockProvider(cfg config.ProviderConfig) (*bedrockProvider, error) {
	client := bedrockruntime.NewFromConfig(aws.Config{Region: cfg.Region})
	return &bedrockProvider{
		model:     cfg.Model,
		maxTokens: effectiveMaxTokens(cfg),
		client:    client,
	}, nil
}

// Stream invokes ConverseStream and relays each contentBlockDelta text Chunk.
func (p *bedrockProvider) Stream(ctx context.Context, prompt string) (<-chan Chunk, error) {
	maxTokens := int32(p.maxTokens) //nolint:gosec // bounded by config.MaxTokens (small positive int)
	in := &bedrockruntime.ConverseStreamInput{
		ModelId: &p.model,
		Messages: []types.Message{{
			Role:    types.ConversationRoleUser,
			Content: []types.ContentBlock{&types.ContentBlockMemberText{Value: prompt}},
		}},
		InferenceConfig: &types.InferenceConfiguration{MaxTokens: &maxTokens},
	}

	resp, err := p.client.ConverseStream(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("bedrock: %s", sanitiseErr(err.Error()))
	}

	out := make(chan Chunk)
	go p.run(ctx, resp.GetStream(), out)
	return out, nil
}

// run drains the event stream into out, closing it when the stream ends or
// fails. Only delta text is emitted; metadata/usage events are ignored.
func (p *bedrockProvider) run(ctx context.Context, stream bedrockEventStream, out chan<- Chunk) {
	defer close(out)
	defer func() { _ = stream.Close() }()

	for event := range stream.Events() {
		delta, ok := event.(*types.ConverseStreamOutputMemberContentBlockDelta)
		if !ok {
			continue
		}
		text, ok := delta.Value.Delta.(*types.ContentBlockDeltaMemberText)
		if !ok || text.Value == "" {
			continue
		}
		if !send(ctx, out, Chunk{Text: text.Value}) {
			return
		}
	}
	if err := stream.Err(); err != nil {
		sendErr(out, fmt.Errorf("bedrock: %s", sanitiseErr(err.Error())))
	}
}
