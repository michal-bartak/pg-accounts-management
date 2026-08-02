---
title: Altering attributes
description: The role flags, enabled or disabled per cluster
---

**Attributes** are PostgreSQL's role flags. They use the same rows and the same per-cluster
editor as [privileges](/pg-accounts-management/usage/privileges/).

<figure class="shot-todo" data-shot="attributes-rows.png">
  <figcaption>Attributes section — all seven flags with their scope labels</figcaption>
</figure>

| Attribute | Keyword |
|-----------|---------|
| Superuser | `SUPERUSER` / `NOSUPERUSER` |
| Create role | `CREATEROLE` / `NOCREATEROLE` |
| Create DB | `CREATEDB` / `NOCREATEDB` |
| Inherit | `INHERIT` / `NOINHERIT` |
| Login | `LOGIN` / `NOLOGIN` |
| Replication | `REPLICATION` / `NOREPLICATION` |
| Bypass RLS | `BYPASSRLS` / `NOBYPASSRLS` |

All seven rows always render, whether or not the flag is set. Green marks a **pending
enable** — an attribute that is simply off looks neutral, not pending.

**✎** opens the per-cluster editor, **×** disables everywhere, **↺** discards pending changes
on that row.

## One statement per cluster

All of a cluster's attribute changes are combined into a single
`ALTER ROLE … WITH keyword keyword …`, so enabling `LOGIN` and disabling `SUPERUSER` is one
statement, not two.
