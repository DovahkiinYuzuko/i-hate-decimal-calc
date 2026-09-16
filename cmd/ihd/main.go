package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/updater"
)

func main() {
	code := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	os.Exit(code)
}

type runOptions struct {
	opts          calc.FormatOptions
	showApprox    bool
	latex         bool
	pretty        bool
	deg           bool
	explain       bool
	noUpdateCheck bool
}

func run(args []string, in io.Reader, out, errOut io.Writer) int {
	// Partition args into flags and expressions.
	var flagArgs []string
	var exprArgs []string

	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "-lang" || a == "--lang" {
			flagArgs = append(flagArgs, a)
			if i+1 < len(args) {
				i++
				flagArgs = append(flagArgs, args[i])
			}
			continue
		}
		if strings.HasPrefix(a, "-lang=") || strings.HasPrefix(a, "--lang=") {
			flagArgs = append(flagArgs, a)
			continue
		}
		switch a {
		case "-ascii", "--ascii", "-approx", "--approx", "-latex", "--latex", "-pretty", "--pretty", "-deg", "--deg", "-explain", "--explain", "-h", "--help", "-v", "--version", "-version", "--no-update-check", "-no-update-check":
			flagArgs = append(flagArgs, a)
		default:
			exprArgs = append(exprArgs, a)
		}
	}

	// Pre-detect language flag to ensure FlagSet usage text is localized
	earlyLang := ""
	for i := 0; i < len(flagArgs); i++ {
		if (flagArgs[i] == "-lang" || flagArgs[i] == "--lang") && i+1 < len(flagArgs) {
			earlyLang = flagArgs[i+1]
			break
		} else if strings.HasPrefix(flagArgs[i], "--lang=") {
			earlyLang = strings.TrimPrefix(flagArgs[i], "--lang=")
			break
		} else if strings.HasPrefix(flagArgs[i], "-lang=") {
			earlyLang = strings.TrimPrefix(flagArgs[i], "-lang=")
			break
		}
	}
	_ = i18n.Init(earlyLang)

	fs := flag.NewFlagSet("ihd", flag.ContinueOnError)
	fs.SetOutput(errOut)

	langFlag := fs.String("lang", "", i18n.T("cli.flag_lang"))
	asciiFlag := fs.Bool("ascii", false, i18n.T("cli.flag_ascii"))
	approxFlag := fs.Bool("approx", false, i18n.T("cli.flag_approx"))
	latexFlag := fs.Bool("latex", false, i18n.T("cli.flag_latex"))
	prettyFlag := fs.Bool("pretty", false, i18n.T("cli.flag_pretty"))
	degFlag := fs.Bool("deg", false, i18n.T("cli.flag_deg"))
	explainFlag := fs.Bool("explain", false, i18n.T("cli.flag_explain"))
	versionFlag := fs.Bool("version", false, i18n.T("cli.flag_version"))
	fs.BoolVar(versionFlag, "v", false, i18n.T("cli.flag_version"))
	noUpdateCheckFlag := fs.Bool("no-update-check", false, i18n.T("cli.flag_no_update_check"))
	helpFlag := fs.Bool("help", false, i18n.T("cli.flag_help"))
	fs.BoolVar(helpFlag, "h", false, i18n.T("cli.flag_help"))

	if err := fs.Parse(flagArgs); err != nil {
		return 1
	}

	// Re-initialize i18n engine if parsed flag specifies or updates locale
	if *langFlag != "" {
		_ = i18n.Init(*langFlag)
		_ = i18n.SaveConfigLocale(*langFlag)
	}

	if *helpFlag {
		fmt.Fprintln(out, i18n.T("cli.help"))
		return 0
	}

	if *versionFlag {
		fmt.Fprintf(out, "ihd %s\n", updater.CurrentVersion)
		if !*noUpdateCheckFlag && updater.IsUpdateCheckEnabled() {
			if info, _ := updater.CheckUpdate(false); info != nil {
				fmt.Fprintln(out)
				fmt.Fprint(out, updater.FormatNotification(info))
			}
		}
		return 0
	}

	if *noUpdateCheckFlag {
		_ = os.Setenv("IHD_NO_UPDATE_CHECK", "1")
	}

	ro := runOptions{
		opts:          calc.FormatOptions{AsciiOnly: *asciiFlag},
		showApprox:    *approxFlag,
		latex:         *latexFlag,
		pretty:        *prettyFlag,
		deg:           *degFlag,
		explain:       *explainFlag,
		noUpdateCheck: *noUpdateCheckFlag,
	}

	if len(exprArgs) > 0 {
		// 1. Explicit 'run' subcommand: ihd run <file.ihd> [flags]
		if exprArgs[0] == "run" {
			if len(exprArgs) < 2 {
				fmt.Fprintf(errOut, "%s%s\n", i18n.T("cli.error_prefix"), i18n.T("cli.err_run_script_required"))
				return 1
			}
			return runScriptFile(exprArgs[1], ro, out, errOut)
		}

		// 2. Direct script file invocation: ihd <file.ihd> [flags]
		if len(exprArgs) == 1 {
			target := exprArgs[0]
			if info, err := os.Stat(target); err == nil && !info.IsDir() && strings.HasSuffix(strings.ToLower(target), ".ihd") {
				return runScriptFile(target, ro, out, errOut)
			}
		}

		// 3. One-shot mode: join all non-flag arguments as the expression
		expr := strings.Join(exprArgs, " ")
		if err := evaluateLine(expr, ro, out, errOut); err != nil {
			return 1
		}
		return 0
	}

	// Check if stdin is a terminal or pipe
	if isTerminal(in) {
		return runREPL(in, out, errOut, ro)
	}

	// Pipe / redirect mode: process line by line
	scanner := bufio.NewScanner(in)
	hadError := false
	env := calc.NewEnv()
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if err := evaluateLineWithEnv(line, ro, env, out, errOut); err != nil {
			hadError = true
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(errOut, "%s%v\n", i18n.T("cli.error_prefix"), err)
		return 1
	}
	if hadError {
		return 1
	}
	return 0
}

