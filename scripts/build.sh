#!/bin/bash

# Build script for bank-api
# This script builds the application for multiple platforms

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get the directory of this script
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# Build configuration
BINARY_NAME="bank-api"
VERSION=${VERSION:-"dev"}
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

echo -e "${GREEN}Building ${BINARY_NAME}...${NC}"

# Create build directory
BUILD_DIR="${PROJECT_ROOT}/build"
mkdir -p "$BUILD_DIR"

# Build flags
LDFLAGS="-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}"

# Build for current platform
echo -e "${YELLOW}Building for current platform...${NC}"
cd "$PROJECT_ROOT"
go build -ldflags "$LDFLAGS" -o "${BUILD_DIR}/${BINARY_NAME}" cmd/server/main.go

# Build for Linux AMD64
echo -e "${YELLOW}Building for Linux AMD64...${NC}"
GOOS=linux GOARCH=amd64 go build -ldflags "$LDFLAGS" -o "${BUILD_DIR}/${BINARY_NAME}-linux-amd64" cmd/server/main.go

# Build for Windows AMD64
echo -e "${YELLOW}Building for Windows AMD64...${NC}"
GOOS=windows GOARCH=amd64 go build -ldflags "$LDFLAGS" -o "${BUILD_DIR}/${BINARY_NAME}-windows-amd64.exe" cmd/server/main.go

# Build for macOS AMD64
echo -e "${YELLOW}Building for macOS AMD64...${NC}"
GOOS=darwin GOARCH=amd64 go build -ldflags "$LDFLAGS" -o "${BUILD_DIR}/${BINARY_NAME}-darwin-amd64" cmd/server/main.go

# Build for macOS ARM64 (Apple Silicon)
echo -e "${YELLOW}Building for macOS ARM64...${NC}"
GOOS=darwin GOARCH=arm64 go build -ldflags "$LDFLAGS" -o "${BUILD_DIR}/${BINARY_NAME}-darwin-arm64" cmd/server/main.go

echo -e "${GREEN}Build completed successfully!${NC}"
echo -e "Binaries available in: ${BUILD_DIR}/"
ls -la "$BUILD_DIR"