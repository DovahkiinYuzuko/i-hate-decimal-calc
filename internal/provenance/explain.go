package provenance

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// ExplainVisitor projects a derivation tree into human-readable pedagogical explanations or JSON.
type ExplainVisitor struct{}

// NewExplainVisitor creates a new ExplainVisitor.
func NewExplainVisitor() *ExplainVisitor {
	return &ExplainVisitor{}
}

// FormattedStep represents a flattened, pedagogical human-readable step.
type FormattedStep struct {
	StepNumber  int              `json:"step_number"`
	RuleID      string           `json:"rule_id"`
	Title       string           `json:"title"`
	Relation    RelationKind     `json:"relation"`
	Before      string           `json:"before"`
	After       string           `json:"after"`
	Conditions  []string         `json:"conditions,omitempty"`
	Certificate string           `json:"certificate,omitempty"`
	SubSteps    []*FormattedStep `json:"sub_steps,omitempty"`
}

// TreeJSON represents the full serialized derivation tree for JSON export.
type TreeJSON struct {
	InputExpr string           `json:"input_expression,omitempty"`
	Result    string           `json:"result"`
	Steps     []*FormattedStep `json:"steps"`
}

// RenderTree renders a derivation tree into a 2D indented Unicode tree.
func (v *ExplainVisitor) RenderTree(node *DerivationNode, inputExpr string) string {
	var sb strings.Builder

	if inputExpr != "" {
		sb.WriteString(i18n.T("trace.expression_header", inputExpr))
	} else if node != nil && node.Before != nil {
		sb.WriteString(i18n.T("trace.expression_header", node.Before.String()))
	}

	if node == nil {
		sb.WriteString(fmt.Sprintf("├── [%s]\n", i18n.T("trace.step_0_title")))
		sb.WriteString(fmt.Sprintf("│   %s\n", i18n.T("trace.step_0_desc")))
		sb.WriteString(fmt.Sprintf("└── %s\n    = %s\n", i18n.T("trace.result_header"), inputExpr))
		return sb.String()
	}

	steps := v.collectPresentableSteps(node)

	if len(steps) == 0 {
		sb.WriteString(fmt.Sprintf("├── [%s]\n", i18n.T("trace.step_0_title")))
		sb.WriteString(fmt.Sprintf("│   %s\n", i18n.T("trace.step_0_desc")))
	} else {
		for i, s := range steps {
			isLast := (i == len(steps)-1)
			v.renderStepTree(&sb, s, "", isLast)
		}
	}

	resStr := ""
	if node.After != nil {
		resStr = node.After.String()
	} else if len(steps) > 0 {
		resStr = steps[len(steps)-1].After
	}

	sb.WriteString(fmt.Sprintf("└── %s\n", i18n.T("trace.result_header")))
	sb.WriteString(fmt.Sprintf("    = %s", resStr))

	return sb.String()
}

func (v *ExplainVisitor) renderStepTree(sb *strings.Builder, s *FormattedStep, prefix string, isLast bool) {
	connector := "├── "
	childPrefix := prefix + "│   "

	sb.WriteString(fmt.Sprintf("%s%s%s\n", prefix, connector, i18n.T("trace.step_header", s.StepNumber, s.Title)))
	sb.WriteString(fmt.Sprintf("%s│   %s %s %s\n", prefix, s.Before, relationSymbol(s.Relation), s.After))

	for _, cond := range s.Conditions {
		sb.WriteString(fmt.Sprintf("%s│   [%s: %s]\n", prefix, i18n.T("explain.label_requires"), cond))
	}

	if s.Certificate != "" {
		sb.WriteString(fmt.Sprintf("%s│   [%s: %s]\n", prefix, i18n.T("explain.label_certificate"), s.Certificate))
	}

	for j, sub := range s.SubSteps {
		subLast := (j == len(s.SubSteps)-1)
		v.renderStepTree(sb, sub, childPrefix, subLast)
	}
}

// RenderJSON serializes the derivation tree into indented JSON format.
func (v *ExplainVisitor) RenderJSON(node *DerivationNode, inputExpr string) ([]byte, error) {
	steps := v.collectPresentableSteps(node)
	resStr := ""
	if node != nil && node.After != nil {
		resStr = node.After.String()
	} else if len(steps) > 0 {
		resStr = steps[len(steps)-1].After
	}

	out := TreeJSON{
		InputExpr: inputExpr,
		Result:    resStr,
		Steps:     steps,
	}

	return json.MarshalIndent(out, "", "  ")
}

func (v *ExplainVisitor) collectPresentableSteps(node *DerivationNode) []*FormattedStep {
	if node == nil {
		return nil
	}

	var results []*FormattedStep
	stepNum := 1

	var walk func(n *DerivationNode, parentStep *FormattedStep)
	walk = func(n *DerivationNode, parentStep *FormattedStep) {
		if n == nil {
			return
		}

		if n.Vis == VisStructural {
			// Fold structural steps into children walk
			for _, ch := range n.Children {
				walk(ch, parentStep)
			}
			return
		}

		// Presentable milestone
		step := &FormattedStep{
			StepNumber: stepNum,
			RuleID:     n.RuleID,
			Title:      resolveRuleTitle(n.RuleID),
			Relation:   n.Relation,
			Before:     nodeString(n.Before),
			After:      nodeString(n.After),
		}
		stepNum++

		for _, c := range n.Conditions {
			step.Conditions = append(step.Conditions, fmt.Sprintf("%s: %s (%v)", c.Kind, c.Description, c.Target))
		}

		if n.Certificate != nil {
			step.Certificate = n.Certificate.Summary()
		}

		for _, ch := range n.Children {
			walk(ch, step)
		}

		if parentStep != nil {
			parentStep.SubSteps = append(parentStep.SubSteps, step)
		} else {
			results = append(results, step)
		}
	}

	// If root has children, walk children; otherwise walk root
	if len(node.Children) > 0 {
		for _, ch := range node.Children {
			walk(ch, nil)
		}
	} else {
		walk(node, nil)
	}

	return results
}

func resolveRuleTitle(ruleID string) string {
	normalized := strings.ToLower(strings.ReplaceAll(ruleID, ".", "_"))
	key := "explain.rule_" + normalized
	translated := i18n.T(key)
	if translated != key {
		return translated
	}

	// Fallback mappings
	switch ruleID {
	case RuleAlgebraExpand:
		return i18n.T("explain.rule_algebra_expand")
	case RuleAlgebraFactor:
		return i18n.T("explain.rule_algebra_factor")
	case RuleAlgebraApart:
		return i18n.T("explain.rule_algebra_apart")
	case RuleAlgebraTogether:
		return i18n.T("explain.rule_algebra_together")
	case RuleAlgebraNormalize:
		return i18n.T("explain.rule_algebra_normalize")
	case "calculus.diff":
		return i18n.T("explain.rule_diffchain")
	case "calculus.integrate":
		return i18n.T("explain.rule_integrate")
	case "solver.solve":
		return i18n.T("explain.rule_solvelinear")
	}

	return ruleID
}

func relationSymbol(rel RelationKind) string {
	switch rel {
	case RelationEq:
		return "="
	case RelationIff:
		return "↔"
	case RelationImplies:
		return "→"
	case RelationLe:
		return "≤"
	case RelationLt:
		return "<"
	case RelationEquivalentModuloC:
		return "= mod C"
	default:
		return "="
	}
}

func nodeString(n ast.Node) string {
	if n == nil {
		return ""
	}
	return n.String()
}
