package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// OAuthCredentials holds OAuth 2.0 tokens for a provider.
type OAuthCredentials struct {
	Provider     string `json:"provider"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	AccountID    string `json:"account_id,omitempty"`  // OpenAI: chatgpt_account_id from JWT
	ProjectID    string `json:"project_id,omitempty"`  // Google: Cloud Code Assist project ID
	ExpiresAt    int64  `json:"expires_at"`            // unix milliseconds
}

// IsExpired returns true if the access token is expired (with 5-minute buffer).
func (c *OAuthCredentials) IsExpired() bool {
	if c == nil {
		return true
	}
	return time.Now().UnixMilli() >= c.ExpiresAt
}

// NeedsRefresh returns true if the token is expired or within 5 minutes of expiry.
func (c *OAuthCredentials) NeedsRefresh() bool {
	if c == nil {
		return true
	}
	buffer := int64(5 * 60 * 1000) // 5 minutes in ms
	return time.Now().UnixMilli() >= (c.ExpiresAt - buffer)
}

// ExpiresIn returns how long until the token expires.
func (c *OAuthCredentials) ExpiresIn() time.Duration {
	if c == nil {
		return 0
	}
	ms := c.ExpiresAt - time.Now().UnixMilli()
	if ms < 0 {
		return 0
	}
	return time.Duration(ms) * time.Millisecond
}

// oauthPath returns the path to the OAuth credentials file.
func oauthPath() string {
	return filepath.Join(".agi", "oauth.json")
}

// LoadAllOAuth loads all OAuth credentials from .agi/oauth.json.
// Returns nil (no error) if the file doesn't exist.
// Supports both legacy (single cred) and new (multi-provider map) format.
func LoadAllOAuth() (map[string]*OAuthCredentials, error) {
	data, err := os.ReadFile(oauthPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	// Try new format first: {"anthropic": {...}, "openai": {...}}
	var store map[string]*OAuthCredentials
	if err := json.Unmarshal(data, &store); err == nil {
		// Verify it's actually a map and not a flat object parsed as map.
		// A flat OAuthCredentials has "access_token" at top level, a store does not.
		if _, isFlat := store["access_token"]; !isFlat && len(store) > 0 {
			return store, nil
		}
	}

	// Fall back to legacy format: single OAuthCredentials object
	var creds OAuthCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}
	if creds.AccessToken == "" {
		return nil, nil
	}

	provider := creds.Provider
	if provider == "" {
		provider = "anthropic"
	}
	return map[string]*OAuthCredentials{provider: &creds}, nil
}

// SaveAllOAuth saves all OAuth credentials to .agi/oauth.json.
func SaveAllOAuth(store map[string]*OAuthCredentials) error {
	if err := os.MkdirAll(".agi", 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(oauthPath(), data, 0600)
}

// SaveOAuth saves a single provider's OAuth credentials.
// Merges into the existing store so other providers are preserved.
func SaveOAuth(creds *OAuthCredentials) error {
	store, err := LoadAllOAuth()
	if err != nil {
		store = make(map[string]*OAuthCredentials)
	}
	if store == nil {
		store = make(map[string]*OAuthCredentials)
	}
	provider := creds.Provider
	if provider == "" {
		provider = "anthropic"
	}
	store[provider] = creds
	return SaveAllOAuth(store)
}
