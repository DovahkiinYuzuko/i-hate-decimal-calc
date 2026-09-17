package calc

import (
	"math/big"
	"testing"
)

func TestEvalSturm(t *testing.T) {
	tests := []struct {
		expr    string
		minDeg  int
		wantLen int
	}{
		{
			expr:    "sturm(x^3 - 3*x + 1, x)",
			wantLen: 4, // deg 3 -> 4 terms
		},
		{
			expr:    "sturm(x^2 - 2, x)",
			wantLen: 3, // deg 2 -> 3 terms
		},
		{
			expr:    "sturm(x + 5, x)",
			wantLen: 2, // deg 1 -> 2 terms
		},
	}

	for _, tc := range tests {
		t.Run(tc.expr, func(t *testing.T) {
			res, err := EvalString(tc.expr)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			list, ok := res.(*ListNode)
			if !ok {
				t.Fatalf("expected ListNode, got %T (%v)", res, res)
			}
			if len(list.Elements) != tc.wantLen {
				t.Errorf("expected len %d, got %d", tc.wantLen, len(list.Elements))
			}
		})
	}
}

func TestEvalRootCount(t *testing.T) {
	tests := []struct {
		expr string
		want int64
	}{
		// x^2 + 1 has 0 real roots
		{expr: "root_count(x^2 + 1, x)", want: 0},
		// x^2 - 2 has 2 real roots on (-inf, inf)
		{expr: "root_count(x^2 - 2, x)", want: 2},
		// x^2 - 2 in (0, 2) has 1 real root (sqrt(2) approx 1.414)
		{expr: "root_count(x^2 - 2, x, 0, 2)", want: 1},
		// x^2 - 2 in (-2, 0) has 1 real root (-sqrt(2))
		{expr: "root_count(x^2 - 2, x, -2, 0)", want: 1},
		// x^2 - 2 in (2, 5) has 0 real roots
		{expr: "root_count(x^2 - 2, x, 2, 5)", want: 0},
		// x^3 - 3*x + 1 has 3 real roots on (-inf, inf)
		{expr: "root_count(x^3 - 3*x + 1, x)", want: 3},
		// Roots of x^3 - 3*x + 1 are approx -1.879, 0.347, 1.532
		{expr: "root_count(x^3 - 3*x + 1, x, -2, -1)", want: 1},
		{expr: "root_count(x^3 - 3*x + 1, x, 0, 1)", want: 1},
		{expr: "root_count(x^3 - 3*x + 1, x, 1, 2)", want: 1},
		// Multiple roots: (x - 1)^2 * (x + 2) = x^3 - 3*x + 2
		// Distinct real roots: x = 1, x = -2 (total 2)
		{expr: "root_count(x^3 - 3*x + 2, x)", want: 2},
		// Endpoint is root: x^2 - 4 in [0, 2] -> 1 root (x = 2)
		{expr: "root_count(x^2 - 4, x, 0, 2)", want: 1},
		// Endpoint is root: x^2 - 4 in [2, 3] -> 1 root (x = 2 at lower bound)
		{expr: "root_count(x^2 - 4, x, 2, 3)", want: 1},
	}

	for _, tc := range tests {
		t.Run(tc.expr, func(t *testing.T) {
			res, err := EvalString(tc.expr)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			rat, ok := res.(*RationalNode)
			if !ok {
				t.Fatalf("expected RationalNode, got %T (%v)", res, res)
			}
			if rat.Val.Num().Int64() != tc.want {
				t.Errorf("expected %d, got %v", tc.want, rat.Val)
			}
		})
	}
}

func TestEvalIsolateRoots(t *testing.T) {
	tests := []struct {
		expr      string
		wantCount int
	}{
		// x^2 - 2 -> 2 intervals
		{expr: "isolate_roots(x^2 - 2, x)", wantCount: 2},
		// x^2 + 1 -> 0 intervals
		{expr: "isolate_roots(x^2 + 1, x)", wantCount: 0},
		// x^3 - 3*x + 1 -> 3 intervals
		{expr: "isolate_roots(x^3 - 3*x + 1, x)", wantCount: 3},
		// Exact rational roots: x^2 - 4 = 0 -> [-2, -2], [2, 2]
		{expr: "isolate_roots(x^2 - 4, x)", wantCount: 2},
		// Cubic with single real root: x^3 - 2 = 0 -> 1 interval around 2^(1/3) approx 1.2599
		{expr: "isolate_roots(x^3 - 2, x)", wantCount: 1},
	}

	for _, tc := range tests {
		t.Run(tc.expr, func(t *testing.T) {
			res, err := EvalString(tc.expr)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			list, ok := res.(*ListNode)
			if !ok {
				t.Fatalf("expected ListNode, got %T (%v)", res, res)
			}
			if len(list.Elements) != tc.wantCount {
				t.Fatalf("expected %d intervals, got %d (%v)", tc.wantCount, len(list.Elements), list)
			}

			// Verify intervals are valid [l, r] with l <= r
			for i, elem := range list.Elements {
				iv, ok := elem.(*ListNode)
				if !ok || len(iv.Elements) != 2 {
					t.Fatalf("interval %d must be a 2-element ListNode, got %v", i, elem)
				}
				lRat, okL := iv.Elements[0].(*RationalNode)
				rRat, okR := iv.Elements[1].(*RationalNode)
				if !okL || !okR {
					t.Fatalf("interval %d endpoints must be rationals", i)
				}
				if lRat.Val.Cmp(rRat.Val) > 0 {
					t.Errorf("interval %d invalid: l (%v) > r (%v)", i, lRat.Val, rRat.Val)
				}
			}
		})
	}
}

func TestEvalIsolateRootsVerification(t *testing.T) {
	// Check that each interval [l, r] for x^3 - 3*x + 1 contains a sign change or root
	res, err := EvalString("isolate_roots(x^3 - 3*x + 1, x)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list := res.(*ListNode)
	if len(list.Elements) != 3 {
		t.Fatalf("expected 3 roots, got %d", len(list.Elements))
	}

	polyNode, _ := EvalString("x^3 - 3*x + 1")
	pPoly, _ := extractPoly(polyNode, "x")

	for i, elem := range list.Elements {
		iv := elem.(*ListNode)
		l := iv.Elements[0].(*RationalNode).Val
		r := iv.Elements[1].(*RationalNode).Val

		valL, _ := evalPolyAtRat(pPoly, l)
		valR, _ := evalPolyAtRat(pPoly, r)

		prod := new(big.Rat).Mul(valL, valR)
		if prod.Sign() > 0 {
			t.Errorf("interval %d [%v, %v] does not bracket a root (valL=%v, valR=%v)", i, l, r, valL, valR)
		}
	}
}
