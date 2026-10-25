package calc

import (
	"fmt"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// DiffStatus represents the semantic comparison status between two mathematical expressions or statements.
type DiffStatus string

const (
	// DiffIdentical indicates both syntax and semantics are identical.
	DiffIdentical DiffStatus = "Identical"
	// DiffEquivalent indicates expressions differ syntactically (e.g. factored vs expanded) but are algebraically equivalent.
	DiffEquivalent DiffStatus = "SemanticallyEquivalent"
	// DiffChanged indicates mathematical semantics have changed.
	DiffChanged DiffStatus = "Changed"
	// DiffAdded indicates a statement was added in the right script.
	DiffAdded DiffStatus = "Added"
	// DiffRemoved indicates a statement was removed from the left script.
	DiffRemoved DiffStatus = "Removed"
)

// StatementDiff holds the comparison result for an individual script statement.
type StatementDiff struct {
	Index    int
	Status   DiffStatus
	LeftRaw  string
	RightRaw string
	Details  string
}

// CompareExpressions evaluates two AST nodes to determine if they are identical, semantically equivalent, or changed.
func CompareExpressions(left, right ast.Node, env *Env) (DiffStatus, error) {
	if left == nil && right == nil {
		return DiffIdentical, nil
	}
	if left == nil || right == nil {
		return DiffChanged, nil
	}

	// 1. Direct structural equality check via Node.Equal
	if left.Equal(right) {
		return DiffIdentical, nil
	}

	if env == nil {
		env = NewEnv()
	}

	// 2. Evaluate both expressions
	leftEval, err := EvalWithEnv(left, env)
	if err != nil {
		leftEval = left
	}
	rightEval, err := EvalWithEnv(right, env)
	if err != nil {
		rightEval = right
	}

	if leftEval.Equal(rightEval) {
		return DiffEquivalent, nil
	}

	// 3. Algebraic zero check: checkIsZeroAlgebraically(left - right)
	negRight := &ast.UnaryOpNode{Op: "-", Expr: rightEval}
	subExpr := ast.NewAdd([]ast.Node{leftEval, negRight})

	if checkIsZeroAlgebraically(subExpr, env) {
		return DiffEquivalent, nil
	}

	return DiffChanged, nil
}

// CompareScripts compares two multiline script contents line by line / statement by statement.
func CompareScripts(leftContent, rightContent string, env *Env) ([]StatementDiff, error) {
	if env == nil {
		env = NewEnv()
	}

	leftLines := splitScriptLines(leftContent)
	rightLines := splitScriptLines(rightContent)

	maxLen := len(leftLines)
	if len(rightLines) > maxLen {
		maxLen = len(rightLines)
	}

	var results []StatementDiff

	for i := 0; i < maxLen; i++ {
		var leftRaw, rightRaw string
		if i < len(leftLines) {
			leftRaw = leftLines[i]
		}
		if i < len(rightLines) {
			rightRaw = rightLines[i]
		}

		diff := StatementDiff{
			Index:    i + 1,
			LeftRaw:  leftRaw,
			RightRaw: rightRaw,
		}

		if leftRaw == "" && rightRaw != "" {
			diff.Status = DiffAdded
			diff.Details = i18n.T("cli.diff_stat_added")
			results = append(results, diff)
			continue
		}
		if leftRaw != "" && rightRaw == "" {
			diff.Status = DiffRemoved
			diff.Details = i18n.T("cli.diff_stat_removed")
			results = append(results, diff)
			continue
		}
		if leftRaw == rightRaw {
			diff.Status = DiffIdentical
			diff.Details = i18n.T("cli.diff_stat_identical")
			results = append(results, diff)
			continue
		}

		// Handle variable assignments: name = expr
		leftVar, leftVal, isAssignL := parseAssignment(leftRaw)
		rightVar, rightVal, isAssignR := parseAssignment(rightRaw)

		if isAssignL && isAssignR {
			if leftVar != rightVar {
				diff.Status = DiffChanged
				diff.Details = i18n.T("cli.diff_stat_changed")
			} else {
				nodeL, errL := Parse(leftVal)
				nodeR, errR := Parse(rightVal)
				if errL != nil || errR != nil {
					diff.Status = DiffChanged
					diff.Details = i18n.T("cli.diff_stat_changed")
				} else {
					st, cmpErr := CompareExpressions(nodeL, nodeR, env)
					if cmpErr != nil || st == DiffChanged {
						diff.Status = DiffChanged
						diff.Details = i18n.T("cli.diff_stat_changed")
					} else if st == DiffEquivalent {
						diff.Status = DiffEquivalent
						diff.Details = fmt.Sprintf("%s (%s = %s)", i18n.T("cli.diff_stat_equivalent"), leftRaw, rightRaw)
					} else {
						diff.Status = DiffIdentical
						diff.Details = i18n.T("cli.diff_stat_identical")
					}
				}
			}
			results = append(results, diff)
			continue
		}

		// Parse expressions if possible
		leftNode, errL := Parse(leftRaw)
		rightNode, errR := Parse(rightRaw)

		if errL != nil || errR != nil {
			// If not parseable as pure expressions, treat as text changed
			diff.Status = DiffChanged
			diff.Details = i18n.T("cli.diff_stat_changed")
			results = append(results, diff)
			continue
		}

		st, cmpErr := CompareExpressions(leftNode, rightNode, env)
		if cmpErr != nil || st == DiffChanged {
			diff.Status = DiffChanged
			diff.Details = i18n.T("cli.diff_stat_changed")
		} else if st == DiffEquivalent {
			diff.Status = DiffEquivalent
			diff.Details = fmt.Sprintf("%s (%s = %s)", i18n.T("cli.diff_stat_equivalent"), leftRaw, rightRaw)
		} else {
			diff.Status = DiffIdentical
			diff.Details = i18n.T("cli.diff_stat_identical")
		}

		results = append(results, diff)
	}

	return results, nil
}

func parseAssignment(line string) (string, string, bool) {
	if idx := strings.Index(line, "="); idx > 0 {
		left := strings.TrimSpace(line[:idx])
		right := strings.TrimSpace(line[idx+1:])
		// Ensure left is a valid identifier
		if !strings.ContainsAny(left, "+-*/^%()[]{},><=!") && len(left) > 0 {
			return left, right, true
		}
	}
	return "", "", false
}

func splitScriptLines(content string) []string {
	var lines []string
	raw := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	for _, l := range raw {
		trimmed := strings.TrimSpace(l)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
			continue
		}
		lines = append(lines, trimmed)
	}
	return lines
}
