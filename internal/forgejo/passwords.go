package forgejo

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func dbURL(acahtiURL string) string {
	u, err := url.Parse(strings.TrimSpace(acahtiURL))
	if err != nil || u.Host == "" {
		return ""
	}
	u.Path = "/forgejo"
	return u.String()
}

func UserPasswordFlags(ctx context.Context, acahtiURL string) (map[string]bool, error) {
	dsn := dbURL(acahtiURL)
	if dsn == "" {
		return nil, fmt.Errorf("database url required")
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 8*time.Second)
		defer cancel()
	}
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, err
	}
	defer conn.Close(ctx)
	rows, err := conn.Query(ctx, `SELECT name, lower_name, COALESCE(BTRIM(passwd), '') <> '' FROM "user"`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var name, lower string
		var has bool
		if err := rows.Scan(&name, &lower, &has); err != nil {
			return nil, err
		}
		if name != "" {
			out[name] = has
		}
		if lower != "" {
			out[lower] = has
		}
	}
	return out, rows.Err()
}
