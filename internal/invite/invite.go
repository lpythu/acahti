package invite

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Code struct {
	Code string `json:"code"`
}

type Store struct {
	path string
	mu   sync.Mutex
	all  []Code
}

func Open(dir string) (*Store, error) {
	if dir == "" {
		dir = "."
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	s := &Store{path: filepath.Join(dir, "invites.json")}
	b, err := os.ReadFile(s.path)
	if err == nil {
		_ = json.Unmarshal(b, &s.all)
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if err := s.persist(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) persist() error {
	b, err := json.MarshalIndent(s.all, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *Store) List() []Code {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Code, len(s.all))
	copy(out, s.all)
	return out
}

func (s *Store) Create() (Code, error) {
	var raw [10]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return Code{}, err
	}
	c := Code{Code: hex.EncodeToString(raw[:])}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.all = append(s.all, c)
	return c, s.persist()
}

func (s *Store) Valid(code string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.all {
		if c.Code == code {
			return true
		}
	}
	return false
}

func (s *Store) Delete(code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.all[:0]
	for _, c := range s.all {
		if c.Code != code {
			next = append(next, c)
		}
	}
	s.all = next
	return s.persist()
}
