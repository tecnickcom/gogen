# MAKEFILE
#
# @author      Nicola Asuni
# @link        https://github.com/gosrvlibexample/gosrvlibexample
# ------------------------------------------------------------------------------

# Use bash as shell (Note: Ubuntu now uses dash which doesn't support PIPESTATUS).
SHELL=/bin/bash

# CVS path (path to the parent dir containing the project)
CVSPATH=github.com/gosrvlibexample

# Project owner
OWNER=gosrvlibexample

# Project vendor
VENDOR=gosrvlibexample

# Project name
PROJECT=gosrvlibexample

# Project version
VERSION=$(shell cat VERSION)

# Project release number (packaging build number)
RELEASE=$(shell cat RELEASE)

# Name of RPM or DEB package
PKGNAME=${VENDOR}-${PROJECT}

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

# Path for binary files (where the executable files will be installed)
BINPATH=usr/bin/

# Path for configuration files
CONFIGPATH=etc/$(PROJECT)/

# Path for custom root CA certificates
SSLCONFIGPATH=etc/ssl/

# Search paths for system root CA certificates to include
CACERTPATH=/etc/ssl/certs/ca-certificates.crt /etc/pki/tls/certs/ca-bundle.crt /etc/ssl/ca-bundle.pem /etc/pki/tls/cacert.pem /etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem /etc/ssl/cert.pem

# Path for init script
INITPATH=

# Path path for documentation
DOCPATH=usr/share/doc/$(PKGNAME)/

# Path path for man pages
MANPATH=usr/share/man/man1/

# Installation path for the binary files
PATHINSTBIN=$(DESTDIR)/$(BINPATH)

# Installation path for the configuration files
PATHINSTCFG=$(DESTDIR)/$(CONFIGPATH)

# Installation path for the ssl root certs
PATHINSTSSLCFG=$(DESTDIR)/$(SSLCONFIGPATH)

# Installation path for the init file
PATHINSTINIT=$(DESTDIR)/$(INITPATH)

# Installation path for documentation
PATHINSTDOC=$(DESTDIR)/$(DOCPATH)

# Installation path for man pages
PATHINSTMAN=$(DESTDIR)/$(MANPATH)

# DOCKER Packaging path (where BZ2s will be stored)
PATHDOCKERPKG=$(CURRENTDIR)/target/DOCKER

# STATIC is a flag to indicate whether to build using static or dynamic linking
STATIC=1
ifeq ($(STATIC),0)
	STATIC_TAG=dynamic
	STATIC_FLAG=
else
	STATIC_TAG=static
	STATIC_FLAG=-static
endif

# Docker tag
DOCKERTAG="$(VERSION)-$(RELEASE)"
ifeq ($(RELEASE),0)
	DOCKERTAG="latest"
endif

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
CMDDIR=./cmd
SRCDIR=./internal

# List of packages
GOPKGS=$(shell $(GO) list $(CMDDIR)/... $(SRCDIR)/...)

# Enable junit report when not in LOCAL mode
ifeq ($(strip $(DEVMODE)),LOCAL)
	TESTEXTRACMD=&& $(GO) tool cover -func=$(TARGETDIR)/report/coverage.out
else
	TESTEXTRACMD=2>&1 | tee >(PATH=$(GOPATH)/bin:$(PATH) go-junit-report > $(TARGETDIR)/test/report.xml); test $${PIPESTATUS[0]} -eq 0
endif

# Specify api test configuration files to execute (venom YAML files or * for all)
ifeq ($(API_TEST_FILE),)
	API_TEST_FILE=*.yaml
endif

# Deployment environment
ifeq ($(DEPLOY_ENV),)
	DEPLOY_ENV=int
endif

# INT - integration environment via docker-compose
ifeq ($(DEPLOY_ENV), int)
	GOSRVLIBEXAMPLE_URL=http://gosrvlibexample:8081
	GOSRVLIBEXAMPLE_MONITORING_URL=http://gosrvlibexample:8082
	API_TEST_FILE=*.yaml
endif

# Display general help about this command
.PHONY: help
help:
	@echo ""
	@echo "$(PROJECT) Makefile."
	@echo "GOPATH=$(GOPATH)"
	@echo "The following commands are available:"
	@echo ""
	@echo "    make qa          : Run all tests and static analysis tools"
	@echo "    make test        : Run unit tests"
	@echo "    make coverage    : Generate the coverage report"
	@echo ""                    
	@echo "    make format      : Format the source code"
	@echo "    make generate    : Generate go code automatically"
	@echo "    make linter      : Check code against multiple linters"
	@echo ""                    
	@echo "    make deps        : Get dependencies"
	@echo "    make mod         : Download and vendor dependencies"
	@echo "    make clean       : Remove any build artifact"
	@echo ""
	@echo "    make build       : Compile the application"
	@echo "    make install     : Install this application"
	@echo "    make buildall    : Full build and test sequence"
	@echo "    make dbuild      : Build everything inside a Docker container"
	@echo ""
	@echo "    make docker      : Build a scratch docker container to run this service"
	@echo "    make dockertest  : Test the newly built docker container"
	@echo ""
	@echo "    make openapitest : Test the OpenAPI specification"
	@echo "    make apitest     : Execute API tests"
	@echo ""

