package analysis

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

var (
	ErrInvalidUsername     = errors.New("invalid username")
	ErrUserNotFound        = errors.New("github user not found")
	ErrGitHubTokenMissing  = errors.New("github token not configured")
	ErrUpstreamUnavailable = errors.New("github upstream unavailable")
)

type ActivityFetcher interface {
	FetchUserActivity(ctx context.Context, username string, from, to time.Time) (Snapshot, error)
}

type Service struct {
	fetcher ActivityFetcher
	now     func() time.Time
}

type Snapshot struct {
	Profile           Profile
	ContributionDays  []ContributionDay
	ContributionTypes ContributionTypes
	TopRepositories   []RepositoryActivity
}

type Profile struct {
	Login     string
	Name      string
	URL       string
	AvatarURL string
}

type ContributionDay struct {
	Date  time.Time
	Count int
}

type ContributionTypes struct {
	Commits      int
	PullRequests int
	Issues       int
	Reviews      int
}

type RepositoryActivity struct {
	NameWithOwner string `json:"nameWithOwner"`
	URL           string `json:"url"`
	Commits       int    `json:"commits"`
	PullRequests  int    `json:"pullRequests"`
	Issues        int    `json:"issues"`
	Reviews       int    `json:"reviews"`
	Total         int    `json:"total"`
}

type Result struct {
	Username           string               `json:"username"`
	PersonaTitle       string               `json:"personaTitle"`
	Summary            string               `json:"summary"`
	Traits             []string             `json:"traits"`
	RecommendedAsset   string               `json:"recommendedAsset"`
	VisualDirection    string               `json:"visualDirection"`
	ContributionSignal string               `json:"contributionSignal"`
	Stats              Stats                `json:"stats"`
	Profile            ResultProfile        `json:"profile"`
	TopRepositories    []RepositoryActivity `json:"topRepositories"`
}

type ResultProfile struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	AvatarURL string `json:"avatarUrl"`
}

type Stats struct {
	From                 string  `json:"from"`
	To                   string  `json:"to"`
	TotalContributions   int     `json:"totalContributions"`
	ActiveDays           int     `json:"activeDays"`
	PeakDayContributions int     `json:"peakDayContributions"`
	LongestStreak        int     `json:"longestStreak"`
	CurrentStreak        int     `json:"currentStreak"`
	WeekendRatio         float64 `json:"weekendRatio"`
	BusiestWeekday       string  `json:"busiestWeekday"`
	DominantActivity     string  `json:"dominantActivity"`
	CommitCount          int     `json:"commitCount"`
	PullRequestCount     int     `json:"pullRequestCount"`
	IssueCount           int     `json:"issueCount"`
	ReviewCount          int     `json:"reviewCount"`
}

type metrics struct {
	totalContributions   int
	activeDays           int
	peakDayContributions int
	longestStreak        int
	currentStreak        int
	weekendRatio         float64
	busiestWeekday       string
	dominantActivity     string
	averageActiveDay     float64
}

func NewService(fetcher ActivityFetcher, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}

	return &Service{
		fetcher: fetcher,
		now:     now,
	}
}

func (s *Service) AnalyzeUser(ctx context.Context, username string) (Result, error) {
	normalized := strings.TrimSpace(username)
	if !isValidUsername(normalized) {
		return Result{}, ErrInvalidUsername
	}

	from := s.now().UTC().AddDate(-1, 0, 0)
	to := s.now().UTC()

	snapshot, err := s.fetcher.FetchUserActivity(ctx, normalized, from, to)
	if err != nil {
		return Result{}, err
	}

	return buildResult(snapshot, from, to), nil
}

func buildResult(snapshot Snapshot, from, to time.Time) Result {
	metrics := calculateMetrics(snapshot)
	persona := selectPersona(metrics, snapshot.ContributionTypes)
	traits := buildTraits(metrics, snapshot.ContributionTypes)

	return Result{
		Username:           snapshot.Profile.Login,
		PersonaTitle:       persona.title,
		Summary:            buildSummary(snapshot, metrics, persona.title),
		Traits:             traits,
		RecommendedAsset:   persona.asset,
		VisualDirection:    persona.visualDirection,
		ContributionSignal: buildContributionSignal(metrics),
		Stats: Stats{
			From:                 from.Format(time.DateOnly),
			To:                   to.Format(time.DateOnly),
			TotalContributions:   metrics.totalContributions,
			ActiveDays:           metrics.activeDays,
			PeakDayContributions: metrics.peakDayContributions,
			LongestStreak:        metrics.longestStreak,
			CurrentStreak:        metrics.currentStreak,
			WeekendRatio:         round2(metrics.weekendRatio),
			BusiestWeekday:       metrics.busiestWeekday,
			DominantActivity:     metrics.dominantActivity,
			CommitCount:          snapshot.ContributionTypes.Commits,
			PullRequestCount:     snapshot.ContributionTypes.PullRequests,
			IssueCount:           snapshot.ContributionTypes.Issues,
			ReviewCount:          snapshot.ContributionTypes.Reviews,
		},
		Profile: ResultProfile{
			Name:      snapshot.Profile.Name,
			URL:       snapshot.Profile.URL,
			AvatarURL: snapshot.Profile.AvatarURL,
		},
		TopRepositories: snapshot.TopRepositories,
	}
}

