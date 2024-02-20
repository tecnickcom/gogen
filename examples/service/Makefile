# MAKEFILE
#
# @author      Nicola Asuni
# @link        https://github.com/gosrvlibexampleowner/gosrvlibexample
# ------------------------------------------------------------------------------

SHELL=/bin/bash
.SHELLFLAGS=-o pipefail -c

# Project owner
OWNER=gosrvlibexampleowner

# Project vendor
VENDOR=${OWNER}

# Lowercase VENDOR name for Docker
LCVENDOR=$(shell echo "${VENDOR}" | tr '[:upper:]' '[:lower:]')

# CVS path (path to the parent dir containing the project)
CVSPATH=github.com/${VENDOR}

# Project name
PROJECT=gosrvlibexample

# Project version
VERSION=$(shell cat VERSION)

# Project release number (packaging build number)
RELEASE=$(shell cat RELEASE)

# Name of RPM or DEB package
PKGNAME=${LCVENDOR}-${PROJECT}

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
INITPATH=etc/init.d/

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
PATHDOCKERPKG=$(CURRENTDIR)target/DOCKER

# RPM Packaging path (where RPMs will be stored)
PATHRPMPKG=$(CURRENTDIR)target/RPM

# DEB Packaging path (where DEBs will be stored)
PATHDEBPKG=$(CURRENTDIR)target/DEB

# Prefix for the published docker container name
DOCKERPREFIX=

# Suffix for the published docker container name
DOCKERSUFFIX=

# Command used to check the configuration files
CONFCHECKCMD=check-jsonschema --schemafile resources/etc/${PROJECT}/config.schema.json

# Set default AWS region (if using AWS for deployments)
ifeq ($(AWS_DEFAULT_REGION),)
	AWS_DEFAULT_REGION=eu-west-1
endif

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
DOCKERTAG=$(VERSION)-$(RELEASE)

# Docker command
ifeq ($(DOCKER),)
	DOCKER=$(shell which docker)
endif

# Docker compose command
ifeq ($(DOCKERCOMPOSE),)
	DOCKERCOMPOSE=docker-compose
	DOCKERCOMPOSEPLUGIN=$(DOCKER) compose
	HASDOCKERCOMPOSE := $(shell $(DOCKERCOMPOSEPLUGIN) 2> /dev/null)
	ifdef HASDOCKERCOMPOSE
		DOCKERCOMPOSE=$(DOCKERCOMPOSEPLUGIN)
	endif
endif

DOCKERCOMPOSECMD=COMPOSE_PROJECT_NAME=$(VENDOR) $(DOCKERCOMPOSE)
DOCKERBUILDARG=--build-arg HOST_USER="$(shell id -u ${USER})" --build-arg HOST_GROUP="$(shell id -u ${GROUP})"

# Common commands
GO=GOPATH=$(GOPATH) GOPRIVATE=$(CVSPATH) $(shell which go)
GOVERSION=${shell go version | grep -Po '(go[0-9]+.[0-9]+)'}
GOFMT=$(shell which gofmt)
GOTEST=GOPATH=$(GOPATH) $(shell which gotest)
GODOC=GOPATH=$(GOPATH) $(shell which godoc)
GOLANGCILINT=$(BINUTIL)/golangci-lint
GOLANGCILINTVERSION=v1.56.2

# Current operating system and architecture as one string.
GOOSARCH=$(shell go env GOOS GOARCH | tr -d \\n)

# OS and Architecture used to build the Go binary for Docker.
LINUXGOBUILDENV=GOOS=linux GOARCH=amd64

# Environment variables for the go build command (uncomment the one that is appropriate):
#
# 32bit Raspbian:
#GOBUILDENV=env GOOS=linux GOARCH=arm GOARM=5
#
# 64bit Raspbian:
#GOBUILDENV=env GOOS=linux GOARCH=arm64
#
# Current environment
GOBUILDENV=

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

