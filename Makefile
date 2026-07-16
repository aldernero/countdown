BINARY := countdown

.PHONY: all build test lint tidy snapshot clean

all: test lint build

build:
	go build -o $(BINARY) .

test:
	go test -race ./...

lint:
	golangci-lint run

tidy:
	go mod tidy

## snapshot: build a local release snapshot without publishing
snapshot:
	goreleaser release --snapshot --clean

clean:
	rm -rf $(BINARY) dist
