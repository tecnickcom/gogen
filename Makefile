# MAKEFILE
#
# @author      Nicola Asuni <info@tecnick.com>
# @link        https://github.com/tecnickcom/gogen
#
# This file is intended to be executed in a Linux-compatible system.
# ------------------------------------------------------------------------------

# List special make targets that are not associated with files
.PHONY: help all new newproject renameapp renamesrv renamelib template confirm clean

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
help:
	@echo ""
	@echo "gogen Makefile."
	@echo "The following commands are available:"
	@echo ""
	@echo "    make new TYPE=app CONFIG=myproject.cfg  :  Generate a new go project"
	@echo "    make clean                              :  Remove all generated projects"
	@echo ""
	@echo "    * TYPE is the project type:"
	@echo "        lib      : library"
	@echo "        app      : command-line application"
	@echo "        srv      : HTTP API service"
	@echo "        srvnosql : HTTP API service with MongoDB and Elasticsearch backend support"
	@echo ""
	@echo "    * CONFIG is the configuration file containing the project settings."
	@echo ""

# Alias for help target
all: help

# Generate a new go project
new: newproject rename$(TYPE) template confirm

# Copy the project template in the output folder
newproject:
	@mkdir -p ./target/$(CVSPATH)/$(PROJECT)
	@rm -rf ./target/$(CVSPATH)/$(PROJECT)/*
	@cp -rf ./src/$(TYPE)/. ./target/$(CVSPATH)/$(PROJECT)/

# Rename some application files
renameapp:
	@mv ./target/$(CVSPATH)/$(PROJECT)/resources/usr/share/man/man1/project.1 ./target/$(CVSPATH)/$(PROJECT)/resources/usr/share/man/man1/$(PROJECT).1
	@mv ./target/$(CVSPATH)/$(PROJECT)/resources/etc/project ./target/$(CVSPATH)/$(PROJECT)/resources/etc/$(PROJECT)
	@mv ./target/$(CVSPATH)/$(PROJECT)/resources/test/etc/project ./target/$(CVSPATH)/$(PROJECT)/resources/test/etc/$(PROJECT)

# Rename some service files
renamesrv: renameapp
	@mv ./target/$(CVSPATH)/$(PROJECT)/resources/etc/init.d/project ./target/$(CVSPATH)/$(PROJECT)/resources/etc/init.d/$(PROJECT)
	
# Rename some service files
renamesrvnosql: renamesrv

# Rename some lib files
renamelib:
	@mv ./target/$(CVSPATH)/$(PROJECT)/project.go ./target/$(CVSPATH)/$(PROJECT)/$(LIBPACKAGE).go
	@mv ./target/$(CVSPATH)/$(PROJECT)/project_test.go ./target/$(CVSPATH)/$(PROJECT)/$(LIBPACKAGE)_test.go

# Replace text templates in the code
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
confirm:
	@echo "A new "$(TYPE)" project has been created: "target/$(CVSPATH)/$(PROJECT)

# Remove all generated projects
clean:
	@rm -rf ./target
