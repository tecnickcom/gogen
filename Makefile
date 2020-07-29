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
GO := GOPATH=$(GOPATH) GOPRIVATE=$(CVSPATH) go
GOFMT := gofmt
GOTEST := GOPATH=$(GOPATH) gotest
GODOC := GOPATH=$(GOPATH) godoc

# Directory containing the source code
SRCDIR=./pkg

# List of packages
GOPKGS := $(shell $(GO) list $(SRCDIR)/...)

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
	@echo "    make qa       : Run all tests and static analysis tools"
	@echo "    make test     : Run unit tests"
	@echo "    make coverage : Generate the coverage report"
	@echo ""
	@echo "    make format   : Format the source code"
	@echo "    make generate : Generate go code automatically"
	@echo "    make linter   : Check code against multiple linters"
	@echo ""
	@echo "    make deps     : Get dependencies"
	@echo "    make mod      : Download and vendor dependencies"
	@echo "    make clean    : Remove any build artifact"
	@echo ""
	@echo "    make example  : Build and test the service example"
	@echo "    make dbuild   : Build everything inside a Docker container"
	@echo "    make tag      : Tag the Git repository"
	@echo ""

# Alias for help target
all: help

# Create the trget directories if missing
.PHONY: ensuretarget
ensuretarget:
	@mkdir -p $(TARGETDIR)/test
	@mkdir -p $(TARGETDIR)/report
	@mkdir -p $(TARGETDIR)/binutil

# Generate test mocks
.PHONY: generate
generate:
	$(GO) generate $(GOPKGS)

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

# Format the source code
.PHONY: format
format:
	@find $(SRCDIR) -type f -name "*.go" -exec $(GOFMT) -s -w {} \;

# Generate the coverage report
.PHONY: coverage
coverage: ensuretarget
	$(GO) tool cover -html=$(TARGETDIR)/report/coverage.out -o $(TARGETDIR)/report/coverage.html

# Execute multiple linter tools
.PHONY: linter
linter:
	@echo -e "\n\n>>> START: Static code analysis <<<\n\n"
	$(BINUTIL)/golangci-lint run --exclude-use-default=false $(SRCDIR)/...
	@echo -e "\n\n>>> END: Static code analysis <<<\n\n"

# Run all tests and static analysis tools
.PHONY: qa
qa: linter test coverage

.PHONY: mod
mod:
	$(GO) mod download
	#$(GO) mod vendor
	#rm -f vendor/github.com/coreos/etcd/client/keys.generated.go || true

# Get the test dependencies
.PHONY: deps
deps: ensuretarget
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(BINUTIL) v1.27.0
	(GO111MODULE=off $(GO) get github.com/jstemmer/go-junit-report)
	(GO111MODULE=off $(GO) get github.com/rakyll/gotest)
	(GO111MODULE=off $(GO) get github.com/golang/mock/mockgen)

# Remove any build artifact
.PHONY: clean
clean:
	rm -rf $(TARGETDIR)
	$(GO) clean -i ./...

# Build everything inside a Docker container
.PHONY: dbuild
dbuild:
	@mkdir -p $(TARGETDIR)
	@rm -rf $(TARGETDIR)/*
	@echo 0 > $(TARGETDIR)/make.exit
	CVSPATH=$(CVSPATH) VENDOR=$(VENDOR) PROJECT=$(PROJECT) MAKETARGET='$(MAKETARGET)' $(CURRENTDIR)/dockerbuild.sh
	@exit `cat $(TARGETDIR)/make.exit`

# Tag the Git repository
.PHONY: tag
tag:
	echo git tag -a v$(VERSION) -m "Version $(VERSION)" && \
	git push origin --tags

# Build and test the example
.PHONY: example
example:
	cd examples/service && make clean deps mod qa build
