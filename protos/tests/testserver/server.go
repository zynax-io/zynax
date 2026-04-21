// SPDX-License-Identifier: Apache-2.0
// Package testserver provides a shared in-memory gRPC server helper for BDD contract tests.
package testserver

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1 << 20

// NewBufconnServer starts a gRPC server on an in-memory listener.
// Returns the server (for registering services) and a dial function.
// Call t.Cleanup to stop the server.
func NewBufconnServer(t *testing.T) (*grpc.Server, func(context.Context, string) (net.Conn, error)) {
	t.Helper()
	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer()
	t.Cleanup(func() {
		srv.GracefulStop()
		lis.Close()
	})
	go func() { _ = srv.Serve(lis) }()
	dialer := func(ctx context.Context, _ string) (net.Conn, error) {
		return lis.DialContext(ctx)
	}
	return srv, dialer
}
