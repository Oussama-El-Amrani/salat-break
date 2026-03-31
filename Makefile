BINARY_NAME=salat-break
BUILD_DIR=cmd/salat-break

all: build

build:
	go build -o $(BINARY_NAME) ./$(BUILD_DIR)

clean:
	rm -f $(BINARY_NAME)

run: build
	./$(BINARY_NAME)

test:
	go test ./...

install: build
	mkdir -p $(HOME)/.local/bin
	cp $(BINARY_NAME) $(HOME)/.local/bin/

.PHONY: all build clean run test install
