package testutils

import (
	"bank-api/internal/audit"
	"bank-api/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// InitTestDB initializes an in-memory database for testing using AutoMigrate
func InitTestDB() *gorm.DB {
	gormDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to create in-memory database")
	}

	// Use AutoMigrate for testing - simpler and more reliable for in-memory DB
	if err := gormDB.AutoMigrate(
		&models.User{},
		&models.Account{},
		&models.Transaction{},
		&audit.AuditLog{},
	); err != nil {
		panic("failed to auto migrate: " + err.Error())
	}

	return gormDB
}

// InitTestDBWithMigrations initializes a test database with migrations
// Note: For in-memory testing, AutoMigrate is more reliable
func InitTestDBWithMigrations() *gorm.DB {
	return InitTestDB()
}

// InitTestDBWithAutoMigrate initializes a test database using AutoMigrate (legacy)
func InitTestDBWithAutoMigrate() *gorm.DB {
	return InitTestDB()
}
