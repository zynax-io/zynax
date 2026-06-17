// SPDX-License-Identifier: Apache-2.0

package credential

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// genKeyPEM returns a fresh PKCS#1 RSA private key in PEM form for tests.
func genKeyPEM(t *testing.T) []byte {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	der := x509.MarshalPKCS1PrivateKey(key)
	return pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: der})
}

// genKeyPEMPKCS8 returns a fresh PKCS#8 RSA private key in PEM form.
func genKeyPEMPKCS8(t *testing.T) []byte {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal pkcs8: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
}

func TestNewGitHubAppMinter_Validation(t *testing.T) {
	valid := genKeyPEM(t)
	tests := []struct {
		name string
		c    AppCredentials
	}{
		{"zero app id", AppCredentials{AppID: 0, InstallationID: 1, PrivateKeyPEM: valid}},
		{"zero installation id", AppCredentials{AppID: 1, InstallationID: 0, PrivateKeyPEM: valid}},
		{"empty key", AppCredentials{AppID: 1, InstallationID: 1, PrivateKeyPEM: nil}},
		{"garbage key", AppCredentials{AppID: 1, InstallationID: 1, PrivateKeyPEM: []byte("not pem")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewGitHubAppMinter(tt.c); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestParseRSAPrivateKey_PKCS8(t *testing.T) {
	if _, err := parseRSAPrivateKey(genKeyPEMPKCS8(t)); err != nil {
		t.Fatalf("PKCS#8 key should parse: %v", err)
	}
}

func TestParseRSAPrivateKey_NonRSAPKCS8Rejected(t *testing.T) {
	// A PEM block that decodes but is not an RSA key.
	block := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte("bogus")})
	if _, err := parseRSAPrivateKey(block); err == nil {
		t.Fatal("expected error for non-RSA PKCS#8 bytes")
	}
}

// fakeInstallationServer serves the installation-token endpoint, asserting that a
// Bearer App JWT is presented, and returns a token with the requested expiry.
func fakeInstallationServer(t *testing.T, token string, exp time.Time) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") || len(auth) < len("Bearer ")+10 {
			t.Errorf("expected a Bearer App JWT, got %q", auth)
		}
		if !strings.Contains(r.URL.Path, "/access_tokens") {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprintf(w, `{"token":%q,"expires_at":%q}`, token, exp.UTC().Format(time.RFC3339))
	}))
}

func TestGitHubAppMinter_MintAgainstFakeServer(t *testing.T) {
	exp := time.Now().Add(time.Hour).Truncate(time.Second)
	srv := fakeInstallationServer(t, "ghs_minted_token_abc", exp)
	defer srv.Close()

	m, err := NewGitHubAppMinter(AppCredentials{
		AppID:          12345,
		InstallationID: 67890,
		PrivateKeyPEM:  genKeyPEM(t),
		BaseURL:        srv.URL,
	})
	if err != nil {
		t.Fatalf("new minter: %v", err)
	}

	tok, gotExp, err := m.Mint(context.Background())
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	if tok != "ghs_minted_token_abc" {
		t.Fatalf("token = %q, want minted value", tok)
	}
	if !gotExp.Equal(exp) {
		t.Fatalf("expiry = %v, want %v", gotExp, exp)
	}
}

func TestGitHubAppMinter_MintServerErrorSurfaced(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprint(w, `{"message":"bad credentials"}`)
	}))
	defer srv.Close()

	m, err := NewGitHubAppMinter(AppCredentials{
		AppID:          1,
		InstallationID: 1,
		PrivateKeyPEM:  genKeyPEM(t),
		BaseURL:        srv.URL,
	})
	if err != nil {
		t.Fatalf("new minter: %v", err)
	}
	if _, _, err := m.Mint(context.Background()); err == nil {
		t.Fatal("expected error from a failing installation-token endpoint")
	}
}

// AC: an App source wired to a minter survives past the original token TTL,
// re-minting against the (fake) endpoint without a restart.
func TestAppSource_WithRealMinter_RefreshAfterTTL(t *testing.T) {
	base := time.Now()
	cur := base
	clock := func() time.Time { return cur }

	// Server hands out a fresh token each call so we can observe the rotation.
	var n int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n++
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprintf(w, `{"token":"ghs_v%d","expires_at":%q}`,
			n, cur.Add(time.Hour).UTC().Format(time.RFC3339))
	}))
	defer srv.Close()

	m, err := NewGitHubAppMinter(AppCredentials{
		AppID: 1, InstallationID: 1, PrivateKeyPEM: genKeyPEM(t), BaseURL: srv.URL, Now: clock,
	})
	if err != nil {
		t.Fatalf("new minter: %v", err)
	}
	s := NewAppSource(m, clock)

	first, err := s.Token(context.Background())
	if err != nil {
		t.Fatalf("first token: %v", err)
	}
	cur = base.Add(2 * time.Hour) // past the original TTL
	second, err := s.Token(context.Background())
	if err != nil {
		t.Fatalf("second token: %v", err)
	}
	if first == second {
		t.Fatalf("expected token rotation after TTL, both were %q", first)
	}
}
