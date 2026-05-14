package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/saku0512/GitHub-Contribution-Visualizer/backend/internal/analysis"
	"github.com/saku0512/GitHub-Contribution-Visualizer/backend/internal/githubapi"
)

type Server struct {
	Addr                  string
	mux                   *http.ServeMux
	analyzer              analyzer
	gitHubTokenConfigured bool
}

type analyzer interface {
	AnalyzeUser(ctx context.Context, username string) (analysis.Result, error)
}

func NewServer() *http.Server {
	githubClient := githubapi.NewClient(
		os.Getenv("GITHUB_TOKEN"),
		envOrDefault("GITHUB_GRAPHQL_URL", ""),
	)

	server := &Server{
		Addr:                  envOrDefault("PORT", "8080"),
		mux:                   http.NewServeMux(),
		analyzer:              analysis.NewService(githubClient, time.Now),
		gitHubTokenConfigured: githubClient.HasToken(),
	}

	server.routes()

	return &http.Server{
		Addr:         ":" + server.Addr,
		Handler:      withCORS(withLogging(server.mux)),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}
}

func (s *Server) routes() {
	s.mux.HandleFunc("/api/health", s.handleHealth)
	s.mux.HandleFunc("/api/v1/analyze", s.handleAnalyze)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":                "ok",
		"githubTokenConfigured": s.gitHubTokenConfigured,
	})
}

func (s *Server) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	var req struct {
		Username string `json:"username"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	if req.Username == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "username is required",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

	result, err := s.analyzer.AnalyzeUser(ctx, req.Username)
	if err != nil {
		log.Printf("analyze request failed for %q: %v", req.Username, err)
		s.writeAnalyzeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) writeAnalyzeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, analysis.ErrInvalidUsername):
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid GitHub username",
		})
	case errors.Is(err, analysis.ErrUserNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "GitHub user not found",
		})
	case errors.Is(err, analysis.ErrGitHubTokenMissing):
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "GITHUB_TOKEN is not configured",
		})
	case errors.Is(err, analysis.ErrUpstreamUnavailable):
		writeJSON(w, http.StatusBadGateway, map[string]string{
			"error": publicUpstreamError(err),
		})
	default:
		writeJSON(w, http.StatusBadGateway, map[string]string{
			"error": "failed to fetch GitHub activity",
		})
	}
}

func publicUpstreamError(err error) string {
	message := err.Error()
	message = strings.TrimPrefix(message, analysis.ErrUpstreamUnavailable.Error())
	message = strings.TrimPrefix(message, ":")
	message = strings.TrimSpace(message)

	if message == "" {
		return "failed to fetch GitHub activity"
	}

	return fmt.Sprintf("GitHub API request failed: %s", message)
}

func writeMethodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
		"error": "method not allowed",
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("failed to encode json response: %v", err)
	}
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
