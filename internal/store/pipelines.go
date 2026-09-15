package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"acahti/internal/page"
	"acahti/internal/woodpecker"
	"github.com/jackc/pgx/v5"
)

type Filter struct {
	Repos  []string
	Status []string
	SHA    string
	Branch string
}

func (s *Store) Upsert(p woodpecker.Pipeline) error {
	if !s.ready() || p.Repo == "" || p.Number == 0 {
		return nil
	}
	jobs, err := json.Marshal(p.Jobs)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = s.pool.Exec(ctx, `
INSERT INTO pipelines (
  repo, number, status, event, branch, ref, title, message, author, avatar, commit, error,
  created, started, finished, jobs
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
)
ON CONFLICT (repo, number) DO UPDATE SET
  status = EXCLUDED.status,
  event = EXCLUDED.event,
  branch = EXCLUDED.branch,
  ref = EXCLUDED.ref,
  title = EXCLUDED.title,
  message = EXCLUDED.message,
  author = EXCLUDED.author,
  avatar = EXCLUDED.avatar,
  commit = EXCLUDED.commit,
  error = EXCLUDED.error,
  created = EXCLUDED.created,
  started = EXCLUDED.started,
  finished = EXCLUDED.finished,
  jobs = EXCLUDED.jobs
`, p.Repo, p.Number, p.Status, p.Event, p.Branch, p.Ref, p.Title, p.Message, p.Author, p.Avatar, p.Commit, p.Error,
		p.Created, p.Started, p.Finished, jobs)
	return err
}

func (s *Store) Get(repo string, number int64) (woodpecker.Pipeline, bool, error) {
	if !s.ready() {
		return woodpecker.Pipeline{}, false, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	p, err := scanPipe(s.pool.QueryRow(ctx, `
SELECT repo, number, status, event, branch, ref, title, message, author, avatar, commit, error,
       created, started, finished, jobs
FROM pipelines WHERE repo = $1 AND number = $2
`, repo, number))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return woodpecker.Pipeline{}, false, nil
		}
		return woodpecker.Pipeline{}, false, err
	}
	return p, true, nil
}

func (s *Store) List(f Filter, q page.Query) (page.Result[woodpecker.Pipeline], error) {
	if !s.ready() {
		return page.Of([]woodpecker.Pipeline{}, q, false), nil
	}
	if f.Repos != nil && len(f.Repos) == 0 {
		return page.Of([]woodpecker.Pipeline{}, q, false), nil
	}
	q = q.Norm()
	var b strings.Builder
	args := []any{}
	b.WriteString(`SELECT repo, number, status, event, branch, ref, title, message, author, avatar, commit, error,
created, started, finished, jobs FROM pipelines WHERE 1=1`)
	if f.Repos != nil {
		args = append(args, f.Repos)
		fmt.Fprintf(&b, ` AND repo = ANY($%d)`, len(args))
	}
	if len(f.Status) > 0 {
		args = append(args, f.Status)
		fmt.Fprintf(&b, ` AND status = ANY($%d)`, len(args))
	}
	if sha := strings.TrimSpace(f.SHA); sha != "" {
		args = append(args, strings.ToLower(sha)+"%")
		fmt.Fprintf(&b, ` AND lower(commit) LIKE $%d`, len(args))
	}
	if br := strings.TrimSpace(f.Branch); br != "" {
		args = append(args, br)
		fmt.Fprintf(&b, ` AND branch = $%d`, len(args))
	}
	b.WriteString(` ORDER BY created DESC, number DESC`)
	args = append(args, q.LimitPlus(), (q.Page-1)*q.Size)
	fmt.Fprintf(&b, ` LIMIT $%d OFFSET $%d`, len(args)-1, len(args))
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx, b.String(), args...)
	if err != nil {
		return page.Result[woodpecker.Pipeline]{}, err
	}
	defer rows.Close()
	var items []woodpecker.Pipeline
	for rows.Next() {
		p, err := scanPipe(rows)
		if err != nil {
			return page.Result[woodpecker.Pipeline]{}, err
		}
		items = append(items, p)
	}
	if err := rows.Err(); err != nil {
		return page.Result[woodpecker.Pipeline]{}, err
	}
	return page.Clip(items, q), nil
}

// LatestByRepo returns the newest pipeline for each repo (by number).
func (s *Store) LatestByRepo(repos []string) ([]woodpecker.Pipeline, error) {
	if !s.ready() {
		return nil, nil
	}
	if repos != nil && len(repos) == 0 {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	var (
		q    string
		args []any
	)
	if repos == nil {
		q = `SELECT DISTINCT ON (repo) repo, number, status, event, branch, ref, title, message, author, avatar, commit, error,
created, started, finished, jobs FROM pipelines ORDER BY repo, number DESC`
	} else {
		args = append(args, repos)
		q = `SELECT DISTINCT ON (repo) repo, number, status, event, branch, ref, title, message, author, avatar, commit, error,
created, started, finished, jobs FROM pipelines WHERE repo = ANY($1) ORDER BY repo, number DESC`
	}
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []woodpecker.Pipeline
	for rows.Next() {
		p, err := scanPipe(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	return items, rows.Err()
}

type row interface {
	Scan(dest ...any) error
}

func scanPipe(r row) (woodpecker.Pipeline, error) {
	var p woodpecker.Pipeline
	var jobs []byte
	err := r.Scan(&p.Repo, &p.Number, &p.Status, &p.Event, &p.Branch, &p.Ref, &p.Title, &p.Message,
		&p.Author, &p.Avatar, &p.Commit, &p.Error, &p.Created, &p.Started, &p.Finished, &jobs)
	if err != nil {
		return woodpecker.Pipeline{}, err
	}
	if len(jobs) > 0 {
		_ = json.Unmarshal(jobs, &p.Jobs)
	}
	return p, nil
}
