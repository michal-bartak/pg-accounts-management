# Changelog

All notable changes to DbAccounts. Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/);
versioning: [SemVer](https://semver.org/spec/v2.0.0.html).

> **Feeds releases:** the section matching `VERSION` is extracted by
> [`.github/workflows/release.yml`](.github/workflows/release.yml) into the GitHub release
> description. Keep `[Unreleased]` current; at release time rename it to `## [X.Y.Z] - YYYY-MM-DD`.
> See [RELEASING.md](RELEASING.md).

## [Unreleased]

### Added

- **Configurable search columns** — choose what shows next to the role name when searching, each column built from comment keys (`${{first_name}} ${{last_name}}`) or the raw comment.

### Changed

- **"Privileges" is now "Role Parents"** — the role-form section, its dialog (*Assign parents…*) and the Settings list (*Preconfigured role parents*) are named for what they hold: any role can be a parent, not just a group.
- **Assign several parents by typing** — the Assign-parents dialog takes a comma-separated list of role names, mixable with the preconfigured chips; it previously accepted only one typed name.
- **Denser UI** — text, controls, table rows and spacing render about 12% smaller, so noticeably more fits on screen.
- **System requirements are now stated explicitly**: macOS 12+, Windows 10/11 with WebView2, or Linux with WebKitGTK 2.38+. Windows 7/8/8.1 cannot run the app at all (they have not been able to since before the first release — the Go toolchain builds Windows 10+ binaries only). On an older macOS or Linux the Find-role columns stop lining up, as they now use CSS subgrid. The macOS installer previously claimed to support 10.13, which the app has never actually run on.
- **Breaking: comment keys in templates now use double braces** — `${{full_name}}`. Single braces are reserved for built-ins (`${loginname}`, `${comment}`, `${parent_roles}`, …), so a comment key named `comment` or `loginname` is finally reachable. Update any custom `create_role` / `set_comment` template that referenced a comment key in single braces.
- Find-role results line up as a table, and no longer assume the comment has a `full_name` key.
- An unknown `${name}` in a template is now reported instead of silently rendering empty.

### Fixed

- A comment field named like a built-in no longer corrupts what `set_comment` writes in function mode (the comment could be re-serialized, or an empty one stored as NULL).
- Adding a comment field and using it in a call template in the same Settings save now works.
- **Settings saves are all-or-nothing** — a rejected template or field no longer leaves the earlier sections of the page already written to the config file while the error is reported.
- A comment containing `${` no longer fails the operation.

## [0.9.0] - 2026-08-12

### Added

- **Changelog** — releases open with the new-in-this-version summary and link to the full file.
- **Alter role** — search a role across clusters, edit privileges, attributes and settings per cluster or group.
- **Presence editor** — add or drop a role on individual clusters.
- **Dependency pre-flight** — removals check each cluster first; clusters with dependencies default to *Skip*.
- **Transactional runs** — one transaction per cluster, rolled back on the first error.
- **Run status chip** — live per-cluster progress, duration, message and executed SQL.
- **Role comments in any format** — plain text or JSON, edited as fields or raw, reconciled when they differ.
- **Configurable comment fields** — labelled JSON keys, usable as template placeholders.
- **Preconfigured parent groups** — pick-list chips when granting.
- **Role settings** — view, set and reset per-role GUCs.
- **Templatable introspection queries** — point the four reads at your own views or functions.
- **Password generator** — configurable length and character classes, with copy and reveal.
- **Per-cluster password** — optional, stored in the private config.
- **Cluster groups** — own colour plus a *production confirm* gate.
- **Check for updates** — About-dialog button, opt-in startup check, badge until you upgrade.
- **Documentation site** on GitHub Pages.
- **Native installers** — `.dmg`, `.msi`, `.deb`, `.rpm`; archives dropped.
- Window size and target selection survive restarts.

### Changed

- Default call templates are vanilla PostgreSQL DDL — works against a plain cluster out of the box.
- Redesigned shell: fixed header and tabs, independent scrolling, pinned footers, one form for Create and Alter.
- Feedback moved out of toasts — buttons flash, errors render inline, runs report in the status chip.
- Clusters and Settings are staged: edits are drafts until Save, Discard reverts.
- Cluster groups moved from Settings to the Clusters tab.

### Fixed

- Role names double-quoted — case preserved, special characters allowed.
- A failed run keeps your pending edits instead of clearing the form.
- Focus rings only for keyboard navigation, no longer clipped or left behind.
- Find-role dialog no longer opens with stale results.
- An unreachable cluster is reported per cluster instead of failing the whole read.

## [0.2.0] - 2026-06-05

### Added

- Custom application icon.

### Fixed

- Windows `GOPATH` handling in the Makefile.
- Linux builds use the `webkit2_41` tag on Ubuntu 24.04+.

## [0.1.2] - 2026-06-05

First tagged builds: cluster config, target selection, role create and remove via SQL call
templates. Earlier history:
[git tags](https://github.com/michal-bartak/pg-accounts-management/tags).
