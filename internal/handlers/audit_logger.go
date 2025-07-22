package handlers

import (
	"bank-api/internal/models"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AuditLog represents an audit log entry
type AuditLog struct {
	ID            uint      `gorm:"primaryKey"`
	UserID        uint      `gorm:"not null;index"`
	AccountID     uint      `gorm:"not null;index"`
	TransactionID uint      `gorm:"index"`
	Action        string    `gorm:"not null"`
	Details       string    `gorm:"type:text"`
	IPAddress     string    `gorm:"not null"`
	UserAgent     string    `gorm:"not null"`
	CreatedAt     time.Time `gorm:"not null"`
}

// AuditLogger handles audit logging
type AuditLogger struct {
	db *gorm.DB
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(db *gorm.DB) *AuditLogger {
	return &AuditLogger{db: db}
}

// LogTransaction logs a transaction to the audit log
func (a *AuditLogger) LogTransaction(c *gin.Context, transaction *models.Transaction, accountID uint, userID uint) {
	log := AuditLog{
		UserID:        userID,
		AccountID:     accountID,
		TransactionID: transaction.ID,
		Action:        transaction.TransactionType,
		Details:       transaction.Reference,
		IPAddress:     c.ClientIP(),
		UserAgent:     c.GetHeader("User-Agent"),
		CreatedAt:     time.Now(),
	}

	a.db.Create(&log)
}

// LogAccountAction logs an account action to the audit log
func (a *AuditLogger) LogAccountAction(c *gin.Context, action string, accountID uint, userID uint, details string) {
	log := AuditLog{
		UserID:    userID,
		AccountID: accountID,
		Action:    action,
		Details:   details,
		IPAddress: c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
		CreatedAt: time.Now(),
	}

	a.db.Create(&log)
}

// LogUserAction logs a user action to the audit log
func (a *AuditLogger) LogUserAction(c *gin.Context, action string, userID uint, details string) {
	log := AuditLog{
		UserID:    userID,
		Action:    action,
		Details:   details,
		IPAddress: c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
		CreatedAt: time.Now(),
	}

	a.db.Create(&log)
}

// GetAuditLogs retrieves audit logs for an account
func (a *AuditLogger) GetAuditLogs(accountID uint, limit int) ([]AuditLog, error) {
	var logs []AuditLog
	err := a.db.Where("account_id = ?", accountID).
		Order("created_at desc").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}
