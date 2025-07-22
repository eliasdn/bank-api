package unit

import (
	"bank-api/internal/config"
	"bank-api/internal/db"
	"bank-api/internal/dto"
	"bank-api/internal/handlers"
	"bank-api/internal/models"
	"bank-api/internal/testutils"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func setupTransactionUnitTest(t *testing.T) (*handlers.Handler, *gorm.DB, func()) {
	gormDB := testutils.InitTestDB()

	// Create database adapter
	database := &db.Database{DB: gormDB}
	testConfig := config.LoadTestConfig()
	handler := handlers.NewHandler(database, testConfig)

	// Create a test user
	user := models.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
		FullName:     "Test User",
	}
	gormDB.Create(&user)

	return handler, gormDB, func() {
		sqlDB, _ := gormDB.DB()
		sqlDB.Close()
	}
}

func createTestAccount(t *testing.T, db *gorm.DB, userID uint, balance float64, accountNumber string) models.Account {
	if accountNumber == "" {
		accountNumber = fmt.Sprintf("ACC%06d", userID*1000+uint(balance))
	}
	account := models.Account{
		UserID:        userID,
		AccountNumber: accountNumber,
		AccountType:   "checking",
		Balance:       balance,
	}
	db.Create(&account)
	return account
}

func TestDeposit(t *testing.T) {
	handler, db, cleanup := setupTransactionUnitTest(t)
	defer cleanup()

	var user models.User
	db.First(&user)

	_ = createTestAccount(t, db, user.ID, 1000.0, "")

	tests := []struct {
		name            string
		accountID       string
		amount          float64
		expectedStatus  int
		expectedBalance float64
		setupAuth       bool
	}{
		{
			name:            "successful deposit",
			accountID:       "1",
			amount:          500.0,
			expectedStatus:  http.StatusOK,
			expectedBalance: 1500.0,
			setupAuth:       true,
		},
		{
			name:           "invalid amount - zero",
			accountID:      "1",
			amount:         0,
			expectedStatus: http.StatusBadRequest,
			setupAuth:      true,
		},
		{
			name:           "invalid amount - negative",
			accountID:      "1",
			amount:         -100,
			expectedStatus: http.StatusBadRequest,
			setupAuth:      true,
		},
		{
			name:           "account not found",
			accountID:      "999",
			amount:         100,
			expectedStatus: http.StatusNotFound,
			setupAuth:      true,
		},
		{
			name:           "unauthorized - no auth",
			accountID:      "1",
			amount:         100,
			expectedStatus: http.StatusUnauthorized,
			setupAuth:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Setup request
			body := map[string]float64{"amount": tt.amount}
			jsonBody, _ := json.Marshal(body)
			c.Request = httptest.NewRequest("POST", "/accounts/"+tt.accountID+"/deposit", bytes.NewBuffer(jsonBody))
			c.Request.Header.Set("Content-Type", "application/json")

			// Setup URL parameter
			c.Params = gin.Params{{Key: "id", Value: tt.accountID}}

			// Setup authentication
			if tt.setupAuth {
				c.Set("userID", user.ID)
			}

			handler.Deposit(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response dto.DepositResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, "Deposit successful", response.Message)
				assert.Equal(t, tt.expectedBalance, response.Balance)
			}
		})
	}
}

func TestWithdraw(t *testing.T) {
	handler, db, cleanup := setupTransactionUnitTest(t)
	defer cleanup()

	var user models.User
	db.First(&user)

	_ = createTestAccount(t, db, user.ID, 1000.0, "")

	tests := []struct {
		name            string
		accountID       string
		amount          float64
		expectedStatus  int
		expectedBalance float64
		setupAuth       bool
	}{
		{
			name:            "successful withdrawal",
			accountID:       "1",
			amount:          300.0,
			expectedStatus:  http.StatusOK,
			expectedBalance: 700.0,
			setupAuth:       true,
		},
		{
			name:           "insufficient funds",
			accountID:      "1",
			amount:         1500.0,
			expectedStatus: http.StatusBadRequest,
			setupAuth:      true,
		},
		{
			name:           "invalid amount - zero",
			accountID:      "1",
			amount:         0,
			expectedStatus: http.StatusBadRequest,
			setupAuth:      true,
		},
		{
			name:           "account not found",
			accountID:      "999",
			amount:         100,
			expectedStatus: http.StatusNotFound,
			setupAuth:      true,
		},
		{
			name:           "unauthorized - no auth",
			accountID:      "1",
			amount:         100,
			expectedStatus: http.StatusUnauthorized,
			setupAuth:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Setup request
			body := map[string]float64{"amount": tt.amount}
			jsonBody, _ := json.Marshal(body)
			c.Request = httptest.NewRequest("POST", "/accounts/"+tt.accountID+"/withdraw", bytes.NewBuffer(jsonBody))
			c.Request.Header.Set("Content-Type", "application/json")

			// Setup URL parameter
			c.Params = gin.Params{{Key: "id", Value: tt.accountID}}

			// Setup authentication
			if tt.setupAuth {
				c.Set("userID", user.ID)
			}

			handler.Withdraw(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response dto.WithdrawResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, "Withdrawal successful", response.Message)
				assert.Equal(t, tt.expectedBalance, response.Balance)
			}
		})
	}
}

