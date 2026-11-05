package runner

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// LoadTestCases walks the given root directory recursively, discovers all CSV files,
// and parses them into TestCase structs based on the filters.
func LoadTestCases(rootDir string, categoryFilter string, nameFilter string) ([]TestCase, error) {
	var cases []TestCase

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(strings.ToLower(info.Name()), ".csv") {
			return nil
		}

		fileCases, err := parseCSVFile(path, categoryFilter, nameFilter)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", path, err)
		}
		cases = append(cases, fileCases...)
		return nil
	})

	if err != nil {
		return nil, err
	}
	return cases, nil
}

func parseCSVFile(filePath string, categoryFilter string, nameFilter string) ([]TestCase, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	r := csv.NewReader(file)
	r.FieldsPerRecord = -1 // Allow variable fields if needed, but header map ensures safety
	r.LazyQuotes = true

	header, err := r.Read()
	if err != nil {
		if err == io.EOF {
			return nil, nil // Empty file
		}
		return nil, err
	}

	headerMap := make(map[string]int)
	for i, h := range header {
		headerMap[strings.TrimSpace(strings.ToLower(h))] = i
	}

	var results []TestCase
	lineNum := 1

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		lineNum++
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}

		getField := func(col string) string {
			if idx, ok := headerMap[col]; ok && idx < len(record) {
				return normalizeNewlines(record[idx])
			}
			return ""
		}

		id := getField("id")
		category := getField("category")
		flagsStr := getField("flags")
		pipeInput := getField("pipe_input")
		matchMode := MatchMode(getField("match_mode"))
		expectedStdout := getField("expected_stdout")
		expectedStderr := getField("expected_stderr")
		expectedExitStr := getField("expected_exit_code")
		deductionMethod := getField("deduction_method")
		description := getField("description")

		if id == "" && description == "" && flagsStr == "" && pipeInput == "" {
			continue // Skip blank rows
		}

		// Apply filters
		if categoryFilter != "" {
			matched := strings.Contains(category, categoryFilter) || strings.Contains(filePath, categoryFilter)
			if !matched {
				continue
			}
		}
		if nameFilter != "" {
			matched := strings.Contains(id, nameFilter) || strings.Contains(description, nameFilter)
			if !matched {
				continue
			}
		}

		expectedExit := 0
		if expectedExitStr != "" {
			if n, err := strconv.Atoi(strings.TrimSpace(expectedExitStr)); err == nil {
				expectedExit = n
			}
		}

		if matchMode == "" {
			if expectedStderr != "" {
				matchMode = MatchError
			} else {
				matchMode = MatchExact
			}
		}

		flags := parseFlags(flagsStr)

		tc := TestCase{
			ID:              id,
			Category:        category,
			Flags:           flags,
			PipeInput:       pipeInput,
			MatchMode:       matchMode,
			ExpectedStdout:  strings.TrimSpace(expectedStdout),
			ExpectedStderr:  strings.TrimSpace(expectedStderr),
			ExpectedExit:    expectedExit,
			DeductionMethod: deductionMethod,
			Description:     description,
			Timeout:         15 * time.Second, // Default safe timeout
			SourceFile:      filePath,
			SourceLine:      lineNum,
		}

		results = append(results, tc)
	}

	return results, nil
}

// normalizeNewlines converts all CRLF to LF.
func normalizeNewlines(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

// parseFlags parses a command line flag string into separate arguments,
// preserving internal quoted substrings.
func parseFlags(s string) []string {
	var args []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for _, r := range s {
		switch {
		case (r == '"' || r == '\'') && !inQuote:
			inQuote = true
			quoteChar = r
			current.WriteRune(r)
		case inQuote && r == quoteChar:
			inQuote = false
			quoteChar = 0
			current.WriteRune(r)
		case !inQuote && (r == ' ' || r == '\t'):
			if current.Len() > 0 {
				args = append(args, unwrapOuterQuotes(current.String()))
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		args = append(args, unwrapOuterQuotes(current.String()))
	}
	return args
}

func unwrapOuterQuotes(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
