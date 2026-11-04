package runner

import "time"

// MatchMode represents how the expected output should be matched.
type MatchMode string

const (
	MatchExact    MatchMode = "exact"
	MatchContains MatchMode = "contains"
	MatchRegex    MatchMode = "regex"
	MatchError    MatchMode = "error"
)

// TestCase represents a single test case parsed from a CSV file.
type TestCase struct {
	ID              string        `json:"id"`
	Category        string        `json:"category"`
	Flags           []string      `json:"flags"`
	PipeInput       string        `json:"pipe_input"`
	MatchMode       MatchMode     `json:"match_mode"`
	ExpectedStdout  string        `json:"expected_stdout"`
	ExpectedStderr  string        `json:"expected_stderr"`
	ExpectedExit    int           `json:"expected_exit_code"`
	DeductionMethod string        `json:"deduction_method"`
	Description     string        `json:"description"`
	Timeout         time.Duration `json:"timeout"`
	SourceFile      string        `json:"source_file"`
	SourceLine      int           `json:"source_line"`
}

// TestResult represents the outcome of running a TestCase.
type TestResult struct {
	Case       TestCase      `json:"case"`
	Passed     bool          `json:"passed"`
	ActualOut  string        `json:"actual_stdout"`
	ActualErr  string        `json:"actual_stderr"`
	ActualExit int           `json:"actual_exit_code"`
	Duration   time.Duration `json:"duration"`
	ErrorMsg   string        `json:"error_msg,omitempty"`
}
