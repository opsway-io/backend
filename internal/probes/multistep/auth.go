package multistep

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/opsway-io/backend/internal/entities"
)

const (
	// tokenExpiryLeeway refreshes tokens a bit before they actually expire.
	tokenExpiryLeeway = 30 * time.Second
	// defaultTokenTTL is used when the token endpoint does not return expires_in.
	defaultTokenTTL = 5 * time.Minute
)

// BasicAuthHeader returns the value of an Authorization header for basic auth.
func BasicAuthHeader(username, password string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))
}

type cachedToken struct {
	token     string
	expiresAt time.Time
}

// TokenSource fetches and caches OAuth2 client-credentials access tokens.
type TokenSource struct {
	client *http.Client
	now    func() time.Time

	mu     sync.Mutex
	tokens map[string]cachedToken
}

func NewTokenSource(client *http.Client) *TokenSource {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	return &TokenSource{
		client: client,
		now:    time.Now,
		tokens: map[string]cachedToken{},
	}
}

// Token returns a valid access token for the auth settings, fetching a new one
// from the token URL when none is cached or the cached one is about to expire.
func (s *TokenSource) Token(ctx context.Context, auth entities.MonitorSettingsAuth) (string, error) {
	if auth.TokenURL == "" {
		return "", fmt.Errorf("oauth2: token url is empty")
	}

	key := auth.TokenURL + "\x00" + auth.ClientID + "\x00" + auth.ClientSecret

	s.mu.Lock()
	if t, ok := s.tokens[key]; ok && s.now().Before(t.expiresAt) {
		s.mu.Unlock()

		return t.token, nil
	}
	s.mu.Unlock()

	token, ttl, err := s.fetch(ctx, auth)
	if err != nil {
		return "", err
	}

	expiresAt := s.now().Add(ttl)
	if ttl > 2*tokenExpiryLeeway {
		expiresAt = expiresAt.Add(-tokenExpiryLeeway)
	}

	s.mu.Lock()
	s.tokens[key] = cachedToken{token: token, expiresAt: expiresAt}
	s.mu.Unlock()

	return token, nil
}

func (s *TokenSource) fetch(ctx context.Context, auth entities.MonitorSettingsAuth) (string, time.Duration, error) {
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", auth.ClientID)
	form.Set("client_secret", auth.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, auth.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", 0, fmt.Errorf("oauth2: failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(url.QueryEscape(auth.ClientID), url.QueryEscape(auth.ClientSecret))

	resp, err := s.client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("oauth2: token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", 0, fmt.Errorf("oauth2: failed to read token response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", 0, fmt.Errorf("oauth2: token endpoint returned status %d", resp.StatusCode)
	}

	var payload struct {
		AccessToken string      `json:"access_token"`
		ExpiresIn   json.Number `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", 0, fmt.Errorf("oauth2: invalid token response: %w", err)
	}

	if payload.AccessToken == "" {
		return "", 0, fmt.Errorf("oauth2: token response has no access_token")
	}

	ttl := defaultTokenTTL
	if secs, err := payload.ExpiresIn.Int64(); err == nil && secs > 0 {
		ttl = time.Duration(secs) * time.Second
	}

	return payload.AccessToken, ttl, nil
}
