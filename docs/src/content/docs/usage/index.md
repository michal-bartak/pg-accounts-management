---
title: Usage overview
description: Targets, scope labels, and how a change reaches the clusters
---

The **Operations** tab is where you create and alter roles. Its left sidebar picks the
targets; the right side is one form for the role.

<figure class="shot">
<div class="light-only">

![Operations overview](../../../assets/usage/operations-overview-light.png)

</div>
<div class="dark-only">

![Operations tab](../../../assets/usage/operations-overview-dark.png)

</div>
<figcaption>Operations tab — Target selection on the left, the role form on the right</figcaption>
</figure>

## Everything goes to the selected targets

There is no per-cluster form. Whatever you do on this tab — searching, creating, altering,
removing — applies to **exactly** the clusters selected in the sidebar.

<figure class="shot">
<div class="light-only">

![Target selection](../../../assets/usage/target-selection-light.png)

</div>
<div class="dark-only">

![Target selection](../../../assets/usage/target-selection-dark.png)

</div>
<figcaption>Target selection — group checkboxes and the expanded "Or pick clusters" list</figcaption>
</figure>

- Tick whole **groups**, or expand **Or pick clusters** to choose individual clusters.
- The selection is remembered between sessions and application restarts.

The selection in force when you search becomes the role's **scope**. Everything the form then
shows and does is about those clusters, and nothing else.

:::tip
You can change the selection later, after the role is loaded into the form. The app re-synthesises its content to match the selected targets.
:::

## Scope labels

Wherever clusters are listed — Present on, role-parent rows, attributes, settings, comments,
search results — the app uses the same labels, coloured by group:

<figure class="shot">
<div class="light-only">

![Scope labels](../../../assets/usage/scope-labels-light.png)

</div>
<div class="dark-only">

![Scope labels](../../../assets/usage/scope-labels-dark.png)

</div>
<figcaption>Scope labels — an outlined group label next to filled per-cluster labels</figcaption>
</figure>

- **Outlined label** (bordered, transparent) — carries the **cluster group name**. It is the visual cue that *every* cluster of that group matches — for example, a parent role is assigned on all clusters of the group.
- **Filled labels** (no border) — carry a **cluster name**, used when the setting varies across the clusters of a group. The colour still follows the group colour.

### Pending changes

Labels also show edits you haven't saved yet:


- **Pending addition** — a leading **`+`** on the label. The label follows its group colour.
- **Pending removal** — the label turns **red and struck through**, and drops its group colour.

Both are staged only. Nothing reaches a database until you Save.

## How changes are applied

1. You edit the form. Changes are being staged within the app
1. **Save changes** (or **Create role**) builds an ordered list of operations **per cluster**.
1. Each cluster's list runs as **one transaction** — commit on success, roll back on the first error.
1. Progress and results appear in the [command log](/pgcowboy/usage/command-log/).

Runs that touch a group flagged **require confirmation** stop at a dialog first.

If a role drop or creation are involved, the app performs a pre-flight check showing the result.

## Where to go next

- [Finding a role](/pgcowboy/usage/find-role/) — the search that starts every alter.
- [Creating a role](/pgcowboy/usage/creating-roles/) — the other entry point.
