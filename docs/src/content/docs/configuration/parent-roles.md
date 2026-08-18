---
title: Preconfigured role parents
description: Role names offered as quick picks when assigning role parents
---

A short list of role names you grant often. They appear as **toggle chips** in the [Assign-parents](/pgcowboy/usage/parent-roles/#new-parents) dialog, so you can pick several at once instead of typing them.

<figure class="shot">
<div class="light-only">

![Settings → Preconfigured role parents](../../../assets/configuration/settings-parent-roles-light.png)

</div>
<div class="dark-only">

![Settings → Preconfigured role parents](../../../assets/configuration/settings-parent-roles-dark.png)

</div>
<figcaption>Preconfigured role parents settings</figcaption>
</figure>

- Each entry is a bare role identifier. Invalid names are rejected on save, duplicates are dropped.
- <svg class="doc-ic" width="1.05em" height="1.05em" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg> removes one.
- Drag the handle to reorder — the order is the chip order.

:::tip
The list is a convenience, not a restriction. Any role can be a parent, and you can still assign one by typing its name — several at once, separated by commas — mixing typed names with picked chips.
:::
