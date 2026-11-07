package calc

import (
	"fmt"
	"math/big"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// -------------------------------------------------------------------------
// DenseFPS: Dense Truncated Formal Power Series over Q[[x]] / (x^Precision)
// -------------------------------------------------------------------------

// DenseFPS represents a univariate formal power series with rational coefficients truncated modulo x^Precision.
type DenseFPS struct {
	Var       string
	Coeffs    []*big.Rat // Coeffs[k] is the coefficient of Var^k
	Precision int
}

// NewDenseFPS creates a DenseFPS with given precision, initializing coefficients to 0.
func NewDenseFPS(varName string, prec int) *DenseFPS {
	if prec < 1 {
		prec = 1
	}
	coeffs := make([]*big.Rat, prec)
	for i := range coeffs {
		coeffs[i] = big.NewRat(0, 1)
	}
	return &DenseFPS{
		Var:       varName,
		Coeffs:    coeffs,
		Precision: prec,
	}
}

// Clone creates a deep copy of DenseFPS.
func (f *DenseFPS) Clone() *DenseFPS {
	cp := NewDenseFPS(f.Var, f.Precision)
	for i := 0; i < f.Precision; i++ {
		if i < len(f.Coeffs) && f.Coeffs[i] != nil {
			cp.Coeffs[i] = new(big.Rat).Set(f.Coeffs[i])
		}
	}
	return cp
}

// Add computes f + g mod x^prec.
func (f *DenseFPS) Add(g *DenseFPS) *DenseFPS {
	prec := f.Precision
	if g.Precision > prec {
		prec = g.Precision
	}
	res := NewDenseFPS(f.Var, prec)
	for i := 0; i < prec; i++ {
		if i < len(f.Coeffs) {
			res.Coeffs[i].Add(res.Coeffs[i], f.Coeffs[i])
		}
		if i < len(g.Coeffs) {
			res.Coeffs[i].Add(res.Coeffs[i], g.Coeffs[i])
		}
	}
	return res
}

// Sub computes f - g mod x^prec.
func (f *DenseFPS) Sub(g *DenseFPS) *DenseFPS {
	prec := f.Precision
	if g.Precision > prec {
		prec = g.Precision
	}
	res := NewDenseFPS(f.Var, prec)
	for i := 0; i < prec; i++ {
		if i < len(f.Coeffs) {
			res.Coeffs[i].Add(res.Coeffs[i], f.Coeffs[i])
		}
		if i < len(g.Coeffs) {
			res.Coeffs[i].Sub(res.Coeffs[i], g.Coeffs[i])
		}
	}
	return res
}

// MulTrunc computes f * g mod x^prec using direct truncated multiplication (no NTT blowup).
func (f *DenseFPS) MulTrunc(g *DenseFPS, prec int) *DenseFPS {
	res := NewDenseFPS(f.Var, prec)
	term := new(big.Rat)
	for i := 0; i < prec && i < len(f.Coeffs); i++ {
		if f.Coeffs[i].Sign() == 0 {
			continue
		}
		maxJ := prec - 1 - i
		if maxJ >= len(g.Coeffs) {
			maxJ = len(g.Coeffs) - 1
		}
		for j := 0; j <= maxJ; j++ {
			if g.Coeffs[j].Sign() == 0 {
				continue
			}
			term.Mul(f.Coeffs[i], g.Coeffs[j])
			res.Coeffs[i+j].Add(res.Coeffs[i+j], term)
		}
	}
	return res
}

// Deriv computes the formal derivative f'(x) mod x^(Precision-1).
func (f *DenseFPS) Deriv() *DenseFPS {
	prec := f.Precision - 1
	if prec < 1 {
		prec = 1
	}
	res := NewDenseFPS(f.Var, prec)
	factor := new(big.Rat)
	for i := 1; i < f.Precision; i++ {
		if i-1 < prec {
			factor.SetInt64(int64(i))
			res.Coeffs[i-1].Mul(f.Coeffs[i], factor)
		}
	}
	return res
}

// Integ computes the formal antiderivative integral f(x) dx with constant term 0 mod x^(Precision+1).
func (f *DenseFPS) Integ(prec int) *DenseFPS {
	res := NewDenseFPS(f.Var, prec)
	divisor := new(big.Rat)
	for i := 0; i < f.Precision && i+1 < prec; i++ {
		divisor.SetInt64(int64(i + 1))
		res.Coeffs[i+1].Quo(f.Coeffs[i], divisor)
	}
	return res
}

// -------------------------------------------------------------------------
// FPS Core Algorithms: ODE Recurrence (Main) & Newton Iteration (Oracle)
// -------------------------------------------------------------------------

// InvRecurrence computes 1/A mod x^prec using Brent & Kung (1978) ODE recurrence.
// b_k = -1/a_0 * sum_{i=1}^k a_i * b_{k-i}
func (f *DenseFPS) InvRecurrence(prec int) (*DenseFPS, error) {
	if len(f.Coeffs) == 0 || f.Coeffs[0].Sign() == 0 {
		return nil, fmt.Errorf("%s", i18n.T("fps.err_zero_constant_inv"))
	}
	res := NewDenseFPS(f.Var, prec)
	invA0 := new(big.Rat).Inv(f.Coeffs[0])
	res.Coeffs[0].Set(invA0)

	term := new(big.Rat)
	sum := new(big.Rat)
	for k := 1; k < prec; k++ {
		sum.SetInt64(0)
		for i := 1; i <= k && i < len(f.Coeffs); i++ {
			if f.Coeffs[i].Sign() == 0 || res.Coeffs[k-i].Sign() == 0 {
				continue
			}
			term.Mul(f.Coeffs[i], res.Coeffs[k-i])
			sum.Add(sum, term)
		}
		sum.Mul(sum, invA0)
		res.Coeffs[k].Neg(sum)
	}
	return res, nil
}

// InvNewton computes 1/A mod x^prec using Newton iteration: X_{k+1} = X_k * (2 - A * X_k) mod x^(2m).
func (f *DenseFPS) InvNewton(prec int) (*DenseFPS, error) {
	if len(f.Coeffs) == 0 || f.Coeffs[0].Sign() == 0 {
		return nil, fmt.Errorf("%s", i18n.T("fps.err_zero_constant_inv"))
	}
	invA0 := new(big.Rat).Inv(f.Coeffs[0])
	curr := NewDenseFPS(f.Var, 1)
	curr.Coeffs[0].Set(invA0)

	two := big.NewRat(2, 1)

	m := 1
	for m < prec {
		mNext := m * 2
		if mNext > prec {
			mNext = prec
		}

		// A truncated to mNext
		aTrunc := NewDenseFPS(f.Var, mNext)
		for i := 0; i < mNext && i < len(f.Coeffs); i++ {
			aTrunc.Coeffs[i].Set(f.Coeffs[i])
		}

		// R = A * curr mod x^mNext
		r := aTrunc.MulTrunc(curr, mNext)

		// twoMinusR = 2 - R mod x^mNext
		twoMinusR := NewDenseFPS(f.Var, mNext)
		twoMinusR.Coeffs[0].Sub(two, r.Coeffs[0])
		for i := 1; i < mNext; i++ {
			twoMinusR.Coeffs[i].Neg(r.Coeffs[i])
		}

		// curr = curr * twoMinusR mod x^mNext
		curr = curr.MulTrunc(twoMinusR, mNext)
		m = mNext
	}
	return curr, nil
}

// Inv calculates 1/A mod x^prec using the primary ODE recurrence path.
func (f *DenseFPS) Inv(prec int) (*DenseFPS, error) {
	return f.InvRecurrence(prec)
}

// ExpRecurrence computes exp(A) mod x^prec using ODE recurrence E' = A' * E:
// e_k = 1/k * sum_{i=1}^k i * a_i * e_{k-i} (with e_0 = 1, requires a_0 = 0).
func (f *DenseFPS) ExpRecurrence(prec int) (*DenseFPS, error) {
	if len(f.Coeffs) > 0 && f.Coeffs[0].Sign() != 0 {
		return nil, fmt.Errorf("%s", i18n.T("fps.err_nonzero_constant_exp"))
	}
	res := NewDenseFPS(f.Var, prec)
	res.Coeffs[0].SetInt64(1)

	term := new(big.Rat)
	sum := new(big.Rat)
	kRat := new(big.Rat)
	iRat := new(big.Rat)

	for k := 1; k < prec; k++ {
		sum.SetInt64(0)
		for i := 1; i <= k && i < len(f.Coeffs); i++ {
			if f.Coeffs[i].Sign() == 0 || res.Coeffs[k-i].Sign() == 0 {
				continue
			}
			iRat.SetInt64(int64(i))
			term.Mul(iRat, f.Coeffs[i])
			term.Mul(term, res.Coeffs[k-i])
			sum.Add(sum, term)
		}
		kRat.SetInt64(int64(k))
		res.Coeffs[k].Quo(sum, kRat)
	}
	return res, nil
}

// ExpNewton computes exp(A) mod x^prec using Newton iteration X_{k+1} = X_k * (1 + A - log(X_k)) mod x^(2m).
func (f *DenseFPS) ExpNewton(prec int) (*DenseFPS, error) {
	if len(f.Coeffs) > 0 && f.Coeffs[0].Sign() != 0 {
		return nil, fmt.Errorf("%s", i18n.T("fps.err_nonzero_constant_exp"))
	}
	curr := NewDenseFPS(f.Var, 1)
	curr.Coeffs[0].SetInt64(1)

	m := 1
	for m < prec {
		mNext := m * 2
		if mNext > prec {
			mNext = prec
		}

		logCurr, err := curr.Log(mNext)
		if err != nil {
			return nil, err
		}

		// factor = 1 + A - logCurr mod x^mNext
		factor := NewDenseFPS(f.Var, mNext)
		factor.Coeffs[0].SetInt64(1)
		for i := 0; i < mNext; i++ {
			if i < len(f.Coeffs) {
				factor.Coeffs[i].Add(factor.Coeffs[i], f.Coeffs[i])
			}
			if i < len(logCurr.Coeffs) {
				factor.Coeffs[i].Sub(factor.Coeffs[i], logCurr.Coeffs[i])
			}
		}

		curr = curr.MulTrunc(factor, mNext)
		m = mNext
	}
	return curr, nil
}

// Exp calculates exp(A) mod x^prec using the primary ODE recurrence path.
func (f *DenseFPS) Exp(prec int) (*DenseFPS, error) {
	return f.ExpRecurrence(prec)
}

// LogRecurrence computes ln(A) mod x^prec using ODE recurrence A * L' = A':
// l_k = 1 / (k * a_0) * (k * a_k - sum_{i=1}^{k-1} i * l_i * a_{k-i}) (requires a_0 = 1).
func (f *DenseFPS) LogRecurrence(prec int) (*DenseFPS, error) {
	if len(f.Coeffs) == 0 || f.Coeffs[0].Cmp(big.NewRat(1, 1)) != 0 {
		return nil, fmt.Errorf("%s", i18n.T("fps.err_constant_must_be_one_log"))
	}
	res := NewDenseFPS(f.Var, prec)
	res.Coeffs[0].SetInt64(0) // log(1) = 0

	term := new(big.Rat)
	sum := new(big.Rat)
	kRat := new(big.Rat)
	iRat := new(big.Rat)

	for k := 1; k < prec; k++ {
		sum.SetInt64(0)
		for i := 1; i < k; i++ {
			if res.Coeffs[i].Sign() == 0 {
				continue
			}
			idxA := k - i
			if idxA < len(f.Coeffs) && f.Coeffs[idxA].Sign() != 0 {
				iRat.SetInt64(int64(i))
				term.Mul(iRat, res.Coeffs[i])
				term.Mul(term, f.Coeffs[idxA])
				sum.Add(sum, term)
			}
		}
		kRat.SetInt64(int64(k))
		num := new(big.Rat)
		if k < len(f.Coeffs) {
			num.Mul(kRat, f.Coeffs[k])
		}
		num.Sub(num, sum)
		res.Coeffs[k].Quo(num, kRat)
	}
	return res, nil
}

// Log computes ln(A) mod x^prec.
func (f *DenseFPS) Log(prec int) (*DenseFPS, error) {
	return f.LogRecurrence(prec)
}

// SqrtRecurrence computes sqrt(A) mod x^prec using recurrence S^2 = A:
// s_k = 1 / (2 * s_0) * (a_k - sum_{i=1}^{k-1} s_i * s_{k-i})
// Requires a_0 to be a positive rational whose numerator and denominator are perfect squares.
func (f *DenseFPS) SqrtRecurrence(prec int) (*DenseFPS, error) {
	if len(f.Coeffs) == 0 || f.Coeffs[0].Sign() <= 0 {
		return nil, fmt.Errorf("%s", i18n.T("fps.err_non_square_constant_sqrt"))
	}
	numSqrt, okNum := exactIntSqrt(f.Coeffs[0].Num())
	denSqrt, okDen := exactIntSqrt(f.Coeffs[0].Denom())
	if !okNum || !okDen {
		return nil, fmt.Errorf("%s", i18n.T("fps.err_non_square_constant_sqrt"))
	}

	s0 := new(big.Rat).SetFrac(numSqrt, denSqrt)
	twoS0 := new(big.Rat).Mul(big.NewRat(2, 1), s0)

	res := NewDenseFPS(f.Var, prec)
	res.Coeffs[0].Set(s0)

	term := new(big.Rat)
	sum := new(big.Rat)
	for k := 1; k < prec; k++ {
		sum.SetInt64(0)
		for i := 1; i < k; i++ {
			if res.Coeffs[i].Sign() == 0 || res.Coeffs[k-i].Sign() == 0 {
				continue
			}
			term.Mul(res.Coeffs[i], res.Coeffs[k-i])
			sum.Add(sum, term)
		}
		ak := big.NewRat(0, 1)
		if k < len(f.Coeffs) {
			ak = f.Coeffs[k]
		}
		diff := new(big.Rat).Sub(ak, sum)
		res.Coeffs[k].Quo(diff, twoS0)
	}
	return res, nil
}

// Sqrt computes sqrt(A) mod x^prec.
func (f *DenseFPS) Sqrt(prec int) (*DenseFPS, error) {
	return f.SqrtRecurrence(prec)
}

// exactIntSqrt returns integer square root if n is a perfect square.
func exactIntSqrt(n *big.Int) (*big.Int, bool) {
	if n.Sign() < 0 {
		return nil, false
	}
	if n.Sign() == 0 {
		return big.NewInt(0), true
	}
	sqrt := new(big.Int).Sqrt(n)
	check := new(big.Int).Mul(sqrt, sqrt)
	if check.Cmp(n) == 0 {
		return sqrt, true
	}
	return nil, false
}

// -------------------------------------------------------------------------
// Combinatorics Functions
// -------------------------------------------------------------------------

// EvalPartitions computes the number of partitions p(n) via Euler's pentagonal number theorem.
func EvalPartitions(n int64) (*big.Int, error) {
	if n < 0 {
		return nil, fmt.Errorf("%s", i18n.T("fps.err_negative_partitions"))
	}
	if n == 0 {
		return big.NewInt(1), nil
	}

	p := make([]*big.Int, n+1)
	p[0] = big.NewInt(1)

	for i := int64(1); i <= n; i++ {
		sum := big.NewInt(0)
		for k := int64(1); ; k++ {
			// g1 = k(3k - 1) / 2
			g1 := (k * (3*k - 1)) / 2
			// g2 = k(3k + 1) / 2
			g2 := (k * (3*k + 1)) / 2

			if g1 > i && g2 > i {
				break
			}

			sign := int64(1)
			if k%2 == 0 {
				sign = -1
			}

			if g1 <= i {
				if sign > 0 {
					sum.Add(sum, p[i-g1])
				} else {
					sum.Sub(sum, p[i-g1])
				}
			}
			if g2 <= i {
				if sign > 0 {
					sum.Add(sum, p[i-g2])
				} else {
					sum.Sub(sum, p[i-g2])
				}
			}
		}
		p[i] = sum
	}
	return p[n], nil
}

// EvalBell computes the n-th Bell number B(n) using Aitken's array (Bell triangle).
func EvalBell(n int64) (*big.Int, error) {
	if n < 0 {
		return nil, fmt.Errorf("%s", i18n.T("fps.err_negative_bell"))
	}
	if n == 0 {
		return big.NewInt(1), nil
	}

	prevRow := []*big.Int{big.NewInt(1)}
	for r := int64(1); r <= n; r++ {
		currRow := make([]*big.Int, r+1)
		currRow[0] = new(big.Int).Set(prevRow[len(prevRow)-1])
		for c := int64(1); c <= r; c++ {
			currRow[c] = new(big.Int).Add(currRow[c-1], prevRow[c-1])
		}
		prevRow = currRow
	}
	return prevRow[0], nil
}

// EvalCatalan computes the n-th Catalan number C_n = (2n)! / ((n+1)! * n!).
func EvalCatalan(n int64) (*big.Int, error) {
	if n < 0 {
		return nil, fmt.Errorf("%s", i18n.T("fps.err_negative_catalan"))
	}
	if n == 0 {
		return big.NewInt(1), nil
	}

	c := big.NewInt(1)
	num := new(big.Int)
	den := new(big.Int)
	for i := int64(0); i < n; i++ {
		// c_{i+1} = c_i * 2*(2i + 1) / (i + 2)
		num.SetInt64(2 * (2*i + 1))
		den.SetInt64(i + 2)
		c.Mul(c, num)
		c.Quo(c, den)
	}
	return c, nil
}

// EvalStirling1 computes unsigned Stirling numbers of the first kind [n, k].
func EvalStirling1(n, k int64) (*big.Int, error) {
	if n < 0 || k < 0 {
		return nil, fmt.Errorf("%s", i18n.T("fps.err_negative_stirling"))
	}
	if n == 0 && k == 0 {
		return big.NewInt(1), nil
	}
	if n == 0 || k == 0 || k > n {
		return big.NewInt(0), nil
	}

	row := make([]*big.Int, k+1)
	for j := range row {
		row[j] = big.NewInt(0)
	}
	row[0] = big.NewInt(0)
	row[1] = big.NewInt(1)

	nMinus1 := new(big.Int)
	term := new(big.Int)
	for i := int64(2); i <= n; i++ {
		nextRow := make([]*big.Int, k+1)
		for j := range nextRow {
			nextRow[j] = big.NewInt(0)
		}
		nMinus1.SetInt64(i - 1)
		for j := int64(1); j <= k && j <= i; j++ {
			// s1(i, j) = (i-1)*s1(i-1, j) + s1(i-1, j-1)
			term.Mul(nMinus1, row[j])
			nextRow[j].Add(term, row[j-1])
		}
		row = nextRow
	}
	return row[k], nil
}

// EvalStirling2 computes Stirling numbers of the second kind {n, k}.
func EvalStirling2(n, k int64) (*big.Int, error) {
	if n < 0 || k < 0 {
		return nil, fmt.Errorf("%s", i18n.T("fps.err_negative_stirling"))
	}
	if n == 0 && k == 0 {
		return big.NewInt(1), nil
	}
	if n == 0 || k == 0 || k > n {
		return big.NewInt(0), nil
	}

	row := make([]*big.Int, k+1)
	for j := range row {
		row[j] = big.NewInt(0)
	}
	row[1] = big.NewInt(1)

	jBig := new(big.Int)
	term := new(big.Int)
	for i := int64(2); i <= n; i++ {
		nextRow := make([]*big.Int, k+1)
		for j := range nextRow {
			nextRow[j] = big.NewInt(0)
		}
		for j := int64(1); j <= k && j <= i; j++ {
			// s2(i, j) = j * s2(i-1, j) + s2(i-1, j-1)
			jBig.SetInt64(j)
			term.Mul(jBig, row[j])
			nextRow[j].Add(term, row[j-1])
		}
		row = nextRow
	}
	return row[k], nil
}

// -------------------------------------------------------------------------
// AST Node Conversion Helpers
// -------------------------------------------------------------------------

// DenseFPSFromNode converts a polynomial AST node into a DenseFPS.
func DenseFPSFromNode(n ast.Node, varName string, prec int) (*DenseFPS, error) {
	fps := NewDenseFPS(varName, prec)
	if n == nil {
		return fps, nil
	}

	poly, err := NodeToPoly(n, []string{varName}, ast.OrderLex)
	if err != nil {
		// Fallback for simple rational number
		if r, ok := n.(*ast.RationalNode); ok {
			fps.Coeffs[0].Set(r.Val)
			return fps, nil
		}
		return nil, fmt.Errorf("%s", i18n.T("fps.err_cannot_convert_node_to_fps", n.String()))
	}

	varIdx := -1
	for idx, v := range poly.Vars {
		if v == varName {
			varIdx = idx
			break
		}
	}

	for _, m := range poly.Terms {
		deg := 0
		if varIdx >= 0 && varIdx < len(m.Exponents) {
			deg = m.Exponents[varIdx]
		}
		for idx, exp := range m.Exponents {
			if idx != varIdx && exp > 0 {
				return nil, fmt.Errorf("%s", i18n.T("fps.err_univariate_required_found_var", poly.Vars[idx]))
			}
		}

		if deg < prec {
			fps.Coeffs[deg].Add(fps.Coeffs[deg], m.Coeff)
		}
	}
	return fps, nil
}

// DenseFPSToNode converts DenseFPS back to an AST Node (polynomial sum).
func DenseFPSToNode(fps *DenseFPS) ast.Node {
	var terms []ast.Node
	for deg, coeff := range fps.Coeffs {
		if coeff == nil || coeff.Sign() == 0 {
			continue
		}
		var term ast.Node
		coeffNode := ast.NewRationalFromBigRat(new(big.Rat).Set(coeff))
		if deg == 0 {
			term = coeffNode
		} else {
			var varTerm ast.Node
			if deg == 1 {
				varTerm = ast.NewVar(fps.Var)
			} else {
				degNode := ast.NewRationalFromBigRat(big.NewRat(int64(deg), 1))
				powNode, err := ast.NewPow(ast.NewVar(fps.Var), degNode)
				if err != nil {
					continue
				}
				varTerm = powNode
			}

			if coeff.Cmp(big.NewRat(1, 1)) == 0 {
				term = varTerm
			} else if coeff.Cmp(big.NewRat(-1, 1)) == 0 {
				negOne := ast.NewRationalFromBigRat(big.NewRat(-1, 1))
				term = ast.NewMul([]ast.Node{negOne, varTerm})
			} else {
				term = ast.NewMul([]ast.Node{coeffNode, varTerm})
			}
		}
		terms = append(terms, term)
	}

	if len(terms) == 0 {
		return ast.NewRationalFromBigRat(big.NewRat(0, 1))
	}
	if len(terms) == 1 {
		return terms[0]
	}
	return ast.NewAdd(terms)
}

// -------------------------------------------------------------------------
// CLI / REPL Handlers
// -------------------------------------------------------------------------

func init() {
	RegisterFunction(FunctionSpec{
		Name:            "fps_inv",
		MinArgs:         2,
		MaxArgs:         3,
		AllowBareSymbol: true,
		Handler:         handleFPSInv,
	})
	RegisterFunction(FunctionSpec{
		Name:            "fps_exp",
		MinArgs:         2,
		MaxArgs:         3,
		AllowBareSymbol: true,
		Handler:         handleFPSExp,
	})
	RegisterFunction(FunctionSpec{
		Name:            "fps_log",
		MinArgs:         2,
		MaxArgs:         3,
		AllowBareSymbol: true,
		Handler:         handleFPSLog,
	})
	RegisterFunction(FunctionSpec{
		Name:            "fps_sqrt",
		MinArgs:         2,
		MaxArgs:         3,
		AllowBareSymbol: true,
		Handler:         handleFPSSqrt,
	})
	RegisterFunction(FunctionSpec{
		Name:            "partitions",
		MinArgs:         1,
		MaxArgs:         1,
		AllowBareSymbol: true,
		Handler:         handlePartitions,
	})
	RegisterFunction(FunctionSpec{
		Name:            "bell",
		MinArgs:         1,
		MaxArgs:         1,
		AllowBareSymbol: true,
		Handler:         handleBell,
	})
	RegisterFunction(FunctionSpec{
		Name:            "catalan",
		MinArgs:         1,
		MaxArgs:         1,
		AllowBareSymbol: true,
		Handler:         handleCatalan,
	})
	RegisterFunction(FunctionSpec{
		Name:            "stirling1",
		MinArgs:         2,
		MaxArgs:         2,
		AllowBareSymbol: true,
		Handler:         handleStirling1,
	})
	RegisterFunction(FunctionSpec{
		Name:            "stirling2",
		MinArgs:         2,
		MaxArgs:         2,
		AllowBareSymbol: true,
		Handler:         handleStirling2,
	})
}

func parseFPSArgs(args []Node) (ast.Node, string, int, error) {
	if len(args) == 2 {
		expr := args[0]
		r, ok := args[1].(*ast.RationalNode)
		if !ok || !r.Val.IsInt() || r.Val.Sign() <= 0 {
			return nil, "", 0, fmt.Errorf("%s", i18n.T("fps.err_prec_must_be_positive_int", args[1].String()))
		}
		prec := int(r.Val.Num().Int64())
		varName := "x"
		vars := CollectFreeVariables(expr)
		if len(vars) > 0 {
			varName = vars[0]
		}
		return expr, varName, prec, nil
	} else if len(args) == 3 {
		expr := args[0]
		v, ok := args[1].(*ast.VarNode)
		if !ok {
			return nil, "", 0, fmt.Errorf("%s", i18n.T("fps.err_second_arg_must_be_var", args[1].String()))
		}
		r, ok := args[2].(*ast.RationalNode)
		if !ok || !r.Val.IsInt() || r.Val.Sign() <= 0 {
			return nil, "", 0, fmt.Errorf("%s", i18n.T("fps.err_third_arg_must_be_positive_int", args[2].String()))
		}
		prec := int(r.Val.Num().Int64())
		return expr, v.Name, prec, nil
	}
	return nil, "", 0, fmt.Errorf("%s", i18n.T("fps.err_invalid_arg_count"))
}

func handleFPSInv(args []Node, env *Env) (Node, error) {
	expr, varName, prec, err := parseFPSArgs(args)
	if err != nil {
		return nil, err
	}
	fps, err := DenseFPSFromNode(expr, varName, prec)
	if err != nil {
		return nil, err
	}
	res, err := fps.Inv(prec)
	if err != nil {
		return nil, err
	}
	return DenseFPSToNode(res), nil
}

func handleFPSExp(args []Node, env *Env) (Node, error) {
	expr, varName, prec, err := parseFPSArgs(args)
	if err != nil {
		return nil, err
	}
	fps, err := DenseFPSFromNode(expr, varName, prec)
	if err != nil {
		return nil, err
	}
	res, err := fps.Exp(prec)
	if err != nil {
		return nil, err
	}
	return DenseFPSToNode(res), nil
}

func handleFPSLog(args []Node, env *Env) (Node, error) {
	expr, varName, prec, err := parseFPSArgs(args)
	if err != nil {
		return nil, err
	}
	fps, err := DenseFPSFromNode(expr, varName, prec)
	if err != nil {
		return nil, err
	}
	res, err := fps.Log(prec)
	if err != nil {
		return nil, err
	}
	return DenseFPSToNode(res), nil
}

func handleFPSSqrt(args []Node, env *Env) (Node, error) {
	expr, varName, prec, err := parseFPSArgs(args)
	if err != nil {
		return nil, err
	}
	fps, err := DenseFPSFromNode(expr, varName, prec)
	if err != nil {
		return nil, err
	}
	res, err := fps.Sqrt(prec)
	if err != nil {
		return nil, err
	}
	return DenseFPSToNode(res), nil
}

func handlePartitions(args []Node, env *Env) (Node, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("%s", i18n.T("fps.err_partitions_arg_count"))
	}
	r, ok := args[0].(*ast.RationalNode)
	if !ok || !r.Val.IsInt() || r.Val.Sign() < 0 {
		return nil, fmt.Errorf("%s", i18n.T("fps.err_partitions_arg_must_be_non_negative", args[0].String()))
	}
	p, err := EvalPartitions(r.Val.Num().Int64())
	if err != nil {
		return nil, err
	}
	return ast.NewRationalFromBigRat(new(big.Rat).SetInt(p)), nil
}

