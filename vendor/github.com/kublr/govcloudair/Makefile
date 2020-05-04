all: test

# Configuration
GOOS ?=
GOBINARY ?= go

SOURCE=$$($(GOBINARY) list ./... | grep -v /vendor/ | grep -v /gen/)
SOURCE_FILES?=$$(find . -name '*.go' | grep -v /vendor/ | grep -v /gen/)

# Install build dependencies
deps:
	$(GOBINARY) get -u github.com/golang/dep/cmd/dep
.PHONY: deps

# Run code style checks
codestyle:
	@echo "==> Running codestyle checks"
	GOOS=$(GOOS) gofmt -l -e -d -s $(SOURCE_FILES)
	GOOS=$(GOOS) test -z "$(shell gofmt -l $(SOURCE_FILES))"

	GOOS=$(GOOS) golint $(SOURCE)

	GOOS=$(GOOS) $(GOBINARY) vet $(SOURCE)
.PHONY: codestyle

build: codestyle
	@echo "==> Building"
	GOOS=$(GOOS) $(GOBINARY) build -v $(SOURCE)
.PHONY: build

test: build
	@echo "==> Running tests"
	GOOS=$(GOOS) $(GOBINARY) test -v $(SOURCE)
.PHONY: test
