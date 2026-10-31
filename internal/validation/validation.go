package validation

import (
	"bank-api/internal/errors"
	"math"
	"strings"
)

// Validation rules
const (
	MinUsernameLength    = 3
	MaxUsernameLength    = 50
	MinPasswordLength    = 8
	MaxPasswordLength    = 100
	MinFullNameLength    = 2
	MaxFullNameLength    = 100
	MinEmailLength       = 5
	MaxEmailLength       = 255
	MaxDescriptionLength = 255
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
	// Optimization: Fast path evaluates common valid characters first to avoid unnecessary edge case checks (null bytes/newlines) on every iteration.
	for i := 0; i < len(username); i++ {
		c := username[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' {
			continue
		}
		if c == 0 || c == '\r' || c == '\n' {
			return errors.NewValidationError("username cannot contain null bytes or line breaks")
		}
		return errors.NewValidationError("username can only contain letters, numbers, underscores, and hyphens")
	}
	return nil
}

// ValidateEmail validates email format and rejects null bytes and line breaks to prevent CRLF/null-byte injection
func (v *Validator) ValidateEmail(email string) error {
	// Optimization: strings.TrimSpace only creates a slice header (no allocation).
	// Avoid strings.ToLower since the manual byte loop below checks both cases.
	email = strings.TrimSpace(email)
	if len(email) < MinEmailLength || len(email) > MaxEmailLength {
		return errors.NewValidationError("email must be between %d and %d characters", MinEmailLength, MaxEmailLength)
	}

	// Fast path manual email validation
	// Optimization: Merged null byte check and replaced strings.IndexByte with manual search to avoid function call overhead and multiple string iterations.
	atIndex := -1
	// Optimization: Fast path evaluates common valid characters first to avoid unnecessary edge case checks (null bytes/newlines) on every iteration.
	for i := 0; i < len(email); i++ {
		c := email[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '.' || c == '_' || c == '%' || c == '+' || c == '-' {
			continue
		}
		if c == '@' {
			atIndex = i
			break
		}
		if c == 0 || c == '\r' || c == '\n' {
			return errors.NewValidationError("email cannot contain null bytes or line breaks")
		}
		return errors.NewValidationError("invalid email format")
	}

	if atIndex <= 0 || atIndex == len(email)-1 {
		return errors.NewValidationError("invalid email format")
	}

	dotIndex := -1
	for i := atIndex + 1; i < len(email); i++ {
		c := email[i]
		if c == 0 || c == '\r' || c == '\n' {
			return errors.NewValidationError("email cannot contain null bytes or line breaks")
		}
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

// ValidatePassword validates password strength and rejects null bytes and line breaks to prevent truncation and injection vulnerabilities
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
			continue
		case c >= 'a' && c <= 'z':
			hasLower = true
			continue
		case c >= '0' && c <= '9':
			hasNumber = true
			continue
		case c >= 33 && c <= 126:
			// Accepts all printable ASCII special characters (ASCII 33-126 non-alphanumeric) per OWASP guidelines
			hasSpecial = true
			continue
		}
		if c == 0 || c == '\r' || c == '\n' {
			return errors.NewValidationError("password cannot contain null bytes or line breaks")
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

// ValidateFullName validates full name format and rejects null bytes and line breaks to prevent CRLF/null-byte injection
func (v *Validator) ValidateFullName(fullName string) error {
	if len(fullName) < MinFullNameLength || len(fullName) > MaxFullNameLength {
		return errors.NewValidationError("full name must be between %d and %d characters", MinFullNameLength, MaxFullNameLength)
	}
	if strings.TrimSpace(fullName) == "" {
		return errors.NewValidationError("full name cannot be empty")
	}
	for i := 0; i < len(fullName); i++ {
		c := fullName[i]
		if c == 0 || c == '\r' || c == '\n' {
			return errors.NewValidationError("full name cannot contain null bytes or line breaks")
		}
	}
	return nil
}

// ValidateAccountType validates account type
func (v *Validator) ValidateAccountType(accountType string) error {
	switch accountType {
	case "checking", "savings", "credit":
		return nil
	default:
		return errors.NewValidationError("account type must be one of: checking, savings, credit")
	}
}

// ValidateAmount validates transaction amount
func (v *Validator) ValidateAmount(amount float64) error {
	if math.IsNaN(amount) || math.IsInf(amount, 0) {
		return errors.NewValidationError("amount must be a valid finite number")
	}
	if amount <= 0 {
		return errors.NewValidationError("amount must be greater than zero")
	}
	if amount > 1000000 { // $1M limit
		return errors.NewValidationError("amount exceeds maximum allowed limit")
	}
	// Ensure amount has at most 2 decimal places (cents) to prevent sub-cent/salami-slicing attacks
	if math.Abs(amount*100-math.Round(amount*100)) > 1e-6 {
		return errors.NewValidationError("amount cannot have more than 2 decimal places")
	}
	return nil
}

// ValidateTransactionType validates transaction type
func (v *Validator) ValidateTransactionType(transactionType string) error {
	switch transactionType {
	case "deposit", "withdrawal", "transfer":
		return nil
	default:
		return errors.NewValidationError("transaction type must be one of: deposit, withdrawal, transfer")
	}
}

// ValidateDescription validates transaction/transfer description length and rejects null bytes and line breaks to prevent CRLF/null-byte injection
func (v *Validator) ValidateDescription(description string) error {
	if len(description) > MaxDescriptionLength {
		return errors.NewValidationError("description cannot exceed %d characters", MaxDescriptionLength)
	}
	for i := 0; i < len(description); i++ {
		c := description[i]
		if c == 0 || c == '\r' || c == '\n' {
			return errors.NewValidationError("description cannot contain null bytes or line breaks")
		}
	}
	return nil
}

// ValidateTransfer validates transfer details including description length
func (v *Validator) ValidateTransfer(fromAccountID, toAccountID uint, amount float64, description string) error {
	if fromAccountID == 0 || toAccountID == 0 {
		return errors.NewValidationError("account ID must be valid")
	}
	if fromAccountID == toAccountID {
		return errors.NewValidationError("cannot transfer to the same account")
	}
	if err := v.ValidateAmount(amount); err != nil {
		return err
	}
	if err := v.ValidateDescription(description); err != nil {
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

// ParseInt is a highly optimized manual byte loop for parsing positive integers
// from strings (e.g. query parameters). It avoids the reflection and function
// call overhead of strconv.Atoi. If the string is empty or contains non-numeric
// characters, or if it would cause an integer overflow, it returns the provided default value.
func ParseInt(s string, def int) int {
	if s == "" || len(s) > 18 {
		return def
	}
	res := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return def
		}
		res = res*10 + int(c-'0')
	}
	if res < 0 {
		return def
	}
	return res
}

// SanitizeUserAgent truncates user-agent strings to max 512 characters
// and replaces control characters (ASCII < 32 or 127) including null bytes,
// carriage returns, and newlines with spaces to prevent CRLF log injection (CWE-117).
func SanitizeUserAgent(ua string) string {
	if len(ua) > 512 {
		ua = ua[:512]
	}

	// Optimization: Fast path to check if modification is needed before allocating.
	modified := false
	for i := 0; i < len(ua); i++ {
		c := ua[i]
		if c < 32 || c == 127 {
			modified = true
			break
		}
	}

	if !modified {
		return ua
	}

	// Optimization: Use fixed-size stack array to prevent heap allocation.
	var b [512]byte
	n := copy(b[:], ua)
	for i := 0; i < n; i++ {
		c := b[i]
		if c < 32 || c == 127 {
			b[i] = ' '
		}
	}
	return string(b[:n])
}
