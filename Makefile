BINARY_NAME=anti2verge
VERSION=1.0.0
BUILD_DIR=bin

.PHONY: all build build-gui clean check run test

all: build build-gui

deps:
	go mod tidy

build: deps
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "-s -w -X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-cli.exe ./cmd/anti2verge

build-gui: deps
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "-H windowsgui -s -w -X main.version=$(VERSION)" -o $(BUILD_DIR)/Anti2Verge.exe ./cmd/anti2verge

check:
	go run ./cmd/anti2verge check

run:
	go run ./cmd/anti2verge run

clean:
	rm -rf $(BUILD_DIR)
