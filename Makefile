SOURCE_FILES?=./...
TEST_PATTERN?=.
TEST_OPTIONS?=
DOCKER?=docker

export PATH := ./bin:$(PATH)
export GO111MODULE := on
export GOPROXY = https://proxy.golang.org,direct

# Setup pre-commit hooks
dev:
	cp -f scripts/pre-commit.sh .git/hooks/pre-commit
.PHONY: dev

# Install dependencies
setup:
	go mod tidy
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
	gofumpt -w .
.PHONY: fmt

# Run all the tests and code checks
ci: build test
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
	$(DOCKER) run --rm -it -p 8000:8000 -v ${PWD}/www:/docs docker.io/squidfunk/mkdocs-material
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
