# MAKEFILE
#
# @author      Nicola Asuni
# @link        https://github.com/nexmoinc/gosrvlib
# ------------------------------------------------------------------------------

# Use bash as shell (Note: Ubuntu now uses dash which doesn't support PIPESTATUS).
SHELL=/bin/bash

# CVS path (path to the parent dir containing the project)
CVSPATH=github.com/nexmoinc

# Project owner
OWNER=vonage

# Project vendor
VENDOR=vonage

# Project name
PROJECT=gosrvlib

# Project version
VERSION=$(shell cat VERSION)

# Project release number (packaging build number)
RELEASE=$(shell cat RELEASE)

# Current directory
CURRENTDIR=$(dir $(realpath $(firstword $(MAKEFILE_LIST))))

# Target directory
TARGETDIR=$(CURRENTDIR)target

# Directory where to store binary utility tools
BINUTIL=$(TARGETDIR)/binutil

# GO lang path
ifeq ($(GOPATH),)
	# extract the GOPATH
	GOPATH=$(firstword $(subst /src/, ,$(CURRENTDIR)))
endif

# Add the GO binary dir in the PATH
export PATH := $(GOPATH)/bin:$(PATH)

# Docker command
ifeq ($(DOCKER),)
	DOCKER=docker
endif

# Common commands
GO=GOPATH=$(GOPATH) GOPRIVATE=$(CVSPATH) go
GOFMT=gofmt
GOTEST=GOPATH=$(GOPATH) gotest
GODOC=GOPATH=$(GOPATH) godoc

# Directory containing the source code
SRCDIR=./pkg

# List of packages
GOPKGS=$(shell $(GO) list $(SRCDIR)/...)

# Enable junit report when not in LOCAL mode
ifeq ($(strip $(DEVMODE)),LOCAL)
	TESTEXTRACMD=&& $(GO) tool cover -func=$(TARGETDIR)/report/coverage.out
else
	TESTEXTRACMD=2>&1 | tee >(PATH=$(GOPATH)/bin:$(PATH) go-junit-report > $(TARGETDIR)/test/report.xml); test $${PIPESTATUS[0]} -eq 0
endif

# --- MAKE TARGETS ---

# Display general help about this command
.PHONY: help
help:
	@echo ""
	@echo "$(PROJECT) Makefile."
	@echo "GOPATH=$(GOPATH)"
	@echo "The following commands are available:"
	@echo ""
	@echo "    make cleandeps : Remove all dependencies, including go.sum and go.mod entries"
	@echo "    make clean     : Remove any build artifact"
	@echo "    make coverage  : Generate the coverage report"
	@echo "    make dbuild    : Build everything inside a Docker container"
	@echo "    make deps      : Get dependencies"
	@echo "    make example   : Build and test the service example"
	@echo "    make format    : Format the source code"
	@echo "    make generate  : Generate go code automatically"
	@echo "    make linter    : Check code against multiple linters"
	@echo "    make mod       : Download dependencies"
	@echo "    make qa        : Run all tests and static analysis tools"
	@echo "    make tag       : Tag the Git repository"
	@echo "    make test      : Run unit tests"
	@echo ""
	@echo "Use DEVMODE=LOCAL for human friendly output."
	@echo "To test and build everything from scratch:"
	@echo "DEVMODE=LOCAL make clean cleandeps deps generate mod qa example"
	@echo ""

# Alias for help target
all: help

# Remove any build artifact
.PHONY: cleandeps
cleandeps:
	rm -rf vendor
	rm -f go.sum
	sed -i '/require (/,/)/crequire ()' go.mod

# Remove any build artifact
.PHONY: clean
clean:
	rm -rf $(TARGETDIR)

# Generate the coverage report
.PHONY: coverage
coverage: ensuretarget
	$(GO) tool cover -html=$(TARGETDIR)/report/coverage.out -o $(TARGETDIR)/report/coverage.html

# Build everything inside a Docker container
.PHONY: dbuild
dbuild:
	@mkdir -p $(TARGETDIR)
	@rm -rf $(TARGETDIR)/*
	@echo 0 > $(TARGETDIR)/make.exit
	CVSPATH=$(CVSPATH) VENDOR=$(VENDOR) PROJECT=$(PROJECT) MAKETARGET='$(MAKETARGET)' $(CURRENTDIR)/dockerbuild.sh
	@exit `cat $(TARGETDIR)/make.exit`

# Get the test dependencies
.PHONY: deps
deps: ensuretarget
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(BINUTIL) v1.33.0
	(GO111MODULE=off $(GO) get -u github.com/jstemmer/go-junit-report)
	(GO111MODULE=off $(GO) get -u github.com/rakyll/gotest)
	(GO111MODULE=off $(GO) get -u github.com/golang/mock/mockgen)

# Create the trget directories if missing
.PHONY: ensuretarget
ensuretarget:
	@mkdir -p $(TARGETDIR)/test
	@mkdir -p $(TARGETDIR)/report
	@mkdir -p $(TARGETDIR)/binutil

# Build and test the example
.PHONY: example
example:
	cd examples/service && \
	make clean cleandeps deps generate mod qa build

# Format the source code
.PHONY: format
format:
	@find $(SRCDIR) -type f -name "*.go" -exec $(GOFMT) -s -w {} \;
	cd examples/service && make format

# Generate test mocks
.PHONY: generate
generate:
	rm -f internal/mocks/*.go
	$(GO) generate $(GOPKGS)

# Execute multiple linter tools
.PHONY: linter
linter:
	@echo -e "\n\n>>> START: Static code analysis <<<\n\n"
	$(BINUTIL)/golangci-lint run --exclude-use-default=false $(SRCDIR)/...
	@echo -e "\n\n>>> END: Static code analysis <<<\n\n"

# Download and vendor dependencies
.PHONY: mod
mod:
	$(GO) mod download
	#$(GO) mod vendor
	#rm -f vendor/github.com/coreos/etcd/client/keys.generated.go || true

# Run all tests and static analysis tools
.PHONY: qa
qa: linter test coverage

# Tag the Git repository
.PHONY: tag
tag:
	git tag -a "v$(VERSION)" -m "Version $(VERSION)" && \
	git push origin --tags

# Run the unit tests
.PHONY: test
test: ensuretarget
	@echo -e "\n\n>>> START: Unit Tests <<<\n\n"
	$(GOTEST) \
	-count=1 \
	-tags=unit \
	-covermode=atomic \
	-bench=. \
	-race \
	-failfast \
	-coverprofile=$(TARGETDIR)/report/coverage.out \
	-v $(GOPKGS) $(TESTEXTRACMD)
	@echo -e "\n\n>>> END: Unit Tests <<<\n\n"
