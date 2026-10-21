package calc_test

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/units"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/lp"
)

// ProfileType represents the target domain profile for AST generation.
type ProfileType int

const (
	ProfileGeneral ProfileType = iota
	ProfileAlgebra
	ProfileCalculus
	ProfileTransforms
	ProfileSpecial
	ProfilePolynomial
	ProfileRational
	ProfileUnits
	ProfileSimplex
	ProfileMatrix
)

// ByteStreamProvider decodes an arbitrary byte stream into deterministic primitives and AST nodes.
// Invariant: Totality guarantee - it never panics or calls t.Skip, providing deterministic repair on depletion.
type ByteStreamProvider struct {
	data   []byte
	offset int
}

// NewByteStreamProvider creates a new ByteStreamProvider wrapping the given byte slice.
func NewByteStreamProvider(data []byte) *ByteStreamProvider {
	return &ByteStreamProvider{
		data:   data,
		offset: 0,
	}
}

// NextByte returns the next byte from the stream, or 0 if depleted.
func (p *ByteStreamProvider) NextByte() byte {
	if p.offset >= len(p.data) {
		return 0
	}
	b := p.data[p.offset]
	p.offset++
	return b
}

// NextBool returns a pseudo-random boolean from the stream.
func (p *ByteStreamProvider) NextBool() bool {
	return (p.NextByte() & 1) == 1
}

// NextChoice returns an integer in [0, n) determined by the stream, or 0 if n <= 0.
func (p *ByteStreamProvider) NextChoice(n int) int {
	if n <= 1 {
		return 0
	}
	b := p.NextByte()
	return int(b) % n
}

// NextInt returns an integer in [min, max] determined by the stream.
// If min >= max, min is returned.
func (p *ByteStreamProvider) NextInt(min, max int64) int64 {
	if min >= max {
		return min
	}
	diff := uint64(max - min + 1)
	b1 := uint64(p.NextByte())
	b2 := uint64(p.NextByte())
	val := (b1<<8 | b2) % diff
	return min + int64(val)
}

