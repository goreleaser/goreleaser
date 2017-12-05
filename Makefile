SOURCE_FILES?=./...
TEST_PATTERN?=.
TEST_OPTIONS?=

# Install all the build and lint dependencies
setup:
	go get -u github.com/alecthomas/gometalinter
	go get -u github.com/golang/dep/cmd/dep
	go get -u github.com/pierrre/gotestcover
	go get -u golang.org/x/tools/cmd/cover
	go get -u github.com/apex/static/cmd/static-docs
	dep ensure
	gometalinter --install
.PHONY: setup

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
.PHONY: fmt

# Run all the linters
lint:
	gometalinter --vendor ./...
.PHONY: lint

# Run all the tests and code checks
ci: test lint
.PHONY: ci

# Build a beta version of goreleaser
build:
	go build
.PHONY: build

# Generate the static documentation
static:
	@rm -rf dist/goreleaser.github.io
	@mkdir -p dist
	@git clone git@github.com:goreleaser/goreleaser.github.io.git dist/goreleaser.github.io
	@rm -rf dist/goreleaser.github.io/theme
	@static-docs \
		--in docs \
		--out dist/goreleaser.github.io \
		--title GoReleaser \
		--subtitle "Deliver Go binaries as fast and easily as possible" \
		--google UA-106198408-1
.PHONY: static

static-push: static
	@cd dist/goreleaser.github.io && git add -A && git commit -am 'bump: docs' && git diff --exit-code origin/master..master > /dev/null || git push origin master
.PHONY: static-push

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
