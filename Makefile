# SPDX-FileCopyrightText: (C) 2023 Intel Corporation
# SPDX-License-Identifier: LicenseRef-Intel

NAME ?= os-update
BUILD_DIR ?= build/artifacts
SCRIPTS_DIR := ./ci_scripts
TOPDIR = $(shell pwd)/rpm
PACKAGE_BUILD_DIR ?= $(BUILD_DIR)/package
PKG_VERSION := $(shell if grep -q dev os-update-modules/VERSION; then echo $$(cat os-update-modules/VERSION)-$$(git rev-parse --short HEAD); else cat os-update-modules/VERSION; fi)
VERSION := $(shell cat os-update-modules/VERSION)
COMMIT := $(shell git rev-parse --short HEAD)
ifneq (,$(findstring dev,$(VERSION)))
	PKG_VERSION = $(VERSION)-$(COMMIT)
else
	PKG_VERSION = $(VERSION)
endif
TARBALL_DIR := $(BUILD_DIR)/$(NAME)-$(PKG_VERSION)
# Define variables for paths
BINDIR = $(DESTDIR)/usr/bin
MODULEDIR = $(BINDIR)/os-update-modules

.PHONY: all build clean help lint list package test tarball

all: tarball
	@# Help: runs build, lint, test & package targets

clean:
	@# Help: deletes build directory
	rm -rf $(BUILD_DIR)/*
	rm -rf rpm/BUILD rpm/RPMS rpm/SOURCES rpm/SRPMS

tarball:
	@# Help: creates source tarball
	@echo "---MAKEFILE TARBALL---"

	mkdir -p $(TARBALL_DIR)
	mkdir -p rpm/BUILD rpm/RPMS rpm/SOURCES rpm/SRPMS
	cp -r os-update-modules/ Makefile os-update-tool.sh $(TARBALL_DIR)
	sed -i "s#COMMIT := .*#COMMIT := $(COMMIT)#" $(TARBALL_DIR)/Makefile
	tar -zcf $(BUILD_DIR)/$(NAME)-$(PKG_VERSION).tar.gz --directory=$(BUILD_DIR) $(NAME)-$(PKG_VERSION)
	cp $(BUILD_DIR)/$(NAME)-$(PKG_VERSION).tar.gz ./rpm/SOURCES

	@echo "---END MAKEFILE TARBALL---"

rpm_package:
	rpmbuild -ba rpm/SPECS/$(NAME).spec --define "_topdir $(TOPDIR)"

install:
	@echo "Installing to $(BINDIR) and $(MODULEDIR)"
	# Create directories
	mkdir -p $(MODULEDIR)

	# Install the script files
	install -p -m 0770 os-update-tool.sh $(BINDIR)/os-update-tool.sh
	install -p -m 0660 os-update-modules/VERSION $(MODULEDIR)/VERSION
	install -p -m 0660 os-update-modules/common.sh $(MODULEDIR)/common.sh
	install -p -m 0660 os-update-modules/os-update-tool.config $(MODULEDIR)/os-update-tool.config
	install -p -m 0660 os-update-modules/get_image.sh $(MODULEDIR)/get_image.sh
	install -p -m 0660 os-update-modules/log.sh $(MODULEDIR)/log.sh
	install -p -m 0660 os-update-modules/OSutil.sh $(MODULEDIR)/OSutil.sh
	install -p -m 0660 os-update-modules/systemd_boot_config.sh $(MODULEDIR)/systemd_boot_config.sh
	install -p -m 0660 os-update-modules/writes.sh $(MODULEDIR)/writes.sh

list: help
	@# Help: displays make targets

help:
	@printf "%-20s %s\n" "Target" "Description"
	@printf "%-20s %s\n" "------" "-----------"
	@make -pqR : 2>/dev/null \
		| awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' \
		| sort \
		| egrep -v -e '^[^[:alnum:]]' -e '^$@$$' \
		| xargs -I _ sh -c 'printf "%-20s " _; make _ -nB | (grep -i "^# Help:" || echo "") | tail -1 | sed "s/^# Help: //g"'
