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
		b[i] = byte('0' + (u - q*10)) // #nosec G115 -- u - q*10 is mathematically <= 9
		u = q
	}
	i--
	b[i] = byte('0' + u) // #nosec G115 -- u is single digit < 10
	return string(b[i:])
}

const maxUint64 = 1<<64 - 1
const cutoff = maxUint64 / 10

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

// FormatFloatAmount is a highly optimized manual byte loop for formatting positive float64 amounts to 2 decimal places.
// It avoids function call overhead and is faster for formatting monetary amounts than strconv.FormatFloat.
func FormatFloatAmount(amount float64) string {
	isNegative := amount < 0
	if isNegative {
		amount = -amount
	}
	cents := uint64(amount*100 + 0.5)
	if cents == 0 {
		return "0.00"
	}
	var b [24]byte
	i := 24
	for cents >= 10 {
		q := cents / 10
		i--
		b[i] = byte('0' + (cents - q*10)) // #nosec G115
		cents = q
		if i == 22 {
			i--
			b[i] = '.'
		}
	}
	i--
	b[i] = byte('0' + cents) // #nosec G115
	if i == 22 {
		i -= 2
		b[i] = '0'
		b[i+1] = '.'
	} else if i == 23 {
		i -= 3
		b[i] = '0'
		b[i+1] = '.'
		b[i+2] = '0'
	}
	if isNegative {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

