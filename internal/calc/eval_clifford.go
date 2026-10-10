package calc

import (
	"fmt"
	"math/big"
	"math/bits"
	"sort"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
)

// Metric defines an orthogonal diagonal metric signature (p, q, r).
// Basis vectors are canonically ordered:
//
//	Indices 0 .. p-1: positive bases (e_i^2 = +1)
//	Indices p .. p+q-1: negative bases (e_i^2 = -1)
//	Indices p+q .. p+q+r-1: degenerate / null bases (e_i^2 = 0)
type Metric struct {
	P        int
	Q        int
	R        int
	Dim      int
	PosMask  uint64
	NegMask  uint64
	NullMask uint64
}

// NewMetric creates a verified orthogonal diagonal metric.
func NewMetric(p, q, r int) (*Metric, error) {
	if p < 0 || q < 0 || r < 0 {
		return nil, fmt.Errorf("metric signature (p, q, r) components must be non-negative, got (%d, %d, %d)", p, q, r)
	}
	dim := p + q + r
	if dim > 64 {
		return nil, fmt.Errorf("total dimension p+q+r cannot exceed 64 for bitmask representation, got %d", dim)
	}

	var posMask, negMask, nullMask uint64
	if p > 0 {
		if p == 64 {
			posMask = ^uint64(0)
		} else {
			posMask = (uint64(1) << p) - 1
		}
	}
	if q > 0 {
		if q == 64 {
			negMask = ^uint64(0)
		} else {
			negMask = ((uint64(1) << q) - 1) << p
		}
	}
	if r > 0 {
		if r == 64 {
			nullMask = ^uint64(0)
		} else {
			nullMask = ((uint64(1) << r) - 1) << (p + q)
		}
	}

	return &Metric{
		P:        p,
		Q:        q,
		R:        r,
		Dim:      dim,
		PosMask:  posMask,
		NegMask:  negMask,
		NullMask: nullMask,
	}, nil
}

// Multivector represents a canonical sparse element in the Clifford algebra Cl(p, q, r).
type Multivector struct {
	Metric *Metric
	Terms  map[uint64]ast.Node
}

// NewMultivector constructs a Multivector, pruning any zero coefficients.
func NewMultivector(m *Metric, terms map[uint64]ast.Node) *Multivector {
	mv := &Multivector{
		Metric: m,
		Terms:  make(map[uint64]ast.Node),
	}
	for blade, coeff := range terms {
		if coeff != nil && !isASTZero(coeff) {
			mv.Terms[blade] = coeff
		}
	}
	return mv
}

// NewBasisBlade creates a basis vector multivector e_index.
func NewBasisBlade(m *Metric, index int) *Multivector {
	if index < 0 || index >= m.Dim {
		panic(fmt.Sprintf("basis index %d out of bounds for dimension %d", index, m.Dim))
	}
	terms := map[uint64]ast.Node{
		uint64(1) << index: ast.NewRationalFromBigRat(big.NewRat(1, 1)),
	}
	return NewMultivector(m, terms)
}

// NewScalarMultivector creates a scalar multivector.
func NewScalarMultivector(m *Metric, scalar ast.Node) *Multivector {
	terms := map[uint64]ast.Node{
		0: scalar,
	}
	return NewMultivector(m, terms)
}

func isASTZero(n ast.Node) bool {
	if n == nil {
		return true
	}
	if rat, ok := n.(*ast.RationalNode); ok {
		return rat.Val.Sign() == 0
	}
	return n.String() == "0"
}

// computeSwapSign counts basis swaps to bring blade u followed by blade v to ascending order.
func computeSwapSign(u, v uint64) int {
	swaps := 0
	for j := 0; j < 64; j++ {
		if (v & (uint64(1) << j)) != 0 {
			swaps += bits.OnesCount64(u >> (j + 1))
		}
	}
	if swaps%2 != 0 {
		return -1
	}
	return 1
}

// computeMetricFactor returns the scale factor resulting from squaring repeated basis vectors in u & v.
// If any repeated vector belongs to the null subspace (metric 0), returns (0, false).
func (m *Metric) computeMetricFactor(u, v uint64) (int, bool) {
	common := u & v
	if common == 0 {
		return 1, true
	}
	if (common & m.NullMask) != 0 {
		return 0, false
	}
	negCount := bits.OnesCount64(common & m.NegMask)
	if negCount%2 != 0 {
		return -1, true
	}
	return 1, true
}

func simplifyCoeff(n ast.Node) ast.Node {
	if n == nil {
		return ast.NewRationalFromBigRat(big.NewRat(0, 1))
	}
	res, err := Eval(n)
	if err != nil {
		return n
	}
	return res
}

