package models

import (
	"net/http"
	"time"

	"bank-api/internal/errors"

	"gorm.io/gorm"
)

// TransactionLimit represents transaction limits for an account
type TransactionLimit struct {
	ID              uint      `gorm:"primaryKey"`
	AccountID       uint      `gorm:"not null;uniqueIndex"`
	DailyLimit      float64   `gorm:"not null;default:10000"`  // Daily transaction limit
	MonthlyLimit    float64   `gorm:"not null;default:100000"` // Monthly transaction limit
	MinAmount       float64   `gorm:"not null;default:0.01"`   // Minimum transaction amount
	MaxAmount       float64   `gorm:"not null;default:50000"`  // Maximum transaction amount
	DailyUsed       float64   `gorm:"not null;default:0"`      // Amount used today
	MonthlyUsed     float64   `gorm:"not null;default:0"`      // Amount used this month
	LastResetDate   time.Time `gorm:"not null"`                // Last time limits were reset
	TransferLimit   float64   `gorm:"not null;default:5000"`   // Daily transfer limit
	WithdrawalLimit float64   `gorm:"not null;default:5000"`   // Daily withdrawal limit
	DepositLimit    float64   `gorm:"not null;default:10000"`  // Daily deposit limit
	TransferUsed    float64   `gorm:"not null;default:0"`      // Transfer amount used today
	WithdrawalUsed  float64   `gorm:"not null;default:0"`      // Withdrawal amount used today
	DepositUsed     float64   `gorm:"not null;default:0"`      // Deposit amount used today
	CreatedAt       time.Time `gorm:"not null"`
	UpdatedAt       time.Time `gorm:"not null"`
}

// TransactionUsage tracks daily/monthly transaction usage
type TransactionUsage struct {
	ID            uint      `gorm:"primaryKey"`
	AccountID     uint      `gorm:"not null;index"`
	TransactionID uint      `gorm:"not null;uniqueIndex"`
	Amount        float64   `gorm:"not null"`
	Type          string    `gorm:"not null"` // deposit, withdrawal, transfer
	Date          time.Time `gorm:"not null;index"`
	CreatedAt     time.Time `gorm:"not null"`
}

// TransactionLimitService provides methods for managing transaction limits
type TransactionLimitService struct {
	db *gorm.DB
}

// NewTransactionLimitService creates a new transaction limit service
func NewTransactionLimitService(db *gorm.DB) *TransactionLimitService {
	return &TransactionLimitService{db: db}
}

// GetLimit retrieves transaction limits for an account
func (s *TransactionLimitService) GetLimit(accountID uint) (*TransactionLimit, error) {
	var limit TransactionLimit
	err := s.db.Where("account_id = ?", accountID).First(&limit).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create default limits if not exists
			limit = TransactionLimit{
				AccountID:     accountID,
				LastResetDate: time.Now(),
			}
			err = s.db.Create(&limit).Error
		}
		return &limit, err
	}
	return &limit, nil
}

// CheckLimit checks if a transaction would exceed limits
func (s *TransactionLimitService) CheckLimit(accountID uint, amount float64, transactionType string) error {
	limit, err := s.GetLimit(accountID)
	if err != nil {
		return err
	}

	// Reset daily/monthly usage if needed
	s.resetUsageIfNeeded(limit)

	// Check amount limits
	if amount < limit.MinAmount {
		return errors.New("amount_below_minimum", "Amount is below minimum allowed", http.StatusBadRequest)
	}
	if amount > limit.MaxAmount {
		return errors.New("amount_above_maximum", "Amount exceeds maximum allowed", http.StatusBadRequest)
	}

	// Check daily/monthly limits based on transaction type
	switch transactionType {
	case "deposit":
		if limit.DepositUsed+amount > limit.DepositLimit {
			return errors.New("daily_deposit_limit_exceeded", "Daily deposit limit exceeded", http.StatusBadRequest)
		}
	case "withdrawal":
		if limit.WithdrawalUsed+amount > limit.WithdrawalLimit {
			return errors.New("daily_withdrawal_limit_exceeded", "Daily withdrawal limit exceeded", http.StatusBadRequest)
		}
	case "transfer":
		if limit.TransferUsed+amount > limit.TransferLimit {
			return errors.New("daily_transfer_limit_exceeded", "Daily transfer limit exceeded", http.StatusBadRequest)
		}
	}

	// Check overall daily and monthly limits
	if limit.DailyUsed+amount > limit.DailyLimit {
		return errors.New("daily_limit_exceeded", "Daily transaction limit exceeded", http.StatusBadRequest)
	}
	if limit.MonthlyUsed+amount > limit.MonthlyLimit {
		return errors.New("monthly_limit_exceeded", "Monthly transaction limit exceeded", http.StatusBadRequest)
	}

	return nil
}

// RecordTransaction records a transaction and updates usage
func (s *TransactionLimitService) RecordTransaction(accountID uint, transactionID uint, amount float64, transactionType string) error {
	limit, err := s.GetLimit(accountID)
	if err != nil {
		return err
	}

	// Reset usage if needed
	s.resetUsageIfNeeded(limit)

	// Update usage
	limit.DailyUsed += amount
	limit.MonthlyUsed += amount

	switch transactionType {
	case "deposit":
		limit.DepositUsed += amount
	case "withdrawal":
		limit.WithdrawalUsed += amount
	case "transfer":
		limit.TransferUsed += amount
	}

	// Record transaction usage
	usage := TransactionUsage{
		AccountID:     accountID,
		TransactionID: transactionID,
		Amount:        amount,
		Type:          transactionType,
		Date:          time.Now(),
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(limit).Error; err != nil {
			return err
		}
		return tx.Create(&usage).Error
	})
}

// resetUsageIfNeeded resets daily/monthly usage if date has changed
func (s *TransactionLimitService) resetUsageIfNeeded(limit *TransactionLimit) {
	now := time.Now()

	// Reset daily usage
	if !now.Truncate(24 * time.Hour).Equal(limit.LastResetDate.Truncate(24 * time.Hour)) {
		limit.DailyUsed = 0
		limit.DepositUsed = 0
		limit.WithdrawalUsed = 0
		limit.TransferUsed = 0
		limit.LastResetDate = now
	}

	// Reset monthly usage
	if now.Year() != limit.LastResetDate.Year() || now.Month() != limit.LastResetDate.Month() {
		limit.MonthlyUsed = 0
	}
}

// UpdateLimit updates transaction limits for an account
func (s *TransactionLimitService) UpdateLimit(accountID uint, updates map[string]interface{}) error {
	limit, err := s.GetLimit(accountID)
	if err != nil {
		return err
	}

	return s.db.Model(limit).Updates(updates).Error
}
