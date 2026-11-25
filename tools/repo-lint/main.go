package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/tools/repo-lint/internal/lint"
)

func main() {
	os.Exit(runCLI(os.Args[1:], os.Stdout, os.Stderr))
}

func runCLI(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("repo-lint", flag.ContinueOnError)
	fs.SetOutput(stderr)

	staged := fs.Bool("staged", false, "Run fast checks on git staged files only (for pre-commit)")
	all := fs.Bool("all", false, "Run full repository checks (default)")
	only := fs.String("only", "", "Comma-separated list of checkers/analyzers to run")
	skip := fs.String("skip", "", "Comma-separated list of checkers/analyzers to skip")
	root := fs.String("root", ".", "Repository root directory")
	format := fs.String("format", "text", "Output format: text or json")
	warnAsError := fs.Bool("warn-as-error", false, "Treat warnings as errors (exit 1)")
	listRules := fs.Bool("list-rules", false, "List all registered checkers and AST analyzers")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	runner := lint.DefaultRunner()

	if *listRules {
		var sb strings.Builder
		sb.WriteString("Registered repo-lint checkers & AST analyzers:\n")
		for _, info := range runner.Rules() {
			sb.WriteString(fmt.Sprintf("  - %-30s [%-12s] default=%-7s %s\n", info.Name, info.Kind, info.DefaultSeverity, info.Description))
		}
		_, _ = io.WriteString(stdout, sb.String())
		return 0
	}

	mode := lint.ModeAll
	if *staged && !*all {
		mode = lint.ModeStaged
	}

	res, err := runner.Run(lint.RunOptions{
		RepoRoot: *root,
		Mode:     mode,
		Only:     splitCSV(*only),
		Skip:     splitCSV(*skip),
	})
	if err != nil {
		_, _ = io.WriteString(stderr, "[repo-lint] internal error: "+err.Error()+"\n")
		return 2
	}

	switch strings.ToLower(strings.TrimSpace(*format)) {
	case "json":
		out, err := lint.FormatJSON(res)
		if err != nil {
			_, _ = io.WriteString(stderr, "[repo-lint] json marshal error: "+err.Error()+"\n")
			return 2
		}
		_, _ = io.WriteString(stdout, out)
	default:
		_, _ = io.WriteString(stdout, lint.FormatText(res))
	}

	return res.ExitCode(*warnAsError)
}

func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
