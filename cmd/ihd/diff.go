package main

import (
	"fmt"
	"io"
	"os"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

func runDiff(source1, source2 string, out, errOut io.Writer) int {
	s1 := source1
	if b, err := os.ReadFile(source1); err == nil {
		s1 = string(b)
	}
	s2 := source2
	if b, err := os.ReadFile(source2); err == nil {
		s2 = string(b)
	}

	env := calc.NewEnv()
	diffs, err := calc.CompareScripts(s1, s2, env)
	if err != nil {
		fmt.Fprintf(errOut, "%s%v\n", i18n.T("cli.error_prefix"), err)
		return 1
	}

	fmt.Fprintf(out, i18n.T("cli.diff_header")+"\n", source1, source2)

	allIdentical := true
	for _, d := range diffs {
		if d.Status != calc.DiffIdentical {
			allIdentical = false
			break
		}
	}

	if allIdentical {
		fmt.Fprintln(out, i18n.T("cli.diff_identical"))
		return 0
	}

	for _, d := range diffs {
		switch d.Status {
		case calc.DiffIdentical:
			// Suppress identical lines or print dimmed
			continue
		case calc.DiffEquivalent:
			fmt.Fprintf(out, "  ~ [%s] L%d: %s <=> %s\n", i18n.T("cli.diff_stat_equivalent"), d.Index, d.LeftRaw, d.RightRaw)
		case calc.DiffChanged:
			fmt.Fprintf(out, "  * [%s] L%d: %s -> %s\n", i18n.T("cli.diff_stat_changed"), d.Index, d.LeftRaw, d.RightRaw)
		case calc.DiffAdded:
			fmt.Fprintf(out, "  + [%s] L%d: %s\n", i18n.T("cli.diff_stat_added"), d.Index, d.RightRaw)
		case calc.DiffRemoved:
			fmt.Fprintf(out, "  - [%s] L%d: %s\n", i18n.T("cli.diff_stat_removed"), d.Index, d.LeftRaw)
		}
	}

	return 0
}
