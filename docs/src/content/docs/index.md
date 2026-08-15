---
title: pgCowboy
description: Maintain PostgreSQL roles across many clusters from one desktop app
---

pgCowboy is a desktop app for maintaining **PostgreSQL roles across many clusters**.
Keep your clusters grouped — for example into Production and UAT — then pick the group(s) and add or adjust a role against them.

Thanks to **templating** of SQL queries, you can extend the basic operations, and give access to users who could not otherwise execute them.

<figure class="shot">
<div class="light-only">

![pgCowboy — Alter role](../../assets/usage/operations-overview-light.png)

</div>
<div class="dark-only">

![pgCowboy — Alter role](../../assets/usage/operations-overview-dark.png)

</div>
<figcaption>Altering a role across Production and UAT from one form</figcaption>
</figure>

## Main Features

- **Clusters grouping** - aggregate clusters into color-coded groups. Mark the groups that require extra confirmation.
- **Operations on groups** - combine groups (and single clusters) as operation targets.
- **PGUSER, PGPASSWORD and pgpass** support.
- **Differences visual feedback** - differences in role settings between groups or clusters are easy to spot and even easier to reconcile.
- **Custom information fields** - a role comment can be treated as a JSON object that carries customizable information.
- **Preflight checks** - before role creation or removal, the app checks for possible dependencies and shows the report.
- **Templatable SQL** - every operation the app issues against a database is configurable, so you can extend both what it does and who can do it.
- **Password generator** - configurable password generator.

## How a change is applied

1. Choose the target groups and/or single clusters in **Target selection**.
1. Select **Create role** to clear the form, or **Alter role** to find and load role data from all selected clusters.
1. Edit in the form (parents, attributes, settings, comment, password).
1. On **Save**, the app
   - performs pre-flight checks
   - builds an ordered list of operations per cluster and runs each cluster's list as **one transaction** — commit on success, roll back on the first error.
1. A report is available afterwards. It includes the status and the SQL commands issued against each cluster.

## Where to go next

- [Installation](/pgcowboy/installation/) — download a build or install from source.
- [Usage](/pgcowboy/usage/) — target selection, scope labels, and how a change is applied.
- [Configuration](/pgcowboy/configuration/) — clusters, comment fields, templates, preferences.
- [Troubleshooting](/pgcowboy/troubleshooting/) — connection and permission issues.
