---
title: Setting a password
description: The password row, the generator, and where it applies
---

The password row sits with the rest of the form and is off until you ask for it.

<figure class="shot-todo" data-shot="password-row.png">
  <figcaption>Password row — Set password checkbox, masked field with Copy and reveal, Generate button</figcaption>
</figure>

## Set password

Nothing happens to the password unless **Set password** is ticked. Until then the field,
**Generate**, **Copy**, and the reveal eye are all disabled — so you can't set a password by
accident while editing something else.

| Control | Effect |
|---------|--------|
| **Set password** | Arms the row. Only then is a password sent. |
| Field | Masked. Type your own, or use Generate. |
| 👁 reveal | Shows the value while you check it. |
| **Copy** | Copies the value even while masked, so you can paste it into a ticket or vault. |
| **Generate** | Fills the field with a fresh random password. |

## Generate

**Generate** builds a password from your
[password generator settings](/pg-accounts-management/configuration/password-generator/) —
length, character classes, and whether look-alike characters are excluded — using the
platform's cryptographic random source. Press it again for a different one.

Copy the password before you save. The app never displays it again.

## Where it applies

On **Save changes** the password is set on **every cluster where the role exists**, as part of
that cluster's transaction. It runs through the `change_password`
[call template](/pg-accounts-management/configuration/call-templates/), which is
`ALTER ROLE … PASSWORD …` by default.

Passwords are not stored in the app's configuration, and they are not shown in the
[command log](/pg-accounts-management/usage/command-log/).
