package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
)

func TestHistory_GetHistoryFilePath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("IHD_DIR", tmpDir)

	path, err := GetHistoryFilePath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := filepath.Join(tmpDir, "history")
	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}
}

func TestHistory_LoadSaveAndRotate(t *testing.T) {
	tmpDir := t.TempDir()
	histFile := filepath.Join(tmpDir, "history")

	// 1. Initial load from non-existent file returns empty slice without error
	lines, err := LoadHistoryLines(histFile, 10)
	if err != nil {
		t.Fatalf("unexpected error on non-existent file: %v", err)
	}
	if len(lines) != 0 {
		t.Errorf("expected empty lines, got %v", lines)
	}

	// 2. Save 3 lines
	initial := []string{"1 + 1", "x = 5", "sin(pi)"}
	if err := SaveHistoryLines(histFile, initial, 10); err != nil {
		t.Fatalf("failed to save history: %v", err)
	}

	loaded, err := LoadHistoryLines(histFile, 10)
	if err != nil {
		t.Fatalf("failed to load history: %v", err)
	}
	if !reflect.DeepEqual(loaded, initial) {
		t.Errorf("expected %v, got %v", initial, loaded)
	}

	// 3. Save with rotation (limit = 2) keeps the latest 2 lines
	toSave := []string{"1 + 1", "x = 5", "sin(pi)", "y = 10"}
	if err := SaveHistoryLines(histFile, toSave, 2); err != nil {
		t.Fatalf("failed to save with limit: %v", err)
	}

	rotated, err := LoadHistoryLines(histFile, 10)
	if err != nil {
		t.Fatalf("failed to load rotated history: %v", err)
	}
	expectedRotated := []string{"sin(pi)", "y = 10"}
	if !reflect.DeepEqual(rotated, expectedRotated) {
		t.Errorf("expected rotated %v, got %v", expectedRotated, rotated)
	}
}

func TestHistory_AppendHistoryLine(t *testing.T) {
	tmpDir := t.TempDir()
	histFile := filepath.Join(tmpDir, "history")

	if err := AppendHistoryLine(histFile, "x = 42", 5); err != nil {
		t.Fatalf("AppendHistoryLine failed: %v", err)
	}
	if err := AppendHistoryLine(histFile, "y = 100", 5); err != nil {
		t.Fatalf("AppendHistoryLine failed: %v", err)
	}

	loaded, err := LoadHistoryLines(histFile, 5)
	if err != nil {
		t.Fatalf("LoadHistoryLines failed: %v", err)
	}
	expected := []string{"x = 42", "y = 100"}
	if !reflect.DeepEqual(loaded, expected) {
		t.Errorf("expected %v, got %v", expected, loaded)
	}
}

func TestHistory_AppendHistoryConcurrencyAndNoTrash(t *testing.T) {
	tmpDir := t.TempDir()
	histFile := filepath.Join(tmpDir, "history")

	// Append multiple lines in parallel to test concurrency safety
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			_ = AppendHistoryLine(histFile, "expr", 100)
			done <- true
		}(i)
	}
	for i := 0; i < 10; i++ {
		<-done
	}

	loaded, err := LoadHistoryLines(histFile, 100)
	if err != nil {
		t.Fatalf("LoadHistoryLines failed: %v", err)
	}
	if len(loaded) != 10 {
		t.Errorf("expected 10 lines from concurrent appends, got %d", len(loaded))
	}

	// Verify no .tmp files exist in the directory
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	for _, f := range files {
		if strings.Contains(f.Name(), ".tmp.") {
			t.Errorf("temporary file remained: %s", f.Name())
		}
	}
}


func TestSession_SaveLoadClear(t *testing.T) {
	tmpDir := t.TempDir()
	sessionFile := filepath.Join(tmpDir, "session.ihd")

	env := calc.NewEnv()
	// Set vars and assumptions
	val42, _ := calc.NewRational(42, 1)
	env.Set("my_var", val42)
	valAns, _ := calc.NewRational(999, 1)
	env.Set("ans", valAns)
	_ = env.Assumptions().Assume("x", calc.PropPositive)
	_ = env.Assumptions().Assume("n", calc.PropInteger)

	// 1. Save session
	if err := SaveSession(env, sessionFile); err != nil {
		t.Fatalf("SaveSession failed: %v", err)
	}

	// 2. Clear env
	env.Clear()
	if len(env.All()) != 0 {
		t.Errorf("expected empty vars after clear, got %v", env.All())
	}
	if env.IsPositive("x") {
		t.Errorf("expected assumption for x to be cleared")
	}

	// 3. Load session
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	if err := LoadSession(env, sessionFile, out, errOut); err != nil {
		t.Fatalf("LoadSession failed: %v", err)
	}

	// Verify my_var restored, ans was NOT saved/restored
	v, ok := env.Get("my_var")
	if !ok || v == nil || v.String() != "42" {
		t.Errorf("expected my_var=42, got %v", v)
	}
	if _, ok := env.Get("ans"); ok {
		t.Errorf("expected ans to not be restored")
	}

	// Verify assumptions restored
	if !env.IsPositive("x") {
		t.Errorf("expected x to be positive after load")
	}
	if !env.IsInteger("n") {
		t.Errorf("expected n to be integer after load")
	}

	// 4. Load session with invalid line reports error
	badFile := filepath.Join(tmpDir, "bad.ihd")
	_ = os.WriteFile(badFile, []byte("valid_var = 1\nunknown_func(((\n"), 0644)
	loadErr := LoadSession(env, badFile, out, errOut)
	if loadErr == nil {
		t.Errorf("expected error when loading invalid file")
	}
	if errOut.Len() == 0 {
		t.Errorf("expected error output on errOut")
	}
}

func TestSimilar_FindSimilarExpressions(t *testing.T) {
	history := []string{
		"1 + 1",
		"x^2 + 2*x + 1",
		"sqrt(2 + sqrt(3))",
		"sin(pi / 4)",
		"sqrt(5 + 2*sqrt(6))",
		"invalid syntax ((",
	}

	// 1. Query with similar nested radical sqrt(7 + 4*sqrt(3))
	matches, err := FindSimilarExpressions("sqrt(7 + 4*sqrt(3))", history, 3)
	if err != nil {
		t.Fatalf("FindSimilarExpressions failed: %v", err)
	}

	if len(matches) == 0 {
		t.Fatalf("expected matches, got 0")
	}

	// Top matches should be the sqrt nested expressions
	top := matches[0].Expr
	if top != "sqrt(2 + sqrt(3))" && top != "sqrt(5 + 2*sqrt(6))" {
		t.Errorf("expected top match to be nested sqrt, got %s (score: %f)", top, matches[0].Score)
	}

	// 2. Query with polynomial
	polyMatches, err := FindSimilarExpressions("y^2 + 4*y + 4", history, 3)
	if err != nil {
		t.Fatalf("FindSimilarExpressions failed: %v", err)
	}
	if len(polyMatches) == 0 || polyMatches[0].Expr != "x^2 + 2*x + 1" {
		t.Errorf("expected top match to be x^2 + 2*x + 1, got %v", polyMatches)
	}
}


