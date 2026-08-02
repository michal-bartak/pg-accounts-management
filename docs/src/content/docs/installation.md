---
title: Installation
description: Download a pre-built release or install and run from source
---

## Download a release

Pre-built binaries are on the
[GitHub Releases page](https://github.com/michal-bartak/pg-accounts-management/releases).
CI builds are **not** signed with an Apple or Microsoft developer certificate, so the first
launch may show a security warning.

### macOS

1. Extract `DbAccounts.app` from the `.tar.gz`.
2. **Right-click** the app → **Open** → confirm **Open** (a plain double-click is blocked the
   first time).

Alternatively, allow it from **System Settings → Privacy & Security → Open Anyway**, or clear
the quarantine flag in Terminal:

```bash
xattr -dr com.apple.quarantine /path/to/DbAccounts.app
open /path/to/DbAccounts.app
```

### Windows

If SmartScreen blocks the `.exe`, click **More info → Run anyway**.

### Linux

Extract the tarball and run `./DbAccounts`. If it fails to start, install the WebView
libraries:

```bash
sudo apt install libgtk-3-0 libwebkit2gtk-4.1-0
```

## Configuration file

On first launch the app writes a default config, which you can edit in the app (Clusters and
Settings tabs) or on disk.

| OS | Path |
|----|------|
| macOS | `~/Library/Application Support/DbAccounts/config.yaml` |
| Linux | `~/.config/dbaccounts/config.yaml` |
| Windows | `%AppData%\DbAccounts\config.yaml` |

The config holds clusters, groups, and call templates. See
[`config.example.yaml`](https://github.com/michal-bartak/pg-accounts-management/blob/main/config.example.yaml)
for a reference, and [Configuration](/pg-accounts-management/configuration/) for what each
part does.

## Credentials

The app resolves credentials the way `psql` does. First match wins:

1. **User** — the cluster's **Username**, else `PGUSER`, else your OS login name.
2. **Password** — the cluster's **Password**, else `PGPASSWORD`, else a match in `~/.pgpass`,
   else none (works with trust auth or an empty password).

The per-cluster password is optional and stored in clear text in the config file, which is
written with owner-only permissions. Leave it blank to keep credentials out of the file
entirely. See the PostgreSQL docs on
[`.pgpass`](https://www.postgresql.org/docs/current/libpq-pgpass.html).

Building from source is covered under
[Building from source](/pg-accounts-management/building/).
