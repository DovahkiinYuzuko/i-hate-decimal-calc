package calc

import (
	"strings"
	"testing"
)

func TestEvalZTrans_BasicSignals(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		expected string
	}{
		{
			name:     "delta[n] -> 1",
			expr:     "ztrans(delta(n), n, z)",
			expected: "1",
		},
		{
			name:     "u[n] -> z / (z - 1)",
			expr:     "ztrans(u(n), n, z)",
			expected: "z * (-1 + z)^-1",
		},
		{
			name:     "constant 1 -> z / (z - 1)",
			expr:     "ztrans(1, n, z)",
			expected: "z * (-1 + z)^-1",
		},
		{
			name:     "geometric 2^n -> z / (z - 2)",
			expr:     "ztrans(2^n, n, z)",
			expected: "z * (-2 + z)^-1",
		},
		{
			name:     "geometric (1/2)^n -> z / (z - 1/2)",
			expr:     "ztrans((1/2)^n, n, z)",
			expected: "z * (-1/2 + z)^-1",
		},
		{
			name:     "ramp n -> z / (z - 1)^2",
			expr:     "ztrans(n, n, z)",
			expected: "z * (-1 + z)^2^-1",
		},
		{
			name:     "shifted delta(n-1) -> z^(-1)",
			expr:     "ztrans(delta(n - 1), n, z)",
			expected: "z^-1",
		},
		{
			name:     "shifted delta(n-3) -> z^(-3)",
			expr:     "ztrans(delta(n - 3), n, z)",
			expected: "z^-3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := EvalString(tt.expr)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.expr, err)
			}
			resStr := res.String()
			// Algebraic check: res - expected == 0
			diffExpr := "(" + resStr + ") - (" + tt.expected + ")"
			diffRes, err := EvalString(diffExpr)
			if err != nil {
				t.Fatalf("failed evaluating diff %s: %v", diffExpr, err)
			}
			simpDiff, err := EvalString("simplify(" + diffRes.String() + ")")
			if err == nil && isZero(simpDiff) {
				return // algebraically equivalent
			}
			if resStr != tt.expected {
				t.Errorf("for %s expected %s, got %s", tt.expr, tt.expected, resStr)
			}
		})
	}
}

func TestEvalZTrans_LinearityAndScaling(t *testing.T) {
	// 3*2^n - 4*3^n
	res, err := EvalString("ztrans(3 * 2^n - 4 * 3^n, n, z)")
	if err != nil {
		t.Fatalf("ztrans failed: %v", err)
	}
	resStr := res.String()
	if !strings.Contains(resStr, "z") {
		t.Errorf("expected z in result, got %s", resStr)
	}

	// n * 2^n
	resN2, err := EvalString("ztrans(n * 2^n, n, z)")
	if err != nil {
		t.Fatalf("ztrans(n * 2^n) failed: %v", err)
	}
	if !strings.Contains(resN2.String(), "z") {
		t.Errorf("expected z in result, got %s", resN2.String())
	}
}

func TestEvalZTrans_Trigonometric(t *testing.T) {
	resSin, err := EvalString("ztrans(sin(w * n), n, z)")
	if err != nil {
		t.Fatalf("ztrans(sin) failed: %v", err)
	}
	if !strings.Contains(resSin.String(), "sin(w)") {
		t.Errorf("expected sin(w) in numerator, got %s", resSin.String())
	}

	resCos, err := EvalString("ztrans(cos(w * n), n, z)")
	if err != nil {
		t.Fatalf("ztrans(cos) failed: %v", err)
	}
	if !strings.Contains(resCos.String(), "cos(w)") {
		t.Errorf("expected cos(w) in numerator, got %s", resCos.String())
	}
}
