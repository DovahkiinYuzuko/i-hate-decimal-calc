package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
)

func main() {
	exitCode := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	os.Exit(exitCode)
}

type runOptions struct {
	opts       calc.FormatOptions
	showApprox bool
	latex      bool
	pretty     bool
	deg        bool
	explain    bool
}

func run(args []string, in io.Reader, out io.Writer, errOut io.Writer) int {
	var flagArgs []string
	var exprArgs []string

	for _, a := range args {
		switch a {
		case "-ascii", "--ascii", "-approx", "--approx", "-latex", "--latex", "-pretty", "--pretty", "-deg", "--deg", "-explain", "--explain", "-h", "--help":
			flagArgs = append(flagArgs, a)
		default:
			exprArgs = append(exprArgs, a)
		}
	}

	fs := flag.NewFlagSet("ihd", flag.ContinueOnError)
	fs.SetOutput(errOut)

	asciiFlag := fs.Bool("ascii", false, "Output ASCII characters only")
	approxFlag := fs.Bool("approx", false, "Show approximate decimal value")
	latexFlag := fs.Bool("latex", false, "Output expression in LaTeX format ($$ ... $$)")
	prettyFlag := fs.Bool("pretty", false, "Output expression using 2D pretty-printed formatting")
	degFlag := fs.Bool("deg", false, "Use degree mode for trigonometric and inverse trigonometric functions")
	explainFlag := fs.Bool("explain", false, "Show step-by-step algebraic rewriting explanations")
	helpFlag := fs.Bool("help", false, "Show help")
	fs.BoolVar(helpFlag, "h", false, "Show help")

	if err := fs.Parse(flagArgs); err != nil {
		return 1
	}

	if *helpFlag {
		fmt.Fprintln(out, calc.MsgHelp)
		return 0
	}

	ro := runOptions{
		opts:       calc.FormatOptions{AsciiOnly: *asciiFlag},
		showApprox: *approxFlag,
		latex:      *latexFlag,
		pretty:     *prettyFlag,
		deg:        *degFlag,
		explain:    *explainFlag,
	}

	if len(exprArgs) > 0 {
		// One-shot mode: join all non-flag arguments as the expression
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
		fmt.Fprintf(errOut, "%s%v\n", calc.MsgErrorPrefix, err)
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
		if err == nil {
			return (stat.Mode() & os.ModeCharDevice) != 0
		}
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
	return evaluateLineWithEnv(line, ro, calc.NewEnv(), out, errOut)
}

func evaluateLineWithEnv(line string, ro runOptions, env *calc.Env, out, errOut io.Writer) error {
	if strings.HasPrefix(line, "explain ") || strings.HasPrefix(line, "steps ") {
		parts := strings.SplitN(line, " ", 2)
		ro.explain = true
		line = strings.TrimSpace(parts[1])
	}

	parsed, err := calc.ParseStatement(line)
	if err != nil {
		fmt.Fprintf(errOut, "%s%v\n", calc.MsgErrorPrefix, err)
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
			fmt.Fprintf(errOut, "%s%v\n", calc.MsgErrorPrefix, err)
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
			fmt.Fprintf(errOut, "%s%v\n", calc.MsgErrorPrefix, err)
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
		err := fmt.Errorf("unknown statement type: %T", parsed)
		fmt.Fprintf(errOut, "%s%v\n", calc.MsgErrorPrefix, err)
		return err
	}
}
