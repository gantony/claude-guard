.PHONY: fmt vet test build

fmt:
	gofmt -w .

vet:
	go vet ./...

test:
	go test ./...

build:
	go build -o claude-guard .
