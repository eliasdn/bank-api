package fixtures

import (
	"bank-api/internal/models"
)

// TestUsers contains predefined test users
var TestUsers = map[string]models.User{
	"alice": {
		Username:     "alice",
		Email:        "alice@example.com",
		PasswordHash: "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", // password
		FullName:     "Alice Johnson",
	},
	"bob": {
		Username:     "bob",
		Email:        "bob@example.com",
		PasswordHash: "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", // password
		FullName:     "Bob Smith",
	},
}

// TestAccounts contains predefined test accounts
var TestAccounts = map[string]models.Account{
	"alice_checking": {
		UserID:        1,
		AccountNumber: "ACC100001",
		AccountType:   "checking",
		Balance:       1500.00,
	},
	"alice_savings": {
		UserID:        1,
		AccountNumber: "ACC100002",
		AccountType:   "savings",
		Balance:       5000.00,
	},
	"bob_checking": {
		UserID:        2,
		AccountNumber: "ACC200001",
		AccountType:   "checking",
		Balance:       750.00,
	},
}

// TestTransactions contains predefined test transactions
var TestTransactions = map[string]models.Transaction{
	"alice_deposit_1": {
		AccountID:       1,
		Amount:          500.00,
		TransactionType: "deposit",
		Reference:       "Initial deposit",
		Status:          "completed",
	},
	"alice_withdrawal_1": {
		AccountID:       1,
		Amount:          -100.00,
		TransactionType: "withdrawal",
		Reference:       "ATM withdrawal",
		Status:          "completed",
	},
	"bob_deposit_1": {
		AccountID:       3,
		Amount:          250.00,
		TransactionType: "deposit",
		Reference:       "Paycheck deposit",
		Status:          "completed",
	},
}

// GetTestUser returns a test user by key
func GetTestUser(key string) models.User {
	if user, exists := TestUsers[key]; exists {
		return user
	}
	return TestUsers["alice"]
}

// GetTestAccount returns a test account by key
func GetTestAccount(key string) models.Account {
	if account, exists := TestAccounts[key]; exists {
		return account
	}
	return TestAccounts["alice_checking"]
}

// GetTestTransaction returns a test transaction by key
func GetTestTransaction(key string) models.Transaction {
	if transaction, exists := TestTransactions[key]; exists {
		return transaction
	}
	return TestTransactions["alice_deposit_1"]
}
