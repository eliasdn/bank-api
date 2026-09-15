package main

import (
	"regexp"
	"strings"
	"testing"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func IsValidEmailRegex(email string) bool {
	return emailRegex.MatchString(email)
}

func IsValidEmailManual(email string) bool {
	atIdx := strings.IndexByte(email, '@')
	if atIdx <= 0 || atIdx == len(email)-1 {
		return false
	}

	for i := 0; i < atIdx; i++ {
		c := email[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '.' || c == '_' || c == '%' || c == '+' || c == '-') {
			return false
		}
	}

	domainPart := email[atIdx+1:]
	dotIdx := strings.LastIndexByte(domainPart, '.')
	if dotIdx <= 0 || dotIdx == len(domainPart)-1 {
		return false
	}

	for i := 0; i < dotIdx; i++ {
		c := domainPart[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '.' || c == '-') {
			return false
		}
	}

	tld := domainPart[dotIdx+1:]
	if len(tld) < 2 {
		return false
	}
	for i := 0; i < len(tld); i++ {
		c := tld[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
			return false
		}
	}

	return true
}

func TestEmail(t *testing.T) {
	emails := []string{
		"test@example.com",
		"test.user@example.co.uk",
		"a@b.cc",
		"test@example", // invalid
		"test@.com", // invalid
		"@example.com", // invalid
		"test@example.c", // invalid
		"test@example..com", // valid by regex! let's see
		"te@st@example.com", // invalid
	}

	for _, e := range emails {
		r := IsValidEmailRegex(e)
		m := IsValidEmailManual(e)
		if r != m {
			t.Errorf("Mismatch for %q: regex=%v, manual=%v", e, r, m)
		}
	}
}
