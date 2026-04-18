package githubapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/saku0512/GitHub-Contribution-Visualizer/backend/internal/analysis"
)

func TestFetchUserActivity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("unexpected auth header: %q", got)
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		response := map[string]any{
			"data": map[string]any{
				"user": map[string]any{
					"login":     "octocat",
					"name":      "The Octocat",
					"url":       "https://github.com/octocat",
					"avatarUrl": "https://github.com/images/error/octocat_happy.gif",
					"contributionsCollection": map[string]any{
						"totalCommitContributions":            12,
						"totalIssueContributions":             3,
						"totalPullRequestContributions":       5,
						"totalPullRequestReviewContributions": 7,
						"contributionCalendar": map[string]any{
							"weeks": []any{
								map[string]any{
									"contributionDays": []any{
										map[string]any{"date": "2026-04-15", "contributionCount": 2},
										map[string]any{"date": "2026-04-16", "contributionCount": 0},
									},
								},
							},
						},
						"commitContributionsByRepository": []any{
							map[string]any{
								"repository":    map[string]any{"nameWithOwner": "openai/demo", "url": "https://github.com/openai/demo"},
								"contributions": map[string]any{"totalCount": 12},
							},
						},
						"issueContributionsByRepository": []any{
							map[string]any{
								"repository":    map[string]any{"nameWithOwner": "openai/demo", "url": "https://github.com/openai/demo"},
								"contributions": map[string]any{"totalCount": 3},
							},
						},
						"pullRequestContributionsByRepository": []any{
							map[string]any{
								"repository":    map[string]any{"nameWithOwner": "openai/visualizer", "url": "https://github.com/openai/visualizer"},
								"contributions": map[string]any{"totalCount": 5},
							},
						},
						"pullRequestReviewContributionsByRepository": []any{
							map[string]any{
								"repository":    map[string]any{"nameWithOwner": "openai/visualizer", "url": "https://github.com/openai/visualizer"},
								"contributions": map[string]any{"totalCount": 7},
							},
						},
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient("test-token", server.URL)
	snapshot, err := client.FetchUserActivity(context.Background(), "octocat", time.Now().AddDate(-1, 0, 0), time.Now())
	if err != nil {
		t.Fatalf("FetchUserActivity returned error: %v", err)
	}

	if snapshot.Profile.Login != "octocat" {
		t.Fatalf("expected login octocat, got %q", snapshot.Profile.Login)
	}

	if snapshot.ContributionTypes.Commits != 12 {
		t.Fatalf("expected 12 commits, got %d", snapshot.ContributionTypes.Commits)
	}

	if len(snapshot.TopRepositories) != 2 {
		t.Fatalf("expected 2 merged repositories, got %d", len(snapshot.TopRepositories))
	}
}

func TestFetchUserActivityWithoutToken(t *testing.T) {
	client := NewClient("", "http://example.com")
	_, err := client.FetchUserActivity(context.Background(), "octocat", time.Now().AddDate(-1, 0, 0), time.Now())
	if err == nil {
		t.Fatal("expected missing token error")
	}

	if !errors.Is(err, analysis.ErrGitHubTokenMissing) {
		t.Fatalf("expected ErrGitHubTokenMissing, got %v", err)
	}
}
