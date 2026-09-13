package handlers

import (
	"testing"
)

func BenchmarkGenerateAccountNumber(b *testing.B) {
	for i := 0; i < b.N; i++ {
		generateAccountNumber()
	}
}
