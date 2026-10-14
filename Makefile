.PHONY: build test lint

build:
	go build -o bin/ ./cmd/...

test:
	go test -race ./...

lint:
	go vet $$(go list ./... | grep -v /examples/)
