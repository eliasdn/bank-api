package handlers

import (
	"testing"
)

func BenchmarkValidatePassword(b *testing.B) {
	password := "Password123!"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validatePassword(password)
	}
}
