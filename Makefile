BIN_NAME=keg
BIN_DIR=bin
COMPOSE_DIR=./docker/docker-compose.yaml

.PHONY: build clean test deps cluster-down cluster-up cluster-reset


# Build the project
build:
	go build -o $(BIN_DIR)/$(BIN_NAME) -v

# Test the project
test:
	go test -v ./...

# Clean the project
clean:
	go clean -v
	rm -f $(BIN_DIR)/$(BIN_NAME)

# Install dependencies
deps:
	go get -v ./...

cluster-down:
	docker compose -f $(COMPOSE_DIR) down -v

cluster-up:
	 docker compose -f $(COMPOSE_DIR) up -d

cluster-reset: cluster-down cluster-up