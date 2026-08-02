---
title: Configuration
description: Where each setting lives and how staged editing works
---

Configuration lives on two tabs:

- **Clusters** — the clusters you connect to and the groups they belong to.
- **Settings** — everything else: comment fields, call templates, parent groups, the
  password generator, and general preferences.

Both tabs are **staged**. Adds, edits, and deletes change an on-screen draft; nothing is
written until you press **Save**. **Discard** reverts to the saved state. Save is active only
while there are unsaved changes.

<figure class="shot-todo" data-shot="settings-overview.png">
  <figcaption>Settings tab — all sections, with the Discard / Save footer</figcaption>
</figure>

## Sections

| Section | What it controls |
|---------|------------------|
| [Clusters](/pg-accounts-management/configuration/clusters/) | Connection details, credentials, groups |
| [Comment fields](/pg-accounts-management/configuration/comment-fields/) | JSON keys shown as labelled inputs |
| [Call templates](/pg-accounts-management/configuration/call-templates/) | The SQL behind every read and write |
| [Parent groups](/pg-accounts-management/configuration/parent-roles/) | Role names offered as quick picks |
| [Password generator](/pg-accounts-management/configuration/password-generator/) | Length and character classes |
| [General](/pg-accounts-management/configuration/general/) | Appearance, concurrency, update check |

## Configuration file

Everything is stored in one YAML file, written with owner-only permissions:

| OS | Path |
|----|------|
| macOS | `~/Library/Application Support/DbAccounts/config.yaml` |
| Linux | `~/.config/dbaccounts/config.yaml` |
| Windows | `%AppData%\DbAccounts\config.yaml` |

You can edit it by hand while the app is closed. See
[`config.example.yaml`](https://github.com/michal-bartak/pg-accounts-management/blob/main/config.example.yaml)
for a commented reference.