func handleBell(args []Node, env *Env) (Node, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("%s", i18n.T("fps.err_bell_arg_count"))
	}
	r, ok := args[0].(*ast.RationalNode)
	if !ok || !r.Val.IsInt() || r.Val.Sign() < 0 {
		return nil, fmt.Errorf("%s", i18n.T("fps.err_bell_arg_must_be_non_negative", args[0].String()))
	}
	b, err := EvalBell(r.Val.Num().Int64())
	if err != nil {
		return nil, err
	}
	return ast.NewRationalFromBigRat(new(big.Rat).SetInt(b)), nil
}

func handleCatalan(args []Node, env *Env) (Node, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("%s", i18n.T("fps.err_catalan_arg_count"))
	}
	r, ok := args[0].(*ast.RationalNode)
	if !ok || !r.Val.IsInt() || r.Val.Sign() < 0 {
		return nil, fmt.Errorf("%s", i18n.T("fps.err_catalan_arg_must_be_non_negative", args[0].String()))
	}
	c, err := EvalCatalan(r.Val.Num().Int64())
	if err != nil {
		return nil, err
	}
	return ast.NewRationalFromBigRat(new(big.Rat).SetInt(c)), nil
}

func handleStirling1(args []Node, env *Env) (Node, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("%s", i18n.T("fps.err_stirling_arg_count"))
	}
	r1, ok1 := args[0].(*ast.RationalNode)
	r2, ok2 := args[1].(*ast.RationalNode)
	if !ok1 || !ok2 || !r1.Val.IsInt() || !r2.Val.IsInt() || r1.Val.Sign() < 0 || r2.Val.Sign() < 0 {
		return nil, fmt.Errorf("%s", i18n.T("fps.err_stirling_args_must_be_non_negative"))
	}
	s, err := EvalStirling1(r1.Val.Num().Int64(), r2.Val.Num().Int64())
	if err != nil {
		return nil, err
	}
	return ast.NewRationalFromBigRat(new(big.Rat).SetInt(s)), nil
}

func handleStirling2(args []Node, env *Env) (Node, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("%s", i18n.T("fps.err_stirling_arg_count"))
	}
	r1, ok1 := args[0].(*ast.RationalNode)
	r2, ok2 := args[1].(*ast.RationalNode)
	if !ok1 || !ok2 || !r1.Val.IsInt() || !r2.Val.IsInt() || r1.Val.Sign() < 0 || r2.Val.Sign() < 0 {
		return nil, fmt.Errorf("%s", i18n.T("fps.err_stirling_args_must_be_non_negative"))
	}
	s, err := EvalStirling2(r1.Val.Num().Int64(), r2.Val.Num().Int64())
	if err != nil {
		return nil, err
	}
	return ast.NewRationalFromBigRat(new(big.Rat).SetInt(s)), nil
}
