---
title: Building from source
description: Build and run DbAccounts locally
---

## Prerequisites

- Go 1.22+
- [Wails v2](https://wails.io/docs/gettingstarted/installation)
- Platform WebView dependencies: Xcode Command Line Tools (macOS), WebView2 (Windows), or
  `webkit2gtk` (Linux).

## Clone and run

```bash
git clone https://github.com/michal-bartak/pg-accounts-management.git
cd pg-accounts-management
go mod tidy

# Development window with live reload:
wails dev
```

## Build a release bundle

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
make package    # release build + dist/DbAccounts-v*.tar.gz
```

Other useful targets:

```bash
make test               # go test ./... -count=1
make test-vet           # tests + frontend checks + go vet
make version            # print the app version
make build              # build the app bundle (with tests)
make sync-wails-version # align wails.json with VERSION
```

## Running the tests

```bash
make test
```

CI runs the same checks on every push and pull request. The suite covers call-template SQL
generation, comment parsing, command validation, config migration, and batch target
resolution. Database calls need a live server, so those paths are exercised against a
throwaway PostgreSQL when needed.

## Working on these docs

The docs are an [Astro](https://astro.build/) + [Starlight](https://starlight.astro.build/)
site under `docs/`. To preview them offline:

```bash
make docs-install   # one-time: install the docs dependencies
make docs-dev       # live-reload dev server (http://localhost:4321/pg-accounts-management)
# or:
make docs-build     # static build into docs/dist/
make docs-preview   # serve the built site locally
```

On push to `main`, the `deploy-docs` workflow builds the site and publishes it to GitHub
Pages.

### Screenshots

Every figure is a light/dark pair under `docs/src/assets/<section>/`, named after the figure:

```text
docs/src/assets/usage/target-selection-light.png
docs/src/assets/usage/target-selection-dark.png
```

Starlight's theme toggle picks one, so **replacing a screenshot is just overwriting the file
and rebuilding** — no markdown to edit. Capture both themes with the same window size and
scroll position, or the swap reads as a page jump instead of a theme change.

Files not captured yet hold a generated placeholder that names itself, so a missing shot never
breaks the build:

```bash
make docs-shots-status   # which screenshots are real, which are still placeholders
make docs-shots          # generate placeholders for newly referenced images
```

`docs-shots` also runs automatically before `docs-dev` and `docs-build`, so adding a figure to
a page and rebuilding is enough to get its placeholder.
