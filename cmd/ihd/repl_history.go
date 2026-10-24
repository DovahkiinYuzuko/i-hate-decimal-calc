package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
)

var historyMu sync.Mutex

const defaultMaxHistoryLines = 1000

// GetHistoryFilePath returns the absolute path to the history file (~/.ihd/history or $IHD_DIR/history).
func GetHistoryFilePath() (string, error) {
	if custom := os.Getenv("IHD_DIR"); custom != "" {
		return filepath.Join(custom, "history"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ihd", "history"), nil
}

// LoadHistoryLines reads up to maxLines history records from the given file.
func LoadHistoryLines(filePath string, maxLines int) ([]string, error) {
	historyMu.Lock()
	defer historyMu.Unlock()

	if maxLines <= 0 {
		maxLines = defaultMaxHistoryLines
	}

	f, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		text := strings.TrimRight(scanner.Text(), "\r\n")
		if strings.TrimSpace(text) != "" {
			lines = append(lines, text)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	return lines, nil
}

// SaveHistoryLines writes lines to filePath with a line limit, ensuring parent directory exists.
func SaveHistoryLines(filePath string, lines []string, maxLines int) error {
	historyMu.Lock()
	defer historyMu.Unlock()

	if maxLines <= 0 {
		maxLines = defaultMaxHistoryLines
	}

	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	tmpFile := fmt.Sprintf("%s.tmp.%d", filePath, os.Getpid())
	defer func() {
		_ = os.Remove(tmpFile)
	}()

	f, err := os.OpenFile(tmpFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}

	writer := bufio.NewWriter(f)
	for _, l := range lines {
		if _, err := writer.WriteString(l + "\n"); err != nil {
			_ = f.Close()
			return err
		}
	}
	if err := writer.Flush(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	// Rename atomically
	return os.Rename(tmpFile, filePath)
}

// AppendHistoryLine appends a single line directly to the history file in O(1) time using O_APPEND.
func AppendHistoryLine(filePath string, line string, maxLines int) error {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return nil
	}

	historyMu.Lock()
	defer historyMu.Unlock()

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(trimmed + "\n")
	return err
}

// SaveSession exports variables and assumptions from env into a runnable .ihd script atomically.
func SaveSession(env *calc.Env, filePath string) error {
	if env == nil {
		return fmt.Errorf("env is nil")
	}

	dir := filepath.Dir(filePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	tmpFile := fmt.Sprintf("%s.tmp.%d", filePath, os.Getpid())
	defer func() {
		_ = os.Remove(tmpFile)
	}()

	f, err := os.OpenFile(tmpFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	writer := bufio.NewWriter(f)

	// 1. Export Assumptions first
	if env.Assumptions() != nil {
		for _, stmt := range env.Assumptions().ExportStatements() {
			if _, err := writer.WriteString(stmt + "\n"); err != nil {
				_ = f.Close()
				return err
			}
		}
	}

	// 2. Export variables (excluding 'ans')
	vars := env.All()
	var names []string
	for k := range vars {
		if k != "ans" {
			names = append(names, k)
		}
	}
	sort.Strings(names)

	for _, name := range names {
		node := vars[name]
		if node != nil {
			line := fmt.Sprintf("%s = %s\n", name, node.String())
			if _, err := writer.WriteString(line); err != nil {
				_ = f.Close()
				return err
			}
		}
	}

	if err := writer.Flush(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	return os.Rename(tmpFile, filePath)
}


// LoadSession reads an .ihd script and executes lines against env.
func LoadSession(env *calc.Env, filePath string, out, errOut io.Writer) error {
	if env == nil {
		return fmt.Errorf("env is nil")
	}

	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	ro := runOptions{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		_ = evaluateLineInternal(line, ro, env, true, out, errOut, "")
	}
	env.Delete("ans")
	return scanner.Err()
}

// SimilarityMatch represents a matched expression with a similarity score in [0.0, 1.0].
type SimilarityMatch struct {
	Expr  string
	Score float64
}

// ExtractSkeletonKey produces a structural signature string abstracting variable names and numbers.
func ExtractSkeletonKey(n calc.Node) string {
	if n == nil {
		return ""
	}
	switch v := n.(type) {
	case *calc.VarNode:
		return "var"
	case *calc.RationalNode:
		return "num"
	case *calc.AddNode:
		var parts []string
		for _, t := range v.Terms {
			parts = append(parts, ExtractSkeletonKey(t))
		}
		sort.Strings(parts)
		return fmt.Sprintf("Add(%s)", strings.Join(parts, ","))
	case *calc.MulNode:
		var parts []string
		for _, f := range v.Factors {
			parts = append(parts, ExtractSkeletonKey(f))
		}
		sort.Strings(parts)
		return fmt.Sprintf("Mul(%s)", strings.Join(parts, ","))
	case *calc.PowNode:
		return fmt.Sprintf("Pow(%s,%s)", ExtractSkeletonKey(v.Base), ExtractSkeletonKey(v.Exp))
	case *calc.FuncNode:
		var parts []string
		for _, a := range v.Args {
			parts = append(parts, ExtractSkeletonKey(a))
		}
		return fmt.Sprintf("Func[%s](%s)", v.Name, strings.Join(parts, ","))
	case *calc.RelOpNode:
		return fmt.Sprintf("Rel[%s](%s,%s)", v.Op, ExtractSkeletonKey(v.LHS), ExtractSkeletonKey(v.RHS))
	default:
		return fmt.Sprintf("%T", n)
	}
}

// ComputeSkeletonSimilarity computes structural similarity between two AST nodes.
func ComputeSkeletonSimilarity(n1, n2 calc.Node) float64 {
	if n1 == nil || n2 == nil {
		return 0.0
	}
	k1 := ExtractSkeletonKey(n1)
	k2 := ExtractSkeletonKey(n2)
	if k1 == k2 {
		return 1.0
	}

	// Tokenize keys by delimiters to compute Jaccard index
	tokens1 := tokenizeSkeleton(k1)
	tokens2 := tokenizeSkeleton(k2)
	if len(tokens1) == 0 && len(tokens2) == 0 {
		return 1.0
	}
	if len(tokens1) == 0 || len(tokens2) == 0 {
		return 0.0
	}

	freq1 := make(map[string]int)
	for _, t := range tokens1 {
		freq1[t]++
	}
	freq2 := make(map[string]int)
	for _, t := range tokens2 {
		freq2[t]++
	}

	intersection := 0
	union := 0
	allKeys := make(map[string]bool)
	for k := range freq1 {
		allKeys[k] = true
	}
	for k := range freq2 {
		allKeys[k] = true
	}

	for k := range allKeys {
		c1 := freq1[k]
		c2 := freq2[k]
		if c1 < c2 {
			intersection += c1
			union += c2
		} else {
			intersection += c2
			union += c1
		}
	}

	if union == 0 {
		return 0.0
	}
	return float64(intersection) / float64(union)
}

func tokenizeSkeleton(s string) []string {
	var tokens []string
	var cur strings.Builder
	for _, r := range s {
		if r == '(' || r == ')' || r == '[' || r == ']' || r == ',' {
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
			tokens = append(tokens, string(r))
		} else {
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}
	return tokens
}

// FindSimilarExpressions parses queryExpr and scores expressions in historyLines.
func FindSimilarExpressions(queryExpr string, historyLines []string, topN int) ([]SimilarityMatch, error) {
	parsedQuery, err := calc.Parse(queryExpr)
	if err != nil {
		return nil, err
	}

	if topN <= 0 {
		topN = 5
	}

	seen := make(map[string]bool)
	var matches []SimilarityMatch

	for i := len(historyLines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(historyLines[i])
		if line == "" || line == queryExpr || seen[line] {
			continue
		}
		seen[line] = true

		// Check if line is an assignment (e.g. "x = expr")
		exprToParse := line
		if parsedStmt, err := calc.ParseStatement(line); err == nil {
			if assign, ok := parsedStmt.(*calc.AssignStmt); ok {
				if assign.Value != nil {
					exprToParse = assign.Value.String()
				}
			}
		}

		candNode, err := calc.Parse(exprToParse)
		if err != nil {
			continue
		}

		score := ComputeSkeletonSimilarity(parsedQuery, candNode)
		if score > 0.05 {
			matches = append(matches, SimilarityMatch{
				Expr:  line,
				Score: score,
			})
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Score != matches[j].Score {
			return matches[i].Score > matches[j].Score
		}
		return i < j
	})

	if len(matches) > topN {
		matches = matches[:topN]
	}
	return matches, nil
}


