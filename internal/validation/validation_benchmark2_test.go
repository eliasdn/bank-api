package validation

import (
	"testing"
)

func BenchmarkValidateAccountTypeSwitch(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		accountType := "savings"
		if accountType != "checking" && accountType != "savings" && accountType != "credit" {
			_ = false
		}
	}
}
