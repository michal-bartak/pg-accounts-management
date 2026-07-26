package calltemplate

import (
	"strings"
	"testing"

	"github.com/michalbartak/dbaccounts/internal/model"
)

func TestBuild_statement_dropRole(t *testing.T) {
	sql, vals, useQuery, err := Build(
		"DROP ROLE ${loginname}",
		map[string]string{"loginname": "jdoe"},
		"remove_role",
		model.ExecutionStatement,
	)
	if err != nil {
		t.Fatal(err)
	}
	if useQuery || len(vals) != 0 {
		t.Fatalf("useQuery=%v vals=%v", useQuery, vals)
	}
	if sql != `DROP ROLE "jdoe"` {
		t.Fatalf("got: %s", sql)
	}
}

func TestBuild_statement_setComment(t *testing.T) {
	sql, vals, useQuery, err := Build(
		"COMMENT ON ROLE ${loginname} IS ${comment}",
		map[string]string{"loginname": "jdoe", "comment": `{"full_name":"O'Hara"}`},
		"set_comment",
		model.ExecutionStatement,
	)
	if err != nil {
		t.Fatal(err)
	}
	if useQuery || len(vals) != 0 {
		t.Fatalf("useQuery=%v vals=%v", useQuery, vals)
	}
	// login name double-quoted as identifier; comment embedded as an escaped string literal.
	want := `COMMENT ON ROLE "jdoe" IS '{"full_name":"O''Hara"}'`
	if sql != want {
		t.Fatalf("got:  %s\nwant: %s", sql, want)
	}
}

