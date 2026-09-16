package passwd

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Store struct {
	path string
	mu   sync.Mutex
	all  map[string]string
}

func Open(dir string) (*Store, error) {
	if dir == "" {
		dir = "."
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	s := &Store{path: filepath.Join(dir, "passwords.json"), all: map[string]string{}}
	b, err := os.ReadFile(s.path)
	if err == nil {
		_ = json.Unmarshal(b, &s.all)
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if s.all == nil {
		s.all = map[string]string{}
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

func (s *Store) Get(login string) string {
	if s == nil {
		return ""
	}
	login = strings.TrimSpace(login)
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.all[login]
}

func (s *Store) Set(login, password string) error {
	if s == nil {
		return nil
	}
	login = strings.TrimSpace(login)
	if login == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if password == "" {
		delete(s.all, login)
	} else {
		s.all[login] = password
	}
	return s.persist()
}

func Random() (string, error) {
	var raw [12]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}
