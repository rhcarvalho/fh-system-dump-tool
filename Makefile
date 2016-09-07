ifndef FH_SYSTEM_DUMP_TOOL_VERSION
FH_SYSTEM_DUMP_TOOL_VERSION := $(shell git describe --tags --abbrev=14)
endif
LDFLAGS := -X main.Version=$(FH_SYSTEM_DUMP_TOOL_VERSION)

.PHONY: all
all:
	@go install -v -ldflags '$(LDFLAGS)'

.PHONY: clean
clean:
	@-go clean -i
