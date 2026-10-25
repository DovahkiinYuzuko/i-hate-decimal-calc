package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"sort"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/provenance"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/updater"
	"github.com/nyaosorg/go-readline-ny"
	"github.com/nyaosorg/go-readline-ny/keys"
	"github.com/nyaosorg/go-readline-ny/simplehistory"
)

func runREPL(in io.Reader, out, errOut io.Writer, ro runOptions) int {
	fmt.Fprintln(out, i18n.T("cli.repl_welcome"))
	fmt.Fprintf(out, i18n.T("cli.repl_lang_label")+"\n", i18n.CurrentLocaleName(), i18n.CurrentLocale())
	fmt.Fprintln(out, i18n.T("cli.repl_exit_hint"))

	if !ro.noUpdateCheck && updater.IsUpdateCheckEnabled() {
		// Instant check from local cache without blocking (0ms startup)
		if info := updater.GetCachedUpdate(); info != nil {
			fmt.Fprintln(out)
			fmt.Fprint(out, updater.FormatNotification(info))
		}
		// Refresh cache in the background if expired
		updater.CheckUpdateAsync()
	}

	env := calc.NewEnv()

	// If in is not a terminal (e.g. test pipe or script), use standard scanner
	if !isTerminal(in) {
		return runScannerREPL(in, out, errOut, ro, env)
	}

	// Resolve initial syntax highlight & theme settings
	cfg, _ := i18n.LoadConfig()
	currentTheme := "dark"
	if cfg != nil && cfg.Theme != "" {
		currentTheme = cfg.Theme
	}
	if ro.theme != "" {
		currentTheme = ro.theme
	}
	highlightEnabled := true
	if cfg != nil && cfg.SyntaxHighlight != nil {
		highlightEnabled = *cfg.SyntaxHighlight
	}
	if ro.noColor || os.Getenv("NO_COLOR") != "" {
		highlightEnabled = false
	}
	if !highlightEnabled {
		currentTheme = "none"
	}

	history := simplehistory.New()
	histPath, _ := GetHistoryFilePath()
	if histPath != "" {
		if pastLines, err := LoadHistoryLines(histPath, defaultMaxHistoryLines); err == nil {
			for _, pl := range pastLines {
				history.Add(pl)
			}
			// If loaded history was larger than limit, rotate on disk at startup
			if len(pastLines) >= defaultMaxHistoryLines {
				_ = SaveHistoryLines(histPath, pastLines, defaultMaxHistoryLines)
			}
		}
	}

	completionFSM := NewCompletionFSM(env)
	editor := readline.Editor{
		PromptWriter: func(w io.Writer) (int, error) {
			return fmt.Fprint(w, i18n.T("cli.prompt"))
		},
		History:    history,
		Writer:     out,
		Highlight:  BuildREPLHighlights(currentTheme, env),
		ResetColor:   "\x1B[0m",
		DefaultColor: "\x1B[0m",
		OnAfterRender: func(B *readline.Buffer, availWidth int) {
			if currentTheme == "none" || !highlightEnabled {
				return
			}
			ctx, ok := ParseSignatureContext(B.String(), B.Cursor)
			if ok {
				_ = RenderSignatureInline(B.Out, ctx, currentTheme, availWidth)
			}
		},
	}

	editor.BindKey(keys.CtrlI, NewCompletionCommand(completionFSM))

	for {
		line, err := editor.ReadLine(context.Background())
		completionFSM.Reset()
		_ = ClearSignatureGuide(out)
		// Ensure any active terminal attributes or background colors are cleanly reset immediately
		fmt.Fprint(out, "\x1B[0m")
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, readline.CtrlC) {
				fmt.Fprintln(out)
				fmt.Fprintln(out, i18n.T("cli.repl_exit"))
				break
			}
			fmt.Fprint(errOut, "\x1B[0m")
			fmt.Fprintf(errOut, "%s%v\n", i18n.T("cli.error_prefix"), err)
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.EqualFold(line, "exit") || strings.EqualFold(line, "quit") {
			fmt.Fprintln(out, i18n.T("cli.repl_exit"))
			break
		}

		history.Add(line)
		if histPath != "" {
			_ = AppendHistoryLine(histPath, line, defaultMaxHistoryLines)
		}

		if strings.EqualFold(line, "vars") {
			printVars(env, ro, out)
			continue
		}

		if handleREPLCommand(line, env, in, nil, out, errOut) {
			continue
		}

		if handleThemeAndHighlightCommand(line, &currentTheme, &highlightEnabled, out, func() {
			if highlightEnabled {
				editor.Highlight = BuildREPLHighlights(currentTheme, env)
			} else {
				editor.Highlight = nil
			}
		}) {
			continue
		}

		_ = evaluateLineWithEnv(line, ro, env, in, out, errOut)
	}

	return 0
}

