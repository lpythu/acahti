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

type commitWho struct {
	Name   string
	Login  string
	Avatar string
	Email  string
}

type memo struct {
	mu      sync.Mutex
	admin   map[string]stamp[bool]
	authors stamp[map[string]string]
	files   map[string][]FileBlob
	who     map[string]commitWho
}

func newMemo() *memo {
	return &memo{
		admin: map[string]stamp[bool]{},
		files: map[string][]FileBlob{},
		who:   map[string]commitWho{},
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

func (m *memo) authorsOf() (map[string]string, bool) {
	if m == nil {
		return nil, false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.authors.v == nil || time.Since(m.authors.at) > memoTTL {
		return nil, false
	}
	out := make(map[string]string, len(m.authors.v))
	for k, v := range m.authors.v {
		out[k] = v
	}
	return out, true
}

func (m *memo) setAuthors(names map[string]string) {
	if m == nil {
		return
	}
	cp := make(map[string]string, len(names))
	for k, v := range names {
		cp[k] = v
	}
	m.mu.Lock()
	m.authors = stamp[map[string]string]{at: time.Now(), v: cp}
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

func (m *memo) commitWhoOf(repo, sha string) (commitWho, bool) {
	if m == nil || repo == "" || sha == "" {
		return commitWho{}, false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	w, ok := m.who[fileKey(repo, sha)]
	return w, ok
}

func (m *memo) setCommitWho(repo, sha string, w commitWho) {
	if m == nil || repo == "" || sha == "" {
		return
	}
	m.mu.Lock()
	m.who[fileKey(repo, sha)] = w
	m.mu.Unlock()
}

func (m *memo) drop() {
	m.mu.Lock()
	m.admin = map[string]stamp[bool]{}
	m.authors = stamp[map[string]string]{}
	m.who = map[string]commitWho{}
	m.mu.Unlock()
}

func (m *memo) dropAuthors() {
	m.mu.Lock()
	m.authors = stamp[map[string]string]{}
	m.mu.Unlock()
}

func (c *Catalog) forget() {
	if c.mem != nil {
		c.mem.drop()
	}
}

func (c *Catalog) ForgetAuthors() {
	if c != nil && c.mem != nil {
		c.mem.dropAuthors()
	}
}
