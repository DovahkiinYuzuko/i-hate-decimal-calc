package calc

import (
	"errors"
	"math/big"
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/padic"
)

func TestPadicClosedDomainArithmetic(t *testing.T) {
	// Deductive: 1/3 in Q_5 with prec 4 -> 3 * (1/3) == 1 + O(5^4)
	t.Run("Deductive_InverseAndMultiplicativeIdentity", func(t *testing.T) {
		node, err := EvalString("padic_expand(1/3, 5, 4)")
		if err != nil {
			t.Fatalf("EvalString failed: %v", err)
		}
		pn, ok := node.(*PadicNode)
		if !ok {
			t.Fatalf("expected *PadicNode, got %T (%v)", node, node)
		}
		if pn.Prime.Cmp(big.NewInt(5)) != 0 || pn.Valuation != 0 || pn.Precision != 4 || pn.IsZero {
			t.Fatalf("unexpected padic node fields: %+v", pn)
		}

		prod, err := EvalString("3 * padic_expand(1/3, 5, 4)")
		if err != nil {
			t.Fatalf("EvalString 3 * padic failed: %v", err)
		}
		prodPadic, ok := prod.(*PadicNode)
		if !ok {
			t.Fatalf("expected *PadicNode from rational multiplication, got %T (%v)", prod, prod)
		}
		if prodPadic.IsZero || prodPadic.Valuation != 0 || prodPadic.Unit.Cmp(big.NewInt(1)) != 0 || prodPadic.Precision != 4 {
			t.Errorf("3 * padic_expand(1/3, 5, 4) = %+v, want unit=1 v=0 r=4", prodPadic)
		}

		// Inverse of 1/3 mod 5^4 must be 3 + O(5^4)
		invNode, err := EvalString("padic_expand(1/3, 5, 4)^(-1)")
		if err != nil {
			t.Fatalf("EvalString padic^(-1) failed: %v", err)
		}
		invPadic, ok := invNode.(*PadicNode)
		if !ok {
			t.Fatalf("expected *PadicNode from padic^(-1), got %T (%v)", invNode, invNode)
		}
		if invPadic.Unit.Cmp(big.NewInt(3)) != 0 || invPadic.Valuation != 0 || invPadic.Precision != 4 {
			t.Errorf("(1/3)^(-1) = %+v, want unit=3 v=0 r=4", invPadic)
		}
	})

	// Deductive: additive cancellation renormalizes valuation and relative precision while preserving absPrec = v + r
	t.Run("Deductive_AdditiveCancellationAbsolutePrecision", func(t *testing.T) {
		sum, err := EvalString("padic_expand(1, 5, 4) + padic_expand(4, 5, 4)")
		if err != nil {
			t.Fatalf("EvalString failed: %v", err)
		}
		sp, ok := sum.(*PadicNode)
		if !ok {
			t.Fatalf("expected *PadicNode, got %T (%v)", sum, sum)
		}
		// 1 + 4 = 5 = 1 * 5^1 + O(5^4) -> Valuation=1, Precision=3, Unit=1
		if sp.IsZero || sp.Valuation != 1 || sp.Precision != 3 || sp.Unit.Cmp(big.NewInt(1)) != 0 {
			t.Errorf("1 + 4 in Q_5: got %+v, want v=1, r=3, unit=1", sp)
		}

		// Full cancellation preserves zero with O(5^4) instead of degrading to exact integer 0
		zeroSum, err := EvalString("padic_expand(3, 5, 4) - padic_expand(3, 5, 4)")
		if err != nil {
			t.Fatalf("EvalString self-sub failed: %v", err)
		}
		zp, ok := zeroSum.(*PadicNode)
		if !ok {
			t.Fatalf("expected *PadicNode on self-subtraction, got %T (%v)", zeroSum, zeroSum)
		}
		if !zp.IsZero || zp.Precision != 4 {
			t.Errorf("self-subtraction: got %+v, want IsZero=true Precision=4", zp)
		}
	})

	// Contrapositive: p * p^(-1) must NOT cancel via symbolic x*x^-1 -> exact 1; must preserve O(p^k)
	t.Run("Contrapositive_NoSymbolicCancellation", func(t *testing.T) {
		node, err := EvalString("padic_expand(2, 5, 3) * padic_expand(2, 5, 3)^(-1)")
		if err != nil {
			t.Fatalf("EvalString failed: %v", err)
		}
		pn, ok := node.(*PadicNode)
		if !ok {
			t.Fatalf("expected *PadicNode preserving O(5^3), got %T (%v)", node, node)
		}
		if pn.IsZero || pn.Valuation != 0 || pn.Unit.Cmp(big.NewInt(1)) != 0 || pn.Precision != 3 {
			t.Errorf("got %+v, want 1 + O(5^3)", pn)
		}

		// Power shifts precision: (5 + O(5^5))^2 -> 25 + O(5^6) (absPrec = 2*1 + 4 = 6, relPrec = 4)
		sq, err := EvalString("padic_expand(5, 5, 4)^2")
		if err != nil {
			t.Fatalf("EvalString square failed: %v", err)
		}
		sqp, ok := sq.(*PadicNode)
		if !ok {
			t.Fatalf("expected *PadicNode, got %T (%v)", sq, sq)
		}
		if sqp.Valuation != 2 || sqp.Unit.Cmp(big.NewInt(1)) != 0 || sqp.Precision != 4 {
			t.Errorf("padic(5)^2 = %+v, want v=2, unit=1, r=4", sqp)
		}
	})

	// Contradiction: mismatched primes in direct p-adic addition and division by truncated zero
	t.Run("Contradiction_PrimeMismatchAndZeroDivision", func(t *testing.T) {
		p5, err := padic.NewPadicFromRat(big.NewRat(1, 1), big.NewInt(5), 3)
		if err != nil {
			t.Fatalf("NewPadicFromRat(5) failed: %v", err)
		}
		p7, err := padic.NewPadicFromRat(big.NewRat(1, 1), big.NewInt(7), 3)
		if err != nil {
			t.Fatalf("NewPadicFromRat(7) failed: %v", err)
		}
		if _, err := padic.Add(p5, p7); err == nil {
			t.Errorf("expected error for padic.Add(p=5, p=7) mismatch, got nil")
		}
		if mixed, err := EvalString("padic_expand(1, 5, 3) + padic_expand(1, 7, 3)"); err != nil {
			t.Errorf("unexpected error for symbolic sum of distinct primes: %v", err)
		} else if _, isAdd := mixed.(*AddNode); !isAdd {
			t.Errorf("expected unmerged *AddNode for distinct primes, got %T (%v)", mixed, mixed)
		}
		if _, err := EvalString("padic_expand(1, 5, 3) / (padic_expand(1, 5, 3) - padic_expand(1, 5, 3))"); err == nil {
			t.Errorf("expected zero-division error for 0 + O(5^3), got nil")
		}
	})
}

