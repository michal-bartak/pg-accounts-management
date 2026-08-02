---
title: Call templates
description: How DbAccounts turns form input into the SQL it runs
---

Every change the app makes runs through a **call template** — a short piece of SQL with
`${placeholder}` fields. Templates live in **Settings → DB command templates**, or in the
config file under `db_functions.<operation>`. Each has an **execution mode**: `statement`,
`block`, or `function`.

The defaults are plain PostgreSQL and cover everything out of the box. You only edit a
template when you want the app to go through a wrapper function or view — for example, so a
low-privilege connection can create roles via a `SECURITY DEFINER` function, or to add audit
logging.

<figure class="shot-todo" data-shot="settings-templates.png">
  <figcaption>Settings → DB command templates and Introspection queries, side by side</figcaption>
</figure>

Clicking a command opens its editor: execution mode, the template text, clickable placeholder
chips, and a **Default** button that restores the built-in version. The **?** in the title bar
opens the full syntax reference.

<figure class="shot-todo" data-shot="template-editor.png">
  <figcaption>Template editor — execution mode, template text, placeholder chips, Default button</figcaption>
</figure>

## Operations and their defaults

| Operation | Default template |
|-----------|------------------|
| `create_role` | `CREATE ROLE ${loginname}` |
| `remove_role` | `DROP ROLE ${loginname}` |
| `grant_parents` | `GRANT ${parent_roles} TO ${loginname}` |
| `revoke_parents` | `REVOKE ${parent_roles} FROM ${loginname}` |
| `change_password` | `ALTER ROLE ${loginname} PASSWORD ${new_password}` |
| `set_comment` | `COMMENT ON ROLE ${loginname} IS ${comment}` |
| `set_attribute` | `ALTER ROLE ${loginname} WITH ${attributes}` |
| `set_config` | `ALTER ROLE ${loginname} SET ${config_name} = ${config_value}` |
| `reset_config` | `ALTER ROLE ${loginname} RESET ${config_name}` |

## Execution modes

- **statement** / **block** — the template is raw SQL (DDL, `GRANT`, `ALTER ROLE`).
  PostgreSQL can't bind a role name as `$1`, so the app embeds names as quoted **identifiers**
  and literals as **escaped strings**. Use `block` when the SQL is a `DO $$ … $$` block.
- **function** — the template is a function call. Values are passed as real bind parameters
  (`$1`, `$2`, …), which is the safest option when your DDL is wrapped in a function.

## How fields are embedded

The app knows the kind of each field, so you don't quote them yourself:

- **Role names** (`loginname`) → double-quoted identifiers, preserving case.
- **`parent_roles`** (create_role, grant_parents, revoke_parents) → in **statement/block** mode a
  comma-separated list of quoted identifiers (`"a", "b"`); in **function** mode an inline
  `ARRAY['a', 'b']` literal (values verbatim, an empty selection → `NULL`).
- **`new_password`**, **`comment`**, **`config_value`** → escaped string literals.
- **Comment fields** — one placeholder per key configured under
  [Comment fields](/pg-accounts-management/configuration/comment-fields/) (e.g. `${full_name}`,
  `${e_mail}`), available in **`create_role`** and **`set_comment`**.
  The value comes from the role's JSON comment and is embedded by type: string → quoted literal,
  number/boolean → bare literal, array/object → JSON text, and an empty/`null`/missing value →
  bare `NULL`.
- **`config_name`** → a bare, validated GUC name (never quoted).
- **`attributes`** → a space-separated keyword list, each keyword checked against a whitelist
  (`SUPERUSER`/`NOSUPERUSER`, `CREATEROLE`, `LOGIN`, `REPLICATION`, `BYPASSRLS`, …). All of a
  cluster's attribute changes are combined into one `ALTER ROLE … WITH …`.

## A function-mode example

Suppose role creation must go through a helper that also assigns fixed groups. Set
`create_role` to **function** mode with:

```text
admin_access.create_role(
  ${loginname}, NULL, ${full_name}, ${e_mail},
  ARRAY['gr_personal_users', 'gr_personal_users_ldap'] || ${parent_roles}
)
```

- `${loginname}` is a bind; `${full_name}` / `${e_mail}` are comment-field placeholders (their
  values come from the role's comment, typed, `NULL` when empty/absent).
- `NULL` is a plain SQL literal for an unused argument.
- `ARRAY[...] || ${parent_roles}` appends the selected parent roles to a fixed set (it becomes
  `|| NULL` when the selection is empty).

## Read (introspection) queries

Alter-role search and detail use three **read** queries, also editable (Settings →
Introspection queries, or `db_reads` in config). Each takes a single `${rolename}` bind, and
its result columns are matched **by name** against a fixed contract — so you can point one at
a privileged wrapper function or view when the connecting user can't read the catalogs
directly.

| Query | Contract columns |
|-------|------------------|
| `search_roles` | `rolname`, `comment` |
| `role_detail` | the seven `rol*` flags, `comment`, `rolconfig` |
| `role_parents` | `rolname` |

Column order doesn't matter and a missing column is tolerated, but an unexpected extra column
is rejected.

Full syntax, the field whitelist, YAML examples, and common mistakes are in the repository's
[`sql/README.md`](https://github.com/michal-bartak/pg-accounts-management/blob/main/sql/README.md).