func isTerminal(r io.Reader) bool {
	if f, ok := r.(*os.File); ok {
		stat, err := f.Stat()
		if err != nil {
			return false
		}
		return (stat.Mode() & os.ModeCharDevice) != 0
	}
	return false
}

func formatOutput(node calc.Node, ro runOptions) string {
	if ro.latex {
		return calc.FormatLaTeX(node)
	}
	if ro.pretty {
		return calc.FormatPretty2D(node)
	}

	exactStr := calc.FormatWithOptions(node, ro.opts)
	if ro.showApprox {
		approxStr, err := calc.Approx(node)
		if err == nil {
			return fmt.Sprintf("%s (≈ %s)", exactStr, approxStr)
		}
	}
	return exactStr
}

func evaluateLine(line string, ro runOptions, out, errOut io.Writer) error {
	env := calc.NewEnv()
	return evaluateLineWithEnv(line, ro, env, out, errOut)
}

func evaluateLineWithEnv(line string, ro runOptions, env *calc.Env, out, errOut io.Writer) error {
	lowerLine := strings.ToLower(line)
	if strings.HasPrefix(lowerLine, "explain ") || strings.HasPrefix(lowerLine, "steps ") {
		parts := strings.SplitN(line, " ", 2)
		ro.explain = true
		line = strings.TrimSpace(parts[1])
	}

	parsed, err := calc.ParseStatement(line)
	if err != nil {
		fmt.Fprintf(errOut, "%s%v\n", i18n.T("cli.error_prefix"), err)
		return err
	}

	if ro.deg {
		parsed = calc.ApplyDegreeModeToStatement(parsed)
	}

	switch v := parsed.(type) {
	case *calc.AssignStmt:
		var evaled calc.Node
		var steps []calc.PedagogicalStep
		if ro.explain {
			evaled, steps, err = calc.EvalWithTraceAndEnv(v.Value, env)
		} else {
			evaled, err = calc.EvalWithEnv(v.Value, env)
		}
		if err != nil {
			fmt.Fprintf(errOut, "%s%v\n", i18n.T("cli.error_prefix"), err)
			return err
		}
		env.Set(v.Name, evaled)
		env.Set("ans", evaled)
		if ro.explain {
			fmt.Fprint(out, calc.FormatTrace(line, steps, evaled))
		} else {
			fmt.Fprintln(out, formatOutput(evaled, ro))
		}
		return nil

	case calc.Node:
		var evaled calc.Node
		var steps []calc.PedagogicalStep
		if ro.explain {
			evaled, steps, err = calc.EvalWithTraceAndEnv(v, env)
		} else {
			evaled, err = calc.EvalWithEnv(v, env)
		}
		if err != nil {
			fmt.Fprintf(errOut, "%s%v\n", i18n.T("cli.error_prefix"), err)
			return err
		}
		env.Set("ans", evaled)
		if ro.explain {
			fmt.Fprint(out, calc.FormatTrace(line, steps, evaled))
		} else {
			fmt.Fprintln(out, formatOutput(evaled, ro))
		}
		return nil

	default:
		err := fmt.Errorf(i18n.T("cli.err_unknown_stmt"), parsed)
		fmt.Fprintf(errOut, "%s%v\n", i18n.T("cli.error_prefix"), err)
		return err
	}
}

