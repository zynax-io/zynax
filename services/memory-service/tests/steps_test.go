// SPDX-License-Identifier: Apache-2.0

//go:build integration

// Package memory_service_bdd_test provides BDD contract tests for memory-service
// driven by godog against real Redis + Postgres/pgvector containers (testcontainers-go).
// Run with: GOWORK=off go test -tags integration -v -timeout 300s ./tests/...
package memory_service_bdd_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cucumber/godog"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/memory-service/internal/api"
	"github.com/zynax-io/zynax/services/memory-service/internal/infrastructure"
	"github.com/zynax-io/zynax/services/memory-service/internal/infrastructure/postgres"
)

func startRedis(t *testing.T) *redis.Client {
	t.Helper()
	ctx := context.Background()
	ctr, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "redis:7-alpine",
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForLog("Ready to accept connections"),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("start redis container: %v", err)
	}
	t.Cleanup(func() { _ = ctr.Terminate(ctx) })
	host, err := ctr.Host(ctx)
	if err != nil {
		t.Fatalf("redis host: %v", err)
	}
	port, err := ctr.MappedPort(ctx, "6379")
	if err != nil {
		t.Fatalf("redis port: %v", err)
	}
	return redis.NewClient(&redis.Options{Addr: fmt.Sprintf("%s:%s", host, port.Port())})
}

