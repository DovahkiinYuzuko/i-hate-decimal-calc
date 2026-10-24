package main

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// FunctionDoc contains human-readable documentation for a function.
type FunctionDoc struct {
	Name        string
	Signature   string
	Description string
}

// GetFunctionDoc retrieves the formatted doc for a given function name.
func GetFunctionDoc(name string) (*FunctionDoc, bool) {
	sig, ok := calc.LookupSignature(name)
	if !ok {
		return nil, false
	}

	sigStr := fmt.Sprintf("%s(%s)", sig.Name, strings.Join(sig.Args, ", "))
	desc := i18n.T(sig.DescKey)
	if desc == sig.DescKey || strings.HasPrefix(desc, "fn.") {
		desc = fmt.Sprintf("Built-in function %s", sig.Name)
	}

	return &FunctionDoc{
		Name:        sig.Name,
		Signature:   sigStr,
		Description: desc,
	}, true
}

// SearchFunctions searches for functions matching the query in their name or description.
func SearchFunctions(query string) []FunctionDoc {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil
	}

	allNames := calc.AllFunctionNames()
	var results []FunctionDoc

	for _, name := range allNames {
		doc, ok := GetFunctionDoc(name)
		if !ok {
			continue
		}
		if strings.Contains(strings.ToLower(doc.Name), query) || strings.Contains(strings.ToLower(doc.Description), query) {
			results = append(results, *doc)
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})
	return results
}

// ConceptExplanation holds mathematical concept explanation and related function recommendations.
type ConceptExplanation struct {
	Target       string
	Concept      string
	Explanation  string
	RelatedFuncs []string
}

// ExplainConcept inspects a target name or AST expression and returns relevant concept information.
func ExplainConcept(target string) *ConceptExplanation {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil
	}

	lower := strings.ToLower(target)

	// 1. Direct function lookup
	if doc, ok := GetFunctionDoc(lower); ok {
		return &ConceptExplanation{
			Target:       doc.Name,
			Concept:      doc.Signature,
			Explanation:  doc.Description,
			RelatedFuncs: findRelatedFunctions(doc.Name),
		}
	}

	// 2. Mathematical keyword / domain lookup
	if concept := matchDomainConcept(lower); concept != nil {
		return concept
	}

	// 3. Try parsing as AST expression
	if node, err := calc.Parse(target); err == nil {
		return explainASTExpression(target, node)
	}

	return &ConceptExplanation{
		Target:       target,
		Concept:      i18n.T("cli.concept_unknown_title"),
		Explanation:  i18n.T("cli.concept_unknown_desc"),
		RelatedFuncs: []string{"help", "search"},
	}
}

func findRelatedFunctions(funcName string) []string {
	switch strings.ToLower(funcName) {
	case "sin", "cos", "tan", "asin", "acos", "atan", "sinh", "cosh", "tanh", "trig_expand", "trig_reduce":
		return []string{"sin", "cos", "tan", "trig_expand", "trig_reduce"}
	case "diff", "integrate", "taylor", "limit", "dsolve", "laplace", "inv_laplace":
		return []string{"diff", "integrate", "limit", "taylor", "dsolve"}
	case "matrix", "det", "inv", "transpose", "trace", "rank", "rref", "eigenvals", "eigenvects", "lu", "qr", "cholesky":
		return []string{"matrix", "det", "inv", "eigenvals", "rref", "lu"}
	case "gcd", "lcm", "mod", "is_prime", "totient", "solve_diophantine", "cfrac", "pell":
		return []string{"gcd", "lcm", "is_prime", "totient", "solve_diophantine"}
	case "ec_torsion", "ec_rank_bound", "ec_add", "ec_mul", "ec_generators", "ec_p_order":
		return []string{"ec_torsion", "ec_rank_bound", "ec_add", "ec_generators"}
	case "groebner", "ideal_intersection", "ideal_colon", "ideal_saturation", "ideal_radical", "primary_decomposition":
		return []string{"groebner", "primary_decomposition", "ideal_radical", "ideal_colon"}
	case "cad", "qe", "sturm", "root_count", "isolate_roots":
		return []string{"cad", "qe", "sturm", "isolate_roots"}
	case "lp_maximize", "lp_minimize":
		return []string{"lp_maximize", "lp_minimize"}
	case "ztrans", "iztrans", "rsolve":
		return []string{"ztrans", "iztrans", "rsolve"}
	default:
		return nil
	}
}

