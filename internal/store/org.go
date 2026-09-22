package store

import (
	"context"
	"errors"
	"time"

	"acahti/internal/page"
	"github.com/jackc/pgx/v5"
)

type OrgTeam struct {
	Name    string
	WriteID int64
	ReadID  int64
	AdminID int64
}

type OrgRepo struct {
	FullName      string
	DefaultBranch string
	Description   string
	Updated       int64
	Archived      bool
}

type OrgMember struct {
	Team  string
	Login string
	Role  string
}

type TeamRepoLink struct {
	Team    string
	Repo    string
	Granted bool
}

type UserRepoACL struct {
	Login  string
	Repo   string
	Role   string
	Team   string
	Direct bool
}

type OrgCollaborator struct {
	Repo  string
	Login string
	Role  string
}

func strongerRole(a, b string) string {
	rank := func(role string) int {
		switch role {
		case "admin":
			return 3
		case "write":
			return 2
		case "read":
			return 1
		}
		return 0
	}
	if rank(b) > rank(a) {
		return b
	}
	if a == "" {
		return b
	}
	return a
}

type NavTeam struct {
	Team  string
	Repos []OrgRepo
}

func (s *Store) UpsertRepo(r OrgRepo) error {
	if !s.ready() || r.FullName == "" {
		return nil
	}
	if r.Updated == 0 {
		r.Updated = time.Now().Unix()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx, `
INSERT INTO repos (full_name, default_branch, description, updated, archived)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (full_name) DO UPDATE SET
  default_branch = EXCLUDED.default_branch,
  description = EXCLUDED.description,
  updated = EXCLUDED.updated,
  archived = EXCLUDED.archived
`, r.FullName, r.DefaultBranch, r.Description, r.Updated, r.Archived)
	return err
}

func (s *Store) DeleteRepo(fullName string) error {
	if !s.ready() || fullName == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx, `DELETE FROM repos WHERE full_name = $1`, fullName)
	return err
}

func (s *Store) UpsertTeam(t OrgTeam) error {
	if !s.ready() || t.Name == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx, `
INSERT INTO teams (name, write_id, read_id, admin_id)
VALUES ($1, $2, $3, $4)
ON CONFLICT (name) DO UPDATE SET
  write_id = EXCLUDED.write_id,
  read_id = EXCLUDED.read_id,
  admin_id = EXCLUDED.admin_id
`, t.Name, t.WriteID, t.ReadID, t.AdminID)
	return err
}

func (s *Store) DeleteTeam(name string) error {
	if !s.ready() || name == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx, `DELETE FROM teams WHERE name = $1`, name)
	return err
}

func (s *Store) GetTeam(name string) (OrgTeam, bool, error) {
	if !s.ready() || name == "" {
		return OrgTeam{}, false, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var t OrgTeam
	err := s.pool.QueryRow(ctx, `SELECT name, write_id, read_id, admin_id FROM teams WHERE name = $1`, name).
		Scan(&t.Name, &t.WriteID, &t.ReadID, &t.AdminID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return OrgTeam{}, false, nil
		}
		return OrgTeam{}, false, err
	}
	return t, true, nil
}

func (s *Store) SetTeamRepo(team, repo string) error {
	if !s.ready() || team == "" || repo == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx, `
INSERT INTO team_repos (team, repo) VALUES ($1, $2)
ON CONFLICT (team, repo) DO NOTHING
`, team, repo)
	return err
}

func (s *Store) SetTeamRepoGranted(team, repo string, granted bool) error {
	if !s.ready() || team == "" || repo == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx, `
INSERT INTO team_repos (team, repo, granted) VALUES ($1, $2, $3)
ON CONFLICT (team, repo) DO UPDATE SET granted = EXCLUDED.granted
`, team, repo, granted)
	return err
}

func (s *Store) RemoveTeamRepo(team, repo string) error {
	if !s.ready() || team == "" || repo == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx, `DELETE FROM team_repos WHERE team = $1 AND repo = $2`, team, repo)
	return err
}

func (s *Store) SetTeamMember(team, login, role string) error {
	if !s.ready() || team == "" || login == "" {
		return nil
	}
	if role == "" {
		role = "write"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx, `
INSERT INTO team_members (team, login, role) VALUES ($1, $2, $3)
ON CONFLICT (team, login) DO UPDATE SET role = EXCLUDED.role
`, team, login, role)
	return err
}

func (s *Store) SetRepoCollaborator(repo, login, role string) error {
	if !s.ready() || repo == "" || login == "" {
		return nil
	}
	if role == "" {
		role = "write"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx, `
INSERT INTO repo_collaborators (repo, login, role) VALUES ($1, $2, $3)
ON CONFLICT (repo, login) DO UPDATE SET role = EXCLUDED.role
`, repo, login, role)
	return err
}

