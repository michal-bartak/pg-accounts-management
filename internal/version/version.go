// Package version holds application release metadata (overridden at link time via -ldflags).
package version

import "strings"

// Defaults match VERSION at repo root; release builds set these via Makefile / wails build -ldflags.
var (
	Version   = "0.1.0"
	Commit    = "dev"
	BuildDate = ""
)

// Info is exposed to the UI and tooling.
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"buildDate"`
}

func Get() Info {
	return Info{
		Version:   strings.TrimSpace(Version),
		Commit:    strings.TrimSpace(Commit),
		BuildDate: strings.TrimSpace(BuildDate),
	}
}

// String returns a short display label, e.g. "1.0.0 (abc1234)".
func (i Info) String() string {
	v := i.Version
	if v == "" {
		v = "unknown"
	}
	if i.Commit != "" && i.Commit != "dev" {
		return v + " (" + i.Commit + ")"
	}
	return v
}
