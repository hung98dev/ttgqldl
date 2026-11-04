package domain

import (
	"errors"
	"math"
	"math/big"
	"testing"
)

func TestCheckedAddition(t *testing.T) {
	cases := []struct {
		a, b    int64
		want    int64
		wantErr error
	}{
		{0, 0, 0, nil},
		{1, 2, 3, nil},
		{-5, 5, 0, nil},
		{math.MaxInt64, 0, math.MaxInt64, nil},
		{math.MinInt64, 0, math.MinInt64, nil},
		{math.MaxInt64, 1, 0, ErrOverflow},
		{math.MaxInt64, math.MaxInt64, 0, ErrOverflow},
		{math.MinInt64, -1, 0, ErrOverflow},
		{math.MinInt64, math.MinInt64, 0, ErrOverflow},
	}
	for _, tc := range cases {
		got, err := AddInt64(tc.a, tc.b)
		if tc.wantErr != nil {
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("AddInt64(%d,%d) error = %v, want %v", tc.a, tc.b, err, tc.wantErr)
			}
			continue
		}
		if err != nil || got != tc.want {
			t.Errorf("AddInt64(%d,%d) = %d, %v; want %d", tc.a, tc.b, got, err, tc.want)
		}
	}
}

func TestCheckedSubtraction(t *testing.T) {
	cases := []struct {
		a, b    int64
		want    int64
		wantErr bool
	}{
		{5, 3, 2, false},
		{0, 0, 0, false},
		{math.MinInt64, 0, math.MinInt64, false},
		{math.MaxInt64, math.MaxInt64, 0, false},
		{math.MinInt64, 1, 0, true},
		{math.MaxInt64, -1, 0, true},
	}
	for _, tc := range cases {
		got, err := SubInt64(tc.a, tc.b)
		if tc.wantErr {
			if !errors.Is(err, ErrOverflow) {
				t.Errorf("SubInt64(%d,%d) error = %v, want overflow", tc.a, tc.b, err)
			}
			continue
		}
		if err != nil || got != tc.want {
			t.Errorf("SubInt64(%d,%d) = %d, %v; want %d", tc.a, tc.b, got, err, tc.want)
		}
	}
}

func TestCheckedMultiplication(t *testing.T) {
	cases := []struct {
		a, b    int64
		want    int64
		wantErr bool
	}{
		{0, math.MaxInt64, 0, false},
		{3, 4, 12, false},
		{-3, 4, -12, false},
		{math.MaxInt64, 1, math.MaxInt64, false},
		{math.MaxInt64, 2, 0, true},
		{math.MinInt64, -1, 0, true},
		{1 << 32, 1 << 32, 0, true},
	}
	for _, tc := range cases {
		got, err := MulInt64(tc.a, tc.b)
		if tc.wantErr {
			if !errors.Is(err, ErrOverflow) {
				t.Errorf("MulInt64(%d,%d) error = %v, want overflow", tc.a, tc.b, err)
			}
			continue
		}
		if err != nil || got != tc.want {
			t.Errorf("MulInt64(%d,%d) = %d, %v; want %d", tc.a, tc.b, got, err, tc.want)
		}
	}
}

func TestIsqrt(t *testing.T) {
	cases := []struct {
		n    int64
		want int64
	}{
		{0, 0},
		{1, 1},
		{2, 1},
		{3, 1},
		{4, 2},
		{8, 2},
		{9, 3},
		{15, 3},
		{16, 4},
		{99, 9},
		{100, 10},
		{1024, 32},
		{4096, 64},
		{1_000_000, 1000},
		{3_037_001_881, 55109}, // 55109^2, exact
	}
	for _, tc := range cases {
		got, err := Isqrt(tc.n)
		if err != nil {
			t.Fatalf("Isqrt(%d): %v", tc.n, err)
		}
		if got != tc.want {
			t.Errorf("Isqrt(%d) = %d, want %d", tc.n, got, tc.want)
		}
	}

	// Large values are checked against the standard library's exact integer
	// square root (math/big), which is an independent implementation.
	largeValues := []int64{
		2, 3, 4, 8, 15, 16, 99, 100, 1024, 4096, 1_000_000,
		3_037_000_499, 3_037_001_881, 3_037_001_880,
		9_223_372_030_926_249_001, 9_223_372_030_926_249_000,
		math.MaxInt64 - 1, math.MaxInt64,
	}
	for _, n := range largeValues {
		want := new(big.Int).Sqrt(big.NewInt(n)).Int64()
		got, err := Isqrt(n)
		if err != nil {
			t.Fatalf("Isqrt(%d): %v", n, err)
		}
		if got != want {
			t.Errorf("Isqrt(%d) = %d, want %d (math/big)", n, got, want)
		}
	}

	if _, err := Isqrt(-1); !errors.Is(err, ErrNegative) {
		t.Fatalf("Isqrt(-1) error = %v, want ErrNegative", err)
	}

	// The maximum signed 64-bit value must not overflow or hang.
	got, err := Isqrt(math.MaxInt64)
	if err != nil {
		t.Fatalf("Isqrt(MaxInt64): %v", err)
	}
	const wantMaxRoot = int64(3037000499)
	if got != wantMaxRoot {
		t.Errorf("Isqrt(MaxInt64) = %d, want %d", got, wantMaxRoot)
	}
	if got*got > math.MaxInt64 {
		t.Fatal("Isqrt(MaxInt64) squared exceeds int64")
	}
}
func TestCeilDiv(t *testing.T) {
	cases := []struct {
		a, b    int64
		want    int64
		wantErr error
	}{
		{0, 3, 0, nil},
		{1, 3, 1, nil},
		{3, 3, 1, nil},
		{4, 3, 2, nil},
		{math.MaxInt64, 2, 4611686018427387904, nil},
		{1, 0, 0, ErrDivisionByZero},
		{-1, 3, 0, ErrNegative},
		{1, -3, 0, ErrInvalidRange},
	}
	for _, tc := range cases {
		got, err := CeilDiv(tc.a, tc.b)
		if tc.wantErr != nil {
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("CeilDiv(%d,%d) error = %v, want %v", tc.a, tc.b, err, tc.wantErr)
			}
			continue
		}
		if err != nil || got != tc.want {
			t.Errorf("CeilDiv(%d,%d) = %d, %v; want %d", tc.a, tc.b, got, err, tc.want)
		}
	}
}