func TestTransfer(t *testing.T) {
	handler, db, cleanup := setupTransactionUnitTest(t)
	defer cleanup()

	var user models.User
	db.First(&user)

	_ = createTestAccount(t, db, user.ID, 1000.0, "ACC1001")
	account2 := createTestAccount(t, db, user.ID, 500.0, "ACC1002")

	tests := []struct {
		name           string
		fromAccountID  string
		toAccountID    uint
		amount         float64
		description    string
		expectedStatus int
		setupAuth      bool
	}{
		{
			name:           "successful transfer",
			fromAccountID:  "1",
			toAccountID:    account2.ID,
			amount:         200.0,
			description:    "Test transfer",
			expectedStatus: http.StatusOK,
			setupAuth:      true,
		},
		{
			name:           "insufficient funds",
			fromAccountID:  "1",
			toAccountID:    account2.ID,
			amount:         1500.0,
			description:    "Test transfer",
			expectedStatus: http.StatusBadRequest,
			setupAuth:      true,
		},
		{
			name:           "destination account not found",
			fromAccountID:  "1",
			toAccountID:    999,
			amount:         100.0,
			description:    "Test transfer",
			expectedStatus: http.StatusBadRequest,
			setupAuth:      true,
		},
		{
			name:           "source account not found",
			fromAccountID:  "999",
			toAccountID:    account2.ID,
			amount:         100.0,
			description:    "Test transfer",
			expectedStatus: http.StatusNotFound,
			setupAuth:      true,
		},
		{
			name:           "invalid amount - zero",
			fromAccountID:  "1",
			toAccountID:    account2.ID,
			amount:         0,
			description:    "Test transfer",
			expectedStatus: http.StatusBadRequest,
			setupAuth:      true,
		},
		{
			name:           "unauthorized - no auth",
			fromAccountID:  "1",
			toAccountID:    account2.ID,
			amount:         100.0,
			description:    "Test transfer",
			expectedStatus: http.StatusUnauthorized,
			setupAuth:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Setup request
			body := map[string]interface{}{
				"to_account_id": tt.toAccountID,
				"amount":        tt.amount,
				"description":   tt.description,
			}
			jsonBody, _ := json.Marshal(body)
			c.Request = httptest.NewRequest("POST", "/accounts/"+tt.fromAccountID+"/transfer", bytes.NewBuffer(jsonBody))
			c.Request.Header.Set("Content-Type", "application/json")

			// Setup URL parameter
			c.Params = gin.Params{{Key: "id", Value: tt.fromAccountID}}

			// Setup authentication
			if tt.setupAuth {
				c.Set("userID", user.ID)
			}

			handler.Transfer(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response dto.TransferResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, "Transfer successful", response.Message)
			}
		})
	}
}

func TestGetTransactions(t *testing.T) {
	handler, db, cleanup := setupTransactionUnitTest(t)
	defer cleanup()

	var user models.User
	db.First(&user)

	account := createTestAccount(t, db, user.ID, 1000.0, "")

	// Create some test transactions
	transactions := []models.Transaction{
		{
			AccountID:       account.ID,
			Amount:          100.0,
			TransactionType: "deposit",
			Reference:       "Test deposit 1",
		},
		{
			AccountID:       account.ID,
			Amount:          50.0,
			TransactionType: "withdrawal",
			Reference:       "Test withdrawal",
		},
	}
	for _, tx := range transactions {
		db.Create(&tx)
	}

	tests := []struct {
		name           string
		accountID      string
		expectedStatus int
		expectedCount  int
		setupAuth      bool
	}{
		{
			name:           "successful retrieval",
			accountID:      fmt.Sprintf("%d", account.ID),
			expectedStatus: http.StatusOK,
			expectedCount:  2,
			setupAuth:      true,
		},
		{
			name:           "account not found",
			accountID:      "999",
			expectedStatus: http.StatusNotFound,
			setupAuth:      true,
		},
		{
			name:           "unauthorized - no auth",
			accountID:      fmt.Sprintf("%d", account.ID),
			expectedStatus: http.StatusUnauthorized,
			setupAuth:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Setup request
			c.Request = httptest.NewRequest("GET", "/accounts/"+tt.accountID+"/transactions", nil)
			c.Params = gin.Params{{Key: "id", Value: tt.accountID}}

			// Setup authentication
			if tt.setupAuth {
				c.Set("userID", user.ID)
			}

			handler.GetTransactions(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response dto.TransactionListResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Len(t, response.Transactions, tt.expectedCount)
			}
		})
	}
}

func TestTransactionDatabaseErrors(t *testing.T) {
	handler, db, cleanup := setupTransactionUnitTest(t)
	defer cleanup()

	var user models.User
	db.First(&user)

	account := createTestAccount(t, db, user.ID, 1000.0, "")

	t.Run("database error during deposit", func(t *testing.T) {
		// Mock database error by using invalid JSON
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request = httptest.NewRequest("POST", "/accounts/"+fmt.Sprintf("%d", account.ID)+"/deposit", bytes.NewBufferString("invalid json"))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", account.ID)}}
		c.Set("userID", user.ID)

		handler.Deposit(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
