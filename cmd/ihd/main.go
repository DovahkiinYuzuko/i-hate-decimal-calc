package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/provenance"
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
	typst         bool
	format        string
	pretty        bool
	deg           bool
	explain       bool
	step          bool
	trace         bool
	verify        bool
	lean          bool
	leanFile      string
	certFile      string
	replayFile    string
	noUpdateCheck bool
	interval      bool
	intervalEps   string
	noColor       bool
	theme         string
	warnLoop      *big.Int
	maxLoop       *big.Int
}

func run(args []string, in io.Reader, out, errOut io.Writer) int {
	// Partition args into flags and expressions.
	var flagArgs []string
	var exprArgs []string

	valuedFlags := map[string]bool{
		"lang": true, "warn-loop": true, "max-loop": true, "lean-file": true,
		"certificate": true, "replay": true, "interval-eps": true, "theme": true, "format": true,
	}

	for i := 0; i < len(args); i++ {
		a := args[i]
		trimmed := strings.TrimLeft(a, "-")
		name, _, hasEq := strings.Cut(trimmed, "=")
		if valuedFlags[name] {
			flagArgs = append(flagArgs, a)
			if !hasEq && i+1 < len(args) {
				i++
				flagArgs = append(flagArgs, args[i])
			}
			continue
		}
		switch a {
		case "-ascii", "--ascii", "-approx", "--approx", "-latex", "--latex", "-typst", "--typst", "-pretty", "--pretty", "-deg", "--deg", "-explain", "--explain", "-verify", "--verify", "-lean", "--lean", "-h", "--help", "-v", "--version", "-version", "--no-update-check", "-no-update-check", "-i", "--i", "-interval", "--interval", "-no-color", "--no-color", "-step", "--step", "-trace", "--trace":
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
	formatFlag := fs.String("format", "", i18n.T("cli.flag_format"))
	typstFlag := fs.Bool("typst", false, i18n.T("cli.flag_typst"))
	latexFlag := fs.Bool("latex", false, i18n.T("cli.flag_latex"))
	prettyFlag := fs.Bool("pretty", false, i18n.T("cli.flag_pretty"))
	degFlag := fs.Bool("deg", false, i18n.T("cli.flag_deg"))
	explainFlag := fs.Bool("explain", false, i18n.T("cli.flag_explain"))
	stepFlag := fs.Bool("step", false, i18n.T("cli.flag_step"))
	traceFlag := fs.Bool("trace", false, i18n.T("cli.flag_trace"))
	verifyFlag := fs.Bool("verify", false, i18n.T("cli.flag_verify"))
	certificateFlag := fs.String("certificate", "", i18n.T("cli.flag_certificate"))
	replayFlag := fs.String("replay", "", i18n.T("cli.flag_replay"))
	leanFlag := fs.Bool("lean", false, i18n.T("cli.flag_lean"))
	leanFileFlag := fs.String("lean-file", "", i18n.T("cli.flag_lean_file"))
	intervalFlag := fs.Bool("interval", false, i18n.T("cli.flag_interval"))
	fs.BoolVar(intervalFlag, "i", false, i18n.T("cli.flag_interval"))
	intervalEpsFlag := fs.String("interval-eps", "", i18n.T("cli.flag_interval"))
	noColorFlag := fs.Bool("no-color", false, i18n.T("cli.flag_no_color"))
	themeFlag := fs.String("theme", "", i18n.T("cli.flag_theme"))
	warnLoopFlag := fs.String("warn-loop", "", i18n.T("cli.flag_warn_loop"))
	maxLoopFlag := fs.String("max-loop", "", i18n.T("cli.flag_max_loop"))
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

	var warnLoopBig *big.Int
	if *warnLoopFlag != "" {
		if n, ok := new(big.Int).SetString(*warnLoopFlag, 10); ok {
			warnLoopBig = n
		}
	}
	var maxLoopBig *big.Int
	if *maxLoopFlag != "" {
		if n, ok := new(big.Int).SetString(*maxLoopFlag, 10); ok {
			maxLoopBig = n
		}
	}

	if *helpFlag {
		fmt.Fprintln(out, i18n.T("cli.help"))
		return 0
	}

	if *versionFlag {
		fmt.Fprintf(out, "ihd %s\n", updater.CurrentVersion)
		if !*noUpdateCheckFlag && updater.IsUpdateCheckEnabled() {
			if info, _ := updater.CheckUpdate(true); info != nil {
				fmt.Fprintln(out)
				fmt.Fprint(out, updater.FormatNotification(info))
			}
		}
		return 0
	}

	if *noUpdateCheckFlag {
		_ = os.Setenv("IHD_NO_UPDATE_CHECK", "1")
	}

	fmtMode := strings.ToLower(*formatFlag)
	ro := runOptions{
		opts:          calc.FormatOptions{AsciiOnly: *asciiFlag || fmtMode == "ascii"},
		showApprox:    *approxFlag,
		latex:         *latexFlag || fmtMode == "latex",
		typst:         *typstFlag || fmtMode == "typst",
		format:        fmtMode,
		pretty:        *prettyFlag || fmtMode == "pretty",
		deg:           *degFlag,
		explain:       *explainFlag,
		step:          *stepFlag,
		trace:         *traceFlag,
		verify:        *verifyFlag,
		certFile:      *certificateFlag,
		replayFile:    *replayFlag,
		lean:          *leanFlag,
		leanFile:      *leanFileFlag,
		noUpdateCheck: *noUpdateCheckFlag,
		interval:      *intervalFlag,
		intervalEps:   *intervalEpsFlag,
		noColor:       *noColorFlag,
		theme:         *themeFlag,
		warnLoop:      warnLoopBig,
		maxLoop:       maxLoopBig,
	}

	if ro.replayFile != "" {
		return replayCertificate(ro.replayFile, out, errOut)
	}

	// In non-interactive modes (one-shot, script, or pipe), notify via errOut if an update
	// is cached, without polluting stdout or breaking pipes. Also trigger async cache refresh.
	if !isTerminal(in) || len(exprArgs) > 0 {
		defer func() {
			if !ro.noUpdateCheck && updater.IsUpdateCheckEnabled() {
				if info := updater.GetCachedUpdate(); info != nil {
					fmt.Fprintln(errOut)
					fmt.Fprint(errOut, updater.FormatNotification(info))
				}
				updater.CheckUpdateAsync()
			}
		}()
	}

	if len(exprArgs) > 0 {
		// 0. Explicit 'diff' subcommand: ihd diff <file1.ihd|expr1> <file2.ihd|expr2>
		if exprArgs[0] == "diff" {
			if len(exprArgs) < 3 {
				fmt.Fprintf(errOut, "%s%s\n", i18n.T("cli.error_prefix"), i18n.T("cli.diff_usage"))
				return 1
			}
			return runDiff(exprArgs[1], exprArgs[2], out, errOut)
		}

		// 1. Explicit 'run' subcommand: ihd run <file.ihd> [flags]
		if exprArgs[0] == "run" {
			if len(exprArgs) < 2 {
				fmt.Fprintf(errOut, "%s%s\n", i18n.T("cli.error_prefix"), i18n.T("cli.err_run_script_required"))
				return 1
			}
			return runScriptFile(exprArgs[1], ro, out, errOut)
		}

		// Explicit 'completion' subcommand: ihd completion <shell>
		if exprArgs[0] == "completion" {
			if len(exprArgs) < 2 {
				fmt.Fprintf(errOut, "%s%s\n", i18n.T("cli.error_prefix"), i18n.T("cli.err_completion_shell_required"))
				return 1
			}
			if err := RunCompletion(exprArgs[1], out); err != nil {
				fmt.Fprintf(errOut, "%s%v\n", i18n.T("cli.error_prefix"), err)
				return 1
			}
			return 0
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
		if err := evaluateLine(expr, ro, in, out, errOut); err != nil {
			return 1
		}
		return 0
	}

	// Check if stdin is a terminal or pipe
	if isTerminal(in) {
		return runREPL(in, out, errOut, ro)
	}

	// Pipe / redirect mode: process line by line with buffered writer
	bufOut := bufio.NewWriterSize(out, 64*1024)
	defer bufOut.Flush()

	scanner := bufio.NewScanner(in)
	hadError := false
	env := calc.NewEnv()
	ensureEnvIterationLimits(env, ro, errOut)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.EqualFold(line, "exit") || strings.EqualFold(line, "quit") {
			break
		}
		if strings.EqualFold(line, "vars") {
			printVars(env, ro, bufOut)
			continue
		}
		if handleREPLCommand(line, env, in, scanner, bufOut, errOut) {
			continue
		}
		lowerLine := strings.ToLower(line)
		if lowerLine == "theme" {
			cfg, _ := i18n.LoadConfig()
			currentTheme := "dark"
			if cfg != nil && cfg.Theme != "" {
				currentTheme = cfg.Theme
			}
			if ro.theme != "" {
				currentTheme = ro.theme
			}
			fmt.Fprintf(bufOut, i18n.T("cli.repl_theme_current")+"\n", currentTheme)
			continue
		}
		if strings.HasPrefix(lowerLine, "theme ") {
			target := strings.TrimSpace(lowerLine[6:])
			if target == "dark" || target == "light" || target == "none" {
				_ = i18n.SaveConfigTheme(target)
				fmt.Fprintf(bufOut, i18n.T("cli.repl_theme_set")+"\n", target)
			} else {
				fmt.Fprintln(bufOut, i18n.T("cli.repl_theme_invalid"))
			}
			continue
		}
		if lowerLine == "highlight" || lowerLine == "color" {
			cfg, _ := i18n.LoadConfig()
			if (cfg != nil && cfg.SyntaxHighlight != nil && !*cfg.SyntaxHighlight) || ro.noColor {
				fmt.Fprintln(bufOut, i18n.T("cli.repl_highlight_status_off"))
			} else {
				currentTheme := "dark"
				if cfg != nil && cfg.Theme != "" {
					currentTheme = cfg.Theme
				}
				if ro.theme != "" {
					currentTheme = ro.theme
				}
				fmt.Fprintf(bufOut, i18n.T("cli.repl_highlight_status_on")+"\n", currentTheme)
			}
			continue
		}
		if lowerLine == "highlight on" || lowerLine == "color on" {
			_ = i18n.SaveConfigHighlight(true)
			fmt.Fprintln(bufOut, i18n.T("cli.repl_highlight_on"))
			continue
		}
		if lowerLine == "highlight off" || lowerLine == "color off" {
			_ = i18n.SaveConfigHighlight(false)
			fmt.Fprintln(bufOut, i18n.T("cli.repl_highlight_off"))
			continue
		}

		if err := evaluateLineWithEnv(line, ro, env, in, bufOut, errOut); err != nil {
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
	if ro.interval {
		eps := big.NewRat(1, 1000000)
		if ro.intervalEps != "" {
			if parsed, ok := new(big.Rat).SetString(ro.intervalEps); ok && parsed.Sign() > 0 {
				eps = parsed
			}
		}
		interval, err := calc.AdaptiveRefineInterval(node, eps, 100)
		if err == nil {
			intNode := calc.NewIntervalNode(interval.Low, interval.High)
			decEnclosure := calc.FormatDecimalEnclosure(interval, 6)
			return fmt.Sprintf("%s ~ %s", intNode.String(), decEnclosure)
		}
	}

	if ro.typst {
		return calc.FormatTypst(node)
	}
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
			return i18n.T("cli.approx_format", exactStr, approxStr)
		}
	}
	return exactStr
}

func outputNode(out io.Writer, node calc.Node, ro runOptions) error {
	if ro.interval {
		eps := big.NewRat(1, 1000000)
		if ro.intervalEps != "" {
			if parsed, ok := new(big.Rat).SetString(ro.intervalEps); ok && parsed.Sign() > 0 {
				eps = parsed
			}
		}
		interval, err := calc.AdaptiveRefineInterval(node, eps, 100)
		if err == nil {
			intNode := calc.NewIntervalNode(interval.Low, interval.High)
			decEnclosure := calc.FormatDecimalEnclosure(interval, 6)
			_, err = fmt.Fprintf(out, "%s ~ %s\n", intNode.String(), decEnclosure)
			return err
		}
	}

	if ro.typst {
		_, err := fmt.Fprintln(out, calc.FormatTypst(node))
		return err
	}
	if ro.latex {
		_, err := fmt.Fprintln(out, calc.FormatLaTeX(node))
		return err
	}
	if ro.pretty {
		_, err := fmt.Fprintln(out, calc.FormatPretty2D(node))
		return err
	}
	if ro.showApprox {
		exactStr := calc.FormatWithOptions(node, ro.opts)
		approxStr, err := calc.Approx(node)
		if err == nil {
			_, err = fmt.Fprintln(out, i18n.T("cli.approx_format", exactStr, approxStr))
			return err
		}
		_, err = fmt.Fprintln(out, exactStr)
		return err
	}

	// High-performance streaming path using WriteNode with bufio
	bw := calc.NewBufferedStreamWriter(out)
	if err := calc.WriteNode(bw, node, ro.opts); err != nil {
		return err
	}
	if err := bw.WriteByte('\n'); err != nil {
		return err
	}
	return bw.Flush()
}

func ensureEnvIterationLimits(env *calc.Env, ro runOptions, errOut io.Writer) {
	currentWarn, currentMax, _ := env.GetIterationLimits()
	if currentWarn == nil && currentMax == nil {
		cfg, _ := i18n.LoadConfig()
		warn := ro.warnLoop
		if warn == nil && cfg != nil {
			warn = cfg.GetWarnIterationsBig()
		}
		max := ro.maxLoop
		if max == nil && cfg != nil {
			max = cfg.GetMaxIterationsBig()
		}
		env.SetIterationLimits(warn, max, errOut)
	} else {
		_, _, w := env.GetIterationLimits()
		if w == nil {
			env.SetIterationLimits(currentWarn, currentMax, errOut)
		}
	}
}

func evaluateLine(line string, ro runOptions, in io.Reader, out, errOut io.Writer) error {
	env := calc.NewEnv()
	return evaluateLineWithEnv(line, ro, env, in, out, errOut)
}

func renderExplainOutput(origNode, evaled calc.Node, steps []calc.PedagogicalStep, line string, ro runOptions, env *calc.Env, out io.Writer) {
	tracer := env.GetLastTrace()
	if ro.typst && tracer != nil && tracer.Root() != nil {
		typstVis := provenance.NewTypstVisitor()
		fmt.Fprint(out, typstVis.RenderDocument(tracer.Root(), line))
		return
	}
	if len(steps) == 0 && tracer != nil && tracer.Root() != nil && (len(tracer.Root().Children) > 0 || tracer.Root().RuleID != "") {
		explainVis := provenance.NewExplainVisitor()
		fmt.Fprintln(out, explainVis.RenderTree(tracer.Root(), line))
		return
	}
	cert, errCert := calc.VerifyComputation(origNode, evaled, env)
	if errCert == nil && cert != nil && cert.IsVerified {
		trace, errTrace := cert.ToProofTrace()
		if errTrace == nil && trace != nil && trace.StepCount() > 1 {
			fmt.Fprint(out, trace.RenderExplain(i18n.CurrentLocale()))
			return
		}
	}
	fmt.Fprint(out, calc.FormatTrace(line, steps, evaled))
}

func handleEvaluationOutputs(origNode, evaled calc.Node, steps []calc.PedagogicalStep, line string, ro runOptions, env *calc.Env, suppressOutput bool, in io.Reader, out, errOut io.Writer) {
	if suppressOutput {
		return
	}
	if ro.trace {
		if c := calc.GetActiveSearchCollector(); c != nil {
			_ = c.DumpJSON(out)
			return
		}
	}
	if ro.step {
		tracer := env.GetLastTrace()
		if tracer == nil || tracer.Root() == nil {
			fmt.Fprintln(out, i18n.T("cli.repl_step_completed"))
			return
		}
		inputReader := in
		if inputReader == nil {
			inputReader = os.Stdin
		}
		explorer := provenance.NewStepExplorer(tracer.Root(), inputReader, out)
		_ = explorer.Run()
		return
	}
	if ro.explain {
		renderExplainOutput(origNode, evaled, steps, line, ro, env, out)
	} else {
		_ = outputNode(out, evaled, ro)
	}
	if ro.verify {
		cert, _ := calc.VerifyComputation(origNode, evaled, env)
		fmt.Fprintln(out, cert.String())
	}
	if ro.lean || ro.leanFile != "" {
		if ro.explain {
			if tracer := env.GetLastTrace(); tracer != nil && tracer.Root() != nil {
				leanVis := provenance.NewLeanCalcVisitor()
				code, err := leanVis.RenderCalc(tracer.Root(), "ihd_certified_derivation")
				if err == nil {
					if ro.lean {
						fmt.Fprintln(out, code)
					}
					if ro.leanFile != "" {
						_ = saveLeanProofFile(ro.leanFile, code)
					}
					return
				}
			}
		}
		cert, _ := calc.VerifyComputation(origNode, evaled, env)
		if cert != nil && cert.IsVerified {
			leanCode, err := calc.GenerateLeanSource("ihd_certified_proof", cert, origNode, evaled)
			if err == nil {
				if ro.lean {
					fmt.Fprintln(out, leanCode)
				}
				if ro.leanFile != "" {
					if err := saveLeanProofFile(ro.leanFile, leanCode); err != nil {
						fmt.Fprintf(errOut, "%s%s\n", i18n.T("cli.error_prefix"), fmt.Sprintf(i18n.T("cli.err_write_file"), ro.leanFile, err))
					} else {
						fmt.Fprintln(out, fmt.Sprintf(i18n.T("cli.lean_file_saved"), ro.leanFile))
					}
				}
			} else {
				fmt.Fprintf(errOut, "%s%v\n", i18n.T("cli.error_prefix"), err)
			}
		} else {
			if cert != nil && cert.Details != "" {
				fmt.Fprintf(errOut, "%s%s\n", i18n.T("cli.error_prefix"), fmt.Sprintf(i18n.T("lean.err_cannot_generate_lean_unverified"), cert.Details))
			} else {
				fmt.Fprintf(errOut, "%s%s\n", i18n.T("cli.error_prefix"), i18n.T("lean.err_cannot_generate_lean_no_cert"))
			}
		}
	}
	if ro.certFile != "" {
		cert, _ := calc.VerifyComputation(origNode, evaled, env)
		if cert != nil {
			trace, err := calc.ConvertVerificationCertificateToProofTrace(cert)
			if err == nil && trace != nil {
				if err := calc.SaveProofTraceJSON(trace, ro.certFile); err != nil {
					fmt.Fprintf(errOut, "%s%s\n", i18n.T("cli.error_prefix"), fmt.Sprintf(i18n.T("cli.err_write_file"), ro.certFile, err))
				} else {
					fmt.Fprintln(out, fmt.Sprintf(i18n.T("cli.cert_file_saved"), ro.certFile))
				}
			}
		}
	}
}

func evaluateLineInternal(line string, ro runOptions, env *calc.Env, suppressOutput bool, in io.Reader, out, errOut io.Writer, errPrefix string) error {
	lowerLine := strings.ToLower(line)
	if strings.HasPrefix(lowerLine, "explain ") || strings.HasPrefix(lowerLine, "steps ") {
		parts := strings.SplitN(line, " ", 2)
		ro.explain = true
		line = strings.TrimSpace(parts[1])
	} else if strings.HasPrefix(lowerLine, "verify ") {
		parts := strings.SplitN(line, " ", 2)
		ro.verify = true
		line = strings.TrimSpace(parts[1])
	} else if strings.HasPrefix(lowerLine, "lean ") {
		parts := strings.SplitN(line, " ", 2)
		ro.lean = true
		line = strings.TrimSpace(parts[1])
	}

	parsed, err := calc.ParseStatement(line)
	if err != nil {
		diagMsg := FormatErrorWithDiagnostics(err, line)
		fmt.Fprintf(errOut, "%s%s\n", errPrefix, diagMsg)
		return err
	}

	if ro.deg {
		parsed = calc.ApplyDegreeModeToStatement(parsed)
	}

	var evalTarget calc.Node
	var assignName string

	switch v := parsed.(type) {
	case *calc.AssignStmt:
		evalTarget = v.Value
		assignName = v.Name

	case *calc.UserFuncDefStmt:
		// Try compiling body to bytecode for VM speedup
		st := calc.NewSymbolTable()
		for _, param := range v.Params {
			st.Define(param)
		}
		compiler := calc.NewCompiler(st)
		bc, _ := compiler.Compile(v.Body)

		if err := env.SetUserFunc(v.Name, v.Params, v.Body); err != nil {
			fmt.Fprintf(errOut, "%s%v\n", errPrefix, err)
			return err
		}
		if bc != nil {
			if fn, ok := env.GetUserFunc(v.Name); ok {
				fn.Bytecode = bc
			}
		}
		if !suppressOutput {
			fmt.Fprintf(out, "%s(%s)\n", v.Name, strings.Join(v.Params, ", "))
		}
		return nil

	case *calc.ImportStmt:
		return runImportFile(v.Path, ro, env, in, out, errOut, errPrefix)

	case calc.Node:
		evalTarget = v
	default:
		err := fmt.Errorf(i18n.T("cli.err_unknown_stmt"), parsed)
		fmt.Fprintf(errOut, "%s%v\n", errPrefix, err)
		return err
	}

	var evaled calc.Node
	var steps []calc.PedagogicalStep
	var searchCollector *calc.MemorySearchTraceCollector
	if ro.trace {
		searchCollector = calc.NewMemorySearchTraceCollector()
		env.SearchTracer = searchCollector
		calc.SetActiveSearchCollector(searchCollector)
		defer func() {
			env.SearchTracer = nil
			calc.SetActiveSearchCollector(nil)
		}()
	}

	if ro.explain || ro.step {
		evaled, steps, err = calc.EvalWithTraceAndEnv(evalTarget, env)
	} else {
		evaled, err = calc.EvalWithEnv(evalTarget, env)
	}
	if err != nil {
		diagMsg := FormatErrorWithDiagnostics(err, line)
		fmt.Fprintf(errOut, "%s%s\n", errPrefix, diagMsg)
		return err
	}
	if rel, ok := evaled.(*calc.RelOpNode); ok && (rel.Op == "==" || rel.Op == "!=") {
		if resolved, errEq := calc.EvaluateRelOpEquality(rel, env); errEq == nil && resolved != nil {
			evaled = resolved
		}
	}
	if assignName != "" {
		env.Set(assignName, evaled)
	}
	env.Set("ans", evaled)

	handleEvaluationOutputs(evalTarget, evaled, steps, line, ro, env, suppressOutput, in, out, errOut)
	return nil
}

func evaluateLineWithEnv(line string, ro runOptions, env *calc.Env, in io.Reader, out, errOut io.Writer) error {
	ensureEnvIterationLimits(env, ro, errOut)
	return evaluateLineInternal(line, ro, env, false, in, out, errOut, i18n.T("cli.error_prefix"))
}

func runImportFile(filePath string, ro runOptions, env *calc.Env, in io.Reader, out, errOut io.Writer, errPrefix string) error {
	cleanPath := filepath.Clean(filePath)
	if err := env.EnterImport(cleanPath); err != nil {
		fmt.Fprintf(errOut, "%s%v\n", errPrefix, err)
		return err
	}
	defer env.LeaveImport(cleanPath)

	file, err := os.Open(cleanPath)
	if err != nil {
		openErr := fmt.Errorf("%s", i18n.T("cli.err_run_open_failed", cleanPath, err))
		fmt.Fprintf(errOut, "%s%v\n", errPrefix, openErr)
		return openErr
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if idx := strings.Index(line, "#"); idx != -1 {
			line = strings.TrimSpace(line[:idx])
		}
		if line == "" {
			continue
		}
		// Execute imported line suppressing standard output
		if err := evaluateLineInternal(line, ro, env, true, in, out, errOut, errPrefix); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(errOut, "%s%v\n", errPrefix, err)
		return err
	}
	return nil
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
	ensureEnvIterationLimits(env, ro, errOut)
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
	prefix := fmt.Sprintf("%s:%d: ", filePath, lineNum)
	return evaluateLineInternal(line, ro, env, suppressOutput, nil, out, errOut, prefix)
}

func saveLeanProofFile(path, code string) error {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, []byte(code), 0644)
}

func replayCertificate(certPath string, out, errOut io.Writer) int {
	trace, err := calc.LoadProofTraceFromJSON(certPath)
	if err != nil {
		fmt.Fprintf(errOut, "%s%s\n", i18n.T("cli.error_prefix"), fmt.Sprintf(i18n.T("cli.err_replay_read"), certPath, err))
		return 1
	}

	env := calc.NewEnv()
	verified, err := calc.VerifyProofTrace(trace, env)
	if err != nil || !verified {
		fmt.Fprintf(errOut, "%s\n", fmt.Sprintf(i18n.T("cli.replay_failure"), certPath))
		if err != nil {
			fmt.Fprintf(errOut, "%s%v\n", i18n.T("cli.error_prefix"), err)
		}
		return 1
	}

	fmt.Fprintln(out, fmt.Sprintf(i18n.T("cli.replay_success"), certPath))
	fmt.Fprint(out, trace.RenderExplain(i18n.CurrentLocale()))
	return 0
}
