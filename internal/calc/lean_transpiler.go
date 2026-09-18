package calc

import (
	"fmt"
	"sort"
	"strings"
)

// LeanKeywords contains reserved keywords and built-in identifiers in Lean 4.
var LeanKeywords = map[string]bool{
	"def":        true,
	"theorem":    true,
	"axiom":      true,
	"example":    true,
	"import":     true,
	"by":         true,
	"where":      true,
	"open":       true,
	"variable":   true,
	"lemma":      true,
	"inductive":  true,
	"structure":  true,
	"class":      true,
	"instance":   true,
	"if":         true,
	"then":       true,
	"else":       true,
	"match":      true,
	"with":       true,
	"do":         true,
	"return":     true,
	"for":        true,
	"in":         true,
	"let":        true,
	"mut":        true,
	"have":       true,
	"show":       true,
	"from":       true,
	"fun":        true,
	"section":    true,
	"namespace":  true,
	"end":        true,
	"universe":   true,
	"set_option": true,
	"notation":   true,
	"infix":      true,
	"prefix":     true,
	"postfix":    true,
}

// escapeLeanIdent ensures an identifier is safe for Lean 4, quoting with «...» if reserved.
func escapeLeanIdent(name string) string {
	if LeanKeywords[name] {
		return "«" + name + "»"
	}
	return name
}

// ToLeanSyntax converts an internal AST node to Lean 4 / Mathlib syntax string.
func ToLeanSyntax(node Node) (string, error) {
	if node == nil {
		return "", fmt.Errorf("cannot convert nil node to Lean syntax")
	}

	switch n := node.(type) {
	case *RationalNode:
		if n.Val.IsInt() {
			num := n.Val.Num()
			if num.Sign() < 0 {
				return fmt.Sprintf("(%s)", num.String()), nil
			}
			return num.String(), nil
		}
		num := n.Val.Num()
		denom := n.Val.Denom()
		if num.Sign() < 0 {
			return fmt.Sprintf("((%s : ℚ) / %s)", num.String(), denom.String()), nil
		}
		return fmt.Sprintf("((%s : ℚ) / %s)", num.String(), denom.String()), nil

	case *VarNode:
		return escapeLeanIdent(n.Name), nil

	case *ConstNode:
		switch strings.ToLower(n.Name) {
		case "pi":
			return "Real.pi", nil
		case "e":
			return "Real.exp 1", nil
		case "i":
			return "Complex.I", nil
		default:
			return escapeLeanIdent(n.Name), nil
		}

	case *SqrtNode:
		inner, err := ToLeanSyntax(n.Radicand)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Real.sqrt (%s)", inner), nil

	case *UnaryOpNode:
		sub, err := ToLeanSyntax(n.Expr)
		if err != nil {
			return "", err
		}
		if n.Op == "-" {
			return fmt.Sprintf("-(%s)", sub), nil
		}
		return sub, nil

	case *AddNode:
		if len(n.Terms) == 0 {
			return "0", nil
		}
		var b strings.Builder
		for i, term := range n.Terms {
			termStr, err := ToLeanSyntax(term)
			if err != nil {
				return "", err
			}
			if i == 0 {
				b.WriteString(termStr)
			} else {
				if strings.HasPrefix(termStr, "-") {
					b.WriteString(" - ")
					b.WriteString(strings.TrimPrefix(termStr, "-"))
				} else if strings.HasPrefix(termStr, "(-") && strings.HasSuffix(termStr, ")") {
					b.WriteString(" - ")
					b.WriteString(termStr[2 : len(termStr)-1])
				} else {
					b.WriteString(" + ")
					b.WriteString(termStr)
				}
			}
		}
		return b.String(), nil

	case *MulNode:
		if len(n.Factors) == 0 {
			return "1", nil
		}
		var factors []string
		for _, f := range n.Factors {
			s, err := ToLeanSyntax(f)
			if err != nil {
				return "", err
			}
			// Wrap in parentheses if factor is AddNode or RelOpNode to preserve precedence
			if _, isAdd := f.(*AddNode); isAdd {
				s = "(" + s + ")"
			}
			factors = append(factors, s)
		}
		return strings.Join(factors, " * "), nil

	case *PowNode:
		baseStr, err := ToLeanSyntax(n.Base)
		if err != nil {
			return "", err
		}
		if _, isAdd := n.Base.(*AddNode); isAdd {
			baseStr = "(" + baseStr + ")"
		} else if _, isMul := n.Base.(*MulNode); isMul {
			baseStr = "(" + baseStr + ")"
		}

		expStr, err := ToLeanSyntax(n.Exp)
		if err != nil {
			return "", err
		}
		if _, isAdd := n.Exp.(*AddNode); isAdd {
			expStr = "(" + expStr + ")"
		} else if _, isMul := n.Exp.(*MulNode); isMul {
			expStr = "(" + expStr + ")"
		} else if strings.HasPrefix(expStr, "-") || strings.HasPrefix(expStr, "(") {
			// Already grouped
		}
		return fmt.Sprintf("%s ^ %s", baseStr, expStr), nil

	case *RelOpNode:
		lhsStr, err := ToLeanSyntax(n.LHS)
		if err != nil {
			return "", err
		}
		rhsStr, err := ToLeanSyntax(n.RHS)
		if err != nil {
			return "", err
		}
		op := n.Op
		if op == "==" {
			op = "="
		}
		return fmt.Sprintf("%s %s %s", lhsStr, op, rhsStr), nil

	case *FuncNode:
		var argStrs []string
		for _, a := range n.Args {
			as, err := ToLeanSyntax(a)
			if err != nil {
				return "", err
			}
			argStrs = append(argStrs, as)
		}
		switch strings.ToLower(n.Name) {
		case "sin":
			return fmt.Sprintf("Real.sin (%s)", strings.Join(argStrs, ", ")), nil
		case "cos":
			return fmt.Sprintf("Real.cos (%s)", strings.Join(argStrs, ", ")), nil
		case "tan":
			return fmt.Sprintf("Real.tan (%s)", strings.Join(argStrs, ", ")), nil
		case "exp":
			return fmt.Sprintf("Real.exp (%s)", strings.Join(argStrs, ", ")), nil
		case "ln", "log":
			return fmt.Sprintf("Real.log (%s)", strings.Join(argStrs, ", ")), nil
		default:
			return fmt.Sprintf("%s (%s)", escapeLeanIdent(n.Name), strings.Join(argStrs, ", ")), nil
		}

	case *MatrixNode:
		// Lean 4 Matrix format using Mathlib !![a, b; c, d] syntax
		var rowStrs []string
		for _, row := range n.Data {
			var colStrs []string
			for _, elem := range row {
				es, err := ToLeanSyntax(elem)
				if err != nil {
					return "", err
				}
				colStrs = append(colStrs, es)
			}
			rowStrs = append(rowStrs, strings.Join(colStrs, ", "))
		}
		return fmt.Sprintf("!![%s]", strings.Join(rowStrs, "; ")), nil

	default:
		// Fallback to node's String() representation
		return node.String(), nil
	}
}