// GenNode constructs a valid ast.Node within the given depth and node budget limits.
// If maxDepth <= 0, *nodeBudget <= 1, or the byte stream is depleted, a terminal leaf node is immediately returned.
func (p *ByteStreamProvider) GenNode(maxDepth int, nodeBudget *int, profile ProfileType) ast.Node {
	if nodeBudget == nil {
		defaultBudget := 16
		nodeBudget = &defaultBudget
	}

	if maxDepth <= 0 || *nodeBudget <= 1 || p.offset >= len(p.data) {
		if *nodeBudget > 0 {
			*nodeBudget--
		}
		return p.genLeafNode(profile)
	}

	*nodeBudget--

	if profile == ProfileCalculus {
		// Composite expression tree of polynomials, trig, rational exponents, exp, and ln
		choice := p.NextChoice(8)
		switch choice {
		case 0: // Add
			t1, t2 := p.genBinaryChildren(maxDepth, nodeBudget, profile)
			return ast.NewAdd([]ast.Node{t1, t2})
		case 1: // Mul
			f1, f2 := p.genBinaryChildren(maxDepth, nodeBudget, profile)
			return ast.NewMul([]ast.Node{f1, f2})
		case 2: // Pow with rational exponent (p/q with q in {1, 2, 3})
			base := p.GenNode(maxDepth-1, nodeBudget, profile)
			if *nodeBudget > 0 {
				*nodeBudget--
			}
			num := p.NextInt(-2, 3)
			den := p.NextInt(1, 3)
			exp := ast.NewRationalFromBigRat(big.NewRat(num, den))
			pow, err := ast.NewPow(base, exp)
			if err != nil {
				return p.genLeafNode(profile)
			}
			return pow
		case 3: // UnaryOp (-)
			expr := p.GenNode(maxDepth-1, nodeBudget, profile)
			uop, err := ast.NewUnaryOp("-", expr)
			if err != nil {
				return p.genLeafNode(profile)
			}
			return uop
		case 4: // Trig: sin or cos
			arg := p.GenNode(maxDepth-1, nodeBudget, profile)
			trigFunc := "sin"
			if p.NextBool() {
				trigFunc = "cos"
			}
			fn, err := ast.NewFunc(trigFunc, []ast.Node{arg})
			if err != nil {
				return p.genLeafNode(profile)
			}
			return fn
		case 5: // exp
			arg := p.GenNode(maxDepth-1, nodeBudget, profile)
			fn, err := ast.NewFunc("exp", []ast.Node{arg})
			if err != nil {
				return p.genLeafNode(profile)
			}
			return fn
		case 6: // ln
			arg := p.GenNode(maxDepth-1, nodeBudget, profile)
			fn, err := ast.NewFunc("ln", []ast.Node{arg})
			if err != nil {
				return p.genLeafNode(profile)
			}
			return fn
		default: // Leaf
			return p.genLeafNode(profile)
		}
	}

	if profile == ProfileTransforms {
		// Discrete sequence terms: a^n, n^k, delta(n), u(n), shifted delta(n-k), linear combinations
		choice := p.NextChoice(7)
		switch choice {
		case 0: // Add
			t1, t2 := p.genBinaryChildren(maxDepth, nodeBudget, profile)
			return ast.NewAdd([]ast.Node{t1, t2})
		case 1: // Mul (e.g. constant * term or n * a^n)
			f1, f2 := p.genBinaryChildren(maxDepth, nodeBudget, profile)
			return ast.NewMul([]ast.Node{f1, f2})
		case 2: // Geometric a^n with rational base
			if *nodeBudget >= 2 {
				*nodeBudget -= 2
			} else {
				*nodeBudget = 0
			}
			baseNum := p.NextInt(-4, 4)
			if baseNum == 0 {
				baseNum = 2
			}
			baseDen := p.NextInt(1, 3)
			base := ast.NewRationalFromBigRat(big.NewRat(baseNum, baseDen))
			pow, err := ast.NewPow(base, ast.NewVar("n"))
			if err != nil {
				return p.genLeafNode(profile)
			}
			return pow
		case 3: // Ramp / Polynomial n^k
			if *nodeBudget >= 2 {
				*nodeBudget -= 2
			} else {
				*nodeBudget = 0
			}
			expVal := p.NextInt(1, 3)
			pow, err := ast.NewPow(ast.NewVar("n"), ast.NewRationalFromBigRat(big.NewRat(expVal, 1)))
			if err != nil {
				return ast.NewVar("n")
			}
			return pow
		case 4: // delta(n) or shifted delta(n - k)
			shift := p.NextInt(0, 3)
			var arg ast.Node = ast.NewVar("n")
			if shift > 0 && *nodeBudget >= 3 {
				*nodeBudget -= 3
				arg = ast.NewAdd([]ast.Node{ast.NewVar("n"), ast.NewRationalFromBigRat(big.NewRat(-shift, 1))})
			} else if *nodeBudget > 0 {
				*nodeBudget--
			}
			fn, err := ast.NewFunc("delta", []ast.Node{arg})
			if err != nil {
				return p.genLeafNode(profile)
			}
			return fn
		case 5: // Step function u(n)
			if *nodeBudget > 0 {
				*nodeBudget--
			}
			fn, err := ast.NewFunc("u", []ast.Node{ast.NewVar("n")})
			if err != nil {
				return p.genLeafNode(profile)
			}
			return fn
		default: // Leaf
			return p.genLeafNode(profile)
		}
	}
	if profile == ProfilePolynomial {
		choice := p.NextChoice(5)
		switch choice {
		case 0: // Add
			t1, t2 := p.genBinaryChildren(maxDepth, nodeBudget, profile)
			return ast.NewAdd([]ast.Node{t1, t2})
		case 1: // Mul
			f1, f2 := p.genBinaryChildren(maxDepth, nodeBudget, profile)
			return ast.NewMul([]ast.Node{f1, f2})
		case 2: // Pow with integer exp [0, 4]
			base := p.GenNode(maxDepth-1, nodeBudget, profile)
			if *nodeBudget > 0 {
				*nodeBudget--
			}
			exp := ast.NewRationalFromBigRat(big.NewRat(p.NextInt(0, 4), 1))
			pow, err := ast.NewPow(base, exp)
			if err != nil {
				return p.genLeafNode(profile)
			}
			return pow
		case 3: // UnaryOp
			expr := p.GenNode(maxDepth-1, nodeBudget, profile)
			uop, err := ast.NewUnaryOp("-", expr)
			if err != nil {
				return p.genLeafNode(profile)
			}
			return uop
		default: // Leaf
			return p.genLeafNode(profile)
		}
	}

	if profile == ProfileRational {
		choice := p.NextChoice(6)
		switch choice {
		case 0: // Add
			t1, t2 := p.genBinaryChildren(maxDepth, nodeBudget, profile)
			return ast.NewAdd([]ast.Node{t1, t2})
		case 1: // Mul
			f1, f2 := p.genBinaryChildren(maxDepth, nodeBudget, profile)
			return ast.NewMul([]ast.Node{f1, f2})
		case 2: // Pow with integer exp [-2, 3]
			base := p.GenNode(maxDepth-1, nodeBudget, profile)
			if *nodeBudget > 0 {
				*nodeBudget--
			}
			expVal := p.NextInt(-2, 3)
			if expVal == 0 {
				expVal = 1
			}
			exp := ast.NewRationalFromBigRat(big.NewRat(expVal, 1))
			pow, err := ast.NewPow(base, exp)
			if err != nil {
				return p.genLeafNode(profile)
			}
			return pow
		case 3: // Rational fraction / division
			num, den := p.genBinaryChildren(maxDepth, nodeBudget, profile)
			if *nodeBudget >= 2 {
				*nodeBudget -= 2
			} else {
				*nodeBudget = 0
			}
			invDen, err := ast.NewPow(den, ast.NewRationalFromBigRat(big.NewRat(-1, 1)))
			if err != nil {
				return num
			}
			return ast.NewMul([]ast.Node{num, invDen})
		case 4: // UnaryOp
			expr := p.GenNode(maxDepth-1, nodeBudget, profile)
			uop, err := ast.NewUnaryOp("-", expr)
			if err != nil {
				return p.genLeafNode(profile)
			}
			return uop
		default: // Leaf
			return p.genLeafNode(profile)
		}
	}

	if profile == ProfileUnits {
		// Physical units expressions: combinations of quantities with +, -, *, /, ^
		choice := p.NextChoice(5)
		switch choice {
		case 0: // Add
			t1, t2 := p.genBinaryChildren(maxDepth, nodeBudget, profile)
			return ast.NewAdd([]ast.Node{t1, t2})
		case 1: // Mul
			f1, f2 := p.genBinaryChildren(maxDepth, nodeBudget, profile)
			return ast.NewMul([]ast.Node{f1, f2})
		case 2: // Div (f1 * f2^(-1))
			f1, f2 := p.genBinaryChildren(maxDepth, nodeBudget, profile)
			if *nodeBudget >= 2 {
				*nodeBudget -= 2
			} else {
				*nodeBudget = 0
			}
			invF2, err := ast.NewPow(f2, ast.NewRationalFromBigRat(big.NewRat(-1, 1)))
			if err != nil {
				return f1
			}
			return ast.NewMul([]ast.Node{f1, invF2})
		case 3: // Pow with small integer exponent [-2, 2]
			base := p.GenNode(maxDepth-1, nodeBudget, profile)
			if *nodeBudget > 0 {
				*nodeBudget--
			}
			expVal := p.NextInt(-2, 2)
			if expVal == 0 {
				expVal = 1
			}
			pow, err := ast.NewPow(base, ast.NewRationalFromBigRat(big.NewRat(expVal, 1)))
			if err != nil {
				return base
			}
			return pow
		default: // Leaf
			return p.genLeafNode(profile)
		}
	}

	if profile == ProfileMatrix {
		dim := p.NextChoice(2) + 2 // 2x2 or 3x3
		return p.GenMatrixNode(dim, dim, maxDepth-1, nodeBudget)
	}

	if profile == ProfileSimplex {
		return p.genLeafNode(profile)
	}

	// 25 distinct AST node types supported across ihd
	nodeChoice := p.NextChoice(25)

	switch nodeChoice {
	case 0: // Rational
		num := p.NextInt(-10, 10)
		denom := p.NextInt(1, 10)
		r, err := ast.NewRational(num, denom)
		if err != nil {
			return p.genLeafNode(profile)
		}
		return r

	case 1: // Sqrt
		radicand := p.GenNode(maxDepth-1, nodeBudget, profile)
		return ast.NewSqrt(radicand)

	case 2: // Const
		consts := []string{"pi", "e", "i", "deg"}
		cIdx := p.NextChoice(len(consts))
		c, err := ast.NewConst(consts[cIdx])
		if err != nil {
			return p.genLeafNode(profile)
		}
		return c

	case 3: // Func
		funcs := []string{"sin", "cos", "tan", "ln", "exp", "abs", "floor", "ceil", "sinh", "cosh"}
		fIdx := p.NextChoice(len(funcs))
		arg := p.GenNode(maxDepth-1, nodeBudget, profile)
		fn, err := ast.NewFunc(funcs[fIdx], []ast.Node{arg})
		if err != nil {
			return p.genLeafNode(profile)
		}
		return fn

	case 4: // Complex
		re, im := p.genBinaryChildren(maxDepth, nodeBudget, profile)
		return ast.NewComplex(re, im)

	case 5: // Add
		t1, t2 := p.genBinaryChildren(maxDepth, nodeBudget, profile)
		return ast.NewAdd([]ast.Node{t1, t2})

	case 6: // Mul
		f1, f2 := p.genBinaryChildren(maxDepth, nodeBudget, profile)
		return ast.NewMul([]ast.Node{f1, f2})

	case 7: // Pow
		base, exp := p.genBinaryChildren(maxDepth, nodeBudget, profile)
		pow, err := ast.NewPow(base, exp)
		if err != nil {
			return p.genLeafNode(profile)
		}
		return pow

	case 8: // UnaryOp
		ops := []string{"-", "+"}
		op := ops[p.NextChoice(len(ops))]
		expr := p.GenNode(maxDepth-1, nodeBudget, profile)
		uop, err := ast.NewUnaryOp(op, expr)
		if err != nil {
			return p.genLeafNode(profile)
		}
		return uop

	case 9: // Var
		vars := []string{"x", "y", "z", "n", "t"}
		vIdx := p.NextChoice(len(vars))
		return ast.NewVar(vars[vIdx])

	case 10: // List
		elemCount := p.NextChoice(3) + 1
		if elemCount > *nodeBudget {
			elemCount = *nodeBudget
		}
		if elemCount <= 0 {
			elemCount = 1
		}
		elems := make([]ast.Node, elemCount)
		perElem := *nodeBudget / elemCount
		for i := 0; i < elemCount; i++ {
			subBudget := perElem
			if i == elemCount-1 {
				subBudget = *nodeBudget
			}
			elems[i] = p.GenNode(maxDepth-1, &subBudget, profile)
			*nodeBudget -= (perElem - subBudget)
		}
		return ast.NewList(elems)

	case 11: // Matrix
		rows := p.NextChoice(2) + 1
		cols := p.NextChoice(2) + 1
		return p.GenMatrixNode(rows, cols, maxDepth-1, nodeBudget)

	case 12: // Plot
		plots := []string{"sin(x)", "cos(x)", "x^2"}
		return ast.NewPlotNode(plots[p.NextChoice(len(plots))])

	case 13: // RelOp
		ops := []string{"==", "!=", "<", "<=", ">", ">="}
		op := ops[p.NextChoice(len(ops))]
		lhs, rhs := p.genBinaryChildren(maxDepth, nodeBudget, profile)
		return ast.NewRelOp(lhs, op, rhs)

	case 14: // Poly
		polyVars := []string{"x"}
		terms := []ast.Monomial{
			{
				Coeff:     big.NewRat(p.NextInt(1, 5), 1),
				Exponents: []int{int(p.NextInt(0, 3))},
			},
		}
		return &ast.PolyNode{
			Vars:  polyVars,
			Order: ast.OrderLex,
			Terms: terms,
		}

	case 15: // AlgebraicNumber
		minPoly := &ast.PolyNode{
			Vars:  []string{"x"},
			Order: ast.OrderLex,
			Terms: []ast.Monomial{
				{Coeff: big.NewRat(1, 1), Exponents: []int{2}},
				{Coeff: big.NewRat(-2, 1), Exponents: []int{0}},
			},
		}
		repPoly := &ast.PolyNode{
			Vars:  []string{"x"},
			Order: ast.OrderLex,
			Terms: []ast.Monomial{
				{Coeff: big.NewRat(1, 1), Exponents: []int{1}},
			},
		}
		return &ast.AlgebraicNumberNode{
			Symbol:  "alpha",
			MinPoly: minPoly,
			RepPoly: repPoly,
		}

	case 16: // Quantifier
		kinds := []ast.QuantifierKind{ast.QuantifierForall, ast.QuantifierExists}
		kind := kinds[p.NextChoice(len(kinds))]
		body := p.GenNode(maxDepth-1, nodeBudget, profile)
		return ast.NewQuantifier(kind, []string{"x"}, body)

	case 17: // Piecewise
		if *nodeBudget >= 4 {
			*nodeBudget -= 4
		} else {
			*nodeBudget = 0
		}
		expr, otherwise := p.genBinaryChildren(maxDepth, nodeBudget, profile)
		cond := ast.NewRelOp(ast.NewVar("x"), ">", ast.NewRationalFromBigRat(big.NewRat(0, 1)))
		return ast.NewPiecewiseNode([]ast.PiecewiseCase{{Expr: expr, Condition: cond}}, otherwise)

	case 18: // Interval
		if *nodeBudget >= 2 {
			*nodeBudget -= 2
		} else {
			*nodeBudget = 0
		}
		low := big.NewRat(p.NextInt(-10, 0), 1)
		high := big.NewRat(p.NextInt(1, 10), 1)
		return ast.NewIntervalNode(low, high)

	case 19: // String
		strs := []string{"alpha", "beta", "theta", "pi", "x"}
		return ast.NewStringNode(strs[p.NextChoice(len(strs))])

	case 20: // Index
		if *nodeBudget >= 4 {
			*nodeBudget -= 4
		} else {
			*nodeBudget = 0
		}
		target := ast.NewList([]ast.Node{p.genLeafNode(profile), p.genLeafNode(profile)})
		idxNode := ast.NewRationalFromBigRat(big.NewRat(0, 1))
		return ast.NewIndexNode(target, idxNode)

	case 21: // Slice
		if *nodeBudget >= 7 {
			*nodeBudget -= 7
		} else {
			*nodeBudget = 0
		}
		target := ast.NewList([]ast.Node{p.genLeafNode(profile), p.genLeafNode(profile), p.genLeafNode(profile)})
		start := ast.NewRationalFromBigRat(big.NewRat(0, 1))
		stop := ast.NewRationalFromBigRat(big.NewRat(2, 1))
		step := ast.NewRationalFromBigRat(big.NewRat(1, 1))
		return ast.NewSliceNode(target, start, stop, step)

	case 22: // Lambda
		body := p.GenNode(maxDepth-1, nodeBudget, profile)
		return ast.NewLambdaNode([]string{"x"}, body)

	case 23: // Apply
		if *nodeBudget >= 2 {
			*nodeBudget -= 2
		} else {
			*nodeBudget = 0
		}
		callee := ast.NewLambdaNode([]string{"x"}, ast.NewVar("x"))
		arg := p.GenNode(maxDepth-1, nodeBudget, profile)
		return ast.NewApplyNode(callee, []ast.Node{arg})

	case 24: // Quantity
		if *nodeBudget > 0 {
			*nodeBudget--
		}
		val := p.genLeafNode(profile)
		dims := ast.NewDimensionVector()
		dims[0] = big.NewRat(1, 1) // e.g. Length
		scale := big.NewRat(1, 1)
		return ast.NewQuantityNode(val, scale, dims, "m")

	default:
		return p.genLeafNode(profile)
	}
}

