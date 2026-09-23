package ast

import (
	"sort"
)

// -------------------------------------------------------------------------
// AST Traversal & Transformation (Walker Pattern)
// -------------------------------------------------------------------------

// Walk traverses an AST in depth-first pre-order.
// If visitor returns false, Walk does not visit node's children.
func Walk(node Node, visitor func(Node) bool) {
	if node == nil {
		return
	}
	if !visitor(node) {
		return
	}

	switch v := node.(type) {
	case *SqrtNode:
		Walk(v.Radicand, visitor)
	case *FuncNode:
		for _, arg := range v.Args {
			Walk(arg, visitor)
		}
	case *ComplexNode:
		Walk(v.Real, visitor)
		Walk(v.Imag, visitor)
	case *AddNode:
		for _, term := range v.Terms {
			Walk(term, visitor)
		}
	case *MulNode:
		for _, factor := range v.Factors {
			Walk(factor, visitor)
		}
	case *PowNode:
		Walk(v.Base, visitor)
		Walk(v.Exp, visitor)
	case *UnaryOpNode:
		Walk(v.Expr, visitor)
	case *ListNode:
		for _, elem := range v.Elements {
			Walk(elem, visitor)
		}
	case *MatrixNode:
		for _, row := range v.Data {
			for _, cell := range row {
				Walk(cell, visitor)
			}
		}
	case *RelOpNode:
		Walk(v.LHS, visitor)
		Walk(v.RHS, visitor)
	case *QuantifierNode:
		Walk(v.Body, visitor)
	case *RationalNode, *ConstNode, *VarNode, *PlotNode, *PolyNode, *AlgebraicNumberNode:
		// Leaf nodes: no children
	}
}

// Inspect traverses an AST unconditionally in depth-first order.
func Inspect(node Node, visitor func(Node)) {
	Walk(node, func(n Node) bool {
		visitor(n)
		return true
	})
}

// Transform traverses an AST in post-order and transforms nodes bottom-up.
// Child nodes are transformed first, and then transformer is called on the newly constructed node.
func Transform(node Node, transformer func(Node) Node) Node {
	if node == nil {
		return nil
	}

	var transformedChild Node
	switch v := node.(type) {
	case *SqrtNode:
		transformedChild = &SqrtNode{
			Radicand: Transform(v.Radicand, transformer),
		}
	case *FuncNode:
		newArgs := make([]Node, len(v.Args))
		for i, a := range v.Args {
			newArgs[i] = Transform(a, transformer)
		}
		transformedChild = &FuncNode{
			Name: v.Name,
			Args: newArgs,
		}
	case *ComplexNode:
		transformedChild = &ComplexNode{
			Real: Transform(v.Real, transformer),
			Imag: Transform(v.Imag, transformer),
		}
	case *AddNode:
		newTerms := make([]Node, len(v.Terms))
		for i, t := range v.Terms {
			newTerms[i] = Transform(t, transformer)
		}
		transformedChild = &AddNode{
			Terms: newTerms,
		}
	case *MulNode:
		newFactors := make([]Node, len(v.Factors))
		for i, f := range v.Factors {
			newFactors[i] = Transform(f, transformer)
		}
		transformedChild = &MulNode{
			Factors: newFactors,
		}
	case *PowNode:
		transformedChild = &PowNode{
			Base: Transform(v.Base, transformer),
			Exp:  Transform(v.Exp, transformer),
		}
	case *UnaryOpNode:
		transformedChild = &UnaryOpNode{
			Op:   v.Op,
			Expr: Transform(v.Expr, transformer),
		}
	case *ListNode:
		newElems := make([]Node, len(v.Elements))
		for i, e := range v.Elements {
			newElems[i] = Transform(e, transformer)
		}
		transformedChild = &ListNode{
			Elements: newElems,
		}
	case *MatrixNode:
		newData := make([][]Node, v.Rows)
		for r := 0; r < v.Rows; r++ {
			newData[r] = make([]Node, v.Cols)
			for c := 0; c < v.Cols; c++ {
				newData[r][c] = Transform(v.Data[r][c], transformer)
			}
		}
		transformedChild = &MatrixNode{
			Rows: v.Rows,
			Cols: v.Cols,
			Data: newData,
		}
	case *RelOpNode:
		transformedChild = &RelOpNode{
			LHS: Transform(v.LHS, transformer),
			Op:  v.Op,
			RHS: Transform(v.RHS, transformer),
		}
	case *QuantifierNode:
		transformedChild = &QuantifierNode{
			Kind: v.Kind,
			Vars: append([]string(nil), v.Vars...),
			Body: Transform(v.Body, transformer),
		}
	case *RationalNode, *ConstNode, *VarNode, *PlotNode, *PolyNode, *AlgebraicNumberNode:
		transformedChild = node
	default:
		transformedChild = node
	}

	return transformer(transformedChild)
}