func TestPuiseuxClosedDomainArithmetic(t *testing.T) {
	// Deductive: puiseux(y^2 - x, y, x, 3) emits *PuiseuxNode branches with ramification index 2
	t.Run("Deductive_FirstClassNodeAndSquaring", func(t *testing.T) {
		res, err := EvalString("puiseux(y^2 - x, y, x, 3)")
		if err != nil {
			t.Fatalf("EvalString puiseux failed: %v", err)
		}
		list, ok := res.(*ListNode)
		if !ok || len(list.Elements) != 2 {
			t.Fatalf("expected 2-branch ListNode, got %T (%v)", res, res)
		}
		b0, ok := list.Elements[0].(*PuiseuxNode)
		if !ok {
			t.Fatalf("expected *PuiseuxNode element, got %T (%v)", list.Elements[0], list.Elements[0])
		}
		if b0.RamificationIndex != 2 || b0.Var != "x" {
			t.Errorf("unexpected branch metadata: var=%s ram=%d", b0.Var, b0.RamificationIndex)
		}

		// Squaring branch: (+-x^(1/2) + O(x^3))^2 -> x + O(x^(7/2))
		sq, err := EvalString("(puiseux(y^2 - x, y, x, 3)[0])^2")
		if err != nil {
			t.Fatalf("squaring branch failed: %v", err)
		}
		sqP, ok := sq.(*PuiseuxNode)
		if !ok {
			t.Fatalf("expected *PuiseuxNode after squaring, got %T (%v)", sq, sq)
		}
		if len(sqP.Terms) != 1 || sqP.Terms[0].Coeff.Cmp(big.NewRat(1, 1)) != 0 || sqP.Terms[0].Exp.Cmp(big.NewRat(1, 1)) != 0 {
			t.Errorf("squared branch terms = %v, want single term 1*x^1", sqP.Terms)
		}
		if sqP.RamificationIndex != 1 {
			t.Errorf("after squaring, ramification index = %d, want 1", sqP.RamificationIndex)
		}
	})

	// Deductive + Metamorphic: (1 - x + O(x^4))^(-1) * (1 - x + O(x^4)) == 1 + O(x^3)
	t.Run("Metamorphic_PuiseuxInverseRoundtripAndDiff", func(t *testing.T) {
		s, err := NewPuiseuxNode("x", []PuiseuxTermAST{
			{Coeff: big.NewRat(1, 1), Exp: big.NewRat(0, 1)},
			{Coeff: big.NewRat(-1, 1), Exp: big.NewRat(1, 1)},
		}, big.NewRat(4, 1))
		if err != nil {
			t.Fatalf("NewPuiseuxNode failed: %v", err)
		}

		inv, err := PuiseuxInv(s)
		if err != nil {
			t.Fatalf("PuiseuxInv failed: %v", err)
		}
		// Expect 1 + x + x^2 + x^3 + O(x^4)
		wantCoeffs := []*big.Rat{big.NewRat(1, 1), big.NewRat(1, 1), big.NewRat(1, 1), big.NewRat(1, 1)}
		if len(inv.Terms) != len(wantCoeffs) {
			t.Fatalf("inv = %s, want 4 terms", inv.String())
		}
		for i, c := range wantCoeffs {
			if inv.Terms[i].Coeff.Cmp(c) != 0 || inv.Terms[i].Exp.Cmp(big.NewRat(int64(i), 1)) != 0 {
				t.Errorf("term %d = %v*x^%v, want %v*x^%d", i, inv.Terms[i].Coeff, inv.Terms[i].Exp, c, i)
			}
		}

		prod, err := PuiseuxMul(s, inv)
		if err != nil {
			t.Fatalf("PuiseuxMul failed: %v", err)
		}
		if len(prod.Terms) != 1 || prod.Terms[0].Coeff.Cmp(big.NewRat(1, 1)) != 0 || prod.Terms[0].Exp.Sign() != 0 {
			t.Errorf("s * inv = %s, want 1 + O(x^...)", prod.String())
		}

		// Differentiate first-class PuiseuxNode via EvalString("diff(...)")
		dNode, err := EvalString("diff(puiseux(y^2 - x, y, x, 3)[0], x)")
		if err != nil {
			t.Fatalf("diff(puiseux) failed: %v", err)
		}
		dP, ok := dNode.(*PuiseuxNode)
		if !ok {
			t.Fatalf("expected *PuiseuxNode from diff, got %T (%v)", dNode, dNode)
		}
		if len(dP.Terms) == 0 || dP.Terms[0].Exp.Cmp(big.NewRat(-1, 2)) != 0 {
			t.Errorf("diff term = %+v, want leading exponent -1/2", dP.Terms)
		}
		absLeading := new(big.Rat).Abs(dP.Terms[0].Coeff)
		if absLeading.Cmp(big.NewRat(1, 2)) != 0 {
			t.Errorf("diff leading |coeff| = %v, want 1/2", absLeading)
		}
		if dP.TruncationOrder.Cmp(big.NewRat(2, 1)) != 0 {
			t.Errorf("diff truncation order = %v, want 2", dP.TruncationOrder)
		}
	})

	// Contrapositive: p - p stays O(x^N), p * p^(-1) stays O(x^N), symbolic parameter stays fallback
	t.Run("Contrapositive_NoSymbolicCancelAndSymbolicFallback", func(t *testing.T) {
		diffNode, err := EvalString("puiseux(y^2 - x, y, x, 3)[0] - puiseux(y^2 - x, y, x, 3)[0]")
		if err != nil {
			t.Fatalf("self-sub failed: %v", err)
		}
		dp, ok := diffNode.(*PuiseuxNode)
		if !ok || len(dp.Terms) != 0 || dp.TruncationOrder.Cmp(big.NewRat(3, 1)) != 0 {
			t.Errorf("p - p = %T (%v), want empty *PuiseuxNode with O(x^3)", diffNode, diffNode)
		}

		// Non-rational root sqrt(2) keeps symbolic fallback
		symRes, err := EvalString("puiseux(y^2 - 2*x, y, x, 2)")
		if err != nil {
			t.Fatalf("symbolic coeff puiseux failed: %v", err)
		}
		symList, ok := symRes.(*ListNode)
		if !ok || len(symList.Elements) != 2 {
			t.Fatalf("expected 2 branches for y^2 - 2*x, got %T (%v)", symRes, symRes)
		}
		if _, isClosed := symList.Elements[0].(*PuiseuxNode); isClosed {
			t.Errorf("expected symbolic fallback (not *PuiseuxNode) when sqrt(2) coefficient is present")
		}
	})

	// Contradiction: variable mismatch in PuiseuxAdd and zero division
	t.Run("Contradiction_VarMismatchAndZeroDivision", func(t *testing.T) {
		px, err := NewPuiseuxNode("x", []PuiseuxTermAST{{Coeff: big.NewRat(1, 1), Exp: big.NewRat(1, 1)}}, big.NewRat(2, 1))
		if err != nil {
			t.Fatalf("NewPuiseuxNode(x) failed: %v", err)
		}
		pt, err := NewPuiseuxNode("t", []PuiseuxTermAST{{Coeff: big.NewRat(1, 1), Exp: big.NewRat(1, 1)}}, big.NewRat(2, 1))
		if err != nil {
			t.Fatalf("NewPuiseuxNode(t) failed: %v", err)
		}
		res, err := PuiseuxAdd(px, pt)
		if err == nil {
			t.Errorf("expected error adding Puiseux series in x and t, got %v", res)
		}
		var ce *CalcError
		if !errors.As(err, &ce) || ce.Code != ErrDomain {
			t.Errorf("expected DomainError, got %T (%v)", err, err)
		}
		if mixed, err := EvalString("puiseux(y - x, y, x, 2)[0] + puiseux(y - t, y, t, 2)[0]"); err != nil {
			t.Errorf("unexpected error for symbolic sum of distinct variables: %v", err)
		} else if _, isAdd := mixed.(*AddNode); !isAdd {
			t.Errorf("expected unmerged *AddNode for distinct variables, got %T (%v)", mixed, mixed)
		}

		if _, err := EvalString("(puiseux(y - x, y, x, 2)[0] - puiseux(y - x, y, x, 2)[0])^(-1)"); err == nil {
			t.Errorf("expected zero-division error for 0 + O(x^2)^(-1)")
		}
	})
}

