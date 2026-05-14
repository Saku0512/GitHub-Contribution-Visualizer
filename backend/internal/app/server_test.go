package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/saku0512/GitHub-Contribution-Visualizer/backend/internal/analysis"
	"github.com/saku0512/GitHub-Contribution-Visualizer/backend/internal/githubapi"
)

type stubAnalyzer struct {
	result analysis.Result
	err    error
}

func (s stubAnalyzer) AnalyzeUser(context.Context, string) (analysis.Result, error) {
	return s.result, s.err
}

type stubOAuthClient struct {
	authorizeURL string
	authorized   githubapi.AuthenticatedUser
	err          error
}

func (s stubOAuthClient) HasOAuthCredentials() bool {
	return true
}

func (s stubOAuthClient) BuildAuthorizationURL(state string, redirectURI string) (string, error) {
	if s.err != nil {
		return "", s.err
	}

	if s.authorizeURL != "" {
		return s.authorizeURL, nil
	}

	return "https://github.com/login/oauth/authorize?state=" + state + "&redirect_uri=" + url.QueryEscape(redirectURI), nil
}

func (s stubOAuthClient) ExchangeCodeForUser(context.Context, string, string) (githubapi.AuthenticatedUser, error) {
	if s.err != nil {
		return githubapi.AuthenticatedUser{}, s.err
	}

	return s.authorized, nil
}

func TestHandleAnalyzeReturnsServiceResult(t *testing.T) {
	server := &Server{
		mux:                   http.NewServeMux(),
		analyzer:              stubAnalyzer{result: analysis.Result{Username: "octocat", PersonaTitle: "Momentum Builder"}},
		oauth:                 stubOAuthClient{},
		store:                 newSessionStore(),
		gitHubTokenConfigured: true,
		gitHubOAuthConfigured: true,
	}
	server.routes()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/analyze", bytes.NewBufferString(`{"username":"octocat"}`))
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var result analysis.Result
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Username != "octocat" {
		t.Fatalf("expected username octocat, got %q", result.Username)
	}
}

func TestHandleAnalyzeMapsKnownErrors(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{name: "invalid username", err: analysis.ErrInvalidUsername, status: http.StatusBadRequest, want: "invalid GitHub username"},
		{name: "not found", err: analysis.ErrUserNotFound, status: http.StatusNotFound, want: "GitHub user not found"},
		{name: "token missing", err: analysis.ErrGitHubTokenMissing, status: http.StatusServiceUnavailable, want: "GITHUB_TOKEN is not configured"},
		{name: "upstream", err: fmt.Errorf("%w: GitHub API returned 401 Unauthorized; check GITHUB_TOKEN", analysis.ErrUpstreamUnavailable), status: http.StatusBadGateway, want: "GitHub API request failed: GitHub API returned 401 Unauthorized; check GITHUB_TOKEN"},
		{name: "unknown", err: errors.New("boom"), status: http.StatusBadGateway, want: "failed to fetch GitHub activity"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &Server{
				mux:                   http.NewServeMux(),
				analyzer:              stubAnalyzer{err: tt.err},
				oauth:                 stubOAuthClient{},
				store:                 newSessionStore(),
				gitHubTokenConfigured: tt.err != analysis.ErrGitHubTokenMissing,
				gitHubOAuthConfigured: true,
			}
			server.routes()

			req := httptest.NewRequest(http.MethodPost, "/api/v1/analyze", bytes.NewBufferString(`{"username":"octocat"}`))
			rec := httptest.NewRecorder()

			server.mux.ServeHTTP(rec, req)

			if rec.Code != tt.status {
				t.Fatalf("expected %d, got %d", tt.status, rec.Code)
			}

			var payload map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if payload["error"] != tt.want {
				t.Fatalf("expected error %q, got %q", tt.want, payload["error"])
			}
		})
	}
}

func TestHandleHealthIncludesGitHubConfigState(t *testing.T) {
	server := &Server{
		mux:                   http.NewServeMux(),
		analyzer:              stubAnalyzer{},
		oauth:                 stubOAuthClient{},
		store:                 newSessionStore(),
		gitHubTokenConfigured: true,
		gitHubOAuthConfigured: true,
	}
	server.routes()

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode health response: %v", err)
	}

	if payload["githubTokenConfigured"] != true {
		t.Fatalf("expected githubTokenConfigured=true, got %#v", payload["githubTokenConfigured"])
	}

	if payload["githubOAuthConfigured"] != true {
		t.Fatalf("expected githubOAuthConfigured=true, got %#v", payload["githubOAuthConfigured"])
	}
}

