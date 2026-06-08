// SPDX-License-Identifier: Apache-2.0

// Package api wires domain interfaces to the MemoryService gRPC contract.
package api

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/memory-service/internal/domain"
)

// Handler implements zynaxv1.MemoryServiceServer.
// Namespace isolation (workflow_id) is enforced in every RPC before
// delegating to the domain store.
type Handler struct {
	zynaxv1.UnimplementedMemoryServiceServer
	kv  domain.KVStore
	vec domain.VectorStore
}

// NewHandler constructs a Handler with the given KV and Vector stores.
func NewHandler(kv domain.KVStore, vec domain.VectorStore) *Handler {
	return &Handler{kv: kv, vec: vec}
}

// ─── namespace guard ────────────────────────────────────────────────────────

// requireNamespace validates that workflowID is non-empty.
// It returns a gRPC status error on failure.
func requireNamespace(workflowID string) error {
	if err := domain.ValidateAccess(workflowID, workflowID); err != nil {
		if errors.Is(err, domain.ErrEmptyNamespace) {
			return status.Errorf(codes.InvalidArgument, "workflow_id must not be empty")
		}
		return status.Errorf(codes.PermissionDenied, "cross-namespace access denied")
	}
	return nil
}

// ─── KV RPCs ────────────────────────────────────────────────────────────────

// Set stores or overwrites a key-value pair scoped to the workflow_id namespace.
// TTL is best-effort (Redis EXPIRE): callers must not rely on sub-second precision.
func (h *Handler) Set(ctx context.Context, req *zynaxv1.SetRequest) (*zynaxv1.SetResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}
	if err := requireNamespace(req.GetWorkflowId()); err != nil {
		return nil, err
	}
	var ttl time.Duration
	if req.GetTtlSeconds() > 0 {
		ttl = time.Duration(req.GetTtlSeconds()) * time.Second
	}
	if err := h.kv.Set(ctx, req.GetWorkflowId(), req.GetKey(), req.GetValue(), ttl); err != nil {
		return nil, status.Errorf(codes.Internal, "set: %v", err)
	}
	return &zynaxv1.SetResponse{}, nil
}

// Get retrieves the value for the given key in the workflow_id namespace.
func (h *Handler) Get(ctx context.Context, req *zynaxv1.GetRequest) (*zynaxv1.GetResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}
	if err := requireNamespace(req.GetWorkflowId()); err != nil {
		return nil, err
	}
	val, err := h.kv.Get(ctx, req.GetWorkflowId(), req.GetKey())
	if err != nil {
		if errors.Is(err, domain.ErrKeyNotFound) {
			return nil, status.Errorf(codes.NotFound, "key not found: %q", req.GetKey())
		}
		return nil, status.Errorf(codes.Internal, "get: %v", err)
	}
	return &zynaxv1.GetResponse{Value: val}, nil
}

// Delete removes a key from the workflow_id namespace.
func (h *Handler) Delete(ctx context.Context, req *zynaxv1.DeleteRequest) (*zynaxv1.DeleteResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}
	if err := requireNamespace(req.GetWorkflowId()); err != nil {
		return nil, err
	}
	if err := h.kv.Delete(ctx, req.GetWorkflowId(), req.GetKey()); err != nil {
		return nil, status.Errorf(codes.Internal, "delete: %v", err)
	}
	return &zynaxv1.DeleteResponse{}, nil
}

// ListKeys returns all keys in the workflow_id namespace matching the pattern.
func (h *Handler) ListKeys(ctx context.Context, req *zynaxv1.ListKeysRequest) (*zynaxv1.ListKeysResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}
	if err := requireNamespace(req.GetWorkflowId()); err != nil {
		return nil, err
	}
	pattern := req.GetPrefix()
	if pattern == "" {
		pattern = "*"
	}
	keys, err := h.kv.ListKeys(ctx, req.GetWorkflowId(), pattern)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list_keys: %v", err)
	}
	return &zynaxv1.ListKeysResponse{Keys: keys}, nil
}

// MGet retrieves values for a batch of keys in the workflow_id namespace.
func (h *Handler) MGet(ctx context.Context, req *zynaxv1.MGetRequest) (*zynaxv1.MGetResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}
	if err := requireNamespace(req.GetWorkflowId()); err != nil {
		return nil, err
	}
	vals, err := h.kv.MGet(ctx, req.GetWorkflowId(), req.GetKeys())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "mget: %v", err)
	}
	entries := make([]*zynaxv1.MGetEntry, 0, len(vals))
	for k, v := range vals {
		if v != nil {
			entries = append(entries, &zynaxv1.MGetEntry{Key: k, Value: v})
		}
	}
	return &zynaxv1.MGetResponse{Entries: entries}, nil
}

