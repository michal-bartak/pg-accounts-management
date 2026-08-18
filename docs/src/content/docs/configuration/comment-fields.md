---
title: Comment fields
description: Which JSON keys in a role comment get labelled inputs
---

In PostgreSQL a role comment lives as plain text. But we can store an JSON formatted string there. **Comment fields** defines the form that helps editing json content.

<figure class="shot">
<div class="light-only">

![Settings → Comments](../../../assets/configuration/settings-comments-light.png)

</div>
<div class="dark-only">

![Settings → Comments](../../../assets/configuration/settings-comments-dark.png)

</div>
<figcaption>Settings → Comments — the field list with drag handles, Add field, and Preferred comment view</figcaption>
</figure>

## The field list

Each entry is a **key** (the JSON key, a bare identifier) and a **label** (what the form shows). Drag the handle to reorder;
<svg class="doc-ic" width="1.05em" height="1.05em" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg> removes an entry; **Add field…** appends one.

Rules:

- Configured fields found in the comment always render, even when the key is absent from the comment.
- Keys **not** found in the comments but missing from the list still render, labelled by their raw key.
- Values that aren't strings (number, boolean, array, object) render **read-only**, so their type survives a round trip. Edit them in the Raw view.
- Leaving the label blank falls back to the key.

:::tip
Each configured key also becomes a placeholder — `${{key}}` — usable in the `create_role` and `set_comment` [call templates](/pgcowboy/configuration/call-templates/) or [role details](/pgcowboy/configuration/role-details/) feature.
:::

## Preferred comment view

Sets which mode a **new or empty** comment opens in — **Fields** or **Raw**. Comments that already have content detect their own mode: JSON opens in Fields, anything else in Raw.

See [Altering the comment](/pgcowboy/usage/comments/) for the editor itself.

## Where else these keys appear

Beyond the role form, a comment key can be shown in the role search results — see [Role Details](/pgcowboy/configuration/role-details/).

:::tip
That list is not limited to the keys configured here — it can address **any** key in the comment.
:::
