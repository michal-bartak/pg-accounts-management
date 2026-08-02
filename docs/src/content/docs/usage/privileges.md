---
title: Altering privileges
description: Parent-role memberships, granted and revoked per cluster
---

**Privileges** are the role's parent-role memberships. Each parent is one row: the name on the
left, [scope labels](/pg-accounts-management/usage/) on the right showing where it's granted.

<figure class="shot-todo" data-shot="privileges-rows.png">
  <figcaption>Privileges section — rows with scope labels and the ✎ / × / ↺ actions</figcaption>
</figure>

## Row actions

Every row shows all three buttons, so the layout doesn't shift. Ones that don't apply are
greyed out.

| Button | Effect |
|--------|--------|
| **✎** | Opens a per-cluster editor that both grants and revokes. |
| **×** | Revokes on every cluster. |
| **↺** | Discards pending changes on that row. |

The editor is a checkbox per cluster showing the desired end state — tick to grant, untick to
revoke. The app works out the difference from what's there now.

<figure class="shot-todo" data-shot="privilege-scope-editor.png">
  <figcaption>Per-cluster editor — checkboxes for the desired end state</figcaption>
</figure>

Pending changes are visible before you save: a pending grant is **green**, a pending revoke is
**struck through**.

## Adding a privilege

**Add privilege…** introduces a new membership on any mix of groups and clusters. Names from
[Preconfigured parent groups](/pg-accounts-management/configuration/parent-roles/) appear as
chips, and you can pick several at once.

<figure class="shot-todo" data-shot="add-privilege.png">
  <figcaption>Add privilege dialog — parent chips, free-text name, and cluster selection</figcaption>
</figure>

On **Save changes** these become `grant_parents` and `revoke_parents` operations, folded into
each cluster's transaction.
