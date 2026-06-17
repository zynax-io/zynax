// SPDX-License-Identifier: Apache-2.0

package domain

import "context"

// Correlation context carries the stable, non-trace identifiers that must follow
// a run across every downstream hop (gRPC metadata, Temporal memo, NATS header)
// so a single operator request is traceable end-to-end. These are distinct from
// the W3C trace context (traceparent/tracestate), which zynaxobs propagates
// separately: the request ID survives even when sampling drops a trace, and the
// namespace scopes every downstream call to the same tenant (canvas C.2).
//
// Only correlation identifiers live here — never auth tokens, secrets, or session
// data (canvas C safeguard: "correlation ids only").
type (
	reqIDKey     struct{}
	namespaceKey struct{}
)

// RequestIDFromContext returns the request ID stored in ctx, or "" if absent.
func RequestIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(reqIDKey{}).(string)
	return v
}

// WithRequestID returns a new context carrying the given request ID.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, reqIDKey{}, id)
}

// NamespaceFromContext returns the namespace stored in ctx, or "" if absent.
func NamespaceFromContext(ctx context.Context) string {
	v, _ := ctx.Value(namespaceKey{}).(string)
	return v
}

// WithNamespace returns a new context carrying the given namespace. An empty
// namespace is stored unchanged so callers can rely on the round-trip; the
// downstream interceptors skip emitting the header when the value is "".
func WithNamespace(ctx context.Context, ns string) context.Context {
	return context.WithValue(ctx, namespaceKey{}, ns)
}