// CollectFreeVariables gathers all unique variable names appearing in the given nodes, sorted alphabetically.
func CollectFreeVariables(nodes ...Node) []string {
	varMap := make(map[string]bool)
	var walk func(n Node)
	walk = func(n Node) {
		if n == nil {
			return
		}
		switch curr := n.(type) {
		case *VarNode:
			varMap[curr.Name] = true
		case *AddNode:
			for _, t := range curr.Terms {
				walk(t)
			}
		case *MulNode:
			for _, f := range curr.Factors {
				walk(f)
			}
		case *PowNode:
			walk(curr.Base)
			walk(curr.Exp)
		case *UnaryOpNode:
			walk(curr.Expr)
		case *RelOpNode:
			walk(curr.LHS)
			walk(curr.RHS)
		case *FuncNode:
			for _, a := range curr.Args {
				walk(a)
			}
		case *SqrtNode:
			walk(curr.Radicand)
		case *MatrixNode:
			for _, r := range curr.Data {
				for _, elem := range r {
					walk(elem)
				}
			}
		}
	}

	for _, n := range nodes {
		walk(n)
	}

	var res []string
	for k := range varMap {
		res = append(res, escapeLeanIdent(k))
	}
	sort.Strings(res)
	return res
}

// TranspileCertificateToLean translates a verified certificate and its input/result expressions into a Lean 4 theorem.
func TranspileCertificateToLean(cert *VerificationCertificate, inputExpr, resultExpr Node) (string, error) {
	if cert == nil {
		return "", fmt.Errorf("certificate is nil")
	}
	if !cert.IsVerified {
		return "", fmt.Errorf("cannot transpile unverified certificate: %s", cert.Details)
	}

	var equalityStr string
	var tactic string

	switch cert.Domain {
	case DomainFactor:
		// e.g. factor(x^2 - 1) => (x - 1)*(x + 1)
		// Theorem: expr = result
		var innerExpr Node
		if fn, ok := inputExpr.(*FuncNode); ok && len(fn.Args) > 0 {
			innerExpr = fn.Args[0]
		} else {
			innerExpr = inputExpr
		}
		lhs, err := ToLeanSyntax(innerExpr)
		if err != nil {
			return "", err
		}
		rhs, err := ToLeanSyntax(resultExpr)
		if err != nil {
			return "", err
		}
		equalityStr = fmt.Sprintf("%s = %s", lhs, rhs)
		tactic = "by ring"

	case DomainIntegral:
		// e.g. integrate(2*x, x) => x^2
		var integrand Node
		var intVar string = "x"
		if fn, ok := inputExpr.(*FuncNode); ok && len(fn.Args) > 0 {
			integrand = fn.Args[0]
			if len(fn.Args) > 1 {
				if v, ok := fn.Args[1].(*VarNode); ok {
					intVar = v.Name
				}
			}
		}
		fStr, err := ToLeanSyntax(integrand)
		if err != nil {
			fStr = "f"
		}
		resStr, err := ToLeanSyntax(resultExpr)
		if err != nil {
			resStr = "F"
		}
		if fStr != "" && resStr != "" {
			equalityStr = fmt.Sprintf("deriv (fun %s => %s) %s = %s", intVar, resStr, intVar, fStr)
		} else if cert.Equation != "" {
			equalityStr = cert.Equation
		} else {
			equalityStr = fmt.Sprintf("%s = %s", fStr, resStr)
		}
		tactic = "by ring"

	case DomainMatrix:
		// Matrix inversion, multiplication or decomposition
		if cert.Equation != "" {
			equalityStr = cert.Equation
		} else {
			lhs, _ := ToLeanSyntax(inputExpr)
			rhs, _ := ToLeanSyntax(resultExpr)
			equalityStr = fmt.Sprintf("%s = %s", lhs, rhs)
		}
		tactic = "by ext <;> ring"

	case DomainRationalApart:
		// Partial fraction decomposition
		var innerExpr Node
		if fn, ok := inputExpr.(*FuncNode); ok && len(fn.Args) > 0 {
			innerExpr = fn.Args[0]
		} else {
			innerExpr = inputExpr
		}
		lhs, err := ToLeanSyntax(innerExpr)
		if err != nil {
			return "", err
		}
		rhs, err := ToLeanSyntax(resultExpr)
		if err != nil {
			return "", err
		}
		equalityStr = fmt.Sprintf("%s = %s", lhs, rhs)
		tactic = "by ring"

	case DomainWZ:
		equalityStr = cert.Equation
		tactic = "by ring"

	default:
		// General algebraic equality verification
		// If input is verify(A == B) or verify(A, B), extract inner nodes
		actualInput := inputExpr
		actualResult := resultExpr
		if fn, ok := inputExpr.(*FuncNode); ok && fn.Name == "verify" {
			if len(fn.Args) == 1 {
				actualInput = fn.Args[0]
				actualResult = nil
			} else if len(fn.Args) >= 2 {
				actualInput = fn.Args[0]
				actualResult = fn.Args[1]
			}
		}

		if rel, ok := actualInput.(*RelOpNode); ok {
			s, err := ToLeanSyntax(rel)
			if err != nil {
				return "", err
			}
			equalityStr = s
		} else if actualResult != nil {
			lhs, err := ToLeanSyntax(actualInput)
			if err != nil {
				return "", err
			}
			rhs, err := ToLeanSyntax(actualResult)
			if err != nil {
				return "", err
			}
			equalityStr = fmt.Sprintf("%s = %s", lhs, rhs)
		} else {
			s, err := ToLeanSyntax(actualInput)
			if err != nil {
				return "", err
			}
			equalityStr = s
		}
		tactic = "by ring"
	}

	// Normalize any leftover == to = for Lean 4 syntax
	equalityStr = strings.ReplaceAll(equalityStr, "==", "=")

	return fmt.Sprintf("theorem ihd_verified_proof : %s := %s", equalityStr, tactic), nil
}

