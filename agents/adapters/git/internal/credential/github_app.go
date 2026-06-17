// SPDX-License-Identifier: Apache-2.0

package credential

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	"github.com/google/go-github/v67/github"
)

// appJWTTTL is the lifetime of the App JWT used to authenticate to the
// installation-token endpoint. GitHub rejects App JWTs with an expiry more than
// 10 minutes out; a conservative value also absorbs clock skew.
const appJWTTTL = 9 * time.Minute

// GitHubAppMinter mints installation tokens for a GitHub App. It signs a
// short-lived RS256 App JWT with the App private key (held only in memory) and
// exchanges it for an installation token via the GitHub API. The private key is
// never logged, returned, or embedded in any error — only minting errors with
// non-secret context are surfaced.
type GitHubAppMinter struct {
	appID          int64
	installationID int64
	key            *rsa.PrivateKey
	now            func() time.Time

	// newClient builds an App-JWT-authenticated client for one mint call. It is a
	// field so tests can point it at an httptest server without real network.
	newClient func(jwt string) *github.Client
}

// AppCredentials are the GitHub App identity inputs resolved at startup. PrivateKeyPEM
// is the PKCS#1 or PKCS#8 RSA private key bytes; it is parsed once into memory and
// never retained as a string field.
type AppCredentials struct {
	AppID          int64
	InstallationID int64
	PrivateKeyPEM  []byte
	// BaseURL overrides the GitHub API endpoint (tests point this at httptest);
	// empty uses the public api.github.com.
	BaseURL string
	// Now is the clock used for JWT timestamps; nil defaults to time.Now.
	Now func() time.Time
}

// NewGitHubAppMinter parses the App private key and builds a Minter. It returns an
// error for a missing/invalid key or non-positive identifiers; the error never
// contains key material.
func NewGitHubAppMinter(c AppCredentials) (*GitHubAppMinter, error) {
	if c.AppID <= 0 {
		return nil, errors.New("credential: app_id must be positive")
	}
	if c.InstallationID <= 0 {
		return nil, errors.New("credential: installation_id must be positive")
	}
	key, err := parseRSAPrivateKey(c.PrivateKeyPEM)
	if err != nil {
		return nil, err
	}
	now := c.Now
	if now == nil {
		now = time.Now
	}
	baseURL := c.BaseURL
	return &GitHubAppMinter{
		appID:          c.AppID,
		installationID: c.InstallationID,
		key:            key,
		now:            now,
		newClient: func(jwt string) *github.Client {
			client := github.NewClient(nil).WithAuthToken(jwt)
			if baseURL != "" {
				if parsed, perr := client.BaseURL.Parse(baseURL + "/"); perr == nil {
					client.BaseURL = parsed
				}
			}
			return client
		},
	}, nil
}

// Mint signs an App JWT and exchanges it for a fresh installation token. The
// returned expiry is GitHub's, so AppSource can schedule the next refresh. Neither
// the JWT nor the private key is logged or returned in an error.
func (m *GitHubAppMinter) Mint(ctx context.Context) (string, time.Time, error) {
	jwt, err := m.signAppJWT()
	if err != nil {
		return "", time.Time{}, err
	}
	client := m.newClient(jwt)
	tok, _, err := client.Apps.CreateInstallationToken(ctx, m.installationID, nil)
	if err != nil {
		// go-github error strings do not embed the JWT or key, but keep context minimal.
		return "", time.Time{}, fmt.Errorf("credential: mint installation token: %w", err)
	}
	expires := tok.GetExpiresAt().Time
	if expires.IsZero() {
		// Defensive: GitHub always sets expires_at (~1 h). If absent, assume a
		// short TTL so the next request forces a refresh rather than trusting forever.
		expires = m.now().Add(refreshSkew)
	}
	return tok.GetToken(), expires, nil
}

// signAppJWT builds and RS256-signs the GitHub App JWT (iss=app-id, ~9 min TTL).
// The signature is computed with stdlib crypto only — no JWT dependency.
func (m *GitHubAppMinter) signAppJWT() (string, error) {
	now := m.now()
	header := map[string]string{"alg": "RS256", "typ": "JWT"}
	claims := map[string]any{
		// iat backdated 30s to tolerate clock drift between the adapter and GitHub.
		"iat": now.Add(-30 * time.Second).Unix(),
		"exp": now.Add(appJWTTTL).Unix(),
		"iss": m.appID,
	}
	hJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("credential: marshal jwt header: %w", err)
	}
	cJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("credential: marshal jwt claims: %w", err)
	}
	signingInput := b64url(hJSON) + "." + b64url(cJSON)
	digest := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, m.key, crypto.SHA256, digest[:])
	if err != nil {
		return "", fmt.Errorf("credential: sign jwt: %w", err)
	}
	return signingInput + "." + b64url(sig), nil
}

func b64url(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

// parseRSAPrivateKey decodes a PEM-encoded RSA private key in PKCS#1 or PKCS#8
// form. Errors carry no key material.
func parseRSAPrivateKey(pemBytes []byte) (*rsa.PrivateKey, error) {
	if len(pemBytes) == 0 {
		return nil, errors.New("credential: private key is empty")
	}
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("credential: private key is not valid PEM")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, errors.New("credential: private key is not a valid PKCS#1 or PKCS#8 RSA key")
	}
	rsaKey, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("credential: private key is not RSA")
	}
	return rsaKey, nil
}
