#!/bin/bash

# Development Database Setup Script
# This script sets up the development environment for the bank API

set -e

echo "🚀 Setting up development environment for Bank API..."

# Create necessary directories
mkdir -p bin
mkdir -p logs
mkdir -p internal/db

# Install dependencies
echo "📦 Installing dependencies..."
go mod tidy
go mod download

# Run database migrations
echo "🗄️  Setting up database..."
go run cmd/migrate/main.go up

# Create .env file if it doesn't exist
if [ ! -f .env ]; then
    echo "⚙️  Creating .env file..."
    cat > .env << EOF
# Database Configuration
DB_PATH=bank.db

# JWT Configuration
JWT_SECRET=your-secret-key-change-this-in-production

# Server Configuration
PORT=8080

# Environment
ENV=development
EOF
    echo "✅ .env file created. Please review and update as needed."
else
    echo "✅ .env file already exists."
fi

# Make the script executable
chmod +x scripts/setup_dev_db.sh

echo ""
echo "✅ Development environment setup complete!"
echo ""
echo "Next steps:"
echo "1. Review .env file and update configuration as needed"
echo "2. Run 'make build' to build the application"
echo "3. Run './bin/bank-api' to start the server"
echo "4. Visit http://localhost:8080/api/v1/health to check if it's running"