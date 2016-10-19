# MAKEFILE
#
# @author      Nicola Asuni <info@tecnick.com>
# @link        https://github.com/tecnickcom/allgen
#
# This file is intended to be executed in a Linux-compatible system.
# ------------------------------------------------------------------------------

# List special make targets that are not associated with files
.PHONY: help all new newproject renameapp renamelib template

# Current directory
CURRENTDIR=$(shell pwd)

# Project type
ifeq ($(TYPE),)
	TYPE=app
endif

# Project configuration file
ifeq ($(CONFIG),)
	CONFIG=default.cfg
endif

# include the configuration file
include $(CONFIG)

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

# Alias for help target
all: help

# Generate a new go project
new: newproject rename$(TYPE) template confirm

# Copy the project template in the output folder
newproject:
	@mkdir -p ./target/$(CVSPATH)/$(PROJECT)
	@rm -rf ./target/$(CVSPATH)/$(PROJECT)/*
	@cp -rf ./src/$(TYPE)/* ./target/$(CVSPATH)/$(PROJECT)/

# Rename some application files
renameapp:
	@mv ./target/$(CVSPATH)/$(PROJECT)/resources/usr/share/man/man1/project.1 ./target/$(CVSPATH)/$(PROJECT)/resources/usr/share/man/man1/$(PROJECT).1
	@mv ./target/$(CVSPATH)/$(PROJECT)/resources/etc/project ./target/$(CVSPATH)/$(PROJECT)/resources/etc/$(PROJECT)
	@mv ./target/$(CVSPATH)/$(PROJECT)/resources/etc/init.d/project ./target/$(CVSPATH)/$(PROJECT)/resources/etc/init.d/$(PROJECT)
	@mv ./target/$(CVSPATH)/$(PROJECT)/resources/test/etc/project ./target/$(CVSPATH)/$(PROJECT)/resources/test/etc/$(PROJECT)

# Rename some lib files
renamelib:
	@mv ./target/$(CVSPATH)/$(PROJECT)/project.go ./target/$(CVSPATH)/$(PROJECT)/$(LIBPACKAGE).go
	@mv ./target/$(CVSPATH)/$(PROJECT)/project_test.go ./target/$(CVSPATH)/$(PROJECT)/$(LIBPACKAGE)_test.go

# Replace text templates in the code
template:
	@find ./target/$(CVSPATH)/$(PROJECT)/ -type f -exec sed -i "s/~#PROJECT#~/$(PROJECT)/" {} \;
	@find ./target/$(CVSPATH)/$(PROJECT)/ -type f -exec sed -i "s/~#UPROJECT#~/$(UPROJECT)/" {} \;
	@find ./target/$(CVSPATH)/$(PROJECT)/ -type f -exec sed -i "s/~#SHORTDESCRIPTION#~/$(SHORTDESCRIPTION)/" {} \;
	@find ./target/$(CVSPATH)/$(PROJECT)/ -type f -exec sed -i "s|~#CVSPATH#~|$(CVSPATH)|" {} \;
	@find ./target/$(CVSPATH)/$(PROJECT)/ -type f -exec sed -i "s/~#VENDOR#~/$(VENDOR)/" {} \;
	@find ./target/$(CVSPATH)/$(PROJECT)/ -type f -exec sed -i "s/~#OWNER#~/$(OWNER)/" {} \;
	@find ./target/$(CVSPATH)/$(PROJECT)/ -type f -exec sed -i "s/~#CURRENTYEAR#~/$(CURRENTYEAR)/" {} \;
	@find ./target/$(CVSPATH)/$(PROJECT)/ -type f -exec sed -i "s/~#LIBPACKAGE#~/$(LIBPACKAGE)/" {} \;

# Print confirmation message
confirm:
	@echo "A new "$(TYPE)" project has been created: "target/$(CVSPATH)/$(PROJECT)

# Remove all generated projects
clean:
	@rm -rf ./target
