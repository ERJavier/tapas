# TAPAS build and test.
# If you see "no such tool compile" on Apple Silicon, you have amd64 Go but arm64 tools.
# Run once: sudo ln -s darwin_arm64 /usr/local/go/pkg/tool/darwin_amd64

.PHONY: build test run

build:
	go build -o tapas .

test:
	go test ./...

run: build
	./tapas
