package analysis

import (
	"context"
	"errors"
	"testing"
	"time"
)

type stubFetcher struct {
	snapshot Snapshot
	err      error
}

func (s stubFetcher) FetchUserActivity(context.Context, string, time.Time, time.Time) (Snapshot, error) {
	return s.snapshot, s.err
}

func TestAnalyzeUserBuildsMetricsFromSnapshot(t *testing.T) {
	service := NewService(stubFetcher{
		snapshot: Snapshot{
			Profile: Profile{
				Login: "octocat",
				Name:  "The Octocat",
				URL:   "https://github.com/octocat",
			},
			ContributionDays: []ContributionDay{
				{Date: mustDate("2026-04-10"), Count: 3},
				{Date: mustDate("2026-04-11"), Count: 4},
				{Date: mustDate("2026-04-12"), Count: 0},
				{Date: mustDate("2026-04-13"), Count: 6},
				{Date: mustDate("2026-04-14"), Count: 8},
			},
			ContributionTypes: ContributionTypes{
				Commits:      8,
				PullRequests: 3,
				Issues:       1,
				Reviews:      2,
			},
			TopRepositories: []RepositoryActivity{
				{NameWithOwner: "openai/demo", Total: 6},
			},
		},
	}, func() time.Time {
		return time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	})

	result, err := service.AnalyzeUser(context.Background(), "octocat")
	if err != nil {
		t.Fatalf("AnalyzeUser returned error: %v", err)
	}

	if result.Username != "octocat" {
		t.Fatalf("expected username octocat, got %q", result.Username)
	}

	if result.Stats.TotalContributions != 21 {
		t.Fatalf("expected total contributions 21, got %d", result.Stats.TotalContributions)
	}

	if result.Stats.ActiveDays != 4 {
		t.Fatalf("expected 4 active days, got %d", result.Stats.ActiveDays)
	}

	if result.Stats.LongestStreak != 2 {
		t.Fatalf("expected longest streak 2, got %d", result.Stats.LongestStreak)
	}

	if result.Stats.DominantActivity != "commits" {
		t.Fatalf("expected dominant activity commits, got %q", result.Stats.DominantActivity)
	}

	if len(result.Traits) == 0 {
		t.Fatal("expected traits to be present")
	}
}

func TestAnalyzeUserRejectsInvalidUsername(t *testing.T) {
	service := NewService(stubFetcher{}, time.Now)

	_, err := service.AnalyzeUser(context.Background(), "not valid")
	if !errors.Is(err, ErrInvalidUsername) {
		t.Fatalf("expected ErrInvalidUsername, got %v", err)
	}
}

func mustDate(value string) time.Time {
	parsed, err := time.Parse(time.DateOnly, value)
	if err != nil {
		panic(err)
	}

	return parsed
}
