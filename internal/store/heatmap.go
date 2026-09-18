package store

import (
	"context"
	"fmt"
	"time"
)

func (s *Store) DailyCounts(repos []string, since, until int64) (map[string]int64, error) {
	out := map[string]int64{}
	if !s.ready() {
		return out, nil
	}
	if repos != nil && len(repos) == 0 {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	q := `
SELECT to_char((to_timestamp(created) AT TIME ZONE 'UTC')::date, 'YYYY-MM-DD'), COUNT(*)
FROM pipelines
WHERE created >= $1 AND created < $2`
	args := []any{since, until}
	if repos != nil {
		args = append(args, repos)
		q += fmt.Sprintf(` AND repo = ANY($%d)`, len(args))
	}
	q += ` GROUP BY 1`
	rows, err := s.pool.Query(ctx, q, args...)
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
