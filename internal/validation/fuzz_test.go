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
		"abc", strings.Repeat("u", 50), strings.Repeat("u", 51), strings.Repeat("u", 49),
		"user@name", "user name", "user#123", "こんにちは", "user_name_12345",
		"user\x00name", "user\r\n", "12345", "---", "___", "ab", "abcde",
		"user.name", "user+1", "USER_NAME_123", "a_b-c", "   user   ",
		"user_name!", "user_name$", "user\tname", "USER-123_abc",
		"user\nname", "usr", "us", "u_1", "A_B_C_D_E",
		"user_name_with_trailing_spaces  ", "  leading_space_user", "null\x00character",
		"emoji_user_😎", "special_chars_$%^&*", "very_long_username_that_is_exactly_50_chars_0123456",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, username string) {
		err := v.ValidateUsername(username)
		isValid := len(username) >= MinUsernameLength && len(username) <= MaxUsernameLength
		if isValid {
			for i := 0; i < len(username); i++ {
				c := username[i]
				if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
					isValid = false
					break
				}
			}
		}
		if (err == nil) != isValid {
			t.Errorf("ValidateUsername mismatch for %q: got err=%v, expected valid=%v", username, err, isValid)
		}
	})
}

func expectedValidEmail(email string) bool {
	email = strings.TrimSpace(strings.ToLower(email))
	if len(email) < MinEmailLength || len(email) > MaxEmailLength {
		return false
	}

	atIndex := strings.IndexByte(email, '@')
	if atIndex <= 0 || atIndex == len(email)-1 {
		return false
	}

	for i := 0; i < atIndex; i++ {
		c := email[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '.' || c == '_' || c == '%' || c == '+' || c == '-') {
			return false
		}
	}

	dotIndex := -1
	for i := atIndex + 1; i < len(email); i++ {
		c := email[i]
		if c == '.' {
			dotIndex = i
		} else if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-') {
			return false
		}
	}

	if dotIndex == -1 || dotIndex == atIndex+1 {
		return false
	}

	if len(email)-dotIndex-1 < 2 {
		return false
	}
	for i := dotIndex + 1; i < len(email); i++ {
		c := email[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
			return false
		}
	}

	return true
}

func FuzzValidateEmail(f *testing.F) {
	v := New()
	seeds := []string{
		"user@example.com", "test.user+tag@domain.co.uk", "invalid", "a@b.co", "",
		"@domain.com", "user@", "a@b.c", "  user@example.com  ",
		"user@domain..com", "user@domain.c", strings.Repeat("a", 245) + "@example.com",
		strings.Repeat("a", 250) + "@example.com", "user name@example.com",
		"user@domain.123", "user@domain.org", "TEST%USER+123@sub.domain.com",
		"user\x00@example.com", "user@ex\r\nample.com", "user@@domain.com",
		"user@domain", "user@domain.", "user@.domain.com", "user@domain_com",
		"u.s.e.r+123_45%67@sub-domain.example.org", "a@b.com", "a@b.c0m",
		"user\t@example.com", "user@domain.info", "user@sub.domain.co.uk",
		"user@sub-domain.domain.org", "plainaddress", "#@%^%#$@#$@#.com",
		"email@domain.com (Joe Smith)", "email@domain@domain.com", ".email@domain.com",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, email string) {
		err := v.ValidateEmail(email)
		expected := expectedValidEmail(email)
		if (err == nil) != expected {
			t.Errorf("ValidateEmail mismatch for %q: got err=%v, expected valid=%v", email, err, expected)
		}
	})
}

