# DbAccounts

Cross-platform desktop app for maintaining PostgreSQL roles across many clusters. Built with **Go**, **Wails v2** (WebView), and **pgx**.

## Features

- Manage PostgreSQL clusters (alias, host, port, database, group)
- **Cluster groups** (default Production / UAT, editable in Settings): each has a base **colour** and a **require-confirmation** flag that gates production-style runs behind a popup
- **Create role** in batch against selected groups and/or clusters (login, full name, email, parent role)
- **Alter role** — find a role from a search popup (matched on role name and comment across the **selected** clusters/groups), then edit its whole identity in **one form**:
  - Privileges (parent-role memberships), role **attributes** (superuser, create role, create DB, inherit, login, replication, bypass RLS), and role **settings** (GUCs like `statement_timeout`, `log_statement`) listed per row and labelled by scope (a filled group label like `PRODUCTION` when a whole group matches, else individual cluster aliases)
  - Add/edit on any mix of **groups and clusters** via a scope picker; settings support per-cluster values
  - View and consolidate **comments** (grouped by identical content) in a popup
  - Change password and remove role
  - Changes are committed together with **Save**
- Operations invoke **your** PostgreSQL functions via **call templates** (`${loginname}`, `ARRAY['fixed'] || ${parent_role}`, etc.)
- Credentials are **not** stored in config — use `PGUSER` / `PGPASSWORD`, per-cluster connect user, or `.pgpass`

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

Application version is defined in [`VERSION`](VERSION) (currently `0.3.0`). Git release tags use the `v` prefix: `v0.3.0`.

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

## Installing releases

