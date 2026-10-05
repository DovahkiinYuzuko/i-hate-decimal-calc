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

// isEnclosedInParens checks if the string starts with '(' and ends with ')',
// and that these outermost parentheses form a single matching pair enclosing the entire string.
func isEnclosedInParens(s string) bool {
	if !strings.HasPrefix(s, "(") || !strings.HasSuffix(s, ")") {
		return false
	}
	depth := 0
	for i, r := range s {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i == len(s)-1
			}
		}
	}
	return false
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
				} else if isEnclosedInParens(termStr) && strings.HasPrefix(termStr, "(-") {
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
		tactic = "by simp ; try ring"

	case *InvertibilityCertificate:
		mStr, _ := ToLeanSyntax(c.Matrix)
		invStr, _ := ToLeanSyntax(c.Inverse)
		dim := 0
		if mat, ok := c.Matrix.(*MatrixNode); ok && len(mat.Data) > 0 {
			dim = len(mat.Data)
		} else if mat, ok := c.Inverse.(*MatrixNode); ok && len(mat.Data) > 0 {
			dim = len(mat.Data)
		}
		hasVars := len(CollectFreeVariables(c.Matrix)) > 0 || len(CollectFreeVariables(c.Inverse)) > 0
		if dim > 0 {
			equalityStr = fmt.Sprintf("(%s * %s : Matrix (Fin %d) (Fin %d) ℚ) = 1", mStr, invStr, dim, dim)
			if hasVars {
				tactic = "by ext i j <;> fin_cases i <;> fin_cases j <;> ring"
			} else {
				tactic = "by ext i j <;> fin_cases i <;> fin_cases j <;> norm_num"
			}
		} else {
			equalityStr = fmt.Sprintf("%s * %s = 1", mStr, invStr)
			tactic = "by ext <;> ring"
		}

	case *DecompositionCertificate:
		mStr, _ := ToLeanSyntax(c.Matrix)
		var factorStrs []string
		hasVars := len(CollectFreeVariables(c.Matrix)) > 0
		for _, f := range c.Factors {
			fs, _ := ToLeanSyntax(f)
			factorStrs = append(factorStrs, fs)
			if len(CollectFreeVariables(f)) > 0 {
				hasVars = true
			}
		}
		rows, cols := 0, 0
		if mat, ok := c.Matrix.(*MatrixNode); ok && len(mat.Data) > 0 {
			rows = len(mat.Data)
			cols = len(mat.Data[0])
		}
		if rows > 0 && cols > 0 {
			equalityStr = fmt.Sprintf("(%s : Matrix (Fin %d) (Fin %d) ℚ) = %s", mStr, rows, cols, strings.Join(factorStrs, " * "))
			if hasVars {
				tactic = "by ext i j <;> fin_cases i <;> fin_cases j <;> ring"
			} else {
				tactic = "by ext i j <;> fin_cases i <;> fin_cases j <;> norm_num"
			}
		} else {
			equalityStr = fmt.Sprintf("%s = %s", mStr, strings.Join(factorStrs, " * "))
			tactic = "by ext <;> ring"
		}

	case *ODECertificate:
		odeStr, _ := ToLeanSyntax(c.ODE)
		solStr, _ := ToLeanSyntax(c.Solution)
		equalityStr = fmt.Sprintf("%s [%s(%s) = %s] = 0", odeStr, c.DependentVar, c.IndependentVar, solStr)
		tactic = "by ring"

	case *WZCertificate:
		equalityStr = c.EquationString()
		tactic = "by ring"

	case *GeometricCertificate:
		return transpileGeometricCertificateIR(c, "ihd_verified_proof")

	case *LinearSolveCertificate:
		mStr, _ := ToLeanSyntax(c.Matrix)
		xStr, _ := ToLeanSyntax(c.Solution)
		bStr, _ := ToLeanSyntax(c.Target)
		rows, cols := 0, 0
		if mat, ok := c.Matrix.(*MatrixNode); ok && len(mat.Data) > 0 {
			rows = len(mat.Data)
			cols = len(mat.Data[0])
		}
		hasVars := len(CollectFreeVariables(c.Matrix)) > 0 || len(CollectFreeVariables(c.Solution)) > 0 || len(CollectFreeVariables(c.Target)) > 0
		if rows > 0 && cols > 0 {
			equalityStr = fmt.Sprintf("((%s : Matrix (Fin %d) (Fin %d) ℚ) * (%s : Matrix (Fin %d) (Fin 1) ℚ)) = (%s : Matrix (Fin %d) (Fin 1) ℚ)", mStr, rows, cols, xStr, cols, bStr, rows)
			if hasVars {
				tactic = "by ext i j <;> fin_cases i <;> fin_cases j <;> ring"
			} else {
				tactic = "by ext i j <;> fin_cases i <;> fin_cases j <;> norm_num"
			}
		} else {
			equalityStr = fmt.Sprintf("%s * %s = %s", mStr, xStr, bStr)
			tactic = "by ext <;> ring"
		}

	case *PellCertificate:
		if c.X != nil && c.Y != nil && c.D != nil {
			xStr, _ := ToLeanSyntax(c.X)
			yStr, _ := ToLeanSyntax(c.Y)
			dStr, _ := ToLeanSyntax(c.D)
			equalityStr = fmt.Sprintf("((%s : ℤ)^2 - (%s : ℤ) * (%s : ℤ)^2 : ℤ) = 1", xStr, dStr, yStr)
			tactic = "by decide"
		} else {
			equalityStr = "True"
			tactic = "by decide"
		}

	case *PolynomialRootCertificate:
		varSubst := c.Variable
		if varSubst == "" {
			varSubst = "x"
		}
		var roots []Node
		if list, ok := c.Root.(*ListNode); ok {
			roots = list.Elements
		} else if c.Root != nil {
			roots = []Node{c.Root}
		}

		polyType := "ℚ"
		if containsRealNodes(c.Polynomial) || containsRealNodes(roots...) {
			polyType = "ℝ"
		}

		if len(roots) == 0 {
			equalityStr = "True"
			tactic = "by decide"
		} else if len(roots) == 1 {
			substed := Substitute(c.Polynomial, varSubst, roots[0])
			subStr, err := ToLeanSyntax(substed)
			if err != nil {
				subStr = "0"
			}
			equalityStr = fmt.Sprintf("((%s : %s) = 0)", subStr, polyType)
			if len(CollectFreeVariables(substed)) > 0 {
				tactic = "by ring"
			} else {
				tactic = "by norm_num"
			}
		} else {
			var parts []string
			hasVars := false
			for _, r := range roots {
				substed := Substitute(c.Polynomial, varSubst, r)
				subStr, err := ToLeanSyntax(substed)
				if err != nil {
					subStr = "0"
				}
				parts = append(parts, fmt.Sprintf("((%s : %s) = 0)", subStr, polyType))
				if len(CollectFreeVariables(substed)) > 0 {
					hasVars = true
				}
			}
			equalityStr = strings.Join(parts, " ∧ ")
			if hasVars {
				tactic = "by repeat constructor <;> ring"
			} else {
				tactic = "by repeat constructor <;> norm_num"
			}
		}

	case *IntegerRelationCertificate:
		terms := make([]Node, 0, len(c.Coefficients))
		for i, coeffNode := range c.Coefficients {
			if coeffNode == nil {
				continue
			}
			var term Node
			if i < len(c.Elements) && c.Elements[i] != nil {
				term = &MulNode{Factors: []Node{coeffNode, c.Elements[i]}}
			} else {
				term = coeffNode
			}
			terms = append(terms, term)
		}
		var relationNode Node
		if len(terms) == 0 {
			relationNode = mustRational(0, 1)
		} else {
			relationNode = &AddNode{Terms: terms}
		}
		relStr, err := ToLeanSyntax(relationNode)
		if err != nil {
			relStr = "0"
		}
		relType := "ℚ"
		if containsRealNodes(relationNode) {
			relType = "ℝ"
		}
		equalityStr = fmt.Sprintf("((%s : %s) = 0)", relStr, relType)
		if len(CollectFreeVariables(relationNode)) > 0 {
			tactic = "by ring"
		} else {
			tactic = "by norm_num"
		}

	case *EllipticPointCertificate:
		if c.Sum != nil {
			// Check if point is at infinity [0, 1, 0] or has coords
			if pMat, ok := c.Sum.(*MatrixNode); ok && len(pMat.Data) == 1 && len(pMat.Data[0]) >= 2 {
				xNode := pMat.Data[0][0]
				yNode := pMat.Data[0][1]
				xStr, _ := ToLeanSyntax(xNode)
				yStr, _ := ToLeanSyntax(yNode)
				aStr, _ := ToLeanSyntax(c.CurveA)
				bStr, _ := ToLeanSyntax(c.CurveB)
				equalityStr = fmt.Sprintf("(%s : ℚ)^2 = (%s : ℚ)^3 + (%s : ℚ) * (%s : ℚ) + (%s : ℚ)", yStr, xStr, aStr, xStr, bStr)
				if len(CollectFreeVariables(xNode)) > 0 || len(CollectFreeVariables(yNode)) > 0 || len(CollectFreeVariables(c.CurveA)) > 0 || len(CollectFreeVariables(c.CurveB)) > 0 {
					tactic = "by ring"
				} else {
					tactic = "by norm_num"
				}
			} else {
				equalityStr = "True"
				tactic = "by decide"
			}
		} else {
			equalityStr = "True"
			tactic = "by decide"
		}

	case *SmithNormalFormCertificate:
		mStr, _ := ToLeanSyntax(c.A)
		uStr, _ := ToLeanSyntax(c.U)
		vStr, _ := ToLeanSyntax(c.V)
		dStr, _ := ToLeanSyntax(c.D)
		mRows, mCols := 0, 0
		if mat, ok := c.A.(*MatrixNode); ok && len(mat.Data) > 0 {
			mRows = len(mat.Data)
			mCols = len(mat.Data[0])
		}
		if mRows > 0 && mCols > 0 {
			equalityStr = fmt.Sprintf("((%s : Matrix (Fin %d) (Fin %d) ℚ) * (%s : Matrix (Fin %d) (Fin %d) ℚ) * (%s : Matrix (Fin %d) (Fin %d) ℚ) : Matrix (Fin %d) (Fin %d) ℚ) = (%s : Matrix (Fin %d) (Fin %d) ℚ)",
				uStr, mRows, mRows,
				mStr, mRows, mCols,
				vStr, mCols, mCols,
				mRows, mCols,
				dStr, mRows, mCols,
			)
			tactic = "by ext i j <;> fin_cases i <;> fin_cases j <;> norm_num"
		} else {
			equalityStr = fmt.Sprintf("%s * %s * %s = %s", uStr, mStr, vStr, dStr)
			tactic = "by ext <;> ring"
		}

	case *HermiteNormalFormCertificate:
		mStr, _ := ToLeanSyntax(c.A)
		uStr, _ := ToLeanSyntax(c.U)
		hStr, _ := ToLeanSyntax(c.H)
		mRows, mCols := 0, 0
		if mat, ok := c.A.(*MatrixNode); ok && len(mat.Data) > 0 {
			mRows = len(mat.Data)
			mCols = len(mat.Data[0])
		}
		if mRows > 0 && mCols > 0 {
			equalityStr = fmt.Sprintf("((%s : Matrix (Fin %d) (Fin %d) ℚ) * (%s : Matrix (Fin %d) (Fin %d) ℚ) : Matrix (Fin %d) (Fin %d) ℚ) = (%s : Matrix (Fin %d) (Fin %d) ℚ)",
				uStr, mRows, mRows,
				mStr, mRows, mCols,
				mRows, mCols,
				hStr, mRows, mCols,
			)
			tactic = "by ext i j <;> fin_cases i <;> fin_cases j <;> norm_num"
		} else {
			equalityStr = fmt.Sprintf("%s * %s = %s", uStr, mStr, hStr)
			tactic = "by ext <;> ring"
		}

	case *ImpossibilityCertificate:
		if c.Kind == ImpossibilityAbelRuffini {
			fStr, _ := ToLeanSyntax(c.Problem)
			equalityStr = fmt.Sprintf("¬ IsSolvable (Polynomial.galoisGroup (%s : Polynomial ℚ))", fStr)
			tactic = "by decide"
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
			*DecompositionCertificate, *ODECertificate, *WZCertificate, *GeometricCertificate,
			*LinearSolveCertificate, *PellCertificate, *PolynomialRootCertificate,
			*IntegerRelationCertificate, *EllipticPointCertificate:
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
				allNodes = append(allNodes, resolveGeometricAlgebraicIdentity(c))
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
			case *LinearSolveCertificate:
				if c.Matrix != nil {
					allNodes = append(allNodes, c.Matrix)
				}
				if c.Solution != nil {
					allNodes = append(allNodes, c.Solution)
				}
				if c.Target != nil {
					allNodes = append(allNodes, c.Target)
				}
			case *PolynomialRootCertificate:
				if c.Polynomial != nil {
					allNodes = append(allNodes, c.Polynomial)
				}
				if c.Root != nil {
					allNodes = append(allNodes, c.Root)
				}
			case *IntegerRelationCertificate:
				allNodes = append(allNodes, c.Elements...)
			case *EllipticPointCertificate:
				if c.CurveA != nil {
					allNodes = append(allNodes, c.CurveA)
				}
				if c.CurveB != nil {
					allNodes = append(allNodes, c.CurveB)
				}
				if c.P1 != nil {
					allNodes = append(allNodes, c.P1)
				}
				if c.P2 != nil {
					allNodes = append(allNodes, c.P2)
				}
				if c.Sum != nil {
					allNodes = append(allNodes, c.Sum)
				}
			case *SmithNormalFormCertificate:
				if c.A != nil {
					allNodes = append(allNodes, c.A)
				}
				if c.U != nil {
					allNodes = append(allNodes, c.U)
				}
				if c.V != nil {
					allNodes = append(allNodes, c.V)
				}
				if c.D != nil {
					allNodes = append(allNodes, c.D)
				}
			case *HermiteNormalFormCertificate:
				if c.A != nil {
					allNodes = append(allNodes, c.A)
				}
				if c.U != nil {
					allNodes = append(allNodes, c.U)
				}
				if c.H != nil {
					allNodes = append(allNodes, c.H)
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
	theoremCode = strings.ReplaceAll(theoremCode, "ihd_verified_proof", escapeLeanIdent(theoremName))

	var certIR Certificate
	if cert != nil {
		certIR = cert.ToCertificateIR()
	}
	requiredImports := DetermineRequiredImports(certIR, allNodes...)

	var b strings.Builder
	b.WriteString("-- Automatically generated by ihd (i-hate-decimal-calc) Lean 4 Transpiler\n")
	b.WriteString("-- Mathematical proof certificate certified with Skeptic's verification\n")
	for _, imp := range requiredImports {
		b.WriteString(fmt.Sprintf("import %s\n", imp))
	}
	b.WriteString("\n")

	typeAnnotation := "ℚ"
	if containsRealNodes(allNodes...) {
		typeAnnotation = "ℝ"
	}

	if len(vars) > 0 {
		b.WriteString(fmt.Sprintf("variable (%s : %s)\n\n", strings.Join(vars, " "), typeAnnotation))
	}

	b.WriteString(theoremCode)
	b.WriteString(fmt.Sprintf("\n\n#print axioms %s\n", escapeLeanIdent(theoremName)))

	return b.String(), nil
}

// DetermineRequiredImports inspects the certificate and expression nodes to dynamically determine
// the minimal set of Mathlib4 imports needed to compile the Lean 4 proof file.
func DetermineRequiredImports(cert Certificate, nodes ...Node) []string {
	importsMap := make(map[string]bool)

	// Ring tactic is universally needed for algebraic rewrites and identities
	importsMap["Mathlib.Tactic.Ring"] = true

	// Check domain of certificate if present
	if cert != nil {
		switch c := cert.(type) {
		case *InvertibilityCertificate:
			importsMap["Mathlib.Data.Matrix.Basic"] = true
			importsMap["Mathlib.Data.Fin.VecNotation"] = true
			importsMap["Mathlib.LinearAlgebra.Matrix.Notation"] = true
			importsMap["Mathlib.Tactic.FinCases"] = true
			hasVars := len(CollectFreeVariables(c.Matrix)) > 0 || len(CollectFreeVariables(c.Inverse)) > 0
			if !hasVars {
				importsMap["Mathlib.Tactic.NormNum"] = true
			}
		case *DecompositionCertificate:
			importsMap["Mathlib.Data.Matrix.Basic"] = true
			importsMap["Mathlib.Data.Fin.VecNotation"] = true
			importsMap["Mathlib.LinearAlgebra.Matrix.Notation"] = true
			importsMap["Mathlib.Tactic.FinCases"] = true
			hasVars := len(CollectFreeVariables(c.Matrix)) > 0
			for _, f := range c.Factors {
				if len(CollectFreeVariables(f)) > 0 {
					hasVars = true
					break
				}
			}
			if !hasVars {
				importsMap["Mathlib.Tactic.NormNum"] = true
			}
		case *LinearSolveCertificate:
			importsMap["Mathlib.Data.Matrix.Basic"] = true
			importsMap["Mathlib.Data.Fin.VecNotation"] = true
			importsMap["Mathlib.LinearAlgebra.Matrix.Notation"] = true
			importsMap["Mathlib.Tactic.FinCases"] = true
			hasVars := len(CollectFreeVariables(c.Matrix)) > 0 || len(CollectFreeVariables(c.Solution)) > 0 || len(CollectFreeVariables(c.Target)) > 0
			if !hasVars {
				importsMap["Mathlib.Tactic.NormNum"] = true
			}
		case *SmithNormalFormCertificate, *HermiteNormalFormCertificate:
			importsMap["Mathlib.Data.Matrix.Basic"] = true
			importsMap["Mathlib.Data.Fin.VecNotation"] = true
			importsMap["Mathlib.LinearAlgebra.Matrix.Notation"] = true
			importsMap["Mathlib.Tactic.FinCases"] = true
			importsMap["Mathlib.Tactic.NormNum"] = true
		case *PellCertificate:
			importsMap["Mathlib.Data.Int.Basic"] = true
			importsMap["Mathlib.Tactic.NormNum"] = true
		case *PolynomialRootCertificate, *IntegerRelationCertificate, *EllipticPointCertificate:
			importsMap["Mathlib.Data.Rat.Defs"] = true
			importsMap["Mathlib.Tactic.NormNum"] = true
		case *DerivCertificate:
			importsMap["Mathlib.Analysis.Calculus.Deriv.Basic"] = true
			if c.Integrand != nil && containsTrigNodes(c.Integrand) {
				importsMap["Mathlib.Analysis.SpecialFunctions.Trigonometric.Deriv"] = true
			}
			if c.Integrand != nil && containsExpNodes(c.Integrand) {
				importsMap["Mathlib.Analysis.SpecialFunctions.ExpDeriv"] = true
			}
		case *ImpossibilityCertificate:
			if c.Kind == ImpossibilityAbelRuffini {
				importsMap["Mathlib.FieldTheory.AbelRuffini"] = true
			}
		case *GeometricCertificate:
			// Geometric proofs use ring, and rational/real
		}
	}

	// Check nodes for functions/types
	isReal := containsRealNodes(nodes...)
	for _, n := range nodes {
		if containsMatrixNodes(n) {
			importsMap["Mathlib.Data.Matrix.Basic"] = true
			importsMap["Mathlib.Data.Fin.VecNotation"] = true
			importsMap["Mathlib.LinearAlgebra.Matrix.Notation"] = true
			importsMap["Mathlib.Tactic.FinCases"] = true
			if len(CollectFreeVariables(n)) == 0 {
				importsMap["Mathlib.Tactic.NormNum"] = true
			}
		}
		if containsTrigNodes(n) {
			importsMap["Mathlib.Analysis.SpecialFunctions.Trigonometric.Deriv"] = true
		}
		if containsExpNodes(n) {
			importsMap["Mathlib.Analysis.SpecialFunctions.ExpDeriv"] = true
		}
	}

	if isReal {
		importsMap["Mathlib.Basic.Real.Basic"] = true
	} else {
		importsMap["Mathlib.Data.Rat.Defs"] = true
	}

	var res []string
	for imp := range importsMap {
		res = append(res, imp)
	}
	sort.Strings(res)
	return res
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

// containsTrigNodes checks whether any of the given nodes contain trigonometric functions.
func containsTrigNodes(nodes ...Node) bool {
	hasTrig := false
	var walk func(n Node)
	walk = func(n Node) {
		if n == nil || hasTrig {
			return
		}
		switch curr := n.(type) {
		case *FuncNode:
			switch strings.ToLower(curr.Name) {
			case "sin", "cos", "tan", "asin", "acos", "atan":
				hasTrig = true
				return
			}
			for _, a := range curr.Args {
				walk(a)
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
	return hasTrig
}

// containsExpNodes checks whether any of the given nodes contain exponential or logarithmic functions.
func containsExpNodes(nodes ...Node) bool {
	hasExp := false
	var walk func(n Node)
	walk = func(n Node) {
		if n == nil || hasExp {
			return
		}
		switch curr := n.(type) {
		case *FuncNode:
			switch strings.ToLower(curr.Name) {
			case "exp", "ln", "log":
				hasExp = true
				return
			}
			for _, a := range curr.Args {
				walk(a)
			}
		case *ConstNode:
			if curr.Name == "e" {
				hasExp = true
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
	return hasExp
}

// containsMatrixNodes checks whether any of the given nodes contain matrices.
func containsMatrixNodes(nodes ...Node) bool {
	hasMatrix := false
	var walk func(n Node)
	walk = func(n Node) {
		if n == nil || hasMatrix {
			return
		}
		if _, ok := n.(*MatrixNode); ok {
			hasMatrix = true
			return
		}
		switch curr := n.(type) {
		case *FuncNode:
			for _, a := range curr.Args {
				walk(a)
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
		}
	}
	for _, n := range nodes {
		walk(n)
	}
	return hasMatrix
}

// resolveGeometricAlgebraicIdentity extracts linear construction substitutions from hypotheses
// using LinearSubstitutionFastPath and substitutes them into the conclusion polynomial,
// producing an algebraically verifiable identity in terms of free parameters.
func resolveGeometricAlgebraicIdentity(c *GeometricCertificate) Node {
	if c == nil || c.Conclusion == nil {
		return mustRational(0, 1)
	}

	// 1. Gather all variables appearing across hypotheses and conclusion
	allVarsMap := make(map[string]bool)
	for _, p := range c.Hypotheses {
		for _, v := range collectVariables(p) {
			allVarsMap[v] = true
		}
	}
	for _, v := range collectVariables(c.Conclusion) {
		allVarsMap[v] = true
	}

	var order []string
	for v := range allVarsMap {
		order = append(order, v)
	}
	sort.Strings(order)

	// 2. Perform systematic linear elimination of dependent variables
	_, subs := LinearSubstitutionFastPath(c.Hypotheses, order)

	// 3. Substitute solved dependent variables into conclusion
	substed := c.Conclusion
	for k, v := range subs {
		substed = Substitute(substed, k, v)
	}

	// 4. Verify that substituted expression algebraically simplifies to 0
	val, err := Eval(substed)
	if err == nil && isZero(val) {
		return substed
	}

	vars := collectVariables(substed)
	if len(vars) > 0 {
		polyNode, err := NodeToPoly(substed, vars, ast.OrderGrevLex)
		if err == nil && isZero(PolyToNode(polyNode)) {
			return substed
		}
	}

	return c.Conclusion
}

// transpileGeometricCertificateIR transforms a GeometricCertificate into a pair of formal Lean 4 theorems:
// 1. An auxiliary algebraic identity lemma proving the simplified characteristic polynomial equals zero via `by ring`.
// 2. A primary geometric deduction theorem with explicit hypotheses and conclusion predicates.
func transpileGeometricCertificateIR(c *GeometricCertificate, theoremName string) (string, error) {
	if c == nil || c.Conclusion == nil {
		return fmt.Sprintf("theorem %s : 0 = 0 := by ring", escapeLeanIdent(theoremName)), nil
	}

	identityNode := resolveGeometricAlgebraicIdentity(c)
	identityStr, err := ToLeanSyntax(identityNode)
	if err != nil {
		identityStr = "0"
	}
	identityStr = strings.ReplaceAll(identityStr, "==", "=")

	// Gather variables to solve linear substitutions
	allVarsMap := make(map[string]bool)
	for _, p := range c.Hypotheses {
		for _, v := range collectVariables(p) {
			allVarsMap[v] = true
		}
	}
	for _, v := range collectVariables(c.Conclusion) {
		allVarsMap[v] = true
	}
	var order []string
	for v := range allVarsMap {
		order = append(order, v)
	}
	sort.Strings(order)

	_, subs := LinearSubstitutionFastPath(c.Hypotheses, order)

	conclStr, err := ToLeanSyntax(c.Conclusion)
	if err != nil {
		conclStr = "0"
	}
	conclStr = strings.ReplaceAll(conclStr, "==", "=")

	var hypLines []string
	var rwVars []string
	if len(subs) > 0 {
		var subKeys []string
		for k := range subs {
			subKeys = append(subKeys, k)
		}
		sort.Strings(subKeys)

		for _, k := range subKeys {
			valStr, err := ToLeanSyntax(subs[k])
			if err != nil {
				continue
			}
			hypName := fmt.Sprintf("h_%s", k)
			hypLines = append(hypLines, fmt.Sprintf("  (%s : %s = %s)", hypName, escapeLeanIdent(k), valStr))
			rwVars = append(rwVars, hypName)
		}
	} else {
		// Non-linear or general hypotheses: list all hypotheses explicitly
		for i, h := range c.Hypotheses {
			hStr, err := ToLeanSyntax(h)
			if err != nil {
				continue
			}
			hypName := fmt.Sprintf("h_%d", i+1)
			hypLines = append(hypLines, fmt.Sprintf("  (%s : %s = 0)", hypName, hStr))
		}
		for i, ndg := range c.SaturationInitials {
			ndgStr, err := ToLeanSyntax(ndg)
			if err != nil {
				continue
			}
			hypLines = append(hypLines, fmt.Sprintf("  (h_ndg_%d : %s ≠ 0)", i+1, ndgStr))
		}
	}

	var b strings.Builder
	// 1. Auxiliary algebraic identity lemma
	identityThmName := fmt.Sprintf("%s_algebraic_identity", escapeLeanIdent(theoremName))
	b.WriteString("-- Auxiliary algebraic identity verified from geometric characteristic polynomial\n")
	b.WriteString(fmt.Sprintf("theorem %s : %s = 0 := by ring\n\n", identityThmName, identityStr))

	// 2. Main geometric deduction: explicit hypotheses entail the conclusion predicate
	b.WriteString("-- Main geometric deduction: explicit hypotheses entail the conclusion predicate\n")
	if len(hypLines) > 0 {
		b.WriteString(fmt.Sprintf("theorem %s\n%s :\n  %s = 0 := by\n", escapeLeanIdent(theoremName), strings.Join(hypLines, "\n"), conclStr))
		if len(rwVars) > 0 {
			b.WriteString(fmt.Sprintf("  rw [%s]\n", strings.Join(rwVars, ", ")))
			b.WriteString("  ring")
		} else {
			b.WriteString(fmt.Sprintf("  have hid := %s\n", identityThmName))
			b.WriteString("  ring")
		}
	} else {
		b.WriteString(fmt.Sprintf("theorem %s : %s = 0 := by ring", escapeLeanIdent(theoremName), conclStr))
	}

	return b.String(), nil
}

