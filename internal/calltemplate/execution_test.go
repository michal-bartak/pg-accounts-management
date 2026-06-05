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
	if sql != "DROP ROLE jdoe" {
		t.Fatalf("got: %s", sql)
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
	if sql != "DROP ROLE testuser" {
		t.Fatalf("got: %s", sql)
	}
}

func TestBuild_block_wrapsBody(t *testing.T) {
	sql, _, useQuery, err := Build(
		"DROP ROLE ${loginname};",
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
	if !strings.HasPrefix(sql, "DO $dbaccounts$") || !strings.Contains(sql, "DROP ROLE jdoe") {
		t.Fatalf("got: %s", sql)
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

func TestValidate_block_rejectsOuterDO(t *testing.T) {
	err := ValidateCallTemplateWithExecution("DO $$ BEGIN NULL; END $$", "remove_role", model.ExecutionBlock)
	if err == nil {
		t.Fatal("expected error")
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
	if sql != "GRANT Gr_devs_all_ro TO test" {
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
	if sql != "GRANT gr_a, gr_b TO testuser" {
		t.Fatalf("got: %s", sql)
	}
}

func TestBuild_statement_grantParents_invalidRoleName(t *testing.T) {
	_, _, _, err := Build(
		"GRANT ${parent_roles} TO ${loginname}",
		map[string]string{"parent_roles": "bad-role", "loginname": "test"},
		"grant_parents",
		model.ExecutionStatement,
	)
	if err == nil {
		t.Fatal("expected error for invalid role name")
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
	if sql != "REVOKE Gr_devs_all_ro FROM test" {
		t.Fatalf("got: %s", sql)
	}
	if strings.Contains(sql, "'") {
		t.Fatalf("role names must not be quoted: %s", sql)
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
