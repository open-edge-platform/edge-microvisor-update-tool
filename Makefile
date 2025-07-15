# Variables
APP_NAME = os-ab-update
SRC_DIR = ./cmd
BUILD_DIR = ./build
COVERAGE_DIR = ./coverage
TOPDIR = $(shell pwd)/rpm
PKG_VERSION := $(shell cat VERSION)
TARBALL_DIR := $(BUILD_DIR)/$(APP_NAME)-$(PKG_VERSION)
BINDIR = $(DESTDIR)/usr/bin

# Commands
build:
	@echo "Building the application..."
	@mkdir -p $(BUILD_DIR)
	@go build -ldflags "-X main.Version=$(PKG_VERSION)" -o $(APP_NAME) $(SRC_DIR)
	@echo "Build completed. Binary is located at ./$(APP_NAME)"

install:
	@echo "Installing to $(BINDIR)"
	mkdir -p $(BINDIR)
	install -p -m 0770 ./$(APP_NAME) $(BINDIR)/$(APP_NAME)
	@echo "Installation completed. Binary is located at /usr/bin/$(APP_NAME)"

lint:
	@echo "Running Go linter..."
	@golangci-lint run ./... --config .golangci.yml --skip-dirs $(shell go env GOPATH)
	@echo "Linting completed."

# Need to modify for all folders in internal folder
unit_test:
	@echo "Running unit tests..."
	@go test -v ./internal/... 
	@echo "unit test execution completed for all modules"

cover_unit:
	mkdir -p $(BUILD_DIR)/coverage/unit
	go test -v ./internal/... -cover -covermode count -args -test.gocoverdir=$(shell pwd)/$(BUILD_DIR)/coverage/unit | tee $(BUILD_DIR)/coverage/unit/unit.out
	go tool covdata percent -i=$(BUILD_DIR)/coverage/unit
	go tool covdata func -i=$(BUILD_DIR)/coverage/unit

.PHONY: build lint unit_test cover_unit tarball rpm_package

tarball:
	@# Help: creates source tarball
	@echo "---MAKEFILE TARBALL---"

	mkdir -p $(TARBALL_DIR)
	mkdir -p rpm/BUILD rpm/RPMS rpm/SOURCES rpm/SRPMS
	cp -r cmd/ internal/ pkg/ Makefile VERSION $(APP_NAME) $(TARBALL_DIR)
	sed -i "s#COMMIT := .*#COMMIT := $(COMMIT)#" $(TARBALL_DIR)/Makefile
	tar -zcf $(BUILD_DIR)/$(APP_NAME)-$(PKG_VERSION).tar.gz --directory=$(BUILD_DIR) $(APP_NAME)-$(PKG_VERSION)
	cp $(BUILD_DIR)/$(APP_NAME)-$(PKG_VERSION).tar.gz ./rpm/SOURCES

	@echo "---END MAKEFILE TARBALL---"

rpm_package:
	rpmbuild -ba rpm/SPECS/$(APP_NAME).spec --define "_topdir $(TOPDIR)"

clean:
	@# Help: deletes build directory
	rm -rf $(BUILD_DIR)/*
	rm ./$(APP_NAME)
	rm -rf rpm/BUILDROOT rpm/BUILD rpm/RPMS rpm/SOURCES rpm/SRPMS
