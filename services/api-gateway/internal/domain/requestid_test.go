// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"context"
	"testing"
)

func TestRequestIDFromContext_Present(t *testing.T) {
	ctx := WithRequestID(context.Background(), "trace-123")
	if got := RequestIDFromContext(ctx); got != "trace-123" {
		t.Errorf("RequestIDFromContext = %q; want %q", got, "trace-123")
	}
}

func TestRequestIDFromContext_Absent(t *testing.T) {
	if got := RequestIDFromContext(context.Background()); got != "" {
		t.Errorf("RequestIDFromContext on empty ctx = %q; want empty string", got)
	}
}