func TestIntervalClosedDomainArithmetic(t *testing.T) {
	// Deductive: interval([low, high]) construction and interval arithmetic
	t.Run("Deductive_ConstructorAndArithmetic", func(t *testing.T) {
		iv, err := EvalString("interval([1, 3]) + interval([2, 4])")
		if err != nil {
			t.Fatalf("interval addition failed: %v", err)
		}
		ivn, ok := iv.(*IntervalNode)
		if !ok {
			t.Fatalf("expected *IntervalNode, got %T (%v)", iv, iv)
		}
		if ivn.Low.Cmp(big.NewRat(3, 1)) != 0 || ivn.High.Cmp(big.NewRat(7, 1)) != 0 {
			t.Errorf("[1,3]+[2,4] = %v, want [3, 7]", ivn)
		}

		// Integer power containing 0 gives tight square [0, 4]
		sq, err := EvalString("interval([-1, 2])^2")
		if err != nil {
			t.Fatalf("interval square failed: %v", err)
		}
		sqn, ok := sq.(*IntervalNode)
		if !ok || sqn.Low.Sign() != 0 || sqn.High.Cmp(big.NewRat(4, 1)) != 0 {
			t.Errorf("[-1,2]^2 = %T (%v), want [0, 4]", sq, sq)
		}

		// Comparison on disjoint intervals evaluates deterministically via 3-valued logic
		cmpTrue, err := EvalString("interval([1, 2]) < interval([3, 4])")
		if err != nil {
			t.Fatalf("interval comparison failed: %v", err)
		}
		bn, ok := cmpTrue.(*VarNode)
		if !ok || bn.Name != "true" {
			t.Errorf("[1,2] < [3,4] = %T (%v), want true", cmpTrue, cmpTrue)
		}
	})

	// Contrapositive: dependency problem preserved (NO false cancellation X - X -> 0 or X/X -> 1)
	t.Run("Contrapositive_NoFalseDependencyCancellation", func(t *testing.T) {
		diff, err := EvalString("interval([1, 2]) - interval([1, 2])")
		if err != nil {
			t.Fatalf("interval self-sub failed: %v", err)
		}
		dn, ok := diff.(*IntervalNode)
		if !ok || dn.Low.Cmp(big.NewRat(-1, 1)) != 0 || dn.High.Cmp(big.NewRat(1, 1)) != 0 {
			t.Errorf("[1,2]-[1,2] = %T (%v), want [-1, 1]", diff, diff)
		}

		div, err := EvalString("interval([1, 2]) / interval([1, 2])")
		if err != nil {
			t.Fatalf("interval self-div failed: %v", err)
		}
		qn, ok := div.(*IntervalNode)
		if !ok || qn.Low.Cmp(big.NewRat(1, 2)) != 0 || qn.High.Cmp(big.NewRat(2, 1)) != 0 {
			t.Errorf("[1,2]/[1,2] = %T (%v), want [1/2, 2]", div, div)
		}

		// Overlapping comparison must stay indeterminate (unevaluated RelOpNode)
		overlap, err := EvalString("interval([1, 3]) < interval([2, 4])")
		if err != nil {
			t.Fatalf("overlapping interval comparison error: %v", err)
		}
		if _, ok := overlap.(*RelOpNode); !ok {
			t.Errorf("overlapping [1,3] < [2,4] should stay indeterminate RelOpNode, got %T (%v)", overlap, overlap)
		}
	})

	// Contradiction: low > high bounds rejected and negative power crossing 0 rejected
	t.Run("Contradiction_InvalidBoundsAndZeroCrossing", func(t *testing.T) {
		_, err := EvalString("interval([5, 2])")
		if err == nil {
			t.Errorf("expected DomainError for interval([5, 2]), got nil")
		}
		var ce *CalcError
		if !errors.As(err, &ce) || ce.Code != ErrDomain {
			t.Errorf("expected ErrDomain, got %T (%v)", err, err)
		}

		if _, err := EvalString("interval([-1, 1])^(-1)"); err == nil {
			t.Errorf("expected division by zero error for [-1, 1]^(-1)")
		}
	})
}

