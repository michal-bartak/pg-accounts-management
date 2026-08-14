// Package update checks the project's published GitHub Releases for a newer version. It is a
// leaf package: stdlib + internal/model only (see the import table in CLAUDE.md).
package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/michal-bartak/pgcowboy/internal/model"
)

// apiBase is the GitHub REST API root; overridable in tests.
var apiBase = "https://api.github.com"

// httpTimeout bounds the whole release lookup so a slow network never blocks the UI.
const httpTimeout = 6 * time.Second

type ghRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	HTMLURL string `json:"html_url"`
	Body    string `json:"body"`
}

// Check queries GitHub for the latest release of repoURL (https://github.com/OWNER/REPO) and
// compares it against currentVersion. A repo with no releases (404) is reported as "up to date"
// (not an error). Network/parse failures return an error.
func Check(ctx context.Context, currentVersion, repoURL string) (model.UpdateInfo, error) {
	info := model.UpdateInfo{CurrentVersion: strings.TrimSpace(currentVersion)}
	owner, repo, ok := ownerRepo(repoURL)
	if !ok {
		return info, fmt.Errorf("not a GitHub repo URL: %q", repoURL)
	}

	ctx, cancel := context.WithTimeout(ctx, httpTimeout)
	defer cancel()
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", apiBase, owner, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return info, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "pgCowboy")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return info, err
	}
	defer resp.Body.Close()

	// No releases published yet — treat as up to date, not an error.
	if resp.StatusCode == http.StatusNotFound {
		return info, nil
	}
	if resp.StatusCode != http.StatusOK {
		return info, fmt.Errorf("github releases: HTTP %d", resp.StatusCode)
	}

	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return info, err
	}

	info.LatestVersion = strings.TrimPrefix(strings.TrimSpace(rel.TagName), "v")
	info.ReleaseURL = rel.HTMLURL
	info.ReleaseName = rel.Name
	info.Notes = rel.Body
	info.UpdateAvailable = compareVersions(info.LatestVersion, info.CurrentVersion) > 0
	return info, nil
}

// Pending reconstructs — without any network call — the update the user was last informed about
// (the persisted seenVersion) if it is still newer than currentVersion. It drives the
// "update available" badge across restarts, including when the startup auto-check is off, and
// naturally reports "not available" once the user upgrades past the seen version. ReleaseURL
// points at the repo's latest-release page (the exact html_url is only known after a Check).
func Pending(seenVersion, currentVersion, repoURL string) model.UpdateInfo {
	info := model.UpdateInfo{CurrentVersion: strings.TrimSpace(currentVersion)}
	seen := strings.TrimPrefix(strings.TrimSpace(seenVersion), "v")
	if seen != "" && compareVersions(seen, info.CurrentVersion) > 0 {
		info.LatestVersion = seen
		info.UpdateAvailable = true
		if r := strings.TrimRight(strings.TrimSpace(repoURL), "/"); r != "" {
			info.ReleaseURL = r + "/releases/latest"
		}
	}
	return info
}

// ownerRepo extracts OWNER, REPO from a https://github.com/OWNER/REPO URL.
func ownerRepo(repoURL string) (owner, repo string, ok bool) {
	rest, found := strings.CutPrefix(strings.TrimSpace(repoURL), "https://github.com/")
	if !found {
		return "", "", false
	}
	parts := strings.SplitN(strings.Trim(rest, "/"), "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], strings.TrimSuffix(parts[1], ".git"), true
}

// compareVersions compares two MAJOR.MINOR.PATCH strings (leading "v" and any "-prerelease" or
// "+build" suffix are ignored). Returns -1, 0, or 1. Missing components are treated as 0, and a
// non-numeric component sorts as 0, so a malformed remote tag can't spuriously look "newer".
func compareVersions(a, b string) int {
	an, bn := parseVersion(a), parseVersion(b)
	for i := 0; i < 3; i++ {
		if an[i] != bn[i] {
			if an[i] > bn[i] {
				return 1
			}
			return -1
		}
	}
	return 0
}

func parseVersion(s string) [3]int {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "v")
	// Drop any prerelease/build metadata: 0.4.0-rc1 / 0.4.0+build → 0.4.0.
	if i := strings.IndexAny(s, "-+"); i >= 0 {
		s = s[:i]
	}
	var out [3]int
	for i, part := range strings.SplitN(s, ".", 3) {
		if i > 2 {
			break
		}
		n, err := strconv.Atoi(strings.TrimSpace(part))
		if err == nil && n > 0 {
			out[i] = n
		}
	}
	return out
}
