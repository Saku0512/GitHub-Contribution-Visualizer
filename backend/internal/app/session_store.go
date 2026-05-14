package app

import (
	"sort"
	"sync"
	"time"

	"github.com/saku0512/GitHub-Contribution-Visualizer/backend/internal/analysis"
)

type sessionStore struct {
	mu       sync.RWMutex
	sessions map[string]sessionEntry
	profiles map[string]analysis.Result
}

type sessionEntry struct {
	Login     string
	UpdatedAt time.Time
}

func newSessionStore() *sessionStore {
	return &sessionStore{
		sessions: make(map[string]sessionEntry),
		profiles: make(map[string]analysis.Result),
	}
}

func (s *sessionStore) SaveSession(sessionID string, login string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[sessionID] = sessionEntry{
		Login:     login,
		UpdatedAt: time.Now().UTC(),
	}
}

func (s *sessionStore) GetLogin(sessionID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.sessions[sessionID]
	if !ok {
		return "", false
	}

	return entry.Login, true
}

func (s *sessionStore) DeleteSession(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, sessionID)
}

func (s *sessionStore) SaveProfile(result analysis.Result) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.profiles[result.Username] = result
}

func (s *sessionStore) GetProfile(login string) (analysis.Result, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result, ok := s.profiles[login]
	return result, ok
}

func (s *sessionStore) Showcase(limit int) []analysis.Result {
	s.mu.RLock()
	defer s.mu.RUnlock()

	type showcaseItem struct {
		login     string
		updatedAt time.Time
	}

	items := make([]showcaseItem, 0, len(s.sessions))
	seen := make(map[string]bool, len(s.sessions))
	for _, session := range s.sessions {
		if seen[session.Login] {
			continue
		}
		seen[session.Login] = true
		items = append(items, showcaseItem{
			login:     session.Login,
			updatedAt: session.UpdatedAt,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].updatedAt.After(items[j].updatedAt)
	})

	results := make([]analysis.Result, 0, min(limit, len(items)))
	for _, item := range items {
		result, ok := s.profiles[item.login]
		if !ok {
			continue
		}
		results = append(results, result)
		if len(results) == limit {
			break
		}
	}

	return results
}