# Docker repository for the DEV environment
DOCKER_REGISTRY_DEV=

# Docker repository for the QA environment
DOCKER_REGISTRY_QA=

# Docker repository for the PROD environment
DOCKER_REGISTRY_PROD=

# Docker repository for the current environment
DOCKER_REGISTRY=

# Docker repository from where to pull the image
DOCKER_REGISTRY_PULL=

# Docker repository where to push the image
DOCKER_REGISTRY_PUSH=

# Command used to login into the Docker pull repository
# Example: "aws --profile MYENVPROFILE ecr get-login --no-include-email --region ${AWS_REGION} | sed 's|https://||'"
DOCKER_LOGIN_PULL="echo skip"

# Command used to login into the Docker push repository
# Example: "aws --profile MYENVPROFILE ecr get-login --no-include-email --region ${AWS_REGION} | sed 's|https://||'"
DOCKER_LOGIN_PUSH="echo skip"

# INT - integration environment via docker-compose
ifeq ($(DEPLOY_ENV), int)
	DOCKER_REGISTRY=${DOCKER_REGISTRY_DEV}
	#DOCKER_REGISTRY_PULL=
	#DOCKER_REGISTRY_PUSH=
	#DOCKER_LOGIN_PULL=
	#DOCKER_LOGIN_PUSH=
	GOSRVLIBEXAMPLE_URL=http://gosrvlibexample:8071
	GOSRVLIBEXAMPLE_MONITORING_URL=http://gosrvlibexample:8072
	API_TEST_FILE=*.yaml
endif

# Development environment
ifeq ($(DEPLOY_ENV), dev)
	DOCKER_REGISTRY=${DOCKER_REGISTRY_DEV}
	#DOCKER_REGISTRY_PULL=
	DOCKER_REGISTRY_PUSH=${DOCKER_REGISTRY_DEV}
	#DOCKER_LOGIN_PULL=
	#DOCKER_LOGIN_PUSH=
	GOSRVLIBEXAMPLE_URL=http://gosrvlibexample:8071
	GOSRVLIBEXAMPLE_MONITORING_URL=http://gosrvlibexample:8072
	API_TEST_FILE=*.yaml
endif

# QA environment
ifeq ($(DEPLOY_ENV), qa)
	DOCKER_REGISTRY=${DOCKER_REGISTRY_QA}
	DOCKER_REGISTRY_PULL=${DOCKER_REGISTRY_DEV}
	DOCKER_REGISTRY_PUSH=${DOCKER_REGISTRY_QA}
	#DOCKER_LOGIN_PULL=
	#DOCKER_LOGIN_PUSH=
	GOSRVLIBEXAMPLE_URL=http://gosrvlibexample:8071
	GOSRVLIBEXAMPLE_MONITORING_URL=http://gosrvlibexample:8072
endif

