package main

import (
	"fmt"
	"math/big"
	"sort"
	"strings"
)

// FormatOptions controls the string representation of expressions.
type FormatOptions struct {
	AsciiOnly bool
}

// Format formats a node using default options (Unicode symbols).
func Format(n Node) string {
	return FormatWithOptions(n, FormatOptions{AsciiOnly: false})
}

// FormatWithOptions formats a node using the provided options.
func FormatWithOptions(n Node, opts FormatOptions) string {
	if n == nil {
		return ""
	}
	return formatNode(n, opts)
}

func formatNode(n Node, opts FormatOptions) string {
	switch v := n.(type) {
	case *RationalNode:
		if v.Val.IsInt() {
			return v.Val.Num().String()
		}
		return v.Val.String()

	case *ConstNode:
		if v.Name == "pi" {
			if opts.AsciiOnly {
				return "pi"
			}
			return "π"
		}
		return v.Name

	case *SqrtNode:
		radStr := formatNode(v.Radicand, opts)
		if opts.AsciiOnly {
			return fmt.Sprintf("sqrt(%s)", radStr)
		}
		// If radicand is pure integer, omit parentheses: √2
		if rat, ok := v.Radicand.(*RationalNode); ok && rat.Val.IsInt() {
			return fmt.Sprintf("√%s", radStr)
		}
		return fmt.Sprintf("√(%s)", radStr)

	case *FuncNode:
		argStrs := make([]string, len(v.Args))
		for i, a := range v.Args {
			argStrs[i] = formatNode(a, opts)
		}
		return fmt.Sprintf("%s(%s)", v.Name, strings.Join(argStrs, ", "))

	case *ComplexNode:
		realIsZero := isZero(v.Real)
		imagIsZero := isZero(v.Imag)

		if realIsZero && imagIsZero {
			return "0"
		}
		if imagIsZero {
			return formatNode(v.Real, opts)
		}
		if realIsZero {
			return formatImagPart(v.Imag, opts, false)
		}

		// Both non-zero: Real +/- Imag*i
		realStr := formatNode(v.Real, opts)
		if isNegative(v.Imag) {
			posImag, _ := simplifyUnaryOp("-", v.Imag)
			imagStr := formatImagPart(posImag, opts, false)
			return fmt.Sprintf("%s - %s", realStr, imagStr)
		}
		imagStr := formatImagPart(v.Imag, opts, false)
		return fmt.Sprintf("%s + %s", realStr, imagStr)

	case *PowNode:
		baseStr := formatNode(v.Base, opts)
		expStr := formatNode(v.Exp, opts)
		if _, isOp := v.Base.(*AddNode); isOp {
			baseStr = fmt.Sprintf("(%s)", baseStr)
		}
		return fmt.Sprintf("%s^%s", baseStr, expStr)

	case *UnaryOpNode:
		if v.Op == "!" {
			exprStr := formatNode(v.Expr, opts)
			if _, isOp := v.Expr.(*AddNode); isOp {
				exprStr = fmt.Sprintf("(%s)", exprStr)
			}
			return fmt.Sprintf("%s!", exprStr)
		}
		exprStr := formatNode(v.Expr, opts)
		if _, isOp := v.Expr.(*AddNode); isOp {
			exprStr = fmt.Sprintf("-(%s)", exprStr)
		} else {
			exprStr = fmt.Sprintf("-%s", exprStr)
		}
		return exprStr

	case *MulNode:
		return formatMul(v, opts)

	case *AddNode:
		return formatAdd(v, opts)

	default:
		return n.String()
	}
}

func formatImagPart(imag Node, opts FormatOptions, isMinus bool) string {
	one := big.NewRat(1, 1)
	negOne := big.NewRat(-1, 1)

	if rat, ok := imag.(*RationalNode); ok {
		if rat.Val.Cmp(one) == 0 {
			return "i"
		}
		if rat.Val.Cmp(negOne) == 0 {
			return "-i"
		}
		return fmt.Sprintf("%s*i", rat.String())
	}
	return fmt.Sprintf("%s*i", formatNode(imag, opts))
}

