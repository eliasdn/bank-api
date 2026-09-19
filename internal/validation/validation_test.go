package validation

import (
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateUsername(t *testing.T) {
	v := New()

	tests := []struct {
		name     string
		username string
		wantErr  bool
	}{
		{"valid username", "john_doe", false},
		{"valid username with numbers", "john123", false},
		{"valid username with hyphen", "john-doe", false},
		{"too short", "jo", true},
		{"too long", strings.Repeat("a", 51), true},
		{"invalid characters", "john@doe", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateUsername(tt.username)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	v := New()

	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{"valid email", "john.doe@example.com", false},
		{"valid email short", "a@b.co", false},
		{"too short", "a@b.", true},
		{"too long", strings.Repeat("a", 245) + "@example.com", true},
		{"invalid format - no @", "johndoe.com", true},
		{"invalid format - no domain", "john@", true},
		{"invalid format - no tld", "john@doe", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateEmail(tt.email)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	v := New()

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"valid password", "Password123!", false},
		{"too short", "Pass1!", true},
		{"no uppercase", "password123!", true},
		{"no lowercase", "PASSWORD123!", true},
		{"no number", "Password!", true},
		{"no special char", "Password123", true},
		{"too long", strings.Repeat("A", 101), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidatePassword(tt.password)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateFullName(t *testing.T) {
	v := New()

	tests := []struct {
		name     string
		fullName string
		wantErr  bool
	}{
		{"valid full name", "John Doe", false},
		{"too short", "J", true},
		{"too long", strings.Repeat("a", 101), true},
		{"empty", "", true},
		{"whitespace only", "   ", true},
		{"full name with null byte", "John\x00Doe", true},
		{"full name with newline", "John\nDoe", true},
		{"full name with carriage return", "John\rDoe", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateFullName(tt.fullName)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateUserRegistration(t *testing.T) {
	v := New()

	tests := []struct {
		name     string
		username string
		email    string
		password string
		fullName string
		wantErr  bool
	}{
		{"valid registration", "john_doe", "john@example.com", "Password123!", "John Doe", false},
		{"invalid username", "j", "john@example.com", "Password123!", "John Doe", true},
		{"invalid email", "john_doe", "invalid-email", "Password123!", "John Doe", true},
		{"invalid password", "john_doe", "john@example.com", "pass", "John Doe", true},
		{"invalid full name", "john_doe", "john@example.com", "Password123!", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateUserRegistration(tt.username, tt.email, tt.password, tt.fullName)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateAccountType(t *testing.T) {
	v := New()

	tests := []struct {
		name        string
		accountType string
		wantErr     bool
	}{
		{"valid checking", "checking", false},
		{"valid savings", "savings", false},
		{"valid credit", "credit", false},
		{"invalid type", "investment", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateAccountType(tt.accountType)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateAmount(t *testing.T) {
	v := New()

	tests := []struct {
		name    string
		amount  float64
		wantErr bool
	}{
		{"valid amount", 100.0, false},
		{"valid cents amount", 10.55, false},
		{"zero amount", 0, true},
		{"negative amount", -10.0, true},
		{"too large amount", 1000001.0, true},
		{"sub-cent amount", 0.001, true},
		{"fractional cent amount", 10.005, true},
		{"NaN amount", math.NaN(), true},
		{"positive infinity amount", math.Inf(1), true},
		{"negative infinity amount", math.Inf(-1), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateAmount(tt.amount)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateTransactionType(t *testing.T) {
	v := New()

	tests := []struct {
		name            string
		transactionType string
		wantErr         bool
	}{
		{"valid deposit", "deposit", false},
		{"valid withdrawal", "withdrawal", false},
		{"valid transfer", "transfer", false},
		{"invalid type", "payment", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateTransactionType(tt.transactionType)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateDescription(t *testing.T) {
	v := New()

	tests := []struct {
		name        string
		description string
		wantErr     bool
	}{
		{"valid description", "Rent payment for March", false},
		{"empty description", "", false},
		{"too long description", strings.Repeat("a", 256), true},
		{"description with null byte", "Payment\x00for item", true},
		{"description with newline", "Payment\nfor item", true},
		{"description with carriage return", "Payment\rfor item", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateDescription(tt.description)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateTransfer(t *testing.T) {
	v := New()

	tests := []struct {
		name          string
		fromAccountID uint
		toAccountID   uint
		amount        float64
		description   string
		wantErr       bool
	}{
		{"valid transfer", 1, 2, 100.0, "Payment", false},
		{"same account", 1, 1, 100.0, "Payment", true},
		{"invalid amount", 1, 2, -10.0, "Payment", true},
		{"description too long", 1, 2, 100.0, strings.Repeat("x", 256), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateTransfer(tt.fromAccountID, tt.toAccountID, tt.amount, tt.description)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePagination(t *testing.T) {
	v := New()

	tests := []struct {
		name    string
		page    int
		limit   int
		wantErr bool
	}{
		{"valid pagination", 1, 10, false},
		{"invalid page", 0, 10, true},
		{"invalid limit - too small", 1, 0, true},
		{"invalid limit - too large", 1, 101, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidatePagination(tt.page, tt.limit)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
