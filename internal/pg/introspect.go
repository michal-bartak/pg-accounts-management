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

// The introspection SQL is no longer hardcoded here — it comes from config.DBReads (with
// vanilla catalog defaults) and is passed in by the batch runner. Result columns are scanned
// BY NAME against these structs' `db` tags (pgx RowToStructByNameLax): column order is
// irrelevant, a NULL comment/rolconfig scans cleanly (pointer / nil slice), and an omitted
// contract column leaves its field zero-valued. A returned column with no matching field is
// rejected by pgx with a clear "struct doesn't have corresponding row field" error.

// searchRoleRow is the row contract for the search_roles query.
type searchRoleRow struct {
	Rolname string  `db:"rolname"`
	Comment *string `db:"comment"` // NULL when the role has no COMMENT ON ROLE
}

// roleDetailRow is the single-row contract for the role_detail query.
type roleDetailRow struct {
	Rolsuper       bool     `db:"rolsuper"`
	Rolcreaterole  bool     `db:"rolcreaterole"`
	Rolcreatedb    bool     `db:"rolcreatedb"`
	Rolinherit     bool     `db:"rolinherit"`
	Rolcanlogin    bool     `db:"rolcanlogin"`
	Rolreplication bool     `db:"rolreplication"`
	Rolbypassrls   bool     `db:"rolbypassrls"`
	Comment        *string  `db:"comment"`   // NULL when no comment
	Rolconfig      []string `db:"rolconfig"` // NULL scans into a nil slice
}

// roleParentRow is the row contract for the role_parents query.
type roleParentRow struct {
	Rolname string `db:"rolname"`
}

// roleDependencyRow is the row contract for the role_dependencies query. Every column is a
// pointer: the default query's CASE expressions yield NULL for an unrecognised deptype.
type roleDependencyRow struct {
	Database   *string `db:"database"`
	Dependency *string `db:"dependency"`
	Class      *string `db:"class"`
	Object     *string `db:"object"`
}

// RoleDependency is one object that depends on a role, read from a cluster.
type RoleDependency struct {
	Database   string
	Dependency string
	Class      string
	Object     string
}

// bindRoleName rewrites the named ${rolename} placeholder to pgx's positional $1 bind. The
// value stays a bind (never string-interpolated), so it is injection-safe. A legacy query that
// already uses $1 is left untouched.
func bindRoleName(query string) string {
	return strings.ReplaceAll(query, "${rolename}", "$1")
}

// SearchRoles runs searchQuery (${rolename} = ILIKE pattern) and returns roles whose name or
// comment matches term. searchQuery must return the search_roles contract columns (rolname,
// comment).
func SearchRoles(ctx context.Context, conn *pgx.Conn, searchQuery, term string) ([]RoleRow, error) {
	rows, err := conn.Query(ctx, bindRoleName(searchQuery), likePattern(term))
	if err != nil {
		return nil, err
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByNameLax[searchRoleRow])
	if err != nil {
		return nil, err
	}
	out := make([]RoleRow, 0, len(items))
	for _, it := range items {
		r := RoleRow{Name: it.Rolname}
		if it.Comment != nil {
			r.Comment = *it.Comment
		}
		out = append(out, r)
	}
	return out, nil
}

// inlineRoleName returns query with the bind (${rolename}, or legacy $1) replaced by loginName
// as a quoted literal — for DISPLAY only (never executed).
func inlineRoleName(query, loginName string) string {
	lit := "'" + strings.ReplaceAll(loginName, "'", "''") + "'"
	return strings.ReplaceAll(strings.ReplaceAll(query, "${rolename}", lit), "$1", lit)
}

// SearchRoleQueries returns the SQL that SearchRoles executes for term, with the bind inlined as
// the ILIKE pattern actually sent — for display in the search-status popup (not re-executed).
func SearchRoleQueries(query, term string) []string {
	return []string{inlineRoleName(query, likePattern(term))}
}

// RoleDetailQueries returns the SQL that RoleDetail executes for loginName, with the bind
// inlined — for display in the load-status popup (not re-executed).
func RoleDetailQueries(detailQuery, parentsQuery, loginName string) []string {
	return []string{inlineRoleName(detailQuery, loginName), inlineRoleName(parentsQuery, loginName)}
}

// RoleDependencyQueries returns the SQL that RoleDependencies executes for loginName, with the
// bind inlined — for display in the dependency popup (not re-executed).
func RoleDependencyQueries(query, loginName string) []string {
	return []string{inlineRoleName(query, loginName)}
}

// RoleDependencies runs the pre-flight dependency read for loginName. query (${rolename} = role
// name) must return the role_dependencies contract columns (database, dependency, class, object).
func RoleDependencies(ctx context.Context, conn *pgx.Conn, query, loginName string) ([]RoleDependency, error) {
	rows, err := conn.Query(ctx, bindRoleName(query), loginName)
	if err != nil {
		return nil, err
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByNameLax[roleDependencyRow])
	if err != nil {
		return nil, err
	}
	deref := func(p *string) string {
		if p == nil {
			return ""
		}
		return *p
	}
	out := make([]RoleDependency, 0, len(items))
	for _, it := range items {
		out = append(out, RoleDependency{
			Database:   deref(it.Database),
			Dependency: deref(it.Dependency),
			Class:      deref(it.Class),
			Object:     deref(it.Object),
		})
	}
	return out, nil
}

// RoleDetail reads whether a login exists, its comment, attribute flags, role GUC settings
// (rolconfig), and direct parent memberships. detailQuery ($1 = role name) must return one
// role_detail-contract row; parentsQuery ($1 = role name) returns the role_parents rows.
func RoleDetail(ctx context.Context, conn *pgx.Conn, detailQuery, parentsQuery, loginName string) (exists bool, comment string, parents []string, attrs map[string]bool, settings map[string]string, err error) {
	rows, err := conn.Query(ctx, bindRoleName(detailQuery), loginName)
	if err != nil {
		return false, "", nil, nil, nil, err
	}
	d, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByNameLax[roleDetailRow])
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, "", nil, nil, nil, nil
		}
		return false, "", nil, nil, nil, err
	}
	exists = true
	if d.Comment != nil {
		comment = *d.Comment
	}
	attrs = map[string]bool{
		"super":       d.Rolsuper,
		"createrole":  d.Rolcreaterole,
		"createdb":    d.Rolcreatedb,
		"inherit":     d.Rolinherit,
		"login":       d.Rolcanlogin,
		"replication": d.Rolreplication,
		"bypassrls":   d.Rolbypassrls,
	}
	settings = parseRoleConfig(d.Rolconfig)

	prows, err := conn.Query(ctx, bindRoleName(parentsQuery), loginName)
	if err != nil {
		return exists, comment, nil, attrs, settings, err
	}
	parentRows, err := pgx.CollectRows(prows, pgx.RowToStructByNameLax[roleParentRow])
	if err != nil {
		return exists, comment, nil, attrs, settings, err
	}
	for _, p := range parentRows {
		parents = append(parents, p.Rolname)
	}
	return exists, comment, parents, attrs, settings, nil
}

// parseRoleConfig turns rolconfig entries ("name=value") into a name→value map.
func parseRoleConfig(rolconfig []string) map[string]string {
	out := make(map[string]string, len(rolconfig))
	for _, entry := range rolconfig {
		if i := strings.IndexByte(entry, '='); i > 0 {
			out[entry[:i]] = entry[i+1:]
		}
	}
	return out
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
