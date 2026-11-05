package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/tools/e2e/runner"
)

func main() {
	var categoryFilter string
	var nameFilter string
	var jobs int
	var verbose bool

	flag.StringVar(&categoryFilter, "c", "", "Filter test cases by category name or number (e.g. '01_arithmetic' or '10')")
	flag.StringVar(&categoryFilter, "category", "", "Filter test cases by category name or number")
	flag.StringVar(&nameFilter, "f", "", "Filter test cases by ID or description")
	flag.StringVar(&nameFilter, "filter", "", "Filter test cases by ID or description")
	flag.IntVar(&jobs, "j", 0, "Number of concurrent workers (default: runtime.NumCPU())")
	flag.IntVar(&jobs, "jobs", 0, "Number of concurrent workers")
	flag.BoolVar(&verbose, "v", false, "Verbose output (print passed cases in detail)")
	flag.BoolVar(&verbose, "verbose", false, "Verbose output")
	flag.Parse()

	if jobs <= 0 {
		jobs = runtime.NumCPU()
	}

	fmt.Println("============================================================")
	fmt.Println("     ihd Go-Native E2E Test Runner")
	fmt.Printf("     Workers: %d (CPUs: %d) | Category: %q | Filter: %q\n", jobs, runtime.NumCPU(), categoryFilter, nameFilter)
	fmt.Println("============================================================")

	// 1. Build temporary test binary
	rootPath, err := findRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error locating repository root: %v\n", err)
		os.Exit(1)
	}

	tmpBinPath := filepath.Join(rootPath, "ihd_e2e_test_bin.exe")
	tmpVerifierBin := filepath.Join(rootPath, "micro_verifier_e2e_test.exe")
	if runtime.GOOS != "windows" {
		tmpBinPath = filepath.Join(rootPath, "ihd_e2e_test_bin")
		tmpVerifierBin = filepath.Join(rootPath, "micro_verifier_e2e_test")
	}

	fmt.Printf("==> Building temporary ihd test binary (%s)...\n", tmpBinPath)
	buildCmd := exec.Command("go", "build", "-buildvcs=false", "-o", tmpBinPath, "./cmd/ihd")
	buildCmd.Dir = rootPath
	if buildOut, err := buildCmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build ihd binary:\n%s\nError: %v\n", string(buildOut), err)
		os.Exit(1)
	}

	fmt.Printf("==> Building temporary micro-verifier test binary (%s)...\n", tmpVerifierBin)
	buildVerifierCmd := exec.Command("go", "build", "-buildvcs=false", "-o", tmpVerifierBin, "./tools/micro-verifier")
	buildVerifierCmd.Dir = rootPath
	if buildOut, err := buildVerifierCmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build micro-verifier binary:\n%s\nError: %v\n", string(buildOut), err)
		os.Exit(1)
	}

	defer func() {
		_ = os.Remove(tmpBinPath)
		_ = os.Remove(tmpVerifierBin)
	}()

	// 2. Discover & Load CSV Test Cases
	testCasesDir := filepath.Join(rootPath, "tools", "e2e", "test-cases")
	fmt.Printf("==> Discovering and loading test cases from %s...\n", testCasesDir)
	cases, err := runner.LoadTestCases(testCasesDir, categoryFilter, nameFilter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load test cases: %v\n", err)
		os.Exit(1)
	}

	totalCases := len(cases)
	if totalCases == 0 {
		fmt.Println("[WARN] No matching test cases found with the given filters.")
		os.Exit(0)
	}
	fmt.Printf("==> Loaded %d test cases. Starting parallel execution...\n\n", totalCases)

	// 3. Execute with Worker Pool
	ctx := context.Background()
	startTime := time.Now()

	results := runner.RunPool(ctx, tmpBinPath, tmpVerifierBin, cases, jobs, func(res runner.TestResult, completed int, total int) {
		if !res.Passed {
			// Immediately highlight failures
			fmt.Printf("[%5.1f%% %4d/%4d] \x1b[31m[FAIL]\x1b[0m %s: %s\n       --> %s\n",
				float64(completed)/float64(total)*100, completed, total, res.Case.ID, res.Case.Description, res.ErrorMsg)
		} else if verbose {
			fmt.Printf("[%5.1f%% %4d/%4d] [PASS] %s: %s (%v)\n",
				float64(completed)/float64(total)*100, completed, total, res.Case.ID, res.Case.Description, res.Duration.Round(time.Millisecond))
		} else if completed%50 == 0 || completed == total {
			fmt.Printf("[%5.1f%% %4d/%4d] Running...\n", float64(completed)/float64(total)*100, completed, total)
		}
	})

	totalDuration := time.Since(startTime)

	// 4. Summarize and write fail log if any
	var passedCases, failedCases []runner.TestResult
	for _, r := range results {
		if r.Passed {
			passedCases = append(passedCases, r)
		} else {
			failedCases = append(failedCases, r)
		}
	}

	fmt.Println("\n============================================================")
	if len(failedCases) == 0 {
		fmt.Printf("\x1b[32m[TEST SUMMARY] SUCCESS: All %d tests passed, 0 failed in %v\x1b[0m\n",
			len(passedCases), totalDuration.Round(time.Millisecond))
		os.Exit(0)
	}

	fmt.Printf("\x1b[31m[TEST SUMMARY] FAILURE: %d passed, %d failed in %v\x1b[0m\n",
		len(passedCases), len(failedCases), totalDuration.Round(time.Millisecond))

	// Write failure details to tools/e2e/logs/
	logPath, err := writeFailureLog(rootPath, failedCases)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to write failure log: %v\n", err)
	} else {
		fmt.Printf("Detailed failure log written to: %s\n", logPath)
	}

	os.Exit(1)
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("go.mod not found in hierarchy")
}