func startPostgres(t *testing.T) *postgres.VectorStore {
	t.Helper()
	ctx := context.Background()
	ctr, err := tcpostgres.Run(ctx, "pgvector/pgvector:pg16",
		tcpostgres.WithDatabase("memory_bdd"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start pgvector container: %v", err)
	}
	t.Cleanup(func() { _ = ctr.Terminate(ctx) })
	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	store, err := postgres.New(ctx, dsn)
	if err != nil {
		t.Fatalf("open vector store: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

type bddCtx struct {
	handler      *api.Handler
	lastGRPCErr  error
	lastGetResp  *zynaxv1.GetResponse
	lastQueryRes []*zynaxv1.VectorResult
}

func initScenario(h *api.Handler) func(*godog.ScenarioContext) {
	return func(sc *godog.ScenarioContext) {
		tc := &bddCtx{handler: h}

		sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
			tc.lastGRPCErr = nil
			tc.lastGetResp = nil
			tc.lastQueryRes = nil
			return ctx, nil
		})

		sc.Step(`^the memory service is running$`, func() error {
			return nil
		})

		sc.Step(`^workflow "([^"]*)" is active$`, func(_ string) error {
			return nil
		})

		sc.Step(`^an entry with key="([^"]*)" value="([^"]*)" ttl=1s is set in workflow "([^"]*)"$`,
			func(key, value, wfID string) error {
				_, err := tc.handler.Set(context.Background(), &zynaxv1.SetRequest{
					WorkflowId: wfID, Key: key, Value: []byte(value), TtlSeconds: 1,
				})
				return err
			})

		sc.Step(`^a vector is stored in workflow "([^"]*)"$`, func(wfID string) error {
			_, err := tc.handler.StoreVector(context.Background(), &zynaxv1.StoreVectorRequest{
				WorkflowId: wfID, Embedding: []float32{1.0, 0.0, 0.0}, Text: "isolation test vector",
			})
			return err
		})

		sc.Step(
			`^3 vectors are stored in workflow "([^"]*)" with cosine similarities \[([0-9., ]+)\] to the query$`,
			func(wfID, _ string) error {
				// Three 3D unit embeddings — query is [1,0,0].
				// Cosine distances to [1,0,0]: ~0.01, ~0.29, ~0.71 (ascending = most similar first).
				for i, emb := range [][]float32{
					{0.990, 0.141, 0.000},
					{0.707, 0.707, 0.000},
					{0.371, 0.928, 0.000},
				} {
					if _, err := tc.handler.StoreVector(context.Background(), &zynaxv1.StoreVectorRequest{
						WorkflowId: wfID, Embedding: emb, Text: fmt.Sprintf("vec-%d", i),
					}); err != nil {
						return err
					}
				}
				return nil
			})

		sc.Step(`^workflow "([^"]*)" has (\d+) KV entries and (\d+) vectors$`,
			func(wfID string, kvCount, vecCount int) error {
				for i := 0; i < kvCount; i++ {
					if _, err := tc.handler.Set(context.Background(), &zynaxv1.SetRequest{
						WorkflowId: wfID,
						Key:        fmt.Sprintf("ns-key-%d", i),
						Value:      []byte(fmt.Sprintf("val-%d", i)),
					}); err != nil {
						return err
					}
				}
				// Use 3D vectors consistent with the 3D query [1,0,0] used in SearchSimilar steps.
				for i := 0; i < vecCount; i++ {
					if _, err := tc.handler.StoreVector(context.Background(), &zynaxv1.StoreVectorRequest{
						WorkflowId: wfID,
						Embedding:  []float32{float32(i) + 0.1, float32(i) + 0.2, 0.0},
						Text:       fmt.Sprintf("ns-vec-%d", i),
					}); err != nil {
						return err
					}
				}
				return nil
			})

		sc.Step(`^entry key="([^"]*)" value="([^"]*)" is set for workflow "([^"]*)"$`,
			func(key, value, wfID string) error {
				_, tc.lastGRPCErr = tc.handler.Set(context.Background(), &zynaxv1.SetRequest{
					WorkflowId: wfID, Key: key, Value: []byte(value),
				})
				return nil
			})

		sc.Step(`^2 seconds pass$`, func() error {
			time.Sleep(2 * time.Second)
			return nil
		})

		sc.Step(`^GetEntry is called for key="([^"]*)" in workflow "([^"]*)"$`,
			func(key, wfID string) error {
				tc.lastGetResp, tc.lastGRPCErr = tc.handler.Get(context.Background(),
					&zynaxv1.GetRequest{WorkflowId: wfID, Key: key})
				return nil
			})

		sc.Step(`^SearchSimilar is called in workflow "([^"]*)" with top_k=(\d+)$`,
			func(wfID string, topK int) error {
				resp, err := tc.handler.QueryVector(context.Background(), &zynaxv1.QueryVectorRequest{
					WorkflowId: wfID,
					Embedding:  []float32{1.0, 0.0, 0.0},
					TopK:       int32(topK),
				})
				tc.lastGRPCErr = err
				if resp != nil {
					tc.lastQueryRes = resp.GetResults()
				}
				return nil
			})

		sc.Step(`^the namespace "([^"]*)" is deleted$`, func(wfID string) error {
			_, tc.lastGRPCErr = tc.handler.DeleteNamespace(context.Background(),
				&zynaxv1.DeleteNamespaceRequest{WorkflowId: wfID})
			return nil
		})

		sc.Step(`^GetEntry for key="([^"]*)" in workflow "([^"]*)" returns "([^"]*)"$`,
			func(key, wfID, expected string) error {
				resp, err := tc.handler.Get(context.Background(),
					&zynaxv1.GetRequest{WorkflowId: wfID, Key: key})
				if err != nil {
					return fmt.Errorf("Get(%q/%q): %w", wfID, key, err)
				}
				if got := string(resp.GetValue()); got != expected {
					return fmt.Errorf("expected %q, got %q", expected, got)
				}
				return nil
			})

		sc.Step(`^GetEntry for key="([^"]*)" in workflow "([^"]*)" returns NOT_FOUND$`,
			func(key, wfID string) error {
				_, err := tc.handler.Get(context.Background(),
					&zynaxv1.GetRequest{WorkflowId: wfID, Key: key})
				if err == nil {
					return fmt.Errorf("expected NOT_FOUND for %q/%q, got nil", wfID, key)
				}
				if s, ok := status.FromError(err); !ok || s.Code() != codes.NotFound {
					return fmt.Errorf("expected NOT_FOUND, got %v", err)
				}
				return nil
			})

		sc.Step(`^the gRPC status is NOT_FOUND$`, func() error {
			if tc.lastGRPCErr == nil {
				return fmt.Errorf("expected NOT_FOUND, got nil")
			}
			if s, ok := status.FromError(tc.lastGRPCErr); !ok || s.Code() != codes.NotFound {
				return fmt.Errorf("expected NOT_FOUND, got %v", tc.lastGRPCErr)
			}
			return nil
		})

		sc.Step(`^(\d+) results are returned$`, func(n int) error {
			if len(tc.lastQueryRes) != n {
				return fmt.Errorf("expected %d results, got %d", n, len(tc.lastQueryRes))
			}
			return nil
		})

		// "ordered by score descending" in the feature means "best results first".
		// pgvector returns cosine distance (lower = more similar), so results are
		// ordered by distance ascending. This assertion verifies that invariant.
		sc.Step(`^(\d+) results are returned ordered by score descending$`, func(n int) error {
			if len(tc.lastQueryRes) != n {
				return fmt.Errorf("expected %d results, got %d", n, len(tc.lastQueryRes))
			}
			for i := 1; i < len(tc.lastQueryRes); i++ {
				prev := tc.lastQueryRes[i-1].GetSimilarityScore()
				curr := tc.lastQueryRes[i].GetSimilarityScore()
				if curr < prev {
					return fmt.Errorf("results not ordered by distance at index %d: %f < %f (distances should be non-decreasing)",
						i, curr, prev)
				}
			}
			return nil
		})

		sc.Step(`^SearchSimilar in workflow "([^"]*)" returns 0 results$`, func(wfID string) error {
			resp, err := tc.handler.QueryVector(context.Background(), &zynaxv1.QueryVectorRequest{
				WorkflowId: wfID,
				Embedding:  []float32{1.0, 0.0, 0.0},
				TopK:       10,
			})
			if err != nil {
				return fmt.Errorf("QueryVector after DeleteNamespace: %w", err)
			}
			if n := len(resp.GetResults()); n != 0 {
				return fmt.Errorf("expected 0 results, got %d", n)
			}
			return nil
		})
	}
}

func TestFeatures(t *testing.T) {
	redisClient := startRedis(t)
	vecStore := startPostgres(t)
	kvStore := infrastructure.NewRedisKV(redisClient)
	h := api.NewHandler(kvStore, vecStore)

	suite := godog.TestSuite{
		Name:                "memory_service_bdd",
		ScenarioInitializer: initScenario(h),
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"features"},
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("BDD scenarios failed")
	}
}
