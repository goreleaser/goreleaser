SOURCE_FILES?=./...
TEST_PATTERN?=.
TEST_OPTIONS?=

export PATH := ./bin:$(PATH)
export GO111MODULE := on
export GOPROXY = https://proxy.golang.org,direct

# Install all the build and lint dependencies
setup:
	go mod download
	go generate -v ./...
.PHONY: setup

# Run all the tests
test:
	LC_ALL=C go test $(TEST_OPTIONS) -failfast -race -coverpkg=./... -covermode=atomic -coverprofile=coverage.txt $(SOURCE_FILES) -run $(TEST_PATTERN) -timeout=5m
.PHONY: test

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
	# TODO: fix tests issues
	# TODO: fix lll issues
	# TODO: fix funlen issues
	# TODO: fix godox issues
	# TODO: fix wsl issues
	golangci-lint run ./...
	misspell -error **/*
.PHONY: lint

# Run all the tests and code checks
ci: build test lint
.PHONY: ci

# Build a beta version of goreleaser
build:
	go build
.PHONY: build

imgs:
	wget -O www/docs/static/logo.png https://github.com/goreleaser/artwork/raw/master/goreleaserfundo.png
	wget -O www/docs/static/card.png "https://og.caarlos0.dev/**GoReleaser**%20%7C%20Deliver%20Go%20binaries%20as%20fast%20and%20easily%20as%20possible.png?theme=light&md=1&fontSize=80px&images=https://github.com/goreleaser.png"
	wget -O www/docs/static/avatar.png https://github.com/goreleaser.png
	convert www/docs/static/avatar.png -define icon:auto-resize=64,48,32,16 docs/static/favicon.ico
	convert www/docs/static/avatar.png -resize x120 www/docs/static/apple-touch-icon.png
.PHONY: imgs

serve:
	@docker run --rm -it -p 8000:8000 -v ${PWD}/www:/docs squidfunk/mkdocs-material
.PHONY: serve

vercel:
	yum install -y jq
	pip install mkdocs-material mkdocs-minify-plugin
	./scripts/get-releases.sh
	(cd www && mkdocs build)

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
