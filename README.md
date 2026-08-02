# DbAccounts

A cross-platform desktop app for maintaining **PostgreSQL roles across many clusters**. Create and alter roles on all your selected clusters from one form, through SQL call templates you can read and edit. Built with Go, Wails v2, and pgx.

📖 **[Full documentation](https://michal-bartak.github.io/pg-accounts-management/)**

## Quick start

Download the latest release for your platform from the [Releases page](../../releases), then follow the [Installation guide](https://michal-bartak.github.io/pg-accounts-management/installation/) for platform notes and how credentials are resolved (an optional per-cluster user/password, else `PGUSER`/`PGPASSWORD`, else `~/.pgpass`, like `psql`).

**Developers:**

```bash
git clone https://github.com/michal-bartak/pg-accounts-management.git
cd pg-accounts-management
go mod tidy
wails dev
```

See [Building from source](https://michal-bartak.github.io/pg-accounts-management/building/) for requirements and `make` targets.

## License

MIT © 2026 Michal Bartak.

---

*Built by Michal Bartak, assisted by [Claude](https://claude.ai).*
