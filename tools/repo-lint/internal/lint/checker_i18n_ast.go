package lint

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// I18nASTChecker validates i18n locale JSON parity, bans %w in locales,
// verifies every i18n.T(...) call's key existence and fmt argument count via Go AST,
// and reports unused locale keys as warnings.
type I18nASTChecker struct{}

func NewI18nASTChecker() *I18nASTChecker { return &I18nASTChecker{} }

func (c *I18nASTChecker) Name() string { return "i18n-ast" }
func (c *I18nASTChecker) Description() string {
	return "Validates i18n locale parity, %w ban, Go AST i18n.T() key & format-arg alignment, and unused keys."
}

func (c *I18nASTChecker) Run(ctx *LintContext) ([]Diagnostic, error) {
	localesDir := filepath.Join(ctx.RepoRoot, "internal", "i18n", "locales")
	enPath := filepath.Join(localesDir, "en.json")
	jaPath := filepath.Join(localesDir, "ja.json")

	if _, err := os.Stat(enPath); os.IsNotExist(err) {
		return nil, nil
	}

	if ctx.Mode == ModeStaged && !hasRelevantI18nStage(ctx.StagedFiles) {
		return nil, nil
	}

	enMap, enErr := loadLocaleFlat(enPath)
	if enErr != nil {
		return []Diagnostic{{
			RuleID:      "i18n-locale-parse",
			Checker:     c.Name(),
			Severity:    SeverityError,
			File:        "internal/i18n/locales/en.json",
			Message:     "Failed to parse en.json: " + enErr.Error(),
			Reason:      "Malformed locale JSON breaks embedded i18n initialization.",
			Remediation: "Fix the JSON syntax error in `internal/i18n/locales/en.json`.",
		}}, nil
	}

	jaMap, jaErr := loadLocaleFlat(jaPath)
	if jaErr != nil {
		return []Diagnostic{{
			RuleID:      "i18n-locale-parse",
			Checker:     c.Name(),
			Severity:    SeverityError,
			File:        "internal/i18n/locales/ja.json",
			Message:     "Failed to parse ja.json: " + jaErr.Error(),
			Reason:      "Malformed locale JSON breaks embedded i18n initialization.",
			Remediation: "Fix the JSON syntax error in `internal/i18n/locales/ja.json`.",
		}}, nil
	}

	var diags []Diagnostic

	// 1. Key parity between en.json and ja.json
	for _, k := range sortedKeys(enMap) {
		if _, ok := jaMap[k]; !ok {
			diags = append(diags, Diagnostic{
				RuleID:      "i18n-locale-parity",
				Checker:     c.Name(),
				Severity:    SeverityError,
				File:        "internal/i18n/locales/ja.json",
				Message:     "Key `" + k + "` exists in en.json but is missing in ja.json",
				Reason:      "Locale dictionaries must maintain an identical key set so Japanese users do not see fallback English or raw keys.",
				Remediation: "Add `" + k + "` with a Japanese translation to `internal/i18n/locales/ja.json`.",
			})
		}
	}
	for _, k := range sortedKeys(jaMap) {
		if _, ok := enMap[k]; !ok {
			diags = append(diags, Diagnostic{
				RuleID:      "i18n-locale-parity",
				Checker:     c.Name(),
				Severity:    SeverityError,
				File:        "internal/i18n/locales/en.json",
				Message:     "Key `" + k + "` exists in ja.json but is missing in en.json",
				Reason:      "Locale dictionaries must maintain an identical key set across en.json and ja.json.",
				Remediation: "Add `" + k + "` with an English translation to `internal/i18n/locales/en.json`.",
			})
		}
	}

	// 2. Ban %w in locale values
	for _, pair := range []struct {
		rel string
		m   map[string]string
	}{
		{"internal/i18n/locales/en.json", enMap},
		{"internal/i18n/locales/ja.json", jaMap},
	} {
		for _, k := range sortedKeys(pair.m) {
			if strings.Contains(pair.m[k], "%w") {
				diags = append(diags, Diagnostic{
					RuleID:      "i18n-percent-w",
					Checker:     c.Name(),
					Severity:    SeverityError,
					File:        pair.rel,
					Message:     "Locale key `" + k + "` contains illegal `%w` verb",
					Reason:      "`i18n.T` formats strings with `fmt.Sprintf`, where `%w` is invalid and renders malformed `%!w(...)` output.",
					Remediation: "Replace `%w` with `%v` or `%s` in `" + pair.rel + "` under key `" + k + "`.",
				})
			}
		}
	}

	// 3. Go AST scan of cmd/ and internal/
	goFiles, err := collectGoFiles(ctx.RepoRoot, ctx.Mode, ctx.StagedFiles)
	if err != nil {
		return nil, err
	}

	usedKeys := make(map[string]bool)
	referencedStrings := make(map[string]bool)
	dynamicPrefixes := make(map[string]bool)

	for _, rel := range goFiles {
		abs := filepath.Join(ctx.RepoRoot, filepath.FromSlash(rel))
		src, err := os.ReadFile(abs)
		if err != nil {
			continue
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, rel, src, parser.ParseComments)
		if err != nil {
			continue
		}

		i18nAliases := findI18nImportAliases(file)
		fmtAliases := findFmtImportAliases(file)

		var stack []ast.Node
		ast.Inspect(file, func(n ast.Node) bool {
			if n == nil {
				if len(stack) > 0 {
					stack = stack[:len(stack)-1]
				}
				return false
			}
			defer func() { stack = append(stack, n) }()

			// Record string literals for table-driven key references (avoiding unused-key false positives).
			if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				if val, err := strconv.Unquote(lit.Value); err == nil && val != "" {
					referencedStrings[val] = true
				}
			}

			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "T" {
				return true
			}
			recvIdent, ok := sel.X.(*ast.Ident)
			if !ok || !i18nAliases[recvIdent.Name] {
				return true
			}
			if len(call.Args) == 0 {
				return true
			}

			keyExpr := call.Args[0]
			if lit, ok := keyExpr.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				key, err := strconv.Unquote(lit.Value)
				if err != nil {
					return true
				}
				usedKeys[key] = true
				pos := fset.Position(call.Pos())

				enVal, enOK := enMap[key]
				jaVal, jaOK := jaMap[key]
				if !enOK || !jaOK {
					missing := "en.json and ja.json"
					if enOK && !jaOK {
						missing = "ja.json"
					} else if !enOK && jaOK {
						missing = "en.json"
					}
					diags = append(diags, Diagnostic{
						RuleID:      "i18n-missing-key",
						Checker:     c.Name(),
						Severity:    SeverityError,
						File:        rel,
						Line:        pos.Line,
						Col:         pos.Column,
						Message:     "i18n.T key `" + key + "` is not defined in " + missing,
						Reason:      "Calling `i18n.T` with an undefined key silently falls back to printing the raw key string to users.",
						Remediation: "Define `" + key + "` in both `internal/i18n/locales/en.json` and `ja.json`, or fix the typo in the key name.",
					})
					return true
				}

				// Skip variadic forwarding `i18n.T(key, args...)` where argument count is dynamic.
				if call.Ellipsis != token.NoPos {
					return true
				}

				actualArgs := len(call.Args) - 1
				// Support `fmt.Fprintf(w, i18n.T("key")+"\n", a, b)` and `fmt.Printf(i18n.T("key"), a)` patterns.
				if actualArgs == 0 {
					if outerArgs, variadic, extraVerbs, ok := enclosingFmtFormatCallArgs(stack, call, fmtAliases); ok {
						if variadic {
							return true
						}
						enExpected := countFmtArgs(enVal) + extraVerbs
						jaExpected := countFmtArgs(jaVal) + extraVerbs
						if outerArgs != enExpected || outerArgs != jaExpected {
							diags = append(diags, Diagnostic{
								RuleID:      "i18n-format-args",
								Checker:     c.Name(),
								Severity:    SeverityError,
								File:        rel,
								Line:        pos.Line,
								Col:         pos.Column,
								Message:     fmt.Sprintf("fmt.*f(i18n.T(%q), ...) passes %d format arg(s), but templates expect en=%d / ja=%d", key, outerArgs, enExpected, jaExpected),
								Reason:      "Mismatched format specifiers and arguments produce `%!(EXTRA ...)` or `%!s(MISSING)` artifacts in CLI output.",
								Remediation: "Align the format verbs in `en.json` and `ja.json` for `" + key + "` with the argument count passed at `" + fmt.Sprintf("%s:%d", rel, pos.Line) + "`.",
							})
						}
						return true
					}
				}

				enExpected := countFmtArgs(enVal)
				jaExpected := countFmtArgs(jaVal)

				if actualArgs != enExpected || actualArgs != jaExpected {
					diags = append(diags, Diagnostic{
						RuleID:      "i18n-format-args",
						Checker:     c.Name(),
						Severity:    SeverityError,
						File:        rel,
						Line:        pos.Line,
						Col:         pos.Column,
						Message:     fmt.Sprintf("i18n.T(%q) passes %d format arg(s), but templates expect en=%d / ja=%d", key, actualArgs, enExpected, jaExpected),
						Reason:      "Mismatched format specifiers and arguments produce `%!(EXTRA ...)` or `%!s(MISSING)` artifacts in CLI output.",
						Remediation: "Align the format verbs in `en.json` and `ja.json` for `" + key + "` with the argument count passed at `" + fmt.Sprintf("%s:%d", rel, pos.Line) + "`.",
					})
				}
				return true
			}

			// Detect dynamic prefix patterns like `i18n.T("prefix." + var)` or `i18n.T(fmt.Sprintf("prefix.%s", ...))`.
			if prefix := extractDynamicPrefix(keyExpr); prefix != "" {
				dynamicPrefixes[prefix] = true
			}
			return true
		})
	}

	// 4. Unused key warning in full mode
	if ctx.Mode == ModeAll {
		for _, k := range sortedKeys(enMap) {
			if strings.HasPrefix(k, "meta.") {
				continue
			}
			if usedKeys[k] || referencedStrings[k] {
				continue
			}
			matchedPrefix := false
			for pref := range dynamicPrefixes {
				if strings.HasPrefix(k, pref) {
					matchedPrefix = true
					break
				}
			}
			if matchedPrefix {
				continue
			}
			// Also check if a shorter suffix is dynamically prefixed in referenced strings
			if isLikelyDynamicTableKey(k, referencedStrings) {
				continue
			}
			diags = append(diags, Diagnostic{
				RuleID:      "i18n-unused-key",
				Checker:     c.Name(),
				Severity:    SeverityWarning,
				File:        "internal/i18n/locales/en.json",
				Message:     "Locale key `" + k + "` is not referenced in cmd/ or internal/",
				Reason:      "Dead translation keys bloat embedded locale bundles and create unnecessary maintenance burden.",
				Remediation: "Remove `" + k + "` from both `en.json` and `ja.json` if obsolete, or verify whether it is constructed dynamically.",
			})
		}
	}

	return diags, nil
}

