package lint

import (
	"go/ast"
	"go/token"
	"strings"
)

const ignorePrefix = "repo-lint:ignore"

// ParseIgnoreDirectives scans Go AST comments for `// repo-lint:ignore <rule-id> <reason>`.
// Standalone comment lines suppress the immediately following line (Line+1);
// inline end-of-line comments suppress only their own line (Line) so they never bleed into subsequent statements.
// Directives omitting a justification reason are rejected and returned as error diagnostics.
func ParseIgnoreDirectives(relPath string, fset *token.FileSet, file *ast.File, src []byte) (map[int][]IgnoreDirective, []Diagnostic) {
	directives := make(map[int][]IgnoreDirective)
	var invalid []Diagnostic

	if file == nil || fset == nil {
		return directives, invalid
	}

	lines := strings.Split(strings.ReplaceAll(string(src), "\r\n", "\n"), "\n")

	for _, cg := range file.Comments {
		for _, c := range cg.List {
			text := strings.TrimSpace(c.Text)
			if strings.HasPrefix(text, "//") {
				text = strings.TrimSpace(strings.TrimPrefix(text, "//"))
			} else if strings.HasPrefix(text, "/*") {
				text = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(text, "/*"), "*/"))
			}

			idx := strings.Index(text, ignorePrefix)
			if idx < 0 {
				continue
			}

			rest := strings.TrimSpace(text[idx+len(ignorePrefix):])
			pos := fset.Position(c.Pos())

			fields := strings.Fields(rest)
			if len(fields) < 2 {
				ruleHint := "<rule-id>"
				if len(fields) == 1 {
					ruleHint = fields[0]
				}
				invalid = append(invalid, Diagnostic{
					RuleID:      "repo-lint-ignore-missing-reason",
					Checker:     "ast-analyzers",
					Severity:    SeverityError,
					File:        relPath,
					Line:        pos.Line,
					Col:         pos.Column,
					Message:     "repo-lint:ignore directive is missing a mandatory justification reason",
					Reason:      "Suppressing lint rules without an explicit reason hides intentional trade-offs and enables silent regressions.",
					Remediation: "Write `// repo-lint:ignore " + ruleHint + " <concrete reason>` or remove the ignore comment and fix the underlying issue.",
				})
				continue
			}

			ruleID := fields[0]
			reason := strings.TrimSpace(strings.TrimPrefix(rest, ruleID))

			targetLine := pos.Line
			if pos.Line >= 1 && pos.Line <= len(lines) {
				trimmedLine := strings.TrimSpace(lines[pos.Line-1])
				if strings.HasPrefix(trimmedLine, "//") || strings.HasPrefix(trimmedLine, "/*") {
					targetLine = pos.Line + 1
				}
			}

			dir := IgnoreDirective{
				RuleID:     ruleID,
				Reason:     reason,
				Line:       pos.Line,
				TargetLine: targetLine,
			}
			directives[targetLine] = append(directives[targetLine], dir)
		}
	}

	return directives, invalid
}

// IsSuppressed checks whether ruleID is suppressed at targetLine.
func IsSuppressed(directives map[int][]IgnoreDirective, targetLine int, ruleID string) bool {
	if len(directives) == 0 || targetLine <= 0 {
		return false
	}
	for _, d := range directives[targetLine] {
		if d.RuleID == ruleID || d.RuleID == "all" {
			return true
		}
	}
	return false
}
