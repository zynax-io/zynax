// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"context"
	"testing"

	"github.com/zynax-io/zynax/services/api-gateway/internal/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func firstMeta(md metadata.MD, key string) string {
	if v := md.Get(key); len(v) > 0 {
		return v[0]
	}
	return ""
}

func TestWithCorrelationMetadata(t *testing.T) {
	tests := []struct {
		name      string
		reqID     string
		namespace string
		wantID    string
		wantNS    string
	}{
		{"both set", "req-1", "team-a", "req-1", "team-a"},
		{"only request id", "req-1", "", "req-1", ""},
		{"only namespace", "", "team-a", "", "team-a"},
		{"neither", "", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.reqID != "" {
				ctx = domain.WithRequestID(ctx, tt.reqID)
			}
			if tt.namespace != "" {
				ctx = domain.WithNamespace(ctx, tt.namespace)
			}
			out := withCorrelationMetadata(ctx)
			md, _ := metadata.FromOutgoingContext(out)
			if got := firstMeta(md, requestIDMetaKey); got != tt.wantID {
				t.Errorf("%s = %q; want %q", requestIDMetaKey, got, tt.wantID)
			}
			if got := firstMeta(md, namespaceMetaKey); got != tt.wantNS {
				t.Errorf("%s = %q; want %q", namespaceMetaKey, got, tt.wantNS)
			}
		})
	}
}

// TestCorrelationInterceptorsAttachMetadata verifies both the unary and stream
// interceptors propagate the correlation context onto the outgoing metadata seen
// by the next handler in the chain.
func TestCorrelationInterceptorsAttachMetadata(t *testing.T) {
	ctx := domain.WithNamespace(domain.WithRequestID(context.Background(), "req-9"), "ns-9")

	var unaryMD metadata.MD
	err := requestIDUnaryInterceptor(ctx, "/svc/Method", nil, nil, nil,
		func(c context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
			unaryMD, _ = metadata.FromOutgoingContext(c)
			return nil
		})
	if err != nil {
		t.Fatalf("unary interceptor returned error: %v", err)
	}
	if got := firstMeta(unaryMD, requestIDMetaKey); got != "req-9" {
		t.Errorf("unary %s = %q; want %q", requestIDMetaKey, got, "req-9")
	}
	if got := firstMeta(unaryMD, namespaceMetaKey); got != "ns-9" {
		t.Errorf("unary %s = %q; want %q", namespaceMetaKey, got, "ns-9")
	}

	var streamMD metadata.MD
	_, err = requestIDStreamInterceptor(ctx, &grpc.StreamDesc{}, nil, "/svc/Method",
		func(c context.Context, _ *grpc.StreamDesc, _ *grpc.ClientConn, _ string, _ ...grpc.CallOption) (grpc.ClientStream, error) {
			streamMD, _ = metadata.FromOutgoingContext(c)
			return nil, nil
		})
	if err != nil {
		t.Fatalf("stream interceptor returned error: %v", err)
	}
	if got := firstMeta(streamMD, requestIDMetaKey); got != "req-9" {
		t.Errorf("stream %s = %q; want %q", requestIDMetaKey, got, "req-9")
	}
	if got := firstMeta(streamMD, namespaceMetaKey); got != "ns-9" {
		t.Errorf("stream %s = %q; want %q", namespaceMetaKey, got, "ns-9")
	}
}