// genBinaryChildren explicitly splits the remaining nodeBudget between the left and right subtrees
// so the left child cannot starve the right child or exceed the overall budget.
func (p *ByteStreamProvider) genBinaryChildren(maxDepth int, nodeBudget *int, profile ProfileType) (ast.Node, ast.Node) {
	if nodeBudget == nil || *nodeBudget < 2 {
		if nodeBudget != nil {
			*nodeBudget = 0
		}
		return p.genLeafNode(profile), p.genLeafNode(profile)
	}
	remaining := *nodeBudget
	leftBudget := (remaining + 1) / 2
	rightBudget := remaining - leftBudget

	left := p.GenNode(maxDepth-1, &leftBudget, profile)
	rightBudget += leftBudget
	right := p.GenNode(maxDepth-1, &rightBudget, profile)
	*nodeBudget = rightBudget
	return left, right
}

// genLeafNode creates a deterministic terminal leaf node to guarantee Totality.
func (p *ByteStreamProvider) genLeafNode(profile ProfileType) ast.Node {
	switch profile {
	case ProfileCalculus:
		choice := p.NextChoice(4)
		switch choice {
		case 0:
			return ast.NewRationalFromBigRat(big.NewRat(p.NextInt(-3, 3), 1))
		case 1:
			return ast.NewRationalFromBigRat(big.NewRat(1, 1))
		case 2:
			return ast.NewVar("x")
		default:
			c, err := ast.NewConst("e")
			if err == nil {
				return c
			}
			return ast.NewVar("x")
		}
	case ProfileTransforms:
		choice := p.NextChoice(3)
		switch choice {
		case 0:
			return ast.NewRationalFromBigRat(big.NewRat(p.NextInt(1, 3), 1))
		case 1:
			return ast.NewVar("n")
		default:
			fn, err := ast.NewFunc("delta", []ast.Node{ast.NewVar("n")})
			if err == nil {
				return fn
			}
			return ast.NewVar("n")
		}
	case ProfileAlgebra, ProfilePolynomial, ProfileRational:
		choice := p.NextChoice(4)
		switch choice {
		case 0:
			return ast.NewRationalFromBigRat(big.NewRat(0, 1))
		case 1:
			return ast.NewRationalFromBigRat(big.NewRat(1, 1))
		case 2:
			return ast.NewVar("x")
		default:
			return ast.NewVar("y")
		}
	case ProfileUnits:
		return p.GenQuantityLeaf()
	case ProfileSimplex:
		return ast.NewRationalFromBigRat(p.NextSmallRat())
	case ProfileMatrix:
		choice := p.NextChoice(3)
		switch choice {
		case 0:
			return ast.NewRationalFromBigRat(big.NewRat(0, 1))
		case 1:
			return ast.NewRationalFromBigRat(big.NewRat(1, 1))
		default:
			return ast.NewRationalFromBigRat(big.NewRat(p.NextInt(-3, 3), 1))
		}
	default:
		choice := p.NextChoice(5)
		switch choice {
		case 0:
			return ast.NewRationalFromBigRat(big.NewRat(0, 1))
		case 1:
			return ast.NewRationalFromBigRat(big.NewRat(1, 1))
		case 2:
			return ast.NewVar("x")
		case 3:
			c, err := ast.NewConst("pi")
			if err == nil {
				return c
			}
			return ast.NewVar("x")
		default:
			return ast.NewRationalFromBigRat(big.NewRat(p.NextInt(-5, 5), 1))
		}
	}
}