func runScriptFile(filePath string, ro runOptions, out, errOut io.Writer) int {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Fprintf(errOut, "%s%s\n", i18n.T("cli.error_prefix"), i18n.T("cli.err_run_open_failed", filePath, err))
		return 1
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	env := calc.NewEnv()
	lineNum := 0
	hadError := false

	for scanner.Scan() {
		lineNum++
		rawLine := scanner.Text()
		line := strings.TrimSpace(rawLine)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Skip comment lines (#)
		if strings.HasPrefix(line, "#") {
			continue
		}

		// Strip inline comment (e.g. "x = 1/2; # comment")
		if idx := strings.Index(line, "#"); idx != -1 {
			line = strings.TrimSpace(line[:idx])
			if line == "" {
				continue
			}
		}

		// Check for output suppression via trailing semicolon ';'
		suppressOutput := false
		if strings.HasSuffix(line, ";") {
			suppressOutput = true
			line = strings.TrimSpace(strings.TrimSuffix(line, ";"))
			if line == "" {
				continue
			}
		}

		if err := executeScriptLine(line, ro, env, suppressOutput, out, errOut, filePath, lineNum); err != nil {
			hadError = true
			break
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(errOut, "%s%s\n", i18n.T("cli.error_prefix"), i18n.T("cli.err_run_read_failed", filePath, err))
		return 1
	}

	if hadError {
		return 1
	}
	return 0
}

func executeScriptLine(line string, ro runOptions, env *calc.Env, suppressOutput bool, out, errOut io.Writer, filePath string, lineNum int) error {
	lowerLine := strings.ToLower(line)
	if strings.HasPrefix(lowerLine, "explain ") || strings.HasPrefix(lowerLine, "steps ") {
		parts := strings.SplitN(line, " ", 2)
		ro.explain = true
		line = strings.TrimSpace(parts[1])
	}

	parsed, err := calc.ParseStatement(line)
	if err != nil {
		fmt.Fprintf(errOut, "%s:%d: %v\n", filePath, lineNum, err)
		return err
	}

	if ro.deg {
		parsed = calc.ApplyDegreeModeToStatement(parsed)
	}

	switch v := parsed.(type) {
	case *calc.AssignStmt:
		var evaled calc.Node
		var steps []calc.PedagogicalStep
		if ro.explain {
			evaled, steps, err = calc.EvalWithTraceAndEnv(v.Value, env)
		} else {
			evaled, err = calc.EvalWithEnv(v.Value, env)
		}
		if err != nil {
			fmt.Fprintf(errOut, "%s:%d: %v\n", filePath, lineNum, err)
			return err
		}
		env.Set(v.Name, evaled)
		env.Set("ans", evaled)
		if !suppressOutput {
			if ro.explain {
				fmt.Fprint(out, calc.FormatTrace(line, steps, evaled))
			} else {
				fmt.Fprintln(out, formatOutput(evaled, ro))
			}
		}
		return nil

	case calc.Node:
		var evaled calc.Node
		var steps []calc.PedagogicalStep
		if ro.explain {
			evaled, steps, err = calc.EvalWithTraceAndEnv(v, env)
		} else {
			evaled, err = calc.EvalWithEnv(v, env)
		}
		if err != nil {
			fmt.Fprintf(errOut, "%s:%d: %v\n", filePath, lineNum, err)
			return err
		}
		env.Set("ans", evaled)
		if !suppressOutput {
			if ro.explain {
				fmt.Fprint(out, calc.FormatTrace(line, steps, evaled))
			} else {
				fmt.Fprintln(out, formatOutput(evaled, ro))
			}
		}
		return nil

	default:
		err := fmt.Errorf(i18n.T("cli.err_unknown_stmt"), parsed)
		fmt.Fprintf(errOut, "%s:%d: %v\n", filePath, lineNum, err)
		return err
	}
}
