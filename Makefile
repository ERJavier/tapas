# TAPAS build, test, and install.
# If you see "no such tool compile" on Apple Silicon, you have amd64 Go but arm64 tools.
# Run once: sudo ln -s darwin_arm64 /usr/local/go/pkg/tool/darwin_arm64

.PHONY: build test run install uninstall

build:
	go build -o tapas .

test:
	go test ./...

run: build
	./tapas

# Install tapas to $GOBIN (or $GOPATH/bin). After this, run "tapas" from anywhere.
install:
	go install .

# Remove the installed binary (if it lives in GOBIN).
uninstall:
	@rm -f $$(go env GOBIN)/tapas 2>/dev/null || rm -f $$(go env GOPATH)/bin/tapas 2>/dev/null; echo "tapas uninstalled"
