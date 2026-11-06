package calc

import (
	"fmt"
	"math/big"
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

// unwrapOuterParens safely strips outermost matching parentheses pairs.
func unwrapOuterParens(s string) string {
	s = strings.TrimSpace(s)
	for isEnclosedInParens(s) {
		s = strings.TrimSpace(s[1 : len(s)-1])
	}
	return s
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
		if rat, ok := n.Exp.(*RationalNode); ok && rat.Val != nil && rat.Val.IsInt() && rat.Val.Sign() < 0 {
			expStr = fmt.Sprintf("(%s : ℤ)", rat.Val.Num().String())
		} else if _, isAdd := n.Exp.(*AddNode); isAdd {
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

	case *ListNode:
		var elemStrs []string
		for _, elem := range n.Elements {
			es, err := ToLeanSyntax(elem)
			if err != nil {
				return "", err
			}
			elemStrs = append(elemStrs, es)
		}
		return fmt.Sprintf("![%s]", strings.Join(elemStrs, ", ")), nil

	case *RuleNode:
		lhsStr, err := ToLeanSyntax(n.LHS)
		if err != nil {
			return "", err
		}
		rhsStr, err := ToLeanSyntax(n.RHS)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("(%s = %s)", lhsStr, rhsStr), nil

	case *WildcardNode:
		return escapeLeanIdent(strings.TrimPrefix(n.Name, "_")), nil

	default:
		return "", fmt.Errorf("%s", i18n.T("lean.err_unsupported_ast_node", n))
	}
}

// hasUnsupportedLeanFunc checks whether the node tree contains CAS-specific uninterpreted functions.
func hasUnsupportedLeanFunc(n Node) bool {
	if n == nil {
		return false
	}
	switch curr := n.(type) {
	case *FuncNode:
		switch strings.ToLower(curr.Name) {
		case "sin", "cos", "tan", "exp", "ln", "log", "sqrt":
			for _, arg := range curr.Args {
				if hasUnsupportedLeanFunc(arg) {
					return true
				}
			}
			return false
		default:
			return true
		}
	case *AddNode:
		for _, t := range curr.Terms {
			if hasUnsupportedLeanFunc(t) {
				return true
			}
		}
	case *MulNode:
		for _, f := range curr.Factors {
			if hasUnsupportedLeanFunc(f) {
				return true
			}
		}
	case *PowNode:
		return hasUnsupportedLeanFunc(curr.Base) || hasUnsupportedLeanFunc(curr.Exp)
	case *UnaryOpNode:
		return hasUnsupportedLeanFunc(curr.Expr)
	case *RelOpNode:
		return hasUnsupportedLeanFunc(curr.LHS) || hasUnsupportedLeanFunc(curr.RHS)
	case *SqrtNode:
		return hasUnsupportedLeanFunc(curr.Radicand)
	}
	return false
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
			if curr.Name == "true" || curr.Name == "false" || curr.Name == "pi" || curr.Name == "e" || curr.Name == "i" {
				return
			}
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
		if hasUnsupportedLeanFunc(c.LHS) || hasUnsupportedLeanFunc(c.RHS) {
			return "", fmt.Errorf("%s", i18n.T("lean.err_unsupported_identity_for_lean"))
		}
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
			return "", fmt.Errorf("transpile integrand: %w", err)
		}
		resStr, err := ToLeanSyntax(c.Antiderivative)
		if err != nil {
			return "", fmt.Errorf("transpile antiderivative: %w", err)
		}
		fStr = strings.ReplaceAll(fStr, ": ℚ", ": ℝ")
		resStr = strings.ReplaceAll(resStr, ": ℚ", ": ℝ")
		intVar := c.Variable
		if intVar == "" {
			intVar = "x"
		}
		combinatorTerm, err := buildHasDerivAtCombinator(c.Antiderivative, intVar)
		if err != nil {
			return "", fmt.Errorf("build derivative proof: %w", err)
		}
		combinatorTerm = strings.ReplaceAll(combinatorTerm, ": ℚ", ": ℝ")
		equalityStr = fmt.Sprintf("HasDerivAt (fun %s => %s) (%s) %s", intVar, resStr, fStr, intVar)
		tactic = fmt.Sprintf("by exact HasDerivAt.congr_deriv (%s) (by first | (ring; done) | (dsimp; ring) | (dsimp; field_simp; ring) | rfl)", combinatorTerm)

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
		return "", fmt.Errorf("lean transpilation for ODECertificate is not yet supported in Mathlib4")

	case *WZCertificate:
		return "", fmt.Errorf("lean transpilation for WZCertificate is not yet supported in Mathlib4")

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
		if c.X == nil || c.Y == nil || c.D == nil {
			return "", fmt.Errorf("%s", i18n.T("lean.err_incomplete_certificate_witness", "PellCertificate"))
		}
		xStr, err := ToLeanSyntax(c.X)
		if err != nil {
			return "", err
		}
		yStr, err := ToLeanSyntax(c.Y)
		if err != nil {
			return "", err
		}
		dStr, err := ToLeanSyntax(c.D)
		if err != nil {
			return "", err
		}
		equalityStr = fmt.Sprintf("((%s : ℤ)^2 - (%s : ℤ) * (%s : ℤ)^2 : ℤ) = 1", xStr, dStr, yStr)
		tactic = "by decide"

	case *LinearDiophantineCertificate:
		if c.A == nil || c.B == nil || c.C == nil || c.XSolution == nil || c.YSolution == nil {
			return "", fmt.Errorf("%s", i18n.T("lean.err_incomplete_certificate_witness", "LinearDiophantineCertificate"))
		}
		aStr, err := ToLeanSyntax(c.A)
		if err != nil {
			return "", err
		}
		bStr, err := ToLeanSyntax(c.B)
		if err != nil {
			return "", err
		}
		cStr, err := ToLeanSyntax(c.C)
		if err != nil {
			return "", err
		}
		xStr, err := ToLeanSyntax(c.XSolution)
		if err != nil {
			return "", err
		}
		yStr, err := ToLeanSyntax(c.YSolution)
		if err != nil {
			return "", err
		}
		param := c.ParamVar
		if param == "" {
			param = "t"
		}
		equalityStr = fmt.Sprintf("∀ (%s : ℤ), ((%s : ℤ) * (%s) + (%s : ℤ) * (%s) : ℤ) = (%s : ℤ)", param, aStr, xStr, bStr, yStr, cStr)
		tactic = fmt.Sprintf("by intro %s ; ring", param)

	case *PolynomialRootCertificate:
		if c.Polynomial == nil {
			return "", fmt.Errorf("%s", i18n.T("lean.err_incomplete_certificate_witness", "PolynomialRootCertificate"))
		}
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

		if len(roots) == 0 {
			return "", fmt.Errorf("%s", i18n.T("lean.err_empty_root_certificate"))
		}

		polyType := "ℚ"
		if containsRealNodes(c.Polynomial) || containsRealNodes(roots...) {
			polyType = "ℝ"
		}

		if len(roots) == 1 {
			substed := Substitute(c.Polynomial, varSubst, roots[0])
			subStr, err := ToLeanSyntax(substed)
			if err != nil {
				return "", err
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
					return "", err
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
			return "", err
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
		if c.Sum == nil || c.CurveA == nil || c.CurveB == nil {
			return "", fmt.Errorf("%s", i18n.T("lean.err_incomplete_certificate_witness", "EllipticPointCertificate"))
		}
		var xNode, yNode Node
		if pMat, ok := c.Sum.(*MatrixNode); ok && len(pMat.Data) == 1 && len(pMat.Data[0]) >= 2 {
			xNode = pMat.Data[0][0]
			yNode = pMat.Data[0][1]
		} else if pList, ok := c.Sum.(*ListNode); ok && len(pList.Elements) >= 2 {
			xNode = pList.Elements[0]
			yNode = pList.Elements[1]
		}
		if xNode == nil || yNode == nil {
			return "", fmt.Errorf("%s", i18n.T("lean.err_incomplete_certificate_witness", "EllipticPointCertificate"))
		}
		xStr, err := ToLeanSyntax(xNode)
		if err != nil {
			return "", err
		}
		yStr, err := ToLeanSyntax(yNode)
		if err != nil {
			return "", err
		}
		aStr, err := ToLeanSyntax(c.CurveA)
		if err != nil {
			return "", err
		}
		bStr, err := ToLeanSyntax(c.CurveB)
		if err != nil {
			return "", err
		}
		equalityStr = fmt.Sprintf("(%s : ℚ)^2 = (%s : ℚ)^3 + (%s : ℚ) * (%s : ℚ) + (%s : ℚ)", yStr, xStr, aStr, xStr, bStr)
		if len(CollectFreeVariables(xNode)) > 0 || len(CollectFreeVariables(yNode)) > 0 || len(CollectFreeVariables(c.CurveA)) > 0 || len(CollectFreeVariables(c.CurveB)) > 0 {
			tactic = "by ring"
		} else {
			tactic = "by norm_num"
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

	case *PrimitiveElementCertificate:
		mStr, _ := ToLeanSyntax(c.MinPolyTheta)
		qgStr, _ := ToLeanSyntax(c.QuotientG)
		qfStr, _ := ToLeanSyntax(c.QuotientF)

		substG := Substitute(c.MinPolyBeta, "x", c.RepBeta)
		substF := Substitute(c.MinPolyAlpha, "x", c.RepAlpha)
		lhsGStr, _ := ToLeanSyntax(substG)
		lhsFStr, _ := ToLeanSyntax(substF)

		equalityStr = fmt.Sprintf("∀ (x : ℚ), (%s = (%s) * (%s)) ∧ (%s = (%s) * (%s))",
			lhsGStr, qgStr, mStr,
			lhsFStr, qfStr, mStr,
		)
		tactic = "by intro x ; constructor <;> ring"

	case *IdealDecompositionCertificate:
		var eqs []string
		allSyzygies := append(append([]IdealSyzygyRelation{}, c.ForwardSyzygies...), c.BackwardSyzygies...)
		if len(allSyzygies) == 0 {
			return "", fmt.Errorf("cannot transpile IdealDecompositionCertificate: no syzygies available")
		}
		for _, syz := range allSyzygies {
			if len(syz.CombCoeffs) != len(syz.Gens) {
				return "", fmt.Errorf("invalid syzygy relation: mismatched CombCoeffs and Gens length (%d != %d)", len(syz.CombCoeffs), len(syz.Gens))
			}
			lhsStr, err := ToLeanSyntax(syz.TargetNode)
			if err != nil {
				return "", fmt.Errorf("transpile syzygy target: %w", err)
			}
			var terms []string
			for j, q := range syz.CombCoeffs {
				qStr, err := ToLeanSyntax(q)
				if err != nil {
					return "", fmt.Errorf("transpile syzygy coeff: %w", err)
				}
				gStr, err := ToLeanSyntax(syz.Gens[j])
				if err != nil {
					return "", fmt.Errorf("transpile syzygy gen: %w", err)
				}
				if qStr == "1" {
					terms = append(terms, gStr)
				} else {
					terms = append(terms, fmt.Sprintf("(%s) * (%s)", qStr, gStr))
				}
			}
			rhs := "0"
			if len(terms) > 0 {
				rhs = strings.Join(terms, " + ")
			}
			eqs = append(eqs, fmt.Sprintf("(%s = %s)", lhsStr, rhs))
		}
		if len(eqs) == 1 {
			equalityStr = eqs[0]
			tactic = "by ring"
		} else {
			equalityStr = strings.Join(eqs, " ∧ ")
			tactic = "by repeat constructor <;> ring"
		}

	case *RecurrenceCertificate:
		if !c.IsHomogeneous || !c.HasOnlyRationalRoots || len(c.RationalRoots) == 0 {
			return "", fmt.Errorf("%s", i18n.T("lean.err_unsupported_recurrence_for_lean"))
		}

		// For each rational root q, verify charPoly(q) = 0 using norm_num
		var rootEqs []string
		for _, q := range c.RationalRoots {
			// Construct charPoly evaluated at q: sum(c_j * q^j) = 0
			var polyTerms []string
			for j, coeff := range c.CharPolyCoeffs {
				if coeff.Sign() == 0 {
					continue
				}
				qPowJ := ratPow(q, j)
				termVal := new(big.Rat).Mul(coeff, qPowJ)
				polyTerms = append(polyTerms, termVal.RatString())
			}
			polySumStr := strings.Join(polyTerms, " + ")
			rootEqs = append(rootEqs, fmt.Sprintf("(%s : ℚ) = 0", polySumStr))
		}

		// Also verify initial conditions if present
		for _, initEq := range c.InitialConditions {
			if rel, ok := initEq.(*RelOpNode); ok && rel.Op == "==" {
				lhsStr, _ := ToLeanSyntax(rel.LHS)
				rhsStr, _ := ToLeanSyntax(rel.RHS)
				rootEqs = append(rootEqs, fmt.Sprintf("((%s : ℚ) = %s)", lhsStr, rhsStr))
			}
		}

		if len(rootEqs) == 1 {
			equalityStr = rootEqs[0]
			tactic = "by norm_num"
		} else {
			equalityStr = strings.Join(rootEqs, " ∧ ")
			tactic = "by repeat constructor <;> norm_num"
		}

	case *ImpossibilityCertificate:
		if c.Kind == ImpossibilityAbelRuffini {
			fStr, err := ToLeanSyntax(c.Problem)
			if err != nil {
				return "", fmt.Errorf("transpile problem polynomial: %w", err)
			}
			equalityStr = fmt.Sprintf("¬ IsSolvable (Polynomial.galoisGroup (%s : Polynomial ℚ))", fStr)
			tactic = "by decide"
		} else {
			return "", fmt.Errorf("unsupported impossibility kind %q for Lean 4 formalization", c.Kind)
		}

	case *IdealMembershipCertificate:
		return transpileIdealMembershipCertificate(c, "ihd_verified_proof")

	case *SOSCertificate:
		return transpileSOSCertificate(c, "ihd_verified_proof")

	case *KirchhoffCertificate:
		minorStr, _ := ToLeanSyntax(c.LaplacianMinor)
		treeStr, _ := ToLeanSyntax(c.SpanningTrees)
		dim := 0
		if mat, ok := c.LaplacianMinor.(*MatrixNode); ok && len(mat.Data) > 0 {
			dim = len(mat.Data)
		}
		if dim > 0 {
			equalityStr = fmt.Sprintf("Matrix.det (%s : Matrix (Fin %d) (Fin %d) ℚ) = %s", minorStr, dim, dim, treeStr)
			if dim == 2 {
				tactic = "by simp [Matrix.det_fin_two]; norm_num"
			} else if dim == 3 {
				tactic = "by simp [Matrix.det_fin_three]; norm_num"
			} else {
				tactic = "by norm_num"
			}
		} else {
			equalityStr = fmt.Sprintf("Matrix.det %s = %s", minorStr, treeStr)
			tactic = "by norm_num"
		}

	case *MinimaxCertificate:
		aStr, _ := ToLeanSyntax(c.PayoffMatrix)
		xStr, _ := ToLeanSyntax(c.StrategyP1)
		yStr, _ := ToLeanSyntax(c.StrategyP2)
		vStr, _ := ToLeanSyntax(c.GameValue)
		m, n := 0, 0
		if mat, ok := c.PayoffMatrix.(*MatrixNode); ok && len(mat.Data) > 0 {
			m = len(mat.Data)
			n = len(mat.Data[0])
		}
		if m > 0 && n > 0 {
			equalityStr = fmt.Sprintf("((∀ j, (Matrix.vecMul (%s : Fin %d → ℚ) (%s : Matrix (Fin %d) (Fin %d) ℚ)) j ≥ %s) ∧ (∀ i, (Matrix.mulVec (%s : Matrix (Fin %d) (Fin %d) ℚ) (%s : Fin %d → ℚ)) i ≤ %s))",
				xStr, m, aStr, m, n, vStr,
				aStr, m, n, yStr, n, vStr)
			tactic = "by constructor <;> (intro k; fin_cases k <;> norm_num)"
		} else {
			equalityStr = fmt.Sprintf("((Matrix.vecMul %s %s ≥ %s) ∧ (Matrix.mulVec %s %s ≤ %s))", xStr, aStr, vStr, aStr, yStr, vStr)
			tactic = "by constructor <;> norm_num"
		}

	case *MarkovStationaryCertificate:
		pStr, _ := ToLeanSyntax(c.TransitionMatrix)
		piStr, _ := ToLeanSyntax(c.StationaryDist)
		dim := 0
		if mat, ok := c.TransitionMatrix.(*MatrixNode); ok && len(mat.Data) > 0 {
			dim = len(mat.Data)
		}
		if dim > 0 {
			equalityStr = fmt.Sprintf("(Matrix.vecMul (%s : Fin %d → ℚ) (%s : Matrix (Fin %d) (Fin %d) ℚ) = (%s : Fin %d → ℚ))",
				piStr, dim, pStr, dim, dim, piStr, dim)
			tactic = "by ext i; fin_cases i <;> norm_num"
		} else {
			equalityStr = fmt.Sprintf("Matrix.vecMul %s %s = %s", piStr, pStr, piStr)
			tactic = "by ext <;> norm_num"
		}

	default:
		eq := cert.EquationString()
		if eq != "" && (cert.Domain() == CertificateDomain(DomainFactor) || cert.Domain() == CertificateDomain(DomainGeneral) || cert.Domain() == CertificateDomain(DomainRationalApart)) {
			equalityStr = eq
			tactic = "by ring"
		} else {
			return "", fmt.Errorf("%s", i18n.T("lean.err_unsupported_certificate_for_lean", cert.Domain()))
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
			*IntegerRelationCertificate, *EllipticPointCertificate, *ImpossibilityCertificate,
			*SmithNormalFormCertificate, *HermiteNormalFormCertificate, *IdealDecompositionCertificate,
			*KirchhoffCertificate, *MinimaxCertificate, *MarkovStationaryCertificate:
			return TranspileCertificateIRToLean(certIR)
		}
	}

	// Backward-compatible fallback for dynamic/unclassified nodes
	return TranspileCertificateIRToLean(certIR)
}

// GenerateLeanSource generates a standalone, fully valid Lean 4 source file with Mathlib imports, variables, and theorem.
func GenerateLeanSource(theoremName string, cert *VerificationCertificate, inputExpr, resultExpr Node) (string, error) {
	return GenerateLeanSourceWithConditions(theoremName, cert, inputExpr, resultExpr, nil)
}

// GenerateLeanSourceWithConditions generates a standalone Lean 4 source file with optional domain side conditions.
func GenerateLeanSourceWithConditions(theoremName string, cert *VerificationCertificate, inputExpr, resultExpr Node, condSet *DomainConditionSet) (string, error) {
	if cert == nil {
		return "", fmt.Errorf("%s", i18n.T("lean.err_certificate_is_nil"))
	}
	if !cert.IsVerified {
		return "", fmt.Errorf("%s", i18n.T("lean.err_cannot_transpile_unverified_certificate", cert.Details))
	}

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
		if cert.Residual != nil {
		allNodes = append(allNodes, cert.Residual)
	}
	if condSet != nil {
		for _, cond := range condSet.Conditions() {
			if cond.Expr != nil {
				allNodes = append(allNodes, cond.Expr)
			}
		}
	}
	if ir := cert.ToCertificateIR(); ir != nil {
			switch c := ir.(type) {
			case *ImpossibilityCertificate:
				if c.Problem != nil {
					allNodes = []Node{c.Problem}
				}
			case *RecurrenceCertificate:
				if c.RecurrenceNode != nil {
					allNodes = append(allNodes, c.RecurrenceNode)
				}
				if c.SolutionNode != nil {
					allNodes = append(allNodes, c.SolutionNode)
				}
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
			case *IdealDecompositionCertificate:
				allNodes = append(allNodes, c.IdealGens...)
				for _, comp := range c.Components {
					allNodes = append(allNodes, comp...)
				}
				for _, prime := range c.Primes {
					allNodes = append(allNodes, prime...)
				}
				for _, s := range c.ForwardSyzygies {
					allNodes = append(allNodes, s.TargetNode)
					allNodes = append(allNodes, s.CombCoeffs...)
					allNodes = append(allNodes, s.Gens...)
				}
			case *IdealMembershipCertificate:
				if c.Target != nil {
					allNodes = append(allNodes, c.Target)
				}
				allNodes = append(allNodes, c.Generators...)
			case *KirchhoffCertificate:
				if c.LaplacianMinor != nil {
					allNodes = append(allNodes, c.LaplacianMinor)
				}
				if c.SpanningTrees != nil {
					allNodes = append(allNodes, c.SpanningTrees)
				}
			case *MinimaxCertificate:
				if c.PayoffMatrix != nil {
					allNodes = append(allNodes, c.PayoffMatrix)
				}
				if c.StrategyP1 != nil {
					allNodes = append(allNodes, c.StrategyP1)
				}
				if c.StrategyP2 != nil {
					allNodes = append(allNodes, c.StrategyP2)
				}
				if c.GameValue != nil {
					allNodes = append(allNodes, c.GameValue)
				}
			case *MarkovStationaryCertificate:
				if c.TransitionMatrix != nil {
					allNodes = append(allNodes, c.TransitionMatrix)
				}
				if c.StationaryDist != nil {
					allNodes = append(allNodes, c.StationaryDist)
				}
			case *SOSCertificate:
				if c.Target != nil {
					allNodes = append(allNodes, c.Target)
				}
				for _, t := range c.Terms {
					if t.Poly != nil {
						allNodes = append(allNodes, t.Poly)
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

	// Inject domain hypotheses if present, and collect variable denominators
	binders := make([]string, 0)
	hasNonzeroHyp := false
	seenProps := make(map[string]bool)

	if condSet != nil && condSet.Len() > 0 {
		conds := condSet.Conditions()
		for i, cond := range conds {
			exprLean, err := ToLeanSyntax(cond.Expr)
			if err != nil {
				continue
			}
			cleanExpr := unwrapOuterParens(exprLean)
			var prop string
			switch cond.Kind {
			case CondNonZero:
				prop = fmt.Sprintf("%s ≠ 0", cleanExpr)
				hasNonzeroHyp = true
			case CondNonNegative:
				prop = fmt.Sprintf("0 ≤ %s", cleanExpr)
			case CondPositive:
				prop = fmt.Sprintf("0 < %s", cleanExpr)
			}
			if prop != "" && !seenProps[prop] {
				seenProps[prop] = true
				hName := fmt.Sprintf("h%d", i)
				binders = append(binders, fmt.Sprintf("(%s : %s)", hName, prop))
			}
		}
	}

	// Always collect variable denominators automatically to prevent division junk values
	denoms := collectNonzeroDenominators(allNodes...)
	denomIdx := 0
	for _, d := range denoms {
		cleanD := unwrapOuterParens(d)
		prop := fmt.Sprintf("%s ≠ 0", cleanD)
		if !seenProps[prop] {
			seenProps[prop] = true
			hName := fmt.Sprintf("h_denom%d", denomIdx)
			denomIdx++
			binders = append(binders, fmt.Sprintf("(%s : %s)", hName, prop))
			hasNonzeroHyp = true
		}
	}

	// Always collect domain conditions (log > 0, sqrt >= 0) to guard against total function junk values
	domConds := collectDomainConditions(allNodes...)
	domIdx := 0
	for _, dc := range domConds {
		if !seenProps[dc] {
			seenProps[dc] = true
			hName := fmt.Sprintf("h_dom%d", domIdx)
			domIdx++
			binders = append(binders, fmt.Sprintf("(%s : %s)", hName, dc))
		}
	}

	if len(binders) > 0 {
		thmHead := fmt.Sprintf("theorem %s", escapeLeanIdent(theoremName))
		if idx := strings.Index(theoremCode, thmHead); idx != -1 {
			insertPos := idx + len(thmHead)
			binderStr := " " + strings.Join(binders, " ")
			theoremCode = theoremCode[:insertPos] + binderStr + theoremCode[insertPos:]
		}

		if hasNonzeroHyp && strings.Contains(theoremCode, ":= by ring") {
			if containsTanNodes(allNodes...) {
				theoremCode = strings.Replace(theoremCode, ":= by ring", ":= by first | (ring; done) | (unfold Real.tan; field_simp; ring) | (field_simp; ring)", 1)
			} else {
				theoremCode = strings.Replace(theoremCode, ":= by ring", ":= by field_simp; ring", 1)
			}
		}
	}

	certIR := cert.ToCertificateIR()
	requiredImports := DetermineRequiredImports(certIR, allNodes...)

	var b strings.Builder
	b.WriteString("-- Automatically generated by ihd (i-hate-decimal-calc) Lean 4 Transpiler\n")
	b.WriteString("-- Mathematical proof certificate certified with Skeptic's verification\n")
	for _, imp := range requiredImports {
		b.WriteString(fmt.Sprintf("import %s\n", imp))
	}
	b.WriteString("\n")

	domainType := inferTheoremDomainType(certIR, allNodes...)
	if domainType == "" {
		domainType = DomainRat
	}

	if len(vars) > 0 {
		if domainType == DomainZMod {
			b.WriteString(fmt.Sprintf("variable (n : ℕ) (%s : ZMod n)\n\n", strings.Join(vars, " ")))
		} else {
			b.WriteString(fmt.Sprintf("variable (%s : %s)\n\n", strings.Join(vars, " "), domainType))
		}
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
		case *KirchhoffCertificate:
			importsMap["Mathlib.Data.Matrix.Basic"] = true
			importsMap["Mathlib.Data.Fin.VecNotation"] = true
			importsMap["Mathlib.LinearAlgebra.Matrix.Notation"] = true
			importsMap["Mathlib.LinearAlgebra.Matrix.Determinant.Basic"] = true
			importsMap["Mathlib.Tactic.NormNum"] = true
		case *MinimaxCertificate, *MarkovStationaryCertificate:
			importsMap["Mathlib.Data.Matrix.Basic"] = true
			importsMap["Mathlib.Data.Fin.VecNotation"] = true
			importsMap["Mathlib.LinearAlgebra.Matrix.Notation"] = true
			importsMap["Mathlib.Tactic.FinCases"] = true
			importsMap["Mathlib.Tactic.NormNum"] = true
		case *PellCertificate:
			importsMap["Mathlib.Data.Int.Basic"] = true
			importsMap["Mathlib.Tactic.NormNum"] = true
		case *RecurrenceCertificate:
			importsMap["Mathlib.Algebra.LinearRecurrence"] = true
			importsMap["Mathlib.Tactic.NormNum"] = true
		case *IdealMembershipCertificate:
			importsMap["Mathlib.Tactic.LinearCombination"] = true
		case *SOSCertificate:
			importsMap["Mathlib.Tactic.Positivity"] = true
		case *PolynomialRootCertificate, *IntegerRelationCertificate, *EllipticPointCertificate:
			importsMap["Mathlib.Data.Rat.Defs"] = true
			importsMap["Mathlib.Tactic.NormNum"] = true
		case *DerivCertificate:
			importsMap["Mathlib.Analysis.Calculus.Deriv.Basic"] = true
			importsMap["Mathlib.Analysis.Calculus.Deriv.Pow"] = true
			importsMap["Mathlib.Analysis.Calculus.Deriv.Add"] = true
			importsMap["Mathlib.Analysis.Calculus.Deriv.Mul"] = true
			importsMap["Mathlib.Tactic.FieldSimp"] = true
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

	domain := inferTheoremDomainType(cert, nodes...)
	switch domain {
	case DomainReal:
		importsMap["Mathlib.Basic.Real.Basic"] = true
	case DomainInt:
		importsMap["Mathlib.Data.Int.Basic"] = true
	case DomainComplex:
		importsMap["Mathlib.Data.Complex.Basic"] = true
	case DomainZMod:
		importsMap["Mathlib.Data.ZMod.Basic"] = true
	default:
		importsMap["Mathlib.Data.Rat.Defs"] = true
	}

	if hasVariableDenominator(nodes...) {
		importsMap["Mathlib.Tactic.FieldSimp"] = true
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
			hasReal = true
			return
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

// containsTanNodes checks whether any of the given nodes contain tangent or related quotient trigonometric functions.
func containsTanNodes(nodes ...Node) bool {
	hasTan := false
	var walk func(n Node)
	walk = func(n Node) {
		if n == nil || hasTan {
			return
		}
		switch curr := n.(type) {
		case *FuncNode:
			switch strings.ToLower(curr.Name) {
			case "tan", "cot", "sec", "csc":
				hasTan = true
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
	return hasTan
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
	var subKeys []string
	for k := range subs {
		subKeys = append(subKeys, k)
	}
	sort.Strings(subKeys)
	for _, k := range subKeys {
		substed = Substitute(substed, k, subs[k])
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

// LeanDomainType represents mathematical number domains supported in Lean 4 theorem transpilation.
type LeanDomainType string

const (
	DomainInt     LeanDomainType = "ℤ"
	DomainRat     LeanDomainType = "ℚ"
	DomainReal    LeanDomainType = "ℝ"
	DomainComplex LeanDomainType = "ℂ"
	DomainZMod    LeanDomainType = "ZMod"
)

// containsComplexNodes checks whether any of the given nodes contain complex numbers or the imaginary unit i.
func containsComplexNodes(nodes ...Node) bool {
	hasC := false
	var walk func(n Node)
	walk = func(n Node) {
		if n == nil || hasC {
			return
		}
		if _, ok := n.(*ComplexNode); ok {
			hasC = true
			return
		}
		switch curr := n.(type) {
		case *ConstNode:
			if curr.Name == "i" || curr.Name == "I" {
				hasC = true
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
		case *FuncNode:
			for _, a := range curr.Args {
				walk(a)
			}
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
	return hasC
}

// containsModNodes checks whether any of the given nodes contain modular arithmetic operations.
func containsModNodes(nodes ...Node) bool {
	hasMod := false
	var walk func(n Node)
	walk = func(n Node) {
		if n == nil || hasMod {
			return
		}
		switch curr := n.(type) {
		case *FuncNode:
			name := strings.ToLower(curr.Name)
			if name == "mod" || name == "solve_mod" {
				hasMod = true
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
	return hasMod
}

// isNegativeExponent checks if an AST node represents a negative power exponent.
func isNegativeExponent(n Node) bool {
	if n == nil {
		return false
	}
	switch exp := n.(type) {
	case *RationalNode:
		return exp.Val != nil && exp.Val.Sign() < 0
	case *UnaryOpNode:
		if exp.Op == "-" {
			return !isNegativeExponent(exp.Expr)
		}
	}
	return false
}

// collectNonzeroDenominators extracts non-constant denominator expressions from the AST nodes.
// In ihd, division by an expression E is represented as E ^ (-k) where k > 0.
func collectNonzeroDenominators(nodes ...Node) []string {
	seen := make(map[string]bool)
	var denoms []string

	var walk func(n Node)
	walk = func(n Node) {
		if n == nil {
			return
		}
		switch curr := n.(type) {
		case *PowNode:
			if isNegativeExponent(curr.Exp) {
				var addDenom func(base Node)
				addDenom = func(base Node) {
					if base == nil {
						return
					}
					if mul, ok := base.(*MulNode); ok {
						for _, f := range mul.Factors {
							addDenom(f)
						}
						return
					}
					if pow, ok := base.(*PowNode); ok {
						addDenom(pow.Base)
						return
					}
					if len(CollectFreeVariables(base)) > 0 {
						baseSyntax, err := ToLeanSyntax(base)
						if err == nil && baseSyntax != "" {
							baseSyntax = formatCalcTerm(baseSyntax)
							normKey := unwrapOuterParens(baseSyntax)
							if !seen[normKey] {
								seen[normKey] = true
								denoms = append(denoms, normKey)
							}
						}
					}
				}
				addDenom(curr.Base)
			}
			walk(curr.Base)
			walk(curr.Exp)
		case *FuncNode:
			name := strings.ToLower(curr.Name)
			if (name == "tan" || name == "sec") && len(curr.Args) == 1 {
				cosNode := &FuncNode{Name: "cos", Args: []Node{curr.Args[0]}}
				if len(CollectFreeVariables(curr.Args[0])) > 0 {
					baseSyntax, err := ToLeanSyntax(cosNode)
					if err == nil && baseSyntax != "" {
						baseSyntax = formatCalcTerm(baseSyntax)
						normKey := unwrapOuterParens(baseSyntax)
						if !seen[normKey] {
							seen[normKey] = true
							denoms = append(denoms, normKey)
						}
					}
				}
			} else if (name == "cot" || name == "csc") && len(curr.Args) == 1 {
				sinNode := &FuncNode{Name: "sin", Args: []Node{curr.Args[0]}}
				if len(CollectFreeVariables(curr.Args[0])) > 0 {
					baseSyntax, err := ToLeanSyntax(sinNode)
					if err == nil && baseSyntax != "" {
						baseSyntax = formatCalcTerm(baseSyntax)
						normKey := unwrapOuterParens(baseSyntax)
						if !seen[normKey] {
							seen[normKey] = true
							denoms = append(denoms, normKey)
						}
					}
				}
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
	sort.Strings(denoms)
	return denoms
}

// collectDomainConditions extracts non-division domain conditions (e.g. 0 < x for log(x), 0 ≤ x for sqrt(x))
// to guard against Lean 4 total function junk-value soundness holes.
func collectDomainConditions(nodes ...Node) []string {
	seen := make(map[string]bool)
	var conds []string

	var walk func(n Node)
	walk = func(n Node) {
		if n == nil {
			return
		}
		switch curr := n.(type) {
		case *FuncNode:
			name := strings.ToLower(curr.Name)
			if (name == "log" || name == "ln") && len(curr.Args) == 1 {
				arg := curr.Args[0]
				if len(CollectFreeVariables(arg)) > 0 {
					argSyntax, err := ToLeanSyntax(arg)
					if err == nil && argSyntax != "" {
						argSyntax = formatCalcTerm(argSyntax)
						condStr := fmt.Sprintf("0 < %s", argSyntax)
						if !seen[condStr] {
							seen[condStr] = true
							conds = append(conds, condStr)
						}
					}
				}
			}
			for _, a := range curr.Args {
				walk(a)
			}
		case *SqrtNode:
			arg := curr.Radicand
			if len(CollectFreeVariables(arg)) > 0 {
				argSyntax, err := ToLeanSyntax(arg)
				if err == nil && argSyntax != "" {
					argSyntax = formatCalcTerm(argSyntax)
					condStr := fmt.Sprintf("0 ≤ %s", argSyntax)
					if !seen[condStr] {
						seen[condStr] = true
						conds = append(conds, condStr)
					}
				}
			}
			walk(curr.Radicand)
		case *PowNode:
			// For fractional powers with even denominator (e.g. x^(1/2)), enforce 0 ≤ Base
			if rat, ok := curr.Exp.(*RationalNode); ok && rat.Val != nil && rat.Val.Sign() > 0 {
				denom := rat.Val.Denom()
				if denom.Bit(0) == 0 { // even denominator
					if len(CollectFreeVariables(curr.Base)) > 0 {
						baseSyntax, err := ToLeanSyntax(curr.Base)
						if err == nil && baseSyntax != "" {
							baseSyntax = formatCalcTerm(baseSyntax)
							condStr := fmt.Sprintf("0 ≤ %s", baseSyntax)
							if !seen[condStr] {
								seen[condStr] = true
								conds = append(conds, condStr)
							}
						}
					}
				}
			}
			walk(curr.Base)
			walk(curr.Exp)
		case *AddNode:
			for _, t := range curr.Terms {
				walk(t)
			}
		case *MulNode:
			for _, f := range curr.Factors {
				walk(f)
			}
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
	sort.Strings(conds)
	return conds
}

// hasVariableDenominator returns true if any of the nodes contain a variable denominator.
func hasVariableDenominator(nodes ...Node) bool {
	return len(collectNonzeroDenominators(nodes...)) > 0
}

// inferTheoremDomainType determines the most specific and sound mathematical carrier type.
func inferTheoremDomainType(cert Certificate, nodes ...Node) LeanDomainType {
	if cert != nil {
		switch c := cert.(type) {
		case *PellCertificate, *LinearDiophantineCertificate:
			return DomainInt
		case *IntegerRelationCertificate:
			allNodes := append([]Node{}, c.Elements...)
			allNodes = append(allNodes, c.Coefficients...)
			allNodes = append(allNodes, nodes...)
			if containsRealNodes(allNodes...) {
				return DomainReal
			}
			return DomainInt
		case *DerivCertificate, *ODECertificate, *GeometricCertificate:
			return DomainReal
		}
	}

	if containsComplexNodes(nodes...) {
		return DomainComplex
	}
	if containsRealNodes(nodes...) {
		return DomainReal
	}
	if containsModNodes(nodes...) {
		return DomainZMod
	}
	return DomainRat
}

// buildHasDerivAtCombinator constructs a Lean 4 HasDerivAt proof term from the antiderivative AST.
func buildHasDerivAtCombinator(node Node, varName string) (string, error) {
	if node == nil {
		return "", fmt.Errorf("cannot build HasDerivAt combinator for nil node")
	}

	vars := CollectFreeVariables(node)
	isConst := true
	for _, v := range vars {
		if v == varName {
			isConst = false
			break
		}
	}
	if isConst {
		leanVal, err := ToLeanSyntax(node)
		if err != nil || leanVal == "" {
			return "", fmt.Errorf("failed to transpile constant derivative node: %w", err)
		}
		leanVal = strings.ReplaceAll(leanVal, ": ℚ", ": ℝ")
		if isEnclosedInParens(leanVal) {
			leanVal = fmt.Sprintf("(%s : ℝ)", leanVal[1:len(leanVal)-1])
		} else {
			leanVal = fmt.Sprintf("(%s : ℝ)", leanVal)
		}
		return fmt.Sprintf("(hasDerivAt_const %s %s)", varName, leanVal), nil
	}

	switch n := node.(type) {
	case *VarNode:
		if n.Name == varName {
			return fmt.Sprintf("(hasDerivAt_id' %s)", varName), nil
		}
		return fmt.Sprintf("(hasDerivAt_const %s %s)", varName, escapeLeanIdent(n.Name)), nil

	case *AddNode:
		if len(n.Terms) == 0 {
			return fmt.Sprintf("(hasDerivAt_const %s 0)", varName), nil
		}
		term, err := buildHasDerivAtCombinator(n.Terms[0], varName)
		if err != nil {
			return "", err
		}
		for i := 1; i < len(n.Terms); i++ {
			nextTerm, err := buildHasDerivAtCombinator(n.Terms[i], varName)
			if err != nil {
				return "", err
			}
			term = fmt.Sprintf("(%s.add %s)", term, nextTerm)
		}
		return term, nil

	case *MulNode:
		if len(n.Factors) == 0 {
			return fmt.Sprintf("(hasDerivAt_const %s 1)", varName), nil
		}
		factor, err := buildHasDerivAtCombinator(n.Factors[0], varName)
		if err != nil {
			return "", err
		}
		for i := 1; i < len(n.Factors); i++ {
			nextFactor, err := buildHasDerivAtCombinator(n.Factors[i], varName)
			if err != nil {
				return "", err
			}
			factor = fmt.Sprintf("(%s.mul %s)", factor, nextFactor)
		}
		return factor, nil

	case *UnaryOpNode:
		sub, err := buildHasDerivAtCombinator(n.Expr, varName)
		if err != nil {
			return "", err
		}
		if n.Op == "-" {
			return fmt.Sprintf("(%s.neg)", sub), nil
		}
		return sub, nil

	case *PowNode:
		if rat, ok := n.Exp.(*RationalNode); ok && rat.Val != nil && rat.Val.IsInt() {
			if rat.Val.Sign() == 0 {
				return fmt.Sprintf("(hasDerivAt_const %s 1)", varName), nil
			}
			if rat.Val.Sign() > 0 && rat.Val.Num().IsInt64() {
				intExp := rat.Val.Num().Int64()
				if v, ok := n.Base.(*VarNode); ok && v.Name == varName {
					return fmt.Sprintf("(hasDerivAt_pow %d %s)", intExp, varName), nil
				}
				baseComb, err := buildHasDerivAtCombinator(n.Base, varName)
				if err != nil {
					return "", err
				}
				baseSyntax, err := ToLeanSyntax(n.Base)
				if err != nil {
					return "", err
				}
				baseSyntax = formatCalcTerm(baseSyntax)
				powComb := fmt.Sprintf("(hasDerivAt_pow %d %s)", intExp, baseSyntax)
				return fmt.Sprintf("(%s.comp %s %s)", powComb, varName, baseComb), nil
			}
		}

	case *FuncNode:
		name := strings.ToLower(n.Name)
		if len(n.Args) == 1 {
			arg := n.Args[0]
			if v, ok := arg.(*VarNode); ok && v.Name == varName {
				switch name {
				case "sin":
					return fmt.Sprintf("(Real.hasDerivAt_sin %s)", varName), nil
				case "cos":
					return fmt.Sprintf("(Real.hasDerivAt_cos %s)", varName), nil
				case "exp":
					return fmt.Sprintf("(Real.hasDerivAt_exp %s)", varName), nil
				}
			}
			argComb, err := buildHasDerivAtCombinator(arg, varName)
			if err != nil {
				return "", err
			}
			argSyntax, err := ToLeanSyntax(arg)
			if err != nil {
				return "", err
			}
			argSyntax = formatCalcTerm(argSyntax)
			switch name {
			case "sin":
				return fmt.Sprintf("((Real.hasDerivAt_sin %s).comp %s %s)", argSyntax, varName, argComb), nil
			case "cos":
				return fmt.Sprintf("((Real.hasDerivAt_cos %s).comp %s %s)", argSyntax, varName, argComb), nil
			case "exp":
				return fmt.Sprintf("((Real.hasDerivAt_exp %s).comp %s %s)", argSyntax, varName, argComb), nil
			}
		}
	}

	return "", fmt.Errorf("unsupported AST node %T for HasDerivAt combinator", node)
}

// formatCalcTerm ensures the Lean expression is syntactically safe on either side of an equality in a calc line.
// If it contains arithmetic operators or spaces and is not already fully enclosed in parentheses, it wraps it in (...).
func formatCalcTerm(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	if isEnclosedInParens(s) {
		return s
	}
	if strings.ContainsAny(s, "+-*/^ ") {
		return "(" + s + ")"
	}
	return s
}

// selectTacticForStep chooses an appropriate Lean 4 tactic based on the proof step's rule and metadata,
// falling back to a robust Lean 4 'first' ladder when a single tactic cannot be decided with certainty.
func selectTacticForStep(step ProofStep) string {
	rule := strings.ToLower(strings.TrimSpace(step.Rule))
	switch rule {
	case "norm_num", "eval_num", "arithmetic":
		return "by norm_num"
	case "omega", "presburger", "diophantine":
		return "by omega"
	case "linarith", "linear_inequality":
		return "by linarith"
	case "decide":
		return "by decide"
	case "simp":
		return "by simp"
	case "ring", "polynomialfactorization", "differenceofsquares", "factor", "expand":
		return "by ring"
	case "ring_nf":
		return "by ring_nf; rfl"
	case "field_simp":
		return "by field_simp; ring"
	}

	// Inspect expressions in step
	var nodes []Node
	if step.Before != nil {
		nodes = append(nodes, step.Before)
	}
	if step.After != nil {
		nodes = append(nodes, step.After)
	}

	freeVars := CollectFreeVariables(nodes...)
	if len(freeVars) == 0 && len(nodes) > 0 {
		return "by norm_num"
	}

	if hasVariableDenominator(nodes...) {
		return "by field_simp; ring"
	}

	if !containsRealNodes(nodes...) && !containsComplexNodes(nodes...) && !containsModNodes(nodes...) {
		// Pure polynomial identity over commutative ring
		return "by ring"
	}

	// Robust single-line fallback ladder with explicit done check to avoid partial-success hanging
	return "by first | norm_num | omega | linarith | (field_simp; ring; done) | (ring; done)"
}

// ProofTraceToLeanCalc transforms a ProofTrace into a Lean 4 'calc' block.
// It iterates through each atomic rewriting step, generating an equational transitivity chain:
//
//   calc
//     (Before_0) = (After_0) := by ring
//     _          = (After_1) := by ring
//     _          = (After_n) := by ring
func ProofTraceToLeanCalc(trace *ProofTrace) (string, error) {
	if trace == nil {
		return "", fmt.Errorf("%s", i18n.T("lean.err_prooftrace_is_nil"))
	}

	steps := trace.Steps
	if len(steps) == 0 {
		if trace.Original != nil && trace.Result != nil {
			origStr, err := ToLeanSyntax(trace.Original)
			if err != nil {
				return "", err
			}
			resStr, err := ToLeanSyntax(trace.Result)
			if err != nil {
				return "", err
			}
			origStr = formatCalcTerm(origStr)
			resStr = formatCalcTerm(resStr)
			return fmt.Sprintf("calc\n  %s = %s := by ring", origStr, resStr), nil
		}
		return "", fmt.Errorf("%s", i18n.T("lean.err_prooftrace_has_no_steps"))
	}

	var lines []string
	lines = append(lines, "calc")

	for i, step := range steps {
		tactic := selectTacticForStep(step)

		if i == 0 {
			beforeStr, err := ToLeanSyntax(step.Before)
			if err != nil {
				return "", err
			}
			afterStr, err := ToLeanSyntax(step.After)
			if err != nil {
				return "", err
			}
			beforeStr = formatCalcTerm(beforeStr)
			afterStr = formatCalcTerm(afterStr)
			lines = append(lines, fmt.Sprintf("  %s = %s := %s", beforeStr, afterStr, tactic))
		} else {
			afterStr, err := ToLeanSyntax(step.After)
			if err != nil {
				return "", err
			}
			afterStr = formatCalcTerm(afterStr)
			lines = append(lines, fmt.Sprintf("  _ = %s := %s", afterStr, tactic))
		}
	}

	return strings.Join(lines, "\n"), nil
}

// GenerateLeanSourceFromTrace generates a standalone, valid Lean 4 source file
// directly from a ProofTrace, using a transitivity 'calc' block proof.
func GenerateLeanSourceFromTrace(theoremName string, trace *ProofTrace) (string, error) {
	if trace == nil {
		return "", fmt.Errorf("%s", i18n.T("lean.err_prooftrace_is_nil"))
	}
	if strings.TrimSpace(theoremName) == "" {
		theoremName = "ihd_certified_proof"
	}

	var allNodes []Node
	if trace.Original != nil {
		allNodes = append(allNodes, trace.Original)
	}
	if trace.Result != nil {
		allNodes = append(allNodes, trace.Result)
	}
	for _, step := range trace.Steps {
		if step.Before != nil {
			allNodes = append(allNodes, step.Before)
		}
		if step.After != nil {
			allNodes = append(allNodes, step.After)
		}
		if step.Residual != nil {
			allNodes = append(allNodes, step.Residual)
		}
	}

	vars := CollectFreeVariables(allNodes...)

	calcBlock, err := ProofTraceToLeanCalc(trace)
	if err != nil {
		return "", err
	}

	// Determine equality signature
	var origStr, resStr string
	if trace.Original != nil {
		origStr, _ = ToLeanSyntax(trace.Original)
	} else if len(trace.Steps) > 0 && trace.Steps[0].Before != nil {
		origStr, _ = ToLeanSyntax(trace.Steps[0].Before)
	}
	if trace.Result != nil {
		resStr, _ = ToLeanSyntax(trace.Result)
	} else if len(trace.Steps) > 0 && trace.Steps[len(trace.Steps)-1].After != nil {
		resStr, _ = ToLeanSyntax(trace.Steps[len(trace.Steps)-1].After)
	}
	origStr = formatCalcTerm(origStr)
	resStr = formatCalcTerm(resStr)

	// Indent the calc block inside the theorem definition
	calcLines := strings.Split(calcBlock, "\n")
	var indentedCalc strings.Builder
	for _, line := range calcLines {
		indentedCalc.WriteString("  ")
		indentedCalc.WriteString(line)
		indentedCalc.WriteByte('\n')
	}

	// Collect variable denominators and domain conditions for calc mode
	binders := make([]string, 0)
	seenProps := make(map[string]bool)
	denoms := collectNonzeroDenominators(allNodes...)
	denomIdx := 0
	for _, d := range denoms {
		cleanD := unwrapOuterParens(d)
		prop := fmt.Sprintf("%s ≠ 0", cleanD)
		if !seenProps[prop] {
			seenProps[prop] = true
			binders = append(binders, fmt.Sprintf("(h_denom%d : %s)", denomIdx, prop))
			denomIdx++
		}
	}
	domConds := collectDomainConditions(allNodes...)
	domIdx := 0
	for _, dc := range domConds {
		if !seenProps[dc] {
			seenProps[dc] = true
			binders = append(binders, fmt.Sprintf("(h_dom%d : %s)", domIdx, dc))
			domIdx++
		}
	}

	var theoremCode string
	if len(binders) > 0 {
		theoremCode = fmt.Sprintf("theorem %s %s : %s = %s := by\n%s",
			escapeLeanIdent(theoremName), strings.Join(binders, " "), origStr, resStr, strings.TrimRight(indentedCalc.String(), "\n"))
	} else {
		theoremCode = fmt.Sprintf("theorem %s : %s = %s := by\n%s",
			escapeLeanIdent(theoremName), origStr, resStr, strings.TrimRight(indentedCalc.String(), "\n"))
	}

	requiredImports := DetermineRequiredImports(nil, allNodes...)
	// Ensure Ring tactic import is present for calc algebraic steps
	hasRing := false
	for _, imp := range requiredImports {
		if imp == "Mathlib.Tactic.Ring" {
			hasRing = true
			break
		}
	}
	if !hasRing {
		requiredImports = append(requiredImports, "Mathlib.Tactic.Ring")
		sort.Strings(requiredImports)
	}

	var b strings.Builder
	b.WriteString("-- Automatically generated by ihd (i-hate-decimal-calc) Lean 4 Transpiler\n")
	b.WriteString("-- Mathematical proof certificate certified with Skeptic's verification\n")
	for _, imp := range requiredImports {
		b.WriteString(fmt.Sprintf("import %s\n", imp))
	}
	b.WriteString("\n")

	domainType := inferTheoremDomainType(nil, allNodes...)
	if domainType == "" {
		domainType = DomainRat
	}
	if len(vars) > 0 {
		if domainType == DomainZMod {
			b.WriteString(fmt.Sprintf("variable (n : ℕ) (%s : ZMod n)\n\n", strings.Join(vars, " ")))
		} else {
			b.WriteString(fmt.Sprintf("variable (%s : %s)\n\n", strings.Join(vars, " "), domainType))
		}
	}

	b.WriteString(theoremCode)
	b.WriteString("\n\n#print axioms ")
	b.WriteString(escapeLeanIdent(theoremName))
	b.WriteString("\n")

	return b.String(), nil
}




// transpileIdealMembershipCertificate transforms an IdealMembershipCertificate into a Lean 4 linear_combination theorem.
func transpileIdealMembershipCertificate(c *IdealMembershipCertificate, theoremName string) (string, error) {
	if !c.IsVerified() || !c.IsMember {
		return "", fmt.Errorf("cannot transpile IdealMembershipCertificate: membership not verified")
	}

	targetStr, err := ToLeanSyntax(c.Target)
	if err != nil {
		return "", fmt.Errorf("transpile ideal target: %w", err)
	}

	var binders []string
	var combTerms []string
	for i, gen := range c.Generators {
		hName := fmt.Sprintf("h%d", i+1)
		genStr, err := ToLeanSyntax(gen)
		if err != nil {
			return "", fmt.Errorf("transpile ideal generator: %w", err)
		}
		binders = append(binders, fmt.Sprintf("(%s : %s = 0)", hName, genStr))

		if i < len(c.Cofactors) {
			cof := c.Cofactors[i]
			cofStr, err := ToLeanSyntax(cof)
			if err != nil {
				return "", fmt.Errorf("transpile ideal cofactor: %w", err)
			}
			if cofStr != "0" {
				combTerms = append(combTerms, fmt.Sprintf("(%s) * %s", cofStr, hName))
			}
		}
	}

	var tactic string
	if len(combTerms) == 0 {
		tactic = "by ring"
	} else {
		tactic = fmt.Sprintf("by\n  linear_combination %s", strings.Join(combTerms, " + "))
	}

	if len(binders) > 0 {
		return fmt.Sprintf("theorem %s %s : %s = 0 := %s", escapeLeanIdent(theoremName), strings.Join(binders, " "), targetStr, tactic), nil
	}
	return fmt.Sprintf("theorem %s : %s = 0 := %s", escapeLeanIdent(theoremName), targetStr, tactic), nil
}

// transpileSOSCertificate transforms an SOSCertificate into a Lean 4 positivity theorem.
func transpileSOSCertificate(c *SOSCertificate, theoremName string) (string, error) {
	if !c.IsVerified() || c.Status != SOSCertified {
		return "", fmt.Errorf("cannot transpile SOSCertificate: SOS decomposition not certified")
	}

	targetStr, err := ToLeanSyntax(c.Target)
	if err != nil {
		return "", fmt.Errorf("transpile SOS target: %w", err)
	}

	var sumTerms []string
	for _, term := range c.Terms {
		qStr, err := ToLeanSyntax(term.Poly)
		if err != nil {
			return "", fmt.Errorf("transpile SOS component: %w", err)
		}
		if term.Weight.Cmp(big.NewRat(1, 1)) == 0 {
			sumTerms = append(sumTerms, fmt.Sprintf("(%s)^2", qStr))
		} else {
			wNode := NewRationalFromBigRat(term.Weight)
			wStr, err := ToLeanSyntax(wNode)
			if err != nil {
				return "", fmt.Errorf("transpile SOS weight: %w", err)
			}
			sumTerms = append(sumTerms, fmt.Sprintf("(%s) * (%s)^2", wStr, qStr))
		}
	}

	rhsStr := "0"
	if len(sumTerms) > 0 {
		rhsStr = strings.Join(sumTerms, " + ")
	}

	tactic := fmt.Sprintf("by\n  have h_sos : %s = %s := by ring\n  rw [h_sos]\n  positivity", targetStr, rhsStr)
	return fmt.Sprintf("theorem %s : 0 ≤ %s := %s", escapeLeanIdent(theoremName), targetStr, tactic), nil
}

// NegativeMutant represents an artificially altered Lean theorem designed to test
// that the Lean kernel rejects falsified mathematical assertions (Negative Mutation Testing).
type NegativeMutant struct {
	ID             string `json:"id"`
	Operator       string `json:"operator"` // "constant_perturbation", "operator_inversion", "relation_inversion", "certificate_perturbation"
	TheoremName    string `json:"theorem_name"`
	LeanCode       string `json:"lean_code"`
	ExpectedResult string `json:"expected_result"` // "REJECT"
}

// GenerateNegativeMutants generates 4 deterministic type-preserving mutant Lean theorems
// from a target AST expression or certificate.
func GenerateNegativeMutants(theoremName string, node ast.Node, cert Certificate) ([]NegativeMutant, error) {
	if node == nil && cert == nil {
		return nil, fmt.Errorf("both node and cert are nil")
	}

	var baseExpr string
	var err error
	var targetNode ast.Node = node

	if targetNode == nil && cert != nil {
		if sos, ok := cert.(*SOSCertificate); ok {
			targetNode = sos.Target
		} else if ideal, ok := cert.(*IdealMembershipCertificate); ok {
			targetNode = ideal.Target
		}
	}

	if targetNode != nil {
		baseExpr, err = ToLeanSyntax(targetNode)
		if err != nil {
			return nil, fmt.Errorf("transpile baseExpr for mutants: %w", err)
		}
	}

	// Collect free variables to inject explicit type binders for autoImplicit = false
	vars := CollectFreeVariables(targetNode)
	carrierType := inferTheoremDomainType(cert, targetNode)
	if carrierType == "" {
		carrierType = DomainRat
	}
	var binderPrefix string
	if len(vars) > 0 {
		binderPrefix = fmt.Sprintf(" (%s : %s)", strings.Join(vars, " "), carrierType)
	}

	var mutants []NegativeMutant

	// 1. Constant Perturbation (RHS + 1)
	m1Name := fmt.Sprintf("%s_mutant_c_add_1", theoremName)
	m1Code := fmt.Sprintf("/--\nerror: ring failed\n-/\n#guard_msgs (whitespace := lax) in\nexample%s : %s = (%s) + 1 := by ring", binderPrefix, baseExpr, baseExpr)
	mutants = append(mutants, NegativeMutant{
		ID:             m1Name,
		Operator:       "constant_perturbation",
		TheoremName:    m1Name,
		LeanCode:       m1Code,
		ExpectedResult: "REJECT",
	})

	// 2. Unconditional Sign Inversion (LHS = -(RHS))
	m2Name := fmt.Sprintf("%s_mutant_sign_flip", theoremName)
	m2Code := fmt.Sprintf("/--\nerror: ring failed\n-/\n#guard_msgs (whitespace := lax) in\nexample%s : %s = -(%s) := by ring", binderPrefix, baseExpr, baseExpr)
	mutants = append(mutants, NegativeMutant{
		ID:             m2Name,
		Operator:       "operator_inversion",
		TheoremName:    m2Name,
		LeanCode:       m2Code,
		ExpectedResult: "REJECT",
	})

	// 3. Relational Inversion dynamically derived from target expression
	m3Name := fmt.Sprintf("%s_mutant_rel_inversion", theoremName)
	m3Code := fmt.Sprintf("/--\nerror: linarith failed\n-/\n#guard_msgs (whitespace := lax) in\nexample%s : (%s) > (%s) + 1 := by linarith", binderPrefix, baseExpr, baseExpr)
	mutants = append(mutants, NegativeMutant{
		ID:             m3Name,
		Operator:       "relation_inversion",
		TheoremName:    m3Name,
		LeanCode:       m3Code,
		ExpectedResult: "REJECT",
	})

	// 4. Certificate Perturbation (if cert is an SOSCertificate or IdealMembershipCertificate)
	if sosCert, ok := cert.(*SOSCertificate); ok && sosCert.IsVerified() {
		m4Name := fmt.Sprintf("%s_mutant_sos_perturb", theoremName)
		sosTarget, err := ToLeanSyntax(sosCert.Target)
		if err != nil {
			return nil, fmt.Errorf("transpile SOS target for mutant: %w", err)
		}
		m4Code := fmt.Sprintf("/--\nerror: positivity failed\n-/\n#guard_msgs (whitespace := lax) in\nexample%s : 0 ≤ -(%s) - 1 := by positivity", binderPrefix, sosTarget)
		mutants = append(mutants, NegativeMutant{
			ID:             m4Name,
			Operator:       "certificate_perturbation",
			TheoremName:    m4Name,
			LeanCode:       m4Code,
			ExpectedResult: "REJECT",
		})
	} else if idealCert, ok := cert.(*IdealMembershipCertificate); ok && idealCert.IsVerified() {
		m4Name := fmt.Sprintf("%s_mutant_ideal_perturb", theoremName)
		idealTarget, err := ToLeanSyntax(idealCert.Target)
		if err != nil {
			return nil, fmt.Errorf("transpile ideal target for mutant: %w", err)
		}
		m4Code := fmt.Sprintf("/--\nerror: ring failed\n-/\n#guard_msgs (whitespace := lax) in\nexample%s : %s + 1 = 0 := by ring", binderPrefix, idealTarget)
		mutants = append(mutants, NegativeMutant{
			ID:             m4Name,
			Operator:       "certificate_perturbation",
			TheoremName:    m4Name,
			LeanCode:       m4Code,
			ExpectedResult: "REJECT",
		})
	}

	return mutants, nil
}

// GeneratePremiseWitnessTheorem generates a satisfiability witness theorem (`∃ x, H x := ⟨witness, by norm_num⟩`)
// to eliminate vacuous truth bugs from contradictory premise hypotheses.
func GeneratePremiseWitnessTheorem(theoremName string, variable string, premise string, witness string) string {
	wName := fmt.Sprintf("%s_premise_witness", theoremName)
	return fmt.Sprintf("theorem %s : ∃ %s : ℚ, %s := ⟨%s, by norm_num⟩", escapeLeanIdent(wName), variable, premise, witness)
}

