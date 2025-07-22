package validation

import (
	"bank-api/internal/errors"
	"regexp"
	"strings"
)

// Validation rules
const (
	MinUsernameLength = 3
	MaxUsernameLength = 50
	MinPasswordLength = 8
	MaxPasswordLength = 100
	MinFullNameLength = 2
	MaxFullNameLength = 100
	MinEmailLength    = 5
	MaxEmailLength    = 255
)

var (
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
)

// Validator provides validation methods
type Validator struct{}

// New creates a new validator instance
func New() *Validator {
	return &Validator{}
}

// ValidateUserRegistration validates user registration input
func (v *Validator) ValidateUserRegistration(username, email, password, fullName string) error {
	if err := v.ValidateUsername(username); err != nil {
		return err
	}
	if err := v.ValidateEmail(email); err != nil {
		return err
	}
	if err := v.ValidatePassword(password); err != nil {
		return err
	}
	if err := v.ValidateFullName(fullName); err != nil {
		return err
	}
	return nil
}

// ValidateUsername validates username format
func (v *Validator) ValidateUsername(username string) error {
	if len(username) < MinUsernameLength || len(username) > MaxUsernameLength {
		return errors.NewValidationError("username must be between %d and %d characters", MinUsernameLength, MaxUsernameLength)
	}
	if !usernameRegex.MatchString(username) {
		return errors.NewValidationError("username can only contain letters, numbers, underscores, and hyphens")
	}
	return nil
}

// ValidateEmail validates email format
func (v *Validator) ValidateEmail(email string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	if len(email) < MinEmailLength || len(email) > MaxEmailLength {
		return errors.NewValidationError("email must be between %d and %d characters", MinEmailLength, MaxEmailLength)
	}
	if !emailRegex.MatchString(email) {
		return errors.NewValidationError("invalid email format")
	}
	return nil
}

// ValidatePassword validates password strength
func (v *Validator) ValidatePassword(password string) error {
	if len(password) < MinPasswordLength || len(password) > MaxPasswordLength {
		return errors.NewValidationError("password must be between %d and %d characters", MinPasswordLength, MaxPasswordLength)
	}

	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
	hasSpecial := regexp.MustCompile(`[!@#$%^&*(),.?":{}|<>]`).MatchString(password)

	if !hasUpper {
		return errors.NewValidationError("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return errors.NewValidationError("password must contain at least one lowercase letter")
	}
	if !hasNumber {
		return errors.NewValidationError("password must contain at least one number")
	}
	if !hasSpecial {
		return errors.NewValidationError("password must contain at least one special character")
	}

	return nil
}

// ValidateFullName validates full name format
func (v *Validator) ValidateFullName(fullName string) error {
	if len(fullName) < MinFullNameLength || len(fullName) > MaxFullNameLength {
		return errors.NewValidationError("full name must be between %d and %d characters", MinFullNameLength, MaxFullNameLength)
	}
	if strings.TrimSpace(fullName) == "" {
		return errors.NewValidationError("full name cannot be empty")
	}
	return nil
}

// ValidateAccountType validates account type
func (v *Validator) ValidateAccountType(accountType string) error {
	validTypes := map[string]bool{
		"checking": true,
		"savings":  true,
		"credit":   true,
	}
	if !validTypes[accountType] {
		return errors.NewValidationError("account type must be one of: checking, savings, credit")
	}
	return nil
}

// ValidateAmount validates transaction amount
func (v *Validator) ValidateAmount(amount float64) error {
	if amount <= 0 {
		return errors.NewValidationError("amount must be greater than zero")
	}
	if amount > 1000000 { // $1M limit
		return errors.NewValidationError("amount exceeds maximum allowed limit")
	}
	return nil
}

// ValidateTransactionType validates transaction type
func (v *Validator) ValidateTransactionType(transactionType string) error {
	validTypes := map[string]bool{
		"deposit":    true,
		"withdrawal": true,
		"transfer":   true,
	}
	if !validTypes[transactionType] {
		return errors.NewValidationError("transaction type must be one of: deposit, withdrawal, transfer")
	}
	return nil
}

// ValidateTransfer validates transfer details
func (v *Validator) ValidateTransfer(fromAccountID, toAccountID uint, amount float64) error {
	if fromAccountID == toAccountID {
		return errors.NewValidationError("cannot transfer to the same account")
	}
	if err := v.ValidateAmount(amount); err != nil {
		return err
	}
	return nil
}

// ValidatePagination validates pagination parameters
func (v *Validator) ValidatePagination(page, limit int) error {
	if page < 1 {
		return errors.NewValidationError("page must be greater than 0")
	}
	if limit < 1 || limit > 100 {
		return errors.NewValidationError("limit must be between 1 and 100")
	}
	return nil
}
