# PostgreSQL call templates

DbAccounts runs role operations using a **call template** per operation in **Settings** or `config.yaml` under `db_functions.<operation>`.

Each operation has:

- **`call`** — the template body
- **`execution`** — how to run it: `function` (default), `statement`, or `block`

---

## Execution modes

| `execution` | You write | App runs | Placeholders |
|-------------|-----------|----------|--------------|
| **function** (default) | `schema.fn(${loginname}, …)` | `SELECT …` with `$1`, `$2` binds | Form values → bind parameters |
| **statement** | `DROP ROLE ${loginname}` | Single statement via `Exec` | Role names **embedded** in SQL (see below) |
| **block** | Inner PL/pgSQL only | `DO $dbaccounts$ BEGIN … END $dbaccounts$;` | Same embedding as statement |

### Why not `$1` in `DROP ROLE`?

PostgreSQL **cannot** bind object names (roles, tables) as `$1`. Templates like `DROP ROLE $1` leave `$1` unchanged or fail. Use **statement** or **block** mode so `${loginname}` becomes a validated role name in the SQL text (e.g. `DROP ROLE jdoe`).

Do **not** put raw `$1` / `$2` in templates — use `${loginname}` etc.

### Function mode (default)

```sql
SELECT <your template with $1, $2, ...>
```

User input is passed as **bind parameters**. Template literals (`NULL`, `'fixed_role'`, `ARRAY[...]`) come from config only.

Rules:

1. Write the call expression only — **no `SELECT`** (the app adds it).
2. **`${name}`** — whitelisted names only (see table below).
3. No semicolons in the template.

### Statement mode

One SQL statement, no `SELECT`, no semicolon at the end of the template.

```yaml
remove_role:
  execution: statement
  call: "DROP ROLE ${loginname}"
```

`${loginname}` and `${rolename}` (alias) are validated identifiers (`[a-zA-Z_][a-zA-Z0-9_]*`) and embedded into the statement.

Grant example:

```yaml
grant_parents:
  execution: statement
  call: "GRANT ${parent_roles} TO ${loginname}"
```

`${parent_roles}` from the form (`Gr_devs_all_ro` or `gr_a, gr_b`) becomes **unquoted** identifiers in SQL: `GRANT Gr_devs_all_ro TO test` or `GRANT gr_a, gr_b TO testuser`. It is **not** `GRANT 'Gr_devs_all_ro'` (string literals are invalid for role names in `GRANT`).

### Statement/block: how placeholders are embedded

| Field kind | Used for | Example expansion |
|------------|----------|-------------------|
| Identifier | Role/login names | `testuser`, `Gr_devs_all_ro` |
| Identifier list | `${parent_roles}` on `grant_parents` | `gr_a, gr_b` (comma-separated in form) |
| Literal (quoted) | Passwords, names, email | `'secret'`, `'John Doe'` |

Function mode always uses `$n` binds instead of embedding.

### Block mode

Write only the statements that go inside `BEGIN … END`. The app adds the `DO $dbaccounts$` wrapper. Semicolons are allowed inside the body.

```yaml
remove_role:
  execution: block
  call: "DROP ROLE ${loginname};"
```

Prefer `DROP ROLE ${loginname};` inside the block rather than `format()` with a placeholder unless you know the required quoting.

---

## Placeholder syntax (function mode)

| Syntax | Behavior |
|--------|----------|
| `${loginname}`, `${fullname}`, … | Value from the Operations form, bound as `$n` (text). |
| `NULL`, `'literal'` | Copied into SQL unchanged (from trusted config). |
| `ARRAY['gr_a', 'gr_b'] \|\| ${parent_role}` | Fixed roles in template; empty parent → `\|\| NULL`; set parent → `\|\| $n::text[]`. |
| `ARRAY[${parent_role}, 'gr_a', 'gr_b']` | Normalized to `ARRAY['gr_a', 'gr_b'] \|\| ${parent_role}`. |

`create_role` with `ARRAY ||` syntax requires **`execution: function`**.

