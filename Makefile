.PHONY: all build install test clean fmt tidy

# Variables
BINARY_NAME=tf-drift

all: test build

build:
	go build -o $(BINARY_NAME)

install:
	go install

test:
	go test -v ./...

clean:
	rm -f $(BINARY_NAME)

fmt:
	go fmt ./...

tidy:
	go mod tidy
