package handlers

import (
	"bank-api/internal/dto"
	"bank-api/internal/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Deposit handles deposit transactions
func (h *Handler) Deposit(c *gin.Context) {
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
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}

	var deposit struct {
		Amount float64 `json:"amount" binding:"required,gt=0"`
	}
	if err := c.ShouldBindJSON(&deposit); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid deposit amount"})
		return
	}

	// Start transaction
	tx := h.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update account balance
	newBalance := account.Balance + deposit.Amount
	if err := tx.Model(&account).Update("balance", newBalance).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update balance"})
		return
	}

	// Create transaction record
	transaction := models.Transaction{
		AccountID:       account.ID,
		Amount:          deposit.Amount,
		TransactionType: "deposit",
		Reference:       "Account deposit",
		Status:          "completed",
	}
	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to record transaction"})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "transaction failed"})
		return
	}

	// Log audit event
	if err := h.AuditService.LogTransaction(c, userID, &transaction, &account); err != nil {
		// Log error but don't fail the transaction
		c.Error(err)
	}

	response := dto.DepositResponse{
		Message:       "Deposit successful",
		Balance:       newBalance,
		TransactionID: transaction.ID,
	}
	c.JSON(http.StatusOK, response)
}

// Withdraw handles withdrawal transactions
func (h *Handler) Withdraw(c *gin.Context) {
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
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}

	var withdrawal struct {
		Amount float64 `json:"amount" binding:"required,gt=0"`
	}
	if err := c.ShouldBindJSON(&withdrawal); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid withdrawal amount"})
		return
	}

	if account.Balance < withdrawal.Amount {
		c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient funds"})
		return
	}

	// Start transaction
	tx := h.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update account balance
	newBalance := account.Balance - withdrawal.Amount
	if err := tx.Model(&account).Update("balance", newBalance).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update balance"})
		return
	}

	// Create transaction record
	transaction := models.Transaction{
		AccountID:       account.ID,
		Amount:          withdrawal.Amount,
		TransactionType: "withdrawal",
		Reference:       "Account withdrawal",
		Status:          "completed",
	}
	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to record transaction"})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "transaction failed"})
		return
	}

	// Log audit event
	if err := h.AuditService.LogTransaction(c, userID, &transaction, &account); err != nil {
		// Log error but don't fail the transaction
		c.Error(err)
	}

	response := dto.WithdrawResponse{
		Message:       "Withdrawal successful",
		Balance:       newBalance,
		TransactionID: transaction.ID,
	}
	c.JSON(http.StatusOK, response)
}

// Transfer handles transfer transactions
func (h *Handler) Transfer(c *gin.Context) {
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

	fromAccountID := c.Param("id")
	var fromAccount models.Account
	if err := h.DB.Where("id = ? AND user_id = ?", fromAccountID, userID).First(&fromAccount).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "source account not found"})
		return
	}

	var transfer struct {
		ToAccountID uint    `json:"to_account_id" binding:"required"`
		Amount      float64 `json:"amount" binding:"required,gt=0"`
		Description string  `json:"description"`
	}
	if err := c.ShouldBindJSON(&transfer); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid transfer request"})
		return
	}

	if fromAccount.Balance < transfer.Amount {
		c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient funds"})
		return
	}

	// Check if destination account exists
	var toAccount models.Account
	if err := h.DB.First(&toAccount, transfer.ToAccountID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "destination account not found"})
		return
	}

	// Start transaction
	tx := h.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update source account
	if err := tx.Model(&fromAccount).Update("balance", fromAccount.Balance-transfer.Amount).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update source account"})
		return
	}

	// Update destination account
	if err := tx.Model(&toAccount).Update("balance", toAccount.Balance+transfer.Amount).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update destination account"})
		return
	}

	// Create transaction records
	outgoingTx := models.Transaction{
		AccountID:       fromAccount.ID,
		Amount:          transfer.Amount,
		TransactionType: "transfer",
		Reference:       "Transfer to account " + toAccount.AccountNumber + ": " + transfer.Description,
		Status:          "completed",
	}
	incomingTx := models.Transaction{
		AccountID:       toAccount.ID,
		Amount:          transfer.Amount,
		TransactionType: "transfer",
		Reference:       "Transfer from account " + fromAccount.AccountNumber + ": " + transfer.Description,
		Status:          "completed",
	}

	if err := tx.Create(&outgoingTx).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to record outgoing transaction"})
		return
	}

	if err := tx.Create(&incomingTx).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to record incoming transaction"})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "transfer failed"})
		return
	}

	// Log audit events for both transactions
	if err := h.AuditService.LogTransaction(c, userID, &outgoingTx, &fromAccount); err != nil {
		c.Error(err)
	}
	if err := h.AuditService.LogTransaction(c, userID, &incomingTx, &toAccount); err != nil {
		c.Error(err)
	}

	response := dto.TransferResponse{
		Message:       "Transfer successful",
		FromBalance:   fromAccount.Balance - transfer.Amount,
		ToBalance:     toAccount.Balance + transfer.Amount,
		TransactionID: outgoingTx.ID,
	}
	c.JSON(http.StatusOK, response)
}

// GetTransactions handles getting transactions with pagination
func (h *Handler) GetTransactions(c *gin.Context) {
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
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
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
	h.DB.Model(&models.Transaction{}).Where("account_id = ?", accountID).Count(&total)

	// Get paginated transactions
	var transactions []models.Transaction
	if err := h.DB.Where("account_id = ?", accountID).
		Order("created_at desc").
		Limit(limit).
		Offset(offset).
		Find(&transactions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve transactions"})
		return
	}

	// Convert to DTOs
	transactionDTOs := make([]dto.TransactionResponse, len(transactions))
	for i, t := range transactions {
		transactionDTOs[i] = dto.TransactionResponse{
			ID:              t.ID,
			AccountID:       t.AccountID,
			Amount:          t.Amount,
			TransactionType: t.TransactionType,
			Reference:       t.Reference,
			Status:          t.Status,
			CreatedAt:       t.CreatedAt,
			UpdatedAt:       t.UpdatedAt,
		}
	}

	// Calculate pagination metadata
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	response := dto.TransactionListResponse{
		Transactions: transactionDTOs,
		Pagination: dto.PaginationInfo{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}
	c.JSON(http.StatusOK, response)
}
