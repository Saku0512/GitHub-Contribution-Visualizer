package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/saku0512/GitHub-Contribution-Visualizer/backend/internal/analysis"
)

type stubAnalyzer struct {
	result analysis.Result
	err    error
}

func (s stubAnalyzer) AnalyzeUser(context.Context, string) (analysis.Result, error) {
	return s.result, s.err
}

func TestHandleAnalyzeReturnsServiceResult(t *testing.T) {
	server := &Server{
		mux:                   http.NewServeMux(),
		analyzer:              stubAnalyzer{result: analysis.Result{Username: "octocat", PersonaTitle: "Momentum Builder"}},
		gitHubTokenConfigured: true,
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
				gitHubTokenConfigured: tt.err != analysis.ErrGitHubTokenMissing,
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
		gitHubTokenConfigured: true,
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
}
