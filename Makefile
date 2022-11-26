SHELL := bash
.ONESHELL:
.SHELLFLAGS := -eu -o pipefail -c
.DELETE_ON_ERROR:
.DEFAULT_GOAL := help
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules

GO111MODULES=on
GIT_TAG := $(shell git describe --always --abbrev=0 --tags)
GIT_BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
GIT_COMMIT := $(shell git log --pretty=format:'%h' -n 1)
VERSION="$(GIT_TAG)-$(GIT_BRANCH).$(GIT_COMMIT)"

BINARY := "go-discord-irc"
BINARY_PKG_BUILD := "."
RELEASE_ZIP := "go-discord-irc.zip"
ARGS := --config config.yml

.PHONY: all build release
all: build release
build: $(BINARY) ## Build Binary
release: $(RELEASE_ZIP) ## Package release artifact

$(BINARY): dep
	@echo "🍳 Building $(BINARY)"
	go build -v -o $(BINARY) -ldflags "-X main.version=$(GIT_TAG)-$(GIT_BRANCH).$(GIT_COMMIT)" $(BINARY_PKG_BUILD)

$(RELEASE_ZIP): $(BINARY)
	@echo "🍳 Building $(RELEASE_ZIP)"
	zip --junk-paths $(RELEASE_ZIP) $(BINARY) README.md

.PHONY:clean
clean: ## Remove previous builds
	@echo "🧹 Cleaning old build"
	go clean
	rm -f $(BINARY) $(RELEASE_ZIP)

.PHONY: dep
dep: ## go get all dependencies
	@echo "🛎 Updating Dependencies"
	go get -v -d ./...

.PHONY: run
run: dep ## Compiles and runs Binary
	@go run -race $(BINARY_PKG_BUILD) --debug $(ARGS)

.PHONY: help
help:  ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
	| sort \
	| awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: test
test: ## Runs go test with default values
	@echo "🍜 Testing $(BINARY)"
	go test -v -count=1 -race ./...
