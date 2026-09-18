package validation

import (
	"math"
	"strconv"
	"strings"
	"testing"
)

func FuzzValidateUsername(f *testing.F) {
	v := New()
	seeds := []string{
		"john_doe", "admin123", "a", "", "user-name",
		"super_long_username_that_exceeds_fifty_characters_limit_by_far",
		"abc", strings.Repeat("u", 50), strings.Repeat("u", 51),
		"user@name", "user name", "user#123", "こんにちは",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, username string) {
		err := v.ValidateUsername(username)
		if err == nil {
			if len(username) < MinUsernameLength || len(username) > MaxUsernameLength {
				t.Errorf("ValidateUsername allowed invalid length %d for %q", len(username), username)
			}
			for i := 0; i < len(username); i++ {
				c := username[i]
				if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
					t.Errorf("ValidateUsername allowed invalid character %q in %q", c, username)
				}
			}
		}
	})
}

func FuzzValidateEmail(f *testing.F) {
	v := New()
	seeds := []string{
		"user@example.com", "test.user+tag@domain.co.uk", "invalid", "a@b.c", "",
		"@domain.com", "user@", "a@b.co", "  user@example.com  ",
		"user@domain..com", "user@domain.c", strings.Repeat("a", 245) + "@example.com",
		strings.Repeat("a", 250) + "@example.com",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, email string) {
		err := v.ValidateEmail(email)
		if err == nil {
			trimmed := strings.TrimSpace(strings.ToLower(email))
			if len(trimmed) < MinEmailLength || len(trimmed) > MaxEmailLength {
				t.Errorf("ValidateEmail allowed invalid length %d for %q", len(trimmed), email)
			}
			if !strings.Contains(trimmed, "@") || !strings.Contains(trimmed, ".") {
				t.Errorf("ValidateEmail allowed email without @ or .: %q", email)
			}
		}
	})
}

func FuzzValidatePassword(f *testing.F) {
	v := New()
	seeds := []string{
		"Password123!", "short", "lowercase123!", "UPPERCASE123!", "NoNumber!", "NoSpecial123", "",
		"P1!aaaaa", strings.Repeat("A1!a", 25), strings.Repeat("A1!a", 26),
		"Password123\x00!", "Password 123!", "🔑Password123!",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, password string) {
		err := v.ValidatePassword(password)
		if err == nil {
			if len(password) < MinPasswordLength || len(password) > MaxPasswordLength {
				t.Errorf("ValidatePassword allowed invalid length %d for %q", len(password), password)
			}
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
			if !hasUpper || !hasLower || !hasNumber || !hasSpecial {
				t.Errorf("ValidatePassword allowed password missing required character categories: %q", password)
			}
		}
	})
}

func FuzzValidateFullName(f *testing.F) {
	v := New()
	seeds := []string{
		"John Doe", "A", "", "   ", "Jane Mary Smith-Doe", "\t\n",
		"Jo", strings.Repeat("a", 100), strings.Repeat("a", 101),
		"John\x00Doe", "John 123", "Jean-Luc Pic'ard",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, fullName string) {
		err := v.ValidateFullName(fullName)
		if err == nil {
			if len(fullName) < MinFullNameLength || len(fullName) > MaxFullNameLength {
				t.Errorf("ValidateFullName allowed invalid length %d for %q", len(fullName), fullName)
			}
			if strings.TrimSpace(fullName) == "" {
				t.Errorf("ValidateFullName allowed whitespace-only full name %q", fullName)
			}
		}
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
		{"john_doe", "invalid-email", "Password123!", "John Doe"},
		{"john_doe", "john@example.com", "short", "John Doe"},
		{"john_doe", "john@example.com", "Password123!", "J"},
	}
	for _, s := range seeds {
		f.Add(s.username, s.email, s.password, s.fullName)
	}

	f.Fuzz(func(t *testing.T, username, email, password, fullName string) {
		err := v.ValidateUserRegistration(username, email, password, fullName)
		uErr := v.ValidateUsername(username)
		eErr := v.ValidateEmail(email)
		pErr := v.ValidatePassword(password)
		fnErr := v.ValidateFullName(fullName)

		expectValid := (uErr == nil) && (eErr == nil) && (pErr == nil) && (fnErr == nil)
		if (err == nil) != expectValid {
			t.Errorf("ValidateUserRegistration mismatch for (%q, %q, %q, %q): got err=%v, expected valid=%v",
				username, email, password, fullName, err, expectValid)
		}
	})
}

func FuzzValidateAccountType(f *testing.F) {
	v := New()
	seeds := []string{"checking", "savings", "credit", "investment", "", "CHECKING", "checking ", " checking"}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, accountType string) {
		err := v.ValidateAccountType(accountType)
		isAllowed := accountType == "checking" || accountType == "savings" || accountType == "credit"
		if (err == nil) != isAllowed {
			t.Errorf("ValidateAccountType mismatch for %q: got err=%v, expected allowed=%v", accountType, err, isAllowed)
		}
	})
}

