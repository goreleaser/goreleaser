SOURCE_FILES?=./...
TEST_PATTERN?=.
TEST_OPTIONS?=

# Install all the build and lint dependencies
setup:
	go get -u golang.org/x/tools/cmd/stringer
	go get -u github.com/alecthomas/gometalinter
	go get -u github.com/golang/dep/cmd/dep
	go get -u github.com/pierrre/gotestcover
	go get -u golang.org/x/tools/cmd/cover
	go get -u github.com/apex/static/cmd/static-docs
	go get -u github.com/caarlos0/bandep
	dep ensure
	gometalinter --install
	echo "make check" > .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit
.PHONY: setup

check:
	bandep --ban github.com/tj/assert
.PHONY: check

# Run all the tests
test:
	gotestcover $(TEST_OPTIONS) -covermode=atomic -coverprofile=coverage.txt $(SOURCE_FILES) -run $(TEST_PATTERN) -timeout=2m
.PHONY: cover

# Run all the tests and opens the coverage report
cover: test
	go tool cover -html=coverage.txt
.PHONY: cover

# gofmt and goimports all go files
fmt:
	find . -name '*.go' -not -wholename './vendor/*' | while read -r file; do gofmt -w -s "$$file"; goimports -w "$$file"; done
	find . -name '*.md' -not -wholename './vendor/*' | xargs prettier --write
.PHONY: fmt

# Run all the linters
lint:
	gometalinter --vendor ./...
	find . -name '*.md' -not -wholename './vendor/*' | xargs prettier -l
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
	@rm -rf dist/goreleaser.github.io
	@mkdir -p dist
	@git clone https://github.com/goreleaser/goreleaser.github.io.git dist/goreleaser.github.io
	@rm -rf dist/goreleaser.github.io/theme
	@static-docs \
		--in docs \
		--out dist/goreleaser.github.io \
		--title GoReleaser \
		--subtitle "Deliver Go binaries as fast and easily as possible" \
		--google UA-106198408-1
.PHONY: static

# Show to-do items per file.
todo:
	@grep \
		--exclude-dir=vendor \
		--exclude-dir=node_modules \
		--exclude=Makefile \
		--text \
		--color \
		-nRo -E ' TODO:.*|SkipNow|nolint:.*' .
.PHONY: todo


.DEFAULT_GOAL := build
