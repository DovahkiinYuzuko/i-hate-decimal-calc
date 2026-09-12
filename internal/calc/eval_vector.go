package calc

import (
	"fmt"
)

// -------------------------------------------------------------------------
// 3D Vector Calculus Evaluation Functions
// -------------------------------------------------------------------------

func toVectorElements(n Node) ([]Node, error) {
	if n == nil {
		return nil, fmt.Errorf("vector cannot be nil")
	}
	switch v := n.(type) {
	case *ListNode:
		return v.Elements, nil
	case *MatrixNode:
		if v.Rows == 1 {
			return v.Data[0], nil
		}
		if v.Cols == 1 {
			elems := make([]Node, v.Rows)
			for r := 0; r < v.Rows; r++ {
				elems[r] = v.Data[r][0]
			}
			return elems, nil
		}
		return nil, fmt.Errorf("argument must be a 1D vector or 1xN/Nx1 matrix, got %dx%d matrix", v.Rows, v.Cols)
	default:
		return nil, fmt.Errorf("argument must be a vector (list or 1D matrix), got %s", n.String())
	}
}

func evalDot(uNode, vNode Node) (Node, error) {
	u, err := toVectorElements(uNode)
	if err != nil {
		return nil, fmt.Errorf("dot error: %w", err)
	}
	v, err := toVectorElements(vNode)
	if err != nil {
		return nil, fmt.Errorf("dot error: %w", err)
	}
	if len(u) != len(v) {
		return nil, fmt.Errorf("dot error: dimension mismatch: %d and %d", len(u), len(v))
	}
	if len(u) == 0 {
		return mustRational(0, 1), nil
	}

	var prods []Node
	for i := range u {
		p, err := simplifyMul([]Node{u[i], v[i]})
		if err != nil {
			return nil, err
		}
		prods = append(prods, p)
	}
	res, err := simplifyAdd(prods)
	if err != nil {
		return nil, err
	}
	return expandNode(res), nil
}

func evalCross(uNode, vNode Node) (*ListNode, error) {
	u, err := toVectorElements(uNode)
	if err != nil {
		return nil, fmt.Errorf("cross error: %w", err)
	}
	v, err := toVectorElements(vNode)
	if err != nil {
		return nil, fmt.Errorf("cross error: %w", err)
	}
	if len(u) != 3 || len(v) != 3 {
		return nil, fmt.Errorf("cross error: cross product requires 3-dimensional vectors, got %d and %d", len(u), len(v))
	}

	w1p1, err := simplifyMul([]Node{u[1], v[2]})
	if err != nil {
		return nil, err
	}
	w1p2, err := simplifyMul([]Node{mustRational(-1, 1), u[2], v[1]})
	if err != nil {
		return nil, err
	}
	w1, err := simplifyAdd([]Node{w1p1, w1p2})
	if err != nil {
		return nil, err
	}

	w2p1, err := simplifyMul([]Node{u[2], v[0]})
	if err != nil {
		return nil, err
	}
	w2p2, err := simplifyMul([]Node{mustRational(-1, 1), u[0], v[2]})
	if err != nil {
		return nil, err
	}
	w2, err := simplifyAdd([]Node{w2p1, w2p2})
	if err != nil {
		return nil, err
	}

	w3p1, err := simplifyMul([]Node{u[0], v[1]})
	if err != nil {
		return nil, err
	}
	w3p2, err := simplifyMul([]Node{mustRational(-1, 1), u[1], v[0]})
	if err != nil {
		return nil, err
	}
	w3, err := simplifyAdd([]Node{w3p1, w3p2})
	if err != nil {
		return nil, err
	}

	return &ListNode{Elements: []Node{expandNode(w1), expandNode(w2), expandNode(w3)}}, nil
}

func evalNorm(vNode Node) (Node, error) {
	v, err := toVectorElements(vNode)
	if err != nil {
		return nil, fmt.Errorf("norm error: %w", err)
	}
	if len(v) == 0 {
		return mustRational(0, 1), nil
	}

	var squares []Node
	for _, elem := range v {
		sq, err := simplifyMul([]Node{elem, elem})
		if err != nil {
			return nil, err
		}
		squares = append(squares, sq)
	}
	sumSq, err := simplifyAdd(squares)
	if err != nil {
		return nil, err
	}
	return simplifySqrt(sumSq)
}

