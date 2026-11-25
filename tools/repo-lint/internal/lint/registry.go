package lint

// DefaultRunner builds the standard repository linter with all built-in checkers and AST analyzers.
// Future issues (such as issue-183 CAS AST rules) register additional analyzers via RegisterASTAnalyzer.
func DefaultRunner() *Runner {
	r := NewRunner()
	r.RegisterChecker(NewGitWhitelistChecker())
	r.RegisterChecker(NewI18nASTChecker())
	r.RegisterChecker(NewNoticeSyncChecker())
	r.RegisterChecker(NewReadmeQuickStartChecker())
	r.RegisterChecker(NewLeanPinningChecker())
	RegisterGoHygieneAnalyzers(r)
	r.RegisterChecker(NewE2ECSVChecker())
	return r
}
