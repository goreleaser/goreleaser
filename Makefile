SOURCE_FILES?=./...
TEST_PATTERN?=.
TEST_OPTIONS?=
OS=$(shell uname -s)

export PATH := ./bin:$(PATH)

# Install all the build and lint dependencies
setup:
	go get -u golang.org/x/tools/cmd/stringer
	go get -u golang.org/x/tools/cmd/cover
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh
	curl -sfL https://install.goreleaser.com/github.com/gohugoio/hugo.sh | sh
	curl -sfL https://install.goreleaser.com/github.com/caarlos0/bandep.sh | sh
ifeq ($(OS), Darwin)
	brew install dep
else
	curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
endif
	dep ensure -vendor-only
	echo "make check" > .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit
.PHONY: setup

check:
	bandep --ban github.com/tj/assert
.PHONY: check

# Run all the tests
test:
	go test $(TEST_OPTIONS) -failfast -race -coverpkg=./... -covermode=atomic -coverprofile=coverage.txt $(SOURCE_FILES) -run $(TEST_PATTERN) -timeout=2m
.PHONY: test

# Run all the tests and opens the coverage report
cover: test
	go tool cover -html=coverage.txt
.PHONY: cover

# gofmt and goimports all go files
fmt:
	find . -name '*.go' -not -wholename './vendor/*' | while read -r file; do gofmt -w -s "$$file"; goimports -w "$$file"; done
	# find . -name '*.md' -not -wholename './vendor/*' | xargs prettier --write
.PHONY: fmt

# Run all the linters
lint:
	# TODO: fix tests and lll issues
	./bin/golangci-lint run --tests=false --enable-all --disable=lll ./...
	# find . -name '*.md' -not -wholename './vendor/*' | xargs prettier -l
.PHONY: lint

# Run all the tests and code checks
ci: build test lint
.PHONY: ci

# Build a beta version of goreleaser
build:
	go generate ./...
	go build
.PHONY: build

# Generate the static documentation
static:
	@hugo --enableGitInfo --source www
.PHONY: static

favicon:
	wget -O www/static/avatar.png https://avatars2.githubusercontent.com/u/24697112
	convert www/static/avatar.png -define icon:auto-resize=64,48,32,16 www/static/favicon.ico
	convert www/static/avatar.png -resize x120 www/static/apple-touch-icon.png
.PHONY: favicon

serve:
	@hugo server --enableGitInfo --watch --source www --disableFastRender
.PHONY: serve

# Show to-do items per file.
todo:
	@grep \
		--exclude-dir=vendor \
		--exclude-dir=node_modules \
		--exclude=Makefile \
		--text \
		--color \
		-nRo -E ' TODO:.*|SkipNow' .
.PHONY: todo


.DEFAULT_GOAL := build
