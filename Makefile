# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=kube-event-generator
BINARY_UNIX=$(BINARY_NAME)_unix
BIN_DIR=bin

# All target
all: test build

# Build the project
build:
	$(GOBUILD) -o $(BIN_DIR)/$(BINARY_NAME) -v

# Run the project
run:
	$(GOBUILD) -o $(BIN_DIR)/$(BINARY_NAME) -v ./...
	./$(BIN_DIR)/$(BINARY_NAME)

# Test the project
test:
	$(GOTEST) -v ./...

# Clean the project
clean:
	$(GOCLEAN)
	rm -f $(BIN_DIR)/$(BINARY_NAME)
	rm -f $(BIN_DIR)/$(BINARY_UNIX)

# Install dependencies
deps:
	$(GOGET) -v ./...

# Cross compilation for Linux
build-linux:
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BIN_DIR)/$(BINARY_UNIX) -v

.PHONY: all build clean run test deps build-linux sim-run sim-gen

sim-run: build
	./$(BIN_DIR)/$(BINARY_NAME) sim run

sim-gen: build
	./$(BIN_DIR)/$(BINARY_NAME) sim gen