func (s *Store) RemoveRepoCollaborator(repo, login string) error {
	if !s.ready() || repo == "" || login == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx, `DELETE FROM repo_collaborators WHERE repo = $1 AND login = $2`, repo, login)
	return err
}

func (s *Store) RemoveTeamMember(team, login string) error {
	if !s.ready() || team == "" || login == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx, `DELETE FROM team_members WHERE team = $1 AND login = $2`, team, login)
	return err
}

func (s *Store) HasMember(team, login string) (bool, error) {
	if !s.ready() || team == "" || login == "" {
		return false, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var ok bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM team_members WHERE team = $1 AND login = $2)`, team, login).Scan(&ok)
	return ok, err
}

func (s *Store) UserRepoPerms(login string) (map[string]string, error) {
	rows, err := s.UserRepoPermsMany([]string{login})
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, row := range rows {
		out[row.Repo] = strongerRole(out[row.Repo], row.Role)
	}
	return out, nil
}

func (s *Store) UserRepoPermsMany(logins []string) ([]UserRepoACL, error) {
	if !s.ready() || len(logins) == 0 {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx, `
SELECT x.login, x.repo, x.role, x.team, x.direct FROM (
  SELECT m.login, tr.repo, m.role, tr.team, false AS direct
  FROM team_members m
  JOIN team_repos tr ON tr.team = m.team AND tr.granted
  WHERE m.login = ANY($1)
  UNION ALL
  SELECT c.login, c.repo, c.role, COALESCE(tr.team, ''), true
  FROM repo_collaborators c
  LEFT JOIN team_repos tr ON tr.repo = c.repo AND tr.granted
  WHERE c.login = ANY($1)
) x
JOIN repos r ON r.full_name = x.repo AND NOT r.archived
`, logins)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UserRepoACL
	for rows.Next() {
		var row UserRepoACL
		if err := rows.Scan(&row.Login, &row.Repo, &row.Role, &row.Team, &row.Direct); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Store) HasVisibleRepo(login, repo string) (bool, error) {
	if !s.ready() || login == "" || repo == "" {
		return false, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var ok bool
	err := s.pool.QueryRow(ctx, `
SELECT EXISTS(
  SELECT 1 FROM team_repos tr
  JOIN team_members m ON m.team = tr.team AND m.login = $1
  WHERE tr.repo = $2 AND tr.granted
  UNION ALL
  SELECT 1 FROM repo_collaborators c
  WHERE c.login = $1 AND c.repo = $2
)`, login, repo).Scan(&ok)
	return ok, err
}

func (s *Store) TeamMembers(team string) ([]OrgMember, error) {
	if !s.ready() || team == "" {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx, `SELECT team, login, role FROM team_members WHERE team = $1 ORDER BY login`, team)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []OrgMember
	for rows.Next() {
		var m OrgMember
		if err := rows.Scan(&m.Team, &m.Login, &m.Role); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) TeamsByLogin() (map[string][]string, error) {
	if !s.ready() {
		return map[string][]string{}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx, `SELECT login, team FROM team_members ORDER BY login, team`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]string{}
	for rows.Next() {
		var login, team string
		if err := rows.Scan(&login, &team); err != nil {
			return nil, err
		}
		out[login] = append(out[login], team)
	}
	return out, rows.Err()
}

func (s *Store) RepoTeamInfo(fullName string) (string, bool, bool, error) {
	if !s.ready() || fullName == "" {
		return "", false, false, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var team string
	var granted bool
	err := s.pool.QueryRow(ctx, `SELECT team, granted FROM team_repos WHERE repo = $1 ORDER BY team LIMIT 1`, fullName).Scan(&team, &granted)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, false, nil
		}
		return "", false, false, err
	}
	return team, granted, true, nil
}

func (s *Store) RepoTeam(fullName string) (string, bool, error) {
	team, _, ok, err := s.RepoTeamInfo(fullName)
	return team, ok, err
}

func (s *Store) ListTeamRepoLinks() ([]TeamRepoLink, error) {
	if !s.ready() {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx, `SELECT team, repo, granted FROM team_repos ORDER BY team, repo`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TeamRepoLink
	for rows.Next() {
		var l TeamRepoLink
		if err := rows.Scan(&l.Team, &l.Repo, &l.Granted); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (s *Store) TeamRepos(team string) ([]OrgRepo, error) {
	return s.teamRepos(team, false)
}

func (s *Store) GrantedTeamRepos(team string) ([]OrgRepo, error) {
	return s.teamRepos(team, true)
}

func (s *Store) teamRepos(team string, grantedOnly bool) ([]OrgRepo, error) {
	if !s.ready() || team == "" {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	q := `
SELECT r.full_name, r.default_branch, r.description, r.updated
FROM team_repos tr
JOIN repos r ON r.full_name = tr.repo
WHERE tr.team = $1
`
	if grantedOnly {
		q += ` AND tr.granted`
	}
	q += ` ORDER BY r.full_name`
	rows, err := s.pool.Query(ctx, q, team)
	if err != nil {
		return nil, err
	}
	return scanRepos(rows)
}

func (s *Store) TeamRepoNames(team string) ([]string, error) {
	repos, err := s.TeamRepos(team)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(repos))
	for _, r := range repos {
		out = append(out, r.FullName)
	}
	return out, nil
}

func (s *Store) VisibleTeams(login string, admin bool) ([]OrgTeam, error) {
	if !s.ready() {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var (
		rows interface {
			Close()
			Next() bool
			Scan(dest ...any) error
			Err() error
		}
		err error
	)
	if admin {
		rows, err = s.pool.Query(ctx, `SELECT name, write_id, read_id, admin_id FROM teams ORDER BY name`)
	} else if login == "" {
		return nil, nil
	} else {
		rows, err = s.pool.Query(ctx, `
SELECT t.name, t.write_id, t.read_id, t.admin_id
FROM teams t
WHERE EXISTS (SELECT 1 FROM team_members m WHERE m.team = t.name AND m.login = $1)
   OR EXISTS (
     SELECT 1 FROM team_repos tr
     JOIN repo_collaborators c ON c.repo = tr.repo AND c.login = $1
     WHERE tr.team = t.name
   )
ORDER BY t.name
`, login)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []OrgTeam
	for rows.Next() {
		var t OrgTeam
		if err := rows.Scan(&t.Name, &t.WriteID, &t.ReadID, &t.AdminID); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) VisibleRepos(login string, admin bool) ([]OrgRepo, error) {
	if !s.ready() {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if admin {
		rows, err := s.pool.Query(ctx, `SELECT full_name, default_branch, description, updated FROM repos WHERE NOT archived ORDER BY updated DESC, full_name`)
		if err != nil {
			return nil, err
		}
		return scanRepos(rows)
	}
	if login == "" {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx, `
SELECT DISTINCT r.full_name, r.default_branch, r.description, r.updated
FROM repos r
WHERE NOT r.archived AND (
  EXISTS (
    SELECT 1 FROM team_repos tr
    JOIN team_members m ON m.team = tr.team AND m.login = $1
    WHERE tr.repo = r.full_name AND tr.granted
  )
  OR EXISTS (
    SELECT 1 FROM repo_collaborators c
    WHERE c.repo = r.full_name AND c.login = $1
  )
)
ORDER BY r.updated DESC, r.full_name
`, login)
	if err != nil {
		return nil, err
	}
	return scanRepos(rows)
}

func (s *Store) VisibleTeamRepos(login, team string, admin bool) ([]OrgRepo, error) {
	if !s.ready() || team == "" {
		return nil, nil
	}
	if admin {
		return s.TeamRepos(team)
	}
	if login == "" {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx, `
SELECT r.full_name, r.default_branch, r.description, r.updated
FROM team_repos tr
JOIN repos r ON r.full_name = tr.repo AND NOT r.archived
WHERE tr.team = $2 AND (
  (tr.granted AND EXISTS (SELECT 1 FROM team_members m WHERE m.team = $2 AND m.login = $1))
  OR EXISTS (SELECT 1 FROM repo_collaborators c WHERE c.repo = r.full_name AND c.login = $1)
)
ORDER BY r.full_name
`, login, team)
	if err != nil {
		return nil, err
	}
	return scanRepos(rows)
}

func (s *Store) VisibleRepoNames(login string, admin bool) ([]string, error) {
	if admin {
		return nil, nil
	}
	repos, err := s.VisibleRepos(login, false)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(repos))
	for _, r := range repos {
		out = append(out, r.FullName)
	}
	return out, nil
}

func (s *Store) TeamCounts(login string, admin bool) ([]RepoTeamCount, error) {
	if !s.ready() {
		return nil, nil
	}
	if admin {
		teams, err := s.VisibleTeams(login, true)
		if err != nil {
			return nil, err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		rows, err := s.pool.Query(ctx, `SELECT team, COUNT(*) FROM team_repos GROUP BY team`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		n := map[string]int{}
		for rows.Next() {
			var team string
			var c int
			if err := rows.Scan(&team, &c); err != nil {
				return nil, err
			}
			n[team] = c
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		out := make([]RepoTeamCount, 0, len(teams))
		for _, t := range teams {
			out = append(out, RepoTeamCount{Team: t.Name, Count: n[t.Name]})
		}
		return out, nil
	}
	if login == "" {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := s.pool.Query(ctx, `
SELECT team, COUNT(*) FROM (
  SELECT tr.team, r.full_name
  FROM team_repos tr
  JOIN repos r ON r.full_name = tr.repo AND NOT r.archived
  JOIN team_members m ON m.team = tr.team AND m.login = $1
  WHERE tr.granted
  UNION
  SELECT tr.team, r.full_name
  FROM repo_collaborators c
  JOIN repos r ON r.full_name = c.repo AND NOT r.archived
  JOIN team_repos tr ON tr.repo = r.full_name
  WHERE c.login = $1
) x
GROUP BY team
ORDER BY team
`, login)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RepoTeamCount
	for rows.Next() {
		var row RepoTeamCount
		if err := rows.Scan(&row.Team, &row.Count); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

type RepoTeamCount struct {
	Team  string
	Count int
}

func (s *Store) NavTree(login string, admin bool) ([]NavTeam, error) {
	if !s.ready() {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	var (
		q    string
		args []any
	)
	if admin {
		q = `
SELECT t.name, COALESCE(r.full_name, ''), COALESCE(r.default_branch, ''), COALESCE(r.description, ''), COALESCE(r.updated, 0)
FROM teams t
LEFT JOIN team_repos tr ON tr.team = t.name
LEFT JOIN repos r ON r.full_name = tr.repo AND NOT r.archived
ORDER BY t.name, r.full_name`
	} else if login == "" {
		return nil, nil
	} else {
		q = `
SELECT t.name, COALESCE(r.full_name, ''), COALESCE(r.default_branch, ''), COALESCE(r.description, ''), COALESCE(r.updated, 0)
FROM teams t
JOIN team_members m ON m.team = t.name AND m.login = $1
LEFT JOIN team_repos tr ON tr.team = t.name
LEFT JOIN repos r ON r.full_name = tr.repo AND NOT r.archived AND (
  tr.granted
  OR EXISTS (SELECT 1 FROM repo_collaborators c WHERE c.repo = r.full_name AND c.login = $1)
)
UNION
SELECT tr.team, r.full_name, COALESCE(r.default_branch, ''), COALESCE(r.description, ''), COALESCE(r.updated, 0)
FROM repo_collaborators c
JOIN repos r ON r.full_name = c.repo AND NOT r.archived
JOIN team_repos tr ON tr.repo = r.full_name
WHERE c.login = $1
  AND NOT EXISTS (SELECT 1 FROM team_members m WHERE m.team = tr.team AND m.login = $1)
ORDER BY 1, 2`
		args = []any{login}
	}
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []NavTeam
	idx := map[string]int{}
	for rows.Next() {
		var team, full, branch, desc string
		var updated int64
		if err := rows.Scan(&team, &full, &branch, &desc, &updated); err != nil {
			return nil, err
		}
		i, ok := idx[team]
		if !ok {
			i = len(out)
			idx[team] = i
			out = append(out, NavTeam{Team: team})
		}
		if full != "" {
			out[i].Repos = append(out[i].Repos, OrgRepo{FullName: full, DefaultBranch: branch, Description: desc, Updated: updated})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if admin {
		extra, err := s.unassignedRepos(ctx)
		if err != nil {
			return nil, err
		}
		out = appendUnassigned(out, extra)
	} else {
		extra, err := s.collabUnassigned(ctx, login)
		if err != nil {
			return nil, err
		}
		out = appendUnassigned(out, extra)
	}
	return out, nil
}

func (s *Store) unassignedRepos(ctx context.Context) ([]OrgRepo, error) {
	rows, err := s.pool.Query(ctx, `
SELECT r.full_name, r.default_branch, r.description, r.updated
FROM repos r
LEFT JOIN team_repos tr ON tr.repo = r.full_name
WHERE tr.repo IS NULL AND NOT r.archived
ORDER BY r.full_name
`)
	if err != nil {
		return nil, err
	}
	return scanRepos(rows)
}

func (s *Store) collabUnassigned(ctx context.Context, login string) ([]OrgRepo, error) {
	rows, err := s.pool.Query(ctx, `
SELECT r.full_name, r.default_branch, r.description, r.updated
FROM repo_collaborators c
JOIN repos r ON r.full_name = c.repo AND NOT r.archived
LEFT JOIN team_repos tr ON tr.repo = r.full_name
WHERE c.login = $1 AND tr.repo IS NULL
ORDER BY r.full_name
`, login)
	if err != nil {
		return nil, err
	}
	return scanRepos(rows)
}

func appendUnassigned(out []NavTeam, repos []OrgRepo) []NavTeam {
	if len(repos) == 0 {
		return out
	}
	return append(out, NavTeam{Team: "", Repos: repos})
}

func (s *Store) ListReposPage(login, team string, admin bool, q page.Query) (page.Result[OrgRepo], error) {
	var (
		repos []OrgRepo
		err   error
	)
	if team != "" {
		repos, err = s.VisibleTeamRepos(login, team, admin)
	} else {
		repos, err = s.VisibleRepos(login, admin)
	}
	if err != nil {
		return page.Result[OrgRepo]{}, err
	}
	return page.Take(repos, q), nil
}

func (s *Store) ReplaceOrg(teams []OrgTeam, repos []OrgRepo, links []TeamRepoLink, members []OrgMember, collabs []OrgCollaborator) error {
	if !s.ready() {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, r := range repos {
		if r.FullName == "" {
			continue
		}
		if r.Updated == 0 {
			r.Updated = time.Now().Unix()
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO repos (full_name, default_branch, description, updated, archived)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (full_name) DO UPDATE SET
  default_branch = EXCLUDED.default_branch,
  description = EXCLUDED.description,
  updated = EXCLUDED.updated,
  archived = EXCLUDED.archived
`, r.FullName, r.DefaultBranch, r.Description, r.Updated, r.Archived); err != nil {
			return err
		}
	}
	for _, t := range teams {
		if t.Name == "" {
			continue
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO teams (name, write_id, read_id, admin_id)
VALUES ($1, $2, $3, $4)
ON CONFLICT (name) DO UPDATE SET
  write_id = EXCLUDED.write_id,
  read_id = EXCLUDED.read_id,
  admin_id = EXCLUDED.admin_id
`, t.Name, t.WriteID, t.ReadID, t.AdminID); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM team_repos`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM team_members`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM repo_collaborators`); err != nil {
		return err
	}
	for _, l := range links {
		if l.Team == "" || l.Repo == "" {
			continue
		}
		if _, err := tx.Exec(ctx, `INSERT INTO team_repos (team, repo, granted) VALUES ($1, $2, $3) ON CONFLICT (team, repo) DO UPDATE SET granted = EXCLUDED.granted`, l.Team, l.Repo, l.Granted); err != nil {
			return err
		}
	}
	for _, m := range members {
		if m.Team == "" || m.Login == "" {
			continue
		}
		if m.Role == "" {
			m.Role = "write"
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO team_members (team, login, role) VALUES ($1, $2, $3)
ON CONFLICT (team, login) DO UPDATE SET role = EXCLUDED.role
`, m.Team, m.Login, m.Role); err != nil {
			return err
		}
	}
	for _, c := range collabs {
		if c.Repo == "" || c.Login == "" {
			continue
		}
		if c.Role == "" {
			c.Role = "write"
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO repo_collaborators (repo, login, role) VALUES ($1, $2, $3)
ON CONFLICT (repo, login) DO UPDATE SET role = EXCLUDED.role
`, c.Repo, c.Login, c.Role); err != nil {
			return err
		}
	}
	names := make([]string, 0, len(teams))
	for _, t := range teams {
		if t.Name != "" {
			names = append(names, t.Name)
		}
	}
	if len(names) == 0 {
		if _, err := tx.Exec(ctx, `DELETE FROM teams`); err != nil {
			return err
		}
	} else if _, err := tx.Exec(ctx, `DELETE FROM teams WHERE NOT (name = ANY($1))`, names); err != nil {
		return err
	}
	full := make([]string, 0, len(repos))
	for _, r := range repos {
		if r.FullName != "" {
			full = append(full, r.FullName)
		}
	}
	if len(full) == 0 {
		if _, err := tx.Exec(ctx, `DELETE FROM repos`); err != nil {
			return err
		}
	} else if _, err := tx.Exec(ctx, `DELETE FROM repos WHERE NOT (full_name = ANY($1))`, full); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type repoRows interface {
	Close()
	Next() bool
	Scan(dest ...any) error
	Err() error
}

func scanRepos(rows repoRows) ([]OrgRepo, error) {
	defer rows.Close()
	var out []OrgRepo
	for rows.Next() {
		var r OrgRepo
		if err := rows.Scan(&r.FullName, &r.DefaultBranch, &r.Description, &r.Updated); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
