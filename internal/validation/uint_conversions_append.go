package validation

// AppendFloatAmount appends a formatted float64 amount to the given byte slice.
// It avoids function call overhead and allocations when formatting monetary amounts
// into pre-allocated buffers.
func AppendFloatAmount(b []byte, amount float64) []byte {
	isNegative := amount < 0
	if isNegative {
		amount = -amount
	}
	cents := uint64(amount*100 + 0.5)

	if cents == 0 {
		return append(b, '0', '.', '0', '0')
	}

	var buf [24]byte
	i := 24
	for cents >= 10 {
		q := cents / 10
		i--
		buf[i] = byte('0' + (cents - q*10)) // #nosec G115
		cents = q
		if i == 22 {
			i--
			buf[i] = '.'
		}
	}
	i--
	buf[i] = byte('0' + cents) // #nosec G115
	if i == 22 {
		i -= 2
		buf[i] = '0'
		buf[i+1] = '.'
	} else if i == 23 {
		i -= 3
		buf[i] = '0'
		buf[i+1] = '.'
		buf[i+2] = '0'
	}
	if isNegative {
		i--
		buf[i] = '-'
	}
	return append(b, buf[i:]...)
}
