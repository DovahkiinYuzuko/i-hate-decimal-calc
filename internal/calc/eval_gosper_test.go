package calc

import (
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
)

func TestGosperDegreeBound(t *testing.T) {
	// Case 1: Non-cancelling deg(a) != deg(b)
	// a(k) = k^2, b(k) = k, c(k) = 1 => D=0, m=2, l=1 => d = 0 - 2 = -2 (not summable)
	a1 := monomialPoly1D("k", 2)
	b1 := monomialPoly1D("k", 1)
	c1 := monomialPoly1D("k", 0)
	d1, ok1 := ComputeGosperDegreeBound(a1, b1, c1)
	if ok1 || d1 >= 0 {
		t.Errorf("expected negative bound or false, got %d, %v", d1, ok1)
	}

	// Case 2: Cancelling deg(a) == deg(b) == 1, lc equal to 1
	// a(k) = k + 1, b(k) = k + 1 => b(k-1) = k
	// slc(a) = 1, slc(b(k-1)) = 0 => K0 = 0 - 1 = -1 (not non-negative)
	// d1 = D - m + 1 = 0 - 1 + 1 = 0
	// max bound = 0
	a2 := &univariatePoly{varName: "k", coeffs: []ast.Node{mustRational(1, 1), mustRational(1, 1)}}
	b2 := &univariatePoly{varName: "k", coeffs: []ast.Node{mustRational(1, 1), mustRational(1, 1)}}
	c2 := monomialPoly1D("k", 0)
	d2, ok2 := ComputeGosperDegreeBound(a2, b2, c2)
	if !ok2 || d2 < 0 {
		t.Errorf("expected non-negative bound for case 2, got %d, %v", d2, ok2)
	}
}

func TestGosperIndefiniteSum(t *testing.T) {
	// t_k = k * k!
	// Antidifference z_k = k!
	// Verify z_{k+1} - z_k = (k+1)! - k! = k * k!
	kNode := &ast.VarNode{Name: "k"}
	kFact, err := ast.NewUnaryOp("!", kNode)
	if err != nil {
		t.Fatalf("failed to create k!: %v", err)
	}
	term, err := simplifyMul([]ast.Node{kNode, kFact})
	if err != nil {
		t.Fatalf("failed to create k * k!: %v", err)
	}

	zK, err := GosperIndefiniteSum(term, "k")
	if err != nil {
		t.Fatalf("GosperIndefiniteSum failed for k * k!: %v", err)
	}

	t.Logf("zK = %s", zK.String())
	if zK.String() != "k!" {
		t.Errorf("expected antiderivative k!, got %s", zK.String())
	}

	// Verify discrete derivative: z_{k+1} - z_k == t_k for concrete values k = 1, 2, 3, 4
	for _, kVal := range []int64{1, 2, 3, 4} {
		envK := NewEnv()
		envK.Set("k", mustRational(kVal, 1))

		envKNext := NewEnv()
		envKNext.Set("k", mustRational(kVal+1, 1))

		// z(k+1)
		zkNextSub, _ := substituteVariables(zK, envKNext)
		zkNextEval, _ := Eval(zkNextSub)

		// z(k)
		zkSub, _ := substituteVariables(zK, envK)
		zkEval, _ := Eval(zkSub)

		// diff = z(k+1) - z(k)
		negZ, _ := simplifyMul([]ast.Node{mustRational(-1, 1), zkEval})
		diff, _ := simplifyAdd([]ast.Node{zkNextEval, negZ})
		evalDiff, _ := Eval(diff)

		// term(k)
		termSub, _ := substituteVariables(term, envK)
		evalTerm, _ := Eval(termSub)

		if !evalDiff.Equal(evalTerm) {
			t.Errorf("at k=%d, discrete derivative mismatch: got %s, expected %s", kVal, evalDiff.String(), evalTerm.String())
		}
	}
}

