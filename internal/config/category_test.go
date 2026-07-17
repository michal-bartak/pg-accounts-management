package config

import (
	"path/filepath"
	"testing"

	"github.com/michalbartak/dbaccounts/internal/model"
)

func tmpStore(t *testing.T) *Store {
	t.Helper()
	return &Store{path: filepath.Join(t.TempDir(), "config.yaml"), cfg: DefaultConfig()}
}

func TestCategoryCRUD(t *testing.T) {
	s := tmpStore(t)

	// Add: id is slugified from the label.
	c, err := s.AddCategory(model.CategoryInput{Label: "Pre Prod", Color: "#123456", Confirm: true})
	if err != nil {
		t.Fatal(err)
	}
	if c.ID != "pre_prod" || c.Color != "#123456" || !c.Confirm {
		t.Fatalf("added category wrong: %+v", c)
	}

	// Duplicate id rejected.
	if _, err := s.AddCategory(model.CategoryInput{Label: "Pre  prod"}); err == nil {
		t.Fatal("expected duplicate id to be rejected")
	}

	// Update label/color/confirm; id stays.
	u, err := s.UpdateCategory("pre_prod", model.CategoryInput{Label: "Pre-Prod", Color: "#ABCDEF", Confirm: false})
	if err != nil || u.ID != "pre_prod" || u.Color != "#abcdef" || u.Confirm {
		t.Fatalf("update wrong: %+v err=%v", u, err)
	}

	// Delete blocked while a cluster references the group.
	s.cfg.Clusters = append(s.cfg.Clusters, model.Cluster{ID: "x", Alias: "x", Category: "pre_prod"})
	if err := s.DeleteCategory("pre_prod"); err == nil {
		t.Fatal("expected in-use group delete to be blocked")
	}
	s.cfg.Clusters = nil
	if err := s.DeleteCategory("pre_prod"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, ok := s.CategoryByID("pre_prod"); ok {
		t.Fatal("category should be gone")
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{"Production": "production", "Pre Prod": "pre_prod", "UAT-2": "uat_2", "  a  ": "a", "!!!": ""}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Fatalf("slugify(%q)=%q want %q", in, got, want)
		}
	}
}
