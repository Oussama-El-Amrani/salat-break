BINARY_NAME=salat-break
BUILD_DIR=cmd/salat-break

all: build

VERSION=$(shell git describe --tags --always --dirty)

build:
	go build -ldflags "-X main.Version=$(VERSION)" -o $(BINARY_NAME) ./$(BUILD_DIR)

build-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Version=$(VERSION)" -o $(BINARY_NAME)-linux-amd64 ./$(BUILD_DIR)

build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.Version=$(VERSION)" -o $(BINARY_NAME)-darwin-amd64 ./$(BUILD_DIR)

build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.Version=$(VERSION)" -o $(BINARY_NAME)-darwin-arm64 ./$(BUILD_DIR)

clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME)-linux-amd64 $(BINARY_NAME)-darwin-amd64 $(BINARY_NAME)-darwin-arm64

run: build
	./$(BINARY_NAME)

test:
	go test ./...

install: build
	mkdir -p $(HOME)/.local/bin
	cp $(BINARY_NAME) $(HOME)/.local/bin/

.PHONY: all build build-linux build-darwin-amd64 build-darwin-arm64 clean run test install
