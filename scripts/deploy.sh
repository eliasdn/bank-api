#!/bin/bash

# Deployment script for bank-api
# This script handles deployment to different environments

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get the directory of this script
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# Configuration
BINARY_NAME="bank-api"
ENVIRONMENT=${1:-"dev"}
PORT=${PORT:-"8080"}
DB_PATH=${DB_PATH:-"./bank.db"}

echo -e "${GREEN}Deploying ${BINARY_NAME} to ${ENVIRONMENT}...${NC}"

# Check if binary exists
BINARY_PATH="${PROJECT_ROOT}/build/${BINARY_NAME}"
if [ ! -f "$BINARY_PATH" ]; then
    echo -e "${YELLOW}Binary not found, building first...${NC}"
    "$SCRIPT_DIR/build.sh"
fi

# Create necessary directories
mkdir -p "$(dirname "$DB_PATH")"

# Set environment variables
export PORT=$PORT
export DB_PATH=$DB_PATH
export JWT_SECRET=${JWT_SECRET:-"$(openssl rand -base64 32)"}

# Create systemd service file for production
if [ "$ENVIRONMENT" = "prod" ]; then
    echo -e "${YELLOW}Creating systemd service...${NC}"
    cat > "/etc/systemd/system/${BINARY_NAME}.service" << EOF
[Unit]
Description=Bank API Service
After=network.target

[Service]
Type=simple
User=bank-api
WorkingDirectory=/opt/bank-api
ExecStart=/opt/bank-api/${BINARY_NAME}
Restart=always
RestartSec=5
Environment=PORT=8080
Environment=DB_PATH=/opt/bank-api/bank.db
Environment=JWT_SECRET=${JWT_SECRET}

[Install]
WantedBy=multi-user.target
EOF

    echo -e "${YELLOW}Setting up directories and permissions...${NC}"
    sudo mkdir -p /opt/bank-api
    sudo cp "$BINARY_PATH" /opt/bank-api/
    sudo useradd -r -s /bin/false bank-api || true
    sudo chown -R bank-api:bank-api /opt/bank-api
    sudo chmod +x /opt/bank-api/${BINARY_NAME}

    echo -e "${YELLOW}Starting service...${NC}"
    sudo systemctl daemon-reload
    sudo systemctl enable ${BINARY_NAME}
    sudo systemctl start ${BINARY_NAME}
    
    echo -e "${GREEN}Service deployed and started!${NC}"
    echo -e "Check status: sudo systemctl status ${BINARY_NAME}"
else
    # Development deployment
    echo -e "${YELLOW}Starting in development mode...${NC}"
    echo -e "Environment variables:"
    echo -e "  PORT: ${PORT}"
    echo -e "  DB_PATH: ${DB_PATH}"
    echo -e "  JWT_SECRET: [hidden]"
    
    cd "$PROJECT_ROOT"
    exec "$BINARY_PATH"
fi