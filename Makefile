MODULE_NAME = cookiejar

GOLANGCI_LINT_VERSION ?= v2.13.0
MOCKERY_VERSION ?= v3.7.4

GO ?= go
GOLANGCI_LINT ?= $(shell go env GOPATH)/bin/golangci-lint-$(GOLANGCI_LINT_VERSION)
MOCKERY ?= $(shell go env GOPATH)/bin/mockery-$(MOCKERY_VERSION)

VENDOR_DIR = vendor
GOROOT_DIR = $(shell $(GO) env GOROOT)

GITHUB_OUTPUT ?= /dev/null

# Other config
NO_COLOR=\033[0m
OK_COLOR=\033[32;01m
ERROR_COLOR=\033[31;01m
WARN_COLOR=\033[33;01m

ifeq ($(V),1)
  Q = @set -x;
else
  Q = @
endif

.PHONY: $(VENDOR_DIR)
$(VENDOR_DIR):
	$(Q)mkdir -p $(VENDOR_DIR)
	$(Q)$(GO) mod vendor
	$(Q)$(GO) mod tidy

.PHONY: bump-deps
bump-deps:
	$(Q)$(GO) get -u ./...

.PHONY: tidy
tidy:
	$(Q)$(GO) mod tidy

.PHONY: lint
ifeq ($(V),1)
  GOLANGCI_LINT_FLAGS = -vvvv
else
  GOLANGCI_LINT_FLAGS =
endif

lint: $(GOLANGCI_LINT)
	@printf -- "$(OK_COLOR)==> lint$(NO_COLOR)\n"
	$(Q)GOROOT=$(GOROOT_DIR) PATH="$(GOROOT_DIR)/bin:$$PATH" $(GOLANGCI_LINT) run -c .golangci.yaml --color always $(GOLANGCI_LINT_FLAGS)

.PHONY: test
test: test-unit

## Run unit tests
.PHONY: test-unit
test-unit:
	@printf -- "$(OK_COLOR)==> unit test$(NO_COLOR)\n"
	$(Q)$(GO) test -gcflags=-l -coverprofile=unit.coverprofile -covermode=atomic -race ./...


.PHONY: generate
generate: generate-jar generate-mocks

.PHONY: generate-mocks
generate-mocks: $(MOCKERY)
	@printf -- "$(OK_COLOR)==> generate mocks$(NO_COLOR)\n"
	$(Q)GOROOT=$(GOROOT_DIR) PATH="$(GOROOT_DIR)/bin:$$PATH" $(MOCKERY)

.PHONY: generate-jar
generate-jar:
	@printf -- "$(OK_COLOR)==> sync$(NO_COLOR)\n"
	$(Q)ls -1 *.go | grep -v persistent_jar | xargs rm -f
	$(Q)rm -f internal/ascii/*
	$(Q)cp $(shell $(GO) env GOROOT)/src/net/http/cookiejar/*.go ./
	$(Q)cp $(shell $(GO) env GOROOT)/src/net/http/internal/ascii/*.go ./internal/ascii/
	$(Q)sed -i '' -E 's#net/http/internal/ascii#go.nhat.io/$(MODULE_NAME)/internal/ascii#g' *.go
	$(Q)gofumpt -l -w .

.PHONY: $(GITHUB_OUTPUT)
$(GITHUB_OUTPUT):
	$(Q)echo "MODULE_NAME=$(MODULE_NAME)" >> "$@"
	$(Q)echo "GOLANGCI_LINT_VERSION=$(GOLANGCI_LINT_VERSION)" >> "$@"

$(GOLANGCI_LINT):
	@printf -- "$(OK_COLOR)==> Installing golangci-lint $(GOLANGCI_LINT_VERSION)$(NO_COLOR)\n"
	$(Q)curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b /tmp "$(GOLANGCI_LINT_VERSION)"
	$(Q)$(call install-dep,/tmp/golangci-lint,$(GOLANGCI_LINT))

$(MOCKERY):
	@printf -- "$(OK_COLOR)==> Installing mockery $(MOCKERY_VERSION)$(NO_COLOR)\n"
	$(Q)GOBIN=/tmp $(GO) install github.com/vektra/mockery/$(shell echo "$(MOCKERY_VERSION)" | cut -d '.' -f 1)@$(MOCKERY_VERSION)
	$(Q)$(call install-dep,/tmp/mockery,$(MOCKERY))

define install-dep
	if [ "$(1)" != "$(2)" ]; then \
		mkdir -p $$(dirname $(2)) || true; \
		mv $(1) $(2); \
	fi

	chmod +x "$(2)"
endef