// MSet stores multiple key-value pairs in the workflow_id namespace.
// Each entry may carry its own TTL; TTL is best-effort (Redis EXPIRE).
func (h *Handler) MSet(ctx context.Context, req *zynaxv1.MSetRequest) (*zynaxv1.MSetResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}
	if err := requireNamespace(req.GetWorkflowId()); err != nil {
		return nil, err
	}
	ns := req.GetWorkflowId()
	// Group entries by TTL and batch each group.
	// Per-entry TTL is supported by calling Set individually.
	for _, e := range req.GetEntries() {
		var ttl time.Duration
		if e.GetTtlSeconds() > 0 {
			ttl = time.Duration(e.GetTtlSeconds()) * time.Second
		}
		if err := h.kv.Set(ctx, ns, e.GetKey(), e.GetValue(), ttl); err != nil {
			return nil, status.Errorf(codes.Internal, "mset entry %q: %v", e.GetKey(), err)
		}
	}
	count := len(req.GetEntries())
	return &zynaxv1.MSetResponse{Count: int32(count)}, nil //nolint:gosec // count is bounded by proto message size limits
}

// DeleteNamespace removes all KV keys in the workflow_id namespace.
// Returns OK even if the namespace has no entries (idempotent).
func (h *Handler) DeleteNamespace(ctx context.Context, req *zynaxv1.DeleteNamespaceRequest) (*zynaxv1.DeleteNamespaceResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}
	if err := requireNamespace(req.GetWorkflowId()); err != nil {
		return nil, err
	}
	if err := h.kv.DeleteNamespace(ctx, req.GetWorkflowId()); err != nil {
		return nil, status.Errorf(codes.Internal, "delete_namespace: %v", err)
	}
	return &zynaxv1.DeleteNamespaceResponse{}, nil
}

// ─── Vector RPCs ─────────────────────────────────────────────────────────────

// StoreVector upserts an embedding in the workflow_id namespace.
// The service generates a UUID vector_id that is returned in the response.
func (h *Handler) StoreVector(ctx context.Context, req *zynaxv1.StoreVectorRequest) (*zynaxv1.StoreVectorResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}
	if err := requireNamespace(req.GetWorkflowId()); err != nil {
		return nil, err
	}
	vectorID := uuid.New().String()
	meta := req.GetMetadata()
	if meta == nil {
		meta = map[string]string{}
	}
	if req.GetText() != "" {
		meta["text"] = req.GetText()
	}
	if err := h.vec.StoreVector(ctx, req.GetWorkflowId(), vectorID, req.GetEmbedding(), meta); err != nil {
		return nil, status.Errorf(codes.Internal, "store_vector: %v", err)
	}
	return &zynaxv1.StoreVectorResponse{VectorId: vectorID}, nil
}

// QueryVector returns the k nearest embeddings in the workflow_id namespace.
func (h *Handler) QueryVector(ctx context.Context, req *zynaxv1.QueryVectorRequest) (*zynaxv1.QueryVectorResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}
	if err := requireNamespace(req.GetWorkflowId()); err != nil {
		return nil, err
	}
	results, err := h.vec.QueryVector(ctx, req.GetWorkflowId(), req.GetEmbedding(), int(req.GetTopK()), "")
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query_vector: %v", err)
	}
	vr := make([]*zynaxv1.VectorResult, len(results))
	for i, r := range results {
		vr[i] = &zynaxv1.VectorResult{
			VectorId:        r.ID,
			SimilarityScore: r.Score,
			Metadata:        r.Metadata,
		}
	}
	return &zynaxv1.QueryVectorResponse{Results: vr}, nil
}

// DeleteVector removes an embedding from the workflow_id namespace.
func (h *Handler) DeleteVector(ctx context.Context, req *zynaxv1.DeleteVectorRequest) (*zynaxv1.DeleteVectorResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}
	if err := requireNamespace(req.GetWorkflowId()); err != nil {
		return nil, err
	}
	if err := h.vec.DeleteVector(ctx, req.GetWorkflowId(), req.GetVectorId()); err != nil {
		return nil, status.Errorf(codes.Internal, "delete_vector: %v", err)
	}
	return &zynaxv1.DeleteVectorResponse{}, nil
}
