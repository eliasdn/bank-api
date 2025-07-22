package dto

import "time"

// TransactionResponse represents a transaction in API responses
type TransactionResponse struct {
	ID              uint      `json:"id"`
	AccountID       uint      `json:"account_id"`
	Amount          float64   `json:"amount"`
	TransactionType string    `json:"transaction_type"`
	Reference       string    `json:"reference"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// TransactionListResponse represents a paginated list of transactions
type TransactionListResponse struct {
	Transactions []TransactionResponse `json:"transactions"`
	Pagination   PaginationInfo        `json:"pagination"`
}

// PaginationInfo contains pagination metadata
type PaginationInfo struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"totalPages"`
}

// DepositRequest represents a deposit request
type DepositRequest struct {
	Amount float64 `json:"amount" binding:"required,gt=0"`
}

// WithdrawRequest represents a withdrawal request
type WithdrawRequest struct {
	Amount float64 `json:"amount" binding:"required,gt=0"`
}

// TransferRequest represents a transfer request
type TransferRequest struct {
	ToAccountID uint    `json:"to_account_id" binding:"required"`
	Amount      float64 `json:"amount" binding:"required,gt=0"`
	Description string  `json:"description"`
}

// DepositResponse represents a deposit response
type DepositResponse struct {
	Message       string  `json:"message"`
	Balance       float64 `json:"balance"`
	TransactionID uint    `json:"transaction_id"`
}

// WithdrawResponse represents a withdrawal response
type WithdrawResponse struct {
	Message       string  `json:"message"`
	Balance       float64 `json:"balance"`
	TransactionID uint    `json:"transaction_id"`
}

// TransferResponse represents a transfer response
type TransferResponse struct {
	Message       string  `json:"message"`
	FromBalance   float64 `json:"from_balance"`
	ToBalance     float64 `json:"to_balance"`
	TransactionID uint    `json:"transaction_id"`
}
