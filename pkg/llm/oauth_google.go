package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"ClosedWheeler/pkg/config"
)

// Google Gemini CLI OAuth constants (Cloud Code Assist).
const (
	GoogleOAuthClientID     = "681255809395-oo8ft2oprdrnp9e3aqf6av3hmdib135j.apps.googleusercontent.com"
	GoogleOAuthClientSecret = "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"
	GoogleOAuthAuthorizeURL = "https://accounts.google.com/o/oauth2/v2/auth"
	GoogleOAuthTokenURL     = "https://oauth2.googleapis.com/token"
	GoogleOAuthRedirectURI  = "http://localhost:8085/oauth2callback"
	GoogleOAuthScopes       = "https://www.googleapis.com/auth/cloud-platform https://www.googleapis.com/auth/userinfo.email https://www.googleapis.com/auth/userinfo.profile"
	GoogleOAuthCallbackPort = 8085
	GoogleCodeAssistAPI     = "https://cloudcode-pa.googleapis.com"
)

// BuildGoogleAuthURL constructs the Google OAuth authorization URL using PKCE.
func BuildGoogleAuthURL(challenge, state string) string {
	q := url.Values{}
	q.Set("client_id", GoogleOAuthClientID)
	q.Set("response_type", "code")
	q.Set("redirect_uri", GoogleOAuthRedirectURI)
	q.Set("scope", GoogleOAuthScopes)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	q.Set("state", state)
	q.Set("access_type", "offline")
	q.Set("prompt", "consent")
	return GoogleOAuthAuthorizeURL + "?" + q.Encode()
}

// GoogleCallbackResult is the result from the localhost callback server.
type GoogleCallbackResult struct {
	Code  string
	State string
	Err   error
}

// StartGoogleCallbackServer starts a temporary HTTP server on localhost:8085
// to capture the OAuth callback. Returns a channel that receives the result.
func StartGoogleCallbackServer(ctx context.Context) (<-chan GoogleCallbackResult, error) {
	resultCh := make(chan GoogleCallbackResult, 1)
	doneCh := make(chan struct{})

	mux := http.NewServeMux()
	mux.HandleFunc("/oauth2callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")
		errParam := r.URL.Query().Get("error")

		if errParam != "" {
			errDesc := r.URL.Query().Get("error_description")
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, "<html><body><h2>Login failed</h2><p>%s: %s</p><p>You can close this tab.</p></body></html>", errParam, errDesc)
			resultCh <- GoogleCallbackResult{Err: fmt.Errorf("%s: %s", errParam, errDesc)}
			close(doneCh)
			return
		}

		if code == "" {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, "<html><body><h2>Missing code</h2><p>No authorization code received.</p></body></html>")
			resultCh <- GoogleCallbackResult{Err: fmt.Errorf("no authorization code in callback")}
			close(doneCh)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<html><body><h2>Login successful!</h2><p>You can close this tab and return to the terminal.</p></body></html>")
		resultCh <- GoogleCallbackResult{Code: code, State: state}
		close(doneCh)
	})

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", GoogleOAuthCallbackPort))
	if err != nil {
		return nil, fmt.Errorf("failed to start callback server on port %d: %w", GoogleOAuthCallbackPort, err)
	}

	server := &http.Server{Handler: mux}

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			resultCh <- GoogleCallbackResult{Err: fmt.Errorf("callback server error: %w", err)}
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

// ExchangeGoogleCode exchanges an authorization code for Google OAuth tokens.
// Google requires client_secret in the token exchange (unlike Anthropic/OpenAI).
func ExchangeGoogleCode(code, verifier string) (*config.OAuthCredentials, error) {
	form := url.Values{}
	form.Set("client_id", GoogleOAuthClientID)
	form.Set("client_secret", GoogleOAuthClientSecret)
	form.Set("code", code)
	form.Set("grant_type", "authorization_code")
	form.Set("redirect_uri", GoogleOAuthRedirectURI)
	form.Set("code_verifier", verifier)

	resp, err := oauthHTTPClient.Post(GoogleOAuthTokenURL, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
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

	if tokenResp.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh token received — try again")
	}

	// Discover Cloud Code Assist project
	projectID, err := discoverGoogleProject(tokenResp.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("project discovery failed: %w", err)
	}

	expiresAt := time.Now().UnixMilli() + tokenResp.ExpiresIn*1000
	return &config.OAuthCredentials{
		Provider:     "google",
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ProjectID:    projectID,
		ExpiresAt:    expiresAt,
	}, nil
}

