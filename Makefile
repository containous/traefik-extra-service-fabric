.PHONY: default test dependencies clean build checks

GOFILES = $(shell git ls-files '*.go' | grep -v '^vendor/')

default: clean checks test build

test: clean
	go test -v -cover ./...

dependencies:
	dep ensure -v --vendor-only

clean:
	rm -f cover.out

build:
	go build

checks:
	golangci-lint run

fmt:
	gofmt -s -l -w $(GOFILES)
