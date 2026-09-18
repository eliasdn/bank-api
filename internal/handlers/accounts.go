package handlers

import (
	"bank-api/internal/models"
	"bank-api/internal/validation"
	"crypto/rand"
	"math/big"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// generateAccountNumber generates a cryptographically secure random 9-character account number.
// Uses crypto/rand instead of math/rand to prevent predictable account number generation.
// Optimized to avoid fmt.Sprintf overhead by formatting directly into a byte slice.
func generateAccountNumber() string {
	nBig, err := rand.Int(rand.Reader, big.NewInt(1000000))
	var n int
	if err == nil {
		n = int(nBig.Int64())
	}
	b := make([]byte, 9)
	b[0] = 'A'
	b[1] = 'C'
	b[2] = 'C'
	for i := 8; i >= 3; i-- {
		b[i] = byte(n%10) + '0'
		n /= 10
	}
	return string(b)
}

type CreateAccountRequest struct {
	AccountType string `json:"account_type"`
}

func (h *Handler) GetAccounts(c *gin.Context) {
	userIDVal, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID, ok := userIDVal.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Parse pagination parameters
	page := validation.ParsePaginationParam(c.DefaultQuery("page", "1"), 1)
	limit := validation.ParsePaginationParam(c.DefaultQuery("limit", "20"), 20)
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	// Get total count
	var total int64
	if val, ok := h.AccountCountCache.Load(userID); ok {
		total = val.(int64)
	} else {
		h.DB.Model(&models.Account{}).Where("user_id = ?", userID).Count(&total)
		h.AccountCountCache.Store(userID, total)
	}

	// Get paginated accounts
	var accounts []models.Account
	if err := h.DB.Where("user_id = ?", userID).
		Order("created_at desc").
		Limit(limit).
		Offset(offset).
		Find(&accounts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve accounts"})
		return
	}

	if accounts == nil {
		accounts = []models.Account{} // Ensure empty array instead of nil
	}

	// Calculate pagination metadata
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	c.JSON(http.StatusOK, gin.H{
		"data": accounts,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": totalPages,
		},
	})
}

func (h *Handler) CreateAccount(c *gin.Context) {
	userIDVal, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID, ok := userIDVal.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	accountType := req.AccountType
	// Set default account type if not provided
	if accountType == "" {
		accountType = "checking"
	}

	validator := validation.New()
	if err := validator.ValidateAccountType(accountType); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	account := models.Account{
		UserID:        userID,
		AccountNumber: generateAccountNumber(),
		AccountType:   accountType,
		Balance:       0, // Explicitly set to 0 to prevent mass assignment
	}

	if err := h.DB.Create(&account).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create account"})
		return
	}

	// Invalidate account count cache
	h.AccountCountCache.Delete(userID)

	// Log audit event
	if err := h.AuditService.LogAccountCreation(c, userID, &account); err != nil {
		c.Error(err)
	}

	c.JSON(http.StatusCreated, gin.H{"data": account})
}

func (h *Handler) GetAccount(c *gin.Context) {
	userIDVal, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID, ok := userIDVal.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	accountID := c.Param("id")
	var account models.Account
	if err := h.DB.Where("id = ? AND user_id = ?", accountID, userID).First(&account).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve account"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": account})
}
