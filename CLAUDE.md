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
- **Most writes go through configurable call templates** (`db_functions.<op>`) executed by
  `pg.ExecuteOperation`, driven by the batched `RunRoleBatch` (per-cluster transaction; see the
  two-data-paths section). Exceptions with **no template** (hardcoded SQL in `pg.ExecuteOperation`):
  `set_config` (`ALTER ROLE … SET x = '…'`) and `reset_config` (`ALTER ROLE … RESET x`) for role GUCs.
- **Categories = cluster groups**, edited from the **Clusters** tab (a right-aligned
  *Cluster groups* toolbar button opens the `#groups-dialog` list popup; *Add group*
  opens the `#group-dialog` form — label + base `color` + `confirm`). Group id is a slug of the
  label, immutable on edit; delete blocked while a cluster uses it. Group colours are applied
  by a generated `<style id="cat-colors">` keyed on `data-cat`; there are no hardcoded
  production/uat colours.
- **The Clusters tab is STAGED** (like Settings): add/edit/delete of clusters *and* groups mutate
  in-memory drafts (`clustersDraft`/`categoriesDraft`, seeded once from saved `state` in
  `loadConfig`, kept across other `loadConfig` calls), and nothing persists until the footer
  **Save** (`btn-save-clusters` → `App.SaveClusters(ClustersConfig)` →
  `Store.SaveClustersAndCategories`, an atomic validate-and-replace: category label required /
  slug id / no dup ids, cluster fields validated + port/sslmode defaulted + UUID minted for new,
  and referential integrity — every `cluster.category` must exist). **Discard** reverts drafts to
  saved. Save is **enabled only when dirty, disabled (inert) when clean**
  (`clustersDirty`/`refreshClustersDirty` → `setDirty`). New
  draft clusters carry a `tmp_<n>` id (sent as `""` so the backend mints a UUID); new groups get
  their slug id immediately so clusters can reference them pre-save. **Only the Clusters editor
  reads the drafts** — Operations target selection / run resolution keep reading saved `state`, so
  unsaved cluster edits never affect what a run targets. `Test connections` tests the on-screen
  draft values via `TestConnectionInput`, writing per-row Status.
