// SPDX-License-Identifier: Apache-2.0
// BDD contract tests for the gRPC Health Checking Protocol (grpc.health.v1.Health).
package health_service_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cucumber/godog"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/protos/tests/testserver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

// namedKey is the per-service named serving key under test. Any of the Zynax
// service descriptors works; agent-registry is representative.
var namedKey = zynaxv1.AgentRegistryService_ServiceDesc.ServiceName

type healthCtx struct {
	client     grpc_health_v1.HealthClient
	healthSrv  *health.Server
	lastStatus grpc_health_v1.HealthCheckResponse_ServingStatus
	grpcErr    error
}

func (hc *healthCtx) setupServer(t *testing.T) error {
	t.Helper()
	srv, dialer := testserver.NewBufconnServer(t)
	hc.healthSrv = health.NewServer()
	grpc_health_v1.RegisterHealthServer(srv, hc.healthSrv)
	hc.healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	hc.healthSrv.SetServingStatus(namedKey, grpc_health_v1.HealthCheckResponse_SERVING)

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return err //nolint:wrapcheck
	}
	t.Cleanup(func() { _ = conn.Close() }) //nolint:errcheck
	hc.client = grpc_health_v1.NewHealthClient(conn)
	return nil
}

func (hc *healthCtx) check(service string) {
	callCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	resp, err := hc.client.Check(callCtx, &grpc_health_v1.HealthCheckRequest{Service: service})
	hc.grpcErr = err
	if resp != nil {
		hc.lastStatus = resp.GetStatus()
	}
}

func (hc *healthCtx) assertStatus(want grpc_health_v1.HealthCheckResponse_ServingStatus) error {
	if hc.grpcErr != nil {
		return fmt.Errorf("expected status %v but got error: %w", want, hc.grpcErr)
	}
	if hc.lastStatus != want {
		return fmt.Errorf("expected status %v, got %v", want, hc.lastStatus)
	}
	return nil
}

//nolint:funlen
func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: func(sc *godog.ScenarioContext) {
			var hc *healthCtx

			sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
				hc = &healthCtx{}
				return ctx, nil
			})

			sc.Step(`^a gRPC service with the standard Health server is running$`, func(ctx context.Context) (context.Context, error) {
				return ctx, hc.setupServer(t)
			})

			sc.Step(`^a HealthCheckRequest is sent with service ""$`, func(ctx context.Context) (context.Context, error) {
				hc.check("")
				return ctx, nil
			})

			sc.Step(`^a HealthCheckRequest is sent with the service's named key$`, func(ctx context.Context) (context.Context, error) {
				hc.check(namedKey)
				return ctx, nil
			})

			sc.Step(`^a HealthCheckRequest is sent with service "([^"]*)"$`, func(ctx context.Context, service string) (context.Context, error) {
				hc.check(service)
				return ctx, nil
			})

			sc.Step(`^the service receives a graceful shutdown signal$`, func(ctx context.Context) (context.Context, error) {
				// Mirror the cmd/<svc>/main.go drain ordering: NOT_SERVING before stop.
				hc.healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
				hc.healthSrv.SetServingStatus(namedKey, grpc_health_v1.HealthCheckResponse_NOT_SERVING)
				return ctx, nil
			})

			sc.Step(`^the health status is SERVING$`, func(ctx context.Context) (context.Context, error) {
				return ctx, hc.assertStatus(grpc_health_v1.HealthCheckResponse_SERVING)
			})

			sc.Step(`^the health status is NOT_SERVING$`, func(ctx context.Context) (context.Context, error) {
				return ctx, hc.assertStatus(grpc_health_v1.HealthCheckResponse_NOT_SERVING)
			})

			sc.Step(`^the gRPC health status is NOT_FOUND$`, func(ctx context.Context) (context.Context, error) {
				if hc.grpcErr == nil {
					return ctx, fmt.Errorf("expected NOT_FOUND error, got nil")
				}
				st, _ := status.FromError(hc.grpcErr)
				if st.Code() != codes.NotFound {
					return ctx, fmt.Errorf("expected NOT_FOUND, got %v", st.Code())
				}
				return ctx, nil
			})
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../features/health.feature"},
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}