// TestByteStreamProvider_TotalityAndRepair tests that ByteStreamProvider safely generates
// valid AST nodes for empty or minimal inputs without panicking.
func TestByteStreamProvider_TotalityAndRepair(t *testing.T) {
	testCases := []struct {
		name string
		data []byte
	}{
		{"Empty stream", []byte{}},
		{"Single zero byte", []byte{0x00}},
		{"Single 0xFF byte", []byte{0xFF}},
		{"Small stream (4 bytes)", []byte{0x01, 0x02, 0x03, 0x04}},
		{"16 bytes arbitrary stream", []byte{
			0xDE, 0xAD, 0xBE, 0xEF, 0x00, 0x12, 0x34, 0x56,
			0x78, 0x9A, 0xBC, 0xDE, 0xF0, 0x11, 0x22, 0x33,
		}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := NewByteStreamProvider(tc.data)

			// Primitive getters should not panic even when empty
			_ = p.NextByte()
			_ = p.NextBool()
			_ = p.NextChoice(10)
			_ = p.NextInt(-10, 10)

			// Node budget and maxDepth variations across all profiles
			profiles := []ProfileType{
				ProfileGeneral,
				ProfileAlgebra,
				ProfileCalculus,
				ProfileTransforms,
				ProfileSpecial,
				ProfilePolynomial,
				ProfileRational,
				ProfileUnits,
				ProfileSimplex,
				ProfileMatrix,
			}
			for _, prof := range profiles {
				for maxDepth := 0; maxDepth <= 3; maxDepth++ {
					budget := 16
					node := p.GenNode(maxDepth, &budget, prof)
					if node == nil {
						t.Fatalf("[%s] GenNode returned nil at maxDepth=%d, prof=%v", tc.name, maxDepth, prof)
					}
					if node.String() == "" {
						t.Fatalf("[%s] GenNode returned node with empty string representation: %v", tc.name, node)
					}
				}
			}
		})
	}
}

