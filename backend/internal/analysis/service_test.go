package analysis

import "testing"

func TestAnalyzeUser(t *testing.T) {
	result := AnalyzeUser("octocat")

	if result.Username != "octocat" {
		t.Fatalf("expected username to be preserved, got %q", result.Username)
	}

	if result.PersonaTitle == "" {
		t.Fatal("expected persona title to be present")
	}

	if len(result.Traits) == 0 {
		t.Fatal("expected traits to be present")
	}
}
