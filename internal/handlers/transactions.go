package handlers

import (
	"bank-api/internal/dto"
	"bank-api/internal/models"
	"bank-api/internal/validation"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Deposit handles deposit transactions
func (h *Handler) Deposit(c *gin.Context) {
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
		c.Data(http.StatusNotFound, "application/json; charset=utf-8", accountNotFoundErrorJSON)
		return
	}

	var deposit dto.DepositRequest
	if err := c.ShouldBindJSON(&deposit); err != nil {
		c.Data(http.StatusBadRequest, "application/json; charset=utf-8", invalidDepositAmountErrorJSON)
		return
	}

	validator := validation.New()
	if err := validator.ValidateAmount(deposit.Amount); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	if err := h.LimitService.CheckLimit(account.ID, deposit.Amount, "deposit"); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
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
	if err := tx.Model(&account).UpdateColumn("balance", gorm.Expr("balance + ?", deposit.Amount)).Error; err != nil {
		tx.Rollback()
		c.Data(http.StatusInternalServerError, "application/json; charset=utf-8", failedToUpdateBalanceErrorJSON)
		return
	}

	// Reload account to get the new balance
	if err := tx.First(&account, account.ID).Error; err != nil {
		tx.Rollback()
		c.Data(http.StatusInternalServerError, "application/json; charset=utf-8", failedToReloadAccountErrorJSON)
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
		c.Data(http.StatusInternalServerError, "application/json; charset=utf-8", failedToRecordTransactionErrorJSON)
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.Data(http.StatusInternalServerError, "application/json; charset=utf-8", transactionFailedErrorJSON)
		return
	}

	// Record transaction limits
	if err := h.LimitService.RecordTransaction(account.ID, transaction.ID, deposit.Amount, "deposit"); err != nil {
		// Log error but don't fail the transaction since it's already committed
		c.Error(err)
	}

	// Invalidate cache
	h.TxCountCache.Delete(account.ID)

	// Log audit event
	if err := h.AuditService.LogTransaction(c, userID, &transaction, &account); err != nil {
		// Log error but don't fail the transaction
		c.Error(err)
	}

	response := dto.DepositResponse{
		Message:       "Deposit successful",
		Balance:       account.Balance,
		TransactionID: transaction.ID,
	}
	c.JSON(http.StatusOK, response)
}

// Withdraw handles withdrawal transactions
func (h *Handler) Withdraw(c *gin.Context) {
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
		c.Data(http.StatusNotFound, "application/json; charset=utf-8", accountNotFoundErrorJSON)
		return
	}

	var withdrawal dto.WithdrawRequest
	if err := c.ShouldBindJSON(&withdrawal); err != nil {
		c.Data(http.StatusBadRequest, "application/json; charset=utf-8", invalidWithdrawalAmountErrorJSON)
		return
	}

	validator := validation.New()
	if err := validator.ValidateAmount(withdrawal.Amount); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	if err := h.LimitService.CheckLimit(account.ID, withdrawal.Amount, "withdrawal"); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	if account.Balance < withdrawal.Amount {
		c.Data(http.StatusBadRequest, "application/json; charset=utf-8", insufficientFundsErrorJSON)
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
	res := tx.Model(&account).Where("balance >= ?", withdrawal.Amount).UpdateColumn("balance", gorm.Expr("balance - ?", withdrawal.Amount))
	if res.Error != nil {
		tx.Rollback()
		c.Data(http.StatusInternalServerError, "application/json; charset=utf-8", failedToUpdateBalanceErrorJSON)
		return
	}
	if res.RowsAffected == 0 {
		tx.Rollback()
		c.Data(http.StatusBadRequest, "application/json; charset=utf-8", insufficientFundsErrorJSON)
		return
	}

	// Reload account to get the new balance
	if err := tx.First(&account, account.ID).Error; err != nil {
		tx.Rollback()
		c.Data(http.StatusInternalServerError, "application/json; charset=utf-8", failedToReloadAccountErrorJSON)
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
		c.Data(http.StatusInternalServerError, "application/json; charset=utf-8", failedToRecordTransactionErrorJSON)
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.Data(http.StatusInternalServerError, "application/json; charset=utf-8", transactionFailedErrorJSON)
		return
	}

	// Record transaction limits
	if err := h.LimitService.RecordTransaction(account.ID, transaction.ID, withdrawal.Amount, "withdrawal"); err != nil {
		c.Error(err)
	}

	// Invalidate cache
	h.TxCountCache.Delete(account.ID)

	// Log audit event
	if err := h.AuditService.LogTransaction(c, userID, &transaction, &account); err != nil {
		// Log error but don't fail the transaction
		c.Error(err)
	}

	response := dto.WithdrawResponse{
		Message:       "Withdrawal successful",
		Balance:       account.Balance,
		TransactionID: transaction.ID,
	}
	c.JSON(http.StatusOK, response)
}

