//go:build linux

package main

import (
	"os/exec"
	"strings"
)

// IsSystemDark reports whether the desktop environment prefers a dark theme.
//
// WebKitGTK does not reliably answer `prefers-color-scheme` — it reports "light" whatever the
// desktop is set to — so the "System" appearance setting cannot be resolved in the webview on
// Linux. The frontend asks the backend instead, which probes GNOME (gsettings) then KDE
// (kreadconfig). Everything else falls back to light, which is also what the webview would say.
func (a *App) IsSystemDark() bool {
	if dark, ok := gnomeColorScheme(); ok {
		return dark
	}
	if dark, ok := gnomeGtkTheme(); ok {
		return dark
	}
	if dark, ok := kdeColorScheme(); ok {
		return dark
	}
	return false
}

// gsettingsGet returns the raw (still quoted) value of a gsettings key, and whether the read
// succeeded — a missing gsettings binary or schema is "unknown", not "light".
func gsettingsGet(schema, key string) (string, bool) {
	out, err := exec.Command("gsettings", "get", schema, key).Output()
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(out)), true
}

// gnomeColorScheme reads the modern GNOME 42+ preference. 'default' means "no dark preference".
func gnomeColorScheme() (bool, bool) {
	val, ok := gsettingsGet("org.gnome.desktop.interface", "color-scheme")
	if !ok {
		return false, false
	}
	switch val {
	case "'prefer-dark'":
		return true, true
	case "'prefer-light'", "'default'":
		return false, true
	default:
		return false, false
	}
}

// gnomeGtkTheme is the pre-GNOME-42 fallback: a theme name such as 'Adwaita-dark'.
func gnomeGtkTheme() (bool, bool) {
	val, ok := gsettingsGet("org.gnome.desktop.interface", "gtk-theme")
	if !ok {
		return false, false
	}
	theme := strings.ToLower(strings.Trim(val, "'"))
	return strings.Contains(theme, "dark"), true
}

// kdeColorScheme reads Plasma's colour scheme name (e.g. "BreezeDark"), trying Plasma 6 then 5.
func kdeColorScheme() (bool, bool) {
	out, err := exec.Command("kreadconfig6", "--file", "kdeglobals", "--group", "General", "--key", "ColorScheme").Output()
	if err != nil {
		out, err = exec.Command("kreadconfig5", "--file", "kdeglobals", "--group", "General", "--key", "ColorScheme").Output()
		if err != nil {
			return false, false
		}
	}
	scheme := strings.ToLower(strings.TrimSpace(string(out)))
	return strings.Contains(scheme, "dark"), true
}
