// SPDX-License-Identifier: Apache-2.0
// Package memory_service provides BDD contract tests for MemoryService.
package memory_service_test

import (
	"context"
	"fmt"
	"math"
	"sort"
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

// ─── In-memory stub ──────────────────────────────────────────────────────────

type kvEntry struct {
	value     []byte
	expiresAt time.Time // zero = no expiry
}

type vectorEntry struct {
	id        string
	embedding []float32
	text      string
	metadata  map[string]string
	wfID      string
}

type memStub struct {
	zynaxv1.UnimplementedMemoryServiceServer
	mu      sync.Mutex
	kv      map[string]map[string]*kvEntry // wfID -> key -> entry
	vectors map[string][]*vectorEntry       // wfID -> entries
	nextVec int
}

func newMemStub() *memStub {
	return &memStub{
		kv:      make(map[string]map[string]*kvEntry),
		vectors: make(map[string][]*vectorEntry),
	}
}

func (s *memStub) isExpired(e *kvEntry) bool {
	return !e.expiresAt.IsZero() && time.Now().After(e.expiresAt)
}

func (s *memStub) Set(_ context.Context, req *zynaxv1.SetRequest) (*zynaxv1.SetResponse, error) {
	if req.WorkflowId == "" {
		return nil, status.Error(codes.InvalidArgument, "workflow_id must not be empty")
	}
	if req.Key == "" {
		return nil, status.Error(codes.InvalidArgument, "key must not be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.kv[req.WorkflowId] == nil {
		s.kv[req.WorkflowId] = make(map[string]*kvEntry)
	}
	e := &kvEntry{value: req.Value}
	if req.TtlSeconds > 0 {
		e.expiresAt = time.Now().Add(time.Duration(req.TtlSeconds) * time.Second)
	}
	s.kv[req.WorkflowId][req.Key] = e

	return &zynaxv1.SetResponse{StoredAt: timestamppb.Now()}, nil
}

func (s *memStub) Get(_ context.Context, req *zynaxv1.GetRequest) (*zynaxv1.GetResponse, error) {
	if req.WorkflowId == "" {
		return nil, status.Error(codes.InvalidArgument, "workflow_id must not be empty")
	}
	if req.Key == "" {
		return nil, status.Error(codes.InvalidArgument, "key must not be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	wfMap, ok := s.kv[req.WorkflowId]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "key %q not found", req.Key)
	}
	e, ok := wfMap[req.Key]
	if !ok || s.isExpired(e) {
		return nil, status.Errorf(codes.NotFound, "key %q not found", req.Key)
	}
	return &zynaxv1.GetResponse{Value: e.value}, nil
}

func (s *memStub) Delete(_ context.Context, req *zynaxv1.DeleteRequest) (*zynaxv1.DeleteResponse, error) {
	if req.WorkflowId == "" {
		return nil, status.Error(codes.InvalidArgument, "workflow_id must not be empty")
	}
	if req.Key == "" {
		return nil, status.Error(codes.InvalidArgument, "key must not be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	wfMap, ok := s.kv[req.WorkflowId]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "key %q not found", req.Key)
	}
	e, ok := wfMap[req.Key]
	if !ok || s.isExpired(e) {
		return nil, status.Errorf(codes.NotFound, "key %q not found", req.Key)
	}
	delete(wfMap, req.Key)
	return &zynaxv1.DeleteResponse{DeletedAt: timestamppb.Now()}, nil
}

func (s *memStub) ListKeys(_ context.Context, req *zynaxv1.ListKeysRequest) (*zynaxv1.ListKeysResponse, error) {
	if req.WorkflowId == "" {
		return nil, status.Error(codes.InvalidArgument, "workflow_id must not be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	wfMap := s.kv[req.WorkflowId]
	var keys []string
	for k, e := range wfMap {
		if !s.isExpired(e) {
			if req.Prefix == "" || strings.HasPrefix(k, req.Prefix) {
				keys = append(keys, k)
			}
		}
	}
	sort.Strings(keys)
	return &zynaxv1.ListKeysResponse{Keys: keys}, nil
}

func cosine(a, b []float32) float32 {
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return float32(dot / (math.Sqrt(normA) * math.Sqrt(normB)))
}

func (s *memStub) StoreVector(_ context.Context, req *zynaxv1.StoreVectorRequest) (*zynaxv1.StoreVectorResponse, error) {
	if req.WorkflowId == "" {
		return nil, status.Error(codes.InvalidArgument, "workflow_id must not be empty")
	}
	if len(req.Embedding) == 0 {
		return nil, status.Error(codes.InvalidArgument, "embedding must not be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextVec++
	vecID := fmt.Sprintf("vec-%04d", s.nextVec)
	s.vectors[req.WorkflowId] = append(s.vectors[req.WorkflowId], &vectorEntry{
		id:        vecID,
		embedding: req.Embedding,
		text:      req.Text,
		metadata:  req.Metadata,
		wfID:      req.WorkflowId,
	})
	return &zynaxv1.StoreVectorResponse{VectorId: vecID, StoredAt: timestamppb.Now()}, nil
}

func (s *memStub) QueryVector(_ context.Context, req *zynaxv1.QueryVectorRequest) (*zynaxv1.QueryVectorResponse, error) {
	if req.WorkflowId == "" {
		return nil, status.Error(codes.InvalidArgument, "workflow_id must not be empty")
	}
	if req.TopK == 0 {
		return nil, status.Error(codes.InvalidArgument, "top_k must be greater than zero")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	vecs := s.vectors[req.WorkflowId]
	type scored struct {
		v     *vectorEntry
		score float32
	}
	var results []scored
	for _, v := range vecs {
		if len(v.embedding) == len(req.Embedding) {
			score := cosine(v.embedding, req.Embedding)
			results = append(results, scored{v, score})
		}
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})
	topK := int(req.TopK)
	if topK > len(results) {
		topK = len(results)
	}
	var out []*zynaxv1.VectorResult
	for i := 0; i < topK; i++ {
		out = append(out, &zynaxv1.VectorResult{
			VectorId:        results[i].v.id,
			Text:            results[i].v.text,
			Metadata:        results[i].v.metadata,
			SimilarityScore: results[i].score,
		})
	}
	return &zynaxv1.QueryVectorResponse{Results: out}, nil
}

func (s *memStub) DeleteVector(_ context.Context, req *zynaxv1.DeleteVectorRequest) (*zynaxv1.DeleteVectorResponse, error) {
	if req.WorkflowId == "" {
		return nil, status.Error(codes.InvalidArgument, "workflow_id must not be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	vecs := s.vectors[req.WorkflowId]
	for i, v := range vecs {
		if v.id == req.VectorId {
			s.vectors[req.WorkflowId] = append(vecs[:i], vecs[i+1:]...)
			return &zynaxv1.DeleteVectorResponse{DeletedAt: timestamppb.Now()}, nil
		}
	}
	return nil, status.Errorf(codes.NotFound, "vector %q not found", req.VectorId)
}

// ─── Test context ─────────────────────────────────────────────────────────────

type memCtx struct {
	client       zynaxv1.MemoryServiceClient
	stub         *memStub
	getResp      *zynaxv1.GetResponse
	listResp     *zynaxv1.ListKeysResponse
	queryResp    *zynaxv1.QueryVectorResponse
	storeVecResp *zynaxv1.StoreVectorResponse
	grpcErr      error
	// Pending request state for input-validation scenarios
	pendingSetReq         *zynaxv1.SetRequest
	pendingStoreVectorReq *zynaxv1.StoreVectorRequest
	pendingQueryVectorReq *zynaxv1.QueryVectorRequest
	// Stored vector id for DeleteVector scenario
	storedVecID string
}

type godogMKey struct{}

// ─── TestFeatures ─────────────────────────────────────────────────────────────

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		Name: "memory_service",
		ScenarioInitializer: func(sc *godog.ScenarioContext) {
			srv, dialer := testserver.NewBufconnServer(t)
			stub := newMemStub()
			zynaxv1.RegisterMemoryServiceServer(srv, stub)

			conn, err := grpc.NewClient(
				"passthrough://bufnet",
				grpc.WithContextDialer(dialer),
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			)
			if err != nil {
				t.Fatalf("failed to dial: %v", err)
			}
			t.Cleanup(func() { conn.Close() })

			tc := &memCtx{
				client: zynaxv1.NewMemoryServiceClient(conn),
				stub:   stub,
			}

			sc.Before(func(ctx context.Context, scenario *godog.Scenario) (context.Context, error) {
				tc.getResp = nil
				tc.listResp = nil
				tc.queryResp = nil
				tc.storeVecResp = nil
				tc.grpcErr = nil
				tc.pendingSetReq = nil
				tc.pendingStoreVectorReq = nil
				tc.pendingQueryVectorReq = nil
				tc.storedVecID = ""
				// Reset stub store per scenario
				stub.mu.Lock()
				stub.kv = make(map[string]map[string]*kvEntry)
				stub.vectors = make(map[string][]*vectorEntry)
				stub.nextVec = 0
				stub.mu.Unlock()
				return context.WithValue(ctx, godogMKey{}, t), nil
			})

			// ── Given steps ──────────────────────────────────────────────────────

			sc.Step(`^a MemoryService is running on a test gRPC server$`, func() error {
				return nil
			})

			sc.Step(`^workflow "([^"]*)" is running$`, func(_ string) error {
				return nil
			})

			sc.Step(`^Set is called with key "([^"]*)" value "([^"]*)" scoped to "([^"]*)"$`, func(key, value, wfID string) error {
				_, err := tc.client.Set(context.Background(), &zynaxv1.SetRequest{
					WorkflowId: wfID,
					Key:        key,
					Value:      []byte(value),
				})
				return err
			})

			sc.Step(`^Set is called with key "([^"]*)" value "([^"]*)" scoped to workflow "([^"]*)"$`, func(key, value, wfID string) error {
				_, err := tc.client.Set(context.Background(), &zynaxv1.SetRequest{
					WorkflowId: wfID,
					Key:        key,
					Value:      []byte(value),
				})
				return err
			})

			sc.Step(`^Set has been called with key "([^"]*)" value "([^"]*)" scoped to "([^"]*)"$`, func(key, value, wfID string) error {
				_, err := tc.client.Set(context.Background(), &zynaxv1.SetRequest{
					WorkflowId: wfID,
					Key:        key,
					Value:      []byte(value),
				})
				return err
			})

			sc.Step(`^Set has been called with key "([^"]*)" scoped to "([^"]*)"$`, func(key, wfID string) error {
				_, err := tc.client.Set(context.Background(), &zynaxv1.SetRequest{
					WorkflowId: wfID,
					Key:        key,
					Value:      []byte("placeholder"),
				})
				return err
			})

			sc.Step(`^Set is called with key "([^"]*)" value "([^"]*)" ttl_seconds (\d+) scoped to "([^"]*)"$`, func(key, value string, ttl int, wfID string) error {
				_, err := tc.client.Set(context.Background(), &zynaxv1.SetRequest{
					WorkflowId: wfID,
					Key:        key,
					Value:      []byte(value),
					TtlSeconds: int32(ttl),
				})
				return err
			})

			sc.Step(`^Set is called with key "([^"]*)" value "([^"]*)" and no TTL scoped to "([^"]*)"$`, func(key, value, wfID string) error {
				_, err := tc.client.Set(context.Background(), &zynaxv1.SetRequest{
					WorkflowId: wfID,
					Key:        key,
					Value:      []byte(value),
				})
				return err
			})

			sc.Step(`^(\d+) seconds elapse$`, func(n int) error {
				time.Sleep(time.Duration(n) * time.Second)
				return nil
			})

			sc.Step(`^a vector with embedding \[([0-9., ]+)\] and text "([^"]*)"$`, func(embStr, text string) error {
				// Parse embedding from string like "0.1, 0.2, 0.3"
				tc.pendingStoreVectorReq = &zynaxv1.StoreVectorRequest{
					Embedding: parseEmbedding(embStr),
					Text:      text,
				}
				return nil
			})

			sc.Step(`^vectors "([^"]*)" "([^"]*)" "([^"]*)" are stored with varying similarity to the query$`, func(_, _, _ string) error {
				// Store vectors A, B, C with different embeddings
				for _, v := range []struct {
					text      string
					embedding []float32
				}{
					{"A", []float32{1.0, 0.0, 0.0}},
					{"B", []float32{0.7, 0.7, 0.0}},
					{"C", []float32{0.0, 0.0, 1.0}},
				} {
					if _, err := tc.client.StoreVector(context.Background(), &zynaxv1.StoreVectorRequest{
						WorkflowId: "wf-vec-test",
						Embedding:  v.embedding,
						Text:       v.text,
					}); err != nil {
						return err
					}
				}
				tc.pendingQueryVectorReq = &zynaxv1.QueryVectorRequest{
					WorkflowId: "wf-vec-test",
					Embedding:  []float32{1.0, 0.1, 0.0},
					TopK:       3,
				}
				return nil
			})

			sc.Step(`^(\d+) vectors are stored scoped to "([^"]*)"$`, func(n int, wfID string) error {
				for i := 0; i < n; i++ {
					if _, err := tc.client.StoreVector(context.Background(), &zynaxv1.StoreVectorRequest{
						WorkflowId: wfID,
						Embedding:  []float32{float32(i), float32(i + 1)},
						Text:       fmt.Sprintf("vector-%d", i),
					}); err != nil {
						return err
					}
				}
				return nil
			})

			sc.Step(`^a vector is stored scoped to workflow "([^"]*)"$`, func(wfID string) error {
				resp, err := tc.client.StoreVector(context.Background(), &zynaxv1.StoreVectorRequest{
					WorkflowId: wfID,
					Embedding:  []float32{0.5, 0.5},
					Text:       "test vector",
				})
				if err != nil {
					return err
				}
				tc.storedVecID = resp.VectorId
				return nil
			})

			sc.Step(`^a vector with id "([^"]*)" is stored scoped to "([^"]*)"$`, func(_, wfID string) error {
				resp, err := tc.client.StoreVector(context.Background(), &zynaxv1.StoreVectorRequest{
					WorkflowId: wfID,
					Embedding:  []float32{0.1, 0.2, 0.3},
					Text:       "test vector",
				})
				if err != nil {
					return err
				}
				tc.storedVecID = resp.VectorId
				return nil
			})

			sc.Step(`^a SetRequest with workflow_id set to ""$`, func() error {
				tc.pendingSetReq = &zynaxv1.SetRequest{WorkflowId: "", Key: "k", Value: []byte("v")}
				return nil
			})

			sc.Step(`^a SetRequest with key set to ""$`, func() error {
				tc.pendingSetReq = &zynaxv1.SetRequest{WorkflowId: "wf-x", Key: "", Value: []byte("v")}
				return nil
			})

			sc.Step(`^a StoreVectorRequest with no embedding values$`, func() error {
				tc.pendingStoreVectorReq = &zynaxv1.StoreVectorRequest{WorkflowId: "wf-x", Embedding: nil}
				return nil
			})

			sc.Step(`^a StoreVectorRequest with workflow_id set to ""$`, func() error {
				tc.pendingStoreVectorReq = &zynaxv1.StoreVectorRequest{WorkflowId: "", Embedding: []float32{0.1}}
				return nil
			})

			sc.Step(`^a QueryVectorRequest with top_k set to 0$`, func() error {
				tc.pendingQueryVectorReq = &zynaxv1.QueryVectorRequest{WorkflowId: "wf-x", Embedding: []float32{0.1}, TopK: 0}
				return nil
			})

			// ── When steps ───────────────────────────────────────────────────────

			sc.Step(`^Get is called with key "([^"]*)" scoped to "([^"]*)"$`, func(key, wfID string) error {
				tc.getResp, tc.grpcErr = tc.client.Get(context.Background(), &zynaxv1.GetRequest{
					WorkflowId: wfID,
					Key:        key,
				})
				return nil
			})

			sc.Step(`^Get is called with key "([^"]*)" scoped to workflow "([^"]*)"$`, func(key, wfID string) error {
				tc.getResp, tc.grpcErr = tc.client.Get(context.Background(), &zynaxv1.GetRequest{
					WorkflowId: wfID,
					Key:        key,
				})
				return nil
			})

			sc.Step(`^Set is called again with key "([^"]*)" value "([^"]*)" scoped to "([^"]*)"$`, func(key, value, wfID string) error {
				_, tc.grpcErr = tc.client.Set(context.Background(), &zynaxv1.SetRequest{
					WorkflowId: wfID,
					Key:        key,
					Value:      []byte(value),
				})
				return nil
			})

			sc.Step(`^Delete is called with key "([^"]*)" scoped to "([^"]*)"$`, func(key, wfID string) error {
				_, tc.grpcErr = tc.client.Delete(context.Background(), &zynaxv1.DeleteRequest{
					WorkflowId: wfID,
					Key:        key,
				})
				return nil
			})

			sc.Step(`^Get for key "([^"]*)" scoped to "([^"]*)" returns NOT_FOUND$`, func(key, wfID string) error {
				_, err := tc.client.Get(context.Background(), &zynaxv1.GetRequest{
					WorkflowId: wfID,
					Key:        key,
				})
				if err == nil {
					return fmt.Errorf("expected NOT_FOUND, got nil error")
				}
				if s, ok := status.FromError(err); !ok || s.Code() != codes.NotFound {
					return fmt.Errorf("expected NOT_FOUND, got: %v", err)
				}
				return nil
			})

			sc.Step(`^ListKeys is called scoped to workflow "([^"]*)"$`, func(wfID string) error {
				tc.listResp, tc.grpcErr = tc.client.ListKeys(context.Background(), &zynaxv1.ListKeysRequest{
					WorkflowId: wfID,
				})
				return nil
			})

			sc.Step(`^ListKeys is called scoped to "([^"]*)" with prefix "([^"]*)"$`, func(wfID, prefix string) error {
				tc.listResp, tc.grpcErr = tc.client.ListKeys(context.Background(), &zynaxv1.ListKeysRequest{
					WorkflowId: wfID,
					Prefix:     prefix,
				})
				return nil
			})

			sc.Step(`^StoreVector is called with the embedding scoped to "([^"]*)"$`, func(wfID string) error {
				if tc.pendingStoreVectorReq != nil {
					tc.pendingStoreVectorReq.WorkflowId = wfID
					tc.storeVecResp, tc.grpcErr = tc.client.StoreVector(context.Background(), tc.pendingStoreVectorReq)
				}
				return nil
			})

			sc.Step(`^QueryVector is called with embedding \[([0-9., ]+)\] top_k (\d+) scoped to "([^"]*)"$`, func(embStr string, topK int, wfID string) error {
				tc.queryResp, tc.grpcErr = tc.client.QueryVector(context.Background(), &zynaxv1.QueryVectorRequest{
					WorkflowId: wfID,
					Embedding:  parseEmbedding(embStr),
					TopK:       int32(topK),
				})
				return nil
			})

			sc.Step(`^QueryVector is called with top_k (\d+)$`, func(topK int) error {
				tc.queryResp, tc.grpcErr = tc.client.QueryVector(context.Background(), tc.pendingQueryVectorReq)
				return nil
			})

			sc.Step(`^QueryVector is called with top_k (\d+) scoped to "([^"]*)"$`, func(topK int, wfID string) error {
				tc.queryResp, tc.grpcErr = tc.client.QueryVector(context.Background(), &zynaxv1.QueryVectorRequest{
					WorkflowId: wfID,
					Embedding:  []float32{0.5, 0.5},
					TopK:       int32(topK),
				})
				return nil
			})

			sc.Step(`^QueryVector is called scoped to workflow "([^"]*)"$`, func(wfID string) error {
				tc.queryResp, tc.grpcErr = tc.client.QueryVector(context.Background(), &zynaxv1.QueryVectorRequest{
					WorkflowId: wfID,
					Embedding:  []float32{0.5, 0.5},
					TopK:       5,
				})
				return nil
			})

			sc.Step(`^QueryVector is called scoped to workflow "([^"]*)" with top_k (\d+)$`, func(wfID string, topK int) error {
				tc.queryResp, tc.grpcErr = tc.client.QueryVector(context.Background(), &zynaxv1.QueryVectorRequest{
					WorkflowId: wfID,
					Embedding:  []float32{0.5, 0.5},
					TopK:       int32(topK),
				})
				return nil
			})

			sc.Step(`^DeleteVector is called with id "([^"]*)" scoped to "([^"]*)"$`, func(vecID, wfID string) error {
				actualID := vecID
				if vecID == "vec-001" && tc.storedVecID != "" {
					actualID = tc.storedVecID
				}
				_, tc.grpcErr = tc.client.DeleteVector(context.Background(), &zynaxv1.DeleteVectorRequest{
					WorkflowId: wfID,
					VectorId:   actualID,
				})
				return nil
			})

			sc.Step(`^QueryVector is called with a similar embedding scoped to "([^"]*)"$`, func(wfID string) error {
				tc.queryResp, tc.grpcErr = tc.client.QueryVector(context.Background(), &zynaxv1.QueryVectorRequest{
					WorkflowId: wfID,
					Embedding:  []float32{0.11, 0.21, 0.31},
					TopK:       5,
				})
				return nil
			})

			sc.Step(`^Set is called$`, func() error {
				if tc.pendingSetReq != nil {
					_, tc.grpcErr = tc.client.Set(context.Background(), tc.pendingSetReq)
				}
				return nil
			})

			sc.Step(`^StoreVector is called$`, func() error {
				if tc.pendingStoreVectorReq != nil {
					_, tc.grpcErr = tc.client.StoreVector(context.Background(), tc.pendingStoreVectorReq)
				}
				return nil
			})

			sc.Step(`^QueryVector is called$`, func() error {
				if tc.pendingQueryVectorReq != nil {
					tc.queryResp, tc.grpcErr = tc.client.QueryVector(context.Background(), tc.pendingQueryVectorReq)
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

			sc.Step(`^the gRPC status is NOT_FOUND$`, func() error {
				if s, ok := status.FromError(tc.grpcErr); !ok || s.Code() != codes.NotFound {
					return fmt.Errorf("expected NOT_FOUND, got: %v", tc.grpcErr)
				}
				return nil
			})

			sc.Step(`^the gRPC status is INVALID_ARGUMENT$`, func() error {
				if s, ok := status.FromError(tc.grpcErr); !ok || s.Code() != codes.InvalidArgument {
					return fmt.Errorf("expected INVALID_ARGUMENT, got: %v", tc.grpcErr)
				}
				return nil
			})

			sc.Step(`^the response value is "([^"]*)"$`, func(expected string) error {
				if tc.getResp == nil {
					return fmt.Errorf("get response is nil")
				}
				got := string(tc.getResp.Value)
				if got != expected {
					return fmt.Errorf("expected value %q, got %q", expected, got)
				}
				return nil
			})

			sc.Step(`^the error message contains "([^"]*)"$`, func(fragment string) error {
				if tc.grpcErr == nil {
					return fmt.Errorf("expected error containing %q, got no error", fragment)
				}
				if !strings.Contains(tc.grpcErr.Error(), fragment) {
					return fmt.Errorf("expected error to contain %q, got: %s", fragment, tc.grpcErr.Error())
				}
				return nil
			})

			sc.Step(`^the response contains key "([^"]*)"$`, func(key string) error {
				if tc.listResp == nil {
					return fmt.Errorf("list response is nil")
				}
				for _, k := range tc.listResp.Keys {
					if k == key {
						return nil
					}
				}
				return fmt.Errorf("key %q not found in response %v", key, tc.listResp.Keys)
			})

			sc.Step(`^the response does not contain key "([^"]*)"$`, func(key string) error {
				if tc.listResp == nil {
					return nil
				}
				for _, k := range tc.listResp.Keys {
					if k == key {
						return fmt.Errorf("key %q found in response but should not be", key)
					}
				}
				return nil
			})

			sc.Step(`^the response contains no keys$`, func() error {
				if tc.listResp != nil && len(tc.listResp.Keys) > 0 {
					return fmt.Errorf("expected no keys, got %v", tc.listResp.Keys)
				}
				return nil
			})

			sc.Step(`^the response contains (\d+) VectorResult$`, func(n int) error {
				if tc.queryResp == nil {
					return fmt.Errorf("query response is nil")
				}
				if len(tc.queryResp.Results) != n {
					return fmt.Errorf("expected %d VectorResult(s), got %d", n, len(tc.queryResp.Results))
				}
				return nil
			})

			sc.Step(`^the top VectorResult text is "([^"]*)"$`, func(text string) error {
				if tc.queryResp == nil || len(tc.queryResp.Results) == 0 {
					return fmt.Errorf("no VectorResults")
				}
				if tc.queryResp.Results[0].Text != text {
					return fmt.Errorf("expected top result text %q, got %q", text, tc.queryResp.Results[0].Text)
				}
				return nil
			})

			sc.Step(`^the top VectorResult similarity_score is greater than (\d+\.\d+)$`, func(threshold float64) error {
				if tc.queryResp == nil || len(tc.queryResp.Results) == 0 {
					return fmt.Errorf("no VectorResults")
				}
				score := float64(tc.queryResp.Results[0].SimilarityScore)
				if score <= threshold {
					return fmt.Errorf("expected similarity_score > %f, got %f", threshold, score)
				}
				return nil
			})

			sc.Step(`^the results are ordered by similarity_score descending$`, func() error {
				if tc.queryResp == nil {
					return fmt.Errorf("query response is nil")
				}
				for i := 1; i < len(tc.queryResp.Results); i++ {
					if tc.queryResp.Results[i].SimilarityScore > tc.queryResp.Results[i-1].SimilarityScore {
						return fmt.Errorf("results not ordered descending: index %d score %f > index %d score %f",
							i, tc.queryResp.Results[i].SimilarityScore, i-1, tc.queryResp.Results[i-1].SimilarityScore)
					}
				}
				return nil
			})

			sc.Step(`^the response contains exactly (\d+) VectorResults$`, func(n int) error {
				if tc.queryResp == nil {
					return fmt.Errorf("query response is nil")
				}
				if len(tc.queryResp.Results) != n {
					return fmt.Errorf("expected exactly %d VectorResults, got %d", n, len(tc.queryResp.Results))
				}
				return nil
			})

			sc.Step(`^the response contains no VectorResults$`, func() error {
				if tc.queryResp != nil && len(tc.queryResp.Results) > 0 {
					return fmt.Errorf("expected no VectorResults, got %d", len(tc.queryResp.Results))
				}
				return nil
			})

			sc.Step(`^the response does not contain vector id "([^"]*)"$`, func(vecID string) error {
				if tc.queryResp == nil {
					return nil
				}
				for _, r := range tc.queryResp.Results {
					if r.VectorId == tc.storedVecID || r.VectorId == vecID {
						return fmt.Errorf("vector id %q still present after deletion", vecID)
					}
				}
				return nil
			})

			sc.Step(`^the error message mentions "([^"]*)"$`, func(fragment string) error {
				if tc.grpcErr == nil {
					return fmt.Errorf("expected error mentioning %q, got no error", fragment)
				}
				if !strings.Contains(tc.grpcErr.Error(), fragment) {
					return fmt.Errorf("expected error to mention %q, got: %s", fragment, tc.grpcErr.Error())
				}
				return nil
			})
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../features/memory_service.feature"},
			TestingT: t,
		},
	}

	if suite.Run() != 0 {
		t.Fatal("BDD scenarios failed")
	}
}

// parseEmbedding parses "0.1, 0.2, 0.3" into []float32.
func parseEmbedding(s string) []float32 {
	parts := strings.Split(s, ",")
	var out []float32
	for _, p := range parts {
		p = strings.TrimSpace(p)
		var f float64
		fmt.Sscanf(p, "%f", &f)
		out = append(out, float32(f))
	}
	return out
}