func expectedValidPassword(password string) bool {
	if len(password) < MinPasswordLength || len(password) > MaxPasswordLength {
		return false
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
	return hasUpper && hasLower && hasNumber && hasSpecial
}

func FuzzValidatePassword(f *testing.F) {
	v := New()
	seeds := []string{
		"Password123!", "short", "lowercase123!", "UPPERCASE123!", "NoNumber!", "NoSpecial123", "",
		"P1!aaaaa", strings.Repeat("A1!a", 25), strings.Repeat("A1!a", 26), strings.Repeat("A1!a", 24),
		"Password123\x00!", "Password 123!", "🔑Password123!", "Pass#123",
		"A1!aA1!a", "Abcdefg123$", "VeryLongPasswordWithSpecialChars!99",
		"P@ssw0rd2026", "12345678Aa!", "~~~~~Aa1!", "A1!a" + strings.Repeat("x", 96),
		"Password123\n!", "Password123\r!", "Password123\t!",
		"Aa1!4567", "1234567Aa!", "Aa1!Aa1!",
		"P@ssword1", "ComplexP@ssw0rd2026!", "P1#a2$b3", "password_NO_UPPER1!",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, password string) {
		err := v.ValidatePassword(password)
		expected := expectedValidPassword(password)
		if (err == nil) != expected {
			t.Errorf("ValidatePassword mismatch for %q: got err=%v, expected valid=%v", password, err, expected)
		}
	})
}

func expectedValidFullName(fullName string) bool {
	if len(fullName) < MinFullNameLength || len(fullName) > MaxFullNameLength {
		return false
	}
	if strings.TrimSpace(fullName) == "" {
		return false
	}
	for i := 0; i < len(fullName); i++ {
		c := fullName[i]
		if c == 0 || c == '\r' || c == '\n' {
			return false
		}
	}
	return true
}