// NextSmallRat returns a deterministic small rational number.
func (p *ByteStreamProvider) NextSmallRat() *big.Rat {
	num := p.NextInt(-5, 5)
	den := p.NextInt(1, 4)
	return big.NewRat(num, den)
}

// GenQuantityLeaf returns a deterministic physical quantity with SI base or derived units and prefix.
func (p *ByteStreamProvider) GenQuantityLeaf() *ast.QuantityNode {
	val := ast.NewRationalFromBigRat(p.NextSmallRat())
	if val.Val.Sign() == 0 {
		val = ast.NewRationalFromBigRat(big.NewRat(1, 1))
	}

	unitsList := []string{
		"m", "kg", "s", "A", "K", "mol", "cd",
		"N", "Pa", "J", "W", "C", "V", "Hz",
		"km", "cm", "mm", "ms", "h", "min", "kN", "kJ",
	}
	unitChoice := unitsList[p.NextChoice(len(unitsList))]

	scale, dims, err := units.ResolveUnitSymbol(unitChoice)
	if err != nil {
		scale = big.NewRat(1, 1)
		dims = ast.NewDimensionVector()
		dims[0] = big.NewRat(1, 1)
		unitChoice = "m"
	}
	return ast.NewQuantityNode(val, scale, dims, unitChoice)
}

