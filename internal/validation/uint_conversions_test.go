package validation

import (
	"strconv"
	"testing"
)

func TestFormatUint(t *testing.T) {
	tests := []struct {
		input    uint64
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{12345, "12345"},
		{18446744073709551615, "18446744073709551615"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			res := FormatUint(tt.input)
			if res != tt.expected {
				t.Errorf("FormatUint(%d): expected %s, got %s", tt.input, tt.expected, res)
			}
		})
	}
}

func TestParseUint(t *testing.T) {
	tests := []struct {
		input    string
		def      uint64
		expected uint64
		err      bool
	}{
		{"0", 1, 0, false},
		{"12345", 0, 12345, false},
		{"18446744073709551615", 0, 18446744073709551615, false},
		{"", 5, 5, true},
		{"12a3", 5, 5, true},
		{"999999999999999999999", 5, 5, true}, // len > 20
		{"18446744073709551616", 5, 5, true}, // exact overflow
		{"99999999999999999999", 5, 5, true}, // overflow len 20
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			res, err := ParseUint(tt.input, tt.def)
			if (err != nil) != tt.err {
				t.Errorf("ParseUint(%q): expected err=%v, got %v", tt.input, tt.err, err)
			}
			if res != tt.expected {
				t.Errorf("ParseUint(%q): expected %d, got %d", tt.input, tt.expected, res)
			}
		})
	}
}

func BenchmarkFormatUintString(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = strconv.FormatUint(1234567890, 10)
	}
}

func BenchmarkFormatUintManual(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FormatUint(1234567890)
	}
}

func BenchmarkParseUintString(b *testing.B) {
	s := "1234567890"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = strconv.ParseUint(s, 10, 64)
	}
}

func BenchmarkParseUintManual(b *testing.B) {
	s := "1234567890"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseUint(s, 0)
	}
}
