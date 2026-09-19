package validation

import "bank-api/internal/errors"

// FormatUint is a highly optimized manual byte loop for formatting positive uint64.
// It avoids function call overhead and is faster for simple decimal conversions in hot paths.
func FormatUint(u uint64) string {
	if u == 0 {
		return "0"
	}
	var b [20]byte
	i := 20
	for u >= 10 {
		q := u / 10
		i--
		b[i] = byte('0' + u - q*10)
		u = q
	}
	i--
	b[i] = byte('0' + u)
	return string(b[i:])
}

const maxUint64 = 1<<64 - 1
const cutoff = maxUint64/10

// ParseUint is a highly optimized manual byte loop for parsing positive uint64 from strings.
// It avoids function call overhead and is faster for simple decimal parsing in hot paths.
func ParseUint(s string, def uint64) (uint64, error) {
	if s == "" || len(s) > 20 {
		return def, errors.NewValidationError("invalid uint format")
	}

	var res uint64
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return def, errors.NewValidationError("invalid uint format")
		}

		val := uint64(c - '0')

		// Overflow check
		if res > cutoff || (res == cutoff && val > 5) {
			return def, errors.NewValidationError("value out of range")
		}

		res = res*10 + val
	}
	return res, nil
}
