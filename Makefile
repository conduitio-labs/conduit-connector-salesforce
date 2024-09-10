.PHONY: build test lint fmt proto

VERSION=$(shell git describe --tags --dirty --always)

build:
	go build -ldflags "-X 'github.com/conduitio-labs/conduit-connector-salesforce.version=${VERSION}'" -o conduit-connector-salesforce cmd/connector/main.go

test:
	go test $(GOTEST_FLAGS) -race ./...

fmt:
	gofumpt -l -w .
	gci write --skip-generated  .

lint:
	golangci-lint run -v

proto:
	cd proto && buf generate

generate:
	go generate ./...
	mockery


.PHONY: install-tools
install-tools:
	@echo Installing tools from tools.go
	@go list -e -f '{{ join .Imports "\n" }}' tools.go | xargs -tI % go install %
	@go mod tidy
