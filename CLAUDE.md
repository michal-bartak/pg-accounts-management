# CLAUDE.md — DbAccounts

AI/agent guide for this repo. Human docs: [README.md](README.md) (features/usage),
[sql/README.md](sql/README.md) (call-template DSL), [RELEASING.md](RELEASING.md).
A parallel Cursor rules file lives at
[.cursor/rules/dbaccounts-project.mdc](.cursor/rules/dbaccounts-project.mdc); keep the
two consistent when documenting architecture changes.

## What this is

Cross-platform desktop app (**Wails v2** + embedded WebView, vanilla HTML/CSS/JS
frontend, **pgx/v5** backend) for maintaining **PostgreSQL roles across many
clusters**. Real DDL/privilege changes are performed by **user-defined SQL call
templates** run against each cluster — the app does not hardcode DDL. Module:
`github.com/michalbartak/dbaccounts`. Current version: see [VERSION](VERSION) (`0.3.0`).

## Product decisions (don't regress)

- **Secrets never in config, and no in-UI credential fields.** Auth resolution: user =
  cluster `connect_user` → `PGUSER`; password = `PGPASSWORD` → `~/.pgpass` → empty
  (trust, like psql). `getAuth()` returns empty strings. See
  [internal/pg/auth.go](internal/pg/auth.go).
- **Production gate is a per-group `confirm` flag**, surfaced only as the confirm popup
  (`askConfirm('Production', …)`); no checkbox. `commands.RequiresProductionConfirm`
  and the frontend `hasProductionTargets`/`categoryConfirm` read the flag (not the id
  `"production"`). The frontend sends `confirmProduction: true` after the dialog.
- **Most writes go through configurable call templates** (`db_functions.<op>`) via the
  single bound `RunOperation`. Exceptions with **no template** (hardcoded SQL in
  `pg.ExecuteOperation`): `set_config` (`ALTER ROLE … SET x = '…'`) and `reset_config`
  (`ALTER ROLE … RESET x`) for role GUCs.
- **Categories = cluster groups**, edited from the **Clusters** tab (a right-aligned
  *Cluster groups* toolbar button opens the `#groups-dialog` list popup; *Add group*
  opens the `#group-dialog` form — label + base `color` + `confirm`); CRUD via
  `AddCategory/UpdateCategory/DeleteCategory` (id is a slug of the
  label, immutable; delete blocked while a cluster uses it). Group colours are applied
  by a generated `<style id="cat-colors">` keyed on `data-cat`; there are no hardcoded
  production/uat colours.
- **Preconfigured parent groups** (`Config.ParentRoles`, YAML `parent_roles`) are a
  Settings-managed list of bare-identifier role names, saved via `SaveParentRoles`
  (`Store.UpdateParentRoles` validates identifier syntax, dedupes). They are offered as
  pick-list choices (toggle **chips**) in both the Create-role and Alter-role
  Add-privilege dialogs, several at once. Create-role's `parent_role` placeholder accepts
  a **comma-separated** value — `calltemplate` splits it into multiple `text[]` elements
  in the `ARRAY[...] || ${parent_role}` path.
