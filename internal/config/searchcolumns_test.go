package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/michal-bartak/pgcowboy/internal/model"
)

func TestValidateSearchColumns(t *testing.T) {
	got, err := validateSearchColumns([]model.SearchColumn{
		{Label: " Full name ", Template: " ${{first_name}} ${{last_name}} "},
		{Label: "", Template: "${comment}"}, // blank label is kept: an empty header cell
		{Label: "Dropped", Template: "   "},
		// Arbitrary comment keys are allowed — they are JSON keys, not SQL identifiers.
		{Label: "Mail", Template: "${{e-mail}}"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("want 3 columns, got %d: %v", len(got), got)
	}
	if got[0].Label != "Full name" || got[0].Template != "${{first_name}} ${{last_name}}" {
		t.Fatalf("trim failed: %+v", got[0])
	}
	if got[1].Label != "" || got[1].Template != "${comment}" {
		t.Fatalf("blank label should be kept: %+v", got[1])
	}
	if got[2].Template != "${{e-mail}}" {
		t.Fatalf("non-identifier comment key should be allowed: %+v", got[2])
	}
	if _, err := validateSearchColumns([]model.SearchColumn{{Label: "x", Template: "${oops"}}); err == nil {
		t.Fatal("expected error for unterminated ${")
	}
}

// TestValidateSearchColumns_bareNameIsGuided pins the closed bare namespace: a comment key written
// in the single-brace form is rejected, and the error tells the user the double-brace form.
func TestValidateSearchColumns_bareNameIsGuided(t *testing.T) {
	_, err := validateSearchColumns([]model.SearchColumn{{Label: "Full name", Template: "${full_name}"}})
	if err == nil {
		t.Fatal("bare ${full_name} should be rejected")
	}
	if !strings.Contains(err.Error(), "${{full_name}}") {
		t.Fatalf("error should name the fix, got: %v", err)
	}
	// ${comment} is the one built-in, so it passes.
	if _, err := validateSearchColumns([]model.SearchColumn{{Label: "Raw", Template: "pre ${comment} post"}}); err != nil {
		t.Fatalf("${comment} is a built-in: %v", err)
	}
	// Empty names and malformed braces are structural errors.
	for _, tmpl := range []string{"${}", "${{}}", "${{x}", "${a{b}"} {
		if _, err := validateSearchColumns([]model.SearchColumn{{Label: "x", Template: tmpl}}); err == nil {
			t.Fatalf("expected %q to be rejected", tmpl)
		}
	}
}

func TestSanitizeSearchColumns(t *testing.T) {
	got := sanitizeSearchColumns([]model.SearchColumn{
		{Label: " A ", Template: " ${{a}} "},
		{Label: "empty", Template: ""},     // dropped
		{Label: "broken", Template: "${x"}, // dropped (structurally invalid, non-failing)
		// A stale bare name is KEPT: it still renders (as the literal token) and the user gets the
		// guiding error on their next save — better than silently deleting their column.
		{Label: "Stale", Template: "${full_name}"},
	})
	if len(got) != 2 {
		t.Fatalf("want 2 columns, got %d: %v", len(got), got)
	}
	if got[0].Label != "A" || got[0].Template != "${{a}}" {
		t.Fatalf("unexpected column: %+v", got[0])
	}
	if got[1].Template != "${full_name}" {
		t.Fatalf("a stale bare name should survive load: %+v", got[1])
	}
	// Nil-ness is preserved, so Load can tell an absent key from an explicit empty list.
	if sanitizeSearchColumns(nil) != nil {
		t.Fatal("nil input should stay nil")
	}
	if got := sanitizeSearchColumns([]model.SearchColumn{}); got == nil {
		t.Fatal("an explicitly empty list must not become nil")
	}
}

func TestDefaultSearchColumns(t *testing.T) {
	got := defaultSearchColumns()
	if len(got) != 1 || got[0].Label != "Full name" || got[0].Template != "${{full_name}}" {
		t.Fatalf("unexpected defaults: %v", got)
	}
}

func TestUpdateSearchColumnsPersistsAndValidates(t *testing.T) {
	s := tmpStore(t)
	if err := s.UpdateSearchColumns([]model.SearchColumn{
		{Label: "Name", Template: "${{first_name}} ${{last_name}}"},
		{Label: "Raw", Template: "${comment}"},
	}); err != nil {
		t.Fatal(err)
	}
	if got := s.Get().SearchColumns; len(got) != 2 || got[1].Template != "${comment}" {
		t.Fatalf("not persisted: %v", got)
	}
	// Reload from disk to confirm it round-trips through YAML.
	s2 := &Store{path: s.path}
	if err := s2.Load(); err != nil {
		t.Fatal(err)
	}
	if got := s2.Get().SearchColumns; len(got) != 2 || got[0].Label != "Name" {
		t.Fatalf("did not round-trip: %v", got)
	}
	if err := s.UpdateSearchColumns([]model.SearchColumn{{Label: "x", Template: "${oops"}}); err == nil {
		t.Fatal("expected error for unterminated ${")
	}
}

func TestLoadInjectsDefaultSearchColumnsOnlyWhenAbsent(t *testing.T) {
	// A config written by an older version has no search_columns key at all → the built-in
	// column. (Every config this version saves carries the key explicitly, so the absent case
	// can only come from an existing file on disk.)
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("version: 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	s2 := &Store{path: path}
	if err := s2.Load(); err != nil {
		t.Fatal(err)
	}
	if got := s2.Get().SearchColumns; len(got) != 1 || got[0].Template != "${{full_name}}" {
		t.Fatalf("Load should inject the default when search_columns is omitted: %v", got)
	}

	// "Role name only" is a legitimate saved choice and must survive a restart, so an
	// explicitly empty list must NOT be replaced by the default.
	s3 := tmpStore(t)
	if err := s3.UpdateSearchColumns([]model.SearchColumn{}); err != nil {
		t.Fatal(err)
	}
	s4 := &Store{path: s3.path}
	if err := s4.Load(); err != nil {
		t.Fatal(err)
	}
	if got := s4.Get().SearchColumns; len(got) != 0 {
		t.Fatalf("an explicitly empty search_columns must stay empty, got: %v", got)
	}
}
