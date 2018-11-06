# By default, build only.
all: build

include main.properties

# Configuration.
GOOS ?= linux
GOARCH ?= amd64
GOBINARY ?= go
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
build: clean fmtcheck
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
prepare-release: build upload-binary
.PHONY: prepare-release
####################################
install: build
	$(GOBINARY)  install
	mkdir -p $(HOME)/.terraform.d/plugins
	mv $(BINARY) $(HOME)/.terraform.d/plugins/terraform-provider-vcd_v1.0.0_x4

test: fmtcheck
	$(GOBINARY)  test -i $(FILES) || exit 1
	echo $(FILES) | \
		xargs -t -n4 go test $(TESTARGS) -timeout=30s -parallel=4

testacc: fmtcheck
	TF_ACC=1 $(GOBINARY)  test $(FILES) -v $(TESTARGS) -timeout 120m

vet:
	@echo "go vet ."
	@$(GOBINARY) vet $$(go list ./... | grep -v vendor/) ; if [ $$? -eq 1 ]; then \
		echo ""; \
		echo "Vet found suspicious constructs. Please check the reported constructs"; \
		echo "and fix them if necessary before submitting the code for review."; \
		exit 1; \
	fi

errcheck:
	@sh -c "'$(CURDIR)/scripts/errcheck.sh'"

vendor-status:
	@govendor status

test-compile:
	@if [ "$(TEST)" = "./..." ]; then \
		echo "ERROR: Set TEST to a specific package. For example,"; \
		echo "  make test-compile TEST=./aws"; \
		exit 1; \
	fi
	$(GOBINARY) test -c $(TEST) $(TESTARGS)

.PHONY: test testacc vet fmt fmtcheck errcheck vendor-status test-compile

