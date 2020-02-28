# MAKEFILE
#
# @author      Nicola Asuni <info@tecnick.com>
# @link        https://github.com/tecnickcom/gogen
#
# This file is intended to be executed in a Linux-compatible system.
# ------------------------------------------------------------------------------

# Current directory
CURRENTDIR=$(dir $(realpath $(firstword $(MAKEFILE_LIST))))

# Set default project type
ifeq ($(TYPE),)
	TYPE=app
endif

# Set default project configuration file
ifeq ($(CONFIG),)
	CONFIG=default.cfg
endif

# Include the configuration file
include $(CONFIG)

# Generate a prefix for environmental variables
UPROJECT=$(shell echo $(PROJECT) | tr a-z A-Z | tr - _)


# --- MAKE TARGETS ---

# Display general help about this command
.PHONY: help
help:
	@echo ""
	@echo "gogen Makefile."
	@echo "The following commands are available:"
	@echo ""
	@echo "    make new TYPE=app CONFIG=myproject.cfg  :  Generate a new go project"
	@echo "    make clean                              :  Remove all generated projects"
	@echo ""
	@echo "    * TYPE is the project type:"
	@echo "        lib : library"
	@echo "        app : command-line application"
	@echo "        srv : HTTP API service"
	@echo ""
	@echo "    * CONFIG is the configuration file containing the project settings."
	@echo ""

# Alias for help target
all: help

# Generate a new go project
.PHONY: new
new: newproject rename$(TYPE) template confirm

# Copy the project template in the output folder
.PHONY: newproject
newproject:
	@mkdir -p ./target/$(CVSPATH)/$(PROJECT)
	@rm -rf ./target/$(CVSPATH)/$(PROJECT)/*
	@cp -rf ./src/$(TYPE)/. ./target/$(CVSPATH)/$(PROJECT)/

# Rename some application files
.PHONY: renameapp
renameapp:
	@mv ./target/$(CVSPATH)/$(PROJECT)/resources/usr/share/man/man1/project.1 ./target/$(CVSPATH)/$(PROJECT)/resources/usr/share/man/man1/$(PROJECT).1
	@mv ./target/$(CVSPATH)/$(PROJECT)/resources/etc/project ./target/$(CVSPATH)/$(PROJECT)/resources/etc/$(PROJECT)
	@mv ./target/$(CVSPATH)/$(PROJECT)/resources/test/etc/project ./target/$(CVSPATH)/$(PROJECT)/resources/test/etc/$(PROJECT)

# Rename some service files
.PHONY: renamesrv
renamesrv: renameapp
	@mv ./target/$(CVSPATH)/$(PROJECT)/resources/etc/init.d/project ./target/$(CVSPATH)/$(PROJECT)/resources/etc/init.d/$(PROJECT)
	@mv ./target/$(CVSPATH)/$(PROJECT)/project.test.Dockerfile ./target/$(CVSPATH)/$(PROJECT)/$(PROJECT).test.Dockerfile
	
# Rename some service files
.PHONY: renamesrvnosql
renamesrvnosql: renamesrv

# Rename some lib files
.PHONY: renamelib
renamelib:
	@mv ./target/$(CVSPATH)/$(PROJECT)/project.go ./target/$(CVSPATH)/$(PROJECT)/$(LIBPACKAGE).go
	@mv ./target/$(CVSPATH)/$(PROJECT)/project_test.go ./target/$(CVSPATH)/$(PROJECT)/$(LIBPACKAGE)_test.go

# Replace text templates in the code
.PHONY: template
template:
	@find ./target/$(CVSPATH)/$(PROJECT)/ -type f -exec sed -i "s/~#PROJECT#~/$(PROJECT)/g" {} \;
	@find ./target/$(CVSPATH)/$(PROJECT)/ -type f -exec sed -i "s/~#UPROJECT#~/$(UPROJECT)/g" {} \;
	@find ./target/$(CVSPATH)/$(PROJECT)/ -type f -exec sed -i "s/~#SHORTDESCRIPTION#~/$(SHORTDESCRIPTION)/g" {} \;
	@find ./target/$(CVSPATH)/$(PROJECT)/ -type f -exec sed -i "s|~#CVSPATH#~|$(CVSPATH)|g" {} \;
	@find ./target/$(CVSPATH)/$(PROJECT)/ -type f -exec sed -i "s|~#PROJECTLINK#~|$(PROJECTLINK)|g" {} \;
	@find ./target/$(CVSPATH)/$(PROJECT)/ -type f -exec sed -i "s/~#VENDOR#~/$(VENDOR)/g" {} \;
	@find ./target/$(CVSPATH)/$(PROJECT)/ -type f -exec sed -i "s/~#OWNER#~/$(OWNER)/g" {} \;
	@find ./target/$(CVSPATH)/$(PROJECT)/ -type f -exec sed -i "s/~#OWNEREMAIL#~/$(OWNEREMAIL)/g" {} \;
	@find ./target/$(CVSPATH)/$(PROJECT)/ -type f -exec sed -i "s/~#CURRENTYEAR#~/$(CURRENTYEAR)/g" {} \;
	@find ./target/$(CVSPATH)/$(PROJECT)/ -type f -exec sed -i "s/~#LIBPACKAGE#~/$(LIBPACKAGE)/g" {} \;

# Print confirmation message
.PHONY: confirm
confirm:
	@echo "A new "$(TYPE)" project has been created: "target/$(CVSPATH)/$(PROJECT)

# Remove all generated projects
.PHONY: clean
clean:
	@rm -rf ./target

.PHONY: test
test:
	@echo "*** SRV ***"
	make clean new TYPE=srv
	cd target/github.com/dummyvendor/dummy && make buildall
	@echo "*** APP ***"
	make clean new TYPE=app
	cd target/github.com/dummyvendor/dummy && make buildall
	@echo "*** LIB ***"
	make clean new TYPE=lib
	cd target/github.com/dummyvendor/dummy && make buildall
