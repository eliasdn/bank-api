package validation

import (
	"strconv"
	"testing"
)

func TestAppendFloatAmount(t *testing.T) {
	tests := []struct {
		amount float64
		want   string
	}{
		{0, "0.00"},
		{0.00, "0.00"},
		{0.01, "0.01"},
		{0.1, "0.10"},
		{0.10, "0.10"},
		{0.99, "0.99"},
		{1, "1.00"},
		{1.0, "1.00"},
		{1.01, "1.01"},
		{1.1, "1.10"},
		{10, "10.00"},
		{12.34, "12.34"},
		{123.45, "123.45"},
		{1234.56, "1234.56"},
		{12345.67, "12345.67"},
		{123456.78, "123456.78"},
		{1234567.89, "1234567.89"},
		{-0.01, "-0.01"},
		{-1234.56, "-1234.56"},
	}

	for _, tt := range tests {
		t.Run(strconv.FormatFloat(tt.amount, 'f', 2, 64), func(t *testing.T) {
			b := make([]byte, 0, 50)
			b = append(b, "prefix-"...)
			got := string(AppendFloatAmount(b, tt.amount))
			want := "prefix-" + tt.want
			if got != want {
				t.Errorf("AppendFloatAmount(%v) = %v, want %v", tt.amount, got, want)
			}
		})
	}
}

func BenchmarkAppendFloatAmount(b *testing.B) {
	amount := 1234.56
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := make([]byte, 0, 50)
		buf = AppendFloatAmount(buf, amount)
		_ = buf
	}
}
