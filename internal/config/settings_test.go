package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/michal-bartak/pgcowboy/internal/model"
)

// goodSettings is a payload that validates cleanly, as the baseline for the rejection cases.
func goodSettings() model.SettingsPayload {
	def := DefaultConfig()
	return model.SettingsPayload{
		ParentRoles:   []string{"gr_readonly"},
		CommentFields: []model.CommentField{{Key: "full_name", Label: "Full name"}},
		SearchColumns: []model.SearchColumn{{Label: "Name", Template: "${{full_name}}"}},
		DBFunctions:   def.DBFunctions,
		DBReads:       def.DBReads,
		Batch:         model.BatchSettings{MaxConcurrency: 7},
		UI:            model.UISettings{Theme: model.ThemeDark},
	}
}

func newTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	s := &Store{path: path, cfg: DefaultConfig()}
	if err := s.save(); err != nil {
		t.Fatal(err)
	}
	return s, path
}

func TestSaveSettingsAppliesEverythingAtOnce(t *testing.T) {
	s, path := newTestStore(t)
	if err := s.SaveSettings(goodSettings()); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	reloaded := &Store{path: path}
	if err := reloaded.Load(); err != nil {
		t.Fatal(err)
	}
	cfg := reloaded.Get()
	if len(cfg.ParentRoles) != 1 || cfg.ParentRoles[0] != "gr_readonly" {
		t.Errorf("ParentRoles = %v", cfg.ParentRoles)
	}
	if len(cfg.CommentFields) != 1 || cfg.CommentFields[0].Key != "full_name" {
		t.Errorf("CommentFields = %v", cfg.CommentFields)
	}
	if len(cfg.SearchColumns) != 1 || cfg.SearchColumns[0].Template != "${{full_name}}" {
		t.Errorf("SearchColumns = %v", cfg.SearchColumns)
	}
	if cfg.Batch.MaxConcurrency != 7 {
		t.Errorf("MaxConcurrency = %d, want 7", cfg.Batch.MaxConcurrency)
	}
	if cfg.UI.Theme != model.ThemeDark {
		t.Errorf("Theme = %q, want dark", cfg.UI.Theme)
	}
}

// The point of the atomic save: a payload rejected on ANY field must leave the persisted config
// untouched. The sequential Save* calls it replaced would have written the valid leading fields
// before hitting the bad one.
func TestSaveSettingsRejectsWithoutPartialWrite(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*model.SettingsPayload)
		wantErr string
	}{
		{
			name:    "invalid role parent",
			mutate:  func(p *model.SettingsPayload) { p.ParentRoles = []string{"has space"} },
			wantErr: "invalid role parent",
		},
		{
			name:    "invalid comment field key",
			mutate:  func(p *model.SettingsPayload) { p.CommentFields = []model.CommentField{{Key: "no-dashes"}} },
			wantErr: "invalid comment field key",
		},
		{
			name: "unknown bare placeholder in a search column",
			mutate: func(p *model.SettingsPayload) {
				p.SearchColumns = []model.SearchColumn{{Label: "X", Template: "${full_name}"}}
			},
			wantErr: "is not supported",
		},
		{
			name: "broken command template",
			mutate: func(p *model.SettingsPayload) {
				p.DBFunctions.RemoveRole = model.DBFunction{Call: "DROP ROLE ${nope}", Execution: model.ExecutionStatement}
			},
			wantErr: "unknown placeholder",
		},
		{
			name: "read query missing its bind",
			mutate: func(p *model.SettingsPayload) {
				p.DBReads.RoleDetail = model.DBRead{Query: "SELECT 1"}
			},
			wantErr: "must reference",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, path := newTestStore(t)
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}

			p := goodSettings()
			tc.mutate(&p)
			err = s.SaveSettings(p)
			if err == nil {
				t.Fatalf("SaveSettings accepted an invalid payload")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error = %v, want it to mention %q", err, tc.wantErr)
			}

			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(before) != string(after) {
				t.Error("config file changed despite the save being rejected — the write was not atomic")
			}
			// The in-memory config must not have moved either.
			if got := s.Get().Batch.MaxConcurrency; got == 7 {
				t.Error("in-memory config was mutated by a rejected save")
			}
		})
	}
}

// A comment field added in the SAME payload as a template that uses it must validate — the old
// sequential save only worked because the caller happened to save comment fields first.
func TestSaveSettingsValidatesTemplatesAgainstItsOwnCommentFields(t *testing.T) {
	s, _ := newTestStore(t)
	p := goodSettings()
	p.CommentFields = []model.CommentField{{Key: "department", Label: "Department"}}
	p.SearchColumns = nil
	p.DBFunctions.SetComment = model.DBFunction{
		Call:      "COMMENT ON ROLE ${loginname} IS ${{department}}",
		Execution: model.ExecutionStatement,
	}
	if err := s.SaveSettings(p); err != nil {
		t.Fatalf("a template using a comment field from the same payload was rejected: %v", err)
	}
}
