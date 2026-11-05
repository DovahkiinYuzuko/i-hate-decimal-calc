package runner

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// ExecuteCase executes a single test case against the given target binary.
func ExecuteCase(ctx context.Context, defaultBinary string, verifierBinary string, tc TestCase) TestResult {
	start := time.Now()
	timeout := tc.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	caseCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	binPath := defaultBinary
	var actualFlags []string

	for _, f := range tc.Flags {
		if strings.HasPrefix(f, "--target-bin=") {
			target := strings.TrimPrefix(f, "--target-bin=")
			if target == "$tmpVerifierBin" || strings.Contains(target, "verifier") {
				binPath = verifierBinary
			} else {
				binPath = target
			}
		} else {
			actualFlags = append(actualFlags, f)
		}
	}

	cmd := exec.CommandContext(caseCtx, binPath, actualFlags...)
	cmd.Env = append(os.Environ(),
		"IHD_LANG=en",
		"IHD_NO_UPDATE_CHECK=1",
	)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	if tc.PipeInput != "" {
		cmd.Stdin = strings.NewReader(tc.PipeInput)
	}

	runErr := cmd.Run()
	duration := time.Since(start)

	actualExit := 0
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			actualExit = exitErr.ExitCode()
		} else if caseCtx.Err() == context.DeadlineExceeded {
			return TestResult{
				Case:       tc,
				Passed:     false,
				ActualOut:  stdoutBuf.String(),
				ActualErr:  stderrBuf.String(),
				ActualExit: -1,
				Duration:   duration,
				ErrorMsg:   fmt.Sprintf("Execution timed out after %v", timeout),
			}
		} else {
			actualExit = -1
		}
	}

	actualOut := strings.TrimSpace(normalizeNewlines(stdoutBuf.String()))
	actualErr := strings.TrimSpace(normalizeNewlines(stderrBuf.String()))

	passed, failMsg := evaluateMatch(tc, actualOut, actualErr, actualExit)

	return TestResult{
		Case:       tc,
		Passed:     passed,
		ActualOut:  actualOut,
		ActualErr:  actualErr,
		ActualExit: actualExit,
		Duration:   duration,
		ErrorMsg:   failMsg,
	}
}

func evaluateMatch(tc TestCase, actualOut string, actualErr string, actualExit int) (bool, string) {
	// 1. Verify Exit Code
	if tc.ExpectedExit != actualExit {
		return false, fmt.Sprintf("Exit code mismatch: expected %d, got %d", tc.ExpectedExit, actualExit)
	}

	// 2. Evaluate according to MatchMode
	switch tc.MatchMode {
	case MatchError:
		if tc.ExpectedStderr != "" {
			if !matchWildcardOrContains(tc.ExpectedStderr, actualErr) && !matchWildcardOrContains(tc.ExpectedStderr, actualOut) {
				return false, fmt.Sprintf("Expected error substring %q not found in stderr (%q) or stdout (%q)",
					tc.ExpectedStderr, actualErr, actualOut)
			}
		}
		return true, ""

	case MatchContains:
		if !matchWildcardOrContains(tc.ExpectedStdout, actualOut) {
			return false, fmt.Sprintf("Expected substring %q not found in stdout (%q)", tc.ExpectedStdout, actualOut)
		}
		return true, ""

	case MatchRegex:
		matched, err := regexp.MatchString(tc.ExpectedStdout, actualOut)
		if err != nil {
			return false, fmt.Sprintf("Invalid regex %q: %v", tc.ExpectedStdout, err)
		}
		if !matched {
			return false, fmt.Sprintf("Output %q did not match regex %q", actualOut, tc.ExpectedStdout)
		}
		return true, ""

	case MatchExact:
		fallthrough
	default:
		if actualOut != tc.ExpectedStdout {
			return false, fmt.Sprintf("Output mismatch:\nExpected: %q\nActual:   %q", tc.ExpectedStdout, actualOut)
		}
		return true, ""
	}
}

func matchWildcardOrContains(pattern string, text string) bool {
	if pattern == "" {
		return true
	}
	if strings.Contains(pattern, "*") {
		// Convert wildcard pattern to regex
		regexPat := "(?s)^.*" + strings.ReplaceAll(regexp.QuoteMeta(pattern), `\*`, `.*`) + ".*$"
		matched, err := regexp.MatchString(regexPat, text)
		if err == nil && matched {
			return true
		}
	}
	return strings.Contains(text, pattern)
}
