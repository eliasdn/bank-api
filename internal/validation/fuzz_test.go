package validation

import (
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

func FuzzParseInt(f *testing.F) {
	seeds := []string{"123", "0", "99999999999999999999", "-10", "abc", "", "0007"}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, s string) {
		_ = ParseInt(s, 10)
	})
}
