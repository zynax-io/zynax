// SPDX-License-Identifier: Apache-2.0

package infrastructure

import "testing"

func TestTemporalTracingInterceptor(t *testing.T) {
	ic, err := TemporalTracingInterceptor()
	if err != nil {
		t.Fatalf("TemporalTracingInterceptor() error = %v", err)
	}
	if ic == nil {
		t.Fatal("TemporalTracingInterceptor() returned a nil interceptor")
	}
}