// GenSimplexLP generates a deterministic linear programming problem in standard form.
func (p *ByteStreamProvider) GenSimplexLP(n, m int) *lp.Problem {
	if n <= 0 {
		n = 2
	}
	if m <= 0 {
		m = 2
	}

	varNames := make([]string, n)
	for j := 0; j < n; j++ {
		varNames[j] = fmt.Sprintf("x%d", j+1)
	}

	objCoeffs := make([]*big.Rat, n)
	for j := 0; j < n; j++ {
		objCoeffs[j] = p.NextSmallRat()
	}

	constraints := make([]lp.Constraint, m)
	for i := 0; i < m; i++ {
		rowCoeffs := make([]*big.Rat, n)
		for j := 0; j < n; j++ {
			rowCoeffs[j] = p.NextSmallRat()
		}
		// Include negative, zero, and positive RHS to exercise Phase I (artificial variables) and infeasible/feasible regions
		rhsNum := p.NextInt(-3, 10)
		rhsDen := p.NextInt(1, 3)
		rhs := big.NewRat(rhsNum, rhsDen)

		constraints[i] = lp.Constraint{
			Name:   fmt.Sprintf("c%d", i+1),
			Coeffs: rowCoeffs,
			Sense:  lp.RelLe,
			RHS:    rhs,
		}
	}

	return &lp.Problem{
		Sense:       lp.OptMaximize,
		Objective:   objCoeffs,
		ObjConstant: new(big.Rat),
		VarNames:    varNames,
		Constraints: constraints,
	}
}

