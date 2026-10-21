package calc

import (
	"strings"
	"testing"
)

func TestEvalInvZTrans_BasicPoles(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		contains []string
	}{
		{
			name:     "constant 1 -> delta(n)",
			expr:     "iztrans(1, z, n)",
			contains: []string{"delta(n)"},
		},
		{
			name:     "z / (z - 1) -> 1",
			expr:     "iztrans(z / (z - 1), z, n)",
			contains: []string{"1"},
		},
		{
			name:     "z / (z - 2) -> 2^n",
			expr:     "iztrans(z / (z - 2), z, n)",
			contains: []string{"2^n"},
		},
		{
			name:     "z / (z - 1/2) -> (1/2)^n",
			expr:     "iztrans(z / (z - 1/2), z, n)",
			contains: []string{"1/2^n"},
		},
		{
			name:     "1 / z -> delta(n - 1)",
			expr:     "iztrans(1 / z, z, n)",
			contains: []string{"delta", "-1"},
		},
		{
			name:     "1 / z^2 -> delta(n - 2)",
			expr:     "iztrans(1 / z^2, z, n)",
			contains: []string{"delta", "-2"},
		},
		{
			name:     "1 / (z - 2) -> 2^(n-1)",
			expr:     "iztrans(1 / (z - 2), z, n)",
			contains: []string{"2^n"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := EvalString(tt.expr)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.expr, err)
			}
			resStr := res.String()
			for _, c := range tt.contains {
				if !strings.Contains(resStr, c) {
					t.Errorf("for %s expected result to contain %q, got: %s", tt.expr, c, resStr)
				}
			}
		})
	}
}

func TestEvalInvZTrans_LinearityAndRepeatedPoles(t *testing.T) {
	// z / (z - 1) + z / (z - 3) -> 1 + 3^n
	resSum, err := EvalString("iztrans(z / (z - 1) + z / (z - 3), z, n)")
	if err != nil {
		t.Fatalf("iztrans sum failed: %v", err)
	}
	resStr := resSum.String()
	if !strings.Contains(resStr, "3^n") {
		t.Errorf("expected 3^n in result, got %s", resStr)
	}

	// Repeated pole: z / (z - 1)^2 -> n
	resRamp, err := EvalString("iztrans(z / (z - 1)^2, z, n)")
	if err != nil {
		t.Fatalf("iztrans ramp failed: %v", err)
	}
	if !strings.Contains(resRamp.String(), "n") {
		t.Errorf("expected n in ramp result, got %s", resRamp.String())
	}
}

func TestEvalInvZTrans_PolynomialDivision(t *testing.T) {
	// Improper rational with causal quotient: (z + 1) / z = 1 + 1/z -> delta(n) + delta(n-1)
	res, err := EvalString("iztrans((z + 1) / z, z, n)")
	if err != nil {
		t.Fatalf("iztrans improper rational failed: %v", err)
	}
	resStr := res.String()
	if !strings.Contains(resStr, "delta") {
		t.Errorf("expected delta terms in polynomial division, got %s", resStr)
	}
}

func TestEvalInvZTrans_NonCausalBoundary(t *testing.T) {
	// Pure positive power z^2 -> non-causal
	_, err := EvalString("iztrans(z^2, z, n)")
	if err == nil {
		t.Errorf("expected error for non-causal term z^2, got nil")
	}

	// Improper rational with positive quotient degree: (z^2 + 1) / z = z + 1/z -> non-causal
	_, err2 := EvalString("iztrans((z^2 + 1) / z, z, n)")
	if err2 == nil {
		t.Errorf("expected error for non-causal term (z^2 + 1)/z, got nil")
	}
}
