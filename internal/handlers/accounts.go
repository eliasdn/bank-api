package handlers

import (
	"bank-api/internal/models"
	"bank-api/internal/validation"
	"crypto/rand"
	"encoding/binary"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// generateAccountNumber generates a cryptographically secure random 9-character account number.
// Uses crypto/rand instead of math/rand to prevent predictable account number generation.
// Optimized to avoid fmt.Sprintf overhead by formatting directly into a byte slice.
// Uses rejection sampling to prevent modulo bias while avoiding math/big allocations.
func generateAccountNumber() string {
	var buf [4]byte
	var val uint32

	// maxAcceptable = 2^32 - (2^32 % 1000000) = 4294000000
	const maxAcceptable uint32 = 4294000000

	for {
		_, _ = rand.Read(buf[:])
		val = binary.LittleEndian.Uint32(buf[:])
		if val < maxAcceptable {
			break
		}
	}

	n := int(val % 1000000)

	var b [9]byte
	b[0] = 'A'
	b[1] = 'C'
	b[2] = 'C'
	for i := 8; i >= 3; i-- {
		b[i] = byte(n%10) + '0'
		n /= 10
	}
	return string(b[:])
}

type PaginationInfo struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"totalPages"`
}

type AccountListResponse struct {
	Data       []models.Account `json:"data"`
	Pagination PaginationInfo   `json:"pagination"`
}

type AccountResponse struct {
	Data models.Account `json:"data"`
}

type CreateAccountRequest struct {
	AccountType string `json:"account_type" binding:"omitempty,max=20"`
}

func (h *Handler) GetAccounts(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		c.Data(http.StatusUnauthorized, "application/json", unauthorizedErrorJSON)
		return
	}

	// Parse pagination parameters
	page := validation.ParseInt(c.Query("page"), 1)
	limit := validation.ParseInt(c.Query("limit"), 20)
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
	// Pre-allocate slice with expected capacity to prevent intermediate array re-allocations
	allocLimit := limit
	if allocLimit <= 0 {
		allocLimit = 1
	}
	accounts := make([]models.Account, 0, allocLimit)
	if int64(offset) < total {
		if err := h.DB.Where("user_id = ?", userID).
			Order("created_at desc").
			Limit(limit).
			Offset(offset).
			Find(&accounts).Error; err != nil {
			c.Data(http.StatusInternalServerError, "application/json; charset=utf-8", failedToRetrieveAccountsErrorJSON)
			return
		}
	}

	if accounts == nil {
		accounts = []models.Account{} // Ensure empty array instead of nil
	}

	// Calculate pagination metadata
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	c.JSON(http.StatusOK, AccountListResponse{
		Data: accounts,
		Pagination: PaginationInfo{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

func (h *Handler) CreateAccount(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		c.Data(http.StatusUnauthorized, "application/json", unauthorizedErrorJSON)
		return
	}

	var req CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Data(http.StatusBadRequest, "application/json; charset=utf-8", invalidRequestErrorJSON)
		return
	}

	accountType := req.AccountType
	// Set default account type if not provided
	if accountType == "" {
		accountType = "checking"
	}

	validator := validation.New()
	if err := validator.ValidateAccountType(accountType); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	account := models.Account{
		UserID:        userID,
		AccountNumber: generateAccountNumber(),
		AccountType:   accountType,
		Balance:       0, // Explicitly set to 0 to prevent mass assignment
	}

	if err := h.DB.Create(&account).Error; err != nil {
		c.Data(http.StatusInternalServerError, "application/json; charset=utf-8", failedToCreateAccountErrorJSON)
		return
	}

	// Invalidate account count cache
	h.AccountCountCache.Delete(userID)

	// Log audit event
	if err := h.AuditService.LogAccountCreation(c, userID, &account); err != nil {
		c.Error(err)
	}

	c.JSON(http.StatusCreated, struct {
		Data models.Account `json:"data"`
	}{Data: account})
}

func (h *Handler) GetAccount(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		c.Data(http.StatusUnauthorized, "application/json", unauthorizedErrorJSON)
		return
	}

	accountIDStr := c.Param("id")
	accountID := validation.ParseInt(accountIDStr, 0)
	if accountID == 0 {
		c.Data(http.StatusBadRequest, "application/json; charset=utf-8", invalidAccountIDErrorJSON)
		return
	}

	var account models.Account
	if err := h.DB.Where("id = ? AND user_id = ?", accountID, userID).First(&account).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.Data(http.StatusNotFound, "application/json; charset=utf-8", accountNotFoundErrorJSON)
		} else {
			c.Data(http.StatusInternalServerError, "application/json; charset=utf-8", failedToRetrieveAccountErrorJSON)
		}
		return
	}

	c.JSON(http.StatusOK, struct {
		Data models.Account `json:"data"`
	}{Data: account})
}