Pre-built binaries are on [GitHub Releases](https://github.com/michal-bartak/pg-accounts-management/releases). CI builds are **not** signed with an Apple or Microsoft developer certificate, so the first launch may show a security warning.

**macOS** — extract `DbAccounts.app` from the `.tar.gz`, then **right-click** the app → **Open** → confirm **Open**. Alternatively: **System Settings → Privacy & Security → Open Anyway**, or in Terminal:

```bash
xattr -dr com.apple.quarantine /path/to/DbAccounts.app
open /path/to/DbAccounts.app
```

**Windows** — if SmartScreen blocks the `.exe`, click **More info** → **Run anyway**.

**Linux** — extract the tarball and run `./DbAccounts`; install `libgtk-3-0` and `libwebkit2gtk-4.1-0` if the app fails to start.

See [`RELEASING.md`](RELEASING.md) for maintainers.

## Authentication

1. **User**: cluster `connect_user`, else `PGUSER`.
2. **Password**: `PGPASSWORD`, else `~/.pgpass`, else **no password** (same as `psql` without `-W` — works with trust auth or empty password).

The app has no in-UI user/password fields — credentials come from the cluster's
`connect_user`, the environment, or `~/.pgpass`. (The Clusters *Test connection*
action still prompts for a one-off password.) See
[PostgreSQL .pgpass](https://www.postgresql.org/docs/current/libpq-pgpass.html).

## Database call templates

Each operation has a **call template** and **execution mode** in **Settings** or `db_functions.<operation>` in config (`execution`: `function` | `statement` | `block`).
The **defaults are vanilla PostgreSQL DDL** (`CREATE ROLE`, `DROP ROLE`, `GRANT`/`REVOKE`,
`ALTER ROLE … PASSWORD`/`WITH`/`SET`/`RESET`, `COMMENT ON ROLE`), all in **statement** mode.
Override any of them — e.g. point an operation at a `SECURITY DEFINER` wrapper function (see the
function example below) so a low-privilege user can act through it, or to add audit logging /
notifications. Each template's dialog has a **Default** button that reverts it to the vanilla built-in.

Use **statement** (or **block**) when the template is raw SQL such as DDL/GRANT — PostgreSQL cannot bind role names as `$1`, and the app must embed them as identifiers:

- Remove role: `DROP ROLE ${loginname}`
- Grant parents: `GRANT ${parent_roles} TO ${loginname}`
- Revoke parents: `REVOKE ${parent_roles} FROM ${loginname}` (comma-separated parent roles as unquoted identifiers)
- Set comment: `COMMENT ON ROLE ${loginname} IS ${comment}` (comment embedded as an escaped string literal)

Example of a **function**-mode override (create role) — the default is simply `CREATE ROLE ${loginname}`:

```text
admin_access.create_role(${loginname}, NULL, ${fullname}, ${email}, ARRAY['gr_personal_users', 'gr_personal_users_ldap'] || ${parent_role})
```

- `${loginname}`, `${fullname}`, `${email}` — from the form, bound as `$1`, `$2`, …
- `NULL` — SQL literal (unused argument).
- `ARRAY['gr_personal_users', 'gr_personal_users_ldap'] || ${parent_role}` — fixed groups in config; optional parent role from form (`|| NULL` when empty).

Full syntax, whitelist, YAML examples, and common mistakes: [`sql/README.md`](sql/README.md).

The **read** queries behind Alter-role search/detail are configurable too, under Settings →
**Introspection queries** (config `db_reads`). They are plain SQL taking a single `${rolename}` bind, with
their result columns scanned by name against a fixed contract — so you can point one at a
privileged wrapper function or view when the connect user cannot read the catalogs directly. See
[`sql/README.md`](sql/README.md#introspection-queries-reads).

## Alter role

The **Alter role** button (right side of the top tab bar, shown while Operations is
active) avoids mistyped role names by searching first.

1. Pick the clusters/groups to compare in **Target selection** (left sidebar), then
   click **Alter role** and enter a term (≥ 2 characters) in the search popup. Only the **selected**
   clusters are scanned (concurrently); the term is matched case-insensitively
   against the role name **and** the role's `COMMENT ON ROLE` (read from
   `pg_shdescription`). Unreachable clusters are reported but do not abort the
   search. Results are grouped by login name; if a role's comment is **JSON
   containing a `full_name` key**, that value is shown — e.g.
   `COMMENT ON ROLE alice IS '{"full_name":"Alice Example","email":"a@x.com"}'`.
2. Picking a user closes the popup and opens **one form** for the whole identity
   (not one per cluster). Identity is shown once, with a "varies across clusters"
   note when clusters disagree (comments are compared **as JSON by value**, so
   formatting/key-order differences don't count as a difference).
3. Everywhere clusters are listed (Present on, privileges, attributes, comments) the
   same **scope labels** are used, coloured by environment. Completeness is judged
   **per group** (relative to the selected clusters):
   - every selected cluster of a group matched → one filled, uppercase **group**
     label named after the group (e.g. `PRODUCTION`);
   - otherwise one **cluster** label (transparent, group-coloured) per matched
     cluster — e.g. `PRODUCTION  UAT LIVE` when it is on every production cluster
     plus one UAT cluster.
   **Privileges** (parent-role memberships from `pg_auth_members`) and role
   **attributes** (superuser, create role, create DB, inherit, login, replication,
   bypass RLS) are listed one per row — name on the left, scope labels on the right.
   Use **✎ Edit** to open a per-cluster checkbox editor that both **grants and
   revokes** on individual clusters/groups (pending grants show green, pending
   revokes struck through); **×** removes everywhere; **↺** discards pending changes;
   **Add privilege…** adds a new membership. Attribute changes run `ALTER ROLE … WITH
   SUPERUSER` / `… WITH NOSUPERUSER` via the `set_attribute` template.
   **Settings** (role GUCs from `pg_roles.rolconfig`) work the same way, keyed by
   `name = value`; **Add setting…** takes a name + value, and a setting can hold
   different values on different clusters. They run `ALTER ROLE … SET x = '…'`
   / `RESET x` via the configurable `set_config` / `reset_config` templates.
4. **Comments** — the *View / edit comments* popup groups clusters by comment
   content (JSON compared by value), labels each group by scope, and lets you edit a
   comment and save it to all its clusters (writes `COMMENT ON ROLE` via the
   `set_comment` template).
5. **Change password** applies to all clusters where the user exists. **Remove
   role** is a dedicated red button next to Save that removes the role on every
   cluster where it exists (with confirmation).
6. **Save** computes the per-cluster diff and applies every change, reusing the
   configured `grant_parents`, `revoke_parents`, and `change_password` templates.
   Per-cluster outcomes appear in the **Status** panel. Production clusters trigger a
   confirm dialog before anything runs.

## Safety

- Production clusters use a distinct badge colour.
- Runs touching **production** require confirmation via a popup dialog before executing.
- **Remove role** asks for confirmation before execution.

## Tests

```bash
make test          # or: go test ./... -count=1
```

CI runs the same on every push/PR (`.github/workflows/test.yml`). Run tests before committing — they catch import cycles and compile errors that only show up in `_test.go` packages.

Covers call-template SQL generation for all operations (including `set_comment` literal escaping), role-comment full-name parsing, command validation/args, config migration, and batch target resolution (DB calls fail without a live server).

## Manual test checklist

- [ ] First launch creates config file at the OS path above
- [ ] Add/edit/delete clusters; assign production vs UAT
- [ ] Import from environment (`PGHOST`, `PGPORT`, `PGDATABASE`, `PGUSER`)
- [ ] Save call templates + execution mode in Settings (create role: `function` + `ARRAY['...'] || ${parent_role}`; optional `statement` for DROP ROLE / GRANT)
- [ ] Test connection on one cluster (`.pgpass` and prompted password)
- [ ] Preview target count when toggling categories/clusters
- [ ] Create role against a single UAT cluster
- [ ] Alter role: search/compare is limited to the selected clusters/groups (Target selection); empty selection is blocked
- [ ] Alter role: Create role / Alter role buttons sit in the tab bar, visible only when Operations is active; Alter role opens the search popup
- [ ] Alter role: search by name and by comment finds the role, grouped by login; picking closes the popup
- [ ] Alter role: full name shown when comment is JSON with `full_name`
- [ ] Alter role: one consolidated form; scope labels judged per group (filled group label vs transparent per-cluster labels)
- [ ] Alter role: ✎ Edit opens a per-cluster editor that grants AND revokes on specific clusters/groups; whole-group match shows a filled group label, partial shows transparent cluster labels
- [ ] Alter role: Add privilege… adds a new membership; × removes everywhere; ↺ discards pending
- [ ] Alter role: comments popup groups by content (JSON compared by value — formatting differences collapse), labels coloured by environment, edit + save writes to that group's clusters
- [ ] Alter role: scope labels consistent (Present on / privileges / comments), coloured by environment, ordered by group
- [ ] Alter role: attributes (superuser/createrole/createdb/inherit/login/replication/bypassrls) shown per row with scope; ＋ enables, × disables via set_attribute
- [ ] Alter role: privileges/attributes listed one per row (name left, scope labels right); search results use the same scope labels
- [ ] Alter role: change password via Save (field + checkbox stacked under PASSWORD); Remove role via the red button; Status panel hidden until a run produces output
- [ ] Alter role: one unreachable cluster is reported but search still returns others
- [ ] Production run/save blocked without checkbox; succeeds with checkbox + confirm
- [ ] One failing cluster does not prevent others from completing

## Project layout

```
main.go, app.go          Wails entry and bindings
internal/config/         YAML persistence
internal/calltemplate/   Template parse/build (function / statement / block)
internal/pg/             Connections, .pgpass, ExecuteOperation, role introspection
internal/batch/          Concurrent batch runner
internal/commands/       Operation validation
frontend/                Web UI
```

## License

Use and modify as needed for your organization.
