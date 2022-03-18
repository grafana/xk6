MAKEFLAGS += --silent

all: clean format test build

## help: Prints a list of available build targets.
help:
	echo "Usage: make <OPTIONS> ... <TARGETS>"
	echo ""
	echo "Available targets are:"
	echo ''
	sed -n 's/^##//p' ${PWD}/Makefile | column -t -s ':' | sed -e 's/^/ /'
	echo
	echo "Targets run by default are: `sed -n 's/^all: //p' ./Makefile | sed -e 's/ /, /g' | sed -e 's/\(.*\), /\1, and /'`"

## clean: Removes any previously created build artifacts.
clean:
	rm -f ./xk6
	rm -f ./k6

## build: Builds the 'xk6' binary.
build:
	go build -work ./cmd/xk6

## format: Applies Go formatting to code.
format:
	go fmt ./...

## test: Executes any unit tests.
test:
	go test -cover -race ./...

## vendor: Pulls source for external dependencies.
vendor:
	go mod vendor

.PHONY: build clean format help test vendor