func hasRelevantI18nStage(staged []string) bool {
	for _, f := range staged {
		if strings.HasPrefix(f, "internal/i18n/locales/") || isProdGoFile(f) {
			return true
		}
	}
	return false
}

func loadLocaleFlat(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	flat := make(map[string]string)
	var walk func(prefix string, m map[string]any)
	walk = func(prefix string, m map[string]any) {
		for k, v := range m {
			fk := k
			if prefix != "" {
				fk = prefix + "." + k
			}
			switch val := v.(type) {
			case string:
				flat[fk] = val
			case map[string]any:
				walk(fk, val)
			}
		}
	}
	walk("", raw)
	return flat, nil
}

func collectGoFiles(absRoot string, mode RunMode, staged []string) ([]string, error) {
	if mode == ModeStaged {
		var out []string
		for _, f := range staged {
			if isProdGoFile(f) {
				out = append(out, f)
			}
		}
		sort.Strings(out)
		return out, nil
	}

	var out []string
	for _, dir := range []string{"cmd", "internal"} {
		base := filepath.Join(absRoot, dir)
		if _, err := os.Stat(base); os.IsNotExist(err) {
			continue
		}
		err := filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			rel, err := filepath.Rel(absRoot, p)
			if err != nil {
				return err
			}
			relSlash := filepath.ToSlash(rel)
			if isProdGoFile(relSlash) {
				out = append(out, relSlash)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(out)
	return out, nil
}

func findI18nImportAliases(file *ast.File) map[string]bool {
	aliases := map[string]bool{"i18n": true}
	for _, imp := range file.Imports {
		val, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			continue
		}
		if strings.HasSuffix(val, "/internal/i18n") || val == "internal/i18n" {
			if imp.Name != nil && imp.Name.Name != "_" && imp.Name.Name != "." {
				aliases[imp.Name.Name] = true
			}
		}
	}
	return aliases
}

func findFmtImportAliases(file *ast.File) map[string]bool {
	aliases := map[string]bool{"fmt": true}
	for _, imp := range file.Imports {
		val, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			continue
		}
		if val == "fmt" {
			if imp.Name != nil && imp.Name.Name != "_" && imp.Name.Name != "." {
				aliases[imp.Name.Name] = true
			}
		}
	}
	return aliases
}