# Production environment
ifeq ($(DEPLOY_ENV), prod)
	DOCKER_REGISTRY=${DOCKER_REGISTRY_PROD}
	DOCKER_REGISTRY_PULL=${DOCKER_REGISTRY_QA}
	DOCKER_REGISTRY_PUSH=${DOCKER_REGISTRY_PROD}
	#DOCKER_LOGIN_PULL=
	#DOCKER_LOGIN_PUSH=
	GOSRVLIBEXAMPLE_URL=http://gosrvlibexample:8071
	GOSRVLIBEXAMPLE_MONITORING_URL=http://gosrvlibexample:8072
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
	@echo "    make apitest       : Execute API tests"
	@echo "    make buildall      : Full build and test sequence"
	@echo "    make build         : Compile the application"
	@echo "    make clean         : Remove any build artifact"
	@echo "    make confcheck     : Check the configuration files"
	@echo "    make coverage      : Generate the coverage report"
	@echo "    make dbuild        : Build everything inside a Docker container"
	@echo "    make deb           : Build a DEB package"
	@echo "    make deps          : Get dependencies"
	@echo "    make docker        : Build a scratch docker container to run this service"
	@echo "    make dockerpromote : Promote docker image from  DEV to PROD reporitory"
	@echo "    make dockerpush    : Push docker container to a remote repository"
	@echo "    make dockertest    : Test the newly built docker container"
	@echo "    make format        : Format the source code"
	@echo "    make generate      : Generate go code automatically"
	@echo "    make gendoc        : Generate static documentation from /doc/src"
	@echo "    make install       : Install this application"
	@echo "    make linter        : Check code against multiple linters"
	@echo "    make mod           : Download dependencies"
	@echo "    make modupdate     : Update dependencies"
	@echo "    make openapitest   : Test the OpenAPI specification"
	@echo "    make qa            : Run all tests and static analysis tools"
	@echo "    make rpm           : Build an RPM package"
	@echo "    make test          : Run unit tests"
	@echo ""
	@echo "Use DEVMODE=LOCAL for human friendly output."
	@echo ""
	@echo "To test and build everything from scratch:"
	@echo "    DEVMODE=LOCAL make format clean mod deps gendoc generate qa build docker dockertest"
	@echo "or use the shortcut:"
	@echo "    make x"
	@echo ""

# Alias for help target
all: help

# Alias to test and build everything from scratch
.PHONY: x
x:
	DEVMODE=LOCAL make format clean mod deps gendoc generate qa build docker dockertest

# Run venom tests (https://github.com/ovh/venom)
.PHONY: apitest
apitest:
	$(MAKE) venomtest API_TEST_DIR=monitoring API_TEST_URL=${GOSRVLIBEXAMPLE_MONITORING_URL} API_TEST_FILE=api.yaml
	$(MAKE) venomtest API_TEST_DIR=public API_TEST_URL=${GOSRVLIBEXAMPLE_URL} API_TEST_FILE=${API_TEST_FILE}

# Full build and test sequence
# You may want to change this and remove the options you don't need
#buildall: deps qa rpm deb bz2 crossbuild
.PHONY: buildall
buildall: build qa docker

# Compile the application
.PHONY: build
build:
	CGO_ENABLED=0 $(GOBUILDENV) \
	$(GO) build \
	-tags ${STATIC_TAG} \
	-ldflags '-w -s -X main.programVersion=${VERSION} -X main.programRelease=${RELEASE} -extldflags "-fno-PIC ${STATIC_FLAG}"' \
	-o ./target/${BINPATH}$(PROJECT) $(CMDDIR)

# Remove any build artifact
.PHONY: clean
clean:
	rm -rf $(TARGETDIR)

# Validate JSON configuration files against the JSON schema
.PHONY: confcheck
confcheck:
	${CONFCHECKCMD} resources/test/etc/${PROJECT}/config.json
	${CONFCHECKCMD} resources/etc/${PROJECT}/config.json

# Generate the coverage report
.PHONY: coverage
coverage: ensuretarget
	$(GO) tool cover -html=$(TARGETDIR)/report/coverage.out -o $(TARGETDIR)/report/coverage.html

# Build everything inside a Docker container
.PHONY: dbuild
dbuild: clean ensuretarget dockerdev
	@echo 0 > $(TARGETDIR)/make.exit
	CVSPATH=$(CVSPATH) VENDOR=$(LCVENDOR) PROJECT=$(PROJECT) MAKETARGET='$(MAKETARGET)' DOCKERTAG='$(DOCKERTAG)' $(CURRENTDIR)dockerbuild.sh
	@exit `cat $(TARGETDIR)/make.exit`