func addCoeffs(a, b ast.Node) ast.Node {
	if isASTZero(a) {
		return b
	}
	if isASTZero(b) {
		return a
	}
	return simplifyCoeff(&ast.AddNode{Terms: []ast.Node{a, b}})
}

func mulCoeffs(a, b ast.Node) ast.Node {
	if isASTZero(a) || isASTZero(b) {
		return ast.NewRationalFromBigRat(big.NewRat(0, 1))
	}
	return simplifyCoeff(&ast.MulNode{Factors: []ast.Node{a, b}})
}

func scaleCoeff(c ast.Node, scale int) ast.Node {
	if scale == 0 || isASTZero(c) {
		return ast.NewRationalFromBigRat(big.NewRat(0, 1))
	}
	if scale == 1 {
		return c
	}
	if scale == -1 {
		return simplifyCoeff(&ast.UnaryOpNode{Op: "-", Expr: c})
	}
	return simplifyCoeff(&ast.MulNode{Factors: []ast.Node{ast.NewRationalFromBigRat(big.NewRat(int64(scale), 1)), c}})
}

// GeometricProduct computes A * B.
func (mv *Multivector) GeometricProduct(other *Multivector) (*Multivector, error) {
	if mv.Metric != other.Metric {
		return nil, fmt.Errorf("cannot multiply multivectors with mismatched metrics: %v vs %v", mv.Metric, other.Metric)
	}

	resTerms := make(map[uint64]ast.Node)

	for u, cU := range mv.Terms {
		for v, cV := range other.Terms {
			metricSign, ok := mv.Metric.computeMetricFactor(u, v)
			if !ok || metricSign == 0 {
				continue
			}

			swapSign := computeSwapSign(u, v)
			totalSign := swapSign * metricSign
			resBlade := u ^ v

			termCoeff := mulCoeffs(cU, cV)
			termCoeff = scaleCoeff(termCoeff, totalSign)

			if !isASTZero(termCoeff) {
				resTerms[resBlade] = addCoeffs(resTerms[resBlade], termCoeff)
			}
		}
	}

	return NewMultivector(mv.Metric, resTerms), nil
}

// Wedge computes the outer product A ^ B (terms with disjoint basis blades).
func (mv *Multivector) Wedge(other *Multivector) (*Multivector, error) {
	if mv.Metric != other.Metric {
		return nil, fmt.Errorf("cannot compute wedge product with mismatched metrics")
	}

	resTerms := make(map[uint64]ast.Node)

	for u, cU := range mv.Terms {
		for v, cV := range other.Terms {
			if (u & v) != 0 {
				continue // Shared basis vectors vanish in exterior product
			}

			swapSign := computeSwapSign(u, v)
			resBlade := u ^ v

			termCoeff := mulCoeffs(cU, cV)
			termCoeff = scaleCoeff(termCoeff, swapSign)

			if !isASTZero(termCoeff) {
				resTerms[resBlade] = addCoeffs(resTerms[resBlade], termCoeff)
			}
		}
	}

	return NewMultivector(mv.Metric, resTerms), nil
}

// LeftContract computes the left contraction A ⌟ B (terms where grade(res) == grade(B) - grade(A)).
func (mv *Multivector) LeftContract(other *Multivector) (*Multivector, error) {
	if mv.Metric != other.Metric {
		return nil, fmt.Errorf("cannot compute contraction with mismatched metrics")
	}

	resTerms := make(map[uint64]ast.Node)

	for u, cU := range mv.Terms {
		gradeU := bits.OnesCount64(u)
		for v, cV := range other.Terms {
			gradeV := bits.OnesCount64(v)
			if gradeU > gradeV {
				continue
			}

			// Left contraction requires u to be a sub-blade of v (all set bits of u are in v)
			if (u & v) != u {
				continue
			}

			metricSign, ok := mv.Metric.computeMetricFactor(u, v)
			if !ok || metricSign == 0 {
				continue
			}

			swapSign := computeSwapSign(u, v)
			totalSign := swapSign * metricSign
			resBlade := u ^ v

			if bits.OnesCount64(resBlade) != gradeV-gradeU {
				continue
			}

			termCoeff := mulCoeffs(cU, cV)
			termCoeff = scaleCoeff(termCoeff, totalSign)

			if !isASTZero(termCoeff) {
				resTerms[resBlade] = addCoeffs(resTerms[resBlade], termCoeff)
			}
		}
	}

	return NewMultivector(mv.Metric, resTerms), nil
}

