.PHONY: all

PKGS := $(shell go list ./... | grep -v '/vendor/')
GOFILES := $(shell go list -f '{{range $$index, $$element := .GoFiles}}{{$$.Dir}}/{{$$element}}{{"\n"}}{{end}}' ./... | grep -v '/vendor/')
TXT_FILES := $(shell find * -type f -not -path 'vendor/**')


default: clean checks test build

test: clean
	go test -v -cover $(PKGS)

dependencies:
	dep ensure -v

clean:
	rm -f cover.out

build:
	go build

checks: check-fmt
	gometalinter --vendor --enable=misspell ./...

check-fmt: SHELL := /bin/bash
check-fmt:
	diff -u <(echo -n) <(gofmt -d $(GOFILES))