func TestClamp(t *testing.T) {
	cases := []struct {
		v, lo, hi int64
		want      int64
		wantErr   bool
	}{
		{5, 0, 10, 5, false},
		{-1, 0, 10, 0, false},
		{11, 0, 10, 10, false},
		{0, 0, 0, 0, false},
		{5, 10, 0, 0, true},
	}
	for _, tc := range cases {
		got, err := Clamp(tc.v, tc.lo, tc.hi)
		if tc.wantErr {
			if !errors.Is(err, ErrInvalidRange) {
				t.Errorf("Clamp(%d,%d,%d) error = %v", tc.v, tc.lo, tc.hi, err)
			}
			continue
		}
		if err != nil || got != tc.want {
			t.Errorf("Clamp(%d,%d,%d) = %d, %v", tc.v, tc.lo, tc.hi, got, err)
		}
	}
}

func TestSafePercentAndRatio(t *testing.T) {
	percent, err := SafePercent(50, 200)
	if err != nil || percent != 25 {
		t.Fatalf("SafePercent(50,200) = %d, %v", percent, err)
	}
	percent, err = SafePercent(math.MaxInt64, math.MaxInt64)
	if err != nil || percent != 100 {
		t.Fatalf("SafePercent(MaxInt64,MaxInt64) = %d, %v", percent, err)
	}
	if _, err := SafePercent(1, 0); !errors.Is(err, ErrDivisionByZero) {
		t.Fatal("SafePercent must reject a zero denominator")
	}
	ratio, err := SafeRatio(7, 3, 1000)
	if err != nil || ratio != 2333 {
		t.Fatalf("SafeRatio(7,3,1000) = %d, %v", ratio, err)
	}
	if _, err := SafeRatio(1, 0, 100); !errors.Is(err, ErrDivisionByZero) {
		t.Fatal("SafeRatio must reject a zero denominator")
	}
	if _, err := SafeRatio(-1, 2, 100); !errors.Is(err, ErrNegative) {
		t.Fatal("SafeRatio must reject negative operands")
	}
}

func TestInt64TextConversion(t *testing.T) {
	values := []int64{0, 1, -1, 42, -42, math.MaxInt64, math.MinInt64}
	for _, value := range values {
		text := FormatInt64(value)
		parsed, err := ParseInt64Decimal(text)
		if err != nil || parsed != value {
			t.Errorf("round trip %d -> %q -> %d, %v", value, text, parsed, err)
		}
	}
	for _, bad := range []string{"", "-", "+1", " 1", "1 ", "1.0", "1e3", "0x10", "abc", "99999999999999999999"} {
		if _, err := ParseInt64Decimal(bad); err == nil {
			t.Errorf("ParseInt64Decimal(%q) accepted invalid input", bad)
		}
	}
	if _, err := ParseInt64Decimal("9223372036854775808"); !errors.Is(err, ErrOverflow) {
		t.Fatal("overflowing text must report ErrOverflow")
	}
}

func FuzzIsqrt(f *testing.F) {
	f.Add(int64(0))
	f.Add(int64(1))
	f.Add(int64(1_000_000))
	f.Fuzz(func(t *testing.T, n int64) {
		if n < 0 {
			return
		}
		root, err := Isqrt(n)
		if err != nil {
			t.Fatalf("Isqrt(%d): %v", n, err)
		}
		if root < 0 {
			t.Fatalf("Isqrt(%d) = %d", n, root)
		}
		// Compare with a division-only check to avoid overflowing root*root.
		if root != 0 && root > n/root {
			t.Fatalf("Isqrt(%d) = %d: square exceeds n", n, root)
		}
		next := root + 1
		if next != 0 && next <= (n-1)/next {
			t.Fatalf("Isqrt(%d) = %d is not the floor", n, root)
		}
	})
}

func FuzzParseInt64Decimal(f *testing.F) {
	f.Add("0")
	f.Add("-1")
	f.Add("9223372036854775807")
	f.Fuzz(func(t *testing.T, text string) {
		value, err := ParseInt64Decimal(text)
		if err != nil {
			return
		}
		if FormatInt64(value) != text {
			t.Fatalf("non-canonical accepted text %q (value %d)", text, value)
		}
	})
}
