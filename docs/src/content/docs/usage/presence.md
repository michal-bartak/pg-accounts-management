---
title: Where a role exists
description: Add the role to more clusters, drop it from some, or remove it entirely
---

The **Present on** block, just under the login name, shows which clusters the role exists on.

<figure class="shot">
<div class="light-only">

![Present on](../../../assets/usage/present-on-light.png)

</div>
<div class="dark-only">

![Present on](../../../assets/usage/present-on-dark.png)

</div>
<figcaption>Present on — existing clusters, a pending addition prefixed with +, a pending removal in red strikethrough</figcaption>
</figure>

It uses the shared [scope labels](/pg-accounts-management/usage/):

- Plain label — the role exists there now.
- Leading **`+`**, same colour — a pending addition.
- **Red and struck through** — a pending removal.

## Editing presence

The <svg class="doc-ic" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M21.174 6.812a1 1 0 0 0-3.986-3.987L3.842 16.174a2 2 0 0 0-.5.83l-1.321 4.352a.5.5 0 0 0 .623.622l4.353-1.32a2 2 0 0 0 .83-.497z"/><path d="m15 5 4 4"/></svg> **edit** button opens a picker listing the role's
[scope](/pg-accounts-management/usage/) — the clusters that Target selection resolved to when
you searched. Tick to add, untick to drop.

<figure class="shot">
<div class="light-only">

![Presence picker](../../../assets/usage/presence-editor-light.png)

</div>
<div class="dark-only">

![Presence picker](../../../assets/usage/presence-editor-dark.png)

</div>
<figcaption>Presence picker — the clusters in scope, with the role's current presence ticked</figcaption>
</figure>

- **Adding** a cluster brings the whole form to bear on it: privileges, attributes, settings,
  and the comment all target it too. A `CREATE ROLE` is prepended to that cluster's transaction
  on Save. Whether creation is staged automatically is a
  [general setting](/pg-accounts-management/configuration/general/).
- **Dropping** a cluster records a `DROP ROLE` for that cluster alone.

The picker can only offer clusters that are in the role's scope. To bring in a cluster that
wasn't part of Target selection when you searched, tick it in the sidebar and search again.

## Remove role

The red **Remove role** button drops the role from **every cluster in scope where it exists**.
It always asks for confirmation first, and groups flagged *require confirmation* add their own.

<figure class="shot">
<div class="light-only">

![Remove role confirmation, listing the affected clusters](../../../assets/usage/remove-role-confirm-light.png)

</div>
<div class="dark-only">

![Remove role confirmation, listing the affected clusters](../../../assets/usage/remove-role-confirm-dark.png)

</div>
<figcaption>Remove role confirmation, listing the affected clusters</figcaption>
</figure>
