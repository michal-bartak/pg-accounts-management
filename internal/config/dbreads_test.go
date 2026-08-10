package config

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/michalbartak/dbaccounts/internal/model"
)

func TestDefaultDBReads_contract(t *testing.T) {
	reads := DefaultConfig().DBReads
	// Each default query must be non-empty, reference $1, and SELECT the contract columns
	// (scanned by name downstream — guard against accidental alias drift).
	cases := []struct {
		name  string
		query string
		cols  []string
	}{
		{"search_roles", reads.SearchRoles.Query, []string{"rolname", "comment"}},
		{"role_detail", reads.RoleDetail.Query, []string{
			"rolsuper", "rolcreaterole", "rolcreatedb", "rolinherit",
			"rolcanlogin", "rolreplication", "rolbypassrls", "comment", "rolconfig",
		}},
		{"role_parents", reads.RoleParents.Query, []string{"rolname"}},
		{"role_dependencies", reads.RoleDependencies.Query, []string{
			"database", "dependency", "class", "object",
		}},
	}
	for _, c := range cases {
		if strings.TrimSpace(c.query) == "" {
			t.Fatalf("%s: default query is empty", c.name)
		}
		if !strings.Contains(c.query, "${rolename}") {
			t.Fatalf("%s: default query missing ${rolename}", c.name)
		}
		for _, col := range c.cols {
			if !strings.Contains(c.query, col) {
				t.Fatalf("%s: default query missing contract column %q:\n%s", c.name, col, c.query)
			}
		}
	}
}

func TestMigrateDBReads_fillsBlanksKeepsCustom(t *testing.T) {
	def := DefaultConfig().DBReads
	reads := model.DBReads{
		SearchRoles: model.DBRead{Query: "SELECT rolname, comment FROM my.search($1)"}, // custom, kept
		// RoleDetail / RoleParents left blank → filled from defaults
	}
	migrateDBReads(&reads)
	if reads.SearchRoles.Query != "SELECT rolname, comment FROM my.search($1)" {
		t.Fatalf("custom search query not kept: %s", reads.SearchRoles.Query)
	}
	if reads.RoleDetail.Query != def.RoleDetail.Query {
		t.Fatalf("blank role_detail not filled with default")
	}
	if reads.RoleParents.Query != def.RoleParents.Query {
		t.Fatalf("blank role_parents not filled with default")
	}
	if reads.RoleDependencies.Query != def.RoleDependencies.Query {
		t.Fatalf("blank role_dependencies not filled with default")
	}
}

func TestValidateDBReads(t *testing.T) {
	// Defaults validate.
	if err := validateDBReads(DefaultConfig().DBReads); err != nil {
		t.Fatalf("defaults rejected: %v", err)
	}
	// Empty query rejected.
	bad := DefaultConfig().DBReads
	bad.SearchRoles.Query = "   "
	if err := validateDBReads(bad); err == nil {
		t.Fatal("expected empty query to be rejected")
	}
	// Missing $1 rejected.
	bad = DefaultConfig().DBReads
	bad.RoleDetail.Query = "SELECT rolsuper FROM pg_roles"
	if err := validateDBReads(bad); err == nil {
		t.Fatal("expected missing $1 to be rejected")
	}
	// The pre-flight read is validated like every other one.
	bad = DefaultConfig().DBReads
	bad.RoleDependencies.Query = "SELECT * FROM pg_shdepend"
	if err := validateDBReads(bad); err == nil {
		t.Fatal("expected role_dependencies without ${rolename} to be rejected")
	}
}

func TestUpdateDBReads_blankFallsBackToDefault(t *testing.T) {
	s := &Store{path: filepath.Join(t.TempDir(), "config.yaml"), cfg: DefaultConfig()}
	// A blank query from the editor should fall back to the default rather than fail.
	reads := DefaultConfig().DBReads
	reads.RoleParents.Query = ""
	if err := s.UpdateDBReads(reads); err != nil {
		t.Fatalf("UpdateDBReads rejected blank (should default): %v", err)
	}
	if s.cfg.DBReads.RoleParents.Query != DefaultConfig().DBReads.RoleParents.Query {
		t.Fatalf("blank role_parents not defaulted on update")
	}
}
