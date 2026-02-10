package llm

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"ClosedWheeler/pkg/config"
)

// OpenAI Codex CLI OAuth constants.
const (
	OpenAIOAuthClientID     = "app_EMoamEEZ73f0CkXaXp7hrann"
	OpenAIOAuthAuthorizeURL = "https://auth.openai.com/oauth/authorize"
	OpenAIOAuthTokenURL     = "https://auth.openai.com/oauth/token"
	OpenAIOAuthRedirectURI  = "http://localhost:1455/auth/callback"
	OpenAIOAuthScopes       = "openid profile email offline_access"
	OpenAIOAuthCallbackPort = 1455
)

// BuildOpenAIAuthURL constructs the OpenAI OAuth authorization URL using PKCE.
// Matches Codex CLI parameters exactly to trigger the ChatGPT subscription login flow.
func BuildOpenAIAuthURL(challenge, state string) string {
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", OpenAIOAuthClientID)
	q.Set("redirect_uri", OpenAIOAuthRedirectURI)
	q.Set("scope", OpenAIOAuthScopes)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	q.Set("state", state)
	q.Set("id_token_add_organizations", "true")
	q.Set("codex_cli_simplified_flow", "true")
	q.Set("originator", "codex_cli_rs")
	return OpenAIOAuthAuthorizeURL + "?" + q.Encode()
}

// OpenAICallbackResult is the result from the localhost callback server.
type OpenAICallbackResult struct {
	Code  string
	State string
	Err   error
}

// StartOpenAICallbackServer starts a temporary HTTP server on localhost:1455
// to capture the OAuth callback. It returns a channel that receives the result.
// The server auto-shuts down after receiving the callback or when ctx is cancelled.
func StartOpenAICallbackServer(ctx context.Context) (<-chan OpenAICallbackResult, error) {
	resultCh := make(chan OpenAICallbackResult, 1)
	doneCh := make(chan struct{})
	var once sync.Once

	mux := http.NewServeMux()
	mux.HandleFunc("/auth/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")
		errParam := r.URL.Query().Get("error")

		if errParam != "" {
			errDesc := r.URL.Query().Get("error_description")
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, "<html><body><h2>Login failed</h2><p>%s: %s</p><p>You can close this tab.</p></body></html>", errParam, errDesc)
			once.Do(func() {
				resultCh <- OpenAICallbackResult{Err: fmt.Errorf("%s: %s", errParam, errDesc)}
				close(doneCh)
			})
			return
		}

		if code == "" {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, "<html><body><h2>Missing code</h2><p>No authorization code received.</p></body></html>")
			once.Do(func() {
				resultCh <- OpenAICallbackResult{Err: fmt.Errorf("no authorization code in callback")}
				close(doneCh)
			})
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<html><body><h2>Login successful!</h2><p>You can close this tab and return to the terminal.</p></body></html>")
		once.Do(func() {
			resultCh <- OpenAICallbackResult{Code: code, State: state}
			close(doneCh)
		})
	})

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", OpenAIOAuthCallbackPort))
	if err != nil {
		return nil, fmt.Errorf("failed to start callback server on port %d: %w", OpenAIOAuthCallbackPort, err)
	}

	server := &http.Server{Handler: mux}

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			resultCh <- OpenAICallbackResult{Err: fmt.Errorf("callback server error: %w", err)}
		}
	}()

	go func() {
		select {
		case <-ctx.Done():
		case <-doneCh:
			time.Sleep(500 * time.Millisecond)
		}
		server.Close()
	}()

	return resultCh, nil
}

// ExchangeOpenAICode exchanges an authorization code for OpenAI OAuth tokens.
// The access_token returned IS the API key — no secondary exchange needed.
// The accountId is extracted from the access_token JWT claims.
func ExchangeOpenAICode(code, verifier string) (*config.OAuthCredentials, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", OpenAIOAuthClientID)
	form.Set("code", code)
	form.Set("code_verifier", verifier)
	form.Set("redirect_uri", OpenAIOAuthRedirectURI)

	resp, err := oauthHTTPClient.Post(OpenAIOAuthTokenURL, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("token exchange request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed (status %d): %s", resp.StatusCode, truncateError(respBody))
	}

	var tokenResp OAuthTokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	// Extract chatgpt_account_id from the access_token JWT
	accountID := extractClaimFromJWT(tokenResp.AccessToken, "chatgpt_account_id")

	expiresAt := time.Now().UnixMilli() + tokenResp.ExpiresIn*1000
	return &config.OAuthCredentials{
		Provider:     "openai",
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		AccountID:    accountID,
		ExpiresAt:    expiresAt,
	}, nil
}

// RefreshOpenAIToken refreshes an expired OpenAI OAuth token.
func RefreshOpenAIToken(refreshToken string) (*config.OAuthCredentials, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", OpenAIOAuthClientID)

	resp, err := oauthHTTPClient.Post(OpenAIOAuthTokenURL, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("token refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read refresh response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed (status %d): %s", resp.StatusCode, truncateError(respBody))
	}

	var tokenResp OAuthTokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse refresh response: %w", err)
	}

	accountID := extractClaimFromJWT(tokenResp.AccessToken, "chatgpt_account_id")

	expiresAt := time.Now().UnixMilli() + tokenResp.ExpiresIn*1000
	return &config.OAuthCredentials{
		Provider:     "openai",
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		AccountID:    accountID,
		ExpiresAt:    expiresAt,
	}, nil
}

// extractClaimFromJWT extracts a named claim from the "https://api.openai.com/auth"
// namespace in a JWT token.
func extractClaimFromJWT(token, claimName string) string {
	parts := strings.SplitN(token, ".", 3)
	if len(parts) < 2 {
		return ""
	}
	payload := parts[1]
	switch len(payload) % 4 {
	case 2:
		payload += "=="
	case 3:
		payload += "="
	}
	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return ""
	}
	var claims map[string]interface{}
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return ""
	}
	if auth, ok := claims["https://api.openai.com/auth"].(map[string]interface{}); ok {
		if val, ok := auth[claimName].(string); ok {
			return val
		}
	}
	return ""
}
