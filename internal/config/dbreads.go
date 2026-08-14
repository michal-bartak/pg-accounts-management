package config

import (
	"fmt"
	"strings"

	"github.com/michal-bartak/pgcowboy/internal/model"
)

// Default introspection queries — vanilla PostgreSQL catalog reads. Result columns are
// scanned BY NAME (see internal/pg/introspect.go), so column aliases matter and order does
// not. The single bind is written as the named placeholder ${rolename} (converted to $1 before
// execution) for consistency with the write templates; it carries the search pattern
// (search_roles) or the role name (role_detail/role_parents). A NULL comment / rolconfig is
// fine — the scanner is NULL-safe, so no COALESCE is required.
const defaultSearchRolesQuery = `SELECT r.rolname AS rolname,
       d.description AS comment
FROM pg_roles r
LEFT JOIN pg_shdescription d
  ON d.objoid = r.oid AND d.classoid = 'pg_authid'::regclass
WHERE r.rolname ILIKE ${rolename} OR d.description ILIKE ${rolename}
ORDER BY r.rolname`

const defaultRoleDetailQuery = `SELECT r.rolsuper AS rolsuper,
       r.rolcreaterole AS rolcreaterole,
       r.rolcreatedb AS rolcreatedb,
       r.rolinherit AS rolinherit,
       r.rolcanlogin AS rolcanlogin,
       r.rolreplication AS rolreplication,
       r.rolbypassrls AS rolbypassrls,
       d.description AS comment,
       r.rolconfig AS rolconfig
FROM pg_roles r
LEFT JOIN pg_shdescription d
  ON d.objoid = r.oid AND d.classoid = 'pg_authid'::regclass
WHERE r.rolname = ${rolename}`

const defaultRoleParentsQuery = `SELECT g.rolname AS rolname
FROM pg_auth_members m
JOIN pg_roles g ON g.oid = m.roleid
JOIN pg_roles u ON u.oid = m.member
WHERE u.rolname = ${rolename}
ORDER BY g.rolname`

// defaultRoleDependenciesQuery is the pre-flight check run before a role is dropped: every
// object recorded in pg_shdepend as depending on the role (ownership, ACL entries, default
// privileges, RLS policies, tablespaces). pg_shdepend only describes objects of the CURRENT
// database plus shared ones, so rows from other databases are reported as such rather than
// described. pg_database is LEFT joined so shared dependencies (dbid = 0) are kept.
const defaultRoleDependenciesQuery = `SELECT COALESCE(d.datname, current_database()) AS database,
       CASE s.deptype
           WHEN 'o' THEN 'owner'
           WHEN 'a' THEN 'privileges (ACL)'
           WHEN 'i' THEN 'initial privileges'
           WHEN 'r' THEN 'RLS policy'
           WHEN 't' THEN 'tablespace'
           WHEN 'p' THEN 'pinned (system)'
       END AS dependency,
       s.classid::regclass::TEXT AS class,
       CASE WHEN s.dbid = 0 OR d.datname = current_database()
            THEN pg_describe_object(s.classid, s.objid, s.objsubid)
            ELSE 'Located in other database' END AS object
FROM pg_shdepend s
JOIN pg_authid r ON r.oid = s.refobjid
LEFT JOIN pg_database d ON s.dbid = d.oid
WHERE s.refclassid = 'pg_authid'::regclass
  AND r.rolname = ${rolename}
ORDER BY 1, 2, 3, 4`

// migrateDBReads fills any blank read query with its built-in default, so an older config
// (or a hand-edited one that dropped a key) still has every introspection query. A user's
// non-empty query is kept verbatim.
func migrateDBReads(reads *model.DBReads) {
	def := DefaultConfig().DBReads
	reads.SearchRoles = migrateReadOne(reads.SearchRoles, def.SearchRoles)
	reads.RoleDetail = migrateReadOne(reads.RoleDetail, def.RoleDetail)
	reads.RoleParents = migrateReadOne(reads.RoleParents, def.RoleParents)
	reads.RoleDependencies = migrateReadOne(reads.RoleDependencies, def.RoleDependencies)
}

func migrateReadOne(read, def model.DBRead) model.DBRead {
	if strings.TrimSpace(read.Query) == "" {
		return def
	}
	return read
}

// validateDBReads checks each read query is non-empty and references the bind — written as the
// named placeholder ${rolename} (converted to $1 at execution) or the raw $1 for legacy configs.
// It cannot verify the returned columns without executing the query — that contract is enforced
// at scan time (scan-by-name errors clearly on a mismatch).
func validateDBReads(reads model.DBReads) error {
	checks := []struct {
		op   string
		read model.DBRead
	}{
		{"search_roles", reads.SearchRoles},
		{"role_detail", reads.RoleDetail},
		{"role_parents", reads.RoleParents},
		{"role_dependencies", reads.RoleDependencies},
	}
	for _, c := range checks {
		q := strings.TrimSpace(c.read.Query)
		if q == "" {
			return fmt.Errorf("%s: query is required", c.op)
		}
		if !strings.Contains(q, "${rolename}") && !strings.Contains(q, "$1") {
			return fmt.Errorf("%s: query must reference the ${rolename} parameter", c.op)
		}
	}
	return nil
}
