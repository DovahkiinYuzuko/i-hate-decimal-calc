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
}

func run(args []string, in io.Reader, out io.Writer, errOut io.Writer) int {
	var flagArgs []string
	var exprArgs []string

	for _, a := range args {
		switch a {
		case "-ascii", "--ascii", "-approx", "--approx", "-latex", "--latex", "-pretty", "--pretty", "-h", "--help":
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
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if err := evaluateLine(line, ro, out, errOut); err != nil {
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
	node, err := calc.EvalString(line)
	if err != nil {
		fmt.Fprintf(errOut, "%s%v\n", calc.MsgErrorPrefix, err)
		return err
	}

	fmt.Fprintln(out, formatOutput(node, ro))
	return nil
}

func runREPL(in io.Reader, out, errOut io.Writer, ro runOptions) int {
	fmt.Fprintln(out, calc.MsgREPLWelcome)
	scanner := bufio.NewScanner(in)

	for {
		fmt.Fprint(out, calc.PromptREPL)
		if !scanner.Scan() {
			// EOF or interrupt
			fmt.Fprintln(out)
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" {
			fmt.Fprintln(out, calc.MsgREPLExit)
			break
		}

		_ = evaluateLine(line, ro, out, errOut)
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(errOut, "%s%v\n", calc.MsgErrorPrefix, err)
		return 1
	}
	return 0
}
