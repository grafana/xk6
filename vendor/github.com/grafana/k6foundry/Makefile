work_dir = $(shell pwd)
GOLANGCI_CONFIG ?= .golangci.yml

all: lint test

## linter-config: Checks if the linter config exists, if not, downloads it from the main k6 repository.
.PHONY: linter-config
linter-config:
	test -s "${GOLANGCI_CONFIG}" || (echo "No linter config, downloading from main k6 repository..." && curl --silent --show-error --fail --no-location https://raw.githubusercontent.com/grafana/k6/master/.golangci.yml --output "${GOLANGCI_CONFIG}")

## check-linter-version: Checks if the linter version is the same as the one specified in the linter config.
.PHONY: check-linter-version
check-linter-version:
	(golangci-lint version | grep "version $(shell head -n 1 .golangci.yml | tr -d '\# ')") || echo "Your installation of golangci-lint is different from the one that is specified in k6's linter config (there it's $(shell head -n 1 .golangci.yml | tr -d '\# ')). Results could be different in the CI."

## lint: Runs the linters.
.PHONY: lint
lint: linter-config check-linter-version
	echo "Running linters..."
	golangci-lint run ./...

.PHONY: test
test:
	go test -race  ./...

