SHELL := /bin/bash

ifndef FH_SYSTEM_DUMP_TOOL_VERSION
FH_SYSTEM_DUMP_TOOL_VERSION := $(shell git describe --tags --abbrev=14)
endif
LDFLAGS := -X main.Version=$(FH_SYSTEM_DUMP_TOOL_VERSION)

IMPORT_PATH := github.com/feedhenry/fh-system-dump-tool
GO_DOCKER_IMAGE := golang:1.8

.PHONY: all
all:
	@go install -v -ldflags '$(LDFLAGS)'

.PHONY: clean
clean:
	@-go clean -i

.PHONY: ci
ci: check-gofmt check-goimports check-golint vet test test-race

# goimports doesn't support the -s flag to simplify code, therefore we use both
# goimports and gofmt -s.
.PHONY: check-gofmt
check-gofmt:
	diff <(gofmt -s -d .) <(printf "")

.PHONY: check-goimports
check-goimports:
	diff <(goimports -d .) <(printf "")

.PHONY: check-golint
check-golint:
	diff <(golint ./...) <(printf "")

.PHONY: vet
vet:
	go vet ./...

.PHONY: test
test:
	go test -v -cpu=2 ./...

.PHONY: test-race
test-race:
	go test -v -cpu=1,2,4 -short -race ./...

.PHONY: release
release:
	docker run --rm \
		-v "$(PWD):/go/src/$(IMPORT_PATH)" \
		-w "/go/src/$(IMPORT_PATH)" \
		"$(GO_DOCKER_IMAGE)" make dist

.PHONY: dist
dist: all
	mkdir -p dist
	tar -C "/go/bin" -czf "dist/fh-system-dump-tool-$(FH_SYSTEM_DUMP_TOOL_VERSION)-linux-amd64.tar.gz" "fh-system-dump-tool"
