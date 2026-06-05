package config

import (
	"strings"
	"testing"

	"github.com/michalbartak/dbaccounts/internal/model"
)

func TestMigrateOne_brokenCreateRoleName(t *testing.T) {
	def := DefaultConfig()
	broken := model.DBFunction{
		Name: "admin_access.create_role(${loginname}, NULL)",
	}
	got := migrateOne(broken, def.DBFunctions.CreateRole, "create_role")
	if !strings.Contains(got.Call, "ARRAY['gr_personal_users'") || !strings.Contains(got.Call, "|| ${parent_role}") {
		t.Fatalf("got: %s", got.Call)
	}
}

func TestMigrateOne_legacyNameAndParams(t *testing.T) {
	fn := model.DBFunction{
		Name:   "app.remove",
		Params: []string{"loginname"},
	}
	got := migrateOne(fn, model.DBFunction{}, "remove_role")
	if got.Call != "app.remove(${loginname})" {
		t.Fatalf("got: %s", got.Call)
	}
}

func TestValidateDBFunctions_rejectsInvalidPlaceholder(t *testing.T) {
	fns := DefaultConfig().DBFunctions
	fns.RemoveRole = model.DBFunction{Call: "fn(${not_allowed})"}
	err := validateDBFunctions(fns)
	if err == nil || !strings.Contains(err.Error(), "remove_role") {
		t.Fatalf("got: %v", err)
	}
}

func TestValidateDBFunctions_acceptsDefaults(t *testing.T) {
	if err := validateDBFunctions(DefaultConfig().DBFunctions); err != nil {
		t.Fatal(err)
	}
}

func TestValidateDBFunctions_acceptsRevokeParentsDefault(t *testing.T) {
	fns := DefaultConfig().DBFunctions
	if err := validateDBFunctions(fns); err != nil {
		t.Fatal(err)
	}
	if fns.RevokeParents.Execution != model.ExecutionStatement {
		t.Fatalf("execution: %q", fns.RevokeParents.Execution)
	}
	if !strings.Contains(fns.RevokeParents.Call, "REVOKE") {
		t.Fatalf("call: %s", fns.RevokeParents.Call)
	}
}

func TestValidateDBFunctions_acceptsRemoveRoleStatement(t *testing.T) {
	fns := DefaultConfig().DBFunctions
	fns.RemoveRole = model.DBFunction{
		Call:      "DROP ROLE ${loginname}",
		Execution: model.ExecutionStatement,
	}
	if err := validateDBFunctions(fns); err != nil {
		t.Fatal(err)
	}
}

func TestNormalizeDBFunction_invalidExecution(t *testing.T) {
	fn := normalizeDBFunction(model.DBFunction{Execution: "invalid"})
	if fn.Execution != model.ExecutionFunction {
		t.Fatalf("got %q", fn.Execution)
	}
}

func TestNeedsCreateRoleTemplateFix(t *testing.T) {
	cases := []struct {
		call string
		want bool
	}{
		{"admin_access.create_role(${loginname}, ${Array['x']})", true},
		{"admin_access.create_role(${loginname}, ARRAY['a'] || ${parent_role})", false},
		{"fn(${array_concat:parent_role,a,b})", true},
	}
	for _, c := range cases {
		if got := needsCreateRoleTemplateFix("create_role", c.call); got != c.want {
			t.Fatalf("%q: got %v want %v", c.call, got, c.want)
		}
	}
}
