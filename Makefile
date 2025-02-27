.PHONY: build clean run

BIN_NAME=kube-event-generator

build:
	@go build -o bin/$(BIN_NAME) main.go

clean:
	@echo "Cleaning..."
	@rm -rf bin/

run: build
	@./bin/$(BIN_NAME) run