# Build the DEB package for Debian-like Linux distributions
.PHONY: deb
deb:
	rm -rf $(PATHDEBPKG)
	$(MAKE) install DESTDIR=$(PATHDEBPKG)/$(PKGNAME)-$(VERSION)
	rm -f $(PATHDEBPKG)/$(PKGNAME)-$(VERSION)/$(DOCPATH)LICENSE
	tar -zcvf $(PATHDEBPKG)/$(PKGNAME)_$(VERSION).orig.tar.gz -C $(PATHDEBPKG)/ $(PKGNAME)-$(VERSION)
	cp -rf ./resources/debian $(PATHDEBPKG)/$(PKGNAME)-$(VERSION)/debian
	mkdir -p $(PATHDEBPKG)/$(PKGNAME)-$(VERSION)/debian/missing-sources
	echo "// fake source for lintian" > $(PATHDEBPKG)/$(PKGNAME)-$(VERSION)/debian/missing-sources/$(PROJECT).c
	find $(PATHDEBPKG)/$(PKGNAME)-$(VERSION)/debian/ -type f -exec sed -i "s/~#DATE#~/`date -R`/" {} \;
	find $(PATHDEBPKG)/$(PKGNAME)-$(VERSION)/debian/ -type f -exec sed -i "s/~#PKGNAME#~/$(PKGNAME)/" {} \;
	find $(PATHDEBPKG)/$(PKGNAME)-$(VERSION)/debian/ -type f -exec sed -i "s/~#VERSION#~/$(VERSION)/" {} \;
	find $(PATHDEBPKG)/$(PKGNAME)-$(VERSION)/debian/ -type f -exec sed -i "s/~#RELEASE#~/$(RELEASE)/" {} \;
	echo $(BINPATH) > $(PATHDEBPKG)/$(PKGNAME)-$(VERSION)/debian/$(PKGNAME).dirs
	echo "$(BINPATH)* $(BINPATH)" > $(PATHDEBPKG)/$(PKGNAME)-$(VERSION)/debian/install
	echo $(DOCPATH) >> $(PATHDEBPKG)/$(PKGNAME)-$(VERSION)/debian/$(PKGNAME).dirs
	echo "$(DOCPATH)* $(DOCPATH)" >> $(PATHDEBPKG)/$(PKGNAME)-$(VERSION)/debian/install
ifneq ($(strip $(INITPATH)),)
	echo $(INITPATH) >> $(PATHDEBPKG)/$(PKGNAME)-$(VERSION)/debian/$(PKGNAME).dirs
	echo "$(INITPATH)* $(INITPATH)" >> $(PATHDEBPKG)/$(PKGNAME)-$(VERSION)/debian/install
endif
ifneq ($(strip $(CONFIGPATH)),)
	echo $(CONFIGPATH) >> $(PATHDEBPKG)/$(PKGNAME)-$(VERSION)/debian/$(PKGNAME).dirs
	echo "$(CONFIGPATH)* $(CONFIGPATH)" >> $(PATHDEBPKG)/$(PKGNAME)-$(VERSION)/debian/install
endif
ifneq ($(strip $(MANPATH)),)
	echo $(MANPATH) >> $(PATHDEBPKG)/$(PKGNAME)-$(VERSION)/debian/$(PKGNAME).dirs
	echo "$(MANPATH)* $(MANPATH)" >> $(PATHDEBPKG)/$(PKGNAME)-$(VERSION)/debian/install
endif
	echo "statically-linked-binary usr/bin/$(PROJECT)" > $(PATHDEBPKG)/$(PKGNAME)-$(VERSION)/debian/$(PKGNAME).lintian-overrides
	echo "new-package-should-close-itp-bug" >> $(PATHDEBPKG)/$(PKGNAME)-$(VERSION)/debian/$(PKGNAME).lintian-overrides
	echo "hardening-no-relro $(BINPATH)$(PROJECT)" >> $(PATHDEBPKG)/$(PKGNAME)-$(VERSION)/debian/$(PKGNAME).lintian-overrides
	echo "embedded-library $(BINPATH)$(PROJECT): libyaml" >> $(PATHDEBPKG)/$(PKGNAME)-$(VERSION)/debian/$(PKGNAME).lintian-overrides
	cd $(PATHDEBPKG)/$(PKGNAME)-$(VERSION) && debuild -us -uc

