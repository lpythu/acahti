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

const pipeSelect = `repo, number, status, event, branch, ref, title, message, author, avatar, commit, error,
created, started, finished, jobs`

func appendFilter(b *strings.Builder, args *[]any, f Filter) {
	if f.Repos != nil {
		*args = append(*args, f.Repos)
		fmt.Fprintf(b, ` AND repo = ANY($%d)`, len(*args))
	}
	if len(f.Status) > 0 {
		*args = append(*args, f.Status)
		fmt.Fprintf(b, ` AND status = ANY($%d)`, len(*args))
	}
	if sha := strings.TrimSpace(f.SHA); sha != "" {
		*args = append(*args, strings.ToLower(sha)+"%")
		fmt.Fprintf(b, ` AND lower(commit) LIKE $%d`, len(*args))
	}
	if br := strings.TrimSpace(f.Branch); br != "" {
		*args = append(*args, br)
		fmt.Fprintf(b, ` AND branch = $%d`, len(*args))
	}
}

func (s *Store) queryPipes(sql string, args []any) ([]woodpecker.Pipeline, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx, sql, args...)
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
	fmt.Fprintf(&b, `SELECT %s FROM pipelines WHERE 1=1`, pipeSelect)
	appendFilter(&b, &args, f)
	b.WriteString(` ORDER BY created DESC, number DESC`)
	args = append(args, q.LimitPlus(), (q.Page-1)*q.Size)
	fmt.Fprintf(&b, ` LIMIT $%d OFFSET $%d`, len(args)-1, len(args))
	items, err := s.queryPipes(b.String(), args)
	if err != nil {
		return page.Result[woodpecker.Pipeline]{}, err
	}
	return page.Clip(items, q), nil
}

// LatestByRepo returns the newest pipeline for each repo (by number), optionally
// filtered by status, already paginated.
func (s *Store) LatestByRepo(repos, status []string, q page.Query) (page.Result[woodpecker.Pipeline], error) {
	if !s.ready() {
		return page.Of([]woodpecker.Pipeline{}, q, false), nil
	}
	if repos != nil && len(repos) == 0 {
		return page.Of([]woodpecker.Pipeline{}, q, false), nil
	}
	q = q.Norm()
	var b strings.Builder
	args := []any{}
	fmt.Fprintf(&b, `SELECT %s FROM (SELECT DISTINCT ON (repo) %s FROM pipelines WHERE 1=1`, pipeSelect, pipeSelect)
	if repos != nil {
		args = append(args, repos)
		fmt.Fprintf(&b, ` AND repo = ANY($%d)`, len(args))
	}
	b.WriteString(` ORDER BY repo, number DESC) latest WHERE 1=1`)
	if len(status) > 0 {
		args = append(args, status)
		fmt.Fprintf(&b, ` AND status = ANY($%d)`, len(args))
	}
	b.WriteString(` ORDER BY created DESC, number DESC`)
	args = append(args, q.LimitPlus(), (q.Page-1)*q.Size)
	fmt.Fprintf(&b, ` LIMIT $%d OFFSET $%d`, len(args)-1, len(args))
	items, err := s.queryPipes(b.String(), args)
	if err != nil {
		return page.Result[woodpecker.Pipeline]{}, err
	}
	return page.Clip(items, q), nil
}

// LatestStatusByCommit maps repo+"\n"+lower(sha) to the newest pipeline status
// whose commit starts with that sha. Missing rows are omitted.
func (s *Store) LatestStatusByCommit(repos, shas []string) (map[string]string, error) {
	out := map[string]string{}
	if !s.ready() || len(repos) == 0 || len(shas) == 0 {
		return out, nil
	}
	n := min(len(repos), len(shas))
	repos, shas = repos[:n], shas[:n]
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx, `
SELECT DISTINCT ON (w.repo, w.sha) w.repo, w.sha, COALESCE(p.status, '')
FROM unnest($1::text[], $2::text[]) AS w(repo, sha)
LEFT JOIN pipelines p ON p.repo = w.repo AND lower(p.commit) LIKE lower(w.sha) || '%'
ORDER BY w.repo, w.sha, p.number DESC NULLS LAST
`, repos, shas)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var repo, sha, status string
		if err := rows.Scan(&repo, &sha, &status); err != nil {
			return nil, err
		}
		if status == "" {
			continue
		}
		out[repo+"\n"+strings.ToLower(sha)] = status
	}
	return out, rows.Err()
}

func (s *Store) DeletePipeline(repo string, number int64) error {
	if !s.ready() || repo == "" || number == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx, `DELETE FROM pipelines WHERE repo = $1 AND number = $2`, repo, number)
	return err
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
