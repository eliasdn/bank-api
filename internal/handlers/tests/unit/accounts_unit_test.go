package unit

import (
	"bank-api/internal/config"
	"bank-api/internal/db"
	"bank-api/internal/handlers"
	"bank-api/internal/models"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type AccountsUnitTestSuite struct {
	suite.Suite
	handler *handlers.Handler
	db      *gorm.DB
	router  *gin.Engine
}

func (suite *AccountsUnitTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
	var err error
	suite.db, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	// Migrate the schema
	suite.db.AutoMigrate(&models.Account{})

	// Create a handler with the real database
	testConfig := config.LoadTestConfig()
	suite.handler = handlers.NewHandler(&db.Database{DB: suite.db}, testConfig)
	suite.router = gin.New()

	// Register test routes with auth middleware
	suite.router.GET("/accounts", func(c *gin.Context) {
		c.Set("userID", uint(1))
		suite.handler.GetAccounts(c)
	})
	suite.router.POST("/accounts", func(c *gin.Context) {
		c.Set("userID", uint(1))
		suite.handler.CreateAccount(c)
	})
	suite.router.GET("/accounts/:id", func(c *gin.Context) {
		c.Set("userID", uint(1))
		suite.handler.GetAccount(c)
	})
}

func (suite *AccountsUnitTestSuite) setUserID(req *http.Request, userID uint) *http.Request {
	ctx := req.Context()
	ctx = context.WithValue(ctx, "userID", userID)
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	return req
}

func (suite *AccountsUnitTestSuite) TestGetAccounts_Success() {
	// Create test accounts
	suite.db.Create(&models.Account{UserID: 1, AccountNumber: "ACC001", AccountType: "checking", Balance: 1000})
	suite.db.Create(&models.Account{UserID: 1, AccountNumber: "ACC002", AccountType: "savings", Balance: 2000})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/accounts", nil)
	req = suite.setUserID(req, 1)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response struct {
		Data []models.Account `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), response.Data, 2)
}

func (suite *AccountsUnitTestSuite) TestCreateAccount_Success() {
	accountJSON := `{"account_number":"ACC003","account_type":"checking","balance":1000}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/accounts", strings.NewReader(accountJSON))
	req.Header.Set("Content-Type", "application/json")
	req = suite.setUserID(req, 1)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	// Verify the account was created
	var account models.Account
	suite.db.First(&account, "user_id = ?", 1)
	assert.NotEmpty(suite.T(), account.AccountNumber)
	assert.Equal(suite.T(), "checking", account.AccountType)
	assert.Equal(suite.T(), float64(1000), account.Balance)
}

func (suite *AccountsUnitTestSuite) TestGetAccount_Success() {
	// Create a test account
	suite.db.Create(&models.Account{Model: gorm.Model{ID: 1}, UserID: 1, AccountNumber: "ACC004", AccountType: "checking", Balance: 1000})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/accounts/1", nil)
	req = suite.setUserID(req, 1)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response struct {
		Data models.Account `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), uint(1), response.Data.ID)
	assert.Equal(suite.T(), float64(1000), response.Data.Balance)
}

func (suite *AccountsUnitTestSuite) TestGetAccount_NotFound() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/accounts/999", nil)
	req = suite.setUserID(req, 1)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
}

func TestAccountsUnitSuite(t *testing.T) {
	suite.Run(t, new(AccountsUnitTestSuite))
}
