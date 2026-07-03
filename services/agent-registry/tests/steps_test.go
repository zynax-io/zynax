// SPDX-License-Identifier: Apache-2.0

// BDD suite for the CRD-era agent-registry surface (ADR-039, #1584).
// Tests wire the real retired Handler over a bufconn in-process gRPC server
// and pin the deprecation contract: every push-era RPC answers UNIMPLEMENTED
// with a migration pointer until its M9 hard removal.
package tests_test

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/cucumber/godog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/agent-registry/internal/api"
)

type testEnv struct {
	lis     *bufconn.Listener
	srv     *grpc.Server
	conn    *grpc.ClientConn
	client  zynaxv1.AgentRegistryServiceClient
	lastErr error
}

func (e *testEnv) setup() {
	e.lis = bufconn.Listen(1 << 20)
	e.srv = grpc.NewServer()
	zynaxv1.RegisterAgentRegistryServiceServer(e.srv, api.NewHandler())
	go func() { _ = e.srv.Serve(e.lis) }()
	conn, _ := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return e.lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	e.conn = conn
	e.client = zynaxv1.NewAgentRegistryServiceClient(conn)
}

func (e *testEnv) teardown() {
	if e.conn != nil {
		_ = e.conn.Close()
	}
	if e.srv != nil {
		e.srv.GracefulStop()
	}
}

func (e *testEnv) call(rpc string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var err error
	//nolint:staticcheck // SA1019: this suite intentionally exercises the deprecated push-era RPCs to pin their ADR-039 retirement contract (removed with them in M9).
	switch rpc {
	case "RegisterAgent":
		_, err = e.client.RegisterAgent(ctx, &zynaxv1.RegisterAgentRequest{
			Agent: &zynaxv1.AgentDef{AgentId: "a", Endpoint: "x:1",
				Capabilities: []*zynaxv1.CapabilityDef{{Name: "echo"}}},
		})
	case "DeregisterAgent":
		_, err = e.client.DeregisterAgent(ctx, &zynaxv1.DeregisterAgentRequest{AgentId: "a"})
	case "GetAgent":
		_, err = e.client.GetAgent(ctx, &zynaxv1.GetAgentRequest{AgentId: "a"})
	case "ListAgents":
		_, err = e.client.ListAgents(ctx, &zynaxv1.ListAgentsRequest{})
	case "FindByCapability":
		_, err = e.client.FindByCapability(ctx, &zynaxv1.FindByCapabilityRequest{CapabilityName: "echo"})
	default:
		return fmt.Errorf("unknown rpc %q", rpc)
	}
	e.lastErr = err
	return nil
}

func TestFeatures(t *testing.T) {
	env := &testEnv{}

	suite := godog.TestSuite{
		ScenarioInitializer: func(sc *godog.ScenarioContext) {
			sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
				env.setup()
				return ctx, nil
			})
			sc.After(func(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
				env.teardown()
				return ctx, nil
			})

			sc.Step(`^the agent registry is running and healthy$`, func(ctx context.Context) (context.Context, error) {
				if env.client == nil {
					return ctx, fmt.Errorf("bufconn client not ready")
				}
				return ctx, nil
			})

			sc.Step(`^the (\S+) RPC is called$`, func(ctx context.Context, rpc string) (context.Context, error) {
				return ctx, env.call(rpc)
			})

			sc.Step(`^the call fails with code UNIMPLEMENTED$`, func(ctx context.Context) (context.Context, error) {
				if env.lastErr == nil {
					return ctx, fmt.Errorf("expected UNIMPLEMENTED, got success")
				}
				if got := status.Code(env.lastErr); got != codes.Unimplemented {
					return ctx, fmt.Errorf("code = %v, want Unimplemented", got)
				}
				return ctx, nil
			})

			sc.Step(`^the error message mentions "([^"]*)"$`, func(ctx context.Context, needle string) (context.Context, error) {
				if env.lastErr == nil {
					return ctx, fmt.Errorf("expected an error mentioning %q", needle)
				}
				if msg := status.Convert(env.lastErr).Message(); !strings.Contains(msg, needle) {
					return ctx, fmt.Errorf("message %q does not mention %q", msg, needle)
				}
				return ctx, nil
			})
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"features/agent_registry.feature"},
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}
