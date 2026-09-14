package catalog

import (
	"sync"
	"time"
)

const memoTTL = 30 * time.Second

type stamp[T any] struct {
	at time.Time
	v  T
}

type memo struct {
	mu    sync.Mutex
	admin map[string]stamp[bool]
	files map[string][]FileBlob
}

func newMemo() *memo {
	return &memo{
		admin: map[string]stamp[bool]{},
		files: map[string][]FileBlob{},
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

func fileKey(repo, ref string) string {
	return repo + "@" + ref
}

func (m *memo) filesOf(repo, ref string) ([]FileBlob, bool) {
	if m == nil || repo == "" || ref == "" {
		return nil, false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	files, ok := m.files[fileKey(repo, ref)]
	if !ok {
		return nil, false
	}
	return append([]FileBlob{}, files...), true
}

func (m *memo) setFiles(repo, ref string, files []FileBlob) {
	if m == nil || repo == "" || ref == "" {
		return
	}
	m.mu.Lock()
	m.files[fileKey(repo, ref)] = append([]FileBlob{}, files...)
	m.mu.Unlock()
}

func (m *memo) drop() {
	m.mu.Lock()
	m.admin = map[string]stamp[bool]{}
	m.mu.Unlock()
}

func (c *Catalog) forget() {
	if c.mem != nil {
		c.mem.drop()
	}
}
