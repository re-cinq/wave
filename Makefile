BINARY  := wave
PKG     := ./cmd/wave
PREFIX  ?= $(HOME)/.local

.PHONY: build install test lint clean

build:
	go build -o $(BINARY) $(PKG)

install: build
	install -d $(PREFIX)/bin
	install -m 755 $(BINARY) $(PREFIX)/bin/$(BINARY)

test:
	go test -race ./...

lint:
	go vet ./...

clean:
	rm -f $(BINARY)
