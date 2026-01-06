.PHONY: build build-all clean install test

BINARY_NAME=killport
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION)"

# Build for current platform
build:
	go build $(LDFLAGS) -o $(BINARY_NAME) .

# Build for all platforms
build-all: clean
	mkdir -p dist
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/killport_darwin_amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/killport_darwin_arm64 .
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/killport_linux_amd64 .
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/killport_linux_arm64 .
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/killport_windows_amd64.exe .

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME).exe
	rm -rf dist/

# Install locally
install: build
	sudo mv $(BINARY_NAME) /usr/local/bin/

# Run tests
test:
	go test -v ./...
