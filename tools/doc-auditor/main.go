package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
)

// DocSpecs models the structure produced by reference-builder from mdast.
type DocSpecs struct {
	Version     string                     `json:"version"`
	GeneratedAt string                     `json:"generated_at"`
	Languages   map[string]LanguageDocSpec `json:"languages"`
}

type LanguageDocSpec struct {
	Functions []FunctionDocSpec `json:"functions"`
}

type FunctionDocSpec struct {
	Category    string   `json:"category"`
	Names       []string `json:"names"`
	Examples    []string `json:"examples"`
	Description string   `json:"description"`
}

type AuditIssue struct {
	Language string
	Category string
	Symbol   string
	Severity string // "ERROR", "WARNING"
	Type     string // "MISSING_REGISTRY", "GHOST_SYMBOL", "ARG_COUNT_MISMATCH", "PARSE_FAILURE", "EVAL_FAILURE"
	Message  string
	Example  string
}

// knownNonFunctionDocItems contains symbols legitimately appearing in reference tables
// that are REPL commands, CLI flags, syntax wildcards, or Unicode aliases.
var knownNonFunctionDocItems = map[string]bool{
	"√":               true, // Unicode alias for sqrt
	"_var":            true, // TRS pattern matching wildcard variable
	"[unit]":          true, // Dimensional analysis bracket syntax
	"clear":           true, // REPL command
	"exit":            true, // REPL command
	"quit":            true, // REPL command
	"help":            true, // REPL command
	"highlight":       true, // REPL command
	"highlight on":    true, // REPL command
	"off":             true, // REPL command
	"theme":           true, // REPL command
	"theme dark":      true, // REPL command
	"theme light":     true, // REPL command
	"save":            true, // REPL command
	"load":            true, // REPL command
	"similar":         true, // REPL command
	"vars":            true, // REPL command
	"whatis":          true, // REPL command
	"reset":           true, // REPL command
	"config loop":     true, // REPL command
	"warn-loop":       true, // REPL command
	"max-loop":        true, // REPL command
	"search":          true, // REPL command
	"ProofTrace":      true, // CLI verification concept
	"micro-verifier":  true, // CLI verification tool
	"steps":           true, // CLI explanation concept
	"explain":         true, // CLI flag / concept
	"--verify":        true, // CLI flag
	"--explain":       true, // CLI flag
	"--explain --lean":  true, // CLI flag
	"--explain --typst": true, // CLI flag
	"--typst":         true, // CLI flag
	"--format typst":  true, // CLI flag
	"--lean":          true, // CLI flag
	"--lean-file":     true, // CLI flag
	"--replay":        true, // CLI flag
	"--certificate":   true, // CLI flag
}

var optArgPattern = regexp.MustCompile(`,?\s*\[([a-zA-Z0-9_]+)\]`)

