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
	verify        bool
	lean          bool
	leanFile      string
	certFile      string
	replayFile    string
	noUpdateCheck bool
	interval      bool
	intervalEps   string
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
		if a == "-lean-file" || a == "--lean-file" {
			flagArgs = append(flagArgs, a)
			if i+1 < len(args) {
				i++
				flagArgs = append(flagArgs, args[i])
			}
			continue
		}
		if strings.HasPrefix(a, "-lean-file=") || strings.HasPrefix(a, "--lean-file=") {
			flagArgs = append(flagArgs, a)
			continue
		}
		if a == "-certificate" || a == "--certificate" {
			flagArgs = append(flagArgs, a)
			if i+1 < len(args) {
				i++
				flagArgs = append(flagArgs, args[i])
			}
			continue
		}
		if strings.HasPrefix(a, "-certificate=") || strings.HasPrefix(a, "--certificate=") {
			flagArgs = append(flagArgs, a)
			continue
		}
		if a == "-replay" || a == "--replay" {
			flagArgs = append(flagArgs, a)
			if i+1 < len(args) {
				i++
				flagArgs = append(flagArgs, args[i])
			}
			continue
		}
		if strings.HasPrefix(a, "-replay=") || strings.HasPrefix(a, "--replay=") {
			flagArgs = append(flagArgs, a)
			continue
		}
		if a == "-interval-eps" || a == "--interval-eps" {
			flagArgs = append(flagArgs, a)
			if i+1 < len(args) {
				i++
				flagArgs = append(flagArgs, args[i])
			}
			continue
		}
		if strings.HasPrefix(a, "-interval-eps=") || strings.HasPrefix(a, "--interval-eps=") {
			flagArgs = append(flagArgs, a)
			continue
		}
		switch a {
		case "-ascii", "--ascii", "-approx", "--approx", "-latex", "--latex", "-pretty", "--pretty", "-deg", "--deg", "-explain", "--explain", "-verify", "--verify", "-lean", "--lean", "-h", "--help", "-v", "--version", "-version", "--no-update-check", "-no-update-check", "-i", "--i", "-interval", "--interval":
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
	verifyFlag := fs.Bool("verify", false, i18n.T("cli.flag_verify"))
	certificateFlag := fs.String("certificate", "", i18n.T("cli.flag_certificate"))
	replayFlag := fs.String("replay", "", i18n.T("cli.flag_replay"))
	leanFlag := fs.Bool("lean", false, i18n.T("cli.flag_lean"))
	leanFileFlag := fs.String("lean-file", "", i18n.T("cli.flag_lean_file"))
	intervalFlag := fs.Bool("interval", false, i18n.T("cli.flag_interval"))
	fs.BoolVar(intervalFlag, "i", false, i18n.T("cli.flag_interval"))
	intervalEpsFlag := fs.String("interval-eps", "", i18n.T("cli.flag_interval"))
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

	ro := runOptions{
		opts:          calc.FormatOptions{AsciiOnly: *asciiFlag},
		showApprox:    *approxFlag,
		latex:         *latexFlag,
		pretty:        *prettyFlag,
		deg:           *degFlag,
		explain:       *explainFlag,
		verify:        *verifyFlag,
		certFile:      *certificateFlag,
		replayFile:    *replayFlag,
		lean:          *leanFlag,
		leanFile:      *leanFileFlag,
		noUpdateCheck: *noUpdateCheckFlag,
		interval:      *intervalFlag,
		intervalEps:   *intervalEpsFlag,
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
		if rel, ok := evaled.(*calc.RelOpNode); ok && (rel.Op == "==" || rel.Op == "!=") {
			if resolved, errEq := calc.EvaluateRelOpEquality(rel, env); errEq == nil && resolved != nil {
				evaled = resolved
			}
		}
		env.Set(v.Name, evaled)
		env.Set("ans", evaled)
		if ro.explain {
			fmt.Fprint(out, calc.FormatTrace(line, steps, evaled))
		} else {
			_ = outputNode(out, evaled, ro)
		}
		if ro.verify {
			cert, _ := calc.VerifyComputation(v.Value, evaled, env)
			fmt.Fprintln(out, cert.String())
		}
		if ro.lean || ro.leanFile != "" {
			cert, _ := calc.VerifyComputation(v.Value, evaled, env)
			if cert != nil && cert.IsVerified {
				leanCode, err := calc.GenerateLeanSource("ihd_certified_proof", cert, v.Value, evaled)
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
			cert, _ := calc.VerifyComputation(v.Value, evaled, env)
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
		if rel, ok := evaled.(*calc.RelOpNode); ok && (rel.Op == "==" || rel.Op == "!=") {
			if resolved, errEq := calc.EvaluateRelOpEquality(rel, env); errEq == nil && resolved != nil {
				evaled = resolved
			}
		}
		env.Set("ans", evaled)
		if ro.explain {
			fmt.Fprint(out, calc.FormatTrace(line, steps, evaled))
		} else {
			_ = outputNode(out, evaled, ro)
		}
		if ro.verify {
			cert, _ := calc.VerifyComputation(v, evaled, env)
			fmt.Fprintln(out, cert.String())
		}
		if ro.lean || ro.leanFile != "" {
			cert, _ := calc.VerifyComputation(v, evaled, env)
			if cert != nil && cert.IsVerified {
				leanCode, err := calc.GenerateLeanSource("ihd_certified_proof", cert, v, evaled)
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
			cert, _ := calc.VerifyComputation(v, evaled, env)
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
				_ = outputNode(out, evaled, ro)
			}
			if ro.verify {
				cert, _ := calc.VerifyComputation(v.Value, evaled, env)
				fmt.Fprintln(out, cert.String())
			}
			if ro.lean || ro.leanFile != "" {
				cert, _ := calc.VerifyComputation(v.Value, evaled, env)
				if cert != nil && cert.IsVerified {
					leanCode, err := calc.GenerateLeanSource("ihd_certified_proof", cert, v.Value, evaled)
					if err == nil {
						if ro.lean {
							fmt.Fprintln(out, leanCode)
						}
						if ro.leanFile != "" {
							_ = os.WriteFile(ro.leanFile, []byte(leanCode), 0644)
						}
					}
				}
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
				_ = outputNode(out, evaled, ro)
			}
			if ro.verify {
				cert, _ := calc.VerifyComputation(v, evaled, env)
				fmt.Fprintln(out, cert.String())
			}
			if ro.lean || ro.leanFile != "" {
				cert, _ := calc.VerifyComputation(v, evaled, env)
				if cert != nil && cert.IsVerified {
					leanCode, err := calc.GenerateLeanSource("ihd_certified_proof", cert, v, evaled)
					if err == nil {
						if ro.lean {
							fmt.Fprintln(out, leanCode)
						}
						if ro.leanFile != "" {
							_ = saveLeanProofFile(ro.leanFile, leanCode)
						}
					}
				}
			}
		}
		return nil

	default:
		err := fmt.Errorf(i18n.T("cli.err_unknown_stmt"), parsed)
		fmt.Fprintf(errOut, "%s:%d: %v\n", filePath, lineNum, err)
		return err
	}
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
