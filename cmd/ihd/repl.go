package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
	"github.com/nyaosorg/go-readline-ny"
	"github.com/nyaosorg/go-readline-ny/simplehistory"
)

func runREPL(in io.Reader, out, errOut io.Writer, ro runOptions) int {
	fmt.Fprintln(out, i18n.T("cli.repl_welcome"))
	fmt.Fprintf(out, i18n.T("cli.repl_lang_label")+"\n", i18n.CurrentLocaleName(), i18n.CurrentLocale())
	fmt.Fprintln(out, i18n.T("cli.repl_exit_hint"))
	env := calc.NewEnv()

	// If in is not a terminal (e.g. test pipe or script), use standard scanner
	if !isTerminal(in) {
		return runScannerREPL(in, out, errOut, ro, env)
	}

	history := simplehistory.New()
	editor := readline.Editor{
		PromptWriter: func(w io.Writer) (int, error) {
			return fmt.Fprint(w, i18n.T("cli.prompt"))
		},
		History: history,
		Writer:  out,
	}

	for {
		line, err := editor.ReadLine(context.Background())
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, readline.CtrlC) {
				fmt.Fprintln(out)
				fmt.Fprintln(out, i18n.T("cli.repl_exit"))
				break
			}
			fmt.Fprintf(errOut, "%s%v\n", i18n.T("cli.error_prefix"), err)
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" {
			fmt.Fprintln(out, i18n.T("cli.repl_exit"))
			break
		}

		history.Add(line)

		if line == "vars" {
			printVars(env, ro, out)
			continue
		}

		_ = evaluateLineWithEnv(line, ro, env, out, errOut)
	}

	return 0
}

func runScannerREPL(in io.Reader, out, errOut io.Writer, ro runOptions, env *calc.Env) int {
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
		if line == "exit" || line == "quit" {
			fmt.Fprintln(out, i18n.T("cli.repl_exit"))
			break
		}
		if line == "vars" {
			printVars(env, ro, out)
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
