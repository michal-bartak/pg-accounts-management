.PHONY: version sync-wails-version ensure-wails test test-frontend test-vet build build-ci package package-ci clean dist docs-install docs-dev docs-build docs-preview docs-clean docs-lint

VERSION := $(shell tr -d ' \n\r' < VERSION)
VERSION_PKG := github.com/michalbartak/dbaccounts/internal/version
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
# Repo URL from the git remote (any form); version.normalizeRepo canonicalizes it, and the
# GitHub Pages docs URL is derived from it. Empty (no remote) → the built-in default is kept.
GIT_REMOTE := $(shell git config --get remote.origin.url 2>/dev/null)
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
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

LDFLAGS := -s -w \
	-X $(VERSION_PKG).Version=$(VERSION) \
	-X $(VERSION_PKG).Commit=$(GIT_COMMIT) \
	-X $(VERSION_PKG).BuildDate=$(BUILD_DATE)
ifneq ($(strip $(GIT_REMOTE)),)
LDFLAGS += -X $(VERSION_PKG).Repo=$(GIT_REMOTE)
endif

define package_dist
	@mkdir -p dist
	@if [ -d build/bin/DbAccounts.app ]; then \
		tar -czf dist/DbAccounts-v$(VERSION)-$(GOOS)-$(GOARCH).tar.gz -C build/bin DbAccounts.app; \
		echo "dist/DbAccounts-v$(VERSION)-$(GOOS)-$(GOARCH).tar.gz"; \
	elif [ -f build/bin/DbAccounts.exe ]; then \
		cp build/bin/DbAccounts.exe dist/DbAccounts-v$(VERSION)-$(GOOS)-$(GOARCH).exe; \
		echo "dist/DbAccounts-v$(VERSION)-$(GOOS)-$(GOARCH).exe"; \
	elif [ -f build/bin/DbAccounts ]; then \
		tar -czf dist/DbAccounts-v$(VERSION)-$(GOOS)-$(GOARCH).tar.gz -C build/bin DbAccounts; \
		echo "dist/DbAccounts-v$(VERSION)-$(GOOS)-$(GOARCH).tar.gz"; \
	else \
		echo "No build artifact in build/bin/"; exit 1; \
	fi
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
	"$(WAILS)" build $(WAILS_BUILD_FLAGS) -ldflags "$(LDFLAGS)"
	@echo "Built DbAccounts $(VERSION) ($(GIT_COMMIT)) -> build/bin/"

# CI build (no tests; test job gates release pipeline).
build-ci: sync-wails-version ensure-wails
	"$(WAILS)" build $(WAILS_BUILD_FLAGS) -ldflags "$(LDFLAGS)"
	@echo "Built DbAccounts $(VERSION) ($(GIT_COMMIT)) -> build/bin/"

# Archive for distribution under dist/ (adjust platform when cross-compiling).
package: build
	$(package_dist)

package-ci: build-ci
	$(package_dist)

dist: package

clean:
	rm -rf build/bin dist/*.tar.gz dist/*.exe dist/*.zip

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

# Lint the docs prose (Vale) and structure (markdownlint). Config: .vale.ini,
# .markdownlint.yaml. One-time setup: `brew install vale && vale sync`.
docs-lint:
	@command -v vale >/dev/null 2>&1 || { echo "Vale not found — run: brew install vale && vale sync"; exit 1; }
	vale docs/src/content/docs
	npx --yes markdownlint-cli2 "docs/src/content/docs/**/*.md"
