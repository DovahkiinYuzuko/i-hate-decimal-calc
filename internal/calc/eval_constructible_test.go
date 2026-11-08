package calc

import (
	"math/big"
	"testing"
)

func newPolyCoeffs(vals ...int64) []*big.Int {
	res := make([]*big.Int, len(vals))
	for i, v := range vals {
		res[i] = big.NewInt(v)
	}
	return res
}

func TestCheckConstructibility(t *testing.T) {
	tests := []struct {
		name     string
		poly     []*big.Int
		expected ConstructibilityStatus
	}{
		{
			name:     "Degree 1: 2x - 3 = 0",
			poly:     newPolyCoeffs(-3, 2),
			expected: ConstructibleTrue,
		},
		{
			name:     "Degree 2: x^2 - 2 = 0",
			poly:     newPolyCoeffs(-2, 0, 1),
			expected: ConstructibleTrue,
		},
		{
			name:     "Degree 3: x^3 - 2 = 0 (Doubling the Cube)",
			poly:     newPolyCoeffs(-2, 0, 0, 1),
			expected: ConstructibleFalse,
		},
		{
			name:     "Degree 3: 8x^3 - 6x - 1 = 0 (Angle Trisection)",
			poly:     newPolyCoeffs(-1, -6, 0, 8),
			expected: ConstructibleFalse,
		},
		{
			name:     "Degree 4: x^4 - 2 = 0 (Constructible 4th root)",
			poly:     newPolyCoeffs(-2, 0, 0, 0, 1),
			expected: ConstructibleTrue,
		},
		{
			name:     "Degree 4: x^4 - x - 1 = 0 (Non-constructible S4 quartic)",
			poly:     newPolyCoeffs(-1, -1, 0, 0, 1),
			expected: ConstructibleFalse,
		},
		{
			name:     "Degree 8: x^8 - 2 = 0 (Degree 8 power of 2: Undetermined)",
			poly:     newPolyCoeffs(-2, 0, 0, 0, 0, 0, 0, 0, 1),
			expected: ConstructibleUndetermined,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cert := CheckConstructibility(tc.poly)
			if cert == nil {
				t.Fatalf("%s: expected non-nil certificate", tc.name)
			}
			if cert.Status != tc.expected {
				t.Errorf("%s: expected status %v, got %v (reason: %s)", tc.name, tc.expected, cert.Status, cert.Reason)
			}
		})
	}
}