- **No corner toast — feedback goes to the action button or inline.** There is no `showToast`.
  Simple confirmations **flash the action button** ("Saved"/"Created", green, ~1.2 s via
  `flashButton`); Save buttons are **enabled only when there are changes, disabled (inert) when
  clean** — no marker (`setDirty` toggles `disabled`; disabled buttons are `pointer-events:none`
  so they don't react to hover) — Settings (`btn-save-settings`, `settingsDirty`), the role form
  (`btn-alter-save`, dirty = `buildAlterClusterOps().length>0`, in `updateOpsFooter`), and the
  Clusters Save. All right-aligned action buttons + the op-tabs share one right margin (the
  `.tabs`, `.ops-footer`, and Settings/Clusters footers all align to the 1.25rem panel edge). Errors/validation render **inline** in a `.form-error` next to the control
  (`showInlineError`/`clearInlineError`) — `#ops-error` (Operations footer), `#settings-error`,
  `#clusters-error`, `#scope-error`, `#group-error`, `#cluster-test-error`, `#alter-search-errors`
  — plus a red button flash. Run/batch outcomes stay in the **run-status chip** (unchanged); rare
  clipboard failures log to the console.
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
  `baseObj`. A **`null`** value is the exception: it is treated as an empty **editable** string
  field (no read-only / no ⚠), and on save an empty value for a key that was already in the loaded
  comment serializes back as JSON **`null`** (so a loaded null round-trips and clearing a field
  stores null); a key that was never in the comment stays absent, so an all-blank/empty comment
  still assembles to `''` (load stays idempotent — opening a role never marks it dirty).
  `editorFromComment`
  builds the model (Fields for JSON, Raw for non-JSON content, and for an **empty** comment
  the configured **`ui.comment_default_view`** — `fields`|`raw`, `preferredCommentView()`);
  `assembleComment`/`assembleCommentFrom` serialize it (empty value → null for existing keys, else
  drops the key; preserves
  unknown keys); `parseCommentObject` is the shared reader; `switchEditorMode` round-trips
  Fields↔Raw. The **Fields** toggle is disabled whenever the raw text is non-empty and not a
  JSON object (`commentFieldsBlocked`) — Fields can't represent plain text, so switching would
  drop it; edit such comments in Raw. A non-JSON comment saves as plain text with an inline note. The backend
  `set_comment` op stays an opaque quoted literal; `pg.ParseFullName` (`full_name`) is kept
  only for search-result display. The comment UI **mode follows the staged state**
  (`commentEditor.varies`), not the DB baseline: while comments vary the inline editor is hidden and
  reconciliation moves to the **Comments dialog**, whose per-version boxes reuse the
  same Fields/Raw editor (`commentVersionEditors`). Each version box (when there is >1) also has a
  **Use in all clusters** button (left of the Fields/Raw toggle, `data-cv-useall`) that broadcasts
  that version's comment to every version editor. The dialog **stages** edits locally: **OK**
  (`commitCommentsDialog`) commits each version's assembled comment into `commentOverrides`
  (`Map<clusterId,string>`); **Cancel** discards. **OK also folds a now-consistent result**: if the
  reconciled comments are all canonically equal (via Use-in-all or manual editing), it loads that one
  comment into the inline editor (`commentEditor.varies=false`) and clears `commentOverrides`, so the
  "Comments differ" banner clears on **OK** (before Save) and the inline editor takes over. Nothing is
  sent from the dialog — staged comments publish with the rest of the edits on **Save changes**
  (`buildAlterClusterOps` prefers a cluster's override, else the inline editor's comment).
  `commentOverrides` is cleared by `resetEditMaps` (pick / save / mode switch).
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
- **Per-cluster presence editor (Alter).** The static **Present on** block (`#role-present`, a
  sibling of `#alter-detail` with its own click listener) shows existing clusters (plain chips),
  pending-**add** (green) and pending-**remove** (red strikethrough) plus a **✎** button
  (`#btn-present-edit`) that opens the scope dialog with `ctx.kind === 'presence'`. Adding a
  cluster inserts a **synthetic `exists:false` row** into `alterDetails` (empty baseline, like
  `synthCreateBaseline`) so the whole form (privileges/attributes/settings/comment editors) targets
  it; dropping a cluster records it in **`roleRemoveClusters`** (`Set<clusterId>`, reset in
  `resetEditMaps`). The presence editor's universe is `alterScopeClusters` (the searched scope, so a
  cluster must have been in Target selection at search time to be addable); `confirmPresenceScope`
  recomputes remove/add sets from the real `exists:true` rows. `buildAlterClusterOps` emits a lone
  `remove_role` for removed clusters and **prepends `create_role`** for `exists:false` rows *in
  Alter mode only* (`!isCreateMode()`, so Create's `buildCreateClusterOps` doesn't double-prepend);
  the consistent inline comment then flows to new clusters as a `set_comment` (varies ⇒ created
  bare). All of it publishes on the one **Save changes** pass. `removeRole` (red button) targets
  only `exists:true` rows.
