SOURCE_FILES?=./...
TEST_PATTERN?=.
TEST_OPTIONS?=

# Install all the build and lint dependencies
setup:
	go get -u github.com/alecthomas/gometalinter
	go get -u github.com/golang/dep/cmd/dep
	go get -u github.com/pierrre/gotestcover
	go get -u golang.org/x/tools/cmd/cover
	dep ensure
	gometalinter --install

# Run all the tests
test:
	gotestcover $(TEST_OPTIONS) -covermode=atomic -coverprofile=coverage.txt $(SOURCE_FILES) -run $(TEST_PATTERN) -timeout=2m

# Run all the tests and opens the coverage report
cover: test
	go tool cover -html=coverage.txt

# gofmt and goimports all go files
fmt:
	find . -name '*.go' -not -wholename './vendor/*' | while read -r file; do gofmt -w -s "$$file"; goimports -w "$$file"; done

# Run all the linters
lint:
	gometalinter --vendor ./...

# Run all the tests and code checks
ci: test lint

# Build a beta version of goreleaser
build:
	go build

HIGHLIGHT=https://cdnjs.cloudflare.com/ajax/libs/highlight.js/9.12.0

# Generate the static documentation
static:
	@static-docs \
		--in docs \
		--out ../goreleaser.github.io \
		--title GoReleaser \
		--subtitle "Deliver Go binaries as fast and easily as possible" \
		--google UA-106198408-1 \
		--script "$(HIGHLIGHT)/highlight.min.js" \
		--script "$(HIGHLIGHT)/languages/go.min.js" \
		--script "$(HIGHLIGHT)/languages/yaml.min.js" \
		--script "$(HIGHLIGHT)/languages/dockerfile.min.js" \
		--style "$(HIGHLIGHT)/styles/atom-one-dark.min.css" \
		--inline-script 'hljs.initHighlightingOnLoad();' \
		--inline-style 'pre { padding: 0; }'

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