---

## Create role

```text
admin_access.create_role(${loginname}, NULL, ${fullname}, ${email}, ARRAY['gr_personal_users', 'gr_personal_users_ldap'] || ${parent_role})
```

```yaml
db_functions:
  create_role:
    execution: function
    call: "admin_access.create_role(${loginname}, NULL, ${fullname}, ${email}, ARRAY['gr_personal_users', 'gr_personal_users_ldap'] || ${parent_role})"
```

---

## Remove role examples

**Function (wrapper on server):**

```yaml
remove_role:
  execution: function
  call: "admin_access.drop_user(${loginname})"
```

**Direct DDL:**

```yaml
remove_role:
  execution: statement
  call: "DROP ROLE ${loginname}"
```

**PL/pgSQL block:**

```yaml
remove_role:
  execution: block
  call: "DROP ROLE ${loginname};"
```

---

## Grant parent roles

**Function (wrapper on server):**

```yaml
grant_parents:
  execution: function
  call: "your_schema.grant_role_parents(${loginname}, ${parent_roles})"
```

`${parent_roles}` is one text bind (comma-separated string) passed to your function.

**Direct GRANT (statement mode):**

```yaml
grant_parents:
  execution: statement
  call: "GRANT ${parent_roles} TO ${loginname}"
```

In statement/block mode, `${parent_roles}` expands to **unquoted role identifiers**, not `'literal'` strings. Comma-separated values in the form become `GRANT gr_a, gr_b TO user`. `${loginname}` is also an identifier (the member role).

Example runtime SQL: `GRANT Gr_devs_all_ro TO test` (not `GRANT 'Gr_devs_all_ro' TO test`).

---

## Revoke parent roles

Same form fields and placeholders as grant (`loginname`, `parent_roles`). Default in the app:

```yaml
revoke_parents:
  execution: statement
  call: "REVOKE ${parent_roles} FROM ${loginname}"
```

**Function (wrapper on server):**

```yaml
revoke_parents:
  execution: function
  call: "your_schema.revoke_role_parents(${loginname}, ${parent_roles})"
```

Statement/block: `${parent_roles}` → unquoted identifiers (`REVOKE Gr_devs_all_ro FROM test` or `REVOKE gr_a, gr_b FROM testuser`).

---

## Allowed placeholders

| Operation | `${...}` names | Statement/block embedding |
|-----------|----------------|---------------------------|
| `create_role` | `loginname`, `fullname`, `email`, `parent_role` | Identifiers: login/parent role; literals: fullname, email. `ARRAY \|\|` needs `execution: function` |
| `remove_role` | `loginname`, `rolename` | Identifiers |
| `grant_parents` | `loginname`, `parent_roles` | Identifiers; `parent_roles` = comma-separated identifier list |
| `revoke_parents` | `loginname`, `parent_roles` | Same as grant_parents |
| `change_password` | `loginname`, `new_password` | Identifier + literal (password) |

---

## Common mistakes

| Mistake | Result |
|---------|--------|
| `DROP ROLE $1` or `drop_user($1)` in template | Use `${loginname}`; use `execution: statement` for DDL |
| `GRANT 'role_name' TO user` from `${parent_roles}` | Use `execution: statement`; parent roles are identifiers, not quoted literals |
| `REVOKE 'role_name' FROM user` | Same as GRANT — use `revoke_parents` statement mode |
| `${rolename}` without statement/block | Works as alias of login name when whitelisted |
| `SELECT` in function template | Rejected on save |
| `DO $dbaccounts$` inside block template | Rejected — app owns the wrapper |
| Full call pasted into legacy `name` field | Migrated or reset on load |

---

## Security (statement / block)

Role names are embedded after strict identifier validation. Only whitelisted `${...}` placeholders are allowed. Prefer DB functions for complex logic when possible.

---

## Return values

**Function** mode: if the call returns `text`, that value appears in results; otherwise `ok`.

**Statement / block** mode: `ok` on success.
