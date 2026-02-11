BINARY  := wave
PKG     := ./cmd/wave
PREFIX  ?= $(HOME)/.local

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

.PHONY: build install test lint clean

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(PKG)

install: build
	install -d $(PREFIX)/bin
	cp -f $(BINARY) $(PREFIX)/bin/$(BINARY)
	chmod 755 $(PREFIX)/bin/$(BINARY)

test:
	go test -race ./...

lint:
	go vet ./...

clean:
	rm -f $(BINARY)
