package testutils

import (
	"gorm.io/gorm"
)

// CleanupTestDB cleans up all data from the test database
func CleanupTestDB(db *gorm.DB) error {
	// Delete in reverse order of dependencies to avoid foreign key constraints
	tables := []string{
		"audit_logs",
		"transactions",
		"accounts",
		"users",
	}

	for _, table := range tables {
		if err := db.Exec("DELETE FROM " + table).Error; err != nil {
			return err
		}
	}

	// Reset auto-increment counters
	for _, table := range tables {
		db.Exec("DELETE FROM sqlite_sequence WHERE name = ?", table)
	}

	return nil
}

// TruncateTables truncates specific tables
func TruncateTables(db *gorm.DB, tables ...string) error {
	for _, table := range tables {
		if err := db.Exec("DELETE FROM " + table).Error; err != nil {
			return err
		}
		db.Exec("DELETE FROM sqlite_sequence WHERE name = ?", table)
	}
	return nil
}