func FuzzValidateFullName(f *testing.F) {
	v := New()
	seeds := []string{
		"John Doe", "A", "", "   ", "Jane Mary Smith-Doe", "\t\n",
		"Jo", strings.Repeat("a", 100), strings.Repeat("a", 101), strings.Repeat("a", 99),
		"John\x00Doe", "John 123", "Jean-Luc Pic'ard", "   John   ",
		"  A  ", "  AB  ", "Dr. Martin Luther King, Jr.",
		"José González", "María-José", "  John Doe  ", "\r\nJohn\r\n",
		"  John  Doe  ", "A B C D", "John\tDoe",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, fullName string) {
		err := v.ValidateFullName(fullName)
		expected := expectedValidFullName(fullName)
		if (err == nil) != expected {
			t.Errorf("ValidateFullName mismatch for %q: got err=%v, expected valid=%v", fullName, err, expected)
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
		{"valid_user", "valid@domain.org", "ValidPass123#", "Jane Doe"},
		{"bad-user!", "john@example.com", "Password123!", "John Doe"},
		{"john_doe", "john@example.com", "Password123!", "   "},
		{"admin", "admin@bank.com", "AdminSecret123!", "System Administrator"},
		{"user_123", "user123@test.io", "StrongP@ss1", "Test User"},
		{"john_doe", "john@example.com", "Password123\x00!", "John Doe"},
		{"john_doe", "john@ex\r\nample.com", "Password123!", "John Doe"},
		{"user_99", "user99@domain.org", "SecurePass99$", "User NinetyNine"},
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
	seeds := []string{
		"checking", "savings", "credit", "investment", "", "CHECKING", "checking ", " checking",
		"Checking", "Savings", "Credit", "other", "123", "credit\x00",
		"loan", "mortgage", "checking\n", "savings\r\n", "CREDIT",
		"SAVINGS", "check", "save",
	}
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

func expectedValidAmount(amount float64) bool {
	if math.IsNaN(amount) || math.IsInf(amount, 0) {
		return false
	}
	if amount <= 0 || amount > 1000000 {
		return false
	}
	if math.Abs(amount*100-math.Round(amount*100)) > 1e-6 {
		return false
	}
	return true
}

func FuzzValidateAmount(f *testing.F) {
	v := New()
	seeds := []float64{
		100.0, 10.55, 0.0, -10.0, 1000001.0, 0.001, 10.005,
		math.NaN(), math.Inf(1), math.Inf(-1),
		1000000.0, 0.01, -0.0, math.MaxFloat64, math.SmallestNonzeroFloat64,
		999999.99, 0.0001, 123.456, 1.0000001, -100.50,
		0.009, 0.010, 500000.00, 1000000.00, 1000000.001,
		-0.01, -1000000.0, 1e-7, 10.1234, 123456.78,
		0.00001, 100000.0, 1.0, 0.1, 0.05, 999.99,
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, amount float64) {
		err := v.ValidateAmount(amount)
		expected := expectedValidAmount(amount)
		if (err == nil) != expected {
			t.Errorf("ValidateAmount mismatch for %v: got err=%v, expected valid=%v", amount, err, expected)
		}
	})
}

func FuzzValidateTransactionType(f *testing.F) {
	v := New()
	seeds := []string{
		"deposit", "withdrawal", "transfer", "invalid", "", "DEPOSIT", "deposit ", "withdrawal ",
		"Deposit", "Withdrawal", "Transfer", "payment", "refund", "deposit\x00",
		"TRANSFER", "deposit\n", "withdrawal\r\n", "fee", "interest",
		"WITHDRAWAL", "dep", "with", "trans",
	}
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

func expectedValidDescription(description string) bool {
	if len(description) > MaxDescriptionLength {
		return false
	}
	for i := 0; i < len(description); i++ {
		c := description[i]
		if c == 0 || c == '\r' || c == '\n' {
			return false
		}
	}
	return true
}

func FuzzValidateDescription(f *testing.F) {
	v := New()
	seeds := []string{
		"Payment for rent", "Line 1\nLine 2", "Null\x00Byte", "",
		"Very long description that might exceed limit",
		strings.Repeat("d", 255), strings.Repeat("d", 256), "Line 1\rLine 2",
		"Normal payment", "Payment\r", "Payment\n", "\x00",
		strings.Repeat("a", 254), "Emoji 💰 test",
		"Payment\r\nwith newline", "Description with\tTab", "Description\x00Null",
		strings.Repeat("🎉", 64), strings.Repeat("x", 255) + "\n",
		"Grocery store #1024", "Salary payment - March 2026",
		"Special chars !@#$%^&*()", "Unicode test こんにちは", "Control char \x01 test",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, description string) {
		err := v.ValidateDescription(description)
		expected := expectedValidDescription(description)
		if (err == nil) != expected {
			t.Errorf("ValidateDescription mismatch for %q: got err=%v, expected valid=%v", description, err, expected)
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
		{10, 20, 500.25, "Valid transfer"},
		{5, 5, 50.0, "Self transfer"},
		{1, 2, 1000000.0, "Max transfer"},
		{1, 2, 1000000.01, "Exceed max transfer"},
		{1, 2, 100.0, "Payment\nNewLine"},
		{1, 2, 100.0, "Payment\x00Null"},
		{0, 1, 50.0, "Transfer from ID 0"},
		{1, 0, 50.0, "Transfer to ID 0"},
		{100, 200, 250.75, "Gift transfer"},
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
		{"000", 0},
		{"123456789012345678", 1},
		{"-123", 10},
		{" 10 ", 10},
		{"1234567890123456789", 10},
		{"9223372036854775807", 10},
		{"-0", 5},
		{"+10", 5},
		{"10.5", 1},
		{"100", 20},
		{"0000", 0},
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
		{1000, 50},
		{-100, 10},
		{1, -10},
		{0, 0},
		{1, 100},
		{100, 1},
		{2147483647, 100},
		{-2147483648, 10},
		{5, 20},
		{10, 50},
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
		{" 123", 5},
		{"123 ", 5},
		{"-0", 0},
		{"000000000000000000", 0},
		{"+123", 10},
		{"9223372036854775807", 10},
		{"18446744073709551615", 10},
		{"123.45", 5},
		{"500", 100},
		{"00000", 0},
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
