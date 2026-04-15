BINARY   := rebackup
PKG      := rebackup
LDFLAGS  := -s -w
INSTALL  := /usr/local/bin

.PHONY: build install uninstall clean test vet lint help

## build: compile the binary to ./rebackup
build:
	go build -ldflags="$(LDFLAGS)" -o $(BINARY) .

## install: build and install to /usr/local/bin
install: build
	install -m 0755 $(BINARY) $(INSTALL)/$(BINARY)
	@echo "Installed $(INSTALL)/$(BINARY)"

## uninstall: remove from /usr/local/bin
uninstall:
	rm -f $(INSTALL)/$(BINARY)
	@echo "Removed $(INSTALL)/$(BINARY)"

## clean: remove build artefacts
clean:
	rm -f $(BINARY)

## test: run all unit tests
test:
	go test -v -race ./...

## vet: run go vet
vet:
	go vet ./...

## lint: run golangci-lint (must be installed separately)
lint:
	golangci-lint run ./...

## help: show this message
help:
	@grep -E '^## ' Makefile | sed 's/## /  /'
