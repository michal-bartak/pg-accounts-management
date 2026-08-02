---
title: Where a role exists
description: Add the role to more clusters, drop it from some, or remove it entirely
---

The **Present on** block, just under the login name, shows which clusters the role exists on.

<figure class="shot-todo" data-shot="present-on.png">
  <figcaption>Present on — existing clusters, a pending addition in green, a pending removal struck through</figcaption>
</figure>

- Plain labels — the role exists there now.
- **Green** — a pending addition.
- **Struck through** — a pending removal.

## Editing presence

The **✎** button opens a picker over the clusters that were in scope when you searched. Tick to
add, untick to drop.

<figure class="shot-todo" data-shot="presence-editor.png">
  <figcaption>Presence picker — the clusters in scope, with the role's current presence ticked</figcaption>
</figure>

- **Adding** a cluster brings the whole form to bear on it: privileges, attributes, settings,
  and the comment all target it too. A `CREATE ROLE` is prepended to that cluster's transaction
  on Save. Whether creation is staged automatically is a
  [general setting](/pg-accounts-management/configuration/general/).
- **Dropping** a cluster records a `DROP ROLE` for that cluster alone.

A cluster must have been selected at search time to be addable here. To widen the scope,
change Target selection and search again.

## Remove role

The red **Remove role** button, kept apart on the far left of the footer, drops the role from
**every** cluster it exists on. It always asks for confirmation first, and groups flagged
*require confirmation* add their own.

<figure class="shot-todo" data-shot="remove-role-confirm.png">
  <figcaption>Remove role confirmation, listing the affected clusters</figcaption>
</figure>
