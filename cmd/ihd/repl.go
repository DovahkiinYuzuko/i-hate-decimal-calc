package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/updater"
	"github.com/nyaosorg/go-readline-ny"
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
	editor := readline.Editor{
		PromptWriter: func(w io.Writer) (int, error) {
			return fmt.Fprint(w, i18n.T("cli.prompt"))
		},
		History:    history,
		Writer:     out,
		Highlight:  BuildREPLHighlights(currentTheme, env),
		ResetColor:   "\x1B[0m",
		DefaultColor: "\x1B[0m",
	}

	for {
		line, err := editor.ReadLine(context.Background())
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

		if strings.EqualFold(line, "vars") {
			printVars(env, ro, out)
			continue
		}

		lowerLine := strings.ToLower(line)
		if lowerLine == "theme" {
			fmt.Fprintf(out, i18n.T("cli.repl_theme_current")+"\n", currentTheme)
			continue
		}
		if strings.HasPrefix(lowerLine, "theme ") {
			target := strings.TrimSpace(lowerLine[6:])
			if target == "dark" || target == "light" || target == "none" {
				currentTheme = target
				editor.Highlight = BuildREPLHighlights(currentTheme, env)
				_ = i18n.SaveConfigTheme(currentTheme)
				fmt.Fprintf(out, i18n.T("cli.repl_theme_set")+"\n", currentTheme)
			} else {
				fmt.Fprintln(out, i18n.T("cli.repl_theme_invalid"))
			}
			continue
		}

		if lowerLine == "highlight" || lowerLine == "color" {
			if highlightEnabled && editor.Highlight != nil {
				fmt.Fprintf(out, i18n.T("cli.repl_highlight_status_on")+"\n", currentTheme)
			} else {
				fmt.Fprintln(out, i18n.T("cli.repl_highlight_status_off"))
			}
			continue
		}
		if lowerLine == "highlight on" || lowerLine == "color on" {
			highlightEnabled = true
			if currentTheme == "none" || currentTheme == "" {
				currentTheme = "dark"
			}
			editor.Highlight = BuildREPLHighlights(currentTheme, env)
			_ = i18n.SaveConfigHighlight(true)
			fmt.Fprintln(out, i18n.T("cli.repl_highlight_on"))
			continue
		}
		if lowerLine == "highlight off" || lowerLine == "color off" {
			highlightEnabled = false
			editor.Highlight = nil
			_ = i18n.SaveConfigHighlight(false)
			fmt.Fprintln(out, i18n.T("cli.repl_highlight_off"))
			continue
		}

		_ = evaluateLineWithEnv(line, ro, env, out, errOut)
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

		lowerLine := strings.ToLower(line)
		if lowerLine == "theme" {
			fmt.Fprintf(out, i18n.T("cli.repl_theme_current")+"\n", currentTheme)
			continue
		}
		if strings.HasPrefix(lowerLine, "theme ") {
			target := strings.TrimSpace(lowerLine[6:])
			if target == "dark" || target == "light" || target == "none" {
				currentTheme = target
				_ = i18n.SaveConfigTheme(target)
				fmt.Fprintf(out, i18n.T("cli.repl_theme_set")+"\n", target)
			} else {
				fmt.Fprintln(out, i18n.T("cli.repl_theme_invalid"))
			}
			continue
		}
		if lowerLine == "highlight" || lowerLine == "color" {
			if highlightEnabled {
				fmt.Fprintf(out, i18n.T("cli.repl_highlight_status_on")+"\n", currentTheme)
			} else {
				fmt.Fprintln(out, i18n.T("cli.repl_highlight_status_off"))
			}
			continue
		}
		if lowerLine == "highlight on" || lowerLine == "color on" {
			highlightEnabled = true
			if currentTheme == "none" || currentTheme == "" {
				currentTheme = "dark"
			}
			_ = i18n.SaveConfigHighlight(true)
			fmt.Fprintln(out, i18n.T("cli.repl_highlight_on"))
			continue
		}
		if lowerLine == "highlight off" || lowerLine == "color off" {
			highlightEnabled = false
			_ = i18n.SaveConfigHighlight(false)
			fmt.Fprintln(out, i18n.T("cli.repl_highlight_off"))
			continue
		}

		_ = evaluateLineWithEnv(line, ro, env, out, errOut)
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
