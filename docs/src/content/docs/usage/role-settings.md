---
title: Altering settings
description: Role-level GUCs, which can hold a different value per cluster
---

**Settings** are role-level configuration parameters (`GUC`s) — `statement_timeout`,
`log_statement`, `search_path`, and so on. They're read from the role's stored configuration
and use the same rows and per-cluster editor as
[privileges](/pg-accounts-management/usage/privileges/).

<figure class="shot-todo" data-shot="settings-rows.png">
  <figcaption>Settings section — one row per name/value pair, with scope labels</figcaption>
</figure>

## Rows are name **and** value

A setting can hold a different value on different clusters, so rows are keyed by `name=value`.
The same parameter with two values shows as two rows, each with its own scope — the difference
is visible instead of averaged away.

<figure class="shot-todo" data-shot="settings-differing-values.png">
  <figcaption>The same parameter with different values on different clusters, shown as two rows</figcaption>
</figure>

- **✎** — set that name/value on the clusters you tick, and clear it where you untick.
- **×** — reset the parameter everywhere.
- **↺** — discard pending changes on that row.

## Adding a setting

**Add setting…** takes a parameter **name** and a **value**, plus the clusters to apply them
to.

<figure class="shot-todo" data-shot="add-setting.png">
  <figcaption>Add setting dialog — name, value, and cluster selection</figcaption>
</figure>

On save these become `set_config` and `reset_config` operations
(`ALTER ROLE … SET name = value` / `RESET name` by default).
