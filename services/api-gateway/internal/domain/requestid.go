// SPDX-License-Identifier: Apache-2.0

package domain

import "context"

type reqIDKey struct{}

// RequestIDFromContext returns the request ID stored in ctx, or "" if absent.
func RequestIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(reqIDKey{}).(string)
	return v
}

// WithRequestID returns a new context carrying the given request ID.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, reqIDKey{}, id)
}
