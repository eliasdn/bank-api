package validation

import "testing"

func BenchmarkValidateEmail(b *testing.B) {
	v := New()
	email := "test.user_name+tag@example.com"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.ValidateEmail(email)
	}
}