func evalGrad(fNode Node, varsNode Node) (*ListNode, error) {
	vars, err := toVectorElements(varsNode)
	if err != nil {
		return nil, fmt.Errorf("grad error: %w", err)
	}
	if len(vars) == 0 {
		return nil, fmt.Errorf("grad error: coordinate variables vector cannot be empty")
	}

	results := make([]Node, len(vars))
	for i, vn := range vars {
		v, ok := vn.(*VarNode)
		if !ok {
			return nil, fmt.Errorf("grad error: element %d of coordinates must be a variable, got %s", i+1, vn.String())
		}
		d, err := differentiate(fNode, v.Name)
		if err != nil {
			return nil, fmt.Errorf("grad error in d/d%s: %w", v.Name, err)
		}
		results[i] = expandNode(d)
	}
	return &ListNode{Elements: results}, nil
}

func evalDiv(FNode Node, varsNode Node) (Node, error) {
	F, err := toVectorElements(FNode)
	if err != nil {
		return nil, fmt.Errorf("div error: %w", err)
	}
	vars, err := toVectorElements(varsNode)
	if err != nil {
		return nil, fmt.Errorf("div error: %w", err)
	}
	if len(F) != len(vars) {
		return nil, fmt.Errorf("div error: vector field and coordinates dimension mismatch: %d and %d", len(F), len(vars))
	}

	var terms []Node
	for i := range F {
		v, ok := vars[i].(*VarNode)
		if !ok {
			return nil, fmt.Errorf("div error: coordinate %d must be a variable, got %s", i+1, vars[i].String())
		}
		d, err := differentiate(F[i], v.Name)
		if err != nil {
			return nil, fmt.Errorf("div error in d/d%s: %w", v.Name, err)
		}
		terms = append(terms, d)
	}
	res, err := simplifyAdd(terms)
	if err != nil {
		return nil, err
	}
	return expandNode(res), nil
}

func evalCurl(FNode Node, varsNode Node) (*ListNode, error) {
	F, err := toVectorElements(FNode)
	if err != nil {
		return nil, fmt.Errorf("curl error: %w", err)
	}
	vars, err := toVectorElements(varsNode)
	if err != nil {
		return nil, fmt.Errorf("curl error: %w", err)
	}
	if len(F) != 3 || len(vars) != 3 {
		return nil, fmt.Errorf("curl error: curl requires 3-dimensional vectors, got %d and %d", len(F), len(vars))
	}

	vx, okX := vars[0].(*VarNode)
	vy, okY := vars[1].(*VarNode)
	vz, okZ := vars[2].(*VarNode)
	if !okX || !okY || !okZ {
		return nil, fmt.Errorf("curl error: all coordinate elements must be variables")
	}

	dF3dy, err := differentiate(F[2], vy.Name)
	if err != nil {
		return nil, err
	}
	dF2dz, err := differentiate(F[1], vz.Name)
	if err != nil {
		return nil, err
	}
	negDF2dz, err := simplifyMul([]Node{mustRational(-1, 1), dF2dz})
	if err != nil {
		return nil, err
	}
	c1, err := simplifyAdd([]Node{dF3dy, negDF2dz})
	if err != nil {
		return nil, err
	}

	dF1dz, err := differentiate(F[0], vz.Name)
	if err != nil {
		return nil, err
	}
	dF3dx, err := differentiate(F[2], vx.Name)
	if err != nil {
		return nil, err
	}
	negDF3dx, err := simplifyMul([]Node{mustRational(-1, 1), dF3dx})
	if err != nil {
		return nil, err
	}
	c2, err := simplifyAdd([]Node{dF1dz, negDF3dx})
	if err != nil {
		return nil, err
	}

	dF2dx, err := differentiate(F[1], vx.Name)
	if err != nil {
		return nil, err
	}
	dF1dy, err := differentiate(F[0], vy.Name)
	if err != nil {
		return nil, err
	}
	negDF1dy, err := simplifyMul([]Node{mustRational(-1, 1), dF1dy})
	if err != nil {
		return nil, err
	}
	c3, err := simplifyAdd([]Node{dF2dx, negDF1dy})
	if err != nil {
		return nil, err
	}

	return &ListNode{Elements: []Node{expandNode(c1), expandNode(c2), expandNode(c3)}}, nil
}
