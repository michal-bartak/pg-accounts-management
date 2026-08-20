// Only a REAL desktop build needs (or can link) GTK. Wails gates its own cgo the same way —
// without `dev` or `production` its internal/app falls back to a stub that refuses to run — so a
// plain `go build ./...` / `go test ./...` has never required the GTK headers, and this file must
// not be what changes that: CI runs the bare commands with no libgtk-3-dev installed.
// `wails build` passes `production`, `wails dev` passes `dev`; native_theme_other.go covers the
// exact complement of this constraint, so exactly one definition always exists.
//go:build linux && cgo && (dev || production)

package main

/*
#cgo linux pkg-config: gtk+-3.0
#include <gtk/gtk.h>

// Runs on the GTK main loop. g_idle_add() may be called from any thread; the callback it
// schedules is what must not touch GTK from anywhere else.
static gboolean pgcowboyApplyDarkTheme(gpointer data) {
	GtkSettings *settings = gtk_settings_get_default();
	if (settings != NULL) {
		g_object_set(settings, "gtk-application-prefer-dark-theme",
		             GPOINTER_TO_INT(data) ? TRUE : FALSE, NULL);
	}
	return G_SOURCE_REMOVE;
}

static void pgcowboySetDarkTheme(int dark) {
	g_idle_add(pgcowboyApplyDarkTheme, GINT_TO_POINTER(dark));
}
*/
import "C"

// SetNativeDarkTheme asks GTK for the dark variant of whatever desktop theme is in use.
//
// A <select>'s drop-down LIST is not part of the page: WebKitGTK draws it as a native GTK widget,
// so `appearance: none` never reaches it and neither does any rule on the <option>s — it renders
// in the GTK theme's colours, i.e. white under a light theme however dark the app itself is.
// This setting is the only knob that reaches it, and it covers the other natively drawn surfaces
// (context menus, file dialogs) for free. A theme with no dark variant simply ignores it.
//
// The frontend calls this from applyTheme with the resolved appearance, so it follows
// Dark/Light/System including a live desktop switch. GTK is not thread-safe and bound methods do
// not run on the main loop, so the g_object_set is deferred onto it with g_idle_add.
func (a *App) SetNativeDarkTheme(dark bool) {
	v := C.int(0)
	if dark {
		v = C.int(1)
	}
	C.pgcowboySetDarkTheme(v)
}
