package main

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

var posRegex = regexp.MustCompile(`(?:position|pos|at|位置)\s*(\d+)`)

// LevenshteinDistance computes the classic edit distance between two strings.
func LevenshteinDistance(s1, s2 string) int {
	r1, r2 := []rune(s1), []rune(s2)
	len1, len2 := len(r1), len(r2)

	dp := make([][]int, len1+1)
	for i := range dp {
		dp[i] = make([]int, len2+1)
		dp[i][0] = i
	}
	for j := 0; j <= len2; j++ {
		dp[0][j] = j
	}

	for i := 1; i <= len1; i++ {
		for j := 1; j <= len2; j++ {
			cost := 1
			if r1[i-1] == r2[j-1] {
				cost = 0
			}
			dp[i][j] = min(
				dp[i-1][j]+1,      // deletion
				dp[i][j-1]+1,      // insertion
				dp[i-1][j-1]+cost, // substitution
			)
		}
	}
	return dp[len1][len2]
}

func min(a, b, c int) int {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}

// FindClosestFunction finds the closest registered function name within maxDist.
func FindClosestFunction(unknown string, maxDist int) string {
	unknown = strings.ToLower(strings.TrimSpace(unknown))
	if unknown == "" {
		return ""
	}

	allNames := calc.AllFunctionNames()
	bestName := ""
	minDist := math.MaxInt32

	for _, name := range allNames {
		lowerName := strings.ToLower(name)
		dist := LevenshteinDistance(unknown, lowerName)
		if dist < minDist && dist <= maxDist {
			minDist = dist
			bestName = name
		}
	}
	return bestName
}

// FormatErrorWithDiagnostics checks if the error is a syntax or unknown-function error
// and decorates it with caret pointer or "Did you mean?" suggestions.
func FormatErrorWithDiagnostics(err error, line string) string {
	if err == nil {
		return ""
	}
	errMsg := err.Error()

	// 1. Unknown function suggestion
	if strings.Contains(errMsg, "unknown function") || strings.Contains(errMsg, "未知の関数") {
		fields := strings.Fields(errMsg)
		if len(fields) > 0 {
			rawName := strings.Trim(fields[len(fields)-1], ":'\"()[]{}")
			// Suppress false positives on short identifiers (e.g. single-letter custom vars f(x), g(t))
			maxDist := 2
			if len(rawName) <= 1 {
				maxDist = 0
			} else if len(rawName) <= 3 {
				maxDist = 1
			}

			if maxDist > 0 {
				if suggestion := FindClosestFunction(rawName, maxDist); suggestion != "" {
					didYouMean := i18n.T("cli.err_did_you_mean", suggestion)
					if didYouMean == "cli.err_did_you_mean" {
						didYouMean = fmt.Sprintf("Did you mean '%s'?", suggestion)
					}
					errMsg = fmt.Sprintf("%s\n  %s", errMsg, didYouMean)
				}
			}
		}
	}

	// 2. Syntax error caret display
	pos := extractErrorPosition(errMsg)
	if pos >= 0 && len(line) > 0 {
		caretLine := buildCaretLine(line, pos)
		if caretLine != "" {
			errMsg = fmt.Sprintf("%s\n  %s\n  %s", errMsg, line, caretLine)
		}
	}

	return errMsg
}

func extractErrorPosition(msg string) int {
	matches := posRegex.FindStringSubmatch(msg)
	if len(matches) >= 2 {
		if p, err := strconv.Atoi(matches[1]); err == nil {
			return p
		}
	}
	return -1
}

func buildCaretLine(line string, bytePos int) string {
	if bytePos < 0 {
		bytePos = 0
	}
	if bytePos > len(line) {
		bytePos = len(line)
	}

	prefix := line[:bytePos]
	runes := []rune(prefix)

	var sb strings.Builder
	for _, r := range runes {
		if r == '\t' {
			sb.WriteString("    ")
		} else {
			sb.WriteRune(' ')
		}
	}
	sb.WriteRune('^')
	return sb.String()
}