func isNegative(n Node) bool {
	if rat, ok := n.(*RationalNode); ok {
		return rat.Val.Sign() < 0
	}
	if uop, ok := n.(*UnaryOpNode); ok && uop.Op == "-" {
		return true
	}
	if mul, ok := n.(*MulNode); ok && len(mul.Factors) > 0 {
		if rat, ok := mul.Factors[0].(*RationalNode); ok {
			return rat.Val.Sign() < 0
		}
	}
	if c, ok := n.(*ComplexNode); ok {
		if isZero(c.Real) && isNegative(c.Imag) {
			return true
		}
	}
	return false
}

func formatMul(m *MulNode, opts FormatOptions) string {
	if len(m.Factors) == 2 {
		if r, ok := m.Factors[0].(*RationalNode); ok {
			// Check for 1/d * Base -> Base/d
			if !r.Val.IsInt() && r.Val.Num().Cmp(big.NewInt(1)) == 0 {
				baseStr := formatNode(m.Factors[1], opts)
				return fmt.Sprintf("%s/%s", baseStr, r.Val.Denom().String())
			}
			// Check for -1/d * Base -> -Base/d
			if !r.Val.IsInt() && r.Val.Num().Cmp(big.NewInt(-1)) == 0 {
				baseStr := formatNode(m.Factors[1], opts)
				return fmt.Sprintf("-%s/%s", baseStr, r.Val.Denom().String())
			}
			// Check for 1 * Base -> Base
			one := big.NewRat(1, 1)
			if r.Val.Cmp(one) == 0 {
				return formatNode(m.Factors[1], opts)
			}
			// Check for -1 * Base -> -Base
			negOne := big.NewRat(-1, 1)
			if r.Val.Cmp(negOne) == 0 {
				return fmt.Sprintf("-%s", formatNode(m.Factors[1], opts))
			}
		}
	}

	strs := make([]string, len(m.Factors))
	for i, f := range m.Factors {
		fStr := formatNode(f, opts)
		if _, isAdd := f.(*AddNode); isAdd {
			fStr = fmt.Sprintf("(%s)", fStr)
		}
		strs[i] = fStr
	}
	return strings.Join(strs, "*")
}

func formatAdd(a *AddNode, opts FormatOptions) string {
	sortedTerms := sortTerms(a.Terms)

	var sb strings.Builder
	for i, t := range sortedTerms {
		if i == 0 {
			sb.WriteString(formatNode(t, opts))
		} else {
			if isNegative(t) {
				// term is negative -> " - positiveTerm"
				posTerm, _ := simplifyUnaryOp("-", t)
				sb.WriteString(" - ")
				sb.WriteString(formatNode(posTerm, opts))
			} else {
				sb.WriteString(" + ")
				sb.WriteString(formatNode(t, opts))
			}
		}
	}
	return sb.String()
}

// termCategory maps a node to an order index:
// 1: Rational
// 2: Sqrt
// 3: Const (pi, e)
// 4: Func (sin, cos, log)
// 5: Complex
// 6: Other
func termCategory(n Node) int {
	switch v := n.(type) {
	case *RationalNode:
		return 1
	case *SqrtNode:
		return 2
	case *ConstNode:
		return 3
	case *FuncNode:
		return 4
	case *ComplexNode:
		return 5
	case *MulNode:
		if len(v.Factors) >= 2 {
			return termCategory(v.Factors[len(v.Factors)-1])
		}
		return 6
	default:
		return 6
	}
}

func sortTerms(terms []Node) []Node {
	copied := make([]Node, len(terms))
	copy(copied, terms)

	sort.SliceStable(copied, func(i, j int) bool {
		catI := termCategory(copied[i])
		catJ := termCategory(copied[j])
		if catI != catJ {
			return catI < catJ
		}
		// Tie-break: sort by String representation
		return copied[i].String() < copied[j].String()
	})

	return copied
}
