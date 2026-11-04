package domain

import (
	"errors"
	"math"
	"math/big"
	"strconv"
)

// ErrOverflow is returned by every checked operation that would overflow or
// underflow int64. Authoritative values are never allowed to wrap silently.
var ErrOverflow = errors.New("domain: integer overflow")

// ErrNegative is returned where the domain forbids a negative operand.
var ErrNegative = errors.New("domain: negative operand")

// ErrDivisionByZero is returned instead of panicking.
var ErrDivisionByZero = errors.New("domain: division by zero")

// ErrInvalidRange is returned when a clamp or conversion range is unusable.
var ErrInvalidRange = errors.New("domain: invalid range")

// AddInt64 returns a+b, or ErrOverflow.
func AddInt64(a, b int64) (int64, error) {
	sum := a + b
	if (b > 0 && sum < a) || (b < 0 && sum > a) {
		return 0, ErrOverflow
	}
	return sum, nil
}

// SubInt64 returns a-b, or ErrOverflow.
func SubInt64(a, b int64) (int64, error) {
	diff := a - b
	if (b < 0 && diff < a) || (b > 0 && diff > a) {
		return 0, ErrOverflow
	}
	return diff, nil
}

// MulInt64 returns a*b, or ErrOverflow.
func MulInt64(a, b int64) (int64, error) {
	if a == 0 || b == 0 {
		return 0, nil
	}
	product := a * b
	if product/b != a || (a == math.MinInt64 && b == -1) || (b == math.MinInt64 && a == -1) {
		return 0, ErrOverflow
	}
	return product, nil
}

// MustNonNegative returns ErrNegative when v < 0.
func MustNonNegative(v int64) (int64, error) {
	if v < 0 {
		return 0, ErrNegative
	}
	return v, nil
}

// Isqrt returns floor(sqrt(n)) for n >= 0 using integer arithmetic only.
// It is exact at every int64 boundary, including math.MaxInt64.
func Isqrt(n int64) (int64, error) {
	if n < 0 {
		return 0, ErrNegative
	}
	if n < 2 {
		return n, nil
	}

	// Newton iteration starting above the root. The initial value uses the bit
	// length so that the first guess is >= sqrt(n) without overflow.
	x := int64(1) << ((bitsFor(n) + 1) / 2)
	for {
		next := (x + n/x) / 2
		if next >= x {
			return x, nil
		}
		x = next
	}
}

func bitsFor(n int64) uint {
	var bits uint
	for v := n; v > 0; v >>= 1 {
		bits++
	}
	return bits
}

// CeilDiv returns ceil(a/b) for b > 0 and a >= 0, without overflow.
func CeilDiv(a, b int64) (int64, error) {
	if b == 0 {
		return 0, ErrDivisionByZero
	}
	if b < 0 {
		return 0, ErrInvalidRange
	}
	if a < 0 {
		return 0, ErrNegative
	}
	quotient := a / b
	if a%b != 0 {
		quotient++
	}
	return quotient, nil
}

// Clamp bounds v into [lo, hi].
func Clamp(v, lo, hi int64) (int64, error) {
	if lo > hi {
		return 0, ErrInvalidRange
	}
	if v < lo {
		return lo, nil
	}
	if v > hi {
		return hi, nil
	}
	return v, nil
}

// SafePercent returns floor(part*100/den) using SafeRatio.
func SafePercent(part, den int64) (int64, error) {
	return SafeRatio(part, den, 100)
}

// SafeRatio returns floor(num*scale/den) exactly, with no floating point and no
// false overflow: when the intermediate product does not fit in int64 it is
// computed with math/big and the final quotient is range-checked.
//
// Errors: ErrDivisionByZero for den == 0, ErrNegative for negative operands,
// ErrOverflow when the exact result does not fit in int64.
func SafeRatio(num, den, scale int64) (int64, error) {
	// Zero denominators are checked before any division, including the fast-path
	// bound test, so no input can panic.
	if den == 0 {
		return 0, ErrDivisionByZero
	}
	if den < 0 || num < 0 || scale < 0 {
		return 0, ErrNegative
	}
	if num == 0 || scale == 0 {
		return 0, nil
	}
	// Fast path: the product fits in int64, so plain integer arithmetic is exact.
	if num <= math.MaxInt64/scale {
		return (num * scale) / den, nil
	}
	// Exact path: arbitrary precision for the product, then a range check.
	var product, factor, quotient big.Int
	product.SetInt64(num)
	factor.SetInt64(scale)
	product.Mul(&product, &factor)
	factor.SetInt64(den)
	quotient.Quo(&product, &factor)
	if !quotient.IsInt64() {
		return 0, ErrOverflow
	}
	return quotient.Int64(), nil
}

// ParseInt64Decimal parses a canonical base-10 integer string without accepting
// floats, exponents, signs other than a leading '-' or whitespace.
func ParseInt64Decimal(raw string) (int64, error) {
	if raw == "" {
		return 0, errors.New("domain: empty integer")
	}
	for i, r := range raw {
		if i == 0 && r == '-' {
			if len(raw) == 1 {
				return 0, errors.New("domain: bare minus sign")
			}
			continue
		}
		if r < '0' || r > '9' {
			return 0, errors.New("domain: not a base-10 integer")
		}
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, ErrOverflow
	}
	return value, nil
}

// FormatInt64 renders a canonical base-10 integer.
func FormatInt64(value int64) string {
	return strconv.FormatInt(value, 10)
}
