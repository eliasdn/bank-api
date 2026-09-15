package validation

import (
	"bank-api/internal/errors"
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

	// Validate using a manual byte loop to avoid regex overhead in hot paths
	for i := 0; i < len(username); i++ {
		c := username[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
			return errors.NewValidationError("username can only contain letters, numbers, underscores, and hyphens")
		}
	}
	return nil
}

// ValidateEmail validates email format
func (v *Validator) ValidateEmail(email string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	if len(email) < MinEmailLength || len(email) > MaxEmailLength {
		return errors.NewValidationError("email must be between %d and %d characters", MinEmailLength, MaxEmailLength)
	}

	// Fast path manual email validation
	atIndex := strings.IndexByte(email, '@')
	if atIndex <= 0 || atIndex == len(email)-1 {
		return errors.NewValidationError("invalid email format")
	}

	for i := 0; i < atIndex; i++ {
		c := email[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '.' || c == '_' || c == '%' || c == '+' || c == '-') {
			return errors.NewValidationError("invalid email format")
		}
	}

	dotIndex := -1
	for i := atIndex + 1; i < len(email); i++ {
		c := email[i]
		if c == '.' {
			dotIndex = i
		} else if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-') {
			return errors.NewValidationError("invalid email format")
		}
	}

	if dotIndex == -1 || dotIndex == atIndex+1 {
		return errors.NewValidationError("invalid email format")
	}

	if len(email)-dotIndex-1 < 2 {
		return errors.NewValidationError("invalid email format")
	}
	for i := dotIndex + 1; i < len(email); i++ {
		c := email[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
			return errors.NewValidationError("invalid email format")
		}
	}

	return nil
}

// ValidatePassword validates password strength
func (v *Validator) ValidatePassword(password string) error {
	if len(password) < MinPasswordLength || len(password) > MaxPasswordLength {
		return errors.NewValidationError("password must be between %d and %d characters", MinPasswordLength, MaxPasswordLength)
	}

	// Optimization: Manual byte loop is much faster than strings.ContainsAny calls in hot paths
	var hasUpper, hasLower, hasNumber, hasSpecial bool
	for i := 0; i < len(password); i++ {
		c := password[i]
		switch {
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= 'a' && c <= 'z':
			hasLower = true
		case c >= '0' && c <= '9':
			hasNumber = true
		case strings.IndexByte(`!@#$%^&*(),.?":{}|<>`, c) >= 0:
			hasSpecial = true
		}
	}

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
