package store

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func Open(databaseURL string) (*Store, error) {
	databaseURL = strings.TrimSpace(databaseURL)
	if databaseURL == "" {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := ensureDB(ctx, databaseURL); err != nil {
		return nil, err
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	s := &Store{pool: pool}
	if err := s.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() {
	if s != nil && s.pool != nil {
		s.pool.Close()
	}
}

func ensureDB(ctx context.Context, databaseURL string) error {
	u, err := url.Parse(databaseURL)
	if err != nil {
		return err
	}
	name := strings.TrimPrefix(u.Path, "/")
	if name == "" || name == "postgres" {
		return nil
	}
	admin := *u
	admin.Path = "/postgres"
	conn, err := pgx.Connect(ctx, admin.String())
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	var exists bool
	if err := conn.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)`, name).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}
	ident := pgx.Identifier{name}.Sanitize()
	_, err = conn.Exec(ctx, "CREATE DATABASE "+ident)
	return err
}

func (s *Store) migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS pipelines (
  repo text NOT NULL,
  number bigint NOT NULL,
  status text NOT NULL DEFAULT '',
  event text NOT NULL DEFAULT '',
  branch text NOT NULL DEFAULT '',
  ref text NOT NULL DEFAULT '',
  title text NOT NULL DEFAULT '',
  message text NOT NULL DEFAULT '',
  author text NOT NULL DEFAULT '',
  avatar text NOT NULL DEFAULT '',
  commit text NOT NULL DEFAULT '',
  error text NOT NULL DEFAULT '',
  created bigint NOT NULL DEFAULT 0,
  started bigint NOT NULL DEFAULT 0,
  finished bigint NOT NULL DEFAULT 0,
  jobs jsonb NOT NULL DEFAULT '[]',
  PRIMARY KEY (repo, number)
);
CREATE INDEX IF NOT EXISTS pipelines_created_idx ON pipelines (created DESC, number DESC);
CREATE INDEX IF NOT EXISTS pipelines_status_created_idx ON pipelines (status, created DESC);
`)
	return err
}

func (s *Store) ready() bool {
	return s != nil && s.pool != nil
}
