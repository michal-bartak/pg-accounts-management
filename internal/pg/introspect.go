package pg

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/jackc/pgx/v5"
)

// RoleRow is one role read from a cluster's catalogs.
type RoleRow struct {
	Name    string
	Comment string
}

// searchRolesSQL matches on role name or the role's COMMENT ON ROLE (pg_shdescription).
const searchRolesSQL = `SELECT r.rolname,
       COALESCE(d.description, '')
FROM pg_roles r
LEFT JOIN pg_shdescription d
  ON d.objoid = r.oid AND d.classoid = 'pg_authid'::regclass
WHERE r.rolname ILIKE $1 OR d.description ILIKE $1
ORDER BY r.rolname`

const roleDetailSQL = `SELECT r.rolsuper, r.rolcreaterole, r.rolcreatedb, r.rolinherit,
       r.rolcanlogin, r.rolreplication, r.rolbypassrls,
       COALESCE(d.description, '')
FROM pg_roles r
LEFT JOIN pg_shdescription d
  ON d.objoid = r.oid AND d.classoid = 'pg_authid'::regclass
WHERE r.rolname = $1`

const roleParentsSQL = `SELECT g.rolname
FROM pg_auth_members m
JOIN pg_roles g ON g.oid = m.roleid
JOIN pg_roles u ON u.oid = m.member
WHERE u.rolname = $1
ORDER BY g.rolname`

// SearchRoles returns roles whose name or comment matches term (case-insensitive substring).
func SearchRoles(ctx context.Context, conn *pgx.Conn, term string) ([]RoleRow, error) {
	rows, err := conn.Query(ctx, searchRolesSQL, likePattern(term))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []RoleRow
	for rows.Next() {
		var r RoleRow
		if err := rows.Scan(&r.Name, &r.Comment); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// RoleDetail reads whether a login exists, its comment, attribute flags, and direct
// parent memberships.
func RoleDetail(ctx context.Context, conn *pgx.Conn, loginName string) (exists bool, comment string, parents []string, attrs map[string]bool, err error) {
	var super, createRole, createDB, inherit, canLogin, replication, bypassRLS bool
	err = conn.QueryRow(ctx, roleDetailSQL, loginName).Scan(
		&super, &createRole, &createDB, &inherit, &canLogin, &replication, &bypassRLS, &comment,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, "", nil, nil, nil
		}
		return false, "", nil, nil, err
	}
	exists = true
	attrs = map[string]bool{
		"super":       super,
		"createrole":  createRole,
		"createdb":    createDB,
		"inherit":     inherit,
		"login":       canLogin,
		"replication": replication,
		"bypassrls":   bypassRLS,
	}

	rows, err := conn.Query(ctx, roleParentsSQL, loginName)
	if err != nil {
		return exists, comment, nil, attrs, err
	}
	defer rows.Close()
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return exists, comment, parents, attrs, err
		}
		parents = append(parents, p)
	}
	return exists, comment, parents, attrs, rows.Err()
}

// ParseFullName extracts full_name from a JSON comment, or "" if the comment is not
// JSON, has no full_name key, or the value is not a non-empty string.
func ParseFullName(comment string) string {
	comment = strings.TrimSpace(comment)
	if comment == "" || comment[0] != '{' {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(comment), &m); err != nil {
		return ""
	}
	if v, ok := m["full_name"].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

// likePattern wraps a search term as a case-insensitive substring ILIKE pattern,
// escaping LIKE wildcards so they are matched literally.
func likePattern(term string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + r.Replace(strings.TrimSpace(term)) + "%"
}
