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
