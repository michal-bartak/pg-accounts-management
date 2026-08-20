//go:build !linux || !cgo || (!dev && !production)

package main

// SetNativeDarkTheme is a no-op outside a real cgo Linux desktop build: macOS and Windows draw a
// <select>'s drop-down list in the appearance the page already declares through CSS
// `color-scheme`, and an untagged build has no window at all (Wails stubs itself out without
// `dev`/`production`). See native_theme_linux.go for what it does on Linux, why it is needed only
// there, and why the tags matter.
func (a *App) SetNativeDarkTheme(dark bool) {}
