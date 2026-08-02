---
title: Password generator
description: What the Generate button produces
---

Controls the random password produced by **Generate** on the role form's password field.

<figure class="shot-todo" data-shot="settings-pwgen.png">
  <figcaption>Settings → Password generator — character classes, look-alike exclusion, length</figcaption>
</figure>

| Option | Notes |
|--------|-------|
| **Lowercase letters (a–z)** | On by default. |
| **Uppercase letters (A–Z)** | |
| **Digits (0–9)** | |
| **Symbols (!@#$…)** | |
| **Exclude look-alike characters** | Drops `i l 1 I o O 0`, for passwords that get read aloud or retyped. |
| **Length** | Between 6 and 128. |

At least one class stays enabled — turn them all off and lowercase comes back on.

Passwords are generated with the platform's cryptographic random source. See
[Setting a password](/pg-accounts-management/usage/password/) for the field itself.
