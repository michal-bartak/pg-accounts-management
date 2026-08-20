//go:build !linux

package main

// IsSystemDark is only meaningful on Linux — macOS and Windows resolve the "System" appearance
// in the webview via `prefers-color-scheme`, which those engines answer correctly. The frontend
// never calls this off Linux; it exists so the binding is present on every platform.
func (a *App) IsSystemDark() bool {
	return false
}
