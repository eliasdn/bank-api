package helpers

import (
	"bank-api/internal/config"
	"bank-api/internal/db"
	"bank-api/internal/handlers"
	"bank-api/internal/models"
	"bank-api/internal/testutils"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// SetupTestDB initializes a test database with all required migrations
func SetupTestDB(t *testing.T) (*gorm.DB, func()) {
	gormDB := testutils.InitTestDB()

	// Migrations are now handled by testutils.InitTestDB()
	// which uses the new migration system instead of AutoMigrate

	cleanup := func() {
		sqlDB, _ := gormDB.DB()
		sqlDB.Close()
	}

	return gormDB, cleanup
}

// SetupTestHandler creates a handler with test database
func SetupTestHandler(t *testing.T) (*handlers.Handler, *gorm.DB, func()) {
	gormDB, cleanup := SetupTestDB(t)

	database := &db.Database{DB: gormDB}
	testConfig := config.LoadTestConfig()
	handler := handlers.NewHandler(database, testConfig)

	return handler, gormDB, cleanup
}

// CreateTestUser creates a test user in the database
func CreateTestUser(t *testing.T, db *gorm.DB, username string) *models.User {
	user := &models.User{
		Username:     username,
		Email:        fmt.Sprintf("%s@example.com", username),
		PasswordHash: "hashedpassword",
		FullName:     fmt.Sprintf("Test %s", username),
	}

	result := db.Create(user)
	assert.NoError(t, result.Error, "Failed to create test user")

	return user
}

// CreateTestAccount creates a test account for a user
func CreateTestAccount(t *testing.T, db *gorm.DB, userID uint, balance float64, accountType string) *models.Account {
	if accountType == "" {
		accountType = "checking"
	}

	account := &models.Account{
		UserID:        userID,
		AccountNumber: fmt.Sprintf("ACC%06d", userID*1000+uint(balance)),
		AccountType:   accountType,
		Balance:       balance,
	}

	result := db.Create(account)
	assert.NoError(t, result.Error, "Failed to create test account")

	return account
}

// CreateTestTransaction creates a test transaction for an account
func CreateTestTransaction(t *testing.T, db *gorm.DB, accountID uint, amount float64, transactionType string) *models.Transaction {
	transaction := &models.Transaction{
		AccountID:       accountID,
		Amount:          amount,
		TransactionType: transactionType,
		Reference:       fmt.Sprintf("Test %s", transactionType),
	}

	result := db.Create(transaction)
	assert.NoError(t, result.Error, "Failed to create test transaction")

	return transaction
}

// SetAuthContext sets the user ID in the gin context for authentication
func SetAuthContext(c *gin.Context, userID uint) {
	c.Set("userID", userID)
}

// AssertJSONResponse asserts that the response contains valid JSON
func AssertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) {
	assert.Equal(t, expectedStatus, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
}
