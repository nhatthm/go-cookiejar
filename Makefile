MODULE_NAME = cookiejar

VENDOR_DIR = vendor
GITHUB_OUTPUT ?= /dev/null

GOLANGCI_LINT_VERSION ?= v2.0.2
MOCKERY_VERSION ?= v2.53.2

GO ?= go
GOLANGCI_LINT ?= $(shell go env GOPATH)/bin/golangci-lint-$(GOLANGCI_LINT_VERSION)
MOCKERY ?= $(shell go env GOPATH)/bin/mockery-$(MOCKERY_VERSION)

.PHONY: $(VENDOR_DIR)
$(VENDOR_DIR):
	@mkdir -p $(VENDOR_DIR)
	@$(GO) mod vendor
	@$(GO) mod tidy

.PHONY: generate
generate: $(MOCKERY)
	@echo ">> generate mocks"
	@$(MOCKERY)

.PHONY: lint
lint:
	@$(GOLANGCI_LINT) run

.PHONY: test
test: test-unit

## Run unit tests
.PHONY: test-unit
test-unit:
	@echo ">> unit test"
	@$(GO) test -gcflags=-l -coverprofile=unit.coverprofile -covermode=atomic -race ./...

.PHONY: sync
sync:
	@echo ">> sync"
	@ls -1 *.go | grep -v persistent_jar | xargs rm -f
	@rm -f internal/ascii/*
	@cp $(shell $(GO) env GOROOT)/src/net/http/cookiejar/*.go ./
	@cp $(shell $(GO) env GOROOT)/src/net/http/internal/ascii/*.go ./internal/ascii/
	@sed -i '' -E 's#net/http/internal/ascii#go.nhat.io/$(MODULE_NAME)/internal/ascii#g' *.go
	@gofumpt -l -w .

.PHONY: $(GITHUB_OUTPUT)
$(GITHUB_OUTPUT):
	@echo "MODULE_NAME=$(MODULE_NAME)" >> "$@"
	@echo "GOLANGCI_LINT_VERSION=$(GOLANGCI_LINT_VERSION)" >> "$@"

$(GOLANGCI_LINT):
	@echo "$(OK_COLOR)==> Installing golangci-lint $(GOLANGCI_LINT_VERSION)$(NO_COLOR)"; \
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ./bin "$(GOLANGCI_LINT_VERSION)"
	@mv ./bin/golangci-lint $(GOLANGCI_LINT)

$(MOCKERY):
	@echo "$(OK_COLOR)==> Installing mockery $(MOCKERY_VERSION)$(NO_COLOR)"; \
	GOBIN=/tmp $(GO) install github.com/vektra/mockery/$(shell echo "$(MOCKERY_VERSION)" | cut -d '.' -f 1)@$(MOCKERY_VERSION)
	@mv /tmp/mockery $(MOCKERY)
