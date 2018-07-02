.PHONY: all

GOFILES := $(shell go list -f '{{range $$index, $$element := .GoFiles}}{{$$.Dir}}/{{$$element}}{{"\n"}}{{end}}' ./... | grep -v '/vendor/')

default: clean checks test build

test: clean
	go test -v -cover .

integration-tests: 
	go test -v -timeout=20m ./integration/*_test.go -sfintegration.verbose

dependencies:
	dep ensure -v --vendor-only

clean:
	rm -f cover.out

build:
	go build

checks: check-fmt
	gometalinter --vendor --disable=vetshadow --enable=misspell --deadline=2m ./...

check-fmt: SHELL := /bin/bash
check-fmt:
	diff -u <(echo -n) <(gofmt -d $(GOFILES))
