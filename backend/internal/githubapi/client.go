package githubapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/saku0512/GitHub-Contribution-Visualizer/backend/internal/analysis"
)

const defaultGraphQLEndpoint = "https://api.github.com/graphql"

var defaultOAuthAuthorizeEndpoint = "https://github.com/login/oauth/authorize"
var defaultOAuthTokenEndpoint = "https://github.com/login/oauth/access_token"
var defaultRESTEndpoint = "https://api.github.com/user"

type Client struct {
	token             string
	endpoint          string
	oauthClientID     string
	oauthClientSecret string
	http              *http.Client
}

func NewClient(token string, endpoint string) *Client {
	if endpoint == "" {
		endpoint = defaultGraphQLEndpoint
	}

	return &Client{
		token:             token,
		endpoint:          endpoint,
		oauthClientID:     strings.TrimSpace(os.Getenv("GITHUB_OAUTH_CLIENT_ID")),
		oauthClientSecret: strings.TrimSpace(os.Getenv("GITHUB_OAUTH_CLIENT_SECRET")),
		http: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *Client) HasToken() bool {
	return c.token != ""
}

func (c *Client) HasOAuthCredentials() bool {
	return c.oauthClientID != "" && c.oauthClientSecret != ""
}

type AuthenticatedUser struct {
	Login     string `json:"login"`
	Name      string `json:"name"`
	URL       string `json:"html_url"`
	AvatarURL string `json:"avatar_url"`
}

func (c *Client) BuildAuthorizationURL(state string, redirectURI string) (string, error) {
	if !c.HasOAuthCredentials() {
		return "", fmt.Errorf("github oauth credentials are not configured")
	}

	authorizeURL, err := url.Parse(defaultOAuthAuthorizeEndpoint)
	if err != nil {
		return "", fmt.Errorf("parse github oauth authorize endpoint: %w", err)
	}

	query := authorizeURL.Query()
	query.Set("client_id", c.oauthClientID)
	query.Set("redirect_uri", redirectURI)
	query.Set("state", state)
	authorizeURL.RawQuery = query.Encode()

	return authorizeURL.String(), nil
}

func (c *Client) ExchangeCodeForUser(ctx context.Context, code string, redirectURI string) (AuthenticatedUser, error) {
	if !c.HasOAuthCredentials() {
		return AuthenticatedUser{}, fmt.Errorf("%w: GitHub OAuth credentials are not configured", analysis.ErrUpstreamUnavailable)
	}

	payload := map[string]string{
		"client_id":     c.oauthClientID,
		"client_secret": c.oauthClientSecret,
		"code":          code,
		"redirect_uri":  redirectURI,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return AuthenticatedUser{}, fmt.Errorf("marshal github oauth token request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, defaultOAuthTokenEndpoint, bytes.NewReader(body))
	if err != nil {
		return AuthenticatedUser{}, fmt.Errorf("build github oauth token request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "github-contribution-visualizer")

	resp, err := c.http.Do(req)
	if err != nil {
		return AuthenticatedUser{}, fmt.Errorf("%w: %v", analysis.ErrUpstreamUnavailable, err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return AuthenticatedUser{}, fmt.Errorf("read github oauth token response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return AuthenticatedUser{}, fmt.Errorf("%w: %s", analysis.ErrUpstreamUnavailable, describeHTTPError(resp.StatusCode, bodyBytes))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		Description string `json:"error_description"`
	}
	if err := json.Unmarshal(bodyBytes, &tokenResp); err != nil {
		return AuthenticatedUser{}, fmt.Errorf("decode github oauth token response: %w", err)
	}

	if tokenResp.Error != "" {
		description := strings.TrimSpace(tokenResp.Description)
		if description == "" {
			description = tokenResp.Error
		}
		return AuthenticatedUser{}, fmt.Errorf("%w: GitHub OAuth token exchange failed: %s", analysis.ErrUpstreamUnavailable, description)
	}

	if tokenResp.AccessToken == "" {
		return AuthenticatedUser{}, fmt.Errorf("%w: GitHub OAuth token exchange returned no access token", analysis.ErrUpstreamUnavailable)
	}

	return c.fetchAuthenticatedUser(ctx, tokenResp.AccessToken)
}

func (c *Client) fetchAuthenticatedUser(ctx context.Context, accessToken string) (AuthenticatedUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, defaultRESTEndpoint, nil)
	if err != nil {
		return AuthenticatedUser{}, fmt.Errorf("build github user request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "github-contribution-visualizer")

	resp, err := c.http.Do(req)
	if err != nil {
		return AuthenticatedUser{}, fmt.Errorf("%w: %v", analysis.ErrUpstreamUnavailable, err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return AuthenticatedUser{}, fmt.Errorf("read github user response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return AuthenticatedUser{}, fmt.Errorf("%w: %s", analysis.ErrUpstreamUnavailable, describeHTTPError(resp.StatusCode, bodyBytes))
	}

	var user AuthenticatedUser
	if err := json.Unmarshal(bodyBytes, &user); err != nil {
		return AuthenticatedUser{}, fmt.Errorf("decode github user response: %w", err)
	}

	if strings.TrimSpace(user.Login) == "" {
		return AuthenticatedUser{}, fmt.Errorf("%w: GitHub user response did not include a login", analysis.ErrUpstreamUnavailable)
	}

	return user, nil
}

func (c *Client) FetchUserActivity(ctx context.Context, username string, from, to time.Time) (analysis.Snapshot, error) {
	if c.token == "" {
		return analysis.Snapshot{}, analysis.ErrGitHubTokenMissing
	}

	payload := map[string]any{
		"query": githubUserActivityQuery,
		"variables": map[string]string{
			"login": username,
			"from":  from.Format(time.RFC3339),
			"to":    to.Format(time.RFC3339),
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return analysis.Snapshot{}, fmt.Errorf("marshal github graphql request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return analysis.Snapshot{}, fmt.Errorf("build github graphql request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "github-contribution-visualizer")

	resp, err := c.http.Do(req)
	if err != nil {
		return analysis.Snapshot{}, fmt.Errorf("%w: %v", analysis.ErrUpstreamUnavailable, err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return analysis.Snapshot{}, fmt.Errorf("read github graphql response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return analysis.Snapshot{}, analysis.ErrUserNotFound
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return analysis.Snapshot{}, fmt.Errorf("%w: %s", analysis.ErrUpstreamUnavailable, describeHTTPError(resp.StatusCode, bodyBytes))
	}

	var graphResp graphQLResponse
	if err := json.Unmarshal(bodyBytes, &graphResp); err != nil {
		return analysis.Snapshot{}, fmt.Errorf("decode github graphql response: %w", err)
	}

	if len(graphResp.Errors) > 0 {
		if hasNotFoundError(graphResp.Errors) {
			return analysis.Snapshot{}, analysis.ErrUserNotFound
		}

		return analysis.Snapshot{}, fmt.Errorf("%w: %s", analysis.ErrUpstreamUnavailable, graphResp.Errors[0].Message)
	}

	if graphResp.Data.User == nil {
		return analysis.Snapshot{}, analysis.ErrUserNotFound
	}

	return toSnapshot(*graphResp.Data.User)
}

type graphQLResponse struct {
	Data struct {
		User *graphUser `json:"user"`
	} `json:"data"`
	Errors []graphQLError `json:"errors"`
}

type graphQLError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type graphUser struct {
	Login                   string `json:"login"`
	Name                    string `json:"name"`
	URL                     string `json:"url"`
	AvatarURL               string `json:"avatarUrl"`
	ContributionsCollection struct {
		TotalCommitContributions                   int                           `json:"totalCommitContributions"`
		TotalIssueContributions                    int                           `json:"totalIssueContributions"`
		TotalPullRequestContributions              int                           `json:"totalPullRequestContributions"`
		TotalPullRequestReviewContributions        int                           `json:"totalPullRequestReviewContributions"`
		ContributionCalendar                       graphContributionCalendar     `json:"contributionCalendar"`
		CommitContributionsByRepository            []graphRepositoryContribution `json:"commitContributionsByRepository"`
		IssueContributionsByRepository             []graphRepositoryContribution `json:"issueContributionsByRepository"`
		PullRequestContributionsByRepository       []graphRepositoryContribution `json:"pullRequestContributionsByRepository"`
		PullRequestReviewContributionsByRepository []graphRepositoryContribution `json:"pullRequestReviewContributionsByRepository"`
	} `json:"contributionsCollection"`
}

type graphContributionCalendar struct {
	Weeks []struct {
		ContributionDays []struct {
			Date              string `json:"date"`
			ContributionCount int    `json:"contributionCount"`
		} `json:"contributionDays"`
	} `json:"weeks"`
}

type graphRepositoryContribution struct {
	Repository struct {
		NameWithOwner string `json:"nameWithOwner"`
		URL           string `json:"url"`
	} `json:"repository"`
	Contributions struct {
		TotalCount int `json:"totalCount"`
	} `json:"contributions"`
}

func hasNotFoundError(errors []graphQLError) bool {
	for _, err := range errors {
		if err.Type == "NOT_FOUND" {
			return true
		}
	}

	return false
}

func describeHTTPError(statusCode int, body []byte) string {
	bodyText := strings.TrimSpace(string(body))

	switch statusCode {
	case http.StatusUnauthorized:
		return "GitHub API returned 401 Unauthorized; check GITHUB_TOKEN"
	case http.StatusForbidden:
		if strings.Contains(strings.ToLower(bodyText), "rate limit") {
			return "GitHub API rate limit exceeded"
		}
		return "GitHub API returned 403 Forbidden; token may lack required access"
	default:
		if bodyText == "" {
			return fmt.Sprintf("GitHub API returned %d", statusCode)
		}
		if len(bodyText) > 180 {
			bodyText = bodyText[:180] + "..."
		}
		return fmt.Sprintf("GitHub API returned %d: %s", statusCode, bodyText)
	}
}

func toSnapshot(user graphUser) (analysis.Snapshot, error) {
	days := make([]analysis.ContributionDay, 0, 366)
	for _, week := range user.ContributionsCollection.ContributionCalendar.Weeks {
		for _, day := range week.ContributionDays {
			parsedDate, err := time.Parse(time.DateOnly, day.Date)
			if err != nil {
				return analysis.Snapshot{}, fmt.Errorf("parse contribution date %q: %w", day.Date, err)
			}

			days = append(days, analysis.ContributionDay{
				Date:  parsedDate,
				Count: day.ContributionCount,
			})
		}
	}

	repositories := mergeRepositoryActivity(
		user.ContributionsCollection.CommitContributionsByRepository,
		user.ContributionsCollection.PullRequestContributionsByRepository,
		user.ContributionsCollection.IssueContributionsByRepository,
		user.ContributionsCollection.PullRequestReviewContributionsByRepository,
	)

	return analysis.Snapshot{
		Profile: analysis.Profile{
			Login:     user.Login,
			Name:      user.Name,
			URL:       user.URL,
			AvatarURL: user.AvatarURL,
		},
		ContributionDays: days,
		ContributionTypes: analysis.ContributionTypes{
			Commits:      user.ContributionsCollection.TotalCommitContributions,
			PullRequests: user.ContributionsCollection.TotalPullRequestContributions,
			Issues:       user.ContributionsCollection.TotalIssueContributions,
			Reviews:      user.ContributionsCollection.TotalPullRequestReviewContributions,
		},
		TopRepositories: repositories,
	}, nil
}

func mergeRepositoryActivity(
	commits []graphRepositoryContribution,
	pullRequests []graphRepositoryContribution,
	issues []graphRepositoryContribution,
	reviews []graphRepositoryContribution,
) []analysis.RepositoryActivity {
	merged := make(map[string]*analysis.RepositoryActivity)

	apply := func(items []graphRepositoryContribution, target func(*analysis.RepositoryActivity) *int) {
		for _, item := range items {
			entry := merged[item.Repository.NameWithOwner]
			if entry == nil {
				entry = &analysis.RepositoryActivity{
					NameWithOwner: item.Repository.NameWithOwner,
					URL:           item.Repository.URL,
				}
				merged[item.Repository.NameWithOwner] = entry
			}

			*target(entry) += item.Contributions.TotalCount
			entry.Total += item.Contributions.TotalCount
		}
	}

	apply(commits, func(item *analysis.RepositoryActivity) *int { return &item.Commits })
	apply(pullRequests, func(item *analysis.RepositoryActivity) *int { return &item.PullRequests })
	apply(issues, func(item *analysis.RepositoryActivity) *int { return &item.Issues })
	apply(reviews, func(item *analysis.RepositoryActivity) *int { return &item.Reviews })

	result := make([]analysis.RepositoryActivity, 0, len(merged))
	for _, item := range merged {
		result = append(result, *item)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Total == result[j].Total {
			return result[i].NameWithOwner < result[j].NameWithOwner
		}

		return result[i].Total > result[j].Total
	})

	if len(result) > 5 {
		return result[:5]
	}

	return result
}

var githubUserActivityQuery = `
query UserActivity($login: String!, $from: DateTime!, $to: DateTime!) {
  user(login: $login) {
    login
    name
    url
    avatarUrl
    contributionsCollection(from: $from, to: $to) {
      totalCommitContributions
      totalIssueContributions
      totalPullRequestContributions
      totalPullRequestReviewContributions
      contributionCalendar {
        weeks {
          contributionDays {
            date
            contributionCount
          }
        }
      }
      commitContributionsByRepository(maxRepositories: 8) {
        repository {
          nameWithOwner
          url
        }
        contributions(first: 1) {
          totalCount
        }
      }
      issueContributionsByRepository(maxRepositories: 8) {
        repository {
          nameWithOwner
          url
        }
        contributions(first: 1) {
          totalCount
        }
      }
      pullRequestContributionsByRepository(maxRepositories: 8) {
        repository {
          nameWithOwner
          url
        }
        contributions(first: 1) {
          totalCount
        }
      }
      pullRequestReviewContributionsByRepository(maxRepositories: 8) {
        repository {
          nameWithOwner
          url
        }
        contributions(first: 1) {
          totalCount
        }
      }
    }
  }
}
`

func IsNotFound(err error) bool {
	return errors.Is(err, analysis.ErrUserNotFound)
}