func TestHandleGitHubLoginRedirectsToGitHub(t *testing.T) {
	server := &Server{
		mux:                   http.NewServeMux(),
		analyzer:              stubAnalyzer{},
		oauth:                 stubOAuthClient{},
		store:                 newSessionStore(),
		gitHubTokenConfigured: true,
		gitHubOAuthConfigured: true,
	}
	server.routes()

	req := httptest.NewRequest(http.MethodGet, "https://example.com/api/v1/auth/github/login", nil)
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected 307, got %d", rec.Code)
	}

	location := rec.Header().Get("Location")
	if location == "" {
		t.Fatal("expected redirect location")
	}

	stateCookie := rec.Result().Cookies()
	if len(stateCookie) == 0 || stateCookie[0].Name != "github_oauth_state" {
		t.Fatalf("expected github_oauth_state cookie, got %#v", stateCookie)
	}
}

func TestHandleGitHubCallbackRedirectsBackWithLogin(t *testing.T) {
	server := &Server{
		mux: http.NewServeMux(),
		analyzer: stubAnalyzer{
			result: analysis.Result{Username: "octocat", PersonaTitle: "Momentum Builder"},
		},
		oauth: stubOAuthClient{authorized: githubapi.AuthenticatedUser{
			Login: "octocat",
		}},
		store:                 newSessionStore(),
		gitHubTokenConfigured: true,
		gitHubOAuthConfigured: true,
	}
	server.routes()

	req := httptest.NewRequest(http.MethodGet, "https://example.com/api/v1/auth/github/callback?code=test-code&state=test-state", nil)
	req.AddCookie(&http.Cookie{
		Name:  "github_oauth_state",
		Value: "test-state",
	})
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected 307, got %d", rec.Code)
	}

	if got := rec.Header().Get("Location"); got != "https://example.com/my-page" {
		t.Fatalf("unexpected redirect target: %q", got)
	}

	foundSessionCookie := false
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == "github_session" && cookie.Value != "" {
			foundSessionCookie = true
		}
	}

	if !foundSessionCookie {
		t.Fatal("expected github_session cookie to be set")
	}
}

func TestHandleMeReturnsCurrentUserResult(t *testing.T) {
	store := newSessionStore()
	store.SaveSession("session-123", "octocat")
	store.SaveProfile(analysis.Result{Username: "octocat", PersonaTitle: "Momentum Builder"})

	server := &Server{
		mux:                   http.NewServeMux(),
		analyzer:              stubAnalyzer{},
		oauth:                 stubOAuthClient{},
		store:                 store,
		gitHubTokenConfigured: true,
		gitHubOAuthConfigured: true,
	}
	server.routes()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.AddCookie(&http.Cookie{Name: "github_session", Value: "session-123"})
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestHandleShowcaseReturnsLoggedInUsers(t *testing.T) {
	store := newSessionStore()
	store.SaveSession("session-123", "octocat")
	store.SaveProfile(analysis.Result{Username: "octocat", PersonaTitle: "Momentum Builder"})

	server := &Server{
		mux:                   http.NewServeMux(),
		analyzer:              stubAnalyzer{},
		oauth:                 stubOAuthClient{},
		store:                 store,
		gitHubTokenConfigured: true,
		gitHubOAuthConfigured: true,
	}
	server.routes()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/showcase", nil)
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestHandleLogoutClearsSession(t *testing.T) {
	store := newSessionStore()
	store.SaveSession("session-123", "octocat")

	server := &Server{
		mux:                   http.NewServeMux(),
		analyzer:              stubAnalyzer{},
		oauth:                 stubOAuthClient{},
		store:                 store,
		gitHubTokenConfigured: true,
		gitHubOAuthConfigured: true,
	}
	server.routes()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/logout", nil)
	req.AddCookie(&http.Cookie{Name: "github_session", Value: "session-123"})
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if _, ok := store.GetLogin("session-123"); ok {
		t.Fatal("expected session to be removed")
	}
}
