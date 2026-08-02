---
title: Preconfigured parent groups
description: Role names offered as quick picks when granting privileges
---

A short list of role names you grant often. They appear as **toggle chips** in the Create-role
form and in the Add-privilege dialog, so you can pick several at once instead of typing them.

<figure class="shot">
<div class="light-only">

![Settings → Preconfigured parent groups](../../../assets/configuration/settings-parent-roles-light.png)

</div>
<div class="dark-only">

![Settings → Preconfigured parent groups](../../../assets/configuration/settings-parent-roles-dark.png)

</div>
<figcaption>Settings → Preconfigured parent groups — list with drag handles and Add group</figcaption>
</figure>

- Each entry is a bare role identifier. Invalid names are rejected on save, duplicates are
  dropped. <svg class="doc-ic" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg> removes one.
- Drag the handle to reorder — the order is the chip order.
- The list is a convenience, not a restriction: you can still grant any role by typing it.

<figure class="shot">
<div class="light-only">

![Add privilege dialog](../../../assets/configuration/parent-role-chips-light.png)

</div>
<div class="dark-only">

![Add privilege dialog](../../../assets/configuration/parent-role-chips-dark.png)

</div>
<figcaption>Add privilege dialog — preconfigured groups shown as selectable chips</figcaption>
</figure>

Selected parents feed the `${parent_roles}` placeholder in both the `create_role` and
`grant_parents` [call templates](/pg-accounts-management/configuration/call-templates/).
