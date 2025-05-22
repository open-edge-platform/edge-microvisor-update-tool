# Variables
APP_NAME = os-curation-tools
SRC_DIR = ./cmd
BUILD_DIR = ./bin
WORK_DIR = ./build
COVERAGE_DIR = ./coverage

# Commands
build:
	@echo "Building the application..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(APP_NAME) $(SRC_DIR)
	@echo "Build completed. Binary is located at $(BUILD_DIR)/$(APP_NAME)"

lint:
	@echo "Running Go linter..."
	@golangci-lint run ./... --config .golangci.yml --skip-dirs $(shell go env GOPATH)
	@echo "Linting completed."

test:
	@echo "Running unit tests..."
	@mkdir -p $(COVERAGE_DIR)
	@go test ./... -v -coverprofile=$(COVERAGE_DIR)/coverage.out
	@go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "Unit tests completed. Coverage report is located at $(COVERAGE_DIR)/coverage.html"

.PHONY: build lint test