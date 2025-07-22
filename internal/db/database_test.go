package db

import (
	"bank-api/internal/models"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DatabaseTestSuite struct {
	suite.Suite
	db *Database
}

func (suite *DatabaseTestSuite) SetupTest() {
	// Remove existing test database
	os.Remove("test.db")

	// Initialize test database
	var err error
	suite.db, err = InitTestDB()
	assert.NoError(suite.T(), err)
}

func (suite *DatabaseTestSuite) TearDownTest() {
	// Close database connection
	if suite.db != nil {
		suite.db.Close()
	}
	os.Remove("test.db")
}

func InitTestDB() (*Database, error) {
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	// Auto migrate all models
	err = db.AutoMigrate(
		&models.User{},
		&models.Account{},
		&models.Transaction{},
	)
	if err != nil {
		return nil, err
	}

	return &Database{DB: db}, nil
}

func TestDatabaseSuite(t *testing.T) {
	suite.Run(t, new(DatabaseTestSuite))
}

func (suite *DatabaseTestSuite) TestUserCRUD() {
	// Create user
	user := &models.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
		FullName:     "Test User",
	}
	err := suite.db.CreateUser(user)
	assert.NoError(suite.T(), err)
	assert.NotZero(suite.T(), user.ID)

	// Get user by ID
	foundUser, err := suite.db.GetUserByID(user.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), user.Username, foundUser.Username)

	// Get user by username
	foundUser, err = suite.db.GetUserByUsername(user.Username)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), user.Email, foundUser.Email)

	// Update user
	user.FullName = "Updated User"
	err = suite.db.UpdateUser(user)
	assert.NoError(suite.T(), err)

	// Verify update
	updatedUser, err := suite.db.GetUserByID(user.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Updated User", updatedUser.FullName)

	// Delete user
	err = suite.db.DeleteUser(user.ID)
	assert.NoError(suite.T(), err)

	// Verify deletion
	_, err = suite.db.GetUserByID(user.ID)
	assert.ErrorIs(suite.T(), err, gorm.ErrRecordNotFound)
}

func (suite *DatabaseTestSuite) TestAccountCRUD() {
	// Create user first
	user := &models.User{
		Username:     "accountuser",
		Email:        "account@example.com",
		PasswordHash: "hashedpassword",
	}
	err := suite.db.CreateUser(user)
	assert.NoError(suite.T(), err)

	// Create account
	account := &models.Account{
		UserID:      user.ID,
		AccountType: "checking",
		Balance:     1000.00,
	}
	err = suite.db.CreateAccount(account)
	assert.NoError(suite.T(), err)
	assert.NotZero(suite.T(), account.ID)

	// Get account by ID
	foundAccount, err := suite.db.GetAccountByID(account.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), account.Balance, foundAccount.Balance)

	// Get accounts by user ID
	accounts, err := suite.db.GetAccountsByUserID(user.ID)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), accounts, 1)

	// Update account
	newBalance := 1500.00
	account.Balance = newBalance
	err = suite.db.UpdateAccount(account)
	assert.NoError(suite.T(), err)

	// Verify update
	updatedAccount, err := suite.db.GetAccountByID(account.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), newBalance, updatedAccount.Balance)

	// Delete account
	err = suite.db.DeleteAccount(account.ID)
	assert.NoError(suite.T(), err)

	// Verify deletion
	_, err = suite.db.GetAccountByID(account.ID)
	assert.ErrorIs(suite.T(), err, gorm.ErrRecordNotFound)
}

func (suite *DatabaseTestSuite) TestTransactionCRUD() {
	// Create user and account first
	user := &models.User{
		Username:     "txuser",
		Email:        "tx@example.com",
		PasswordHash: "hashedpassword",
	}
	err := suite.db.CreateUser(user)
	assert.NoError(suite.T(), err)

	account := &models.Account{
		UserID:      user.ID,
		AccountType: "savings",
		Balance:     2000.00,
	}
	err = suite.db.CreateAccount(account)
	assert.NoError(suite.T(), err)

	// Create transaction
	tx := &models.Transaction{
		AccountID:       account.ID,
		Amount:          500.00,
		TransactionType: "deposit",
		Reference:       "Test deposit",
	}
	err = suite.db.CreateTransaction(tx)
	assert.NoError(suite.T(), err)
	assert.NotZero(suite.T(), tx.ID)

	// Get transaction by ID
	foundTx, err := suite.db.GetTransactionByID(tx.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), tx.Amount, foundTx.Amount)

	// Get transactions by account ID
	transactions, err := suite.db.GetTransactionsByAccountID(account.ID)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), transactions, 1)

	// Get transactions by user ID
	userTransactions, err := suite.db.GetTransactionsByUserID(user.ID)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), userTransactions, 1)

	// Update transaction
	tx.Reference = "Updated description"
	err = suite.db.UpdateTransaction(tx)
	assert.NoError(suite.T(), err)

	// Verify update
	updatedTx, err := suite.db.GetTransactionByID(tx.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Updated description", updatedTx.Reference)

	// Delete transaction
	err = suite.db.DeleteTransaction(tx.ID)
	assert.NoError(suite.T(), err)

	// Verify deletion
	_, err = suite.db.GetTransactionByID(tx.ID)
	assert.ErrorIs(suite.T(), err, gorm.ErrRecordNotFound)
}

func (suite *DatabaseTestSuite) TestAccountBalanceUpdate() {
	user := &models.User{
		Username:     "balanceuser",
		Email:        "balance@example.com",
		PasswordHash: "hashedpassword",
	}
	err := suite.db.CreateUser(user)
	assert.NoError(suite.T(), err)

	account := &models.Account{
		UserID:      user.ID,
		AccountType: "checking",
		Balance:     1000.00,
	}
	err = suite.db.CreateAccount(account)
	assert.NoError(suite.T(), err)

	// Test balance update
	err = suite.db.UpdateAccountBalance(account.ID, 500.00, nil)
	assert.NoError(suite.T(), err)

	updatedAccount, err := suite.db.GetAccountByID(account.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 1500.00, updatedAccount.Balance)

	// Test balance update in transaction
	tx := suite.db.Begin()
	err = suite.db.UpdateAccountBalance(account.ID, -200.00, tx)
	assert.NoError(suite.T(), err)
	tx.Commit()

	updatedAccount, err = suite.db.GetAccountByID(account.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 1300.00, updatedAccount.Balance)
}

func (suite *DatabaseTestSuite) TestDatabaseHealth() {
	err := suite.db.Health()
	assert.NoError(suite.T(), err)
}
