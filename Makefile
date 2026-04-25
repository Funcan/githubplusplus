.PHONY: build install clean test fmt default

BINARY := gh++

default: fmt build test

build:
	go build -o $(BINARY) .

install:
	go install .
	@echo "Installed. Make sure $$(go env GOPATH)/bin is in your PATH."

clean:
	rm -f $(BINARY) coverage.out coverage.html

test:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

fmt:
	gofmt -w .
