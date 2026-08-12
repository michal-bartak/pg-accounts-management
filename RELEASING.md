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
4. Enter the tag (`v0.1.0` or `0.1.0` — the workflow adds `v` if omitted). Optionally check **draft**.
5. When the workflow completes, download assets from **Releases**.

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

Build injects into the binary:

- `version.Version` — from `VERSION`
- `version.Commit` — `git rev-parse --short HEAD`
- `version.BuildDate` — UTC timestamp

Development (`wails dev`) uses defaults in [`internal/version/version.go`](internal/version/version.go) unless you pass the same `-ldflags`.

## Git tag checklist

- [ ] `VERSION` updated
- [ ] `make sync-wails-version`
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
