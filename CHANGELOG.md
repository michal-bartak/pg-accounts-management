# Changelog

All notable changes to DbAccounts. Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/);
versioning: [SemVer](https://semver.org/spec/v2.0.0.html).

> **Feeds releases:** the section matching `VERSION` is extracted by
> [`.github/workflows/release.yml`](.github/workflows/release.yml) into the GitHub release
> description. Keep `[Unreleased]` current; at release time rename it to `## [X.Y.Z] - YYYY-MM-DD`.
> See [RELEASING.md](RELEASING.md).

## [Unreleased]

## [1.0.0] - 2026-08-14

### Added

- **Configurable search columns** — choose what shows next to a role name when searching.

### Changed

- **Clusters now live in their own file** — `clusters.yaml`, next to `config.yaml`, holds your clusters, groups and last target selection.
- **Comment fields fill the width** — they flow into columns that grow with the window instead of one tall list.
- **"Privileges" renamed to "Role Parents"** — any role can be a parent, not just a group, so the wording finally says so.
- **Assign several parents at once** — type a comma-separated list instead of one name at a time.
- **Denser UI** — about 12% smaller text and controls, so more fits on screen.
- **Breaking: comment placeholders use double braces** — `${{full_name}}`, not `${full_name}`. Update any custom template that referenced a comment key this way.
- **Find-role results line up as a table** and no longer assume a `full_name` comment key.
- **An unknown `${name}` in a template is now reported**, instead of silently rendering empty.

### Fixed

- **A picked cluster is visible again after a restart** — the collapsed cluster list now opens itself to show your selection.
- **Settings save is atomic** — a rejected template no longer leaves part of the page already saved, and a new comment field can be used in a template in the same save.
- **A comment field named like a built-in no longer corrupts what gets saved.**
- **A comment containing `${` no longer breaks the operation.**
- The **`?` help badge** now lines up with its label everywhere.

## [0.9.0] - 2026-08-12

### Added

- **Alter role** — search a role across clusters and edit privileges, attributes and settings per cluster or group.
- **Presence editor** — add or remove a role on individual clusters.
- **Dependency pre-flight** — removing a role checks for dependencies first; clusters with dependencies default to Skip.
- **Transactional runs** — each cluster's changes commit or roll back together.
- **Live run status** — per-cluster progress, duration, message and executed SQL.
- **Flexible role comments** — plain text or JSON, editable as fields or raw text, with configurable labeled fields.
- **Preconfigured parent groups** — pick-list chips when granting.
- **Role settings** — view, set and reset per-role GUCs.
- **Templatable introspection queries** — point searches and reads at your own views or functions.
- **Password generator** — configurable length and character classes, with copy and reveal.
- **Per-cluster password** — optional, stored privately in your config.
- **Cluster groups** — color-coded, with an optional production confirm gate.
- **Check for updates** — from the About dialog, or automatically on startup.
- Window size and target selection now survive restarts.

### Changed

- **Default templates work out of the box** — vanilla PostgreSQL DDL, no setup required.
- **Redesigned interface** — one form for creating and editing roles, with independently scrolling panels.
- **Feedback moved out of toasts** — buttons flash, errors show inline, runs report in a status chip.
- **Clusters and Settings are now staged** — edits are drafts until you Save; Discard reverts them.
- Cluster groups moved from Settings to the Clusters tab.

### Fixed

- Role names are quoted, so special characters and mixed case work.
- A failed run keeps your pending edits instead of clearing the form.
- Focus rings only appear for keyboard navigation.
- The Find-role dialog no longer opens showing stale results.
- An unreachable cluster is reported on its own, instead of failing the whole search.

## [0.2.0] - 2026-06-05

### Added

- New application icon.

## [0.1.2] - 2026-06-05

First tagged release: cluster config, target selection, and create/remove roles via SQL call
templates. Earlier history:
[git tags](https://github.com/michal-bartak/pg-accounts-management/tags).
