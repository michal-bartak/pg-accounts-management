---
title: Command log
description: Live progress, per-cluster results, and the exact SQL that ran
---

Results don't appear in the form. They go to the **status chip** in the footer, next to the
action buttons.

<figure class="shot-todo" data-shot="run-status-chip.png">
  <figcaption>Footer status chip — running, then OK or Error</figcaption>
</figure>

The chip is hidden until something runs, then shows `running… (done/total)` and settles on
**OK** or **Error**. It updates live, one step per cluster. It also reports **role loads**, so
a cluster that was unreachable while reading the role is surfaced the same way as a failed
write.

## Per-cluster results

Click the chip to open the results panel. It fills in while the run is still going.

<figure class="shot-todo" data-shot="run-status-dialog.png">
  <figcaption>Results panel — one row per cluster: Cluster, Category, Status, Duration, Message</figcaption>
</figure>

| Column | Notes |
|--------|-------|
| **Cluster** / **Category** | Which target the row is for. |
| **Status** | Per cluster — one failure doesn't hide the others' success. |
| **Duration** | How long that cluster took. |
| **Message** | The error text, naming the operation that failed. |

Each row has two actions: a **magnifier** and a **copy** button. Copy puts the cluster's
message and all of its SQL on the clipboard.

## The SQL that ran

The magnifier opens the statements the app sent to that cluster, in order.

<figure class="shot-todo" data-shot="run-queries-dialog.png">
  <figcaption>Executed SQL for one cluster, with its own Copy button</figcaption>
</figure>

- The SQL is the real thing, after placeholder substitution — including a failing statement's,
  which is recorded before the error.
- Function-mode bind values are inlined so the statement reads as executed.
- Role-load rows show the introspection queries instead.
- Passwords are not included.

The log is cleared when you switch tabs, so results never leak into the next task.