- **Role comments are format-agnostic** (plain text **or** arbitrary JSON — no forced keys).
  The role form's inline comment editor (`#role-comment-editor`) has a **Fields ↔ Raw**
  toggle: *Fields* edits string values only (never adds/removes keys); *Raw* edits the whole
  comment as free text. Which JSON keys get friendly labels is a Settings-managed list
  **`Config.CommentFields`** (YAML `comment_fields`, ordered `{key,label}`), defaulting to
  `full_name→Full name`, `e_mail→Email`, saved via `SaveCommentFields`
  (`Store.UpdateCommentFields` validates identifier keys, dedupes, defaults blank labels).
  Configured fields always render; **every other key** in the comment also renders (labeled by
  raw key) — string values are editable, non-string values (number/bool/array/object) render
  **read-only** (shown as JSON, edited via Raw; `e.readonly`) so their type is preserved via
  `baseObj`. `editorFromComment`
  builds the model (Fields for JSON, Raw for non-JSON content, and for an **empty** comment
  the configured **`ui.comment_default_view`** — `fields`|`raw`, `preferredCommentView()`);
  `assembleComment`/`assembleCommentFrom` serialize it (empty value drops the key; preserves
  unknown keys); `parseCommentObject` is the shared reader; `switchEditorMode` round-trips
  Fields↔Raw. A non-JSON comment saves as plain text with a toast warning. The backend
  `set_comment` op stays an opaque quoted literal; `pg.ParseFullName` (`full_name`) is kept
  only for search-result display. When comments **vary** across clusters the inline editor is
  hidden and reconciliation moves to the **Comments dialog**, whose per-version boxes reuse the
  same Fields/Raw editor (`commentVersionEditors`). The dialog **stages** edits locally: **OK**
  commits each version's assembled comment into `commentOverrides` (`Map<clusterId,string>`);
  **Cancel** discards. Nothing is sent from the dialog — staged comments publish with the rest
  of the edits on **Save changes** (`buildAlterRequests` prefers a cluster's override, else the
  inline editor's comment). `commentOverrides` is cleared by `resetEditMaps` (pick / save / mode switch).
- **One shared role form for Create and Alter.** Both modes render through
  `renderAlterDetail` over `alterDetails` and the same edit maps (`alterAdd`/`alterRevoke`,
  `alterAttrAdd`/`alterAttrRemove`, `alterConfigSet`/`alterConfigReset`, password). Mode =
  `isCreateMode()` (`currentOp === 'create_role'`). **Create** = editing a not-yet-existing
  role over the selected clusters with an **empty synthetic baseline**
  (`synthCreateBaseline()` builds `alterDetails` rows `{exists:false, parents:[], …}` from
  `resolveSelectedClusters()`); every edit is therefore a pure grant/enable/set. **Alter** =
  search → `pickUser` → `reloadDetails` loads the real per-cluster baseline. The **static**
  `#role-identity` block above `#alter-detail` holds login (editable+required in create,
  readonly in edit) plus an inline **comment editor** (`#role-comment-editor`) — see the
  comment-handling decision below. The op-tab buttons are mode switches: Create resets to an
  empty form; Alter opens the search popup and enters edit only on pick. `updateTargetPreview`
  re-synthesizes + reconciles pending edits when the selection changes in create mode
  (`reconcilePendingWithUniverse`).
- **Run/build.** Create: `runOperation` validates the login, pre-flight-warns if it already
  exists on a selected cluster (`LoadRoleDetails`), then runs
  `buildCreateRoleRequests(base)` (one `create_role` per cluster, **empty** `parent_role` →
  template base groups only) **concatenated** with `buildAlterRequests()` — creates run
  first (sequential `executeAlterRequests`), then the grant/attr/setting/password ops.
  The comment persists via **`set_comment`** in both modes: `buildAlterRequests` emits
  `set_comment` per cluster where the desired comment changes — a per-cluster `commentOverrides`
  entry (staged in the Comments dialog, varies case) wins, else the inline editor's
  `assembleComment()`. **Create** runs `create_role` first (its `${fullname}`/`${email}` sourced
  best-effort from the editor's `full_name`/`e_mail` values), then the follow-up `set_comment` is
  authoritative. Everything (grants/attrs/settings/password/comments) publishes together on the
  one **Save changes** pass.
- Frontend is **vanilla JS** reached through `window.go.main.App` (`backend()` in
  [frontend/app.js](frontend/app.js)). No framework. `frontend/wailsjs/` is
  Wails-generated (regenerated by `wails dev`/`build`); a running `wails dev` watcher
  keeps it in sync, but hand-edit it only to stay consistent between builds.

## Two data paths

1. **Write (fire-and-forget), template-driven.** `App.RunOperation(RunRequest)` →
   [internal/batch/runner.go](internal/batch/runner.go) `Run` → `commands.ValidateRequest`
   → `ResolveClusters` → `commands.BuildArgs` → per-cluster `pg.Connect` +
   `pg.CallFunction` → `calltemplate.Build`. Concurrency bounded by
   `batch.max_concurrency` (default 5), 30s per-cluster timeout, production gate.
2. **Read (introspection), catalog queries — added for "Alter role".**
   [internal/pg/introspect.go](internal/pg/introspect.go): `SearchRoles` (matches
   `rolname` or `COMMENT ON ROLE` via `pg_shdescription`), `RoleDetail` (existence,
   comment, attribute flags, parent memberships from `pg_auth_members`, and role GUCs
   from `pg_roles.rolconfig` → `Settings` map), `ParseFullName` (JSON comment
   `full_name`), `likePattern` (escaped ILIKE). `batch.Runner.SearchRoles` /
   `LoadRoleDetails` fan out over the **resolved selected clusters** (not all) and
   collect per-cluster errors instead of failing.

## Bound methods (`app.go` → `window.go.main.App`)

Config/clusters/groups: `GetConfig`, `GetConfigPath`, `ReloadConfig`, `AddCluster`,
`UpdateCluster`, `DeleteCluster`, `AddCategory`, `UpdateCategory`, `DeleteCategory`,
`ImportFromEnvironment`, `SaveDBFunctions`, `SaveBatchSettings`, `SaveUISettings`,
`SaveParentRoles`, `SaveCommentFields`, `GetAppVersion`.
Run/test: `TestConnection` (by saved cluster id), `TestConnectionInput` (ad-hoc
`ClusterInput`+`Auth`, used by the cluster editor to test on-screen values),
`PreviewTargets`, `RunOperation`.
Introspection (Alter role): `SearchRoles(RoleSearchRequest)`,
`LoadRoleDetails(RoleDetailsRequest)`.

## Operations (call templates, `internal/calltemplate/`)

`db_functions.<op>.call` + optional `.execution` (`function` | `statement` | `block`).
Defaults in [internal/config/store.go](internal/config/store.go); example in
[config.example.yaml](config.example.yaml); DSL in [sql/README.md](sql/README.md).

| op | placeholders | default execution |
|----|--------------|-------------------|
| `create_role` | loginname, fullname, email, parent_role | function (`ARRAY[] \|\| ${parent_role}`) |
| `remove_role` | loginname (rolename alias) | function |
| `grant_parents` | loginname, parent_roles | function |
| `revoke_parents` | loginname, parent_roles | statement (`REVOKE … FROM …`) |
| `change_password` | loginname, new_password | function |
| `set_comment` | loginname, **comment** | statement (`COMMENT ON ROLE ${loginname} IS ${comment}`) |
| `set_attribute` | loginname, **attribute** | statement (`ALTER ROLE ${loginname} WITH ${attribute}`) |
| `set_config` / `reset_config` | loginname, config_name(, config_value) | **no template** — hardcoded `ALTER ROLE … SET/RESET` in `pg.ExecuteOperation` |

Field kinds when embedding (statement/block): role names → **identifiers** (unquoted,
validated `[A-Za-z_][A-Za-z0-9_]*`); `parent_roles` → comma-separated identifier list;
`new_password`/`comment`/`fullname`/`email` → quoted **literals**; `attribute` → an
identifier-style **whitelisted keyword** (`SUPERUSER`/`NOSUPERUSER`, `CREATEROLE`/…,
`CREATEDB`, `INHERIT`, `LOGIN`, `REPLICATION`, `BYPASSRLS`) validated in
`commands.ValidateRequest`.

### Adding a new operation
Extend all of: `calltemplate.AllowedPlaceholders` + `placeholderKindForField`;
`commands` op const + `BuildArgs` + `ValidateRequest`; `model.DBFunctions` +
`*Params` + `RunRequest`; `config.store.DefaultConfig` + `dbfunctions.go`
migrate/validate lists; the `DB_FUNCTIONS` table in `frontend/app.js` (drives the compact
command list + the `#fn-dialog` popup, incl. its allowed-placeholder chips) and
`readDBFunctionsFromEditor`; example config; and tests in `calltemplate`/`commands`/`config`.

## Frontend (`frontend/`)

**App shell**: header + tabs bar are fixed (`body` is a flex column, `overflow:hidden`,
no page scroll); `main` fills the rest. Each panel manages its own scroll. Operations is
a two-column grid — left `.ops-sidebar` and right `.ops-main` scroll **independently**;
`.ops-main` is a flex column with a scrolling `.ops-body` (no horizontal padding, so its
content lines up with the footer buttons) and a pinned `.ops-footer` (**Create role** button
for create; Save changes / Remove role for alter, toggled by `updateOpsFooter()`). Clusters
and Settings are single-column with a fixed toolbar / pinned Save-settings footer.
**Action-button convention (keep consistent):** the primary/commit button is the emphasized
(`.primary`) one, placed **rightmost**; Cancel/secondary sits to its left; destructive actions
(e.g. Remove role) are separated on the **far left**. This holds for dialog `<menu>`s
(`[Cancel] [Primary]`, all right-aligned) and page footers (Create role / Save changes /
Save settings right-aligned; Remove role far left via `#btn-alter-remove{order:-1}` +
`space-between`). The **Test connections** button lives in the Clusters
toolbar (`btn-test-clusters` → `testAllClusters`): it tests every configured cluster and
writes the outcome into a per-row **Status** column (`setClusterStatus`). Cluster rows have
no per-row Test button (testing on-screen values is done from the cluster editor via
`TestConnectionInput`); each row's **Actions** cell holds right-aligned **✎ edit** / **× delete**
icon buttons (`.scope-act`, same as the role form) in a `.row-actions` flex.

Top tabs **Operations / Clusters / Settings**, with **Create role** / **Alter role**
op-tabs right-aligned in the same bar (shown only while Operations is active; toggled in
the `.tab` click handler). The two op-tabs drive **one shared `#role-form`** (there is no
`#form-create_role` / `#form-alter_user` split): Create resets it to an empty form; Alter
opens the search popup and fills it on pick — see the shared-role-form product decision
above. The left sidebar is **Target selection** only (no connection or
confirm-production controls); "Or pick clusters" is a collapse/expand toggle (collapsed
by default; the expanded `.cluster-list` flexes to fill the sidebar). The Clusters
toolbar hosts the right-aligned **Cluster groups** button (`btn-manage-groups` →
`#groups-dialog` list popup → `#group-dialog` add/edit form; edited like clusters, no
Save button, so it lives here rather than Settings). Settings is organised into
divider-separated **`.settings-group`** sections (small uppercase `.settings-group-label`,
same look as the role form): **General** (Appearance theme + Max concurrency), **Preconfigured
parent groups**, **Comments** (Comment fields + Preferred comment view), and **DB command
templates**. Parent groups and Comment fields are drag-orderable add/remove **list editors**
built from the shared `listRowHtml`/`wireListEditor` helpers (drag handle + remove ×, `Add…`
button), staged in `parentRolesDraft` / `commentFieldsDraft` (`#parent-roles-editor` /
`#comment-fields-editor`). DB templates are a compact list of command names; clicking one opens
the `#fn-dialog` popup (execution type, call template, clickable placeholder chips — staged in
`dbFnDraft`). The **Preferred comment view** toggle is `#comment-view-pref`
(`ui.comment_default_view`). All staged on **Save settings**.

Feature descriptions are not shown inline — they live behind a **`?` help badge**
(`.q-hint`; markup via `hintBadge(text)` in JS, or hand-written next to a static heading).
Hovering/focusing the badge reveals the text in a single shared, `position:fixed`
popover (`.q-hint-pop`) that is positioned in JS (centred, viewport-clamped, flips above
when it would overflow) so `overflow:hidden` panels never clip it. One delegated
mouseover/focusin handler drives it, so JS-rendered badges (e.g. the Alter-role sections)
work without per-element wiring. Used on Privileges/Attributes/Settings (Alter role),
and the Cluster-groups / Find-role / Comments dialogs.

**Role form flow** (all in [frontend/app.js](frontend/app.js), styles in
[frontend/styles.css](frontend/styles.css)) — shared by Create and Alter:
- Clicking the **Alter role** op-tab opens the search dialog (`openSearchDialog`; there
  is no separate "Find user" button); results grouped by login; picking one loads the
  shared form. Search + detail load are **restricted to the selected clusters**
  (`alterTargets`/`alterScopeClusters` captured at search time). Clicking **Create role**
  resets the form to an empty synthetic baseline over the selected clusters.
- **Privileges** (parent roles) and **Attributes** (superuser/createrole/createdb/
  inherit/login/replication/bypassrls) render one-per-row: name left, **scope labels**
  right. Completeness is judged **per group**: all selected clusters of a group matched
  → one **outlined** (bordered, transparent) uppercase group label — matching the bordered
  group boxes in Target selection; otherwise one **filled** (no border) per-cluster label,
  coloured by environment. `describeScope`/`scopeLabelsHtml` produce these everywhere
  (present-on, rows, comments, search results).
- Edit model: `alterAdd`/`alterRevoke` and `alterAttrAdd`/`alterAttrRemove` are
  `Map<key, Set<clusterId>>`. **✎ Edit** opens a per-cluster checkbox editor that both
  grants and revokes (diff of desired vs current); **×** removes everywhere; **↺**
  discards pending. All three action buttons always render (stable row layout); the
  inapplicable ones are `disabled` (greyed, `pointer-events:none`) rather than hidden —
  `×` when nothing is granted/pending, `↺` when there are no pending changes.
  `buildAlterRequests` emits one `RunOperation` per (cluster, op);
  `is-added` (green) applies only to a pending grant, not to attributes that are simply
  off.
- **Settings** (role GUCs) reuse the same row/scope-editor as attributes but keyed by
  `name=value` (a name with different values per cluster shows multiple rows); pending
  state is `alterConfigSet: Map<"name=value", Set<clusterId>>` and
  `alterConfigReset: Map<name, Set<clusterId>>`; `buildAlterRequests` emits
  `set_config`/`reset_config`. The scope dialog gains name+value inputs for a new setting.
- **Comment** editor (`#role-comment-editor`, in `#role-identity`) — see the format-agnostic
  comment product decision above. State is the module-level `commentEditor` (rendered FROM /
  written TO it, so `renderAlterDetail` never clobbers pending edits, like the old identity
  block). Loaded on mode entry via `loadCommentEditor()`. When comments **vary** across
  clusters the inline editor is **hidden** (only then does `renderAlterDetail` render a
  "Comments differ — reconcile per cluster" section, flagged *(edited)* when overrides are
  staged); the **Comments** popup (grouped by `canonicalComment`) reconciles per cluster, each
  version editable via its own Fields/Raw editor (`commentVersionEditors`). It has no per-row
  save — **OK** stages edits into `commentOverrides` (`commitCommentsDialog`), **Cancel**
  discards, and the staged comments publish with everything else on **Save changes**.
- **Password** row (field + checkbox) and a red **Remove role** button; results render
  in the **Status** panel (hidden until non-empty).

## Layout

```
main.go, app.go           Wails entry + bound methods
internal/model/           Shared JSON-tagged types + RunRequest (stdlib only)
internal/calltemplate/    Template parse/validate/SQL build (stdlib only)
internal/config/          YAML persistence, DBFunction migrate/validate
internal/pg/              DSN, auth, Connect, CallFunction, introspect.go (reads)
internal/batch/           Concurrent executor + all-cluster scan
internal/commands/        Op validation + arg maps + attribute keyword whitelist
internal/envimport/       PG* env import
frontend/                 Vanilla JS UI (app.js via backend())
sql/README.md             Docs for the user's PostgreSQL functions / templates
```

## Package import rules (avoid cycles)

| Package | May import | Must not import |
|---------|-----------|-----------------|
| `internal/model` | stdlib | other `internal/*` |
| `internal/calltemplate` | stdlib | `config`, `pg`, `commands`, `batch` |
| `internal/config` | `model`, `calltemplate` | `pg` |
| `internal/pg` | `model`, `calltemplate`, pgx | `config` |
| `internal/commands` | `model`, `config` | — |
| `internal/batch` | `model`, `config`, `commands`, `pg` | — |

`internal/pg` **tests must not import `internal/config`** (config → calltemplate ← pg
test → config cycle). Test SQL via `calltemplate` alone or with `commands`.

## Build / test / run

```bash
export PATH="/opt/homebrew/bin:$HOME/go/bin:$PATH"
go build ./... && go test ./... -count=1   # or: make test-vet
wails dev            # dev window (regenerates frontend/wailsjs/)
make package         # build/bin/DbAccounts.app + dist/*.tar.gz
```

- Run `go test ./... -count=1` and `node --check frontend/app.js` before committing.
  CI: `.github/workflows/test.yml`.
- Frontend has **no build step / no package.json**; served statically by Wails. For a
  quick UI smoke test without the Go backend, `python3 -m http.server` in `frontend/`
  and exercise the global functions (`backend()` is undefined, so search/save toast an
  error, but rendering/logic can be driven with mock `state`/`alterDetails`).
- Introspection/write SQL is best verified against a throwaway Postgres
  (`docker run … postgres:16`), calling `pg.RoleDetail` / `pg.ExecuteOperation`
  directly; remove any scratch `_test.go` afterward.

## Versioning

[VERSION](VERSION) is app semver (git tag `v$(cat VERSION)`). `internal/version`
defaults must match; `make sync-wails-version` aligns `wails.json` `productVersion`;
`GetAppVersion()` surfaces it. Config YAML `version:` is the **schema** version only.

## Out of scope (v1)

SSH tunnels, encrypted config vault, audit log, reading remote `pg_hba.conf`.
