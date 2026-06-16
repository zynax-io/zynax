// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"github.com/zynax-io/zynax/libs/zynaxobs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// tracingDialOpts returns the standard dial options for an outgoing gRPC client:
// the given transport credentials plus the shared OTEL client stats handler and
// the "<service>.<rpc>" span-naming interceptors (canvas O.3). Centralizing this
// keeps every task-broker → downstream connection traced identically. Telemetry is
// a no-op when no OTLP endpoint is configured.
func tracingDialOpts(creds credentials.TransportCredentials) []grpc.DialOption {
	unary, stream := zynaxobs.TracingClientInterceptors()
	return []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithStatsHandler(zynaxobs.TracingClientHandler()),
		grpc.WithChainUnaryInterceptor(unary),
		grpc.WithChainStreamInterceptor(stream),
	}
}
