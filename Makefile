NAME = cake-bot
GIT_REF = $(shell git rev-parse --short HEAD)

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