func FuzzValidateAmount(f *testing.F) {
	v := New()
	seeds := []float64{
		100.0, 10.55, 0.0, -10.0, 1000001.0, 0.001, 10.005,
		math.NaN(), math.Inf(1), math.Inf(-1),
		1000000.0, 0.01, -0.0, math.MaxFloat64, math.SmallestNonzeroFloat64,
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, amount float64) {
		err := v.ValidateAmount(amount)
		if err == nil {
			if math.IsNaN(amount) || math.IsInf(amount, 0) {
				t.Errorf("ValidateAmount allowed non-finite float %v", amount)
			}
			if amount <= 0 || amount > 1000000 {
				t.Errorf("ValidateAmount allowed out of range amount %v", amount)
			}
			if math.Abs(amount*100-math.Round(amount*100)) > 1e-6 {
				t.Errorf("ValidateAmount allowed sub-cent precision amount %v", amount)
			}
		}
	})
}

func FuzzValidateTransactionType(f *testing.F) {
	v := New()
	seeds := []string{"deposit", "withdrawal", "transfer", "invalid", "", "DEPOSIT", "deposit ", "withdrawal "}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, transactionType string) {
		err := v.ValidateTransactionType(transactionType)
		isAllowed := transactionType == "deposit" || transactionType == "withdrawal" || transactionType == "transfer"
		if (err == nil) != isAllowed {
			t.Errorf("ValidateTransactionType mismatch for %q: got err=%v, expected allowed=%v", transactionType, err, isAllowed)
		}
	})
}

func FuzzValidateDescription(f *testing.F) {
	v := New()
	seeds := []string{
		"Payment for rent", "Line 1\nLine 2", "Null\x00Byte", "",
		"Very long description that might exceed limit",
		strings.Repeat("d", 255), strings.Repeat("d", 256), "Line 1\rLine 2",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, description string) {
		err := v.ValidateDescription(description)
		if err == nil {
			if len(description) > MaxDescriptionLength {
				t.Errorf("ValidateDescription allowed description length %d > %d", len(description), MaxDescriptionLength)
			}
			if strings.ContainsAny(description, "\x00\r\n") {
				t.Errorf("ValidateDescription allowed illegal control character in %q", description)
			}
		}
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
		{1, 2, 0.001, "Subcent"},
		{1, 2, 100.0, strings.Repeat("x", 256)},
	}
	for _, s := range seeds {
		f.Add(s.fromID, s.toID, s.amount, s.description)
	}

	f.Fuzz(func(t *testing.T, fromID, toID uint, amount float64, description string) {
		err := v.ValidateTransfer(fromID, toID, amount, description)
		amtErr := v.ValidateAmount(amount)
		descErr := v.ValidateDescription(description)
		expectValid := (fromID != toID) && (amtErr == nil) && (descErr == nil)

		if (err == nil) != expectValid {
			t.Errorf("ValidateTransfer mismatch for (%d, %d, %v, %q): got err=%v, expected valid=%v",
				fromID, toID, amount, description, err, expectValid)
		}
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
		{"0007", 7},
		{"9999999999999999999", 10},
		{"123a", 10},
	}
	for _, s := range seeds {
		f.Add(s.s, s.def)
	}

	f.Fuzz(func(t *testing.T, s string, def int) {
		res := ParsePaginationParam(s, def)
		if s == "" || len(s) > 18 {
			if res != def {
				t.Errorf("ParsePaginationParam for %q with def %d returned %d, expected default %d", s, def, res, def)
			}
			return
		}
		allDigits := true
		for i := 0; i < len(s); i++ {
			if s[i] < '0' || s[i] > '9' {
				allDigits = false
				break
			}
		}
		if !allDigits {
			if res != def {
				t.Errorf("ParsePaginationParam non-digit string %q returned %d, expected default %d", s, res, def)
			}
			return
		}
		if expected, err := strconv.Atoi(s); err == nil && expected >= 0 {
			if res != expected {
				t.Errorf("ParsePaginationParam mismatch for %q: got %d, expected %d", s, res, expected)
			}
		}
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
		{1, 1},
		{100, 100},
		{2, 50},
	}
	for _, s := range seeds {
		f.Add(s.page, s.limit)
	}

	f.Fuzz(func(t *testing.T, page, limit int) {
		err := v.ValidatePagination(page, limit)
		expectValid := (page >= 1) && (limit >= 1 && limit <= 100)
		if (err == nil) != expectValid {
			t.Errorf("ValidatePagination mismatch for page=%d, limit=%d: got err=%v, expected valid=%v",
				page, limit, err, expectValid)
		}
	})
}

func FuzzParseInt(f *testing.F) {
	type seed struct {
		s   string
		def int
	}
	seeds := []seed{
		{"123", 10},
		{"0", 0},
		{"99999999999999999999", -1},
		{"-10", 5},
		{"abc", 20},
		{"", 0},
		{"0007", 7},
		{"00", 0},
		{"123456789012345678", 0},
		{"1234567890123456789", 0},
	}
	for _, s := range seeds {
		f.Add(s.s, s.def)
	}

	f.Fuzz(func(t *testing.T, s string, def int) {
		res := ParseInt(s, def)
		if s == "" || len(s) > 18 {
			if res != def {
				t.Errorf("ParseInt for %q with def %d returned %d, expected default %d", s, def, res, def)
			}
			return
		}
		allDigits := true
		for i := 0; i < len(s); i++ {
			if s[i] < '0' || s[i] > '9' {
				allDigits = false
				break
			}
		}
		if !allDigits {
			if res != def {
				t.Errorf("ParseInt non-digit string %q returned %d, expected default %d", s, res, def)
			}
			return
		}
		if expected, err := strconv.Atoi(s); err == nil && expected >= 0 {
			if res != expected {
				t.Errorf("ParseInt mismatch for %q: got %d, expected %d", s, res, expected)
			}
		}
	})
}
