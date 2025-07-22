package services

import (
	"bank-api/internal/audit"
	"bank-api/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AuditService provides audit logging functionality
type AuditService struct {
	logger audit.AuditLogger
}

// NewAuditService creates a new audit service
func NewAuditService(db *gorm.DB) *AuditService {
	return &AuditService{
		logger: audit.NewDatabaseAuditLogger(db),
	}
}

// LogAction logs a generic action
func (s *AuditService) LogAction(c *gin.Context, userID uint, action, resource, resourceID, description string, oldValue, newValue interface{}) error {
	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	return s.logger.LogAction(userID, action, resource, resourceID, description, oldValue, newValue, ipAddress, userAgent)
}

// LogTransaction logs a financial transaction
func (s *AuditService) LogTransaction(c *gin.Context, userID uint, transaction *models.Transaction, account *models.Account) error {
	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	return s.logger.LogTransaction(userID, transaction, account, ipAddress, userAgent)
}

// LogUserAction logs user-related actions
func (s *AuditService) LogUserAction(c *gin.Context, userID uint, action string, oldUser, newUser *models.User) error {
	return s.LogAction(c, userID, action, "user", string(rune(userID)), "User action", oldUser, newUser)
}

// LogAccountAction logs account-related actions
func (s *AuditService) LogAccountAction(c *gin.Context, userID uint, action string, oldAccount, newAccount *models.Account) error {
	return s.LogAction(c, userID, action, "account", string(rune(newAccount.ID)), "Account action", oldAccount, newAccount)
}

// LogAccountCreation logs account creation
func (s *AuditService) LogAccountCreation(c *gin.Context, userID uint, accounts *models.Account) error {
	return s.LogAction(c, userID, "create", "account", accounts.AccountNumber, "Account created", nil, accounts)
}

// LogUserRegistration logs user registration
func (s *AuditService) LogUserRegistration(c *gin.Context, user *models.User) error {
	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	return s.logger.LogAction(user.ID, "register", "user", string(rune(user.ID)), "User registered", nil, user, ipAddress, userAgent)
}

// GetUserIDFromContext extracts user ID from gin context
func (s *AuditService) GetUserIDFromContext(c *gin.Context) (uint, bool) {
	userID, exists := c.Get("userID")
	if !exists {
		return 0, false
	}

	if id, ok := userID.(uint); ok {
		return id, true
	}

	return 0, false
}

// GetIPAddress extracts IP address from request
func (s *AuditService) GetIPAddress(c *gin.Context) string {
	// Check for X-Forwarded-For header (for proxies)
	xff := c.GetHeader("X-Forwarded-For")
	if xff != "" {
		return xff
	}

	// Check for X-Real-IP header
	xri := c.GetHeader("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fallback to direct client IP
	return c.ClientIP()
}

// Middleware to extract and set user ID in context
func (s *AuditService) UserContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// This middleware should be used after auth middleware
		// which sets the user ID in the context
		c.Next()
	}
}
