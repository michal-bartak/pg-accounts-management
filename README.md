# pgCowboy

A cross-platform desktop app for maintaining **PostgreSQL roles across many clusters**. Create and alter roles on all your selected clusters from one form, through SQL call templates you can read and edit. Built with Go, Wails v2, and pgx.

📖 **[Full documentation](https://michal-bartak.github.io/pgcowboy/)**

## Quick start

Download the installer for your platform from the [Releases page](../../releases) — `.dmg` (macOS), `.msi` (Windows), `.deb` or `.rpm` (Linux) — then follow the [Installation guide](https://michal-bartak.github.io/pgcowboy/installation/) for platform notes and how credentials are resolved (an optional per-cluster user/password, else `PGUSER`/`PGPASSWORD`, else `~/.pgpass`, like `psql`).

Requires macOS 12+, Windows 10/11 (WebView2), or Linux with WebKitGTK 2.38+ — see [System requirements](https://michal-bartak.github.io/pgcowboy/installation/#system-requirements).

What changed in each version: [CHANGELOG.md](CHANGELOG.md).

**Developers:**

```bash
git clone https://github.com/michal-bartak/pgcowboy.git
cd pgcowboy
go mod tidy
wails dev
```

See [Building from source](https://michal-bartak.github.io/pgcowboy/building/) for requirements and `make` targets.

## License

MIT © 2026 Michal Bartak.

---

*Built by Michal Bartak, assisted by [Claude](https://claude.ai).*
