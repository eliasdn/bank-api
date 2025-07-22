package models

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username     string    `gorm:"unique;not null"`
	Email        string    `gorm:"unique;not null"`
	PasswordHash string    `gorm:"not null"`
	FullName     string    `gorm:"not null"`
	Accounts     []Account `gorm:"foreignKey:UserID"`
}

// SetPassword hashes and sets the user's password
func (u *User) SetPassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	u.PasswordHash = string(hashedPassword)
	return nil
}

// CheckPassword verifies if the provided password matches the stored hash
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

type Account struct {
	gorm.Model
	UserID        uint          `gorm:"not null"`
	AccountNumber string        `gorm:"unique;not null"`
	AccountType   string        `gorm:"not null;check:account_type IN ('checking', 'savings', 'credit')"`
	Balance       float64       `gorm:"not null;default:0"`
	Transactions  []Transaction `gorm:"foreignKey:AccountID"`
}

type Transaction struct {
	gorm.Model
	AccountID       uint    `gorm:"not null"`
	Amount          float64 `gorm:"not null"`
	TransactionType string  `gorm:"not null;check:transaction_type IN ('deposit', 'withdrawal', 'transfer')"`
	Reference       string
	Status          string `gorm:"not null;check:status IN ('pending', 'completed', 'failed');default:'pending'"`
}
