BINARY := yaawp-daemon
PREFIX ?= $(HOME)/.local

.PHONY: all build run tidy clean install

all: build

tidy:
	go get go.mau.fi/whatsmeow@latest github.com/mattn/go-sqlite3@latest google.golang.org/protobuf@latest
	go mod tidy

build:
	go build -o bin/$(BINARY) ./cmd/yaawp-daemon

run: build
	./bin/$(BINARY)

install: build
	install -Dm755 bin/$(BINARY) $(PREFIX)/bin/$(BINARY)

clean:
	rm -rf bin