func main() {
	specsPath := filepath.Join("docs", "references", "dist", "doc_specs.json")
	bytes, err := os.ReadFile(specsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: unable to read %s: %v\nRun 'node tools/reference-builder/index.js' first.\n", specsPath, err)
		os.Exit(1)
	}

	var specs DocSpecs
	if err := json.Unmarshal(bytes, &specs); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to parse %s: %v\n", specsPath, err)
		os.Exit(1)
	}

	allRegistered := calc.AllFunctionNames()
	regMap := make(map[string]calc.FunctionSpec, len(allRegistered))
	for _, name := range allRegistered {
		if spec, ok := calc.LookupFunction(name); ok {
			regMap[name] = spec
			regMap[strings.ToLower(name)] = spec
		}
	}

	var issues []AuditIssue

	for _, lang := range []string{"ja", "en"} {
		lSpec, ok := specs.Languages[lang]
		if !ok {
			issues = append(issues, AuditIssue{
				Language: lang,
				Severity: "ERROR",
				Type:     "MISSING_LANGUAGE",
				Message:  fmt.Sprintf("Language edition '%s' missing from doc_specs.json", lang),
			})
			continue
		}

		docSymbols := make(map[string]bool)

		for _, fn := range lSpec.Functions {
			for _, name := range fn.Names {
				cleanName := strings.TrimSpace(name)
				docSymbols[cleanName] = true
				docSymbols[strings.ToLower(cleanName)] = true

				// Check for Ghost functions
				if !knownNonFunctionDocItems[cleanName] && !knownNonFunctionDocItems[strings.ToLower(cleanName)] {
					if _, found := regMap[cleanName]; !found {
						issues = append(issues, AuditIssue{
							Language: lang,
							Category: fn.Category,
							Symbol:   cleanName,
							Severity: "WARNING",
							Type:     "GHOST_SYMBOL",
							Message:  fmt.Sprintf("Symbol '%s' documented in table but not registered in registry.go", cleanName),
						})
					}
				}
			}

			// Validate syntax examples
			isReplCategory := strings.Contains(fn.Category, "REPL") || strings.Contains(fn.Category, "対話環境") || strings.Contains(fn.Category, "Interactive Commands")

			for _, example := range fn.Examples {
				ex := strings.TrimSpace(example)
				if ex == "" || strings.HasPrefix(ex, "ihd ") || strings.HasPrefix(ex, "--") || knownNonFunctionDocItems[ex] {
					continue
				}

				// Skip purely schematic REPL / micro-verifier CLI commands
				if isReplCategory || strings.HasPrefix(ex, "micro-verifier ") {
					continue
				}

				// Parse and validate arguments
				validateExample(lang, fn.Category, fn.Names, ex, regMap, &issues)
			}
		}

		// Check for missing registered functions
		for _, name := range allRegistered {
			if !docSymbols[name] && !docSymbols[strings.ToLower(name)] {
				issues = append(issues, AuditIssue{
					Language: lang,
					Symbol:   name,
					Severity: "ERROR",
					Type:     "MISSING_REGISTRY",
					Message:  fmt.Sprintf("Registered function '%s' is missing from %s reference documentation", name, lang),
				})
			}
		}
	}

	reportPath := filepath.Join("docs", "reference_audit_report.md")

	if len(issues) == 0 {
		if _, err := os.Stat(reportPath); err == nil {
			_ = os.Remove(reportPath)
			fmt.Printf("[Doc Auditor] Clean: No inconsistencies detected. Removed %s.\n", reportPath)
		} else {
			fmt.Println("[Doc Auditor] Clean: No inconsistencies detected across all reference editions.")
		}
		os.Exit(0)
	}

	// Format and output reference_audit_report.md
	generateAuditReport(reportPath, issues)
	fmt.Fprintf(os.Stderr, "[Doc Auditor] Failure: %d inconsistencies detected. See %s\n", len(issues), reportPath)
	os.Exit(1)
}

