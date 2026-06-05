# DbAccounts

Cross-platform desktop app for maintaining PostgreSQL roles across many clusters. Built with **Go**, **Wails v2** (WebView), and **pgx**.

## Features

- Manage PostgreSQL clusters (alias, host, port, database, category)
- Categories: **Production** and **UAT** (extensible in config)
- Run role operations in batch against selected categories and/or clusters:
  - Create role (login, full name, email, parent role)
  - Remove role
  - Grant parent role(s)
  - Change password
- Operations invoke **your** PostgreSQL functions via **call templates** (`${loginname}`, `ARRAY['fixed'] || ${parent_role}`, etc.)
- Credentials are **not** stored in config — use `PGUSER` / `PGPASSWORD`, per-cluster connect user, `.pgpass`, or the run dialog

## Config file location

| OS | Path |
|----|------|
| macOS | `~/Library/Application Support/DbAccounts/config.yaml` |
| Linux | `~/.config/dbaccounts/config.yaml` |
| Windows | `%AppData%\DbAccounts\config.yaml` |

Copy [`config.example.yaml`](config.example.yaml) as a reference. The app creates a default config on first launch.

## Prerequisites

- Go 1.22+
- [Wails v2](https://wails.io/docs/gettingstarted/installation)
- Platform WebView dependencies (Xcode CLT on macOS, WebView2 on Windows, `webkit2gtk` on Linux)

## Version

Application version is defined in [`VERSION`](VERSION) (currently `0.1.0`). Git release tags use the `v` prefix: `v0.1.0`.

See [`RELEASING.md`](RELEASING.md) for bump, tag, and packaging steps.

## Build

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
cd /path/to/DbAccounts
go mod tidy
make package    # release build + dist/DbAccounts-v*-.tar.gz
```

Development:

```bash
wails dev
```

Other Makefile targets: `make test`, `make version`, `make build`, `make sync-wails-version`.

## Authentication

1. **User**: cluster `connect_user`, else `PGUSER`, else the Operations sidebar user field.
2. **Password**: Operations password field, else `PGPASSWORD`, else `~/.pgpass`, else **no password** (same as `psql` without `-W` — works with trust auth or empty password).

See [PostgreSQL .pgpass](https://www.postgresql.org/docs/current/libpq-pgpass.html).

## Database call templates

Each operation has a **call template** and **execution mode** in **Settings** or `db_functions.<operation>` in config (`execution`: `function` | `statement` | `block`).

Use **statement** (or **block**) when the template is raw SQL such as DDL/GRANT — PostgreSQL cannot bind role names as `$1`, and the app must embed them as identifiers:

- Remove role: `DROP ROLE ${loginname}`
- Grant parents: `GRANT ${parent_roles} TO ${loginname}`
- Revoke parents: `REVOKE ${parent_roles} FROM ${loginname}` (comma-separated parent roles as unquoted identifiers)

Example (create role):

```text
admin_access.create_role(${loginname}, NULL, ${fullname}, ${email}, ARRAY['gr_personal_users', 'gr_personal_users_ldap'] || ${parent_role})
```

- `${loginname}`, `${fullname}`, `${email}` — from the form, bound as `$1`, `$2`, …
- `NULL` — SQL literal (unused argument).
- `ARRAY['gr_personal_users', 'gr_personal_users_ldap'] || ${parent_role}` — fixed groups in config; optional parent role from form (`|| NULL` when empty).

Full syntax, whitelist, YAML examples, and common mistakes: [`sql/README.md`](sql/README.md).

## Safety

- Production clusters use a distinct badge colour.
- Runs touching **production** require the confirmation checkbox and an extra confirm dialog.
- **Remove role** asks for confirmation before execution.

## Tests

```bash
make test          # or: go test ./... -count=1
```

CI runs the same on every push/PR (`.github/workflows/test.yml`). Run tests before committing — they catch import cycles and compile errors that only show up in `_test.go` packages.

Covers call-template SQL generation for all four operations, command validation/args, config migration, and batch target resolution (DB calls fail without a live server).

## Manual test checklist

- [ ] First launch creates config file at the OS path above
- [ ] Add/edit/delete clusters; assign production vs UAT
- [ ] Import from environment (`PGHOST`, `PGPORT`, `PGDATABASE`, `PGUSER`)
- [ ] Save call templates + execution mode in Settings (create role: `function` + `ARRAY['...'] || ${parent_role}`; optional `statement` for DROP ROLE / GRANT)
- [ ] Test connection on one cluster (`.pgpass` and prompted password)
- [ ] Preview target count when toggling categories/clusters
- [ ] Run each operation against a single UAT cluster
- [ ] Run against multiple clusters; verify per-row results and timing
- [ ] Production run blocked without checkbox; succeeds with checkbox + confirm
- [ ] One failing cluster does not prevent others from completing

## Project layout

```
main.go, app.go          Wails entry and bindings
internal/config/         YAML persistence
internal/calltemplate/   Template parse/build (function / statement / block)
internal/pg/             Connections, .pgpass, ExecuteOperation
internal/batch/          Concurrent batch runner
internal/commands/       Operation validation
frontend/                Web UI
```

## License

Use and modify as needed for your organization.