// enclosingFmtFormatCallArgs inspects ancestors of `i18nCall` to detect patterns like:
//   - fmt.Printf(i18n.T("k"), a)
//   - fmt.Fprintf(w, i18n.T("k")+"\n", a, b)
//   - fmt.Errorf(i18n.T("k"), a)
//   - fmt.Sprintf(i18n.T("k"), a)
//   - NewCalcError(..., i18n.T("k"), ...) / errHelper(i18n.T("k"), a) where outer call formats
func enclosingFmtFormatCallArgs(stack []ast.Node, i18nCall *ast.CallExpr, fmtAliases map[string]bool) (argCount int, variadic bool, extraVerbs int, ok bool) {
	// Walk up through string concatenation binary expressions (e.g. `i18n.T("k") + "\n"`).
	cur := ast.Node(i18nCall)
	extra := 0
	for i := len(stack) - 1; i >= 0; i-- {
		parent := stack[i]
		if bin, isBin := parent.(*ast.BinaryExpr); isBin && bin.Op == token.ADD {
			// Count any format verbs in sibling string literals concatenated to the template.
			var other ast.Expr
			if bin.X == cur {
				other = bin.Y
			} else if bin.Y == cur {
				other = bin.X
			}
			if lit, isLit := other.(*ast.BasicLit); isLit && lit.Kind == token.STRING {
				if val, err := strconv.Unquote(lit.Value); err == nil {
					extra += countFmtArgs(val)
				}
			}
			cur = bin
			continue
		}
		if outerCall, isCall := parent.(*ast.CallExpr); isCall {
			sel, isSel := outerCall.Fun.(*ast.SelectorExpr)
			if !isSel {
				return 0, false, 0, false
			}
			recv, isIdent := sel.X.(*ast.Ident)
			if !isIdent || !fmtAliases[recv.Name] {
				return 0, false, 0, false
			}
			fmtArgIdx := -1
			switch sel.Sel.Name {
			case "Printf", "Sprintf", "Errorf":
				fmtArgIdx = 0
			case "Fprintf":
				fmtArgIdx = 1
			default:
				return 0, false, 0, false
			}
			if len(outerCall.Args) > fmtArgIdx && outerCall.Args[fmtArgIdx] == cur {
				return len(outerCall.Args) - fmtArgIdx - 1, outerCall.Ellipsis != token.NoPos, extra, true
			}
		}
		break
	}
	return 0, false, 0, false
}