# Alias for help target
all: help

# Validate JSON configuration files against the JSON schema
.PHONY: confcheck
confcheck:
	jsonschema -i resources/test/etc/${PROJECT}/config.json resources/etc/${PROJECT}/config.schema.json
	jsonschema -i resources/etc/${PROJECT}/config.json resources/etc/${PROJECT}/config.schema.json

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
	@find $(CMDDIR) -type f -name "*.go" -exec $(GOFMT) -s -w {} \;
	@find $(SRCDIR) -type f -name "*.go" -exec $(GOFMT) -s -w {} \;

# Generate the coverage report
.PHONY: coverage
coverage: ensuretarget
	$(GO) tool cover -html=$(TARGETDIR)/report/coverage.out -o $(TARGETDIR)/report/coverage.html

# Execute multiple linter tools
.PHONY: linter
linter:
	@echo -e "\n\n>>> START: Static code analysis <<<\n\n"
	$(BINUTIL)/golangci-lint run --exclude-use-default=false $(CMDDIR)/... $(SRCDIR)/...
	@echo -e "\n\n>>> END: Static code analysis <<<\n\n"

# Run all tests and static analysis tools
.PHONY: qa
qa: linter confcheck test coverage 

.PHONY: mod
mod:
	$(GO) mod download
	$(GO) mod vendor
	rm -f vendor/github.com/coreos/etcd/client/keys.generated.go || true

# Get the test dependencies
.PHONY: deps
deps: ensuretarget
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(BINUTIL) v1.32.0
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

# Install this application
.PHONY: install
install: uninstall
	mkdir -p $(PATHINSTBIN)
	cp -r ./target/${BINPATH}* $(PATHINSTBIN)
	find $(PATHINSTBIN) -type d -exec chmod 755 {} \;
	find $(PATHINSTBIN) -type f -exec chmod 755 {} \;
	mkdir -p $(PATHINSTDOC)
	cp -f ./LICENSE $(PATHINSTDOC)
	cp -f ./README.md $(PATHINSTDOC)
	cp -f ./CONFIG.md $(PATHINSTDOC)
	cp -f ./VERSION $(PATHINSTDOC)
	cp -f ./RELEASE $(PATHINSTDOC)
	chmod -R 644 $(PATHINSTDOC)*
ifneq ($(strip $(INITPATH)),)
	mkdir -p $(PATHINSTINIT)
	cp -ru ./resources/${INITPATH}* $(PATHINSTINIT)
	find $(PATHINSTINIT) -type d -exec chmod 755 {} \;
	find $(PATHINSTINIT) -type f -exec chmod 755 {} \;
endif
ifneq ($(strip $(CONFIGPATH)),)
	mkdir -p $(PATHINSTCFG)
	touch -c $(PATHINSTCFG)*
	cp -ru ./resources/${CONFIGPATH}* $(PATHINSTCFG)
	find $(PATHINSTCFG) -type d -exec chmod 755 {} \;
	find $(PATHINSTCFG) -type f -exec chmod 644 {} \;
endif
ifneq ($(strip $(MANPATH)),)
	mkdir -p $(PATHINSTMAN)
	cat ./resources/${MANPATH}${PROJECT}.1 | gzip -9 > $(PATHINSTMAN)${PROJECT}.1.gz
	find $(PATHINSTMAN) -type f -exec chmod 644 {} \;
endif

# Install TLS root CA certificates
.PHONY: installssl
installssl:
ifneq ($(strip $(SSLCONFIGPATH)),)
	# add system root CA certificates
	for CERT in ${CACERTPATH} ; do \
		test -f $${CERT} && \
		mkdir -p $${DESTDIR}$$(dirname $${CERT}) && \
		cp $${CERT} $${DESTDIR}$${CERT} && \
		break ; \
	done
	# add custom CA certificates
	mkdir -p $(PATHINSTSSLCFG)
	cp -r ./resources/${SSLCONFIGPATH}* $(PATHINSTSSLCFG)
	rm $(PATHINSTSSLCFG)certs/.keep
	find $(PATHINSTSSLCFG) -type d -exec chmod 755 {} \;
	find $(PATHINSTSSLCFG) -type f -exec chmod 644 {} \;
endif

# Remove all installed files (excluding configuration files)
.PHONY: uninstall
uninstall:
	rm -rf $(PATHINSTBIN)$(PROJECT)
	rm -rf $(PATHINSTDOC)

