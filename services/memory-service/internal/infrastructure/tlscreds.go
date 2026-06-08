// SPDX-License-Identifier: Apache-2.0

// Package infrastructure provides cross-cutting adapters for memory-service,
// including TLS credentials used by the gRPC server.
package infrastructure

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// TLSCreds returns gRPC transport credentials. When certFile, keyFile, and
// caFile are all non-empty, mTLS is configured with hot-reload via callbacks
// so certificate rotation does not require a service restart. When any path
// is empty, insecure credentials are returned (dev / test environments).
func TLSCreds(certFile, keyFile, caFile string) (credentials.TransportCredentials, error) {
	if certFile == "" || keyFile == "" || caFile == "" {
		return insecure.NewCredentials(), nil
	}
	caPEM, err := os.ReadFile(caFile) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("read CA cert %s: %w", caFile, err)
	}
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("parse CA cert: no valid certificates found")
	}
	load := func() (*tls.Certificate, error) {
		c, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("load cert/key pair: %w", err)
		}
		return &c, nil
	}
	return credentials.NewTLS(&tls.Config{
		GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
			return load()
		},
		GetClientCertificate: func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
			return load()
		},
		ClientCAs:  caPool,
		RootCAs:    caPool,
		ClientAuth: tls.RequireAndVerifyClientCert,
	}), nil
}
