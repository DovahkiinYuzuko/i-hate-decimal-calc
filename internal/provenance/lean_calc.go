package provenance

import (
	"fmt"
	"sort"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// LeanCalcVisitor transforms a DerivationNode DAG into formal, machine-checked Lean 4 calc blocks.
// Adheres strictly to the "Zero-Sorry" contract: unsupported steps immediately fail with an error
// rather than emitting unverified 'sorry' placeholders or trivial propositions.
type LeanCalcVisitor struct{}

// NewLeanCalcVisitor constructs a new LeanCalcVisitor.
func NewLeanCalcVisitor() *LeanCalcVisitor {
	return &LeanCalcVisitor{}
}

// SelectTactic chooses a sound Mathlib4 tactic for the derivation step.
// If the rule is not supported by Mathlib automated decision procedures,
// it returns an error rather than allowing an unverified sorry.
func (v *LeanCalcVisitor) SelectTactic(node *DerivationNode) (string, error) {
	if node == nil {
		return "", fmt.Errorf("%s", i18n.T("lean.err_derivation_node_is_nil"))
	}

	rule := strings.ToLower(strings.TrimSpace(node.RuleID))
	switch {
	case rule == RuleAlgebraApartRational:
		return "by field_simp; try ring", nil
	case rule == RuleAlgebraApartPoly || strings.HasPrefix(rule, "algebra.") || rule == "ring" || rule == "expand" || rule == "factor":
		return "by ring", nil
	case rule == RuleNTInvMod || rule == RuleNTCRT || rule == RuleNTBezout || strings.HasPrefix(rule, "arith") || rule == "norm_num" || rule == "eval_num":
		return "by norm_num", nil
	case rule == RuleDiophantineLinear:
		if node.Parameters != nil {
			if pNode, ok := node.Parameters["forall_param"]; ok && pNode != nil {
				return fmt.Sprintf("by intro %s; ring", escapeLeanIdent(pNode.String())), nil
			}
		}
		return "by omega", nil
	case rule == RuleDiophantinePell || rule == RuleDiophantinePythagorean:
		var nodes []ast.Node
		if node.Before != nil {
			nodes = append(nodes, node.Before)
		}
		if node.After != nil {
			nodes = append(nodes, node.After)
		}
		if len(CollectFreeVariables(nodes...)) == 0 {
			return "by norm_num", nil
		}
		return "by ring", nil
	case rule == "decide":
		return "by decide", nil
	default:
		// Zero-tolerance for unverified steps: fail immediately with an explicit error
		return "", fmt.Errorf("%s", i18n.T("lean.err_unsupported_rule_for_lean", node.RuleID))
	}
}

// FlattenSteps extracts an ordered list of atomic rewriting steps from the derivation tree.
func (v *LeanCalcVisitor) FlattenSteps(root *DerivationNode) []*DerivationNode {
	if root == nil {
		return nil
	}

	var steps []*DerivationNode
	var walk func(n *DerivationNode)
	walk = func(n *DerivationNode) {
		if n == nil {
			return
		}
		// Leaf/atomic step with Before and After (container scopes that have children delegate to children)
		if len(n.Children) == 0 && n.Before != nil && n.After != nil {
			steps = append(steps, n)
			return
		}
		for _, child := range n.Children {
			walk(child)
		}
	}

	walk(root)
	return steps
}

// RenderCalc generates a standalone, compilable Lean 4 source string containing a 'calc' transitivity block.
// Includes '#print axioms' to mechanically guarantee that no sorryAx is admitted.
func (v *LeanCalcVisitor) RenderCalc(root *DerivationNode, theoremName string) (string, error) {
	if root == nil {
		return "", fmt.Errorf("%s", i18n.T("lean.err_derivation_node_is_nil"))
	}

	steps := v.FlattenSteps(root)
	if len(steps) == 0 {
		return "", fmt.Errorf("%s", i18n.T("lean.err_prooftrace_has_no_steps"))
	}

	if strings.TrimSpace(theoremName) == "" {
		theoremName = "ihd_certified_derivation"
	}

	var allNodes []ast.Node
	hasIntDomainRule := false
	seenConds := make(map[string]bool)
	var condBinders []string
	condIdx := 0
	hasRationalDomainRule := false

	for _, s := range steps {
		if s.Before != nil {
			allNodes = append(allNodes, s.Before)
		}
		if s.After != nil {
			allNodes = append(allNodes, s.After)
		}
		rLower := strings.ToLower(strings.TrimSpace(s.RuleID))
		if strings.HasPrefix(rLower, "nt.") || strings.HasPrefix(rLower, "diophantine.") {
			hasIntDomainRule = true
		}
		if rLower == RuleAlgebraApartRational {
			hasRationalDomainRule = true
		}
		for _, cond := range s.Conditions {
			if cond.Kind == "NonZero" && cond.Target != nil {
				condStr, err := ToLeanSyntax(cond.Target)
				if err == nil && condStr != "" && !seenConds[condStr] {
					seenConds[condStr] = true
					condBinders = append(condBinders, fmt.Sprintf("(h_nz_%d : (%s) ≠ 0)", condIdx, condStr)) // txthc-ignore
					condIdx++
				}
			}
		}
	}

	// Check whether steps form a single transitive calc chain
	isTransitiveChain := true
	for i := 1; i < len(steps); i++ {
		if steps[i].Before == nil || steps[i-1].After == nil || !steps[i].Before.Equal(steps[i-1].After) {
			isTransitiveChain = false
			break
		}
	}
	for _, step := range steps {
		if step.Parameters != nil && step.Parameters["forall_param"] != nil {
			isTransitiveChain = false
			break
		}
	}

	var requiredImportsMap = map[string]bool{
		"Mathlib.Tactic.Ring": true,
	}

	type stepGoal struct {
		prop   string
		tactic string
	}
	var stepGoals []stepGoal
	var calcLines []string
	calcLines = append(calcLines, "calc")

	for i, step := range steps {
		tactic, err := v.SelectTactic(step)
		if err != nil {
			return "", err
		}
		if strings.Contains(tactic, "norm_num") {
			requiredImportsMap["Mathlib.Tactic.NormNum"] = true
		}
		if strings.Contains(tactic, "field_simp") {
			requiredImportsMap["Mathlib.Tactic.FieldSimp"] = true
		}
		if strings.Contains(tactic, "omega") {
			requiredImportsMap["Mathlib.Tactic.Omega"] = true
		}

		beforeStr, err := ToLeanSyntax(step.Before)
		if err != nil {
			return "", err
		}
		afterStr, err := ToLeanSyntax(step.After)
		if err != nil {
			return "", err
		}
		beforeFmt := formatCalcTerm(beforeStr)
		afterFmt := formatCalcTerm(afterStr)

		if isTransitiveChain {
			if i == 0 {
				calcLines = append(calcLines, fmt.Sprintf("  %s = %s := %s", beforeFmt, afterFmt, tactic))
			} else {
				calcLines = append(calcLines, fmt.Sprintf("  _ = %s := %s", afterFmt, tactic))
			}
		} else {
			prop := fmt.Sprintf("(%s = %s)", beforeFmt, afterFmt)
			if step.Parameters != nil {
				if pNode, ok := step.Parameters["forall_param"]; ok && pNode != nil {
					prop = fmt.Sprintf("(∀ (%s : ℤ), %s = %s)", escapeLeanIdent(pNode.String()), beforeFmt, afterFmt) // txthc-ignore
				}
			}
			stepGoals = append(stepGoals, stepGoal{prop: prop, tactic: tactic})
		}
	}

	// 3. Format imports
	var imports []string
	for imp := range requiredImportsMap {
		imports = append(imports, imp)
	}
	sort.Strings(imports)

	// 4. Collect free variables and determine type domain
	vars := CollectFreeVariables(allNodes...)
	typeAnnot := "ℚ" // txthc-ignore
	if containsRealNodes(allNodes...) {
		typeAnnot = "ℝ" // txthc-ignore
	} else if hasIntDomainRule && !hasRationalDomainRule {
		typeAnnot = "ℤ" // txthc-ignore
	}

	// Exclude bound forall_param variables from module-level variable declarations
	boundVars := make(map[string]bool)
	for _, step := range steps {
		if step.Parameters != nil {
			if pNode, ok := step.Parameters["forall_param"]; ok && pNode != nil {
				boundVars[escapeLeanIdent(pNode.String())] = true
			}
		}
	}
	var freeModuleVars []string
	for _, vr := range vars {
		if !boundVars[vr] {
			freeModuleVars = append(freeModuleVars, vr)
		}
	}

	var b strings.Builder
	b.WriteString("-- Automatically generated by ihd (i-hate-decimal-calc) Lean 4 Transpiler\n")
	b.WriteString("-- Mathematical proof certificate certified with Skeptic's verification\n")
	for _, imp := range imports {
		b.WriteString(fmt.Sprintf("import %s\n", imp))
	}
	b.WriteString("\n")

	if len(freeModuleVars) > 0 {
		b.WriteString(fmt.Sprintf("variable (%s : %s)\n\n", strings.Join(freeModuleVars, " "), typeAnnot))
	}

	hypStr := ""
	if len(condBinders) > 0 {
		hypStr = " " + strings.Join(condBinders, " ")
	}

	// 5. Theorem definition
	if isTransitiveChain {
		firstBefore, err := ToLeanSyntax(steps[0].Before)
		if err != nil {
			return "", err
		}
		lastAfter, err := ToLeanSyntax(steps[len(steps)-1].After)
		if err != nil {
			return "", err
		}
		b.WriteString(fmt.Sprintf("theorem %s%s : %s = %s := by\n", escapeLeanIdent(theoremName), hypStr, formatCalcTerm(firstBefore), formatCalcTerm(lastAfter)))
		for _, line := range calcLines {
			b.WriteString("  ")
			b.WriteString(line)
			b.WriteByte('\n')
		}
	} else if len(stepGoals) == 1 {
		prop := strings.TrimPrefix(strings.TrimSuffix(stepGoals[0].prop, ")"), "(")
		b.WriteString(fmt.Sprintf("theorem %s%s : %s := %s\n", escapeLeanIdent(theoremName), hypStr, prop, stepGoals[0].tactic))
	} else {
		var props []string
		for _, g := range stepGoals {
			props = append(props, g.prop)
		}
		b.WriteString(fmt.Sprintf("theorem %s%s : %s := by\n", escapeLeanIdent(theoremName), hypStr, strings.Join(props, " ∧ "))) // txthc-ignore
		for idx, g := range stepGoals {
			if idx < len(stepGoals)-1 {
				b.WriteString("  constructor\n")
				b.WriteString(fmt.Sprintf("  · %s\n", strings.TrimPrefix(g.tactic, "by "))) // txthc-ignore
			} else {
				b.WriteString(fmt.Sprintf("  · %s\n", strings.TrimPrefix(g.tactic, "by "))) // txthc-ignore
			}
		}
	}

	// 6. Mechanically verifiable axiom verification
	b.WriteString("\n#print axioms ")
	b.WriteString(escapeLeanIdent(theoremName))
	b.WriteString("\n")

	return b.String(), nil
}