// GenMatrixNode generates a deterministic MatrixNode of size rows x cols while respecting nodeBudget.
func (p *ByteStreamProvider) GenMatrixNode(rows, cols int, maxDepth int, nodeBudget *int) *ast.MatrixNode {
	if rows <= 0 {
		rows = 2
	}
	if cols <= 0 {
		cols = 2
	}

	data := make([][]ast.Node, rows)
	for r := 0; r < rows; r++ {
		data[r] = make([]ast.Node, cols)
		for c := 0; c < cols; c++ {
			if nodeBudget != nil && *nodeBudget > 0 {
				*nodeBudget--
			}
			if p.offset >= len(p.data) || p.NextBool() || maxDepth <= 0 || (nodeBudget != nil && *nodeBudget <= 0) {
				// Rational element
				val := p.NextInt(-4, 4)
				data[r][c] = ast.NewRationalFromBigRat(big.NewRat(val, 1))
			} else {
				// Symbolic element
				vars := []string{"x", "y", "a", "b"}
				data[r][c] = ast.NewVar(vars[p.NextChoice(len(vars))])
			}
		}
	}

	mat, err := ast.NewMatrix(rows, cols, data)
	if err != nil {
		// Fallback to identity matrix
		idData := make([][]ast.Node, rows)
		for r := 0; r < rows; r++ {
			idData[r] = make([]ast.Node, cols)
			for c := 0; c < cols; c++ {
				if r == c {
					idData[r][c] = ast.NewRationalFromBigRat(big.NewRat(1, 1))
				} else {
					idData[r][c] = ast.NewRationalFromBigRat(big.NewRat(0, 1))
				}
			}
		}
		fallback, _ := ast.NewMatrix(rows, cols, idData)
		return fallback
	}
	return mat
}

