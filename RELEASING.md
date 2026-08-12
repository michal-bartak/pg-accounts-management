# Releasing DbAccounts

## Version source of truth

`VERSION` is the **only** file where a version is maintained by hand.

| File | Purpose |
|------|---------|
| [`VERSION`](VERSION) | Application semver (e.g. `0.1.0`) — git tags, installer names, bundle metadata, UI |
| [`wails.json`](wails.json) `info.productVersion` | **Generated mirror** of `VERSION` — `make sync-wails-version` owns it, never hand-edit |
| [`config.yaml`](config.example.yaml) `version:` | **Config schema** version (not app version) |
| [`CHANGELOG.md`](CHANGELOG.md) | User-facing release notes — the section heading must match `VERSION` |

Everything else derives from `VERSION` at build time: the app embeds it (`//go:embed VERSION` in
[`main.go`](main.go)), the installer scripts read it for artifact names and the MSI `ProductVersion`,
and `build`/`build-ci` depend on `sync-wails-version`, so `make package` and the release workflow
always package a bundle whose metadata matches. The `wails.json` copy exists only because Wails v2
hardcodes its config path and has no flag or env var for the version — deleting the key would make
Wails fall back to a hardcoded `1.0.0`.

## Changelog

[`CHANGELOG.md`](CHANGELOG.md) is the human-readable history
([Keep a Changelog](https://keepachangelog.com/en/1.1.0/) format). Add user-facing changes to
`## [Unreleased]` **as you work**, not at release time. **One terse line per change** — bold
feature name, then what it does, no narration. Written for someone who uses the app, not for
someone reading the diff.

When cutting a release, rename that heading to `## [X.Y.Z] - YYYY-MM-DD` and start a fresh empty
`## [Unreleased]` above it. **The heading must match `VERSION` exactly** — the release workflow
finds the section by `## [<VERSION>]` and puts its body at the top of the release description. A
mismatch is not fatal: the workflow logs `::warning::No CHANGELOG.md section found for version …`
and publishes the release with an empty summary.

## Bump version

1. Edit `VERSION` (semver: `MAJOR.MINOR.PATCH`).
2. Move `CHANGELOG.md`'s `## [Unreleased]` section to `## [X.Y.Z] - YYYY-MM-DD` and open a new
   empty `## [Unreleased]`.
3. Run `make sync-wails-version` — prints `productVersion <old> -> <new> (updated - commit this)`.
4. Commit all three files: `git commit -am "chore: release v0.1.0"` (example).
5. Tag: `git tag -a v0.1.0 -m "DbAccounts 0.1.0"`.
6. Push branch and tags to GitLab and/or GitHub remotes.

Step 3 is a convenience — any `make build`/`make package` runs it too, and it rewrites `wails.json`
only when the value actually differs, so builds never churn the file.

Tag names must match `VERSION` with a `v` prefix: `v` + contents of `VERSION`.

## GitHub Releases (manual, multi-platform)

Published from [pg-accounts-management](https://github.com/michal-bartak/pg-accounts-management) via [`.github/workflows/release.yml`](.github/workflows/release.yml). Builds **Linux**, **Windows**, and **macOS** artifacts on native GitHub runners when you run the workflow manually.

### Prerequisites

- Code pushed to the `github` remote (`main`).
- Annotated tag exists on GitHub and matches `VERSION` (e.g. tag `v0.1.0` ↔ `VERSION` = `0.1.0`).
- GitHub **Actions** enabled for the repository.

### Release steps

1. Bump version, close the changelog section, and sync:
   ```bash
   # edit VERSION and CHANGELOG.md (Unreleased -> [0.2.0] - YYYY-MM-DD), then:
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
4. Enter the tag (`v0.1.0` or `0.1.0` — the workflow adds `v` if omitted). Optionally check **draft**.
5. When the workflow completes, download assets from **Releases**.

### Release description

The workflow assembles it from four parts, in order:

1. **What's new** — the `CHANGELOG.md` section matching `VERSION`, extracted by the
   *Extract changelog section* step.
2. A link to the **full changelog** (`CHANGELOG.md` on `main` — not the tag, so it resolves for
   tags cut before the file existed and always shows the complete history), so a user who wants
   every version gets the readable file rather than the commit list.
3. The **install instructions** per platform.
4. GitHub's auto-generated **What's Changed** — the raw commit/PR list
   (`generate_release_notes: true`). Accurate but noisy; the changelog is what most users want.

GitLab (`origin`) pushes are independent; the release workflow runs only on GitHub.

### Release artifacts

Every platform ships a **native installer** — there are no archives to unpack.

| Platform | File | Built by |
|----------|------|----------|
| macOS universal | `DbAccounts-v{VERSION}-macos-universal.dmg` | [`build/scripts/make-dmg.sh`](build/scripts/make-dmg.sh) (`hdiutil`) |
| Windows amd64 | `DbAccounts-v{VERSION}-windows-amd64.msi` | [`build/scripts/make-msi.ps1`](build/scripts/make-msi.ps1) (WiX v3, [`product.wxs`](build/windows/installer/product.wxs)) |
| Linux amd64 (deb) | `DbAccounts-v{VERSION}-linux-amd64.deb` | [`build/scripts/make-linux-packages.sh`](build/scripts/make-linux-packages.sh) (fpm) |
| Linux amd64 (rpm) | `DbAccounts-v{VERSION}-linux-amd64.rpm` | same script |

The workflow runs those scripts through `make package-ci`, so a local `make package` produces the same installer as a release.

Packaging notes:

- **DMG** — the app icon is rebuilt as a full multi-size `.icns` and the bundle ad-hoc re-signed, then staged with an `/Applications` drop target and a background image. Finder automation sets the window layout; if it is unavailable the DMG still builds, just without the layout.
- **MSI** — per-machine install into `C:\Program Files\DbAccounts`, Start-menu shortcut, in-place upgrades (`MajorUpgrade`), install path remembered under `HKLM\Software\MichalBartak\DbAccounts`. The `UpgradeCode` GUID in `product.wxs` must never change. Banner and dialog bitmaps are generated from `build/appicon.png` at build time.
- **deb/rpm** — `/usr/bin/DbAccounts`, a `.desktop` entry from [`build/linux/dbaccounts.desktop`](build/linux/dbaccounts.desktop) and an icon; package name `dbaccounts`. Runtime dependencies name the WebKit the binary is linked against (`libwebkit2gtk-4.1-0` / `webkit2gtk4.1`) — update them together with the build tag if that ever changes. Set `MAINTAINER="Name <email>"` to override the maintainer field.

On Ubuntu 24.04+ (WebKit 4.1 only), `make build` auto-detects `webkit2gtk-4.1` and passes `-tags webkit2_41` to Wails. Install dev packages: `libgtk-3-dev libwebkit2gtk-4.1-dev`.

### Unsigned builds (macOS and Windows)

GitHub Actions produces **ad-hoc / unsigned** binaries. There is no Apple Developer ID notarization or Windows Authenticode signing unless you add that separately (requires paid certificates and CI secrets).

The release workflow already puts these instructions in the release description:

**macOS**

1. Open the `.dmg` and drag **DbAccounts** to Applications.
2. Clear the quarantine flag once: `xattr -dr com.apple.quarantine /Applications/DbAccounts.app`
3. Or right-click the app → **Open** → confirm **Open**, or **System Settings → Privacy & Security → Open Anyway** after a blocked launch.
4. Downloading with `curl -LJO <url>` instead of a browser sets no quarantine flag at all.

**Windows**

- If SmartScreen appears on the `.msi`: **More info** → **Run anyway**.

To remove Gatekeeper warnings entirely, you would need Apple Developer Program membership (~$99/year), a Developer ID certificate, and notarization in the macOS CI job. See [Wails macOS signing discussion](https://github.com/wailsapp/wails/issues/3868) if you pursue that later.

## Local build and package

```bash
make test          # optional but recommended
make package       # sync version, wails build, native installer under dist/
```

`make package` builds the installer for the host OS. Extra tooling per host: WiX Toolset v3 on
Windows (`choco install wixtoolset`), `fpm` + `rpm` on Linux
(`sudo apt install rpm ruby-dev && sudo gem install fpm`); macOS needs only the system tools
(plus Pillow if you want the DMG background image).

Build a universal macOS bundle with `make package PLATFORM=darwin/universal` — `PLATFORM` is
passed straight to `wails build` and its last element names the artifact.

For CI-style build without re-running tests: `make package-ci`.

The version needs no ldflags — `main.go` embeds the `VERSION` file and assigns
`version.Version` at startup, so `go run`, `wails dev` and every build report the same number.
`-ldflags` inject only the build metadata:

- `version.Commit` — `git rev-parse --short HEAD`
- `version.BuildDate` — UTC timestamp
- `version.Repo` — the `origin` remote URL

Running without them (`wails dev`, `go run .`) shows the real version with `Commit` as `dev` and the
default repo URL from [`internal/version/version.go`](internal/version/version.go).

## Git tag checklist

- [ ] `VERSION` updated
- [ ] `CHANGELOG.md` `[Unreleased]` renamed to `[$(cat VERSION)] - <today>`, new empty `[Unreleased]` added
- [ ] `make sync-wails-version` run and the `wails.json` change committed
- [ ] `make test` passes
- [ ] `make package` produces the expected installer under `dist/` (local smoke test)
- [ ] Tag `v$(cat VERSION)` created and pushed to GitHub
- [ ] Release workflow run manually; all four installers appear on GitHub Releases

## Artifact naming (local builds)

| OS | Example |
|----|---------|
| macOS | `dist/DbAccounts-v0.1.0-macos-arm64.dmg` (`macos-universal` with `PLATFORM=darwin/universal`) |
| Linux | `dist/DbAccounts-v0.1.0-linux-amd64.deb`, `dist/DbAccounts-v0.1.0-linux-amd64.rpm` |
| Windows | `dist/DbAccounts-v0.1.0-windows-amd64.msi` |
