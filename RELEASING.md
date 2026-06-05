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
5. Push branch and tags: `git push && git push origin v0.1.0`.

Tag names must match `VERSION` with a `v` prefix: `v` + contents of `VERSION`.

## Build and package

```bash
make test          # optional but recommended
make package       # sync version, wails build, dist/*.tar.gz (macOS .app)
```

Build injects into the binary:

- `version.Version` — from `VERSION`
- `version.Commit` — `git rev-parse --short HEAD`
- `version.BuildDate` — UTC timestamp

Development (`wails dev`) uses defaults in [`internal/version/version.go`](internal/version/version.go) unless you pass the same `-ldflags`.

## Git tag checklist

- [ ] `VERSION` updated
- [ ] `make sync-wails-version`
- [ ] `make test` passes
- [ ] `make package` produces expected artifact under `dist/`
- [ ] Tag `v$(cat VERSION)` created and pushed
- [ ] Release notes (optional GitHub Release) reference tag and `dist/` artifact name

## Artifact naming

`dist/DbAccounts-v{VERSION}-{GOOS}-{GOARCH}.tar.gz` — macOS app bundle inside.

Example: `dist/DbAccounts-v0.1.0-darwin-arm64.tar.gz`
