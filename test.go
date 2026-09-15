package main

import (
	"fmt"
	"strings"
	"time"
)

func main() {
	password := "ThisIsATestPassword123!"

	start1 := time.Now()
	for i := 0; i < 1000000; i++ {
		_ = strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
		_ = strings.ContainsAny(password, "abcdefghijklmnopqrstuvwxyz")
		_ = strings.ContainsAny(password, "0123456789")
		_ = strings.ContainsAny(password, "!@#$%^&*()_+-=[]{}|;:,.<>?")
	}
	fmt.Printf("ContainsAny: %v\n", time.Since(start1))

	start2 := time.Now()
	for i := 0; i < 1000000; i++ {
		var hasUpper, hasLower, hasDigit, hasSpecial bool
		for j := 0; j < len(password); j++ {
			c := password[j]
			switch {
			case c >= 'A' && c <= 'Z':
				hasUpper = true
			case c >= 'a' && c <= 'z':
				hasLower = true
			case c >= '0' && c <= '9':
				hasDigit = true
			case strings.IndexByte("!@#$%^&*()_+-=[]{}|;:,.<>?", c) >= 0:
				hasSpecial = true
			}
		}
		_, _, _, _ = hasUpper, hasLower, hasDigit, hasSpecial
	}
	fmt.Printf("IndexByte: %v\n", time.Since(start2))
}