- **Run/build.** Both modes build **per-cluster ordered op lists** and send ONE
  `app.RunRoleBatch({clusters, auth, confirmProduction})` via `executeRoleBatch`; the backend runs
  each cluster's ops as a single transaction. `buildAlterClusterOps()` produces
  `[{clusterId, operations:[{operation, <paramKey>:{…}}]}]` (order per cluster: grant → revoke →
  password → **attributes (all keywords combined into one `set_attribute`)** → set_config →
  reset_config → set_comment). **Create**: `runOperation` validates the login, pre-flight-warns via
  `LoadRoleDetails`, then `buildCreateClusterOps(base)` prepends a `create_role` op (empty
  `parent_role`; `${fullname}`/`${email}` sourced best-effort from the editor) to each cluster's
  diff. **Update**: `saveAlterations` uses `buildAlterClusterOps()`. `removeRole` sends one
  `remove_role` op per cluster. The comment persists via **`set_comment`** (per-cluster
  `commentOverrides` wins, else the inline editor's `assembleComment()`). Everything publishes
  together, atomically per cluster, on the one **Save changes** / **Create role** pass. Progress
  shows **live** in a footer status chip (`#run-status`) fed by `role-batch-progress` Wails events
  (one per cluster start/finish); clicking it opens `#run-status-dialog` with one row per cluster.
  **Errors never mutate the role form.** `executeRoleBatch` returns the results array (or `null`
  when blocked/cancelled/threw); on `null` **or any per-cluster failure** the Save/Create/Remove
  paths leave the form and pending edits untouched (chip + popup are the sole error surface). Only
  a **fully clean** outcome refreshes the form — Save re-reads the baseline via `fetchRoleDetails`
  and adopts it *only* when the reload itself is clean (no empty/error, so a transient unreachable
  cluster can't wipe it). On a **fully successful** Save, though, the comment we sent is known-live,
  so `saveAlterations` **optimistically reconciles the local comment baseline** (from
  `commentOverrides` / the inline editor) and clears the staged overrides *before* the reload, then
  always re-renders — so the "Comments differ" banner reflects what was written even when a cluster
  is momentarily unreachable on reload (a clean reload still overrides with DB truth + resets the
  other edit maps). Create resets to an empty form; Remove calls `reloadDetails` (empty-state
  reset). The genuine "not found" state still renders for an actual search `pickUser`.
  **Role-load reachability** is reported the same way as runs: `reloadDetails` → `reportRoleLoad`
  feeds the per-cluster load results (reachable = ok, unreachable = error) into the shared
  **run-status chip** (`beginRunStatus`/`finishRunStatus`), so the bottom bar shows OK/Error and
  the `#run-status-dialog` shows per-cluster status/duration/message + the executed introspection
  SQL on click — `ClusterRoleDetail` now carries `DurationMs`/`Queries` (from `LoadRoleDetails`
  timing + `pg.RoleDetailQueries`), mirroring `ClusterResult`. The form no longer renders an
  "Unreachable clusters" block (`renderDetailErrors` is gone). `updateOpsFooter` keeps the footer
  visible while `runState` is set so the chip shows even in the empty (not-found) state.
- Frontend is **vanilla JS** reached through `window.go.main.App` (`backend()` in
  [frontend/app.js](frontend/app.js)). No framework. `frontend/wailsjs/` is
  Wails-generated (regenerated by `wails dev`/`build`); a running `wails dev` watcher
  keeps it in sync, but hand-edit it only to stay consistent between builds.

## Two data paths

1. **Write (per-cluster transaction), template-driven.** `App.RunRoleBatch(RoleBatchRequest)` →
   [internal/batch/runner.go](internal/batch/runner.go) `RunRoleBatch` → `commands.ValidateRoleBatch`
   → resolve each `ClusterID` → production gate → per cluster (concurrent, bounded by
   `batch.max_concurrency` default 5) `runClusterTx`: `pg.Connect` → `conn.Begin` → for each op
   `commands.BuildArgs(OperationSpec)` + `pg.ExecuteOperation(tx, …)` → **Commit on success /
   Rollback on the first error** (message names the failing op). `RoleBatchRequest` carries
   per-cluster **ordered** op lists (`[]ClusterOps{ClusterID, []OperationSpec}`); each cluster's
   change is atomic and never interleaved with another's (separate connections). One
   `ClusterResult` per cluster (incl. `Queries []string` — the executed SQL per op, from
   `pg.ExecuteOperation`'s `(sql, msg, err)` return; function-mode binds are inlined for display,
   and `runClusterTx` records each op's SQL before the error check so a failing op's SQL is kept);
   timeout scales with op count. `RunRoleBatch` also takes an optional
   `func(model.ClusterProgress)` callback, invoked from each goroutine on cluster start ("running")
   and finish ("done"); `App.RunRoleBatch` passes a closure that `wailsruntime.EventsEmit`s each as
   a `role-batch-progress` event (the runner stays Wails-free per the import table). `pg.ExecuteOperation`/`execRoleConfig`/
   `runQuery` take a `pg.Querier` (satisfied by `*pgx.Conn` and `pgx.Tx`). The legacy single-op
   `App.RunOperation`/`batch.Run` (one autocommit statement) is kept but no longer used by the UI.
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
`SaveParentRoles`, `SaveCommentFields`, `SaveTargetSelection`, `SaveClusters`
(staged Clusters editor — replaces the whole clusters+categories set at once via
`Store.SaveClustersAndCategories`; the per-item `Add/Update/Delete Cluster/Category` are kept
but no longer used by the UI), `GetAppVersion`.
Run/test: `TestConnection` (by saved cluster id), `TestConnectionInput` (ad-hoc
`ClusterInput`+`Auth`, used by the cluster editor to test on-screen values),
`PreviewTargets`, `RunRoleBatch(RoleBatchRequest)` (per-cluster transactional batch; the UI's
create/update/remove path), `RunOperation(RunRequest)` (legacy single-op, kept but unused by the UI).
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
| `set_attribute` | loginname, **attribute** (space-separated keyword list) | statement (`ALTER ROLE ${loginname} WITH ${attribute}`) |
| `set_config` / `reset_config` | loginname, config_name(, config_value) | **no template** — hardcoded `ALTER ROLE … SET/RESET` in `pg.ExecuteOperation` |

Field kinds when embedding (statement/block): role names → **double-quoted identifiers**
(`quoteSQLIdentifier` → `"name"` with `"`→`""`, so case is preserved and special chars are safe;
rejects only empty/comma/NUL — comma is the list delimiter); `parent_roles` → comma-separated
list, each element double-quoted (`fieldIdentifierList` → `"a", "b"`);
`new_password`/`comment`/`fullname`/`email` → quoted **literals**;
`attribute` → a **space-separated keyword list** (`fieldKeywordList`) so the frontend combines
all of a cluster's attribute changes into ONE `ALTER ROLE … WITH kw1 kw2 …`; each keyword is
whitelisted (`SUPERUSER`/`NOSUPERUSER`, `CREATEROLE`/…, `LOGIN`, `REPLICATION`, `BYPASSRLS`) in
`commands.ValidateOperation`.

### Adding a new operation
Extend all of: `calltemplate.AllowedPlaceholders` + `placeholderKindForField`;
`commands` op const + `BuildArgs` + `ValidateOperation`; `model.DBFunctions` +
`*Params` + `OperationSpec`; `config.store.DefaultConfig` + `dbfunctions.go`
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
Settings & Clusters `[Discard] [Save]` right-aligned; Remove role far left via
`#btn-alter-remove{order:-1}` + `space-between`). The **Test connections** button lives in the Clusters
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
by default; the expanded `.cluster-list` flexes to fill the sidebar). The selection is
**remembered** — held in module-level `selectedCategoryIds` (`null` = the default
all-groups-checked) / `selectedClusterIds`, seeded **once** from persisted `Config.Targets`
on first `loadConfig` (later `loadConfig` calls keep the in-memory selection, so saving
Settings / cluster CRUD no longer resets it), and persisted (debounced) via
`SaveTargetSelection` (`Store.UpdateTargets` → `Config.Targets`, empty = all groups) so it
survives restarts. `render{Category,Cluster}Checkboxes` set `checked` from these; the
checkbox `change` handler is `onTargetChange`. The Clusters
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
(`ui.comment_default_view`). All staged; the Settings panel uses the same padded-panel + inset
footer as Clusters, with **Discard** (`btn-discard-settings` → `discardSettings`) + **Save**
(`btn-save-settings`, enabled-when-dirty / disabled-when-clean) buttons.

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
  `buildAlterClusterOps` groups each cluster's ops into one transactional batch (all of a
  cluster's enable/disable attribute keywords combine into a single `set_attribute`);
  `is-added` (green) applies only to a pending grant, not to attributes that are simply
  off.
