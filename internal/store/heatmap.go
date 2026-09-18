package store

import (
	"context"
	"time"
)

func (s *Store) ReplaceHeatmap(login string, days map[string]int64) error {
	if !s.ready() || login == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM user_heatmap WHERE login = $1`, login); err != nil {
		return err
	}
	if len(days) > 0 {
		logins := make([]string, 0, len(days))
		dates := make([]string, 0, len(days))
		vals := make([]int64, 0, len(days))
		for day, n := range days {
			if day == "" || n == 0 {
				continue
			}
			logins = append(logins, login)
			dates = append(dates, day)
			vals = append(vals, n)
		}
		if len(dates) > 0 {
			if _, err := tx.Exec(ctx, `
INSERT INTO user_heatmap (login, day, value)
SELECT * FROM unnest($1::text[], $2::date[], $3::bigint[])
`, logins, dates, vals); err != nil {
				return err
			}
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) Heatmap(login string, since, until time.Time) (map[string]int64, error) {
	out := map[string]int64{}
	if !s.ready() || login == "" {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx, `
SELECT to_char(day, 'YYYY-MM-DD'), value
FROM user_heatmap
WHERE login = $1 AND day >= $2::date AND day < $3::date
`, login, since.UTC().Format("2006-01-02"), until.UTC().Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var day string
		var n int64
		if err := rows.Scan(&day, &n); err != nil {
			return nil, err
		}
		out[day] = n
	}
	return out, rows.Err()
}

func (s *Store) HeatmapLogins() ([]string, error) {
	if !s.ready() {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx, `
SELECT DISTINCT login FROM (
  SELECT login FROM team_members
  UNION
  SELECT login FROM repo_collaborators
) x WHERE login <> ''
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var login string
		if err := rows.Scan(&login); err != nil {
			return nil, err
		}
		out = append(out, login)
	}
	return out, rows.Err()
}
