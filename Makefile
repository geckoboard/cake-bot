NAME = cake-bot
LINT_VERSION = 1.60.3

# This is needed as test-ci uses bash process substitution
SHELL = /bin/bash

.PHONY: build clean run test test-ci

build:
	@mkdir -p bin
	go build -o bin/$(NAME)

clean:
	rm -f bin/$(NAME)

run: build
	bin/$(NAME)

test:
	go test -v ./...

test-ci:
	go test -v ./... -race

.PHONY: lint
lint: ./bin/golangci-lint ## lints the go code
	./bin/golangci-lint run

./bin/golangci-lint:
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ./bin v$(LINT_VERSION)
