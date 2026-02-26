.PHONY: build run test lint fmt vet tidy clean

build:
	go build -o barker ./cmd/barker

run:
	go run ./cmd/barker

test:
	go test -race ./...

cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

lint:
	golangci-lint run

fmt:
	gofmt -w .
	goimports -w .

vet:
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -f barker coverage.out