// RightContract computes the right contraction A ⌞ B (terms where grade(res) == grade(A) - grade(B)).
func (mv *Multivector) RightContract(other *Multivector) (*Multivector, error) {
	if mv.Metric != other.Metric {
		return nil, fmt.Errorf("cannot compute contraction with mismatched metrics")
	}

	resTerms := make(map[uint64]ast.Node)

	for u, cU := range mv.Terms {
		gradeU := bits.OnesCount64(u)
		for v, cV := range other.Terms {
			gradeV := bits.OnesCount64(v)
			if gradeV > gradeU {
				continue
			}

			// Right contraction requires v to be a sub-blade of u
			if (u & v) != v {
				continue
			}

			metricSign, ok := mv.Metric.computeMetricFactor(u, v)
			if !ok || metricSign == 0 {
				continue
			}

			swapSign := computeSwapSign(u, v)
			totalSign := swapSign * metricSign
			resBlade := u ^ v

			if bits.OnesCount64(resBlade) != gradeU-gradeV {
				continue
			}

			termCoeff := mulCoeffs(cU, cV)
			termCoeff = scaleCoeff(termCoeff, totalSign)

			if !isASTZero(termCoeff) {
				resTerms[resBlade] = addCoeffs(resTerms[resBlade], termCoeff)
			}
		}
	}

	return NewMultivector(mv.Metric, resTerms), nil
}

// InnerProduct computes the dot product A . B (terms where grade(res) == |grade(A) - grade(B)|).
func (mv *Multivector) InnerProduct(other *Multivector) (*Multivector, error) {
	if mv.Metric != other.Metric {
		return nil, fmt.Errorf("cannot compute inner product with mismatched metrics")
	}

	resTerms := make(map[uint64]ast.Node)

	for u, cU := range mv.Terms {
		gradeU := bits.OnesCount64(u)
		for v, cV := range other.Terms {
			gradeV := bits.OnesCount64(v)
			expectedGrade := gradeU - gradeV
			if expectedGrade < 0 {
				expectedGrade = -expectedGrade
			}

			metricSign, ok := mv.Metric.computeMetricFactor(u, v)
			if !ok || metricSign == 0 {
				continue
			}

			swapSign := computeSwapSign(u, v)
			totalSign := swapSign * metricSign
			resBlade := u ^ v

			if bits.OnesCount64(resBlade) != expectedGrade {
				continue
			}

			termCoeff := mulCoeffs(cU, cV)
			termCoeff = scaleCoeff(termCoeff, totalSign)

			if !isASTZero(termCoeff) {
				resTerms[resBlade] = addCoeffs(resTerms[resBlade], termCoeff)
			}
		}
	}

	return NewMultivector(mv.Metric, resTerms), nil
}

// Reverse computes the reversion ~A (reverses order of basis vectors in each blade: (-1)^(k*(k-1)/2)).
func (mv *Multivector) Reverse() *Multivector {
	resTerms := make(map[uint64]ast.Node)
	for blade, coeff := range mv.Terms {
		k := bits.OnesCount64(blade)
		sign := 1
		if (k*(k-1)/2)%2 != 0 {
			sign = -1
		}
		resTerms[blade] = scaleCoeff(coeff, sign)
	}
	return NewMultivector(mv.Metric, resTerms)
}

// GradeInvolution computes the main involution (multiplies grade k by (-1)^k).
func (mv *Multivector) GradeInvolution() *Multivector {
	resTerms := make(map[uint64]ast.Node)
	for blade, coeff := range mv.Terms {
		k := bits.OnesCount64(blade)
		sign := 1
		if k%2 != 0 {
			sign = -1
		}
		resTerms[blade] = scaleCoeff(coeff, sign)
	}
	return NewMultivector(mv.Metric, resTerms)
}

// GradeProject extracts only the components of grade k.
func (mv *Multivector) GradeProject(k int) *Multivector {
	resTerms := make(map[uint64]ast.Node)
	for blade, coeff := range mv.Terms {
		if bits.OnesCount64(blade) == k {
			resTerms[blade] = coeff
		}
	}
	return NewMultivector(mv.Metric, resTerms)
}

// PoincareDual computes the basis-complement dual (Poincaré dual), safe even in degenerate algebras (r > 0).
func (mv *Multivector) PoincareDual() *Multivector {
	dim := mv.Metric.Dim
	var fullMask uint64
	if dim == 64 {
		fullMask = ^uint64(0)
	} else {
		fullMask = (uint64(1) << dim) - 1
	}

	resTerms := make(map[uint64]ast.Node)
	for blade, coeff := range mv.Terms {
		dualBlade := fullMask ^ blade
		// Sign needed to reorder blade and dualBlade to canonical ascending fullMask:
		sign := computeSwapSign(blade, dualBlade)
		resTerms[dualBlade] = scaleCoeff(coeff, sign)
	}
	return NewMultivector(mv.Metric, resTerms)
}

