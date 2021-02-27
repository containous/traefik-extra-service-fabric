.PHONY: default test dependencies clean build checks

default: clean checks test build

test: clean
	go test -v -cover ./...

clean:
	rm -f cover.out

build:
	go build

checks:
	golangci-lint run
