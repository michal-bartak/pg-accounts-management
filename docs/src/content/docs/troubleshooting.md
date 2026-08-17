---
title: Troubleshooting
description: Connection, permission, and first-launch issues
---

## A cluster shows as unreachable

Search and role-load report each cluster's result separately, so one unreachable cluster doesn't stop the others. Click the status chip (or a search result's note) to see the message. Common causes:

- Wrong host, port or `sslmode` — fix it on the **Clusters** tab and use **Test connections**.
- The database refuses the connection (firewall, `pg_hba.conf`, TLS requirement).
- No credentials resolved — see below.

## Authentication fails

The app resolves credentials like `psql`:

1. **User** — the cluster's **Username**, else `PGUSER`, else your OS login name.
1. **Password** — the cluster's **Password**, else `PGPASSWORD`, else `~/.pgpass`, else none.

If you rely on `~/.pgpass`, check that it matches the host/port/database/user and has `0600` permissions.

:::tip
To try a one-off password without changing your environment, use **Test connection** inside the [cluster editor](/pgcowboy/configuration/clusters/).
:::

## A change fails on one cluster

Each cluster's operations run as a single transaction: if one statement fails, that cluster is **rolled back** and its error names the failing operation. Other clusters are unaffected.

Open the [command log](/pgcowboy/usage/command-log/) and use the view button on a row to see the exact SQL that ran, then fix the cause (a missing parent role, insufficient privilege, an existing role) and Save again.

:::tip
Your pending edits are kept — a failed run doesn't touch the form.
:::

## "Permission denied" for catalog reads or DDL

The connecting user needs enough privilege to run the operation. Two options:

- Connect as a user that has the privilege.
- Point the relevant template (write) or introspection query (read) at a wrapper function/view that runs with higher privilege, so a low-privilege connection can act through it. See [Call templates](/pgcowboy/configuration/call-templates/).

## macOS won't open the app

CI builds aren't signed. Right-click the app → **Open** → **Open**, or clear the quarantine flag:

```bash
xattr -dr com.apple.quarantine /path/to/pgCowboy.app
```

## Linux app won't start

Install the WebView libraries:

```bash
sudo apt install libgtk-3-0 libwebkit2gtk-4.1-0
```

## Where is my configuration?

| OS | Directory |
|----|-----------|
| macOS | `~/Library/Application Support/pgCowboy/` |
| Linux | `~/.config/pgcowboy/` |
| Windows | `%AppData%\pgCowboy\` |

`config.yaml` holds templates and app settings. `clusters.yaml` holds clusters, groups and the target selection.

:::caution
`clusters.yaml` also holds the optional per-cluster password, in clear text, when you set one. See [Clusters](/pgcowboy/configuration/clusters/).
:::
