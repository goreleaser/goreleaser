# sourced by https://github.com/octomation/makefiles

.DEFAULT_GOAL = test-with-coverage

SHELL = /bin/bash -euo pipefail

GO111MODULE = on
GOFLAGS     = -mod=vendor
GOPRIVATE   = go.octolab.net
GOPROXY     = direct
LOCAL       = $(MODULE)
MODULE      = `go list -m`
PACKAGES    = `go list ./... 2> /dev/null`
PATHS       = $(shell echo $(PACKAGES) | sed -e "s|$(MODULE)/\{0,1\}||g")
TIMEOUT     = 1s

ifeq (, $(PACKAGES))
	PACKAGES = $(MODULE)
endif

ifeq (, $(PATHS))
	PATHS = .
endif

export GO111MODULE := $(GO111MODULE)
export GOFLAGS     := $(GOFLAGS)
export GOPRIVATE   := $(GOPRIVATE)
export GOPROXY     := $(GOPROXY)

.PHONY: go-env
go-env:
	@echo "GO111MODULE: `go env GO111MODULE`"
	@echo "GOFLAGS:     $(strip `go env GOFLAGS`)"
	@echo "GOPRIVATE:   $(strip `go env GOPRIVATE`)"
	@echo "GOPROXY:     $(strip `go env GOPROXY`)"
	@echo "LOCAL:       $(LOCAL)"
	@echo "MODULE:      $(MODULE)"
	@echo "PACKAGES:    $(PACKAGES)"
	@echo "PATHS:       $(strip $(PATHS))"
	@echo "TIMEOUT:     $(TIMEOUT)"

.PHONY: deps
deps:
	@go mod download
	@if [[ "`go env GOFLAGS`" =~ -mod=vendor ]]; then go mod vendor; fi

.PHONY: deps-clean
deps-clean:
	@go clean -modcache

.PHONY: deps-shake
deps-shake:
	@go mod tidy

.PHONY: update
update: selector = '{{if not (or .Main .Indirect)}}{{.Path}}{{end}}'
update:
	@if command -v egg > /dev/null; then \
		packages="`egg deps list`"; \
	else \
		packages="`go list -f $(selector) -m all`"; \
	fi; go get -mod= -u $$packages

.PHONY: update-all
update-all:
	@go get -mod= -u ./...

.PHONY: test
test:
	@go test -race -timeout $(TIMEOUT) $(PACKAGES)

.PHONY: test-clean
test-clean:
	@go clean -testcache

.PHONY: test-with-coverage
test-with-coverage:
	@go test -cover -timeout $(TIMEOUT) $(PACKAGES) | column -t | sort -r

.PHONY: test-with-coverage-profile
test-with-coverage-profile:
	@go test -cover -covermode count -coverprofile c.out -timeout $(TIMEOUT) $(PACKAGES)

.PHONY: format
format:
	@goimports -local $(LOCAL) -ungroup -w $(PATHS)

.PHONY: generate
generate:
	@go generate $(PACKAGES)


.PHONY: clean
clean: deps-clean test-clean

.PHONY: env
env: go-env

.PHONY: refresh
refresh: deps-shake update deps generate format test
