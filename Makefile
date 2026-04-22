BINARY  := wave
PKG     := ./cmd/wave
PREFIX  ?= $(HOME)/.local

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

.PHONY: build install test coverage lint clean

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