// TryInverse computes the algebraic inverse of a multivector if defined.
func (mv *Multivector) TryInverse() (*Multivector, error) {
	if mv.IsZero() {
		return nil, fmt.Errorf("division by zero: cannot invert zero multivector")
	}

	// 1. Single basis blade: A = c * e_u
	if len(mv.Terms) == 1 {
		for u, coeff := range mv.Terms {
			metricSign, ok := mv.Metric.computeMetricFactor(u, u)
			if !ok || metricSign == 0 {
				return nil, fmt.Errorf("cannot invert null blade with square zero: %s", mv)
			}
			swapSign := computeSwapSign(u, u)
			totalSq := swapSign * metricSign // e_u^2

			// Inverse of c * e_u is (1/c) * (e_u / e_u^2)
			// e_u / totalSq = scaleCoeff(e_u, totalSq)
			invCoeff := simplifyCoeff(&ast.MulNode{Factors: []ast.Node{
				ast.NewRationalFromBigRat(big.NewRat(int64(totalSq), 1)),
				&ast.PowNode{Base: coeff, Exp: ast.NewRationalFromBigRat(big.NewRat(-1, 1))},
			}})
			return NewMultivector(mv.Metric, map[uint64]ast.Node{u: invCoeff}), nil
		}
	}

	// 2. Versors: check if M * ~M is a non-zero scalar
	rev := mv.Reverse()
	normSq, err := mv.GeometricProduct(rev)
	if err == nil {
		if len(normSq.Terms) == 1 && normSq.Terms[0] != nil && !isASTZero(normSq.Terms[0]) {
			scalar := normSq.Terms[0]
			invScalar := simplifyCoeff(&ast.PowNode{Base: scalar, Exp: ast.NewRationalFromBigRat(big.NewRat(-1, 1))})
			resTerms := make(map[uint64]ast.Node)
			for b, c := range rev.Terms {
				resTerms[b] = mulCoeffs(c, invScalar)
			}
			return NewMultivector(mv.Metric, resTerms), nil
		}
	}

	return nil, fmt.Errorf("multivector is singular or general inverse not supported for: %s", mv)
}

// Add computes A + B.
func (mv *Multivector) Add(other *Multivector) (*Multivector, error) {
	if mv.Metric != other.Metric {
		return nil, fmt.Errorf("cannot add multivectors with mismatched metrics")
	}
	resTerms := make(map[uint64]ast.Node)
	for b, c := range mv.Terms {
		resTerms[b] = c
	}
	for b, c := range other.Terms {
		resTerms[b] = addCoeffs(resTerms[b], c)
	}
	return NewMultivector(mv.Metric, resTerms), nil
}

// Sub computes A - B.
func (mv *Multivector) Sub(other *Multivector) (*Multivector, error) {
	negOther := other.Scale(ast.NewRationalFromBigRat(big.NewRat(-1, 1)))
	return mv.Add(negOther)
}

// Scale scales the multivector by a scalar AST node.
func (mv *Multivector) Scale(scalar ast.Node) *Multivector {
	if isASTZero(scalar) {
		return NewMultivector(mv.Metric, nil)
	}
	resTerms := make(map[uint64]ast.Node)
	for b, c := range mv.Terms {
		resTerms[b] = mulCoeffs(c, scalar)
	}
	return NewMultivector(mv.Metric, resTerms)
}

// Equals checks canonical equivalence.
func (mv *Multivector) Equals(other *Multivector) bool {
	if mv.Metric != other.Metric {
		return false
	}
	diff, err := mv.Sub(other)
	if err != nil {
		return false
	}
	return diff.IsZero()
}

// IsZero checks if the multivector is 0.
func (mv *Multivector) IsZero() bool {
	return len(mv.Terms) == 0
}

// String formats the multivector as a readable string.
func (mv *Multivector) String() string {
	if mv.IsZero() {
		return "0"
	}

	// Sort blades for deterministic output
	var blades []uint64
	for b := range mv.Terms {
		blades = append(blades, b)
	}
	sort.Slice(blades, func(i, j int) bool {
		gi := bits.OnesCount64(blades[i])
		gj := bits.OnesCount64(blades[j])
		if gi != gj {
			return gi < gj
		}
		return blades[i] < blades[j]
	})

	var parts []string
	for _, b := range blades {
		coeff := mv.Terms[b]
		coeffStr := coeff.String()

		if b == 0 {
			parts = append(parts, coeffStr)
			continue
		}

		// Basis blade name
		var basisNames []string
		for i := 0; i < mv.Metric.Dim; i++ {
			if (b & (uint64(1) << i)) != 0 {
				basisNames = append(basisNames, fmt.Sprintf("e%d", i))
			}
		}
		bladeName := strings.Join(basisNames, "^")

		switch coeffStr {
		case "1":
			parts = append(parts, bladeName)
		case "-1":
			parts = append(parts, "-"+bladeName)
		default:
			parts = append(parts, fmt.Sprintf("%s*%s", coeffStr, bladeName))
		}
	}

	return strings.Join(parts, " + ")
}