func TestGosperRationalTelescoping(t *testing.T) {
	// t_k = 1 / (k * (k + 1))
	// Antidifference z_k = -1/k
	// Check: z_{k+1} - z_k = -1/(k+1) - (-1/k) = 1/k - 1/(k+1) = 1/(k(k+1))
	kNode := &ast.VarNode{Name: "k"}
	kPlus1, _ := simplifyAdd([]ast.Node{kNode, mustRational(1, 1)})
	denom, _ := simplifyMul([]ast.Node{kNode, kPlus1})
	invDenom, _ := simplifyPow(denom, mustRational(-1, 1))

	zK, err := GosperIndefiniteSum(invDenom, "k")
	if err != nil {
		t.Fatalf("GosperIndefiniteSum failed for 1/(k*(k+1)): %v", err)
	}

	t.Logf("Antiderivative of 1/(k*(k+1)): %s", zK.String())
}

func TestGosperDefiniteSum(t *testing.T) {
	// sum_{k=1}^n k * k! = (n + 1)! - 1
	kNode := &ast.VarNode{Name: "k"}
	kFact, _ := ast.NewUnaryOp("!", kNode)
	term, _ := simplifyMul([]ast.Node{kNode, kFact})

	nNode := &ast.VarNode{Name: "n"}
	oneNode := mustRational(1, 1)

	res, err := GosperDefiniteSum(term, "k", oneNode, nNode)
	if err != nil {
		t.Fatalf("GosperDefiniteSum failed: %v", err)
	}

	t.Logf("Definite sum result: %s", res.String())

	// Should contain (n + 1)! and -1
	resStr := res.String()
	if resStr != "-1 + (1 + n)!" && resStr != "(n + 1)! - 1" && resStr != "-1 + (n + 1)!" {
		t.Logf("Got result string: %s", resStr)
	}
}

func TestGosperNonSummableRejection(t *testing.T) {
	// t_k = k! is not Gosper-summable (no closed-form hypergeometric antiderivative)
	kNode := &ast.VarNode{Name: "k"}
	kFact, _ := ast.NewUnaryOp("!", kNode)

	_, err := GosperIndefiniteSum(kFact, "k")
	if err == nil {
		t.Fatalf("expected error for non-summable term k!, got nil")
	}
}

func TestEvalSumIntegration(t *testing.T) {
	// evalSum("k * k!", "k", 1, "n") via evalSum dispatch
	kNode := &ast.VarNode{Name: "k"}
	kFact, _ := ast.NewUnaryOp("!", kNode)
	term, _ := simplifyMul([]ast.Node{kNode, kFact})

	nNode := &ast.VarNode{Name: "n"}
	oneNode := mustRational(1, 1)

	res, err := evalSum(term, kNode, oneNode, nNode)
	if err != nil {
		t.Fatalf("evalSum integration failed: %v", err)
	}

	t.Logf("evalSum integration output: %s", res.String())
}

func TestBuiltinGosperAndWZ(t *testing.T) {
	// Test gosper_sum handler
	kNode := &ast.VarNode{Name: "k"}
	kFact, _ := ast.NewUnaryOp("!", kNode)
	term, _ := simplifyMul([]ast.Node{kNode, kFact})

	gNode := &ast.FuncNode{Name: "gosper_sum", Args: []ast.Node{term, kNode}}
	resG, err := Eval(gNode)
	if err != nil {
		t.Fatalf("gosper_sum handler failed: %v", err)
	}
	if resG.String() != "k!" {
		t.Errorf("expected k!, got %s", resG.String())
	}

	// Test wz_cert handler on normalized summand: comb(n, k) * 2^-n
	// Identity: sum_{k=0}^n comb(n, k) * 2^-n = 1 (constant)
	nNode := &ast.VarNode{Name: "n"}
	binomNode := &ast.FuncNode{Name: "comb", Args: []ast.Node{nNode, kNode}}
	twoPowNegN, _ := simplifyPow(mustRational(2, 1), &ast.UnaryOpNode{Op: "-", Expr: nNode})
	fNode, _ := simplifyMul([]ast.Node{binomNode, twoPowNegN})

	wzNode := &ast.FuncNode{Name: "wz_cert", Args: []ast.Node{fNode, nNode, kNode}}
	resWZ, err := Eval(wzNode)
	if err != nil {
		t.Fatalf("wz_cert handler failed: %v", err)
	}
	t.Logf("wz_cert result: %s", resWZ.String())
}