func TestIssue163SubIssueRegressions(t *testing.T) {
	// 1. Compound symbolic terms wrapping closed-domain subtrees must not trigger infinite recursion
	t.Run("NoInfiniteRecursionInMixedAdd", func(t *testing.T) {
		res, err := EvalString("x * interval([1, 2]) + y * interval([1, 2])")
		if err != nil {
			t.Fatalf("mixed closed-domain add failed: %v", err)
		}
		if _, ok := res.(*AddNode); !ok {
			t.Errorf("expected *AddNode, got %T (%v)", res, res)
		}

		res2, err := EvalString("sin(interval([1, 2])) + cos(interval([1, 2]))")
		if err != nil {
			t.Fatalf("compound trig closed-domain add failed: %v", err)
		}
		if res2 == nil {
			t.Errorf("expected non-nil AST node")
		}
	})

	// 2. EvalNodeInterval PowNode must propagate zero-crossing negative power errors instead of [0, 0]
	t.Run("EvalNodeIntervalPowChecked", func(t *testing.T) {
		iv := NewIntervalNode(big.NewRat(-1, 1), big.NewRat(1, 1))
		pow := &PowNode{Base: iv, Exp: mustRational(-2, 1)}
		if _, err := EvalNodeInterval(pow, 10); err == nil {
			t.Errorf("expected error for EvalNodeInterval([-1,1]^(-2)), got nil")
		}
	})

	// 3. Nil receiver safety for PadicNode.Equal and PuiseuxNode.Equal
	t.Run("NilReceiverEqualSafe", func(t *testing.T) {
		var nilPadic *PadicNode
		var nilPuiseux *PuiseuxNode
		pOther, _ := NewPadicNode(big.NewInt(5), 0, big.NewInt(1), 3, false)
		sOther, _ := NewPuiseuxNode("x", []PuiseuxTermAST{{Coeff: big.NewRat(1, 1), Exp: big.NewRat(1, 1)}}, big.NewRat(2, 1))
		if nilPadic.Equal(pOther) || nilPadic.Equal(nil) {
			t.Errorf("nil *PadicNode Equal should return false")
		}
		if nilPuiseux.Equal(sOther) || nilPuiseux.Equal(nil) {
			t.Errorf("nil *PuiseuxNode Equal should return false")
		}
	})

	// 4. Composite base p in padic_val/padic_norm/padic_expand must be rejected
	t.Run("PadicCompositePrimeRejected", func(t *testing.T) {
		for _, expr := range []string{
			"padic_val(0, 4)",
			"padic_val(12, 6)",
			"padic_norm(5, 9)",
			"padic_expand(1/3, 4, 4)",
		} {
			if _, err := EvalString(expr); err == nil {
				t.Errorf("%s: expected composite prime error, got nil", expr)
			}
		}
	})
}