- **Settings** (role GUCs) reuse the same row/scope-editor as attributes but keyed by
  `name=value` (a name with different values per cluster shows multiple rows); pending
  state is `alterConfigSet: Map<"name=value", Set<clusterId>>` and
  `alterConfigReset: Map<name, Set<clusterId>>`; `buildAlterClusterOps` emits
  `set_config`/`reset_config` ops. The scope dialog gains name+value inputs for a new setting.
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
- **Password** row (field + checkbox) and a red **Remove role** button. Run results are not an
  in-body table: they surface in the **footer status chip** `#run-status` (left of the action
  buttons; hidden until a run starts, then `running… (D/T)` → `OK`/`Error`), updated live from
  `role-batch-progress` events (`beginRunStatus`/`applyRunProgress`/`finishRunStatus`/
  `renderRunStatus`, state in `runState`). The chip is **button-sized, neutral-colored, and
  glyph-free** (spinner only while running; the *word* OK/Error carries the result — no ✓/✕).
  Clicking it opens `#run-status-dialog` (columns Cluster/Category/Status/Duration/Message +
  actions; **no Host column**), live while running. Each done row's actions cell has a
  **magnifier** (`.rst-view`) that opens a separate, larger `#run-queries-dialog` listing that
  cluster's executed SQL (`ClusterResult.Queries`/`ClusterProgress.Queries` — the queries are
  NOT shown inline in the table) and a **copy button** (`.rst-copy`) that copies the cluster's
  message + all queries it sent (including a failed op's, since `runClusterTx` records each op's
  SQL before the error check); `#run-queries-dialog` also has its own Copy button.
  `clearRunStatus()` (op-tab / page-tab switch) keeps it from leaking across pages.

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

