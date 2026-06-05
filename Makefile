.PHONY: version sync-wails-version ensure-wails test test-vet build build-ci package package-ci clean dist

VERSION := $(shell tr -d ' \n\r' < VERSION)
VERSION_PKG := github.com/michalbartak/dbaccounts/internal/version
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
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

test-vet: test
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
