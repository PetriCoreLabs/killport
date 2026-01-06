#!/bin/bash

# killport installer script
# Usage: curl -sSL https://raw.githubusercontent.com/PetriCoreLabs/killport/main/install.sh | bash

set -e

REPO="PetriCoreLabs/killport"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="killport"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        echo "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

case $OS in
    darwin)
        OS="darwin"
        ;;
    linux)
        OS="linux"
        ;;
    mingw*|msys*|cygwin*)
        OS="windows"
        BINARY_NAME="killport.exe"
        ;;
    *)
        echo "Unsupported OS: $OS"
        exit 1
        ;;
esac

echo "Detected: ${OS}/${ARCH}"

# Get latest release tag
LATEST_RELEASE=$(curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_RELEASE" ]; then
    echo "Could not determine latest release. Installing from source..."
    
    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        echo "Go is not installed. Please install Go first: https://golang.org/dl/"
        exit 1
    fi
    
    go install "github.com/${REPO}@latest"
    echo "killport installed successfully via go install!"
    echo "Make sure $(go env GOPATH)/bin is in your PATH"
    exit 0
fi

echo "Latest release: ${LATEST_RELEASE}"

# Download URL
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST_RELEASE}/killport_${OS}_${ARCH}"
if [ "$OS" = "windows" ]; then
    DOWNLOAD_URL="${DOWNLOAD_URL}.exe"
fi

echo "Downloading from: ${DOWNLOAD_URL}"

# Create temp directory
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

# Download binary
curl -sSL "$DOWNLOAD_URL" -o "${TMP_DIR}/${BINARY_NAME}"

# Make executable
chmod +x "${TMP_DIR}/${BINARY_NAME}"

# Install
if [ -w "$INSTALL_DIR" ]; then
    mv "${TMP_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
else
    echo "Need sudo to install to ${INSTALL_DIR}"
    sudo mv "${TMP_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
fi

echo ""
echo "killport ${LATEST_RELEASE} installed successfully!"
echo "Run 'killport --help' to get started."
