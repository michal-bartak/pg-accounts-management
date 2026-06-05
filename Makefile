.PHONY: version sync-wails-version test test-vet build build-ci package package-ci clean dist

VERSION := $(shell tr -d ' \n\r' < VERSION)
VERSION_PKG := github.com/michalbartak/dbaccounts/internal/version
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
WAILS := $(shell go env GOPATH)/bin/wails

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

$(WAILS):
	go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0

test:
	go test ./... -count=1

test-vet: test
	go vet ./...

# Production app bundle (macOS: build/bin/DbAccounts.app). Requires Wails CLI.
build: sync-wails-version test-vet $(WAILS)
	$(WAILS) build -ldflags "$(LDFLAGS)"
	@echo "Built DbAccounts $(VERSION) ($(GIT_COMMIT)) -> build/bin/"

# CI build (no tests; test job gates release pipeline).
build-ci: sync-wails-version $(WAILS)
	$(WAILS) build -ldflags "$(LDFLAGS)"
	@echo "Built DbAccounts $(VERSION) ($(GIT_COMMIT)) -> build/bin/"

# Archive for distribution under dist/ (adjust platform when cross-compiling).
package: build
	$(package_dist)

package-ci: build-ci
	$(package_dist)

dist: package

clean:
	rm -rf build/bin dist/*.tar.gz dist/*.exe dist/*.zip
