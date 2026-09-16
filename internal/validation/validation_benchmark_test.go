package validation

import (
	"testing"
)

func BenchmarkValidateAccountType(b *testing.B) {
	v := New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.ValidateAccountType("savings")
	}
}

func BenchmarkValidateTransactionType(b *testing.B) {
	v := New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.ValidateTransactionType("deposit")
	}
}
