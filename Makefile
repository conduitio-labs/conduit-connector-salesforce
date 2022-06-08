.PHONY: build test lint

build:
	go build -o conduit-connector-salesforce cmd/sf/main.go

test:
	go test $(GOTEST_FLAGS) -race ./...

lint:
	golangci-lint run
