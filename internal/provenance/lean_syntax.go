package provenance

import (
	"fmt"
	"sort"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// LeanKeywords contains reserved keywords in Lean 4 that cannot be used as raw variable identifiers.
var LeanKeywords = map[string]bool{
	"def":        true,
	"theorem":    true,
	"lemma":      true,
	"example":    true,
	"axiom":      true,
	"variable":   true,
	"universe":   true,
	"import":     true,
	"open":       true,
	"namespace":  true,
	"section":    true,
	"end":        true,
	"inductive":  true,
	"structure":  true,
	"class":      true,
	"instance":   true,
	"where":      true,
	"with":       true,
	"match":      true,
	"fun":        true,
	"have":       true,
	"show":       true,
	"suffices":   true,
	"by":         true,
	"calc":       true,
	"let":        true,
	"if":         true,
	"then":       true,
	"else":       true,
	"for":        true,
	"in":         true,
	"do":         true,
	"return":     true,
	"sorry":      true,
	"admit":      true,
	"set_option": true,
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

// formatCalcTerm ensures the Lean expression is syntactically safe on either side of an equality in a calc line.
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

// ToLeanSyntax converts an internal AST node to Lean 4 / Mathlib syntax string.
func ToLeanSyntax(node ast.Node) (string, error) {
	if node == nil {
		return "", fmt.Errorf("%s", i18n.T("lean.err_unsupported_ast_node", nil))
	}

	switch n := node.(type) {
	case *ast.RationalNode:
		if n.Val.IsInt() {
			num := n.Val.Num()
			if num.Sign() < 0 {
				return fmt.Sprintf("(%s)", num.String()), nil
			}
			return num.String(), nil
		}
		num := n.Val.Num()
		denom := n.Val.Denom()
		return fmt.Sprintf("((%s : ℚ) / %s)", num.String(), denom.String()), nil

	case *ast.VarNode:
		return escapeLeanIdent(n.Name), nil

	case *ast.ConstNode:
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

	case *ast.SqrtNode:
		inner, err := ToLeanSyntax(n.Radicand)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Real.sqrt (%s)", inner), nil

	case *ast.UnaryOpNode:
		sub, err := ToLeanSyntax(n.Expr)
		if err != nil {
			return "", err
		}
		if n.Op == "-" {
			return fmt.Sprintf("-(%s)", sub), nil
		}
		return sub, nil

	case *ast.AddNode:
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

	case *ast.MulNode:
		if len(n.Factors) == 0 {
			return "1", nil
		}
		var factors []string
		for _, f := range n.Factors {
			s, err := ToLeanSyntax(f)
			if err != nil {
				return "", err
			}
			if _, isAdd := f.(*ast.AddNode); isAdd {
				s = "(" + s + ")"
			}
			factors = append(factors, s)
		}
		return strings.Join(factors, " * "), nil

	case *ast.PowNode:
		baseStr, err := ToLeanSyntax(n.Base)
		if err != nil {
			return "", err
		}
		if _, isAdd := n.Base.(*ast.AddNode); isAdd {
			baseStr = "(" + baseStr + ")"
		} else if _, isMul := n.Base.(*ast.MulNode); isMul {
			baseStr = "(" + baseStr + ")"
		}

		expStr, err := ToLeanSyntax(n.Exp)
		if err != nil {
			return "", err
		}
		if _, isAdd := n.Exp.(*ast.AddNode); isAdd {
			expStr = "(" + expStr + ")"
		} else if _, isMul := n.Exp.(*ast.MulNode); isMul {
			expStr = "(" + expStr + ")"
		}
		return fmt.Sprintf("%s ^ %s", baseStr, expStr), nil

	case *ast.RelOpNode:
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

	case *ast.FuncNode:
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

	case *ast.MatrixNode:
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
		return "", fmt.Errorf("%s", i18n.T("lean.err_unsupported_ast_node", n))
	}
}

// CollectFreeVariables gathers all unique variable names appearing in the given nodes, sorted alphabetically.
func CollectFreeVariables(nodes ...ast.Node) []string {
	varMap := make(map[string]bool)
	var walk func(n ast.Node)
	walk = func(n ast.Node) {
		if n == nil {
			return
		}
		switch curr := n.(type) {
		case *ast.VarNode:
			if curr.Name == "true" || curr.Name == "false" || curr.Name == "pi" || curr.Name == "e" || curr.Name == "i" {
				return
			}
			varMap[curr.Name] = true
		case *ast.AddNode:
			for _, t := range curr.Terms {
				walk(t)
			}
		case *ast.MulNode:
			for _, f := range curr.Factors {
				walk(f)
			}
		case *ast.PowNode:
			walk(curr.Base)
			walk(curr.Exp)
		case *ast.UnaryOpNode:
			walk(curr.Expr)
		case *ast.RelOpNode:
			walk(curr.LHS)
			walk(curr.RHS)
		case *ast.FuncNode:
			for _, a := range curr.Args {
				walk(a)
			}
		case *ast.SqrtNode:
			walk(curr.Radicand)
		case *ast.MatrixNode:
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

// containsRealNodes checks if any node represents real or irrational numbers (sqrt, trig, pi, e, etc.).
func containsRealNodes(nodes ...ast.Node) bool {
	var found bool
	var walk func(n ast.Node)
	walk = func(n ast.Node) {
		if n == nil || found {
			return
		}
		switch curr := n.(type) {
		case *ast.SqrtNode:
			found = true
			return
		case *ast.ConstNode:
			name := strings.ToLower(curr.Name)
			if name == "pi" || name == "e" {
				found = true
				return
			}
		case *ast.FuncNode:
			name := strings.ToLower(curr.Name)
			if name == "sin" || name == "cos" || name == "tan" || name == "exp" || name == "ln" || name == "log" {
				found = true
				return
			}
			for _, a := range curr.Args {
				walk(a)
			}
		case *ast.AddNode:
			for _, t := range curr.Terms {
				walk(t)
			}
		case *ast.MulNode:
			for _, f := range curr.Factors {
				walk(f)
			}
		case *ast.PowNode:
			walk(curr.Base)
			walk(curr.Exp)
		case *ast.UnaryOpNode:
			walk(curr.Expr)
		case *ast.RelOpNode:
			walk(curr.LHS)
			walk(curr.RHS)
		case *ast.MatrixNode:
			for _, r := range curr.Data {
				for _, elem := range r {
					walk(elem)
				}
			}
		}
	}

	for _, n := range nodes {
		walk(n)
		if found {
			break
		}
	}
	return found
}