func matchDomainConcept(query string) *ConceptExplanation {
	switch query {
	case "elliptic", "elliptic curve", "楕円曲線":
		return &ConceptExplanation{
			Target:       query,
			Concept:      "Elliptic Curves (y^2 = x^3 + ax + b)",
			Explanation:  i18n.T("cli.concept_elliptic_desc"),
			RelatedFuncs: []string{"ec_torsion", "ec_rank_bound", "ec_add", "ec_generators", "ec_p_order"},
		}
	case "groebner", "grobner", "グレブナー基底", "イデアル":
		return &ConceptExplanation{
			Target:       query,
			Concept:      "Gröbner Bases & Polynomial Ideals",
			Explanation:  i18n.T("cli.concept_groebner_desc"),
			RelatedFuncs: []string{"groebner", "primary_decomposition", "ideal_radical", "ideal_colon", "resultant"},
		}
	case "cad", "qe", "量化子消去", "円筒代数分解":
		return &ConceptExplanation{
			Target:       query,
			Concept:      "Cylindrical Algebraic Decomposition & Quantifier Elimination",
			Explanation:  i18n.T("cli.concept_cad_desc"),
			RelatedFuncs: []string{"cad", "qe", "sturm", "isolate_roots", "piecewise"},
		}
	case "diophantine", "不定方程式", "整数方程式":
		return &ConceptExplanation{
			Target:       query,
			Concept:      "Diophantine Equations",
			Explanation:  i18n.T("cli.concept_diophantine_desc"),
			RelatedFuncs: []string{"solve_diophantine", "pell", "solve_pell", "gcd", "crt"},
		}
	case "matrix", "linear algebra", "行列", "線形代数":
		return &ConceptExplanation{
			Target:       query,
			Concept:      "Linear Algebra & Matrix Decompositions",
			Explanation:  i18n.T("cli.concept_matrix_desc"),
			RelatedFuncs: []string{"matrix", "det", "inv", "eigenvals", "eigenvects", "rref", "lu", "qr", "cholesky"},
		}
	case "calculus", "微積分", "微分", "積分":
		return &ConceptExplanation{
			Target:       query,
			Concept:      "Symbolic Calculus",
			Explanation:  i18n.T("cli.concept_calculus_desc"),
			RelatedFuncs: []string{"diff", "integrate", "limit", "taylor", "dsolve", "laplace"},
		}
	default:
		return nil
	}
}

func explainASTExpression(raw string, node calc.Node) *ConceptExplanation {
	switch node.(type) {
	case *calc.RelOpNode:
		return &ConceptExplanation{
			Target:       raw,
			Concept:      i18n.T("cli.concept_equation_title"),
			Explanation:  i18n.T("cli.concept_equation_desc"),
			RelatedFuncs: []string{"solve", "solve_diophantine", "verify", "cad"},
		}
	case *calc.MatrixNode:
		return &ConceptExplanation{
			Target:       raw,
			Concept:      i18n.T("cli.concept_matrix_lit_title"),
			Explanation:  i18n.T("cli.concept_matrix_lit_desc"),
			RelatedFuncs: []string{"det", "inv", "eigenvals", "eigenvects", "rref", "rank"},
		}
	default:
		// Check for polynomials or general expressions
		vars := extractVariables(node)
		if len(vars) > 0 {
			return &ConceptExplanation{
				Target:       raw,
				Concept:      i18n.T("cli.concept_poly_expr_title"),
				Explanation:  fmt.Sprintf(i18n.T("cli.concept_poly_expr_desc"), strings.Join(vars, ", ")),
				RelatedFuncs: []string{"factor", "expand", "simplify", "diff", "integrate", "solve"},
			}
		}
		return &ConceptExplanation{
			Target:       raw,
			Concept:      i18n.T("cli.concept_const_expr_title"),
			Explanation:  i18n.T("cli.concept_const_expr_desc"),
			RelatedFuncs: []string{"simplify", "approx", "interval", "cfrac"},
		}
	}
}

func extractVariables(node calc.Node) []string {
	varMap := make(map[string]bool)
	calc.Walk(node, func(n calc.Node) bool {
		if vn, ok := n.(*calc.VarNode); ok {
			varMap[vn.Name] = true
		}
		return true
	})

	var vars []string
	for v := range varMap {
		vars = append(vars, v)
	}
	sort.Strings(vars)
	return vars
}

// PrintHelpCommand handles the 'help' and 'help <func>' commands in the REPL.
func PrintHelpCommand(arg string, out io.Writer) {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		fmt.Fprintln(out, i18n.T("cli.repl_help_overview"))
		return
	}

	doc, ok := GetFunctionDoc(arg)
	if !ok {
		fmt.Fprintf(out, i18n.T("cli.repl_help_not_found")+"\n", arg)
		return
	}

	fmt.Fprintf(out, "  %s\n", doc.Signature)
	fmt.Fprintf(out, "  %s\n", doc.Description)
}

// PrintSearchCommand handles the 'search <keyword>' command in the REPL.
func PrintSearchCommand(query string, out io.Writer) {
	query = strings.TrimSpace(query)
	if query == "" {
		fmt.Fprintln(out, i18n.T("cli.repl_search_missing_query"))
		return
	}

	results := SearchFunctions(query)
	if len(results) == 0 {
		fmt.Fprintf(out, i18n.T("cli.repl_search_no_results")+"\n", query)
		return
	}

	fmt.Fprintf(out, i18n.T("cli.repl_search_header")+"\n", len(results), query)
	for _, doc := range results {
		fmt.Fprintf(out, "  %-25s %s\n", doc.Signature, doc.Description)
	}
}

// PrintWhatisCommand handles the 'whatis <target>' command in the REPL.
func PrintWhatisCommand(target string, out io.Writer) {
	target = strings.TrimSpace(target)
	if target == "" {
		fmt.Fprintln(out, i18n.T("cli.repl_whatis_missing_target"))
		return
	}

	expl := ExplainConcept(target)
	if expl == nil {
		fmt.Fprintf(out, i18n.T("cli.repl_help_not_found")+"\n", target)
		return
	}

	fmt.Fprintf(out, "[%s] %s\n", expl.Target, expl.Concept)
	fmt.Fprintf(out, "%s\n", expl.Explanation)
	if len(expl.RelatedFuncs) > 0 {
		fmt.Fprintf(out, "%s: %s\n", i18n.T("cli.repl_whatis_related"), strings.Join(expl.RelatedFuncs, ", "))
	}
}
