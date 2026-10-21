package calc_test

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
)

// ProfileType represents the target domain profile for AST generation.
type ProfileType int

const (
	ProfileGeneral ProfileType = iota
	ProfileAlgebra
	ProfileCalculus
	ProfileSpecial
	ProfilePolynomial
	ProfileRational
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
// If maxDepth <= 0 or *nodeBudget <= 0, a terminal leaf node is immediately returned.
func (p *ByteStreamProvider) GenNode(maxDepth int, nodeBudget *int, profile ProfileType) ast.Node {
	if nodeBudget == nil {
		defaultBudget := 16
		nodeBudget = &defaultBudget
	}

	if maxDepth <= 0 || *nodeBudget <= 0 {
		return p.genLeafNode(profile)
	}

	*nodeBudget--

	if profile == ProfilePolynomial {
		choice := p.NextChoice(5)
		switch choice {
		case 0: // Add
			t1 := p.GenNode(maxDepth-1, nodeBudget, profile)
			t2 := p.GenNode(maxDepth-1, nodeBudget, profile)
			return ast.NewAdd([]ast.Node{t1, t2})
		case 1: // Mul
			f1 := p.GenNode(maxDepth-1, nodeBudget, profile)
			f2 := p.GenNode(maxDepth-1, nodeBudget, profile)
			return ast.NewMul([]ast.Node{f1, f2})
		case 2: // Pow with integer exp [0, 4]
			base := p.GenNode(maxDepth-1, nodeBudget, profile)
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
			t1 := p.GenNode(maxDepth-1, nodeBudget, profile)
			t2 := p.GenNode(maxDepth-1, nodeBudget, profile)
			return ast.NewAdd([]ast.Node{t1, t2})
		case 1: // Mul
			f1 := p.GenNode(maxDepth-1, nodeBudget, profile)
			f2 := p.GenNode(maxDepth-1, nodeBudget, profile)
			return ast.NewMul([]ast.Node{f1, f2})
		case 2: // Pow with integer exp [-2, 3]
			base := p.GenNode(maxDepth-1, nodeBudget, profile)
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
			num := p.GenNode(maxDepth-1, nodeBudget, profile)
			den := p.GenNode(maxDepth-1, nodeBudget, profile)
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
		re := p.GenNode(maxDepth-1, nodeBudget, profile)
		im := p.GenNode(maxDepth-1, nodeBudget, profile)
		return ast.NewComplex(re, im)

	case 5: // Add
		t1 := p.GenNode(maxDepth-1, nodeBudget, profile)
		t2 := p.GenNode(maxDepth-1, nodeBudget, profile)
		return ast.NewAdd([]ast.Node{t1, t2})

	case 6: // Mul
		f1 := p.GenNode(maxDepth-1, nodeBudget, profile)
		f2 := p.GenNode(maxDepth-1, nodeBudget, profile)
		return ast.NewMul([]ast.Node{f1, f2})

	case 7: // Pow
		base := p.GenNode(maxDepth-1, nodeBudget, profile)
		exp := p.GenNode(maxDepth-1, nodeBudget, profile)
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
		elems := make([]ast.Node, elemCount)
		for i := 0; i < elemCount; i++ {
			elems[i] = p.GenNode(maxDepth-1, nodeBudget, profile)
		}
		return ast.NewList(elems)

	case 11: // Matrix
		rows := p.NextChoice(2) + 1
		cols := p.NextChoice(2) + 1
		data := make([][]ast.Node, rows)
		for r := 0; r < rows; r++ {
			data[r] = make([]ast.Node, cols)
			for c := 0; c < cols; c++ {
				data[r][c] = p.GenNode(maxDepth-1, nodeBudget, profile)
			}
		}
		mat, err := ast.NewMatrix(rows, cols, data)
		if err != nil {
			return p.genLeafNode(profile)
		}
		return mat

	case 12: // Plot
		plots := []string{"sin(x)", "cos(x)", "x^2"}
		return ast.NewPlotNode(plots[p.NextChoice(len(plots))])

	case 13: // RelOp
		ops := []string{"==", "!=", "<", "<=", ">", ">="}
		op := ops[p.NextChoice(len(ops))]
		lhs := p.GenNode(maxDepth-1, nodeBudget, profile)
		rhs := p.GenNode(maxDepth-1, nodeBudget, profile)
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
		expr := p.GenNode(maxDepth-1, nodeBudget, profile)
		cond := ast.NewRelOp(ast.NewVar("x"), ">", ast.NewRationalFromBigRat(big.NewRat(0, 1)))
		otherwise := p.GenNode(maxDepth-1, nodeBudget, profile)
		return ast.NewPiecewiseNode([]ast.PiecewiseCase{{Expr: expr, Condition: cond}}, otherwise)

	case 18: // Interval
		low := big.NewRat(p.NextInt(-10, 0), 1)
		high := big.NewRat(p.NextInt(1, 10), 1)
		return ast.NewIntervalNode(low, high)

	case 19: // String
		strs := []string{"alpha", "beta", "theta", "pi", "x"}
		return ast.NewStringNode(strs[p.NextChoice(len(strs))])

	case 20: // Index
		target := ast.NewList([]ast.Node{p.genLeafNode(profile), p.genLeafNode(profile)})
		idxNode := ast.NewRationalFromBigRat(big.NewRat(0, 1))
		return ast.NewIndexNode(target, idxNode)

	case 21: // Slice
		target := ast.NewList([]ast.Node{p.genLeafNode(profile), p.genLeafNode(profile), p.genLeafNode(profile)})
		start := ast.NewRationalFromBigRat(big.NewRat(0, 1))
		stop := ast.NewRationalFromBigRat(big.NewRat(2, 1))
		step := ast.NewRationalFromBigRat(big.NewRat(1, 1))
		return ast.NewSliceNode(target, start, stop, step)

	case 22: // Lambda
		body := p.GenNode(maxDepth-1, nodeBudget, profile)
		return ast.NewLambdaNode([]string{"x"}, body)

	case 23: // Apply
		callee := ast.NewLambdaNode([]string{"x"}, ast.NewVar("x"))
		arg := p.GenNode(maxDepth-1, nodeBudget, profile)
		return ast.NewApplyNode(callee, []ast.Node{arg})

	case 24: // Quantity
		val := p.genLeafNode(profile)
		dims := ast.NewDimensionVector()
		dims[0] = big.NewRat(1, 1) // e.g. Length
		scale := big.NewRat(1, 1)
		return ast.NewQuantityNode(val, scale, dims, "m")

	default:
		return p.genLeafNode(profile)
	}
}

// genLeafNode creates a deterministic terminal leaf node to guarantee Totality.
func (p *ByteStreamProvider) genLeafNode(profile ProfileType) ast.Node {
	switch profile {
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

			// Node budget and maxDepth variations
			for maxDepth := 0; maxDepth <= 4; maxDepth++ {
				budget := 16
				node := p.GenNode(maxDepth, &budget, ProfileGeneral)
				if node == nil {
					t.Fatalf("[%s] GenNode returned nil at maxDepth=%d", tc.name, maxDepth)
				}
				if node.String() == "" {
					t.Fatalf("[%s] GenNode returned node with empty string representation: %v", tc.name, node)
				}
			}
		})
	}
}

// Suppress unused import errors if any
var _ = fmt.Sprintf
var _ = calc.NewEnv
