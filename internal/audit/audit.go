package audit

import (
	"bank-api/internal/models"
	"encoding/json"
	"log"
	"strconv"
	"time"

	"gorm.io/gorm"
)

// AuditLog represents an audit log entry
type AuditLog struct {
	ID          uint      `gorm:"primaryKey"`
	UserID      uint      `gorm:"not null"`
	Action      string    `gorm:"not null"`
	Resource    string    `gorm:"not null"`
	ResourceID  string    `gorm:"not null"`
	Description string    `gorm:"not null"`
	OldValue    string    `gorm:"type:text"`
	NewValue    string    `gorm:"type:text"`
	IPAddress   string    `gorm:"not null"`
	UserAgent   string    `gorm:"not null"`
	CreatedAt   time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
}

// AuditLogger interface for audit logging
type AuditLogger interface {
	LogAction(userID uint, action, resource, resourceID, description string, oldValue, newValue interface{}, ipAddress, userAgent string) error
	LogTransaction(userID uint, transaction *models.Transaction, account *models.Account, ipAddress, userAgent string) error
}

// DatabaseAuditLogger implements AuditLogger using database storage
type DatabaseAuditLogger struct {
	db *gorm.DB
}

// NewDatabaseAuditLogger creates a new database audit logger
func NewDatabaseAuditLogger(db *gorm.DB) *DatabaseAuditLogger {
	return &DatabaseAuditLogger{db: db}
}

// LogAction logs a generic action
func (l *DatabaseAuditLogger) LogAction(userID uint, action, resource, resourceID, description string, oldValue, newValue interface{}, ipAddress, userAgent string) error {
	oldValueJSON, _ := json.Marshal(oldValue)
	newValueJSON, _ := json.Marshal(newValue)

	auditLog := AuditLog{
		UserID:      userID,
		Action:      action,
		Resource:    resource,
		ResourceID:  resourceID,
		Description: description,
		OldValue:    string(oldValueJSON),
		NewValue:    string(newValueJSON),
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
		CreatedAt:   time.Now(),
	}

	return l.db.Create(&auditLog).Error
}

// txLog represents the JSON structure for a logged transaction
type txLog struct {
	TransactionID uint    `json:"transaction_id"`
	Amount        float64 `json:"amount"`
	Type          string  `json:"type"`
	Status        string  `json:"status"`
	Reference     string  `json:"reference"`
	AccountID     uint    `json:"account_id"`
	AccountNumber string  `json:"account_number"`
	NewBalance    float64 `json:"new_balance"`
}

// LogTransaction logs a financial transaction
func (l *DatabaseAuditLogger) LogTransaction(userID uint, transaction *models.Transaction, account *models.Account, ipAddress, userAgent string) error {
	description := transaction.TransactionType + " of " + strconv.FormatFloat(transaction.Amount, 'f', 2, 64) + " on account " + account.AccountNumber

	transactionJSON, _ := json.Marshal(txLog{
		TransactionID: transaction.ID,
		Amount:        transaction.Amount,
		Type:          transaction.TransactionType,
		Status:        transaction.Status,
		Reference:     transaction.Reference,
		AccountID:     account.ID,
		AccountNumber: account.AccountNumber,
		NewBalance:    account.Balance,
	})

	auditLog := AuditLog{
		UserID:      userID,
		Action:      transaction.TransactionType,
		Resource:    "transaction",
		ResourceID:  strconv.FormatUint(uint64(transaction.ID), 10),
		Description: description,
		NewValue:    string(transactionJSON),
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
		CreatedAt:   time.Now(),
	}

	return l.db.Create(&auditLog).Error
}

// ConsoleAuditLogger implements AuditLogger for development/debugging
type ConsoleAuditLogger struct{}

// NewConsoleAuditLogger creates a new console audit logger
func NewConsoleAuditLogger() *ConsoleAuditLogger {
	return &ConsoleAuditLogger{}
}

// LogAction logs a generic action to console
func (l *ConsoleAuditLogger) LogAction(userID uint, action, resource, resourceID, description string, oldValue, newValue interface{}, ipAddress, userAgent string) error {
	oldValueJSON, _ := json.Marshal(oldValue)
	newValueJSON, _ := json.Marshal(newValue)

	log.Printf("[AUDIT] UserID: %d, Action: %s, Resource: %s, ResourceID: %s, Description: %s, OldValue: %s, NewValue: %s, IP: %s, UserAgent: %s",
		userID, action, resource, resourceID, description, string(oldValueJSON), string(newValueJSON), ipAddress, userAgent)
	return nil
}

// LogTransaction logs a financial transaction to console
func (l *ConsoleAuditLogger) LogTransaction(userID uint, transaction *models.Transaction, account *models.Account, ipAddress, userAgent string) error {
	log.Printf("[AUDIT] Transaction: UserID: %d, Type: %s, Amount: %.2f, Account: %s, Status: %s, IP: %s",
		userID, transaction.TransactionType, transaction.Amount, account.AccountNumber, transaction.Status, ipAddress)
	return nil
}
