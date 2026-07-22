package config

import (
	"testing"

	"github.com/michalbartak/dbaccounts/internal/model"
)

func TestDefaultConfigUITheme(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.UI.Theme != model.ThemeSystem {
		t.Fatalf("default ui.theme = %q, want system", cfg.UI.Theme)
	}
}

func TestNormalizeCommentView(t *testing.T) {
	tests := []struct{ in, want string }{
		{"raw", model.CommentViewRaw},
		{"fields", model.CommentViewFields},
		{"", model.CommentViewFields},
		{"invalid", model.CommentViewFields},
		{" RAW ", model.CommentViewRaw},
	}
	for _, tc := range tests {
		if got := model.NormalizeCommentView(tc.in); got != tc.want {
			t.Errorf("NormalizeCommentView(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestUpdateUIPersistsCommentView(t *testing.T) {
	s := tmpStore(t)
	if err := s.UpdateUI(model.UISettings{Theme: "dark", CommentDefaultView: "raw"}); err != nil {
		t.Fatal(err)
	}
	if got := s.Get().UI.CommentDefaultView; got != model.CommentViewRaw {
		t.Fatalf("comment view not persisted: %q", got)
	}
	// invalid normalizes to fields
	if err := s.UpdateUI(model.UISettings{Theme: "dark", CommentDefaultView: "nope"}); err != nil {
		t.Fatal(err)
	}
	if got := s.Get().UI.CommentDefaultView; got != model.CommentViewFields {
		t.Fatalf("invalid comment view should normalize to fields: %q", got)
	}
}

func TestNormalizeTheme(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"light", model.ThemeLight},
		{"dark", model.ThemeDark},
		{"system", model.ThemeSystem},
		{"", model.ThemeSystem},
		{"invalid", model.ThemeSystem},
	}
	for _, tc := range tests {
		if got := model.NormalizeTheme(tc.in); got != tc.want {
			t.Errorf("NormalizeTheme(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
