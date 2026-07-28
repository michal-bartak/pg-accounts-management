// Package version holds application release metadata (overridden at link time via -ldflags).
package version

import "strings"

// Defaults match VERSION at repo root; release builds set these via Makefile / wails build -ldflags.
var (
	Version   = "0.3.0"
	Commit    = "dev"
	BuildDate = ""
	// Repo is the canonical GitHub repository URL. Release builds inject the actual git
	// remote (Makefile -ldflags); the default keeps `go run` / `wails dev` working. Any
	// remote form (https, git@, trailing .git) is normalized by normalizeRepo.
	Repo = "https://github.com/michal-bartak/pg-accounts-management"
)

// Info is exposed to the UI and tooling.
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"buildDate"`
	// RepoURL is the normalized GitHub repo URL; DocsURL is the GitHub Pages site derived
	// from it (https://OWNER.github.io/REPO/).
	RepoURL string `json:"repoURL"`
	DocsURL string `json:"docsURL"`
}

func Get() Info {
	repo := normalizeRepo(Repo)
	return Info{
		Version:   strings.TrimSpace(Version),
		Commit:    strings.TrimSpace(Commit),
		BuildDate: strings.TrimSpace(BuildDate),
		RepoURL:   repo,
		DocsURL:   docsURL(repo),
	}
}

// normalizeRepo canonicalizes a git remote URL to https://github.com/OWNER/REPO.
func normalizeRepo(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if rest, ok := strings.CutPrefix(s, "git@github.com:"); ok {
		s = "https://github.com/" + rest
	}
	s = strings.TrimPrefix(s, "ssh://")
	s = strings.TrimRight(s, "/")
	s = strings.TrimSuffix(s, ".git")
	return s
}

// docsURL derives the GitHub Pages URL (https://OWNER.github.io/REPO/) from a repo URL.
// Falls back to the repo URL if it isn't a recognizable github.com/OWNER/REPO address.
func docsURL(repo string) string {
	rest, ok := strings.CutPrefix(repo, "https://github.com/")
	if !ok {
		return repo
	}
	parts := strings.SplitN(strings.Trim(rest, "/"), "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return repo
	}
	return "https://" + parts[0] + ".github.io/" + parts[1] + "/"
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