# Get the test dependencies
.PHONY: deps
deps: ensuretarget
	curl --silent --show-error --fail --location "https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh" | sh -s -- -b $(BINUTIL) $(GOLANGCILINTVERSION)
	$(GO) install github.com/mikefarah/yq/v4@latest
	$(GO) install github.com/hairyhenderson/gomplate/v3/cmd/gomplate@latest
	$(GO) install github.com/rakyll/gotest
	$(GO) install github.com/jstemmer/go-junit-report
	$(GO) install github.com/golang/mock/mockgen

# Build a docker container to run this service
.PHONY: docker
docker: dockerdir dockerbuild

# Build the docker container in the target/DOCKER directory
.PHONY: dockerbuild
dockerbuild:
	$(DOCKER) build --no-cache --tag=${LCVENDOR}/${PROJECT}$(DOCKERSUFFIX):latest $(PATHDOCKERPKG)

# Delete the Docker image
.PHONY: dockerdelete
dockerdelete:
	$(DOCKER) rmi -f `docker images "${LCVENDOR}/${PROJECT}$(DOCKERSUFFIX)" -q`

# Build a base development Docker image
.PHONY: dockerdev
dockerdev:
	$(DOCKER) build --pull --tag ${LCVENDOR}/dev_${PROJECT}:dev --file ./resources/docker/Dockerfile.dev ./resources/docker/

# Create the directory with docker files to be packaged
.PHONY: dockerdir
dockerdir:
ifneq ($(GOOSARCH),linuxamd64)
	$(MAKE) build GOBUILDENV=$(LINUXGOBUILDENV)
endif
ifneq ($(GOBUILDENV),)
	$(MAKE) build GOBUILDENV=$(LINUXGOBUILDENV)
endif
	rm -rf $(PATHDOCKERPKG)
	$(MAKE) install DESTDIR=$(PATHDOCKERPKG)
	$(MAKE) installssl DESTDIR=$(PATHDOCKERPKG)
	cp resources/docker/Dockerfile.run $(PATHDOCKERPKG)/Dockerfile

# Promote docker image from DEV to PROD
.PHONY: dockerpromote
dockerpromote:
	$(shell eval ${DOCKER_LOGIN_PULL})
	$(DOCKER) pull "${DOCKER_REGISTRY_PULL}/${DOCKERPREFIX}${PROJECT}$(DOCKERSUFFIX):$(DOCKERTAG)"
	$(DOCKER) tag "${DOCKER_REGISTRY_PULL}/${DOCKERPREFIX}${PROJECT}$(DOCKERSUFFIX):$(DOCKERTAG)" "${DOCKER_REGISTRY_PUSH}/${DOCKERPREFIX}${PROJECT}$(DOCKERSUFFIX):$(DOCKERTAG)"
	$(shell eval ${DOCKER_LOGIN_PUSH})
	$(DOCKER) push "${DOCKER_REGISTRY_PUSH}/${DOCKERPREFIX}${PROJECT}$(DOCKERSUFFIX):$(DOCKERTAG)"

# Push docker container to the remote repository
.PHONY: dockerpush
dockerpush:
	$(shell eval ${DOCKER_LOGIN_PUSH})
	$(DOCKER) tag "${LCVENDOR}/${PROJECT}$(DOCKERSUFFIX):latest" "${DOCKER_REGISTRY_PUSH}/${DOCKERPREFIX}${PROJECT}$(DOCKERSUFFIX):$(DOCKERTAG)"
	$(DOCKER) push "${DOCKER_REGISTRY_PUSH}/${DOCKERPREFIX}${PROJECT}$(DOCKERSUFFIX):$(DOCKERTAG)"
	$(DOCKER) tag "${LCVENDOR}/${PROJECT}$(DOCKERSUFFIX):latest" "${DOCKER_REGISTRY_PUSH}/${DOCKERPREFIX}${PROJECT}$(DOCKERSUFFIX):latest"
	$(DOCKER) push "${DOCKER_REGISTRY_PUSH}/${DOCKERPREFIX}${PROJECT}$(DOCKERSUFFIX):latest"

