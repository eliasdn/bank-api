package validation

import (
	"math"
	"testing"
)

func FuzzValidateUsername(f *testing.F) {
	v := New()
	seeds := []string{"john_doe", "admin123", "a", "", "user-name", "super_long_username_that_exceeds_fifty_characters_limit_by_far"}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, username string) {
		_ = v.ValidateUsername(username)
	})
}

func FuzzValidateEmail(f *testing.F) {
	v := New()
	seeds := []string{"user@example.com", "test.user+tag@domain.co.uk", "invalid", "a@b.c", "", "@domain.com", "user@"}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, email string) {
		_ = v.ValidateEmail(email)
	})
}

func FuzzValidatePassword(f *testing.F) {
	v := New()
	seeds := []string{"Password123!", "short", "lowercase123!", "UPPERCASE123!", "NoNumber!", "NoSpecial123", ""}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, password string) {
		_ = v.ValidatePassword(password)
	})
}

func FuzzValidateFullName(f *testing.F) {
	v := New()
	seeds := []string{"John Doe", "A", "", "   ", "Jane Mary Smith-Doe"}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, fullName string) {
		_ = v.ValidateFullName(fullName)
	})
}

func FuzzValidateUserRegistration(f *testing.F) {
	v := New()
	type seed struct {
		username string
		email    string
		password string
		fullName string
	}
	seeds := []seed{
		{"john_doe", "john@example.com", "Password123!", "John Doe"},
		{"a", "invalid", "short", ""},
		{"", "", "", ""},
	}
	for _, s := range seeds {
		f.Add(s.username, s.email, s.password, s.fullName)
	}

	f.Fuzz(func(t *testing.T, username, email, password, fullName string) {
		_ = v.ValidateUserRegistration(username, email, password, fullName)
	})
}

func FuzzValidateAccountType(f *testing.F) {
	v := New()
	seeds := []string{"checking", "savings", "credit", "investment", "", "CHECKING"}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, accountType string) {
		_ = v.ValidateAccountType(accountType)
	})
}

func FuzzValidateAmount(f *testing.F) {
	v := New()
	seeds := []float64{100.0, 10.55, 0.0, -10.0, 1000001.0, 0.001, 10.005, math.NaN(), math.Inf(1), math.Inf(-1)}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, amount float64) {
		_ = v.ValidateAmount(amount)
	})
}

func FuzzValidateTransactionType(f *testing.F) {
	v := New()
	seeds := []string{"deposit", "withdrawal", "transfer", "invalid", "", "DEPOSIT"}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, transactionType string) {
		_ = v.ValidateTransactionType(transactionType)
	})
}

func FuzzValidateDescription(f *testing.F) {
	v := New()
	seeds := []string{"Payment for rent", "Line 1\nLine 2", "Null\x00Byte", "", "Very long description that might exceed limit"}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, description string) {
		_ = v.ValidateDescription(description)
	})
}

func FuzzValidateTransfer(f *testing.F) {
	v := New()
	type seed struct {
		fromID      uint
		toID        uint
		amount      float64
		description string
	}
	seeds := []seed{
		{1, 2, 100.0, "Rent"},
		{1, 1, 100.0, "Same account"},
		{1, 2, -10.0, "Negative"},
		{0, 0, 0.0, ""},
	}
	for _, s := range seeds {
		f.Add(s.fromID, s.toID, s.amount, s.description)
	}

	f.Fuzz(func(t *testing.T, fromID, toID uint, amount float64, description string) {
		_ = v.ValidateTransfer(fromID, toID, amount, description)
	})
}

func FuzzParsePaginationParam(f *testing.F) {
	seeds := []struct {
		s   string
		def int
	}{
		{"10", 1},
		{"0", 10},
		{"-5", 1},
		{"abc", 20},
		{"", 5},
	}
	for _, s := range seeds {
		f.Add(s.s, s.def)
	}

	f.Fuzz(func(t *testing.T, s string, def int) {
		_ = ParsePaginationParam(s, def)
	})
}

func FuzzValidatePagination(f *testing.F) {
	v := New()
	seeds := []struct {
		page  int
		limit int
	}{
		{1, 10},
		{0, 10},
		{1, 0},
		{1, 101},
		{-1, -1},
	}
	for _, s := range seeds {
		f.Add(s.page, s.limit)
	}

	f.Fuzz(func(t *testing.T, page, limit int) {
		_ = v.ValidatePagination(page, limit)
	})
}

func FuzzParseInt(f *testing.F) {
	seeds := []string{"123", "0", "99999999999999999999", "-10", "abc", "", "0007"}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, s string) {
		_ = ParseInt(s, 10)
	})
}
