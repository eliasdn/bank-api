package handlers

import (
	"bank-api/internal/models"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func generateAccountNumber() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("ACC%06d", rand.Intn(1000000))
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
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	// Get total count
	var total int64
	h.DB.Model(&models.Account{}).Where("user_id = ?", userID).Count(&total)

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

	var account models.Account
	if err := c.ShouldBindJSON(&account); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	account.UserID = userID

	// Generate account number if not provided
	if account.AccountNumber == "" {
		account.AccountNumber = generateAccountNumber()
	}

	// Set default account type if not provided
	if account.AccountType == "" {
		account.AccountType = "checking"
	}

	if err := h.DB.Create(&account).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create account"})
		return
	}

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
