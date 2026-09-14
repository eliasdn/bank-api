package validation

import "testing"

func BenchmarkValidatePassword(b *testing.B) {
	v := New()
	password := "StrongP@ssw0rd!"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.ValidatePassword(password)
	}
}