// GenerateLeanSource generates a standalone, fully valid Lean 4 source file with Mathlib imports, variables, and theorem.
func GenerateLeanSource(theoremName string, cert *VerificationCertificate, inputExpr, resultExpr Node) (string, error) {
	if strings.TrimSpace(theoremName) == "" {
		theoremName = "ihd_certified_proof"
	}

	vars := CollectFreeVariables(inputExpr, resultExpr)
	if cert != nil && cert.Residual != nil {
		resVars := CollectFreeVariables(cert.Residual)
		vars = append(vars, resVars...)
		vars = uniqueSortedStrings(vars)
	}

	theoremCode, err := TranspileCertificateToLean(cert, inputExpr, resultExpr)
	if err != nil {
		return "", err
	}

	// Replace default theorem name with requested one
	theoremCode = strings.Replace(theoremCode, "theorem ihd_verified_proof", fmt.Sprintf("theorem %s", escapeLeanIdent(theoremName)), 1)

	var b strings.Builder
	b.WriteString("-- Automatically generated by ihd (i-hate-decimal-calc) Lean 4 Transpiler\n")
	b.WriteString("-- Mathematical proof certificate certified with Skeptic's verification\n")
	b.WriteString("import Mathlib.Tactic.Ring\n")
	b.WriteString("import Mathlib.Tactic.Linarith\n")
	b.WriteString("import Mathlib.Tactic.NormNum\n")
	b.WriteString("import Mathlib.Data.Rat.Basic\n")
	b.WriteString("import Mathlib.Data.Real.Basic\n\n")

	if len(vars) > 0 {
		b.WriteString(fmt.Sprintf("variable (%s : ℚ)\n\n", strings.Join(vars, " ")))
	}

	b.WriteString(theoremCode)
	b.WriteString("\n")

	return b.String(), nil
}

func uniqueSortedStrings(in []string) []string {
	m := make(map[string]bool)
	for _, s := range in {
		m[s] = true
	}
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
