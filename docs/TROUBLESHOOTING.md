# Bank API Troubleshooting Guide

This guide provides solutions for common issues encountered during development, testing, and deployment of the Bank API.

## Table of Contents
- [Database Issues](#database-issues)
- [Migration Problems](#migration-problems)
- [Build and Deployment Issues](#build-and-deployment-issues)
- [Testing Issues](#testing-issues)
- [Runtime Issues](#runtime-issues)
- [Environment Setup](#environment-setup)

## Database Issues

### SQLite Database Locked
**Symptoms**: `database is locked` errors during concurrent operations.

**Solutions**:
1. Ensure proper connection pooling (SQLite handles this automatically)
2. Check for long-running transactions
3. Use WAL mode for better concurrency:
   ```sql
   PRAGMA journal_mode=WAL;
   ```

### Database Path Issues
**Symptoms**: Database file not found or created in wrong location.

**Solutions**:
1. Use absolute paths in configuration
2. Check working directory when running binary:
   ```bash
   # Check current working directory
   pwd
   
   # Use absolute path for DB
   export DB_PATH=/absolute/path/to/bank.db
   ```

### Database Corruption
**Symptoms**: Database errors, missing tables, or data integrity issues.

**Solutions**:
1. Backup database before migrations:
   ```bash
   cp bank.db bank.db.backup
   ```
2. Reset database:
   ```bash
   make migrate-reset
   ```
3. For complete reset:
   ```bash
   make clean
   make dev-setup
   ```

## Migration Problems

### Migration Version Conflicts
**Symptoms**: Duplicate migration files or version conflicts.

**Solutions**:
1. Check migration file naming:
   - Format: `0001_description.up.sql` and `0001_description.down.sql`
   - Ensure sequential numbering
2. List all migrations:
   ```bash
   ls -la internal/db/migrations/
   ```
3. Reset migrations:
   ```bash
   make migrate-reset
   ```

### Migration Failures
**Symptoms**: Migration fails with SQL errors.

**Solutions**:
1. Check migration syntax:
   ```bash
   # Test migration syntax
   sqlite3 bank.db < internal/db/migrations/0001_create_users_table.up.sql
   ```
2. Check database state:
   ```bash
   sqlite3 bank.db ".schema"
   ```
3. Manual rollback:
   ```bash
   make migrate-down
   ```

### Missing Migrations Directory
**Symptoms**: `no such file or directory` for migrations.

**Solutions**:
1. Ensure migrations directory exists:
   ```bash
   mkdir -p internal/db/migrations
   ```
2. Check file permissions:
   ```bash
   chmod -R 755 internal/db/migrations/
   ```

## Build and Deployment Issues

### Binary Build Failures
**Symptoms**: Build fails with compilation errors.

**Solutions**:
1. Check Go version:
   ```bash
   go version
   ```
2. Clean and rebuild:
   ```bash
   make clean
   make build
   ```
3. Check dependencies:
   ```bash
   go mod tidy
   go mod verify
   ```

### Cross-Compilation Issues
**Symptoms**: Build fails for specific platforms.

**Solutions**:
1. Use provided build scripts:
   ```bash
   # Linux/macOS
   chmod +x scripts/build.sh
   ./scripts/build.sh
   
   # Windows
   scripts\build.bat
   ```
2. Manual cross-compilation:
   ```bash
   # Linux
   GOOS=linux GOARCH=amd64 go build -o bin/bank-api-linux cmd/server/main.go
   
   # Windows
   GOOS=windows GOARCH=amd64 go build -o bin/bank-api.exe cmd/server/main.go
   ```

### Missing Build Dependencies
**Symptoms**: Build fails with missing packages.

**Solutions**:
1. Install dependencies:
   ```bash
   make deps
   ```
2. Clear module cache:
   ```bash
   go clean -modcache
   go mod download
   ```

## Testing Issues

### Test Database Issues
**Symptoms**: Tests fail with database connection errors.

**Solutions**:
1. Ensure test database is clean:
   ```bash
   rm -f *.test.db
   ```
2. Run tests with verbose output:
   ```bash
   go test -v ./...
   ```
3. Run specific test suites:
   ```bash
   make test-unit
   make test-integration
   ```

### Test Data Issues
**Symptoms**: Tests fail due to missing or incorrect test data.

**Solutions**:
1. Check test fixtures:
   ```bash
   ls -la internal/handlers/tests/fixtures/
   ```
2. Reset test database:
   ```bash
   go test -v ./internal/testutils/...
   ```

### Concurrent Test Failures
**Symptoms**: Tests pass individually but fail when run together.

**Solutions**:
1. Use unique database names for each test:
   ```go
   // In test setup
   dbName := fmt.Sprintf("test_%d.db", time.Now().UnixNano())
   ```
2. Ensure proper cleanup:
   ```go
   defer os.Remove(dbName)
   ```

## Runtime Issues

### Port Already in Use
**Symptoms**: `bind: address already in use` error.

**Solutions**:
1. Find process using port:
   ```bash
   # Linux/macOS
   lsof -i :8080
   
   # Windows
   netstat -ano | findstr :8080
   ```
2. Use different port:
   ```bash
   export PORT=8081
   ./bin/bank-api
   ```

### JWT Secret Issues
**Symptoms**: Authentication failures or JWT validation errors.

**Solutions**:
1. Set JWT secret:
   ```bash
   export JWT_SECRET="your-secret-key-here"
   ```
2. Generate secure secret:
   ```bash
   openssl rand -base64 32
   ```

### Memory Issues
**Symptoms**: High memory usage or OOM errors.

**Solutions**:
1. Monitor memory usage:
   ```bash
   # Linux
   top -p $(pgrep bank-api)
   
   # macOS
   top -pid $(pgrep bank-api)
   ```
2. Check for memory leaks in logs
3. Use connection pooling limits

## Environment Setup

### Go Environment Issues
**Symptoms**: Go commands not found or wrong version.

**Solutions**:
1. Check Go installation:
   ```bash
   which go
   go env
   ```
2. Set GOPATH and GOROOT:
   ```bash
   export GOPATH=$HOME/go
   export PATH=$PATH:$GOPATH/bin
   ```

### SQLite Driver Issues
**Symptoms**: Database driver compilation errors.

**Solutions**:
1. Install build tools:
   ```bash
   # Ubuntu/Debian
   sudo apt-get install gcc
   
   # macOS
   xcode-select --install
   
   # Windows
   # Install MinGW or Visual Studio Build Tools
   ```
2. Check CGO settings:
   ```bash
   go env CGO_ENABLED
   ```

### Permission Issues
**Symptoms**: Permission denied errors.

**Solutions**:
1. Check file permissions:
   ```bash
   ls -la bank.db
   chmod 644 bank.db
   ```
2. Check directory permissions:
   ```bash
   chmod 755 .
   ```

## Debugging Commands

### Database Inspection
```bash
# Connect to database
sqlite3 bank.db

# List tables
.tables

# Check schema
.schema users

# Check data
SELECT * FROM users LIMIT 5;
```

### Application Logs
```bash
# Run with debug logging
export LOG_LEVEL=debug
./bin/bank-api

# Check system logs (Linux)
journalctl -u bank-api -f
```

### Health Check
```bash
# Check if service is running
curl http://localhost:8080/api/v1/health

# Check metrics
curl http://localhost:8080/metrics
```

## Common Error Messages

### "database is locked"
- **Cause**: Concurrent write operations
- **Solution**: Use WAL mode or reduce concurrency

### "no such table"
- **Cause**: Migrations not run or database corruption
- **Solution**: Run migrations: `make migrate-up`

### "invalid memory address"
- **Cause**: Null pointer dereference
- **Solution**: Check for nil values in database queries

### "bind: address already in use"
- **Cause**: Port already in use
- **Solution**: Change port or kill existing process

## Getting Help

1. Check logs for detailed error messages
2. Run with debug mode enabled
3. Check GitHub issues for similar problems
4. Create minimal reproduction case
5. Include environment details when reporting issues

## Quick Fixes

### Reset Everything
```bash
make clean
make dev-setup
```

### Fresh Build
```bash
make clean
make build-all
```

### Test Everything
```bash
make test
make build
```

### Development Setup
```bash
make dev-setup
make deploy