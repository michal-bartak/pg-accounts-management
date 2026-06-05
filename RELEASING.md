# Releasing DbAccounts

## Version source of truth

| File | Purpose |
|------|---------|
| [`VERSION`](VERSION) | Application semver (e.g. `0.1.0`) — used for git tags, builds, UI |
| [`wails.json`](wails.json) `info.productVersion` | macOS/Windows bundle metadata — synced via `make sync-wails-version` |
| [`config.yaml`](config.example.yaml) `version:` | **Config schema** version (not app version) |

## Bump version

1. Edit `VERSION` (semver: `MAJOR.MINOR.PATCH`).
2. Run `make sync-wails-version`.
3. Commit: `git commit -am "chore: release v0.1.0"` (example).
4. Tag: `git tag -a v0.1.0 -m "DbAccounts 0.1.0"`.
5. Push branch and tags to GitLab and/or GitHub remotes.

Tag names must match `VERSION` with a `v` prefix: `v` + contents of `VERSION`.

## GitHub Releases (manual, multi-platform)

Published from [pg-accounts-management](https://github.com/michal-bartak/pg-accounts-management) via [`.github/workflows/release.yml`](.github/workflows/release.yml). Builds **Linux**, **Windows**, and **macOS** artifacts on native GitHub runners when you run the workflow manually.

### Prerequisites

- Code pushed to the `github` remote (`main`).
- Annotated tag exists on GitHub and matches `VERSION` (e.g. tag `v0.1.0` ↔ `VERSION` = `0.1.0`).
- GitHub **Actions** enabled for the repository.

### Release steps

1. Bump version and sync:
   ```bash
   # edit VERSION, then:
   make sync-wails-version
   git commit -am "chore: release v0.2.0"
   git push github main
   ```
2. Create and push the tag:
   ```bash
   git tag -a v0.2.0 -m "DbAccounts 0.2.0"
   git push github v0.2.0
   ```
3. GitHub → **Actions** → **release** → **Run workflow**.
4. Enter the tag (e.g. `v0.2.0`). Optionally check **draft**.
5. When the workflow completes, download assets from **Releases**.

GitLab (`origin`) pushes are independent; the release workflow runs only on GitHub.

### Release artifacts

| Platform | File |
|----------|------|
| Linux amd64 | `DbAccounts-v{VERSION}-linux-amd64.tar.gz` |
| Windows amd64 | `DbAccounts-v{VERSION}-windows-amd64.zip` (contains `.exe`) |
| macOS arm64 | `DbAccounts-v{VERSION}-darwin-arm64.tar.gz` (contains `.app`) |

Linux tarball users need runtime libraries installed (`libgtk-3-0`, `libwebkit2gtk-4.1-0` on Debian/Ubuntu).

## Local build and package

```bash
make test          # optional but recommended
make package       # sync version, wails build, dist/ artifact
```

For CI-style build without re-running tests: `make package-ci`.

Build injects into the binary:

- `version.Version` — from `VERSION`
- `version.Commit` — `git rev-parse --short HEAD`
- `version.BuildDate` — UTC timestamp

Development (`wails dev`) uses defaults in [`internal/version/version.go`](internal/version/version.go) unless you pass the same `-ldflags`.

## Git tag checklist

- [ ] `VERSION` updated
- [ ] `make sync-wails-version`
- [ ] `make test` passes
- [ ] `make package` produces expected artifact under `dist/` (local smoke test)
- [ ] Tag `v$(cat VERSION)` created and pushed to GitHub
- [ ] Release workflow run manually; assets appear on GitHub Releases

## Artifact naming (local builds)

| OS | Example |
|----|---------|
| macOS | `dist/DbAccounts-v0.1.0-darwin-arm64.tar.gz` |
| Linux | `dist/DbAccounts-v0.1.0-linux-amd64.tar.gz` |
| Windows | `dist/DbAccounts-v0.1.0-windows-amd64.exe` |
