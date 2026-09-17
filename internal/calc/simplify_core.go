package calc

import (
	"fmt"
	"math/big"
)

// -------------------------------------------------------------------------
// Unary Operators (- and !)
// -------------------------------------------------------------------------

func simplifyUnaryOp(op string, expr Node) (Node, error) {
	switch op {
	case "-":
		switch v := expr.(type) {
		case *RationalNode:
			neg := new(big.Rat).Neg(v.Val)
			return &RationalNode{Val: neg}, nil
		case *ComplexNode:
			negReal, err := simplifyUnaryOp("-", v.Real)
			if err != nil {
				return nil, err
			}
			negImag, err := simplifyUnaryOp("-", v.Imag)
			if err != nil {
				return nil, err
			}
			return NewComplex(negReal, negImag), nil
		case *AddNode:
			negTerms := make([]Node, len(v.Terms))
			for i, t := range v.Terms {
				nt, err := simplifyUnaryOp("-", t)
				if err != nil {
					return nil, err
				}
				negTerms[i] = nt
			}
			return simplifyAdd(negTerms)
		case *MatrixNode:
			negData := make([][]Node, v.Rows)
			for r := 0; r < v.Rows; r++ {
				negData[r] = make([]Node, v.Cols)
				for c := 0; c < v.Cols; c++ {
					negElem, err := simplifyUnaryOp("-", v.Data[r][c])
					if err != nil {
						return nil, err
					}
					negData[r][c] = negElem
				}
			}
			return NewMatrix(v.Rows, v.Cols, negData)
		default:
			negOne, _ := NewRational(-1, 1)
			return simplifyMul([]Node{negOne, expr})
		}

	case "!":
		rat, ok := expr.(*RationalNode)
		if !ok {
			return &UnaryOpNode{Op: op, Expr: expr}, nil
		}
		if !rat.Val.IsInt() || rat.Val.Sign() < 0 {
			return nil, fmt.Errorf("factorial domain error: factorial requires non-negative integer, got %s", expr.String())
		}
		if !rat.Val.Num().IsInt64() || rat.Val.Num().Int64() > 100000 {
			return nil, fmt.Errorf("factorial domain error: factorial is too large to compute, got %s", expr.String())
		}
		n := rat.Val.Num().Int64()
		if n <= 1 {
			res := big.NewInt(1)
			return &RationalNode{Val: new(big.Rat).SetInt(res)}, nil
		}
		var res *big.Int
		if n >= 128 {
			res = ParallelProductTree(1, n, 64)
		} else {
			res = big.NewInt(1)
			for i := int64(2); i <= n; i++ {
				res.Mul(res, big.NewInt(i))
			}
		}
		return &RationalNode{Val: new(big.Rat).SetInt(res)}, nil

	default:
		return nil, fmt.Errorf("unknown unary operator: %s", op)
	}
}

func isZero(n Node) bool {
	if r, ok := n.(*RationalNode); ok && r.Val.Sign() == 0 {
		return true
	}
	return false
}

func isZeroNode(n Node) bool {
	if n == nil {
		return false
	}
	switch v := n.(type) {
	case *RationalNode:
		return v.Val.Sign() == 0
	case *MulNode:
		for _, f := range v.Factors {
			if isZeroNode(f) {
				return true
			}
		}
	}
	return false
}

// -------------------------------------------------------------------------
// CAS: Polynomial Expansion (expand)
// -------------------------------------------------------------------------

// expandNode recursively expands an AST node using distributive laws and binomial expansion.
func expandNode(n Node) Node {
	if n == nil {
		return nil
	}
	switch v := n.(type) {
	case *AddNode:
		newTerms := make([]Node, len(v.Terms))
		for i, t := range v.Terms {
			newTerms[i] = expandNode(t)
		}
		res, _ := simplifyAdd(newTerms)
		return res

	case *MulNode:
		if len(v.Factors) == 0 {
			return mustRational(1, 1)
		}
		current := expandNode(v.Factors[0])
		for i := 1; i < len(v.Factors); i++ {
			next := expandNode(v.Factors[i])
			current = expandMul2(current, next)
		}
		return current

	case *PowNode:
		base := expandNode(v.Base)
		if r, ok := v.Exp.(*RationalNode); ok && r.Val.IsInt() && r.Val.Sign() >= 0 {
			expInt := r.Val.Num().Int64()
			if expInt == 0 {
				return mustRational(1, 1)
			}
			if expInt == 1 {
				return base
			}
			if expInt <= 10 {
				res := base
				for i := int64(1); i < expInt; i++ {
					res = expandMul2(res, base)
				}
				return res
			}
		}
		res, err := simplifyPow(base, v.Exp)
		if err != nil {
			return &PowNode{Base: base, Exp: v.Exp}
		}
		return res

	case *UnaryOpNode:
		if v.Op == "-" {
			expanded := expandNode(v.Expr)
			res, _ := simplifyUnaryOp("-", expanded)
			return res
		}
		return v

	case *FuncNode:
		newArgs := make([]Node, len(v.Args))
		for i, a := range v.Args {
			newArgs[i] = expandNode(a)
		}
		res, err := simplifyFunc(v.Name, newArgs)
		if err != nil {
			return &FuncNode{Name: v.Name, Args: newArgs}
		}
		return res

	default:
		return n
	}
}

// expandMul2 multiplies two already expanded nodes using the distributive law.
func expandMul2(a, b Node) Node {
	addA, isAddA := a.(*AddNode)
	addB, isAddB := b.(*AddNode)

	if isAddA && isAddB {
		var terms []Node
		for _, ta := range addA.Terms {
			for _, tb := range addB.Terms {
				prod, _ := simplifyMul([]Node{ta, tb})
				terms = append(terms, prod)
			}
		}
		res, _ := simplifyAdd(terms)
		return res
	} else if isAddA {
		var terms []Node
		for _, ta := range addA.Terms {
			prod, _ := simplifyMul([]Node{ta, b})
			terms = append(terms, prod)
		}
		res, _ := simplifyAdd(terms)
		return res
	} else if isAddB {
		var terms []Node
		for _, tb := range addB.Terms {
			prod, _ := simplifyMul([]Node{a, tb})
			terms = append(terms, prod)
		}
		res, _ := simplifyAdd(terms)
		return res
	} else {
		prod, _ := simplifyMul([]Node{a, b})
		return prod
	}
}

// -------------------------------------------------------------------------
// CAS: Symbolic Differentiation (diff)
// -------------------------------------------------------------------------

// containsVar checks if an AST node contains the given variable name.
func containsVar(n Node, varName string) bool {
	return ContainsVar(n, varName)
}
