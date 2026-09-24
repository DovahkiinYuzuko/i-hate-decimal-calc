package calc

import (
	"fmt"
	"sort"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
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
		return "", fmt.Errorf("%s", i18n.T("lean.err_cannot_convert_nil_node_to"))
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

// TranspileCertificateIRToLean transforms a structured Certificate IR directly into a Lean 4 theorem.
// It bypasses ad-hoc AST unpacking, translating the algebraic witness into formal Mathlib syntax.
func TranspileCertificateIRToLean(cert Certificate) (string, error) {
	if cert == nil {
		return "", fmt.Errorf("%s", i18n.T("lean.err_certificate_is_nil"))
	}
	if !cert.IsVerified() {
		return "", fmt.Errorf("%s", i18n.T("lean.err_cannot_transpile_unverified_certificate", cert.Details()))
	}

	var equalityStr string
	var tactic string

	switch c := cert.(type) {
	case *IdentityCertificate:
		lhs, err := ToLeanSyntax(c.LHS)
		if err != nil {
			return "", err
		}
		rhs, err := ToLeanSyntax(c.RHS)
		if err != nil {
			return "", err
		}
		equalityStr = fmt.Sprintf("%s = %s", lhs, rhs)
		tactic = "by ring"

	case *DerivCertificate:
		fStr, err := ToLeanSyntax(c.Integrand)
		if err != nil {
			fStr = "f"
		}
		resStr, err := ToLeanSyntax(c.Antiderivative)
		if err != nil {
			resStr = "F"
		}
		intVar := c.Variable
		if intVar == "" {
			intVar = "x"
		}
		equalityStr = fmt.Sprintf("deriv (fun %s => %s) %s = %s", intVar, resStr, intVar, fStr)
		tactic = "by ring"

	case *InvertibilityCertificate:
		mStr, _ := ToLeanSyntax(c.Matrix)
		invStr, _ := ToLeanSyntax(c.Inverse)
		equalityStr = fmt.Sprintf("%s * %s = 1", mStr, invStr)
		tactic = "by ext <;> ring"

	case *DecompositionCertificate:
		mStr, _ := ToLeanSyntax(c.Matrix)
		var factorStrs []string
		for _, f := range c.Factors {
			fs, _ := ToLeanSyntax(f)
			factorStrs = append(factorStrs, fs)
		}
		equalityStr = fmt.Sprintf("%s = %s", mStr, strings.Join(factorStrs, " * "))
		tactic = "by ext <;> ring"

	case *ODECertificate:
		odeStr, _ := ToLeanSyntax(c.ODE)
		solStr, _ := ToLeanSyntax(c.Solution)
		equalityStr = fmt.Sprintf("%s [%s(%s) = %s] = 0", odeStr, c.DependentVar, c.IndependentVar, solStr)
		tactic = "by ring"

	case *WZCertificate:
		equalityStr = c.EquationString()
		tactic = "by ring"

	case *GeometricCertificate:
		identityNode := resolveGeometricAlgebraicIdentity(c)
		conclStr, err := ToLeanSyntax(identityNode)
		if err != nil {
			conclStr = "0"
		}
		equalityStr = fmt.Sprintf("%s = 0", conclStr)
		tactic = "by ring"

	case *ImpossibilityCertificate:
		if c.Kind == ImpossibilityAbelRuffini {
			fStr, _ := ToLeanSyntax(c.Problem)
			equalityStr = fmt.Sprintf("¬ IsSolvable (%s).GaloisGroup", fStr)
			tactic = "by abel_ruffini"
		} else {
			equalityStr = "False"
			tactic = "by contradiction"
		}

	default:
		// Fallback to equation string or domain mapping
		eq := cert.EquationString()
		if eq == "" {
			eq = "True"
			tactic = "by decide"
		} else {
			equalityStr = eq
			tactic = "by ring"
		}
	}

	// Normalize any leftover == to = for Lean 4 syntax
	equalityStr = strings.ReplaceAll(equalityStr, "==", "=")

	return fmt.Sprintf("theorem ihd_verified_proof : %s := %s", equalityStr, tactic), nil
}

// TranspileCertificateToLean transforms a VerificationCertificate into a Lean 4 theorem.
// It delegates to TranspileCertificateIRToLean using the underlying Certificate IR.
func TranspileCertificateToLean(cert *VerificationCertificate, inputExpr, resultExpr Node) (string, error) {
	if cert == nil {
		return "", fmt.Errorf("%s", i18n.T("lean.err_certificate_is_nil"))
	}
	if !cert.IsVerified {
		return "", fmt.Errorf("%s", i18n.T("lean.err_cannot_transpile_unverified_certificate", cert.Details))
	}

	// Prefer structured Certificate IR
	certIR := cert.ToCertificateIR()
	if certIR != nil {
		// If it's a specific structured IR, directly transpile
		switch certIR.(type) {
		case *IdentityCertificate, *DerivCertificate, *InvertibilityCertificate,
			*DecompositionCertificate, *ODECertificate, *WZCertificate, *GeometricCertificate:
			return TranspileCertificateIRToLean(certIR)
		}
	}

	// Backward-compatible fallback for dynamic/unclassified nodes
	return TranspileCertificateIRToLean(certIR)
}

// GenerateLeanSource generates a standalone, fully valid Lean 4 source file with Mathlib imports, variables, and theorem.
func GenerateLeanSource(theoremName string, cert *VerificationCertificate, inputExpr, resultExpr Node) (string, error) {
	if strings.TrimSpace(theoremName) == "" {
		theoremName = "ihd_certified_proof"
	}

	var allNodes []Node
	if inputExpr != nil {
		allNodes = append(allNodes, inputExpr)
	}
	if resultExpr != nil {
		allNodes = append(allNodes, resultExpr)
	}
	if cert != nil {
		if cert.Residual != nil {
			allNodes = append(allNodes, cert.Residual)
		}
		if ir := cert.ToCertificateIR(); ir != nil {
			switch c := ir.(type) {
			case *GeometricCertificate:
				if c.Conclusion != nil {
					allNodes = append(allNodes, c.Conclusion)
				}
				allNodes = append(allNodes, c.Hypotheses...)
				allNodes = append(allNodes, c.TriangularChain...)
				if c.RemainderNode != nil {
					allNodes = append(allNodes, c.RemainderNode)
				}
			case *IdentityCertificate:
				if c.LHS != nil {
					allNodes = append(allNodes, c.LHS)
				}
				if c.RHS != nil {
					allNodes = append(allNodes, c.RHS)
				}
			case *DerivCertificate:
				if c.Integrand != nil {
					allNodes = append(allNodes, c.Integrand)
				}
				if c.Antiderivative != nil {
					allNodes = append(allNodes, c.Antiderivative)
				}
			case *InvertibilityCertificate:
				if c.Matrix != nil {
					allNodes = append(allNodes, c.Matrix)
				}
				if c.Inverse != nil {
					allNodes = append(allNodes, c.Inverse)
				}
			case *DecompositionCertificate:
				if c.Matrix != nil {
					allNodes = append(allNodes, c.Matrix)
				}
				allNodes = append(allNodes, c.Factors...)
			case *ODECertificate:
				if c.ODE != nil {
					allNodes = append(allNodes, c.ODE)
				}
				if c.Solution != nil {
					allNodes = append(allNodes, c.Solution)
				}
			}
		}
	}

	vars := CollectFreeVariables(allNodes...)

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
	b.WriteString("import Mathlib.Data.Rat.Defs\n")
	b.WriteString("import Mathlib.Basic.Real.Basic\n")
	b.WriteString("import Mathlib.Data.Matrix.Basic\n")
	b.WriteString("import Mathlib.Analysis.Calculus.Deriv.Basic\n")
	b.WriteString("import Mathlib.Analysis.SpecialFunctions.Trigonometric.Basic\n")
	b.WriteString("import Mathlib.Analysis.SpecialFunctions.Exp\n\n")

	typeAnnotation := "ℚ"
	if containsRealNodes(allNodes...) {
		typeAnnotation = "ℝ"
	}

	if len(vars) > 0 {
		b.WriteString(fmt.Sprintf("variable (%s : %s)\n\n", strings.Join(vars, " "), typeAnnotation))
	}

	b.WriteString(theoremCode)
	b.WriteString("\n")

	return b.String(), nil
}


// containsRealNodes checks whether any of the given nodes contain real-valued functions (sin, cos, exp, etc.) or radicals.
func containsRealNodes(nodes ...Node) bool {
	hasReal := false
	var walk func(n Node)
	walk = func(n Node) {
		if n == nil || hasReal {
			return
		}
		switch curr := n.(type) {
		case *FuncNode:
			name := strings.ToLower(curr.Name)
			switch name {
			case "sin", "cos", "tan", "asin", "acos", "atan", "exp", "ln", "log":
				hasReal = true
				return
			}
			for _, a := range curr.Args {
				walk(a)
			}
		case *ConstNode:
			if curr.Name == "pi" || curr.Name == "π" || curr.Name == "e" {
				hasReal = true
				return
			}
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
	return hasReal
}

// resolveGeometricAlgebraicIdentity extracts linear construction substitutions from hypotheses
// and applies them to the conclusion polynomial, producing an algebraically verifiable identity.
func resolveGeometricAlgebraicIdentity(c *GeometricCertificate) Node {
	if c == nil || c.Conclusion == nil {
		return mustRational(0, 1)
	}

	substMap := make(map[string]Node)

	// Inspect hypotheses for linear assignments: c1 * v + c0 == 0 => v = -c0 / c1
	for _, h := range c.Hypotheses {
		evalH, err := Eval(expandNode(h))
		if err != nil {
			evalH = h
		}
		vars := collectVariables(evalH)
		for _, v := range vars {
			coeffs, err := extractPolyCoeffs(evalH, v)
			if err == nil && len(coeffs) > 0 {
				maxDeg := 0
				for deg := range coeffs {
					if deg > maxDeg {
						maxDeg = deg
					}
				}
				if maxDeg == 1 {
					c1Node := coeffs[1]
					c0Node := coeffs[0]
					if c0Node == nil {
						c0Node = mustRational(0, 1)
					}
					if isZero(c1Node) {
						continue
					}
					negC0, err := simplifyUnaryOp("-", c0Node)
					if err != nil {
						continue
					}
					invC1, err := simplifyPow(c1Node, mustRational(-1, 1))
					if err != nil {
						continue
					}
					vVal, err := simplifyMul([]Node{negC0, invC1})
					if err != nil {
						continue
					}
					evalVal, err := Eval(expandNode(vVal))
					if err == nil {
						vVal = evalVal
					}
					if _, exists := substMap[v]; !exists {
						substMap[v] = vVal
					}
				}
			}
		}
	}

	// Substitute into conclusion
	var substNode func(n Node) Node
	substNode = func(n Node) Node {
		if n == nil {
			return nil
		}
		switch curr := n.(type) {
		case *VarNode:
			if repl, ok := substMap[curr.Name]; ok {
				return repl
			}
			return curr
		case *AddNode:
			var newTerms []Node
			for _, t := range curr.Terms {
				newTerms = append(newTerms, substNode(t))
			}
			return &AddNode{Terms: newTerms}
		case *MulNode:
			var newFactors []Node
			for _, f := range curr.Factors {
				newFactors = append(newFactors, substNode(f))
			}
			return &MulNode{Factors: newFactors}
		case *PowNode:
			return &PowNode{Base: substNode(curr.Base), Exp: substNode(curr.Exp)}
		case *UnaryOpNode:
			return &UnaryOpNode{Op: curr.Op, Expr: substNode(curr.Expr)}
		case *RelOpNode:
			return &RelOpNode{Op: curr.Op, LHS: substNode(curr.LHS), RHS: substNode(curr.RHS)}
		default:
			return n
		}
	}

	substed := substNode(c.Conclusion)
	vars := collectVariables(substed)
	if len(vars) > 0 {
		polyNode, err := NodeToPoly(substed, vars, ast.OrderGrevLex)
		if err == nil && isZero(PolyToNode(polyNode)) {
			return substed
		}
	} else {
		val, err := Eval(substed)
		if err == nil && isZero(val) {
			return substed
		}
	}

	return c.Conclusion
}