type personaTemplate struct {
	title           string
	asset           string
	visualDirection string
}

func selectPersona(m metrics, activity ContributionTypes) personaTemplate {
	collaborationTotal := activity.PullRequests + activity.Reviews
	if collaborationTotal >= activity.Commits && collaborationTotal >= 20 {
		return personaTemplate{
			title:           "Collaboration Captain",
			asset:           "3Dモデル",
			visualDirection: "チームの流れを束ねる司令塔風アバター。明快なラインとアクセントカラーで構成。",
		}
	}

	if m.weekendRatio >= 0.35 {
		return personaTemplate{
			title:           "Weekend Inventor",
			asset:           "3Dモデル",
			visualDirection: "ポップで実験的なガジェット風デザイン。遊び心のある差し色を強めに配置。",
		}
	}

	if m.longestStreak >= 21 || m.activeDays >= 180 {
		return personaTemplate{
			title:           "Steady Gardener",
			asset:           "ドットキャラ",
			visualDirection: "安定感のある配色と自然モチーフを組み合わせた、育成ゲーム風のピクセルアート。",
		}
	}

	if m.peakDayContributions >= 12 && m.averageActiveDay > 0 && m.peakDayContributions >= int(m.averageActiveDay*2.2) {
		return personaTemplate{
			title:           "Sprint Crafter",
			asset:           "3Dモデル",
			visualDirection: "スピード感のあるモーションラインと工房感のあるパーツ構成を持つクラフト系デザイン。",
		}
	}

	return personaTemplate{
		title:           "Momentum Builder",
		asset:           "ドットキャラ",
		visualDirection: "前進感のあるシルエットに、整った色面と軽快なアクセントを乗せたヒーロー調。",
	}
}

func buildTraits(m metrics, activity ContributionTypes) []string {
	traits := make([]string, 0, 4)

	if m.activeDays >= 180 {
		traits = append(traits, "年間を通して継続的に手を動かす")
	} else if m.longestStreak >= 14 {
		traits = append(traits, "まとまった連続 streak を作れる")
	}

	if m.weekendRatio >= 0.35 {
		traits = append(traits, "週末に試作や集中実装を進めやすい")
	}

	switch m.dominantActivity {
	case "commits":
		traits = append(traits, "コミットで実装を前に進める駆動力がある")
	case "pull_requests":
		traits = append(traits, "プルリクエストで変化を形にするのが得意")
	case "reviews":
		traits = append(traits, "レビューでチーム全体の品質を押し上げる")
	case "issues":
		traits = append(traits, "課題整理と論点の見える化が得意")
	}

	if m.peakDayContributions >= 10 {
		traits = append(traits, "波が来た日に一気にアウトプットを伸ばせる")
	}

	if activity.PullRequests+activity.Reviews >= 20 {
		traits = append(traits, "コラボレーションの比重が高い")
	}

	if len(traits) < 3 {
		traits = append(traits, "安定したペースで改善を積み重ねる")
	}

	unique := uniqueStrings(traits)
	return unique[:min(3, len(unique))]
}

func buildSummary(snapshot Snapshot, m metrics, personaTitle string) string {
	name := snapshot.Profile.Login
	if snapshot.Profile.Name != "" {
		name = snapshot.Profile.Name
	}

	return fmt.Sprintf(
		"%s は過去1年で %d 件の contribution を積み上げ、%d 日アクティブに動いた %s タイプです。最長 %d 日の streak と %s 優位の動きから、%s な開発スタイルが見えます。",
		name,
		m.totalContributions,
		m.activeDays,
		personaTitle,
		m.longestStreak,
		activityLabel(m.dominantActivity),
		summaryStyle(personaTitle),
	)
}

