VERSION ?= dev
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  = -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

.PHONY: build run test lint fmt vet tidy clean

build:
	go build -o sznuper -ldflags "$(LDFLAGS)" ./cmd/sznuper

run:
	go run ./cmd/sznuper

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
	rm -f sznuper coverage.out
