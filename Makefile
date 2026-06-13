BINARY_NAME=keepalive
BUILD_DIR=build
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X keepalive/internal/cmd.version=$(VERSION) -X keepalive/internal/cmd.commit=$(COMMIT) -X keepalive/internal/cmd.date=$(DATE)"

.PHONY: build run test clean lint

build:
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/keepalive

run:
	go run $(LDFLAGS) ./cmd/keepalive

test:
	go test ./... -v

clean:
	rm -rf $(BUILD_DIR)

lint:
	golangci-lint run ./...

.DEFAULT_GOAL := build
