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
CREATE TABLE IF NOT EXISTS repos (
  full_name text PRIMARY KEY,
  default_branch text NOT NULL DEFAULT '',
  description text NOT NULL DEFAULT '',
  updated bigint NOT NULL DEFAULT 0,
  archived boolean NOT NULL DEFAULT false
);
CREATE TABLE IF NOT EXISTS teams (
  name text PRIMARY KEY,
  write_id bigint NOT NULL DEFAULT 0,
  read_id bigint NOT NULL DEFAULT 0,
  admin_id bigint NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS team_repos (
  team text NOT NULL REFERENCES teams(name) ON DELETE CASCADE,
  repo text NOT NULL REFERENCES repos(full_name) ON DELETE CASCADE,
  PRIMARY KEY (team, repo)
);
CREATE TABLE IF NOT EXISTS team_members (
  team text NOT NULL REFERENCES teams(name) ON DELETE CASCADE,
  login text NOT NULL,
  role text NOT NULL DEFAULT 'write',
  PRIMARY KEY (team, login)
);
CREATE INDEX IF NOT EXISTS team_members_login_idx ON team_members (login);
CREATE INDEX IF NOT EXISTS team_repos_repo_idx ON team_repos (repo);
CREATE TABLE IF NOT EXISTS repo_collaborators (
  repo text NOT NULL REFERENCES repos(full_name) ON DELETE CASCADE,
  login text NOT NULL,
  role text NOT NULL DEFAULT 'write',
  PRIMARY KEY (repo, login)
);
CREATE INDEX IF NOT EXISTS repo_collaborators_login_idx ON repo_collaborators (login);
ALTER TABLE repos ADD COLUMN IF NOT EXISTS archived boolean NOT NULL DEFAULT false;
ALTER TABLE team_repos ADD COLUMN IF NOT EXISTS granted boolean NOT NULL DEFAULT false;
CREATE INDEX IF NOT EXISTS team_repos_granted_idx ON team_repos (team) WHERE granted;
UPDATE pipelines SET jobs = (
  SELECT COALESCE(jsonb_agg(x.elem ORDER BY x.ord), '[]'::jsonb)
  FROM (
    SELECT ordinality AS ord,
      CASE
        WHEN jsonb_typeof(elem) = 'object' AND elem ? 'children' THEN
          (elem - 'children') || jsonb_build_object('steps', COALESCE(elem->'steps', elem->'children', '[]'::jsonb))
        ELSE elem
      END AS elem
    FROM jsonb_array_elements(CASE WHEN jsonb_typeof(jobs) = 'array' THEN jobs ELSE '[]'::jsonb END)
      WITH ORDINALITY AS t(elem, ordinality)
  ) x
)
WHERE jsonb_typeof(jobs) = 'array' AND jobs::text LIKE '%"children"%';
`)
	return err
}

func (s *Store) Ready() bool {
	return s != nil && s.pool != nil
}

func (s *Store) ready() bool {
	return s.Ready()
}
