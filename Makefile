.PHONY: all

GITV := $(shell git describe --tags --always --dirty)
VERSION := $(if ${GITV},${GITV},none)

all:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags="-w -extldflags -static -X main.version=${VERSION}" -o meshsearch .
