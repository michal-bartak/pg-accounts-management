//go:build !linux || !cgo

package main

// SetNativeDarkTheme is a no-op away from a cgo Linux build: macOS and Windows draw a <select>'s
// drop-down list in the appearance the page already declares through CSS `color-scheme`. See
// native_theme_linux.go for what it does on Linux, and why it is needed only there.
func (a *App) SetNativeDarkTheme(dark bool) {}
