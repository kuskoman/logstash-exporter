#!/bin/bash

# Build binary for e2e tests
# This script ensures the binary is always up-to-date

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

BINARY_DIR="test/e2e/bin"
BINARY_NAME="logstash-exporter-e2e"
BINARY_PATH="${BINARY_DIR}/${BINARY_NAME}"

echo -e "${YELLOW}Building e2e test binary...${NC}"

# Create bin directory if it doesn't exist
mkdir -p "${BINARY_DIR}"

# Build the binary
# Set the version information like release builds do, so tests can check it
VERSION_PKG="github.com/kuskoman/logstash-exporter/pkg/config"
LDFLAGS="-X ${VERSION_PKG}.Version=e2e-test -X ${VERSION_PKG}.GitCommit=$(git rev-parse HEAD 2>/dev/null || echo unknown)"

if go build -ldflags="${LDFLAGS}" -o "${BINARY_PATH}" ./cmd/exporter; then
    echo -e "${GREEN}✓ Binary built successfully: ${BINARY_PATH}${NC}"

    # Make it executable
    chmod +x "${BINARY_PATH}"

    # Show binary info
    BINARY_SIZE=$(du -h "${BINARY_PATH}" | cut -f1)
    echo -e "${GREEN}  Size: ${BINARY_SIZE}${NC}"
else
    echo -e "${RED}✗ Failed to build binary${NC}"
    exit 1
fi
