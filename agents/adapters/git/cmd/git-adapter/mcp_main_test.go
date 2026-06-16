// SPDX-License-Identifier: Apache-2.0

// Whitebox coverage for the `git-adapter mcp` stdio path (#1198, G.2).
package main

import (
	"io"
	"strings"
	"testing"

	"github.com/zynax-io/zynax/agents/adapters/git/internal/config"
)

// TestServeMCP_ClosedTransportReturnsNil covers serveMCP: it builds the tool
// allow-list from the configured capabilities and serves until the transport
// closes. An empty reader closes immediately, so the stdio loop returns nil.
func TestServeMCP_ClosedTransportReturnsNil(t *testing.T) {
	cfg := &config.AdapterConfig{
		Capabilities: []config.GitCapabilityConfig{{Name: "open_pr"}, {Name: "get_diff"}},
	}
	if err := serveMCP(cfg, "token", strings.NewReader(""), io.Discard); err != nil {
		t.Fatalf("serveMCP over a closed transport should return nil, got %v", err)
	}
}
