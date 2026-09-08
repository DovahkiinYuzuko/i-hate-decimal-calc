package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	exitCode := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	os.Exit(exitCode)
}

func run(args []string, in io.Reader, out io.Writer, errOut io.Writer) int {
	var flagArgs []string
	var exprArgs []string

	for _, a := range args {
		switch a {
		case "-ascii", "--ascii", "-approx", "--approx", "-h", "--help":
			flagArgs = append(flagArgs, a)
		default:
			exprArgs = append(exprArgs, a)
		}
	}

	fs := flag.NewFlagSet("ihd", flag.ContinueOnError)
	fs.SetOutput(errOut)

	asciiFlag := fs.Bool("ascii", false, "Output ASCII characters only")
	approxFlag := fs.Bool("approx", false, "Show approximate decimal value")
	helpFlag := fs.Bool("help", false, "Show help")
	fs.BoolVar(helpFlag, "h", false, "Show help")

	if err := fs.Parse(flagArgs); err != nil {
		return 1
	}

	if *helpFlag {
		fmt.Fprintln(out, MsgHelp)
		return 0
	}

	opts := FormatOptions{AsciiOnly: *asciiFlag}
	showApprox := *approxFlag

	if len(exprArgs) > 0 {
		// One-shot mode: join all non-flag arguments as the expression
		expr := strings.Join(exprArgs, " ")
		if err := evaluateLine(expr, opts, showApprox, out, errOut); err != nil {
			return 1
		}
		return 0
	}

	// Check if stdin is a terminal or pipe
	if isTerminal(in) {
		return runREPL(in, out, errOut, opts, showApprox)
	}

	// Pipe / redirect mode: process line by line
	scanner := bufio.NewScanner(in)
	hadError := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if err := evaluateLine(line, opts, showApprox, out, errOut); err != nil {
			hadError = true
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(errOut, "%s%v\n", MsgErrorPrefix, err)
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

func evaluateLine(line string, opts FormatOptions, showApprox bool, out, errOut io.Writer) error {
	node, err := EvalString(line)
	if err != nil {
		fmt.Fprintf(errOut, "%s%v\n", MsgErrorPrefix, err)
		return err
	}

	exactStr := FormatWithOptions(node, opts)
	if showApprox {
		approxStr, err := Approx(node)
		if err == nil {
			fmt.Fprintf(out, "%s (≈ %s)\n", exactStr, approxStr)
			return nil
		}
	}
	fmt.Fprintln(out, exactStr)
	return nil
}

func runREPL(in io.Reader, out, errOut io.Writer, opts FormatOptions, showApprox bool) int {
	fmt.Fprintln(out, MsgREPLWelcome)
	scanner := bufio.NewScanner(in)

	for {
		fmt.Fprint(out, PromptREPL)
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
			fmt.Fprintln(out, MsgREPLExit)
			break
		}

		_ = evaluateLine(line, opts, showApprox, out, errOut)
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(errOut, "%s%v\n", MsgErrorPrefix, err)
		return 1
	}
	return 0
}