func extractDynamicPrefix(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.BinaryExpr:
		if e.Op == token.ADD {
			if lit, ok := e.X.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				if val, err := strconv.Unquote(lit.Value); err == nil {
					return val
				}
			}
		}
	case *ast.CallExpr:
		if sel, ok := e.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Sprintf" && len(e.Args) >= 1 {
			if lit, ok := e.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
				if val, err := strconv.Unquote(lit.Value); err == nil {
					if idx := strings.Index(val, "%"); idx > 0 {
						return val[:idx]
					}
				}
			}
		}
	}
	return ""
}

func isLikelyDynamicTableKey(key string, refs map[string]bool) bool {
	// If code holds a dot-prefix literal (e.g. "cli.help.") or suffix used in dynamic lookup, avoid noisy warnings.
	parts := strings.Split(key, ".")
	if len(parts) >= 2 {
		prefix := parts[0] + "."
		if refs[prefix] {
			return true
		}
		if len(parts) >= 3 && refs[parts[0]+"."+parts[1]+"."] {
			return true
		}
	}
	return false
}

// countFmtArgs counts required arguments for a Go fmt.Sprintf format template.
// Handles escaped `%%`, width/precision `*`, and explicit positional verbs `%[n]s`.
func countFmtArgs(tmpl string) int {
	maxArg := 0
	nextArg := 1
	n := len(tmpl)

	for i := 0; i < n; i++ {
		if tmpl[i] != '%' {
			continue
		}
		i++
		if i >= n {
			break
		}
		if tmpl[i] == '%' {
			continue // literal %%
		}

		// Parse flags
		for i < n && strings.ContainsRune("#+- 0", rune(tmpl[i])) {
			i++
		}

		// Optional explicit argument index [n]
		explicitIdx := 0
		if i < n && tmpl[i] == '[' {
			if idx, end, ok := parseBracketIndex(tmpl, i); ok {
				explicitIdx = idx
				i = end
			}
		}

		// Width: * or digits
		if i < n && tmpl[i] == '*' {
			if explicitIdx > 0 {
				if explicitIdx > maxArg {
					maxArg = explicitIdx
				}
				nextArg = explicitIdx + 1
				explicitIdx = 0
			} else {
				if nextArg > maxArg {
					maxArg = nextArg
				}
				nextArg++
			}
			i++
		} else {
			for i < n && tmpl[i] >= '0' && tmpl[i] <= '9' {
				i++
			}
		}

		// Precision
		if i < n && tmpl[i] == '.' {
			i++
			if i < n && tmpl[i] == '[' {
				if idx, end, ok := parseBracketIndex(tmpl, i); ok {
					explicitIdx = idx
					i = end
				}
			}
			if i < n && tmpl[i] == '*' {
				if explicitIdx > 0 {
					if explicitIdx > maxArg {
						maxArg = explicitIdx
					}
					nextArg = explicitIdx + 1
					explicitIdx = 0
				} else {
					if nextArg > maxArg {
						maxArg = nextArg
					}
					nextArg++
				}
				i++
			} else {
				for i < n && tmpl[i] >= '0' && tmpl[i] <= '9' {
					i++
				}
			}
		}

		// Post-width/precision explicit index [n] before verb
		if i < n && tmpl[i] == '[' {
			if idx, end, ok := parseBracketIndex(tmpl, i); ok {
				explicitIdx = idx
				i = end
			}
		}

		if i < n && strings.ContainsRune("vTtbcdoOqxXUeEfFgGspw", rune(tmpl[i])) {
			if explicitIdx > 0 {
				if explicitIdx > maxArg {
					maxArg = explicitIdx
				}
				nextArg = explicitIdx + 1
			} else {
				if nextArg > maxArg {
					maxArg = nextArg
				}
				nextArg++
			}
		}
	}
	return maxArg
}

func parseBracketIndex(s string, start int) (int, int, bool) {
	if start >= len(s) || s[start] != '[' {
		return 0, start, false
	}
	end := strings.IndexByte(s[start+1:], ']')
	if end < 0 {
		return 0, start, false
	}
	numStr := s[start+1 : start+1+end]
	val, err := strconv.Atoi(numStr)
	if err != nil || val <= 0 {
		return 0, start, false
	}
	return val, start + 1 + end + 1, true
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
