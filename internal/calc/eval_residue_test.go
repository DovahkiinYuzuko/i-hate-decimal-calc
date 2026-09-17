package calc

import (
	"testing"
)

func TestEvalResidue(t *testing.T) {
	tests := []struct {
		expr string
		want string
	}{
		// Simple poles (order 1)
		{expr: "residue(1/z, z, 0)", want: "1"},
		{expr: "residue(1/(z - 2), z, 2)", want: "1"},
		{expr: "residue(1/(z^2 - 1), z, 1)", want: "1/2"},
		{expr: "residue(1/(z^2 - 1), z, -1)", want: "-1/2"},
		{expr: "residue(cos(z)/z, z, 0)", want: "1"},

		// Higher-order poles (order >= 2)
		{expr: "residue(1/z^2, z, 0)", want: "0"},
		{expr: "residue(1/z^3, z, 0)", want: "0"},
		{expr: "residue(exp(z)/z^3, z, 0)", want: "1/2"},
		{expr: "residue(z/(z - 1)^2, z, 1)", want: "1"},
		{expr: "residue(1/(z^2*(z + 1)), z, 0)", want: "-1"},
		{expr: "residue(1/(z*(z - 1)^2), z, 1)", want: "-1"},

		// Regular & removable singularities
		{expr: "residue(z^2 + 3, z, 0)", want: "0"},
		{expr: "residue(sin(z)/z, z, 0)", want: "0"},
		{expr: "residue(5, z, 0)", want: "0"},
	}

	for _, tc := range tests {
		t.Run(tc.expr, func(t *testing.T) {
			res, err := EvalString(tc.expr)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tc.expr, err)
			}
			if res.String() != tc.want {
				t.Errorf("got %s, expected %s", res.String(), tc.want)
			}
		})
	}
}

func TestEvalResidueComplex(t *testing.T) {
	// Residue of 1/(z^2 + 1) at z = i is 1/(2*i) = -i/2
	finalRes, err := EvalString("residue(1/(z^2 + 1), z, i)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Logf("finalRes = %s", finalRes.String())

	// Check algebraic equivalence: finalRes * 2 * i == 1
	prod, err := EvalString(finalRes.String() + " * 2 * i")
	if err != nil {
		t.Fatalf("eval product error: %v", err)
	}
	if prod.String() != "1" {
		t.Errorf("expected product to be 1, got %s (res was %s)", prod.String(), finalRes.String())
	}
}