- Run `go test ./... -count=1`, `node --check frontend/app.js`, and `node --test frontend/app.test.mjs`
  before committing. CI: `.github/workflows/test.yml` (go-test + frontend-test jobs).
- Frontend has **no build step / no package.json**; served statically by Wails. Logic tests live in
  [frontend/app.test.mjs](frontend/app.test.mjs) — Node's built-in `node --test`, zero dependencies:
  it loads `app.js` into a `node:vm` context behind a permissive DOM stub (the `DOMContentLoaded`
  init never fires, so loading only defines the globals) and drives the real functions, running
  snippets in the same context so they read/write app.js's top-level `let` state by bare name (same
  trick as the browser console). Covers `canonicalComment`, `editorFromComment`/`assembleCommentFrom`
  (null↔empty), `commentConsensus` (varies / unset-on-one / pending-add), and `buildAlterClusterOps`
  (presence create/remove + comment publishing). For a quick UI smoke test without the Go backend,
  `python3 -m http.server` in `frontend/` and exercise the global functions (`backend()` is
  undefined, so search/save show an inline error, but rendering/logic can be driven with mock
  `window.go.main.App`/`state`/`alterDetails`).
- Introspection/write SQL is best verified against a throwaway Postgres
  (`docker run … postgres:16`), calling `pg.RoleDetail` / `pg.ExecuteOperation`
  directly; remove any scratch `_test.go` afterward.

## Versioning

[VERSION](VERSION) is app semver (git tag `v$(cat VERSION)`). `internal/version`
defaults must match; `make sync-wails-version` aligns `wails.json` `productVersion`;
`GetAppVersion()` surfaces it. Config YAML `version:` is the **schema** version only.

## Out of scope (v1)

SSH tunnels, encrypted config vault, audit log, reading remote `pg_hba.conf`.
