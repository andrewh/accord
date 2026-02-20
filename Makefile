VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X github.com/andrewh/accord/internal/cli.Version=$(VERSION)

.PHONY: build test lint clean

build:
	mkdir -p build
	go build -ldflags "$(LDFLAGS)" -o build/accord ./cmd/accord

test:
	go test ./...

lint:
	@test -z "$$(gofmt -s -l .)" || (gofmt -s -l . && exit 1)
	go vet ./...

clean:
	rm -rf build/
