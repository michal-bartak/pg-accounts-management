.PHONY: version sync-wails-version ensure-wails test test-frontend test-vet build build-ci package package-ci clean dist docs-install docs-dev docs-build docs-preview docs-clean docs-lint docs-shots docs-shots-status

APP := DbAccounts
VERSION := $(shell tr -d ' \n\r' < VERSION)
VERSION_PKG := github.com/michalbartak/dbaccounts/internal/version
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
# Repo URL from the git remote (any form); version.normalizeRepo canonicalizes it, and the
# GitHub Pages docs URL is derived from it. Empty (no remote) → the built-in default is kept.
GIT_REMOTE := $(shell git config --get remote.origin.url 2>/dev/null)
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

# Cross/universal builds: make package PLATFORM=darwin/universal (empty = host platform).
# The trailing element of PLATFORM also names the artifact (…-macos-universal.dmg).
PLATFORM ?=
ifneq ($(strip $(PLATFORM)),)
WAILS_PLATFORM_FLAG := -platform $(PLATFORM)
ARCH_LABEL := $(lastword $(subst /, ,$(PLATFORM)))
else
ARCH_LABEL := $(GOARCH)
endif
GOPATH_DIR := $(subst \,/,$(shell go env GOPATH))
ifeq ($(GOOS),windows)
WAILS := $(GOPATH_DIR)/bin/wails.exe
else
WAILS := $(GOPATH_DIR)/bin/wails
endif

# Ubuntu 24.04+ ships webkit2gtk 4.1 only; Wails defaults to 4.0 without this tag.
ifeq ($(GOOS),linux)
ifneq ($(shell pkg-config --exists webkit2gtk-4.1 2>/dev/null && echo yes),)
WAILS_BUILD_FLAGS := -tags webkit2_41
endif
endif

# Version comes from the embedded VERSION file (main.go //go:embed), not ldflags — so it's
# correct for go run / wails dev too. ldflags supply only the build metadata.
LDFLAGS := -s -w \
	-X $(VERSION_PKG).Commit=$(GIT_COMMIT) \
	-X $(VERSION_PKG).BuildDate=$(BUILD_DATE)
ifneq ($(strip $(GIT_REMOTE)),)
LDFLAGS += -X $(VERSION_PKG).Repo=$(GIT_REMOTE)
endif

# Native installer for the host OS: .dmg (macOS), .msi (Windows), .deb + .rpm (Linux).
# One script per platform under build/scripts/, shared with the release workflow so a local
# package and a CI package are built the same way. Prerequisites: WiX Toolset (Windows),
# fpm + rpm (Linux); macOS needs only the system tools.
define package_dist
	@mkdir -p dist
	@case "$(GOOS)" in \
	  darwin) OUTPUT=$(APP) VERSION=$(VERSION) ARCH_LABEL=$(ARCH_LABEL) build/scripts/make-dmg.sh ;; \
	  linux) OUTPUT=$(APP) VERSION=$(VERSION) ARCH_LABEL=$(ARCH_LABEL) build/scripts/make-linux-packages.sh ;; \
	  windows) powershell -ExecutionPolicy Bypass -File build/scripts/make-msi.ps1 \
	             -Output $(APP) -Version $(VERSION) -ArchLabel $(ARCH_LABEL) ;; \
	  *) echo "No installer recipe for GOOS=$(GOOS)"; exit 1 ;; \
	esac
endef

# Print application version (matches VERSION file / git tag v$(VERSION)).
version:
	@echo $(VERSION)

# Align wails.json productVersion with VERSION (run before release build).
sync-wails-version:
	@python3 -c "import json, pathlib; v=pathlib.Path('VERSION').read_text().strip(); p=pathlib.Path('wails.json'); w=json.loads(p.read_text()); w.setdefault('info', {})['productVersion']=v; p.write_text(json.dumps(w, indent=2)+'\n')"
	@echo "wails.json productVersion -> $(VERSION)"

ensure-wails:
	@if [ ! -f "$(WAILS)" ]; then go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0; fi

test:
	go test ./... -count=1

# Frontend logic tests (Node built-in runner, no deps). Skipped with a note if node is absent.
test-frontend:
	@if command -v node >/dev/null 2>&1; then \
		node --check frontend/app.js && node --test frontend/app.test.mjs; \
	else \
		echo "skipping frontend tests: node not found"; \
	fi

test-vet: test test-frontend
	go vet ./...

# Production app bundle (macOS: build/bin/DbAccounts.app). Requires Wails CLI.
build: sync-wails-version test-vet ensure-wails
	"$(WAILS)" build $(WAILS_BUILD_FLAGS) $(WAILS_PLATFORM_FLAG) -ldflags "$(LDFLAGS)"
	@echo "Built DbAccounts $(VERSION) ($(GIT_COMMIT)) -> build/bin/"

# CI build (no tests; test job gates release pipeline).
build-ci: sync-wails-version ensure-wails
	"$(WAILS)" build $(WAILS_BUILD_FLAGS) $(WAILS_PLATFORM_FLAG) -ldflags "$(LDFLAGS)"
	@echo "Built DbAccounts $(VERSION) ($(GIT_COMMIT)) -> build/bin/"

# Installer for distribution under dist/ (see package_dist above for prerequisites).
package: build
	$(package_dist)

package-ci: build-ci
	$(package_dist)

dist: package

clean:
	rm -rf build/bin dist

# ---- Docs (Astro + Starlight): read them offline, or let CI publish to GitHub Pages ----
DOCS_DIR := docs

# Install docs dependencies only when they're missing (Node + npm required).
$(DOCS_DIR)/node_modules:
	cd $(DOCS_DIR) && npm install

# Explicit one-time install target.
docs-install: $(DOCS_DIR)/node_modules

# Live-reload dev server at http://localhost:4321/pg-accounts-management
docs-dev: $(DOCS_DIR)/node_modules
	cd $(DOCS_DIR) && npm run dev

# Static build into docs/dist/
docs-build: $(DOCS_DIR)/node_modules
	cd $(DOCS_DIR) && npm run build

# Serve the built site locally (build first if needed)
docs-preview: docs-build
	cd $(DOCS_DIR) && npm run preview

docs-clean:
	rm -rf $(DOCS_DIR)/dist $(DOCS_DIR)/.astro

# Screenshots. Every figure references a real .png under docs/src/assets, so replacing one is
# "overwrite the file, rebuild" — no markdown edit. Files not captured yet hold a generated
# placeholder naming themselves, so the build never breaks on a missing shot.
#   docs-shots         create placeholders for any newly referenced image (also runs on dev/build)
#   docs-shots-status  list which screenshots are real and which are still placeholders
docs-shots: $(DOCS_DIR)/node_modules
	cd $(DOCS_DIR) && npm run shots

docs-shots-status: $(DOCS_DIR)/node_modules
	cd $(DOCS_DIR) && npm run shots:status

# Lint the docs prose (Vale) and structure (markdownlint). Config: .vale.ini,
# .markdownlint.yaml. One-time setup: `brew install vale && vale sync`.
docs-lint:
	@command -v vale >/dev/null 2>&1 || { echo "Vale not found — run: brew install vale && vale sync"; exit 1; }
	vale docs/src/content/docs
	npx --yes markdownlint-cli2 "docs/src/content/docs/**/*.md"
