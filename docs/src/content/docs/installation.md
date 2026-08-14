---
title: Installation
description: Download a pre-built release or install and run from source
---

## Download an installer

Each release on the
[GitHub Releases page](https://github.com/michal-bartak/pgcowboy/releases)
ships a native installer per platform:

| Platform | File |
|----------|------|
| macOS (Intel + Apple silicon) | `pgCowboy-v{VERSION}-macos-universal.dmg` |
| Windows 10/11 (x64) | `pgCowboy-v{VERSION}-windows-amd64.msi` |
| Debian, Ubuntu (x64) | `pgCowboy-v{VERSION}-linux-amd64.deb` |
| Fedora, RHEL, openSUSE (x64) | `pgCowboy-v{VERSION}-linux-amd64.rpm` |

`{VERSION}` here and in the commands below stands for the release you downloaded, e.g. `1.0.0`.

## System requirements

pgCowboy draws its interface with the operating system's own web engine, so the engine
version — not just the OS version — matters.

| Platform | Requirement |
|----------|-------------|
| macOS | **12 Monterey or later**, kept up to date. |
| Windows | **10 or 11** with the **WebView2 runtime**, which Windows installs and updates automatically. |
| Linux | **WebKitGTK 2.38 or later** (`libwebkit2gtk-4.1-0`), which the `.deb`/`.rpm` pull in. Ships with Ubuntu 22.04+, Debian 12+ and Fedora 37+. |

**Windows 7, 8 and 8.1 are not supported and cannot run pgCowboy at all.** This is not a
matter of the app degrading: the Go toolchain has produced Windows 10+ binaries only since
Go 1.21, so the executable will not start on those versions. Microsoft also ended WebView2
support for them in 2023. There is no build of pgCowboy that runs on Windows 7 — the
project has always been built with a toolchain that excludes it.

On macOS and Linux the failure mode is gentler: a system old enough to carry a pre-2022 web
engine may still launch the app, but the Find-role results table lays its columns out with
CSS subgrid, so on those engines the columns stop lining up. Updating the OS fixes it.

Each release description opens with what is new in that version; the
[changelog](https://github.com/michal-bartak/pgcowboy/blob/main/CHANGELOG.md)
covers every version in one page.

Builds are **not** signed with an Apple or Microsoft developer certificate, so the first
launch may show a security warning.

### macOS

Open the `.dmg` and drag **pgCowboy** to the Applications folder, then eject the disk
image. Gatekeeper blocks the first launch of an unsigned app; clear the quarantine flag
once:

```bash
xattr -dr com.apple.quarantine /Applications/pgCowboy.app
open -a pgCowboy
```

Downloading with `curl -LJO <url>` rather than a browser avoids the quarantine flag
altogether. **Right-click → Open**, or **System Settings → Privacy & Security → Open
Anyway** after a blocked launch, works too.

To uninstall, move `/Applications/pgCowboy.app` to the Trash.

### Windows

Run the `.msi`. It installs to `C:\Program Files\pgCowboy`, adds a Start-menu entry, and
registers an uninstaller under **Settings → Apps**. Installing a newer version upgrades in
place, keeping the install location.

SmartScreen may warn about the unsigned installer: click **More info → Run anyway**.

Silent install (per machine, needs an elevated prompt):

```bat
msiexec /i pgCowboy-v{VERSION}-windows-amd64.msi /qn
```

### Linux

The packages install `/usr/bin/pgCowboy` plus a desktop entry and icon, and pull in the
GTK and WebKit runtime libraries.

```bash
# Debian / Ubuntu
sudo apt install ./pgCowboy-v{VERSION}-linux-amd64.deb

# Fedora / RHEL
sudo dnf install ./pgCowboy-v{VERSION}-linux-amd64.rpm
```

To remove: `sudo apt remove pgcowboy` or `sudo dnf remove pgcowboy`.

If the app fails to start, the WebView libraries are missing or the wrong version —
pgCowboy links against webkit2gtk **4.1** (`libwebkit2gtk-4.1-0` on Debian/Ubuntu,
`webkit2gtk4.1` on Fedora/RHEL).

## Configuration files

On first launch the app writes two default files, which you can edit in the app (Clusters and
Settings tabs) or on disk.

| OS | Directory |
|----|-----------|
| macOS | `~/Library/Application Support/pgCowboy/` |
| Linux | `~/.config/pgcowboy/` |
| Windows | `%AppData%\pgCowboy\` |

`config.yaml` holds call templates and app settings; `clusters.yaml` holds your clusters, their
groups, and the remembered target selection. See
[`config.example.yaml`](https://github.com/michal-bartak/pgcowboy/blob/main/config.example.yaml)
and
[`clusters.example.yaml`](https://github.com/michal-bartak/pgcowboy/blob/main/clusters.example.yaml)
for references, and [Configuration](/pgcowboy/configuration/) for what each
part does.

### Upgrading from DbAccounts

The app was called **DbAccounts** up to v1.0.0. Settings are **not** migrated: pgCowboy uses a
new configuration directory and writes fresh defaults on first launch. To keep your clusters,
groups, templates and any saved per-cluster passwords, copy both files across before you start it:

| OS | From | To |
|----|------|----|
| macOS | `~/Library/Application Support/DbAccounts/` | `~/Library/Application Support/pgCowboy/` |
| Linux | `~/.config/dbaccounts/` | `~/.config/pgcowboy/` |
| Windows | `%AppData%\DbAccounts\` | `%AppData%\pgCowboy\` |

The old directory is left untouched, so you can delete it once you have confirmed everything came
across. The application itself does not upgrade in place either — remove
`/Applications/DbAccounts.app` on macOS, or `sudo apt remove dbaccounts` / `sudo dnf remove
dbaccounts` on Linux. The Windows installer does replace the old install automatically.

## Credentials

The app resolves credentials the way `psql` does. First match wins:

1. **User** — the cluster's **Username**, else `PGUSER`, else your OS login name.
2. **Password** — the cluster's **Password**, else `PGPASSWORD`, else a match in `~/.pgpass`,
   else none (works with trust auth or an empty password).

The per-cluster password is optional and stored in clear text in `clusters.yaml`, which is
written with owner-only permissions. Leave it blank to keep credentials out of the file
entirely. See the PostgreSQL docs on
[`.pgpass`](https://www.postgresql.org/docs/current/libpq-pgpass.html).

Building from source is covered under
[Building from source](/pgcowboy/building/).