// A backslash in a comment must be emitted as an E'…' escape string so the literal is safe
// even when the server has standard_conforming_strings off (otherwise a trailing backslash
// could escape the closing quote and swallow following SQL).
func TestBuild_statement_setComment_backslashUsesEString(t *testing.T) {
	cases := []struct {
		comment string
		want    string
	}{
		{`domain\user`, `COMMENT ON ROLE "jdoe" IS E'domain\\user'`},
		{`ends with backslash \`, `COMMENT ON ROLE "jdoe" IS E'ends with backslash \\'`},
		{`quote'and\slash`, `COMMENT ON ROLE "jdoe" IS E'quote''and\\slash'`},
	}
	for _, tc := range cases {
		sql, _, _, err := Build(
			"COMMENT ON ROLE ${loginname} IS ${comment}",
			map[string]string{"loginname": "jdoe", "comment": tc.comment},
			"set_comment",
			model.ExecutionStatement,
		)
		if err != nil {
			t.Fatalf("comment %q: %v", tc.comment, err)
		}
		if sql != tc.want {
			t.Fatalf("comment %q:\ngot:  %s\nwant: %s", tc.comment, sql, tc.want)
		}
	}
}

func TestBuild_statement_setComment_empty(t *testing.T) {
	sql, _, _, err := Build(
		"COMMENT ON ROLE ${loginname} IS ${comment}",
		map[string]string{"loginname": "jdoe", "comment": ""},
		"set_comment",
		model.ExecutionStatement,
	)
	if err != nil {
		t.Fatal(err)
	}
	if sql != `COMMENT ON ROLE "jdoe" IS ''` {
		t.Fatalf("got: %s", sql)
	}
}

func TestBuild_statement_setAttribute(t *testing.T) {
	// ${attribute} (singular) is kept as a backward-compat alias for ${attributes}.
	sql, _, _, err := Build(
		"ALTER ROLE ${loginname} WITH ${attribute}",
		map[string]string{"loginname": "jdoe", "attribute": "NOSUPERUSER", "attributes": "NOSUPERUSER"},
		"set_attribute",
		model.ExecutionStatement,
	)
	if err != nil {
		t.Fatal(err)
	}
	// Login is double-quoted; the attribute keyword is embedded unquoted.
	if sql != `ALTER ROLE "jdoe" WITH NOSUPERUSER` {
		t.Fatalf("got: %s", sql)
	}
}

func TestBuild_statement_setAttributes_plural(t *testing.T) {
	sql, _, _, err := Build(
		"ALTER ROLE ${loginname} WITH ${attributes}",
		map[string]string{"loginname": "jdoe", "attributes": "NOSUPERUSER NOLOGIN", "attribute": "NOSUPERUSER NOLOGIN"},
		"set_attribute",
		model.ExecutionStatement,
	)
	if err != nil {
		t.Fatal(err)
	}
	if sql != `ALTER ROLE "jdoe" WITH NOSUPERUSER NOLOGIN` {
		t.Fatalf("got: %s", sql)
	}
}

func TestBuild_statement_setAttribute_multipleKeywords(t *testing.T) {
	sql, _, _, err := Build(
		"ALTER ROLE ${loginname} WITH ${attribute}",
		map[string]string{"loginname": "jdoe", "attribute": "NOSUPERUSER NOLOGIN CREATEDB"},
		"set_attribute",
		model.ExecutionStatement,
	)
	if err != nil {
		t.Fatal(err)
	}
	if sql != `ALTER ROLE "jdoe" WITH NOSUPERUSER NOLOGIN CREATEDB` {
		t.Fatalf("got: %s", sql)
	}
}

func TestBuild_statement_setConfig(t *testing.T) {
	sql, _, _, err := Build(
		"ALTER ROLE ${loginname} SET ${config_name} = ${config_value}",
		map[string]string{"loginname": "jdoe", "config_name": "search_path", "config_value": "public"},
		"set_config",
		model.ExecutionStatement,
	)
	if err != nil {
		t.Fatal(err)
	}
	// Login double-quoted, GUC name embedded unquoted, value a quoted literal.
	if sql != `ALTER ROLE "jdoe" SET search_path = 'public'` {
		t.Fatalf("got: %s", sql)
	}
}

func TestBuild_statement_setConfig_namespacedName(t *testing.T) {
	sql, _, _, err := Build(
		"ALTER ROLE ${loginname} SET ${config_name} = ${config_value}",
		map[string]string{"loginname": "jdoe", "config_name": "auto_explain.log_min_duration", "config_value": "0"},
		"set_config",
		model.ExecutionStatement,
	)
	if err != nil {
		t.Fatal(err)
	}
	if sql != `ALTER ROLE "jdoe" SET auto_explain.log_min_duration = '0'` {
		t.Fatalf("got: %s", sql)
	}
}

func TestBuild_statement_setConfig_rejectsBadName(t *testing.T) {
	_, _, _, err := Build(
		"ALTER ROLE ${loginname} SET ${config_name} = ${config_value}",
		map[string]string{"loginname": "jdoe", "config_name": "bad name; DROP", "config_value": "x"},
		"set_config",
		model.ExecutionStatement,
	)
	if err == nil {
		t.Fatal("expected error for a non-identifier setting name")
	}
}

func TestBuild_statement_setConfig_valueEscaped(t *testing.T) {
	sql, _, _, err := Build(
		"ALTER ROLE ${loginname} SET ${config_name} = ${config_value}",
		map[string]string{"loginname": "jdoe", "config_name": "search_path", "config_value": `a\b`},
		"set_config",
		model.ExecutionStatement,
	)
	if err != nil {
		t.Fatal(err)
	}
	// A backslash-bearing value uses an E'…' escape string.
	if sql != `ALTER ROLE "jdoe" SET search_path = E'a\\b'` {
		t.Fatalf("got: %s", sql)
	}
}

func TestBuild_statement_resetConfig(t *testing.T) {
	sql, _, _, err := Build(
		"ALTER ROLE ${loginname} RESET ${config_name}",
		map[string]string{"loginname": "jdoe", "config_name": "search_path"},
		"reset_config",
		model.ExecutionStatement,
	)
	if err != nil {
		t.Fatal(err)
	}
	if sql != `ALTER ROLE "jdoe" RESET search_path` {
		t.Fatalf("got: %s", sql)
	}
}

func TestBuild_statement_setAttribute_rejectsBadToken(t *testing.T) {
	_, _, _, err := Build(
		"ALTER ROLE ${loginname} WITH ${attribute}",
		map[string]string{"loginname": "jdoe", "attribute": "SUPERUSER; DROP"},
		"set_attribute",
		model.ExecutionStatement,
	)
	if err == nil {
		t.Fatal("expected error for a non-identifier attribute token")
	}
}

func TestBuild_statement_rolenameAlias(t *testing.T) {
	sql, _, _, err := Build(
		"DROP ROLE ${rolename}",
		map[string]string{"loginname": "testuser", "rolename": "testuser"},
		"remove_role",
		model.ExecutionStatement,
	)
	if err != nil {
		t.Fatal(err)
	}
	if sql != `DROP ROLE "testuser"` {
		t.Fatalf("got: %s", sql)
	}
}

// Block mode runs the user's complete anonymous code block verbatim, only embedding
// placeholder values — the app adds no DO/delimiter wrapper of its own.
func TestBuild_block_runsVerbatim(t *testing.T) {
	call := "DO $do$ BEGIN\n  DROP ROLE ${loginname};\nEND $do$;"
	sql, _, useQuery, err := Build(
		call,
		map[string]string{"loginname": "jdoe"},
		"remove_role",
		model.ExecutionBlock,
	)
	if err != nil {
		t.Fatal(err)
	}
	if useQuery {
		t.Fatal("expected Exec mode")
	}
	// The user's DO wrapper is preserved; only ${loginname} is substituted (as an identifier).
	want := "DO $do$ BEGIN\n  DROP ROLE \"jdoe\";\nEND $do$;"
	if sql != want {
		t.Fatalf("got:  %s\nwant: %s", sql, want)
	}
}

func TestBuild_function_unchanged(t *testing.T) {
	call := "your_schema.remove_app_role(${loginname})"
	sql, vals, useQuery, err := Build(
		call,
		map[string]string{"loginname": "u1"},
		"remove_role",
		model.ExecutionFunction,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !useQuery || len(vals) != 1 {
		t.Fatalf("useQuery=%v vals=%v", useQuery, vals)
	}
	if sql != "SELECT your_schema.remove_app_role($1)" {
		t.Fatalf("got: %s", sql)
	}
}

func TestValidate_rejectsRawDollarParams(t *testing.T) {
	err := ValidateCallTemplateWithExecution("drop_user($1)", "remove_role", model.ExecutionFunction)
	if err == nil || !strings.Contains(err.Error(), "$1") {
		t.Fatalf("got: %v", err)
	}
}

// Block mode now runs a complete user-authored anonymous block, so an outer DO with its own
// delimiter is accepted (the app no longer owns the wrapper).
func TestValidate_block_acceptsOuterDO(t *testing.T) {
	if err := ValidateCallTemplateWithExecution("DO $$ BEGIN NULL; END $$", "remove_role", model.ExecutionBlock); err != nil {
		t.Fatalf("outer DO should be allowed in block mode: %v", err)
	}
}

func TestValidate_statement_allowsDrop(t *testing.T) {
	if err := ValidateCallTemplateWithExecution("DROP ROLE ${loginname}", "remove_role", model.ExecutionStatement); err != nil {
		t.Fatal(err)
	}
}

func TestBuild_statement_grantParents_singleRole(t *testing.T) {
	sql, _, _, err := Build(
		"GRANT ${parent_roles} TO ${loginname}",
		map[string]string{"parent_roles": "Gr_devs_all_ro", "loginname": "test"},
		"grant_parents",
		model.ExecutionStatement,
	)
	if err != nil {
		t.Fatal(err)
	}
	if sql != `GRANT "Gr_devs_all_ro" TO "test"` {
		t.Fatalf("got: %s", sql)
	}
	if strings.Contains(sql, "'") {
		t.Fatalf("role names must not be quoted as literals: %s", sql)
	}
}

func TestBuild_statement_grantParents_multipleRoles(t *testing.T) {
	sql, _, _, err := Build(
		"GRANT ${parent_roles} TO ${loginname}",
		map[string]string{"parent_roles": "gr_a, gr_b", "loginname": "testuser"},
		"grant_parents",
		model.ExecutionStatement,
	)
	if err != nil {
		t.Fatal(err)
	}
	if sql != `GRANT "gr_a", "gr_b" TO "testuser"` {
		t.Fatalf("got: %s", sql)
	}
}

func TestBuild_statement_grantParents_hyphenNowQuoted(t *testing.T) {
	// A hyphenated name is valid once identifiers are double-quoted.
	sql, _, _, err := Build(
		"GRANT ${parent_roles} TO ${loginname}",
		map[string]string{"parent_roles": "bad-role", "loginname": "test"},
		"grant_parents",
		model.ExecutionStatement,
	)
	if err != nil {
		t.Fatal(err)
	}
	if sql != `GRANT "bad-role" TO "test"` {
		t.Fatalf("got: %s", sql)
	}
}

func TestBuild_statement_grantParents_emptyElementRejected(t *testing.T) {
	// An empty list element (trailing comma) is still rejected.
	_, _, _, err := Build(
		"GRANT ${parent_roles} TO ${loginname}",
		map[string]string{"parent_roles": "gr_a,", "loginname": "test"},
		"grant_parents",
		model.ExecutionStatement,
	)
	if err == nil {
		t.Fatal("expected error for empty list element")
	}
}

func TestBuild_statement_revokeParents(t *testing.T) {
	sql, _, _, err := Build(
		"REVOKE ${parent_roles} FROM ${loginname}",
		map[string]string{"parent_roles": "Gr_devs_all_ro", "loginname": "test"},
		"revoke_parents",
		model.ExecutionStatement,
	)
	if err != nil {
		t.Fatal(err)
	}
	if sql != `REVOKE "Gr_devs_all_ro" FROM "test"` {
		t.Fatalf("got: %s", sql)
	}
	if strings.Contains(sql, "'") {
		t.Fatalf("role names must not be single-quoted: %s", sql)
	}
}

// Identifiers are double-quoted, so mixed case is preserved, hyphens/dots are allowed, and an
// embedded double-quote is doubled.
func TestBuild_statement_identifiersQuoted_preserveCaseAndSpecial(t *testing.T) {
	sql, _, _, err := Build(
		"REVOKE ${parent_roles} FROM ${loginname}",
		map[string]string{"parent_roles": `My-Role,db.reader,od"d`, "loginname": "AdminUser"},
		"revoke_parents",
		model.ExecutionStatement,
	)
	if err != nil {
		t.Fatal(err)
	}
	want := `REVOKE "My-Role", "db.reader", "od""d" FROM "AdminUser"`
	if sql != want {
		t.Fatalf("got:  %s\nwant: %s", sql, want)
	}
}

func TestBuild_function_grantParents_stillBinds(t *testing.T) {
	sql, vals, useQuery, err := Build(
		"your_schema.grant_role_parents(${loginname}, ${parent_roles})",
		map[string]string{"loginname": "u1", "parent_roles": "gr_a,gr_b"},
		"grant_parents",
		model.ExecutionFunction,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !useQuery || len(vals) != 2 {
		t.Fatalf("useQuery=%v vals=%v", useQuery, vals)
	}
	if sql != "SELECT your_schema.grant_role_parents($1, $2)" {
		t.Fatalf("got: %s", sql)
	}
}