func buildContributionSignal(m metrics) string {
	return fmt.Sprintf(
		"年間 %d contributions / active %d 日 / 最長 streak %d 日 / busiest %s",
		m.totalContributions,
		m.activeDays,
		m.longestStreak,
		m.busiestWeekday,
	)
}

func calculateMetrics(snapshot Snapshot) metrics {
	days := append([]ContributionDay(nil), snapshot.ContributionDays...)
	sort.Slice(days, func(i, j int) bool {
		return days[i].Date.Before(days[j].Date)
	})

	total := 0
	activeDays := 0
	peak := 0
	currentStreak := 0
	longestStreak := 0
	rolling := 0
	weekend := 0
	weekdayTotals := make(map[time.Weekday]int, 7)

	for _, day := range days {
		total += day.Count
		weekdayTotals[day.Date.Weekday()] += day.Count

		if day.Count > peak {
			peak = day.Count
		}

		if day.Count > 0 {
			activeDays++
			rolling++
			if day.Date.Weekday() == time.Saturday || day.Date.Weekday() == time.Sunday {
				weekend += day.Count
			}
		} else {
			if rolling > longestStreak {
				longestStreak = rolling
			}
			rolling = 0
		}
	}

	if rolling > longestStreak {
		longestStreak = rolling
	}

	for i := len(days) - 1; i >= 0; i-- {
		if days[i].Count == 0 {
			break
		}
		currentStreak++
	}

	weekendRatio := 0.0
	if total > 0 {
		weekendRatio = float64(weekend) / float64(total)
	}

	averageActiveDay := 0.0
	if activeDays > 0 {
		averageActiveDay = float64(total) / float64(activeDays)
	}

	return metrics{
		totalContributions:   total,
		activeDays:           activeDays,
		peakDayContributions: peak,
		longestStreak:        longestStreak,
		currentStreak:        currentStreak,
		weekendRatio:         weekendRatio,
		busiestWeekday:       busiestWeekday(weekdayTotals),
		dominantActivity:     dominantActivity(snapshot.ContributionTypes),
		averageActiveDay:     averageActiveDay,
	}
}

func busiestWeekday(totals map[time.Weekday]int) string {
	bestDay := time.Monday
	bestTotal := -1

	for day := time.Sunday; day <= time.Saturday; day++ {
		if totals[day] > bestTotal {
			bestDay = day
			bestTotal = totals[day]
		}
	}

	labels := map[time.Weekday]string{
		time.Sunday:    "Sunday",
		time.Monday:    "Monday",
		time.Tuesday:   "Tuesday",
		time.Wednesday: "Wednesday",
		time.Thursday:  "Thursday",
		time.Friday:    "Friday",
		time.Saturday:  "Saturday",
	}

	return labels[bestDay]
}

func dominantActivity(activity ContributionTypes) string {
	type score struct {
		label string
		count int
	}

	scores := []score{
		{label: "commits", count: activity.Commits},
		{label: "pull_requests", count: activity.PullRequests},
		{label: "issues", count: activity.Issues},
		{label: "reviews", count: activity.Reviews},
	}

	sort.SliceStable(scores, func(i, j int) bool {
		return scores[i].count > scores[j].count
	})

	if scores[0].count == 0 {
		return "commits"
	}

	return scores[0].label
}

func activityLabel(label string) string {
	switch label {
	case "pull_requests":
		return "プルリクエスト"
	case "issues":
		return "Issue"
	case "reviews":
		return "レビュー"
	default:
		return "コミット"
	}
}

func summaryStyle(personaTitle string) string {
	switch personaTitle {
	case "Collaboration Captain":
		return "周囲を巻き込みながら進める"
	case "Weekend Inventor":
		return "試作と発明を楽しむ"
	case "Steady Gardener":
		return "着実に育てていく"
	case "Sprint Crafter":
		return "短い集中で一気に形にする"
	default:
		return "安定して前進する"
	}
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))

	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}

	return result
}

func round2(value float64) float64 {
	return float64(int(value*100+0.5)) / 100
}

func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}

func isValidUsername(username string) bool {
	if username == "" || len(username) > 39 {
		return false
	}

	if username[0] == '-' || username[len(username)-1] == '-' {
		return false
	}

	prevHyphen := false
	for _, r := range username {
		isLetter := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'
		isDigit := r >= '0' && r <= '9'

		switch {
		case isLetter || isDigit:
			prevHyphen = false
		case r == '-':
			if prevHyphen {
				return false
			}
			prevHyphen = true
		default:
			return false
		}
	}

	return true
}
