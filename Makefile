# Build, test, and tooling targets for Wave.
# Front-end / Tailwind regen workflow is documented in docs/build.md.

BINARY  := wave
PKG     := ./cmd/wave
PREFIX  ?= $(HOME)/.local

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

# Tailwind standalone CLI (pinned). The compiled CSS is committed under
# internal/webui/static/tailwind.css so plain `go build` works without Node
# or this binary; `make tailwind` regenerates, `make tailwind-check` enforces
# sync in CI. See docs/build.md.
TAILWIND_VERSION := v3.4.17
TAILWIND_OS      := $(shell uname -s | tr '[:upper:]' '[:lower:]')
TAILWIND_ARCH    := $(shell uname -m)
ifeq ($(TAILWIND_ARCH),x86_64)
TAILWIND_ARCH := x64
endif
ifeq ($(TAILWIND_ARCH),aarch64)
TAILWIND_ARCH := arm64
endif
TAILWIND_BIN := tools/tailwindcss-$(TAILWIND_VERSION)-$(TAILWIND_OS)-$(TAILWIND_ARCH)
TAILWIND_URL := https://github.com/tailwindlabs/tailwindcss/releases/download/$(TAILWIND_VERSION)/tailwindcss-$(TAILWIND_OS)-$(TAILWIND_ARCH)
TAILWIND_INPUT  := internal/webui/tailwind.input.css
TAILWIND_OUTPUT := internal/webui/static/tailwind.css
TAILWIND_CONFIG := internal/webui/tailwind.config.js

.PHONY: build install test coverage lint clean tailwind tailwind-check

# NOTE: Running `go build ./cmd/wave` directly will produce a binary that
# reports version as "dev". Always use `make build` or pass ldflags manually:
#   go build -ldflags "-X main.version=... -X main.commit=... -X main.date=..." ./cmd/wave
build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(PKG)

install: build
	install -d $(PREFIX)/bin
	cp -f $(BINARY) $(PREFIX)/bin/$(BINARY)
	chmod 755 $(PREFIX)/bin/$(BINARY)

test:
	go test -race ./...

coverage:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -n 1

lint:
	golangci-lint run ./...

clean:
	rm -f $(BINARY)

# Download the pinned standalone Tailwind CLI binary into tools/ if missing.
$(TAILWIND_BIN):
	@mkdir -p tools
	@echo "Downloading Tailwind CLI $(TAILWIND_VERSION) for $(TAILWIND_OS)-$(TAILWIND_ARCH)..."
	@curl -fsSL -o $(TAILWIND_BIN) $(TAILWIND_URL)
	@chmod +x $(TAILWIND_BIN)

# Compile internal/webui/static/tailwind.css from the templates content scan.
# The output is committed; CI runs `tailwind-check` to enforce sync.
tailwind: $(TAILWIND_BIN)
	cd internal/webui && ../../$(TAILWIND_BIN) \
		--config tailwind.config.js \
		--input  tailwind.input.css \
		--output static/tailwind.css \
		--minify

# Regenerate and fail if the committed CSS drifts from templates.
tailwind-check: tailwind
	@git diff --exit-code -- $(TAILWIND_OUTPUT) || { \
		echo "ERROR: $(TAILWIND_OUTPUT) is out of sync. Run 'make tailwind' and commit."; \
		exit 1; \
	}