.PHONY: dockertest
dockertest: dockertestenv dockerdev
	test -f "$(BINUTIL)/dockerize" || curl --silent --show-error --fail --location https://github.com/jwilder/dockerize/releases/download/v0.6.1/dockerize-linux-amd64-v0.6.1.tar.gz | tar -xz -C $(BINUTIL)
	@echo 0 > $(TARGETDIR)/make.exit
	$(DOCKERCOMPOSECMD) down --volumes || true
	$(DOCKERCOMPOSECMD) build $(DOCKERBUILDARG)
	$(DOCKERCOMPOSECMD) run ${PROJECT}_integration || echo $${?} > $(TARGETDIR)/make.exit
	$(DOCKERCOMPOSECMD) down --rmi local --volumes --remove-orphans || true
	@exit `cat $(TARGETDIR)/make.exit`

# Run the integration tests; locally we need to execute 'build' and 'docker' targets first
.PHONY: dockertestenv
dockertestenv: ensuretarget
	@echo "GOSRVLIBEXAMPLE_REMOTECONFIGPROVIDER=envvar" > $(TARGETDIR)/gosrvlibexample.integration.env
	@echo "GOSRVLIBEXAMPLE_REMOTECONFIGDATA=$(shell cat resources/test/integration/gosrvlibexample/config.json | base64 | tr -d \\n)" >> $(TARGETDIR)/gosrvlibexample.integration.env

# Create the trget directories if missing
.PHONY: ensuretarget
ensuretarget:
	@mkdir -p $(TARGETDIR)/test
	@mkdir -p $(TARGETDIR)/report
	@mkdir -p $(BINUTIL)

# Format the source code
.PHONY: format
format:
	@find $(CMDDIR) -type f -name "*.go" -exec $(GOFMT) -s -w {} \;
	@find $(SRCDIR) -type f -name "*.go" -exec $(GOFMT) -s -w {} \;