// Substitute replaces occurrences of varName with valNode in an expression tree.
func Substitute(node Node, varName string, valNode Node) Node {
	return Transform(node, func(curr Node) Node {
		if v, ok := curr.(*VarNode); ok && v.Name == varName {
			return valNode
		}
		return curr
	})
}

// ContainsVar checks if an AST node contains the given variable name.
func ContainsVar(node Node, varName string) bool {
	found := false
	Walk(node, func(curr Node) bool {
		if found {
			return false
		}
		if v, ok := curr.(*VarNode); ok && v.Name == varName {
			found = true
			return false
		}
		if p, ok := curr.(*PolyNode); ok {
			for _, v := range p.Vars {
				if v == varName {
					found = true
					return false
				}
			}
		}
		if a, ok := curr.(*AlgebraicNumberNode); ok {
			if a.Symbol == varName {
				found = true
				return false
			}
			if a.MinPoly != nil {
				for _, v := range a.MinPoly.Vars {
					if v == varName {
						found = true
						return false
					}
				}
			}
			if a.RepPoly != nil {
				for _, v := range a.RepPoly.Vars {
					if v == varName {
						found = true
						return false
					}
				}
			}
		}
		return true
	})
	return found
}

// ExtractFreeVariables collects all distinct free variable names occurring in the node, sorted alphabetically.
// Variables bound by QuantifierNodes are excluded.
func ExtractFreeVariables(node Node) []string {
	varMap := make(map[string]bool)
	boundVars := make(map[string]int)

	var walkScoped func(n Node)
	walkScoped = func(n Node) {
		if n == nil {
			return
		}
		if q, ok := n.(*QuantifierNode); ok {
			for _, v := range q.Vars {
				boundVars[v]++
			}
			walkScoped(q.Body)
			for _, v := range q.Vars {
				boundVars[v]--
				if boundVars[v] <= 0 {
					delete(boundVars, v)
				}
			}
			return
		}
		if v, ok := n.(*VarNode); ok {
			if boundVars[v.Name] == 0 {
				varMap[v.Name] = true
			}
			return
		}
		if p, ok := n.(*PolyNode); ok {
			for _, v := range p.Vars {
				if boundVars[v] == 0 {
					varMap[v] = true
				}
			}
			return
		}
		if a, ok := n.(*AlgebraicNumberNode); ok {
			if a.Symbol != "" && boundVars[a.Symbol] == 0 {
				varMap[a.Symbol] = true
			}
			if a.MinPoly != nil {
				for _, v := range a.MinPoly.Vars {
					if boundVars[v] == 0 {
						varMap[v] = true
					}
				}
			}
			if a.RepPoly != nil {
				for _, v := range a.RepPoly.Vars {
					if boundVars[v] == 0 {
						varMap[v] = true
					}
				}
			}
			return
		}

		switch v := n.(type) {
		case *SqrtNode:
			walkScoped(v.Radicand)
		case *FuncNode:
			for _, arg := range v.Args {
				walkScoped(arg)
			}
		case *ComplexNode:
			walkScoped(v.Real)
			walkScoped(v.Imag)
		case *AddNode:
			for _, t := range v.Terms {
				walkScoped(t)
			}
		case *MulNode:
			for _, f := range v.Factors {
				walkScoped(f)
			}
		case *PowNode:
			walkScoped(v.Base)
			walkScoped(v.Exp)
		case *UnaryOpNode:
			walkScoped(v.Expr)
		case *ListNode:
			for _, e := range v.Elements {
				walkScoped(e)
			}
		case *MatrixNode:
			for _, row := range v.Data {
				for _, cell := range row {
					walkScoped(cell)
				}
			}
		case *RelOpNode:
			walkScoped(v.LHS)
			walkScoped(v.RHS)
		}
	}

	walkScoped(node)

	if len(varMap) == 0 {
		return nil
	}
	vars := make([]string, 0, len(varMap))
	for name := range varMap {
		vars = append(vars, name)
	}
	sort.Strings(vars)
	return vars
}