// RefreshGoogleToken refreshes an expired Google OAuth token.
func RefreshGoogleToken(refreshToken string) (*config.OAuthCredentials, error) {
	form := url.Values{}
	form.Set("client_id", GoogleOAuthClientID)
	form.Set("client_secret", GoogleOAuthClientSecret)
	form.Set("refresh_token", refreshToken)
	form.Set("grant_type", "refresh_token")

	resp, err := oauthHTTPClient.Post(GoogleOAuthTokenURL, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
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

	// Google refresh may not return a new refresh_token
	newRefresh := tokenResp.RefreshToken
	if newRefresh == "" {
		newRefresh = refreshToken
	}

	expiresAt := time.Now().UnixMilli() + tokenResp.ExpiresIn*1000
	return &config.OAuthCredentials{
		Provider:     "google",
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: newRefresh,
		ExpiresAt:    expiresAt,
		// ProjectID is preserved from the original creds by the caller
	}, nil
}

// --- Cloud Code Assist project discovery ---

// loadCodeAssistRequest is the request body for loadCodeAssist.
type loadCodeAssistRequest struct {
	CloudAICompanionProject string                   `json:"cloudaicompanionProject,omitempty"`
	Metadata                loadCodeAssistMetadata    `json:"metadata"`
}

type loadCodeAssistMetadata struct {
	IDEType     string `json:"ideType"`
	Platform    string `json:"platform"`
	PluginType  string `json:"pluginType"`
	DuetProject string `json:"duetProject,omitempty"`
}

type loadCodeAssistResponse struct {
	CurrentTier             interface{} `json:"currentTier,omitempty"`
	CloudAICompanionProject string      `json:"cloudaicompanionProject,omitempty"`
	AllowedTiers            []struct {
		ID        string `json:"id"`
		IsDefault bool   `json:"isDefault"`
	} `json:"allowedTiers,omitempty"`
}

type onboardRequest struct {
	TierID   string                 `json:"tierId"`
	Metadata loadCodeAssistMetadata `json:"metadata"`
}

type onboardResponse struct {
	Done     bool   `json:"done"`
	Name     string `json:"name,omitempty"`
	Response *struct {
		CloudAICompanionProject *struct {
			ID string `json:"id"`
		} `json:"cloudaicompanionProject,omitempty"`
	} `json:"response,omitempty"`
}

// discoverGoogleProject discovers or provisions a Cloud Code Assist project.
func discoverGoogleProject(accessToken string) (string, error) {
	headers := map[string]string{
		"Authorization": "Bearer " + accessToken,
		"Content-Type":  "application/json",
		"User-Agent":    "google-api-nodejs-client/9.15.1",
	}

	// Try to load existing project via loadCodeAssist
	reqBody := loadCodeAssistRequest{
		Metadata: loadCodeAssistMetadata{
			IDEType:    "IDE_UNSPECIFIED",
			Platform:   "PLATFORM_UNSPECIFIED",
			PluginType: "GEMINI",
		},
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", GoogleCodeAssistAPI+"/v1internal:loadCodeAssist", strings.NewReader(string(jsonBody)))
	if err != nil {
		return "", fmt.Errorf("failed to create loadCodeAssist request: %w", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := oauthHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("loadCodeAssist request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read loadCodeAssist response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("loadCodeAssist failed (status %d): %s", resp.StatusCode, truncateError(respBody))
	}

	var data loadCodeAssistResponse
	if err := json.Unmarshal(respBody, &data); err != nil {
		return "", fmt.Errorf("failed to parse loadCodeAssist response: %w", err)
	}

	// User already has a project
	if data.CurrentTier != nil && data.CloudAICompanionProject != "" {
		return data.CloudAICompanionProject, nil
	}

	// User needs to be onboarded — get default tier
	tierID := "free-tier"
	if len(data.AllowedTiers) > 0 {
		for _, t := range data.AllowedTiers {
			if t.IsDefault {
				tierID = t.ID
				break
			}
		}
	}

	// Start onboarding
	onReq := onboardRequest{
		TierID: tierID,
		Metadata: loadCodeAssistMetadata{
			IDEType:    "IDE_UNSPECIFIED",
			Platform:   "PLATFORM_UNSPECIFIED",
			PluginType: "GEMINI",
		},
	}

	onBody, _ := json.Marshal(onReq)
	onHTTPReq, err := http.NewRequest("POST", GoogleCodeAssistAPI+"/v1internal:onboardUser", strings.NewReader(string(onBody)))
	if err != nil {
		return "", fmt.Errorf("failed to create onboardUser request: %w", err)
	}
	for k, v := range headers {
		onHTTPReq.Header.Set(k, v)
	}

	onResp, err := oauthHTTPClient.Do(onHTTPReq)
	if err != nil {
		return "", fmt.Errorf("onboardUser request failed: %w", err)
	}
	defer onResp.Body.Close()

	onRespBody, err := io.ReadAll(onResp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read onboardUser response: %w", err)
	}

	if onResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("onboardUser failed (status %d): %s", onResp.StatusCode, truncateError(onRespBody))
	}

	var lro onboardResponse
	if err := json.Unmarshal(onRespBody, &lro); err != nil {
		return "", fmt.Errorf("failed to parse onboardUser response: %w", err)
	}

	// Poll if not done
	if !lro.Done && lro.Name != "" {
		for i := 0; i < 30; i++ { // max ~2.5 minutes
			time.Sleep(5 * time.Second)

			pollReq, _ := http.NewRequest("GET", GoogleCodeAssistAPI+"/v1internal/"+lro.Name, nil)
			for k, v := range headers {
				pollReq.Header.Set(k, v)
			}

			pollResp, err := oauthHTTPClient.Do(pollReq)
			if err != nil {
				continue
			}

			pollBody, _ := io.ReadAll(pollResp.Body)
			pollResp.Body.Close()

			if err := json.Unmarshal(pollBody, &lro); err != nil {
				continue
			}
			if lro.Done {
				break
			}
		}
	}

	if lro.Response != nil && lro.Response.CloudAICompanionProject != nil {
		return lro.Response.CloudAICompanionProject.ID, nil
	}

	return "", fmt.Errorf("could not discover or provision a Google Cloud project — try setting GOOGLE_CLOUD_PROJECT env var")
}
