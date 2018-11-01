# By default, build only.
all: test

include main.properties

# Configuration.
GOBINARY ?= go
GOOS ?=
GOARCH ?= amd64
TAG ?= dev

ARCHIVE=$(COMPONENT_NAME)-$(TAG)-$(value GOOS)-$(value GOARCH).zip
BINARY=$(COMPONENT_NAME)-$(GOOS)-$(GOARCH)

# All files excluding vendor
FILES=$$($(GOBINARY) list ./... | grep -v /vendor/ | grep -v /gen/)
TEST?="./..."
# List of sources
SOURCES=$$(find . -name '*.go' | grep -v /vendor/ | grep -v /gen/)

# Run code style checks
fmt:
	gofmt -w $(SOURCES)

fmtcheck: fmt
	@sh -c "'$(CURDIR)/scripts/gofmtcheck.sh'"

# Do build
build: fmtcheck vet
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GOBINARY) build -o $(BINARY)
.PHONY: build

upload-binary: build
	zip "$(ARCHIVE)" "$(BINARY)"
	curl -v --progress-bar --user "$(REPO_USERNAME):$(REPO_PASSWORD)" --upload-file "$(ARCHIVE)" "$(GOBINARIES_REPO_URL)/$(COMPONENT_NAME)/$(COMPONENT_VERSION)/$(ARCHIVE)"
.PHONY: upload-binary

# Do cleanup.
clean:
	rm -f $(BINARY)
.PHONY: clean

# Prepare release.
prepare-release: clean build upload-binary
.PHONY: prepare-release

install: build
	$(GOBINARY)  install
	mkdir -p $(HOME)/.terraform.d/plugins
	mv $(BINARY) $(HOME)/.terraform.d/plugins/terraform-provider-vcd_v1.0.0_x4
.PHONY: install

test: build
	$(GOBINARY)  test -i $(FILES) || exit 1
	echo $(FILES) | \
		xargs -t -n4 go test $(TESTARGS) -timeout=30s -parallel=4
.PHONY: test

vet:
	@echo "go vet ."
	@$(GOBINARY) vet $$(go list ./... | grep -v vendor/) ; if [ $$? -eq 1 ]; then \
		echo ""; \
		echo "Vet found suspicious constructs. Please check the reported constructs"; \
		echo "and fix them if necessary before submitting the code for review."; \
		exit 1; \
	fi
.PHONY: vet