func runScannerREPL(in io.Reader, out, errOut io.Writer, ro runOptions, env *calc.Env) int {
	cfg, _ := i18n.LoadConfig()
	currentTheme := "dark"
	if cfg != nil && cfg.Theme != "" {
		currentTheme = cfg.Theme
	}
	if ro.theme != "" {
		currentTheme = ro.theme
	}
	highlightEnabled := true
	if cfg != nil && cfg.SyntaxHighlight != nil {
		highlightEnabled = *cfg.SyntaxHighlight
	}
	if ro.noColor || os.Getenv("NO_COLOR") != "" {
		highlightEnabled = false
	}

	scanner := bufio.NewScanner(in)
	for {
		fmt.Fprint(out, i18n.T("cli.prompt"))
		if !scanner.Scan() {
			fmt.Fprintln(out)
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.EqualFold(line, "exit") || strings.EqualFold(line, "quit") {
			fmt.Fprintln(out, i18n.T("cli.repl_exit"))
			break
		}
		if strings.EqualFold(line, "vars") {
			printVars(env, ro, out)
			continue
		}

		if handleREPLCommand(line, env, in, scanner, out, errOut) {
			continue
		}

		if handleThemeAndHighlightCommand(line, &currentTheme, &highlightEnabled, out, nil) {
			continue
		}

		_ = evaluateLineWithEnv(line, ro, env, in, out, errOut)
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(errOut, "%s%v\n", i18n.T("cli.error_prefix"), err)
		return 1
	}
	return 0
}

func printVars(env *calc.Env, ro runOptions, out io.Writer) {
	vars := env.All()
	if len(vars) == 0 {
		fmt.Fprintln(out, i18n.T("cli.no_vars"))
		return
	}

	var names []string
	for name := range vars {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		fmt.Fprintf(out, "%s = %s\n", name, formatOutput(vars[name], ro))
	}
}

func handleREPLCommand(line string, env *calc.Env, in io.Reader, scanner provenance.LineScanner, out, errOut io.Writer) bool {
	lowerLine := strings.ToLower(line)

	if lowerLine == "clear" || lowerLine == "reset" {
		env.Clear()
		fmt.Fprintln(out, i18n.T("cli.repl_clear_success"))
		return true
	}

	if strings.HasPrefix(lowerLine, "save ") || lowerLine == "save" {
		arg := strings.TrimSpace(line[4:])
		arg = strings.Trim(arg, "\"'")
		if arg == "" {
			fmt.Fprintln(out, i18n.T("cli.repl_save_missing_path"))
			return true
		}
		if err := SaveSession(env, arg); err != nil {
			fmt.Fprintf(errOut, "%s%v\n", i18n.T("cli.error_prefix"), err)
		} else {
			fmt.Fprintf(out, i18n.T("cli.repl_save_success")+"\n", arg)
		}
		return true
	}

	if strings.HasPrefix(lowerLine, "load ") || lowerLine == "load" {
		arg := strings.TrimSpace(line[4:])
		arg = strings.Trim(arg, "\"'")
		if arg == "" {
			fmt.Fprintln(out, i18n.T("cli.repl_load_missing_path"))
			return true
		}
		if err := LoadSession(env, arg, out, errOut); err != nil {
			fmt.Fprintf(errOut, "%s%v\n", i18n.T("cli.error_prefix"), err)
		} else {
			fmt.Fprintf(out, i18n.T("cli.repl_load_success")+"\n", arg)
		}
		return true
	}

	if strings.HasPrefix(lowerLine, "similar ") || strings.HasPrefix(lowerLine, "similar(") {
		query := ""
		if strings.HasPrefix(lowerLine, "similar(") && strings.HasSuffix(lowerLine, ")") {
			query = strings.TrimSpace(line[8 : len(line)-1])
		} else if strings.HasPrefix(lowerLine, "similar ") {
			query = strings.TrimSpace(line[8:])
		}
		if query == "" {
			fmt.Fprintln(out, i18n.T("cli.repl_similar_no_matches"))
			return true
		}

		histPath, _ := GetHistoryFilePath()
		var histLines []string
		if histPath != "" {
			histLines, _ = LoadHistoryLines(histPath, defaultMaxHistoryLines)
		}

		matches, err := FindSimilarExpressions(query, histLines, 5)
		if err != nil || len(matches) == 0 {
			fmt.Fprintln(out, i18n.T("cli.repl_similar_no_matches"))
			return true
		}

		fmt.Fprintln(out, i18n.T("cli.repl_similar_header"))
		for _, m := range matches {
			fmt.Fprintf(out, "  - %s  (similarity: %.0f%%)\n", m.Expr, m.Score*100)
		}
		return true
	}

	if strings.HasPrefix(lowerLine, "prove ") || strings.HasPrefix(lowerLine, "prove(") || lowerLine == "prove" {
		exprStr := ""
		if strings.HasPrefix(lowerLine, "prove(") && strings.HasSuffix(lowerLine, ")") {
			exprStr = strings.TrimSpace(line[6 : len(line)-1])
		} else if strings.HasPrefix(lowerLine, "prove ") {
			exprStr = strings.TrimSpace(line[6:])
		}
		if exprStr == "" {
			fmt.Fprintln(out, i18n.T("cli.repl_prove_missing_expr"))
			return true
		}

		node, err := calc.Parse(exprStr)
		if err != nil {
			fmt.Fprintf(errOut, "%s%v\n", i18n.T("cli.error_prefix"), err)
			return true
		}

		_, root, evalErr := calc.EvalWithProvenanceAndEnv(node, env)
		if evalErr != nil {
			fmt.Fprintf(errOut, "%s%v\n", i18n.T("cli.error_prefix"), evalErr)
			return true
		}
		if root == nil {
			fmt.Fprintln(out, i18n.T("cli.repl_step_completed"))
			return true
		}

		var explorer *provenance.StepExplorer
		if scanner != nil {
			explorer = provenance.NewStepExplorerWithScanner(root, scanner, out)
		} else {
			inputReader := in
			if inputReader == nil {
				inputReader = os.Stdin
			}
			explorer = provenance.NewStepExplorer(root, inputReader, out)
		}
		_ = explorer.Run()
		return true
	}

	if lowerLine == "help" || strings.HasPrefix(lowerLine, "help ") || strings.HasPrefix(lowerLine, "help(") {
		arg := ""
		if strings.HasPrefix(lowerLine, "help(") && strings.HasSuffix(lowerLine, ")") {
			arg = strings.TrimSpace(line[5 : len(line)-1])
		} else if strings.HasPrefix(lowerLine, "help ") {
			arg = strings.TrimSpace(line[4:])
		}
		arg = strings.Trim(arg, "\"'")
		PrintHelpCommand(arg, out)
		return true
	}

	if strings.HasPrefix(lowerLine, "search ") || strings.HasPrefix(lowerLine, "search(") || lowerLine == "search" {
		query := ""
		if strings.HasPrefix(lowerLine, "search(") && strings.HasSuffix(lowerLine, ")") {
			query = strings.TrimSpace(line[7 : len(line)-1])
		} else if strings.HasPrefix(lowerLine, "search ") {
			query = strings.TrimSpace(line[6:])
		}
		query = strings.Trim(query, "\"'")
		PrintSearchCommand(query, out)
		return true
	}

	if strings.HasPrefix(lowerLine, "whatis ") || strings.HasPrefix(lowerLine, "whatis(") || lowerLine == "whatis" {
		target := ""
		if strings.HasPrefix(lowerLine, "whatis(") && strings.HasSuffix(lowerLine, ")") {
			target = strings.TrimSpace(line[7 : len(line)-1])
		} else if strings.HasPrefix(lowerLine, "whatis ") {
			target = strings.TrimSpace(line[7:])
		}
		target = strings.Trim(target, "\"'")
		PrintWhatisCommand(target, out)
		return true
	}

	if lowerLine == "config loop" || lowerLine == "loop" {
		warnBig, maxBig, _ := env.GetIterationLimits()
		warnStr := "100000"
		if warnBig != nil {
			warnStr = warnBig.String()
		}
		maxStr := "unlimited"
		if maxBig != nil && maxBig.Sign() > 0 {
			maxStr = maxBig.String()
		}
		fmt.Fprintf(out, i18n.T("cli.repl_loop_status")+"\n", warnStr, maxStr)
		return true
	}

	if strings.HasPrefix(lowerLine, "warn-loop ") || strings.HasPrefix(lowerLine, "config warn-loop ") {
		idx := strings.LastIndex(line, " ")
		valStr := strings.TrimSpace(line[idx+1:])
		if n, ok := new(big.Int).SetString(valStr, 10); ok && n.Sign() > 0 {
			_ = i18n.SaveConfigWarnIterationsBig(n)
			_, maxLimit, writer := env.GetIterationLimits()
			env.SetIterationLimits(n, maxLimit, writer)
			fmt.Fprintf(out, i18n.T("cli.repl_warn_loop_set")+"\n", n.String())
		} else {
			fmt.Fprintln(out, i18n.T("cli.repl_loop_invalid"))
		}
		return true
	}

	if strings.HasPrefix(lowerLine, "max-loop ") || strings.HasPrefix(lowerLine, "config max-loop ") {
		idx := strings.LastIndex(line, " ")
		valStr := strings.TrimSpace(line[idx+1:])
		lowerVal := strings.ToLower(valStr)
		if lowerVal == "off" || lowerVal == "-1" || lowerVal == "none" || lowerVal == "unlimited" {
			unlimited := big.NewInt(-1)
			_ = i18n.SaveConfigMaxIterationsBig(unlimited)
			warnLimit, _, writer := env.GetIterationLimits()
			env.SetIterationLimits(warnLimit, unlimited, writer)
			fmt.Fprintf(out, i18n.T("cli.repl_max_loop_set")+"\n", "unlimited")
		} else if n, ok := new(big.Int).SetString(valStr, 10); ok && n.Sign() > 0 {
			_ = i18n.SaveConfigMaxIterationsBig(n)
			warnLimit, _, writer := env.GetIterationLimits()
			env.SetIterationLimits(warnLimit, n, writer)
			fmt.Fprintf(out, i18n.T("cli.repl_max_loop_set")+"\n", n.String())
		} else {
			fmt.Fprintln(out, i18n.T("cli.repl_loop_invalid"))
		}
		return true
	}

	return false
}

func handleThemeAndHighlightCommand(line string, currentTheme *string, highlightEnabled *bool, out io.Writer, onThemeChange func()) bool {
	lowerLine := strings.ToLower(line)
	switch {
	case lowerLine == "theme":
		fmt.Fprintf(out, i18n.T("cli.repl_theme_current")+"\n", *currentTheme)
		return true

	case strings.HasPrefix(lowerLine, "theme "):
		target := strings.TrimSpace(lowerLine[6:])
		if target == "dark" || target == "light" || target == "none" {
			*currentTheme = target
			_ = i18n.SaveConfigTheme(target)
			if onThemeChange != nil {
				onThemeChange()
			}
			fmt.Fprintf(out, i18n.T("cli.repl_theme_set")+"\n", target)
		} else {
			fmt.Fprintln(out, i18n.T("cli.repl_theme_invalid"))
		}
		return true

	case lowerLine == "highlight" || lowerLine == "color":
		if *highlightEnabled {
			fmt.Fprintf(out, i18n.T("cli.repl_highlight_status_on")+"\n", *currentTheme)
		} else {
			fmt.Fprintln(out, i18n.T("cli.repl_highlight_status_off"))
		}
		return true

	case lowerLine == "highlight on" || lowerLine == "color on":
		*highlightEnabled = true
		if *currentTheme == "none" || *currentTheme == "" {
			*currentTheme = "dark"
		}
		_ = i18n.SaveConfigHighlight(true)
		if onThemeChange != nil {
			onThemeChange()
		}
		fmt.Fprintln(out, i18n.T("cli.repl_highlight_on"))
		return true

	case lowerLine == "highlight off" || lowerLine == "color off":
		*highlightEnabled = false
		_ = i18n.SaveConfigHighlight(false)
		if onThemeChange != nil {
			onThemeChange()
		}
		fmt.Fprintln(out, i18n.T("cli.repl_highlight_off"))
		return true
	}
	return false
}
