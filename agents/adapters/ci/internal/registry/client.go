// SPDX-License-Identifier: Apache-2.0

// Package registry provides helpers for registering and deregistering the
// ci-adapter with AgentRegistryService on startup and graceful shutdown.
package registry

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/zynax-io/zynax/agents/adapters/ci/internal/config"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	maxAttempts = 5
	baseDelay   = 2 * time.Second
)

// RegisterAgent calls AgentRegistryService.RegisterAgent with exponential backoff.
// Retries up to maxAttempts on transient gRPC errors; base delay 2 s, doubles each attempt.
// Non-transient errors (INVALID_ARGUMENT, ALREADY_EXISTS, …) are returned immediately.
func RegisterAgent(ctx context.Context, stub zynaxv1.AgentRegistryServiceClient, def *zynaxv1.AgentDef) error {
	req := &zynaxv1.RegisterAgentRequest{Agent: def}

	var lastErr error
	for attempt := range maxAttempts {
		if attempt > 0 {
			delay := baseDelay * (1 << (attempt - 1))
			slog.Info("retrying agent registration", "attempt", attempt+1, "delay", delay)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return fmt.Errorf("registry: registration cancelled after %d attempts: %w", attempt, ctx.Err())
			}
		}

		_, err := stub.RegisterAgent(ctx, req)
		if err == nil {
			slog.Info("agent registered", "agent_id", def.AgentId)
			return nil
		}
		// CRD-era registry (ADR-039): push registration is retired and the
		// RPC answers UNIMPLEMENTED — discovery flows through the Agent
		// custom resource instead. Keep serving; nothing to retry.
		if status.Code(err) == codes.Unimplemented {
			slog.Info("push registration retired (ADR-039) — relying on Agent CR discovery", "agent_id", def.AgentId)
			return nil
		}
		if !isTransient(err) {
			return fmt.Errorf("registry: register failed (non-transient): %w", err)
		}
		lastErr = err
		slog.Warn("registration attempt failed", "attempt", attempt+1, "err", err)
	}
	return fmt.Errorf("registry: register failed after %d attempts: %w", maxAttempts, lastErr)
}

// DeregisterAgent calls AgentRegistryService.DeregisterAgent once, no retry.
func DeregisterAgent(ctx context.Context, stub zynaxv1.AgentRegistryServiceClient, agentID string) error {
	_, err := stub.DeregisterAgent(ctx, &zynaxv1.DeregisterAgentRequest{AgentId: agentID})
	if status.Code(err) == codes.Unimplemented {
		slog.Info("push deregistration retired (ADR-039) — Agent CR lifecycle owns removal", "agent_id", agentID)
		return nil
	}
	if err != nil {
		return fmt.Errorf("registry: deregister failed: %w", err)
	}
	slog.Info("agent deregistered", "agent_id", agentID)
	return nil
}

// BuildAgentDef constructs the AgentDef proto from the parsed AdapterConfig.
func BuildAgentDef(cfg *config.AdapterConfig) *zynaxv1.AgentDef {
	caps := make([]*zynaxv1.CapabilityDef, 0, len(cfg.Capabilities))
	for _, c := range cfg.Capabilities {
		caps = append(caps, &zynaxv1.CapabilityDef{
			Name:         c.Name,
			Description:  c.Description,
			InputSchema:  []byte(c.InputSchemaJSON),
			OutputSchema: []byte(c.OutputSchemaJSON),
		})
	}
	return &zynaxv1.AgentDef{
		AgentId:      cfg.AgentID,
		Name:         cfg.Name,
		Description:  cfg.Description,
		Endpoint:     cfg.Endpoint,
		Capabilities: caps,
	}
}

// isTransient reports whether a gRPC error is safe to retry.
func isTransient(err error) bool {
	st, ok := status.FromError(err)
	if !ok {
		return false
	}
	switch st.Code() {
	case codes.Unavailable, codes.Internal, codes.DeadlineExceeded:
		return true
	}
	return false
}
