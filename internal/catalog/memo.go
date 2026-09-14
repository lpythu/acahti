package catalog

import (
	"sync"
	"time"

	"acahti/internal/forgejo"
)

const memoTTL = 30 * time.Second

type stamp[T any] struct {
	at time.Time
	v  T
}

type memo struct {
	mu    sync.Mutex
	admin map[string]stamp[bool]
	repos map[string]stamp[[]forgejo.Repo]
	teams stamp[[]forgejo.Team]
}

func newMemo() *memo {
	return &memo{
		admin: map[string]stamp[bool]{},
		repos: map[string]stamp[[]forgejo.Repo]{},
	}
}

func got[T any](tab map[string]stamp[T], key string) (T, bool) {
	e, ok := tab[key]
	if !ok || time.Since(e.at) > memoTTL {
		var z T
		return z, false
	}
	return e.v, true
}

func (m *memo) adminOf(user string) (bool, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return got(m.admin, user)
}

func (m *memo) setAdmin(user string, ok bool) {
	m.mu.Lock()
	m.admin[user] = stamp[bool]{at: time.Now(), v: ok}
	m.mu.Unlock()
}

func (m *memo) reposOf(user string) ([]forgejo.Repo, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	repos, ok := got(m.repos, user)
	if !ok {
		return nil, false
	}
	return append([]forgejo.Repo{}, repos...), true
}

func (m *memo) setRepos(user string, repos []forgejo.Repo) {
	m.mu.Lock()
	m.repos[user] = stamp[[]forgejo.Repo]{at: time.Now(), v: append([]forgejo.Repo{}, repos...)}
	m.mu.Unlock()
}

func (m *memo) teamsOf() ([]forgejo.Team, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.teams.at.IsZero() || time.Since(m.teams.at) > memoTTL {
		return nil, false
	}
	return append([]forgejo.Team{}, m.teams.v...), true
}

func (m *memo) setTeams(teams []forgejo.Team) {
	m.mu.Lock()
	m.teams = stamp[[]forgejo.Team]{at: time.Now(), v: append([]forgejo.Team{}, teams...)}
	m.mu.Unlock()
}

func (m *memo) drop() {
	m.mu.Lock()
	m.admin = map[string]stamp[bool]{}
	m.repos = map[string]stamp[[]forgejo.Repo]{}
	m.teams = stamp[[]forgejo.Team]{}
	m.mu.Unlock()
}

func (c *Catalog) forget() {
	if c.mem != nil {
		c.mem.drop()
	}
}