func validateExample(lang, category string, names []string, example string, regMap map[string]calc.FunctionSpec, issues *[]AuditIssue) {
	primaryName := names[0]

	// Normalize schematic expressions
	normalized := example
	// If it contains schematic ellipsis `...`, replace with a generic identifier
	if strings.Contains(normalized, "...") {
		normalized = strings.ReplaceAll(normalized, ", ...", "")
		normalized = strings.ReplaceAll(normalized, "...", "x")
	}

	// Try parsing
	node, err := calc.Parse(normalized)
	if err != nil {
		// Attempt parsing after stripping optional argument brackets like `[order]`
		withoutOpt := optArgPattern.ReplaceAllString(example, "")
		node, err = calc.Parse(withoutOpt)
	}

	if err != nil {
		*issues = append(*issues, AuditIssue{
			Language: lang,
			Category: category,
			Symbol:   primaryName,
			Severity: "ERROR",
			Type:     "PARSE_FAILURE",
			Example:  example,
			Message:  fmt.Sprintf("Failed to parse documented syntax example: %v", err),
		})
		return
	}

	// If parsed node is a FuncNode, verify argument count against registry
	if fnNode, ok := node.(*calc.FuncNode); ok {
		spec, found := regMap[fnNode.Name]
		if found {
			argCount := len(fnNode.Args)
			if spec.MinArgs == spec.MaxArgs {
				if argCount != spec.MinArgs {
					*issues = append(*issues, AuditIssue{
						Language: lang,
						Category: category,
						Symbol:   fnNode.Name,
						Severity: "ERROR",
						Type:     "ARG_COUNT_MISMATCH",
						Example:  example,
						Message:  fmt.Sprintf("Argument count mismatch: documented %d args, but registry requires exactly %d", argCount, spec.MinArgs),
					})
				}
			} else {
				if argCount < spec.MinArgs || (spec.MaxArgs != -1 && argCount > spec.MaxArgs) {
					*issues = append(*issues, AuditIssue{
						Language: lang,
						Category: category,
						Symbol:   fnNode.Name,
						Severity: "ERROR",
						Type:     "ARG_COUNT_MISMATCH",
						Example:  example,
						Message:  fmt.Sprintf("Argument count mismatch: documented %d args, but registry accepts %d..%d", argCount, spec.MinArgs, spec.MaxArgs),
					})
				}
			}
		}
	}

	// Light evaluation smoke test
	env := calc.NewEnv()
	_, evalErr := calc.EvalWithEnv(node, env)
	if evalErr != nil {
		errMsg := evalErr.Error()
		// Detect genuine documentation bugs (e.g. using 'i' as loop variable, syntax mismatch)
		if strings.Contains(errMsg, "imaginary unit") || strings.Contains(errMsg, "invalid digit") || strings.Contains(errMsg, "syntax error") {
			*issues = append(*issues, AuditIssue{
				Language: lang,
				Category: category,
				Symbol:   primaryName,
				Severity: "ERROR",
				Type:     "EVAL_FAILURE",
				Example:  example,
				Message:  fmt.Sprintf("Documentation example causes evaluation error: %v", evalErr),
			})
		}
	}
}

func generateAuditReport(reportPath string, issues []AuditIssue) {
	var sb strings.Builder
	sb.WriteString("# Reference & Documentation Inconsistency Audit Report\n\n")
	sb.WriteString("> [!WARNING]\n")
	sb.WriteString("> This report is automatically generated by `tools/doc-auditor`.\n")
	sb.WriteString("> It indicates discrepancies between `reference.*.md` and Go internal `registry.go` / engine behavior.\n")
	sb.WriteString("> Resolve the root causes in documentation or source code until this report is cleanly deleted.\n\n")

	sb.WriteString(fmt.Sprintf("Total Inconsistencies Detected: **%d**\n\n", len(issues)))

	// Group by Type
	byType := make(map[string][]AuditIssue)
	for _, issue := range issues {
		byType[issue.Type] = append(byType[issue.Type], issue)
	}

	typeOrder := []string{"MISSING_REGISTRY", "ARG_COUNT_MISMATCH", "PARSE_FAILURE", "EVAL_FAILURE", "GHOST_SYMBOL"}
	for _, t := range typeOrder {
		list, ok := byType[t]
		if !ok || len(list) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("## %s (%d items)\n\n", t, len(list)))
		sb.WriteString("| Language | Category | Symbol | Severity | Example | Details |\n")
		sb.WriteString("| :---: | :--- | :--- | :---: | :--- | :--- |\n")
		for _, item := range list {
			ex := item.Example
			if ex == "" {
				ex = "-"
			} else {
				ex = "`" + ex + "`"
			}
			cat := item.Category
			if cat == "" {
				cat = "-"
			}
			sb.WriteString(fmt.Sprintf("| `%s` | %s | `%s` | **%s** | %s | %s |\n",
				item.Language, cat, item.Symbol, item.Severity, ex, item.Message))
		}
		sb.WriteString("\n")
	}

	_ = os.WriteFile(reportPath, []byte(sb.String()), 0644)
}