func writeFailureLog(rootPath string, failures []runner.TestResult) (string, error) {
	logsDir := filepath.Join(rootPath, "tools", "e2e", "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return "", err
	}

	timestamp := time.Now().Format("20060102_150405")
	logFile := filepath.Join(logsDir, fmt.Sprintf("e2e_failures_%s.log", timestamp))

	var b strings.Builder
	b.WriteString(fmt.Sprintf("=== ihd E2E Failure Report (%s) ===\n", time.Now().Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("Total Failures: %d\n\n", len(failures)))

	for i, f := range failures {
		b.WriteString(fmt.Sprintf("------------------------------------------------------------\n"))
		b.WriteString(fmt.Sprintf("[%d] Case ID:     %s\n", i+1, f.Case.ID))
		b.WriteString(fmt.Sprintf("    Category:    %s\n", f.Case.Category))
		b.WriteString(fmt.Sprintf("    Description: %s\n", f.Case.Description))
		b.WriteString(fmt.Sprintf("    Method:      %s\n", f.Case.DeductionMethod))
		b.WriteString(fmt.Sprintf("    Source:      %s:%d\n", f.Case.SourceFile, f.Case.SourceLine))
		b.WriteString(fmt.Sprintf("    Flags:       %v\n", f.Case.Flags))
		if f.Case.PipeInput != "" {
			b.WriteString(fmt.Sprintf("    Pipe Input:\n%s\n", indent(f.Case.PipeInput, "      | ")))
		}
		b.WriteString(fmt.Sprintf("    Error Msg:   %s\n", f.ErrorMsg))
		b.WriteString(fmt.Sprintf("    Expected Exit: %d, Actual Exit: %d\n", f.Case.ExpectedExit, f.ActualExit))
		b.WriteString(fmt.Sprintf("    Expected Output:\n%s\n", indent(f.Case.ExpectedStdout, "      | ")))
		b.WriteString(fmt.Sprintf("    Actual Stdout:\n%s\n", indent(f.ActualOut, "      | ")))
		if f.ActualErr != "" {
			b.WriteString(fmt.Sprintf("    Actual Stderr:\n%s\n", indent(f.ActualErr, "      | ")))
		}
		b.WriteString("\n")
	}

	if err := os.WriteFile(logFile, []byte(b.String()), 0644); err != nil {
		return "", err
	}
	return logFile, nil
}

func indent(s string, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = prefix + l
	}
	return strings.Join(lines, "\n")
}
