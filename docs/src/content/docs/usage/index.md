---
title: Usage overview
description: Targets, scope labels, and how a change reaches the clusters
---

The **Operations** tab is where you create and alter roles. Its left sidebar picks the
targets; the right side is one form for the role.

![Operations tab](../../../assets/screenshot-operations.png)

## Everything goes to the selected targets

There is no per-cluster form. Whatever you do on this tab — searching, creating, altering,
removing — applies to **exactly** the clusters selected in the sidebar.

<figure class="shot-todo" data-shot="target-selection.png">
  <figcaption>Target selection — group checkboxes and the expanded "Or pick clusters" list</figcaption>
</figure>

- Tick whole **groups**, or expand **Or pick clusters** to choose individual clusters.
- The selection is remembered between sessions.
- Search results and the role form only ever cover the clusters selected **at search time**.

## Scope labels

Wherever clusters are listed — Present on, privilege rows, attributes, settings, comments,
search results — the app uses the same labels, coloured by group:

<figure class="shot-todo" data-shot="scope-labels.png">
  <figcaption>Scope labels — an outlined group label next to filled per-cluster labels</figcaption>
</figure>

- **Outlined label** (bordered, transparent) — *every* selected cluster in that group matches.
  One label stands for the whole group, matching the bordered group boxes in Target selection.
- **Filled labels** (no border) — a partial match. You get one label per matching cluster, so
  a partial state is never hidden behind a group name.

## How a change is applied

1. You edit the form. Nothing is sent while you edit.
2. **Save changes** (or **Create role**) builds an ordered list of operations **per cluster**.
3. Each cluster's list runs as **one transaction** on its own connection — commit on success,
   roll back on the first error. Clusters never interfere with each other.
4. Progress and results appear in the [command log](/pg-accounts-management/usage/command-log/).

Runs that touch a group flagged **require confirmation** stop at a dialog first. Removing a
role always asks.

## Where to go next

- [Finding a role](/pg-accounts-management/usage/find-role/) — the search that starts every alter.
- [Creating a role](/pg-accounts-management/usage/creating-roles/) — the other entry point.