# Compile the application
.PHONY: build
build:
	CGO_ENABLED=0 \
	$(GO) build \
	-tags ${STATIC_TAG} \
	-ldflags '-w -s -X main.ProgramVersion=${VERSION} -X main.ProgramRelease=${RELEASE} -extldflags "-fno-PIC ${STATIC_FLAG}"' \
	-mod=vendor \
	-o ./target/${BINPATH}$(PROJECT) $(CMDDIR)

# Create the directory with docker files to be packaged
.PHONY: dockerdir
dockerdir:
	rm -rf $(PATHDOCKERPKG)
	$(MAKE) install DESTDIR=$(PATHDOCKERPKG)
	$(MAKE) installssl DESTDIR=$(PATHDOCKERPKG)
	cp resources/docker/Dockerfile.run $(PATHDOCKERPKG)/Dockerfile

# Build the docker container in the target/DOCKER directory
.PHONY: dockerbuild
dockerbuild:
	$(DOCKER) build --no-cache --tag=${VENDOR}/${PROJECT}$(DOCKERSUFFIX):latest $(PATHDOCKERPKG)

# Delete the Docker image
.PHONY: dockerdelete
dockerdelete:
	$(DOCKER) rmi -f `docker images "${VENDOR}/${PROJECT}$(DOCKERSUFFIX):latest" -q`

# Build a docker container to run this service
.PHONY: docker
docker: dockerdir dockerbuild

# Full build and test sequence
# You may want to change this and remove the options you don't need
#buildall: deps qa rpm deb bz2 crossbuild
.PHONY: buildall
buildall: build qa docker

# Run the integration tests; locally we need to execute 'build' and 'docker' targets first
.PHONY: dockertestenv
dockertestenv: ensuretarget
	@echo "GOSRVLIBEXAMPLE_REMOTECONFIGPROVIDER=envvar" > $(TARGETDIR)/gosrvlibexample.integration.env
	@echo "GOSRVLIBEXAMPLE_REMOTECONFIGDATA=$(shell cat resources/test/integration/gosrvlibexample/config.json | base64 -w0)" >> $(TARGETDIR)/gosrvlibexample.integration.env

.PHONY: dockertest
dockertest: dockertestenv
ifeq ("$(wildcard $(BINUTIL)/dockerize)","")
	curl -sSfL https://github.com/jwilder/dockerize/releases/download/v0.6.1/dockerize-linux-amd64-v0.6.1.tar.gz | tar -xz -C $(BINUTIL)
endif
	@echo 0 > $(TARGETDIR)/make.exit
	docker-compose down --volumes || true
	docker-compose up --build --exit-code-from gosrvlibexample_integration || echo $${?} > $(TARGETDIR)/make.exit
	docker-compose down --rmi local --volumes --remove-orphans || true
	@exit `cat $(TARGETDIR)/make.exit`

# Test the OpenAPI specification against the real deployed service
.PHONY: schemathesistest
schemathesistest:
	schemathesis run \
	--validate-schema=true \
	--checks=all \
	--request-timeout=2000 \
	--hypothesis-max-examples=100 \
	--hypothesis-deadline=2000 \
	--show-errors-tracebacks \
	--base-url='${API_TEST_URL}' \
	${OPENAPI_FILE}

# Test the OpenAPI specification against the real deployed service
.PHONY: openapitest
openapitest:
	$(MAKE) schemathesistest API_TEST_URL=${GOSRVLIBEXAMPLE_MONITORING_URL} OPENAPI_FILE=openapi_monitoring.yaml
	$(MAKE) schemathesistest API_TEST_URL=${GOSRVLIBEXAMPLE_URL} OPENAPI_FILE=openapi.yaml

# Run venom tests (https://github.com/ovh/venom)
.PHONY: venomtest
venomtest:
	@mkdir -p $(TARGETDIR)/report/${DEPLOY_ENV}/venom/$(API_TEST_DIR)
	venom run \
		--var gosrvlibexample.url="${API_TEST_URL}" \
		--var gosrvlibexample.version="${VERSION}" \
		--var gosrvlibexample.release="${RELEASE}" \
		--strict \
		--log=info \
		--output-dir=$(TARGETDIR)/report/${DEPLOY_ENV}/venom/$(API_TEST_DIR) \
		resources/test/venom/$(API_TEST_DIR)/$(API_TEST_FILE)

# Run venom tests (https://github.com/ovh/venom)
.PHONY: apitest
apitest:
	$(MAKE) venomtest API_TEST_DIR=monitoring API_TEST_URL=${GOSRVLIBEXAMPLE_MONITORING_URL} API_TEST_FILE=api.yaml
	$(MAKE) venomtest API_TEST_DIR=public API_TEST_URL=${GOSRVLIBEXAMPLE_URL} API_TEST_FILE=${API_TEST_FILE}
