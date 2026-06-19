.PHONY: all build install test clean fmt tidy

# Variables
BINARY_NAME=tf-drift
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

all: test build

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) ./cmd/tf-drift

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/tf-drift

test:
	go test -v ./...

clean:
	rm -f $(BINARY_NAME)

fmt:
	go fmt ./...

tidy:
	go mod tidy
