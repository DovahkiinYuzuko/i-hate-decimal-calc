package lint

import (
	"encoding/csv"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// E2ECSVChecker validates header schema, column counts, required fields,
// enum values, and cross-file unique IDs for tools/e2e/test-cases/**/*.csv.
type E2ECSVChecker struct{}

func NewE2ECSVChecker() *E2ECSVChecker { return &E2ECSVChecker{} }

func (c *E2ECSVChecker) Name() string { return "e2e-csv" }
func (c *E2ECSVChecker) Description() string {
	return "Validates E2E test_cases CSV schema, column counts, required fields, enums, and cross-file duplicate IDs."
}

var requiredCSVHeaders = []string{
	"id",
	"category",
	"flags",
	"pipe_input",
	"match_mode",
	"expected_stdout",
	"expected_stderr",
	"expected_exit_code",
	"deduction_method",
	"description",
}

var validMatchModes = map[string]bool{
	"exact":    true,
	"contains": true,
	"regex":    true,
	"error":    true,
}

var validDeductionMethods = map[string]bool{
	"":               true,
	"deductive":      true,
	"contradiction":  true,
	"contrapositive": true,
	"inductive":      true,
	"metamorphic":    true,
	"roundtrip":      true,
}

type csvIDOrigin struct {
	file string
	line int
}

func (c *E2ECSVChecker) Run(ctx *LintContext) ([]Diagnostic, error) {
	casesDir := filepath.Join(ctx.RepoRoot, "tools", "e2e", "test-cases")
	if _, err := os.Stat(casesDir); os.IsNotExist(err) {
		return nil, nil
	}

	if ctx.Mode == ModeStaged && !hasRelevantCSVStage(ctx.StagedFiles) {
		return nil, nil
	}

	var csvFiles []string
	err := filepath.WalkDir(casesDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), ".csv") {
			rel, relErr := filepath.Rel(ctx.RepoRoot, p)
			if relErr != nil {
				return relErr
			}
			csvFiles = append(csvFiles, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(csvFiles)

	stagedSet := make(map[string]bool, len(ctx.StagedFiles))
	for _, f := range ctx.StagedFiles {
		stagedSet[f] = true
	}

	seenIDs := make(map[string]csvIDOrigin)
	var diags []Diagnostic

	for _, rel := range csvFiles {
		abs := filepath.Join(ctx.RepoRoot, filepath.FromSlash(rel))
		fileDiags := c.validateCSVFile(rel, abs, seenIDs)
		if ctx.Mode == ModeStaged && !stagedSet[rel] {
			continue
		}
		diags = append(diags, fileDiags...)
	}

	return diags, nil
}

func (c *E2ECSVChecker) validateCSVFile(rel, abs string, seenIDs map[string]csvIDOrigin) []Diagnostic {
	f, err := os.Open(abs)
	if err != nil {
		return []Diagnostic{{
			RuleID:      "e2e-csv-parse",
			Checker:     c.Name(),
			Severity:    SeverityError,
			File:        rel,
			Message:     "Failed to open CSV file: " + err.Error(),
			Reason:      "Unreadable CSV files break the Go native E2E runner.",
			Remediation: "Ensure `" + rel + "` is readable UTF-8 CSV.",
		}}
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	r.LazyQuotes = true

	header, err := r.Read()
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return []Diagnostic{{
			RuleID:      "e2e-csv-parse",
			Checker:     c.Name(),
			Severity:    SeverityError,
			File:        rel,
			Line:        1,
			Message:     "Failed to parse CSV header: " + err.Error(),
			Reason:      "Malformed CSV header prevents E2E test loading.",
			Remediation: "Ensure line 1 of `" + rel + "` is valid RFC 4180 CSV with standard 10 columns.",
		}}
	}

	headerMap := make(map[string]int, len(header))
	for i, h := range header {
		clean := strings.TrimSpace(strings.TrimPrefix(h, "\uFEFF"))
		headerMap[strings.ToLower(clean)] = i
	}

	var missingCols []string
	for _, req := range requiredCSVHeaders {
		if _, ok := headerMap[req]; !ok {
			missingCols = append(missingCols, req)
		}
	}
	if len(missingCols) > 0 {
		return []Diagnostic{{
			RuleID:      "e2e-csv-header",
			Checker:     c.Name(),
			Severity:    SeverityError,
			File:        rel,
			Line:        1,
			Message:     "CSV header is missing required column(s): " + strings.Join(missingCols, ", "),
			Reason:      "All E2E test CSV files must conform to the standard 10-column schema expected by `tools/e2e/runner`.",
			Remediation: "Use the standard header `" + strings.Join(requiredCSVHeaders, ",") + "` on line 1 of `" + rel + "`.",
		}}
	}

	expectedCols := len(header)
	var diags []Diagnostic
	lineNum := 1

	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		lineNum++
		if err != nil {
			diags = append(diags, Diagnostic{
				RuleID:      "e2e-csv-parse",
				Checker:     c.Name(),
				Severity:    SeverityError,
				File:        rel,
				Line:        lineNum,
				Message:     "CSV parse error: " + err.Error(),
				Reason:      "RFC 4180 syntax errors prevent deterministic E2E execution.",
				Remediation: "Quote fields containing commas or newlines properly at `" + fmt.Sprintf("%s:%d", rel, lineNum) + "`.",
			})
			continue
		}

		if isBlankCSVRow(rec) {
			continue
		}

		if len(rec) != expectedCols {
			diags = append(diags, Diagnostic{
				RuleID:      "e2e-csv-columns",
				Checker:     c.Name(),
				Severity:    SeverityError,
				File:        rel,
				Line:        lineNum,
				Message:     fmt.Sprintf("Row has %d column(s), expected %d", len(rec), expectedCols),
				Reason:      "Column count mismatch causes shifted fields (e.g. expected_stdout read as flags) in E2E runner.",
				Remediation: "Fix unquoted commas or missing columns at `" + fmt.Sprintf("%s:%d", rel, lineNum) + "` so the row has exactly " + fmt.Sprintf("%d", expectedCols) + " columns.",
			})
			continue
		}

		get := func(col string) string {
			return strings.TrimSpace(rec[headerMap[col]])
		}

		id := get("id")
		category := get("category")
		flags := get("flags")
		pipeInput := get("pipe_input")
		matchMode := get("match_mode")
		deductionMethod := get("deduction_method")
		description := get("description")

		if id == "" {
			diags = append(diags, Diagnostic{
				RuleID:      "e2e-csv-required-fields",
				Checker:     c.Name(),
				Severity:    SeverityError,
				File:        rel,
				Line:        lineNum,
				Message:     "Required column `id` is empty",
				Reason:      "Every E2E test case requires a unique non-empty `id` for traceability and failure triage.",
				Remediation: "Assign a unique test case ID in column `id` at `" + fmt.Sprintf("%s:%d", rel, lineNum) + "`.",
			})
		} else if prev, dup := seenIDs[id]; dup {
			diags = append(diags, Diagnostic{
				RuleID:      "e2e-csv-duplicate-id",
				Checker:     c.Name(),
				Severity:    SeverityError,
				File:        rel,
				Line:        lineNum,
				Message:     fmt.Sprintf("Duplicate E2E test case id `%s` (already defined at %s:%d)", id, prev.file, prev.line),
				Reason:      "Duplicate IDs make failure reports ambiguous and break case filtering.",
				Remediation: "Rename `id` at `" + fmt.Sprintf("%s:%d", rel, lineNum) + "` to a globally unique case ID.",
			})
		} else {
			seenIDs[id] = csvIDOrigin{file: rel, line: lineNum}
		}

		for _, reqField := range []struct{ name, val string }{
			{"category", category},
			{"match_mode", matchMode},
			{"description", description},
		} {
			if reqField.val == "" {
				diags = append(diags, Diagnostic{
					RuleID:      "e2e-csv-required-fields",
					Checker:     c.Name(),
					Severity:    SeverityError,
					File:        rel,
					Line:        lineNum,
					Message:     "Required column `" + reqField.name + "` is empty (id=`" + id + "`)",
					Reason:      "Required E2E metadata columns must be populated for reliable execution and reviewability.",
					Remediation: "Fill in `" + reqField.name + "` at `" + fmt.Sprintf("%s:%d", rel, lineNum) + "`.",
				})
			}
		}

		if flags == "" && pipeInput == "" {
			diags = append(diags, Diagnostic{
				RuleID:      "e2e-csv-required-fields",
				Checker:     c.Name(),
				Severity:    SeverityError,
				File:        rel,
				Line:        lineNum,
				Message:     "Both `flags` and `pipe_input` are empty (id=`" + id + "`)",
				Reason:      "An E2E test case without CLI flags or piped input does not execute any target expression.",
				Remediation: "Provide an expression in `flags` or `pipe_input` at `" + fmt.Sprintf("%s:%d", rel, lineNum) + "`.",
			})
		}

		if matchMode != "" && !validMatchModes[matchMode] {
			diags = append(diags, Diagnostic{
				RuleID:      "e2e-csv-enum",
				Checker:     c.Name(),
				Severity:    SeverityError,
				File:        rel,
				Line:        lineNum,
				Message:     "Invalid `match_mode` value `" + matchMode + "` (id=`" + id + "`)",
				Reason:      "`tools/e2e/runner` only supports match modes: exact, contains, regex, error.",
				Remediation: "Change `match_mode` to one of `exact`, `contains`, `regex`, or `error` at `" + fmt.Sprintf("%s:%d", rel, lineNum) + "`.",
			})
		}

		if !validDeductionMethods[deductionMethod] {
			diags = append(diags, Diagnostic{
				RuleID:      "e2e-csv-enum",
				Checker:     c.Name(),
				Severity:    SeverityError,
				File:        rel,
				Line:        lineNum,
				Message:     "Invalid `deduction_method` value `" + deductionMethod + "` (id=`" + id + "`)",
				Reason:      "Valid CAS deduction methods are: deductive, contradiction, contrapositive, inductive, metamorphic, roundtrip (or empty).",
				Remediation: "Set `deduction_method` to a valid enum value at `" + fmt.Sprintf("%s:%d", rel, lineNum) + "`.",
			})
		}
	}

	return diags
}

func isBlankCSVRow(rec []string) bool {
	for _, cell := range rec {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

func hasRelevantCSVStage(staged []string) bool {
	for _, f := range staged {
		if strings.HasPrefix(f, "tools/e2e/test-cases/") && strings.HasSuffix(strings.ToLower(f), ".csv") {
			return true
		}
	}
	return false
}
