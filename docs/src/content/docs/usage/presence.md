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
- **Dropping** a cluster records a `DROP ROLE` for that cluster alone, pre-flighted by the
  [dependency check](#dependency-check-before-a-drop) when you Save.

The picker can only offer clusters that are in the role's scope. To bring in a cluster that
wasn't part of Target selection when you searched, tick it in the sidebar and search again.

## Remove role

The red **Remove role** button drops the role from **every cluster in scope where it exists**.
It always asks for confirmation first — the dependency check below — and groups flagged
*require confirmation* add their own.

## Dependency check before a drop

Nothing is dropped blind. Before a `DROP ROLE` goes out — whether from **Remove role** or from
a cluster you dropped in the presence picker and published with **Save changes** — the app reads
the objects that depend on the role on every targeted cluster and shows them per cluster.

<figure class="shot">
<div class="light-only">

![Dependency check before removing a role](../../../assets/usage/remove-role-confirm-light.png)

</div>
<div class="dark-only">

![Dependency check before removing a role](../../../assets/usage/remove-role-confirm-dark.png)

</div>
<figcaption>Dependency check — per cluster, what depends on the role and whether to drop it there</figcaption>
</figure>

Clusters are grouped into three sections, in this order — and inside each by group, then alias:

1. **No dependencies** — just the list of clusters. They are dropped without further asking.
2. **Could not be checked** — the cluster and why (unreachable, or the connect user lacks
   permission on the catalogs).
3. **Dependencies found** — per cluster, the objects that depend on the role.

Everything in the last two sections is set to **Skip** and left out of the run entirely. That is
the default. Switch a cluster to **Try anyway** to drop it regardless: PostgreSQL then decides, and
the outcome shows in the [command log](/pg-accounts-management/usage/command-log/) like any other
run.

If you go and clear the dependencies while the popup is open — reassigning ownership or dropping
the objects in psql — the **reload** button next to the title re-runs the check on the same
clusters without closing the dialog. Any *Try anyway* you already picked is kept where it still
applies.

Every cluster runs the same query, so the magnifier next to the popup's title shows it once. The
check itself is the configurable `role_dependencies`
[read query](/pg-accounts-management/configuration/call-templates/); its default reads
`pg_shdepend`, which describes the objects of the database the app connects to plus cluster-wide
ones — anything owned in another database on the same cluster is listed but not named.
