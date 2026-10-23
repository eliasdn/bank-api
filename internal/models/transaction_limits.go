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
				AccountID:       accountID,
				DailyLimit:      10000,
				MonthlyLimit:    100000,
				MinAmount:       0.01,
				MaxAmount:       50000,
				TransferLimit:   5000,
				WithdrawalLimit: 5000,
				DepositLimit:    10000,
				LastResetDate:   time.Now(),
			}
			if createErr := s.db.Create(&limit).Error; createErr != nil {
				// Retry query in case of concurrent creation race condition
				var fetchedLimit TransactionLimit
				if retryErr := s.db.Where("account_id = ?", accountID).First(&fetchedLimit).Error; retryErr == nil {
					return &fetchedLimit, nil
				}
				return nil, createErr
			}
			return &limit, nil
		}
		return nil, err
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

// RecordTransaction records a transaction and updates usage atomically using SQL expressions to prevent TOCTOU race conditions.
func (s *TransactionLimitService) RecordTransaction(accountID uint, transactionID uint, amount float64, transactionType string) error {
	limit, err := s.GetLimit(accountID)
	if err != nil {
		return err
	}

	// Capture initial reset date before checking for resets
	lastReset := limit.LastResetDate
	s.resetUsageIfNeeded(limit)

	now := time.Now()
	dateChanged := !now.Truncate(24 * time.Hour).Equal(lastReset.Truncate(24 * time.Hour))
	monthChanged := now.Year() != lastReset.Year() || now.Month() != lastReset.Month()

	updates := map[string]interface{}{
		"updated_at": now,
	}

	if dateChanged {
		updates["daily_used"] = amount
		updates["last_reset_date"] = limit.LastResetDate
		updates["deposit_used"] = 0.0
		updates["withdrawal_used"] = 0.0
		updates["transfer_used"] = 0.0
		switch transactionType {
		case "deposit":
			updates["deposit_used"] = amount
		case "withdrawal":
			updates["withdrawal_used"] = amount
		case "transfer":
			updates["transfer_used"] = amount
		}
		if monthChanged {
			updates["monthly_used"] = amount
		} else {
			updates["monthly_used"] = gorm.Expr("monthly_used + ?", amount)
		}
	} else {
		updates["daily_used"] = gorm.Expr("daily_used + ?", amount)
		updates["monthly_used"] = gorm.Expr("monthly_used + ?", amount)
		switch transactionType {
		case "deposit":
			updates["deposit_used"] = gorm.Expr("deposit_used + ?", amount)
		case "withdrawal":
			updates["withdrawal_used"] = gorm.Expr("withdrawal_used + ?", amount)
		case "transfer":
			updates["transfer_used"] = gorm.Expr("transfer_used + ?", amount)
		}
	}

	// Record transaction usage
	usage := TransactionUsage{
		AccountID:     accountID,
		TransactionID: transactionID,
		Amount:        amount,
		Type:          transactionType,
		Date:          now,
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(limit).UpdateColumns(updates).Error; err != nil {
			return err
		}
		return tx.Create(&usage).Error
	})
}

// ResetUsageIfNeededForTest exports resetUsageIfNeeded for external tests.
func ResetUsageIfNeededForTest(s *TransactionLimitService, limit *TransactionLimit) {
	s.resetUsageIfNeeded(limit)
}

// resetUsageIfNeeded resets daily/monthly usage if date has changed
func (s *TransactionLimitService) resetUsageIfNeeded(limit *TransactionLimit) {
	now := time.Now()
	lastReset := limit.LastResetDate

	// Reset monthly usage if month or year changed
	if now.Year() != lastReset.Year() || now.Month() != lastReset.Month() {
		limit.MonthlyUsed = 0
	}

	// Reset daily usage if date changed
	if !now.Truncate(24 * time.Hour).Equal(lastReset.Truncate(24 * time.Hour)) {
		limit.DailyUsed = 0
		limit.DepositUsed = 0
		limit.WithdrawalUsed = 0
		limit.TransferUsed = 0
		limit.LastResetDate = now
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