// GenRationalMatrix generates a deterministic purely rational MatrixNode.
func (p *ByteStreamProvider) GenRationalMatrix(rows, cols int) *ast.MatrixNode {
	if rows <= 0 {
		rows = 2
	}
	if cols <= 0 {
		cols = 2
	}

	data := make([][]ast.Node, rows)
	for r := 0; r < rows; r++ {
		data[r] = make([]ast.Node, cols)
		for c := 0; c < cols; c++ {
			num := p.NextInt(-4, 4)
			den := p.NextInt(1, 3)
			data[r][c] = ast.NewRationalFromBigRat(big.NewRat(num, den))
		}
	}
	mat, err := ast.NewMatrix(rows, cols, data)
	if err != nil {
		idData := make([][]ast.Node, rows)
		for r := 0; r < rows; r++ {
			idData[r] = make([]ast.Node, cols)
			for c := 0; c < cols; c++ {
				if r == c {
					idData[r][c] = ast.NewRationalFromBigRat(big.NewRat(1, 1))
				} else {
					idData[r][c] = ast.NewRationalFromBigRat(big.NewRat(0, 1))
				}
			}
		}
		fallback, _ := ast.NewMatrix(rows, cols, idData)
		return fallback
	}
	return mat
}

// Suppress unused import errors if any
var _ = fmt.Sprintf
var _ = calc.NewEnv
var _ = units.FindUnit
var _ = lp.Solve

// TestByteStreamProvider_BudgetAndDepletion verifies that depleted byte streams immediately
// produce a leaf node and that node generation strictly respects nodeBudget.
func TestByteStreamProvider_BudgetAndDepletion(t *testing.T) {
	// 1. Empty byte stream should immediately terminate with a single leaf node
	emptyProvider := NewByteStreamProvider([]byte{})
	budget := 16
	node := emptyProvider.GenNode(4, &budget, ProfilePolynomial)
	if count := CountNodes(node); count != 1 {
		t.Fatalf("Expected depleted stream to immediately return 1 leaf node, got %d nodes (%s)", count, node.String())
	}

	// 2. Non-empty byte streams must never exceed the initial nodeBudget
	streams := [][]byte{
		{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		{0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01},
		{0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
	}
	profiles := []ProfileType{
		ProfilePolynomial,
		ProfileRational,
		ProfileCalculus,
		ProfileTransforms,
		ProfileUnits,
	}
	for _, s := range streams {
		for _, prof := range profiles {
			p := NewByteStreamProvider(s)
			initialBudget := 12
			b := initialBudget
			n := p.GenNode(4, &b, prof)
			if actual := CountNodes(n); actual > initialBudget {
				t.Fatalf("Profile %v exceeded node budget %d: got %d nodes (%s)", prof, initialBudget, actual, n.String())
			}
		}
	}
}
