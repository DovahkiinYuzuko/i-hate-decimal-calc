package lint

import (
	"go/ast"
	"go/token"
)

// Severity classifies whether a diagnostic blocks execution or is advisory.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// RunMode controls whether linting checks staged changes or the entire repository.
type RunMode string

const (
	ModeStaged RunMode = "staged"
	ModeAll    RunMode = "all"
)

// Diagnostic represents a single structured finding with mandatory reason and remediation.
type Diagnostic struct {
	RuleID      string   `json:"rule_id"`
	Checker     string   `json:"checker"`
	Severity    Severity `json:"severity"`
	File        string   `json:"file"`
	Line        int      `json:"line,omitempty"`
	Col         int      `json:"col,omitempty"`
	Message     string   `json:"message"`
	Reason      string   `json:"reason"`
	Remediation string   `json:"remediation"`
}

// IgnoreDirective represents a parsed `// repo-lint:ignore <rule-id> <reason>` annotation.
type IgnoreDirective struct {
	RuleID     string
	Reason     string
	Line       int
	TargetLine int
}

// ASTFile holds a parsed Go source file shared across all registered ASTAnalyzers.
type ASTFile struct {
	RelPath string
	AbsPath string
	Fset    *token.FileSet
	File    *ast.File
	Source  []byte
	Ignores map[int][]IgnoreDirective
}

// GitClient abstracts Git queries for deterministic testing without invoking external git.
type GitClient interface {
	TrackedFiles(root string) ([]string, error)
	StagedFiles(root string) ([]string, error)
	StagedDiffUnified0(root, relPath string) (string, error)
}

// LintContext carries repository scope and cached artifacts shared across checkers.
type LintContext struct {
	RepoRoot     string
	Mode         RunMode
	Git          GitClient
	StagedFiles  []string
	TrackedFiles []string
	ASTFiles     []*ASTFile
	EnvLookup    func(string) string
}

// LookupEnv resolves an environment variable through the injectable EnvLookup seam.
func (c *LintContext) LookupEnv(key string) string {
	if c != nil && c.EnvLookup != nil {
		return c.EnvLookup(key)
	}
	return ""
}

// Checker executes a repository governance or static check.
type Checker interface {
	Name() string
	Description() string
	Run(ctx *LintContext) ([]Diagnostic, error)
}

// ASTAnalyzer is the plugin interface for Go AST rules (extended by issue-183 CAS rules).
type ASTAnalyzer interface {
	Name() string
	Description() string
	DefaultSeverity() Severity
	AnalyzeFile(ctx *LintContext, af *ASTFile) ([]Diagnostic, error)
}

// RuleInfo describes a registered checker or AST analyzer for --list-rules.
type RuleInfo struct {
	Name            string
	Kind            string
	DefaultSeverity Severity
	Description     string
}
