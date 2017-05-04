NAME = cake-bot
TEST_PKGS?=$$(go list ./... | grep -v /vendor/)
GIT_REF = $(shell git rev-parse --short HEAD)

# This is needed as test-ci uses bash process substitution
SHELL = /bin/bash

.PHONY: all build clean install-ci-dep run run-web test test-ci test-race update-deps

all: test

build:
	@mkdir -p bin
	go build -o bin/$(NAME)

clean:
	rm -f bin/$(NAME)

run: build
	bin/$(NAME)

test:
	go test -v $(TEST_PKGS)

test-ci: install-ci-deps
	go test -v $(TEST_PKGS) -race | tee >(go-junit-report -package-name $(NAME) > $$CIRCLE_TEST_REPORTS/golang.xml); test $${PIPESTATUS[0]} -eq 0

test-race:
	go test -v $(TEST_PKGS) -race

update-deps:
	go list -f '{{if not .Standard}}{{join .Deps "\n"}}{{end}}' ./... \
	  | sort -u \
	  | grep -v github.com/geckoboard/$(NAME) \
	  | xargs go get -f -u -d -v

install-ci-deps:
	go get -f -u github.com/jstemmer/go-junit-report