// Transfer handles transfer transactions
func (h *Handler) Transfer(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		c.Data(http.StatusUnauthorized, "application/json", unauthorizedErrorJSON)
		return
	}

	fromAccountIDStr := c.Param("id")
	fromAccountID := validation.ParseInt(fromAccountIDStr, 0)
	if fromAccountID == 0 {
		c.Data(http.StatusBadRequest, "application/json; charset=utf-8", invalidAccountIDErrorJSON)
		return
	}

	var fromAccount models.Account
	if err := h.DB.Where("id = ? AND user_id = ?", fromAccountID, userID).First(&fromAccount).Error; err != nil {
		c.Data(http.StatusNotFound, "application/json; charset=utf-8", sourceAccountNotFoundErrorJSON)
		return
	}

	var transfer dto.TransferRequest
	if err := c.ShouldBindJSON(&transfer); err != nil {
		c.Data(http.StatusBadRequest, "application/json; charset=utf-8", invalidTransferRequestErrorJSON)
		return
	}

	validator := validation.New()
	if err := validator.ValidateTransfer(fromAccount.ID, transfer.ToAccountID, transfer.Amount, transfer.Description); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	if err := h.LimitService.CheckLimit(fromAccount.ID, transfer.Amount, "transfer"); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	if fromAccount.Balance < transfer.Amount {
		c.Data(http.StatusBadRequest, "application/json; charset=utf-8", insufficientFundsErrorJSON)
		return
	}

	// Check if destination account exists
	var toAccount models.Account
	if err := h.DB.First(&toAccount, transfer.ToAccountID).Error; err != nil {
		c.Data(http.StatusBadRequest, "application/json; charset=utf-8", destinationAccountNotFoundErrorJSON)
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
	res := tx.Model(&fromAccount).Where("balance >= ?", transfer.Amount).UpdateColumn("balance", gorm.Expr("balance - ?", transfer.Amount))
	if res.Error != nil {
		tx.Rollback()
		c.Data(http.StatusInternalServerError, "application/json; charset=utf-8", failedToUpdateSourceAccountErrorJSON)
		return
	}
	if res.RowsAffected == 0 {
		tx.Rollback()
		c.Data(http.StatusBadRequest, "application/json; charset=utf-8", insufficientFundsErrorJSON)
		return
	}

	// Update destination account
	if err := tx.Model(&toAccount).UpdateColumn("balance", gorm.Expr("balance + ?", transfer.Amount)).Error; err != nil {
		tx.Rollback()
		c.Data(http.StatusInternalServerError, "application/json; charset=utf-8", failedToUpdateDestinationAccountErrorJSON)
		return
	}

	// Reload accounts to get accurate balances
	if err := tx.First(&fromAccount, fromAccount.ID).Error; err != nil {
		tx.Rollback()
		c.Data(http.StatusInternalServerError, "application/json; charset=utf-8", failedToReloadSourceAccountErrorJSON)
		return
	}
	if err := tx.First(&toAccount, toAccount.ID).Error; err != nil {
		tx.Rollback()
		c.Data(http.StatusInternalServerError, "application/json; charset=utf-8", failedToReloadDestinationAccountErrorJSON)
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
		c.Data(http.StatusInternalServerError, "application/json; charset=utf-8", failedToRecordOutgoingTransactionErrorJSON)
		return
	}

	if err := tx.Create(&incomingTx).Error; err != nil {
		tx.Rollback()
		c.Data(http.StatusInternalServerError, "application/json; charset=utf-8", failedToRecordIncomingTransactionErrorJSON)
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.Data(http.StatusInternalServerError, "application/json; charset=utf-8", transferFailedErrorJSON)
		return
	}

	// Record transaction limits
	if err := h.LimitService.RecordTransaction(fromAccount.ID, outgoingTx.ID, transfer.Amount, "transfer"); err != nil {
		c.Error(err)
	}

	// Invalidate cache for both accounts
	h.TxCountCache.Delete(fromAccount.ID)
	h.TxCountCache.Delete(toAccount.ID)

	// Log audit events for both transactions
	if err := h.AuditService.LogTransaction(c, userID, &outgoingTx, &fromAccount); err != nil {
		c.Error(err)
	}
	if err := h.AuditService.LogTransaction(c, userID, &incomingTx, &toAccount); err != nil {
		c.Error(err)
	}

	response := dto.TransferResponse{
		Message:       "Transfer successful",
		TransactionID: outgoingTx.ID,
	}
	c.JSON(http.StatusOK, response)
}

// GetTransactions handles getting transactions with pagination
func (h *Handler) GetTransactions(c *gin.Context) {
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
		c.Data(http.StatusNotFound, "application/json; charset=utf-8", accountNotFoundErrorJSON)
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

	if val, ok := h.TxCountCache.Load(account.ID); ok {
		total = val.(int64)
	} else {
		h.DB.Model(&models.Transaction{}).Where("account_id = ?", account.ID).Count(&total)
		h.TxCountCache.Store(account.ID, total)
	}

	// Get paginated transactions
	var transactions []models.Transaction
	if int64(offset) < total {
		if err := h.DB.Where("account_id = ?", account.ID).
			Order("created_at desc").
			Limit(limit).
			Offset(offset).
			Find(&transactions).Error; err != nil {
			c.Data(http.StatusInternalServerError, "application/json; charset=utf-8", failedToRetrieveTransactionsErrorJSON)
			return
		}
	}

	// Convert to DTOs
	transactionDTOs := make([]dto.TransactionResponse, len(transactions))
	for i := range transactions {
		t := &transactions[i]
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
