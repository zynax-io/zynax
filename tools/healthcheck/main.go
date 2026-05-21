// SPDX-License-Identifier: Apache-2.0

// Package main is a minimal health probe for distroless container images.
// Accepts one argument: http://host:port/path or tcp://host:port.
// Exits 0 on success, 1 on any failure. Timeout: 5 s. No retries (Docker handles retries).
package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

const dialTimeout = 5 * time.Second

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: healthcheck <http://host:port/path|tcp://host:port>")
		os.Exit(1)
	}
	if err := probe(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func probe(target string) error {
	switch {
	case strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://"):
		return probeHTTP(target)
	case strings.HasPrefix(target, "tcp://"):
		return probeTCP(strings.TrimPrefix(target, "tcp://"))
	default:
		return fmt.Errorf("unsupported scheme in %q — want http:// or tcp://", target)
	}
}

func probeHTTP(url string) error {
	c := &http.Client{Timeout: dialTimeout}
	resp, err := c.Get(url) //nolint:noctx
	if err != nil {
		return fmt.Errorf("http get %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http get %s: status %d", url, resp.StatusCode)
	}
	return nil
}

func probeTCP(addr string) error {
	conn, err := net.DialTimeout("tcp", addr, dialTimeout)
	if err != nil {
		return fmt.Errorf("tcp dial %s: %w", addr, err)
	}
	conn.Close()
	return nil
}