# Generate test mocks
.PHONY: generate
generate:
	rm -f internal/mocks/*.go
	$(GO) generate $(GOPKGS)

# Generate static documentation
.PHONY: gendoc
gendoc:
	yq --input-format yaml --output-format json < doc/src/config.yaml . > doc/src/config.json
	check-jsonschema --schemafile doc/src/config.schema.json doc/src/config.json
	ASSUME_NO_MOVING_GC_UNSAFE_RISK_IT_WITH=$(GOVERSION) \
	gomplate \
	--datasource config=./doc/src/config.json \
	--template description=./doc/src/description.tmpl \
	--template development=./doc/src/development.tmpl \
	--template deployment=./doc/src/deployment.tmpl \
	--file ./doc/src/README.tmpl \
	--out README.md

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
	cp -f ./VERSION $(PATHINSTDOC)
	cp -f ./RELEASE $(PATHINSTDOC)
	cp -f ./doc/*.md $(PATHINSTDOC)
	chmod -R 644 $(PATHINSTDOC)*
ifneq ($(strip $(INITPATH)),)
	mkdir -p $(PATHINSTINIT)
	cp -rf ./resources/${INITPATH}* $(PATHINSTINIT)
	find $(PATHINSTINIT) -type d -exec chmod 755 {} \;
	find $(PATHINSTINIT) -type f -exec chmod 755 {} \;
endif
ifneq ($(strip $(CONFIGPATH)),)
	mkdir -p $(PATHINSTCFG)
	touch -c $(PATHINSTCFG)*
	cp -rf ./resources/${CONFIGPATH}* $(PATHINSTCFG)
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

# Execute multiple linter tools
.PHONY: linter
linter:
	@echo -e "\n\n>>> START: Static code analysis <<<\n\n"
	$(GOLANGCILINT) run --exclude-use-default=false --max-issues-per-linter 0 --max-same-issues 0 $(CMDDIR)/... $(SRCDIR)/...
	@echo -e "\n\n>>> END: Static code analysis <<<\n\n"

# Download dependencies
.PHONY: mod
mod:
	$(GO) mod download all

# Update dependencies
.PHONY: modupdate
modupdate:
	# $(GO) get $(shell $(GO) list -f '{{if not (or .Main .Indirect)}}{{.Path}}{{end}}' -m all)
	$(GO) get -t -u -d ./... && go mod tidy -compat=$(shell grep -oP 'go \K[0-9]+\.[0-9]+' go.mod)

# Test the OpenAPI specification against the real deployed service
.PHONY: openapitest
openapitest:
	$(MAKE) schemathesistest API_TEST_URL=${GOSRVLIBEXAMPLE_MONITORING_URL} OPENAPI_FILE=openapi_monitoring.yaml
	$(MAKE) schemathesistest API_TEST_URL=${GOSRVLIBEXAMPLE_URL} OPENAPI_FILE=openapi.yaml

# Ping the deployed service to check if the correct deployed container is alive
.PHONY: ping
ping:
	if [ "200_$(VERSION)_$(RELEASE)_" != "$(shell curl --silent --insecure '$(GOSRVLIBEXAMPLE_MONITORING_URL)/ping' | jq -r '.code,.version,.release' | tr '\n' '_')" ]; then exit 1; fi

# Run all tests and static analysis tools
.PHONY: qa
qa: linter confcheck test coverage

# Retry the ping command automatically (try 60 times every 5 sec = 5 min max)
.PHONY: rping
rping:
	$(call make_retry,ping,60,5)

# Build the RPM package for RedHat-like Linux distributions
.PHONY: rpm
rpm:
	rm -rf $(PATHRPMPKG)
	rpmbuild \
	--define "_topdir $(PATHRPMPKG)" \
	--define "_vendor $(VENDOR)" \
	--define "_owner $(OWNER)" \
	--define "_project $(PROJECT)" \
	--define "_package $(PKGNAME)" \
	--define "_version $(VERSION)" \
	--define "_release $(RELEASE)" \
	--define "_current_directory $(CURRENTDIR)" \
	--define "_binpath /$(BINPATH)" \
	--define "_docpath /$(DOCPATH)" \
	--define "_configpath /$(CONFIGPATH)" \
	--define "_initpath /$(INITPATH)" \
	--define "_manpath /$(MANPATH)" \
	-bb resources/rpm/rpm.spec

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

# Run the unit tests
.PHONY: test
test: ensuretarget
	@echo -e "\n\n>>> START: Unit Tests <<<\n\n"
	$(GOTEST) \
	-shuffle=on \
	-tags=unit,benchmark \
	-covermode=atomic \
	-bench=. \
	-race \
	-failfast \
	-coverprofile=$(TARGETDIR)/report/coverage.out \
	-v $(GOPKGS) $(TESTEXTRACMD)
	@echo -e "\n\n>>> END: Unit Tests <<<\n\n"

# Remove all installed files (excluding configuration files)
.PHONY: uninstall
uninstall:
	rm -rf $(PATHINSTBIN)$(PROJECT)
	rm -rf $(PATHINSTDOC)

# Run venom tests (https://github.com/ovh/venom)
.PHONY: venomtest
venomtest:
	@mkdir -p $(TARGETDIR)/report/${DEPLOY_ENV}/venom/$(API_TEST_DIR)
	venom run \
		--var gosrvlibexample.url="${API_TEST_URL}" \
		--var gosrvlibexample.version="${VERSION}" \
		--var gosrvlibexample.release="${RELEASE}" \
		-vv \
		--output-dir=$(TARGETDIR)/report/${DEPLOY_ENV}/venom/$(API_TEST_DIR) \
		resources/test/venom/$(API_TEST_DIR)/$(API_TEST_FILE)
