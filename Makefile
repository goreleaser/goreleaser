SOURCE_FILES?=$$(glide novendor)
TEST_PATTERN?=.
TEST_OPTIONS?=

setup: ## Install all the build and lint dependencies
	go get -u github.com/Masterminds/glide
	go get -u github.com/alecthomas/gometalinter
	go get -u github.com/kisielk/errcheck
	go get -u github.com/golang/lint/golint
	glide install

test: ## Run all the tests
	go test $(TEST_OPTIONS) -cover $(SOURCE_FILES) -run $(TEST_PATTERN) -timeout=30s

lint: ## Run all the linters
	gometalinter --vendor --disable-all \
	  --enable=errcheck \
	  --enable=vet \
	  --enable=golint \
	  $(glide novendor)

ci: lint test ## Run all the tests and code checks

# Absolutely awesome: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
