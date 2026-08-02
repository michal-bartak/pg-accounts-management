---
title: Creating a role
description: One form, every selected cluster, and what happens when a cluster fails
---

Click the **Create role** tab. The form resets to an empty role over the clusters currently
selected in **Target selection**.

<figure class="shot-todo" data-shot="create-role-form.png">
  <figcaption>Create role — login name, comment editor, privileges, attributes, settings</figcaption>
</figure>

It is the same form as Alter, over an empty baseline: every edit is a grant, an enable, or a
set. Fill in what you need:

| Part | Notes |
|------|-------|
| **Login name** | Required. The only field you must fill. |
| **Comment** | Same [comment editor](/pg-accounts-management/usage/comments/), Fields or Raw. |
| **Privileges** | Parent roles, including the [preconfigured chips](/pg-accounts-management/configuration/parent-roles/). |
| **Attributes** | The role flags, per cluster. |
| **Settings** | Role-level parameters, per cluster. |
| **Password** | Tick [Set password](/pg-accounts-management/usage/password/) to set one at creation. |

Changing the target selection re-synthesises the form and keeps the edits you've already made.

## What runs

Press **Create role**. The app validates the login name and warns you if the role already
exists on any selected cluster.

Then, per cluster and in one transaction: `create_role`, followed by the parent grants, the
password, the attributes, the settings, and the comment. Confirmation-flagged groups ask first.

## When a cluster fails

<figure class="shot-todo" data-shot="create-role-partial-failure.png">
  <figcaption>Command log after a run where one cluster failed and the others succeeded</figcaption>
</figure>

- The failing cluster is **rolled back** completely — no half-created role. The error names
  the operation that failed.
- **Other clusters are unaffected.** Each commits its own transaction, so a run can succeed on
  some clusters and fail on others. The
  [command log](/pg-accounts-management/usage/command-log/) shows which.

What happens next depends on how much got through:

- **Every cluster failed** — you stay on the Create form with your input intact. Fix the cause
  and press Create again.
- **At least one cluster succeeded** — the form hands off to **Alter role** with the new role
  loaded over the same clusters. The ones that failed appear as *not present* in
  [Present on](/pg-accounts-management/usage/presence/), so you can retry just those with
  **Save changes** instead of re-running the whole creation.

Either way the [command log](/pg-accounts-management/usage/command-log/) keeps the creation
results — the follow-up load is appended to it, not written over it.
