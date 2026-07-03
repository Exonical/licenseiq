BUILD_DIR ?= bin

.PHONY: build test lint vet run tidy

build:
	go build ./...

test:
	go test ./...

lint:
	golangci-lint run

vet:
	go vet ./...

run:
	go run ./cmd/licenseiq

tidy:
	go mod tidy
