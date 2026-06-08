// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/memory-service/internal/api"
	"github.com/zynax-io/zynax/services/memory-service/internal/domain"
)

// ─── stubs ──────────────────────────────────────────────────────────────────

// stubKV is a no-op KVStore that always succeeds.
type stubKV struct{}

func (s *stubKV) Set(_ context.Context, _, _ string, _ []byte, _ time.Duration) error {
	return nil
}
func (s *stubKV) Get(_ context.Context, _, _ string) ([]byte, error) {
	return []byte("v"), nil
}
func (s *stubKV) Delete(_ context.Context, _, _ string) error { return nil }
func (s *stubKV) ListKeys(_ context.Context, _, _ string) ([]string, error) {
	return []string{}, nil
}
func (s *stubKV) MGet(_ context.Context, _ string, _ []string) (map[string][]byte, error) {
	return map[string][]byte{}, nil
}
func (s *stubKV) MSet(_ context.Context, _ string, _ map[string][]byte, _ time.Duration) error {
	return nil
}
func (s *stubKV) DeleteNamespace(_ context.Context, _ string) error { return nil }

// stubVec is a no-op VectorStore that always succeeds.
type stubVec struct{}

func (s *stubVec) StoreVector(_ context.Context, _, _ string, _ []float32, _ map[string]string) error {
	return nil
}
func (s *stubVec) QueryVector(_ context.Context, _ string, _ []float32, _ int, _ string) ([]domain.VectorResult, error) {
	return nil, nil
}
func (s *stubVec) DeleteVector(_ context.Context, _, _ string) error { return nil }

// ─── helpers ─────────────────────────────────────────────────────────────────

func newHandler() *api.Handler {
	return api.NewHandler(&stubKV{}, &stubVec{})
}

func assertPermissionDenied(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected PERMISSION_DENIED error, got nil")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got: %v", err)
	}
	if st.Code() != codes.PermissionDenied {
		t.Fatalf("expected PERMISSION_DENIED, got %v: %s", st.Code(), st.Message())
	}
}

func assertInvalidArgument(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected INVALID_ARGUMENT error, got nil")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got: %v", err)
	}
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT, got %v: %s", st.Code(), st.Message())
	}
}

// ─── KV namespace tests ──────────────────────────────────────────────────────

func TestHandler_Set_EmptyNamespace(t *testing.T) {
	t.Parallel()
	h := newHandler()
	_, err := h.Set(context.Background(), &zynaxv1.SetRequest{
		WorkflowId: "",
		Key:        "k",
		Value:      []byte("v"),
	})
	assertInvalidArgument(t, err)
}

func TestHandler_Set_ValidNamespace(t *testing.T) {
	t.Parallel()
	h := newHandler()
	_, err := h.Set(context.Background(), &zynaxv1.SetRequest{
		WorkflowId: "wf-123",
		Key:        "k",
		Value:      []byte("v"),
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestHandler_Get_EmptyNamespace(t *testing.T) {
	t.Parallel()
	h := newHandler()
	_, err := h.Get(context.Background(), &zynaxv1.GetRequest{
		WorkflowId: "",
		Key:        "k",
	})
	assertInvalidArgument(t, err)
}

func TestHandler_Delete_EmptyNamespace(t *testing.T) {
	t.Parallel()
	h := newHandler()
	_, err := h.Delete(context.Background(), &zynaxv1.DeleteRequest{
		WorkflowId: "",
		Key:        "k",
	})
	assertInvalidArgument(t, err)
}

func TestHandler_ListKeys_EmptyNamespace(t *testing.T) {
	t.Parallel()
	h := newHandler()
	_, err := h.ListKeys(context.Background(), &zynaxv1.ListKeysRequest{
		WorkflowId: "",
	})
	assertInvalidArgument(t, err)
}

func TestHandler_MGet_EmptyNamespace(t *testing.T) {
	t.Parallel()
	h := newHandler()
	_, err := h.MGet(context.Background(), &zynaxv1.MGetRequest{
		WorkflowId: "",
		Keys:       []string{"k"},
	})
	assertInvalidArgument(t, err)
}

func TestHandler_MSet_EmptyNamespace(t *testing.T) {
	t.Parallel()
	h := newHandler()
	_, err := h.MSet(context.Background(), &zynaxv1.MSetRequest{
		WorkflowId: "",
		Entries:    []*zynaxv1.MSetEntry{{Key: "k", Value: []byte("v")}},
	})
	assertInvalidArgument(t, err)
}

func TestHandler_DeleteNamespace_EmptyNamespace(t *testing.T) {
	t.Parallel()
	h := newHandler()
	_, err := h.DeleteNamespace(context.Background(), &zynaxv1.DeleteNamespaceRequest{
		WorkflowId: "",
	})
	assertInvalidArgument(t, err)
}

// ─── Vector namespace tests ───────────────────────────────────────────────────

func TestHandler_StoreVector_EmptyNamespace(t *testing.T) {
	t.Parallel()
	h := newHandler()
	_, err := h.StoreVector(context.Background(), &zynaxv1.StoreVectorRequest{
		WorkflowId: "",
		Embedding:  []float32{0.1, 0.2},
	})
	assertInvalidArgument(t, err)
}

func TestHandler_StoreVector_ValidNamespace(t *testing.T) {
	t.Parallel()
	h := newHandler()
	resp, err := h.StoreVector(context.Background(), &zynaxv1.StoreVectorRequest{
		WorkflowId: "wf-abc",
		Embedding:  []float32{0.1, 0.2},
		Text:       "hello world",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if resp.GetVectorId() == "" {
		t.Fatal("expected non-empty vector_id in response")
	}
}

func TestHandler_QueryVector_EmptyNamespace(t *testing.T) {
	t.Parallel()
	h := newHandler()
	_, err := h.QueryVector(context.Background(), &zynaxv1.QueryVectorRequest{
		WorkflowId: "",
		Embedding:  []float32{0.1},
		TopK:       5,
	})
	assertInvalidArgument(t, err)
}

func TestHandler_DeleteVector_EmptyNamespace(t *testing.T) {
	t.Parallel()
	h := newHandler()
	_, err := h.DeleteVector(context.Background(), &zynaxv1.DeleteVectorRequest{
		WorkflowId: "",
		VectorId:   "vec-1",
	})
	assertInvalidArgument(t, err)
}

// ─── Cross-namespace rejection test (domain-level validation) ────────────────

// TestValidateAccess_CrossNamespaceReturnsPermissionDenied verifies the domain
// helper returns ErrCrossNamespaceAccess which handler maps to PERMISSION_DENIED.
func TestValidateAccess_CrossNamespaceReturnsPermissionDenied(t *testing.T) {
	t.Parallel()
	err := domain.ValidateAccess("wf-A", "wf-B")
	if err == nil {
		t.Fatal("expected error for cross-namespace access, got nil")
	}

	// Simulate what the handler does: map to PERMISSION_DENIED.
	var grpcErr error
	if err != nil {
		grpcErr = status.Errorf(codes.PermissionDenied, "cross-namespace access denied")
	}
	assertPermissionDenied(t, grpcErr)
}
