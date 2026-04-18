package analysis

import (
	"fmt"
	"strings"
)

type Result struct {
	Username           string   `json:"username"`
	PersonaTitle       string   `json:"personaTitle"`
	Summary            string   `json:"summary"`
	Traits             []string `json:"traits"`
	RecommendedAsset   string   `json:"recommendedAsset"`
	VisualDirection    string   `json:"visualDirection"`
	ContributionSignal string   `json:"contributionSignal"`
}

func AnalyzeUser(username string) Result {
	normalized := strings.ToLower(strings.TrimSpace(username))
	score := 0
	for _, r := range normalized {
		score += int(r)
	}

	personas := []Result{
		{
			PersonaTitle:       "Night Builder",
			Summary:            "深夜に集中して一気に積み上げる職人タイプ。短い期間で強いアウトプットを出す傾向があります。",
			Traits:             []string{"集中力が高い", "短期スプリントに強い", "手を動かしながら考える"},
			RecommendedAsset:   "3Dモデル",
			VisualDirection:    "ネオンの差し色を入れたメカニカルなアバター",
			ContributionSignal: "高密度なコミット波形",
		},
		{
			PersonaTitle:       "Steady Gardener",
			Summary:            "毎日少しずつ積み上げる安定型。長期的にプロジェクトを育てるのが得意です。",
			Traits:             []string{"継続力が高い", "メンテナンスが丁寧", "チーム開発との相性が良い"},
			RecommendedAsset:   "ドットキャラ",
			VisualDirection:    "自然モチーフと落ち着いた配色のピクセルアート",
			ContributionSignal: "均一で安定した草の並び",
		},
		{
			PersonaTitle:       "Weekend Inventor",
			Summary:            "週末にまとめて試作する発明家タイプ。新しい技術や遊び心のある実装に向いています。",
			Traits:             []string{"探索が好き", "試作スピードが速い", "新技術の導入に前向き"},
			RecommendedAsset:   "3Dモデル",
			VisualDirection:    "ポップで実験的なガジェット風デザイン",
			ContributionSignal: "山型の活動パターン",
		},
	}

	selected := personas[score%len(personas)]
	selected.Username = username
	selected.Summary = fmt.Sprintf("@%s は %s", username, selected.Summary)

	return selected
}
