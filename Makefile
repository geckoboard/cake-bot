NAME = cake-bot
GIT_REF = $(shell git rev-parse --short HEAD)
FPM_VERSION = 1.3.3
DEB_S3_VERSION = 0.7.1
AWS_CLI_VERSION = 1.7.3
# This is needed as test-ci uses bash process substitution
SHELL = /bin/bash

.PHONY: all build clean import install-ci-dep package release run run-web test test-ci test-race update-deps

all: test

build:
	@mkdir -p bin
	go build -o bin/$(NAME)

clean:
	rm -f bin/$(NAME)

run: build
	bin/$(NAME)

run-web: build
	bin/$(NAME) -port 9098

test:
	go test -v ./...

test-ci:
	go test -v ./... -race | tee >(go-junit-report -package-name $(NAME) > $$CIRCLE_TEST_REPORTS/golang.xml); test $${PIPESTATUS[0]} -eq 0

test-race:
	go test -v ./... -race

update-deps:
	go list -f '{{if not .Standard}}{{join .Deps "\n"}}{{end}}' ./... \
	  | sort -u \
	  | grep -v github.com/geckoboard/$(NAME) \
	  | xargs go get -f -u -d -v

install-ci-deps:
	go get -f -u github.com/jstemmer/go-junit-report
