package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	var seed int64
	var count int
	var outDir string

	flag.Int64Var(&seed, "seed", 42, "Deterministic random seed")
	flag.IntVar(&count, "count", 20, "Number of test cases to generate per domain")
	flag.StringVar(&outDir, "out", "tools/e2e/test-cases/08_metamorphic_roundtrip", "Output directory for generated CSV cases")
	flag.Parse()

	r := rand.New(rand.NewSource(seed))

	if err := os.MkdirAll(outDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create directory %s: %v\n", outDir, err)
		os.Exit(1)
	}

	headers := []string{
		"id", "category", "flags", "pipe_input", "match_mode",
		"expected_stdout", "expected_stderr", "expected_exit_code",
		"deduction_method", "description",
	}

	// 1. Generate Metamorphic Roundtrip Cases (factor <-> expand, diff <-> integrate)
	roundtripCSV := filepath.Join(outDir, "01_metamorphic_roundtrip_pairs.csv")
	var records [][]string
	records = append(records, headers)

	// A. Expand <-> Factor Round-tripping
	for i := 1; i <= count/2; i++ {
		r1 := r.Intn(7) - 3
		r2 := r.Intn(7) - 3
		if r1 == 0 {
			r1 = 1
		}
		if r2 == 0 {
			r2 = -2
		}
		// (x - r1)*(x - r2) = x^2 - (r1+r2)x + r1*r2
		expr := fmt.Sprintf("expand(factor((x - (%d))*(x - (%d))))", r1, r2)
		// Expected expand result
		expected := fmt.Sprintf("expand((x - (%d))*(x - (%d)))", r1, r2)

		records = append(records, []string{
			fmt.Sprintf("MET-ALG-%03d", i),
			"cat_88_metamorphic_roundtrip",
			fmt.Sprintf("\"%s\"", expr),
			"",
			"exact",
			// Note: ihd expands this cleanly
			expandQuadratic(r1, r2),
			"",
			"0",
			"inductive",
			fmt.Sprintf("メタモルフィック: 多項式の因数分解と展開の往復写像 (roots: %d, %d)", r1, r2),
		})
		_ = expected
	}

	// B. Diff <-> Integrate Round-tripping
	powers := []int{2, 3, 4, 5}
	for i, p := range powers {
		// diff(integrate(x^p, x), x) == x^p
		expr := fmt.Sprintf("diff(integrate(x^%d, x), x)", p)
		expected := fmt.Sprintf("x^%d", p)

		records = append(records, []string{
			fmt.Sprintf("MET-CAL-%03d", i+1),
			"cat_88_metamorphic_roundtrip",
			fmt.Sprintf("\"%s\"", expr),
			"",
			"exact",
			expected,
			"",
			"0",
			"inductive",
			fmt.Sprintf("メタモルフィック: べき乗多項式 x^%d の微積分往復恒等性", p),
		})
	}

	// C. Sturm Theorem Witness Cases (Dogfooding internal verification)
	for i := 1; i <= 5; i++ {
		// (x - a)^2 + 1 > 0 everywhere
		offset := i * 2
		expr := fmt.Sprintf("root_count((x - %d)^2 + 1, x, -100, 100)", offset)
		records = append(records, []string{
			fmt.Sprintf("STURM-WIT-%03d", i),
			"cat_88_metamorphic_roundtrip",
			fmt.Sprintf("\"%s\"", expr),
			"",
			"exact",
			"0", // 0 real roots in interval
			"",
			"0",
			"deductive",
			fmt.Sprintf("自社証明: スツルム列による区間 [-100, 100] における実根非存在の決定論的証明 ((x-%d)^2+1)", offset),
		})
	}

	f, err := os.Create(roundtripCSV)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create %s: %v\n", roundtripCSV, err)
		os.Exit(1)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	if err := w.WriteAll(records); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write CSV: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated %d metamorphic & dogfooding test cases into %s\n", len(records)-1, roundtripCSV)
}

func expandQuadratic(r1, r2 int) string {
	b := -(r1 + r2)
	c := r1 * r2

	var parts []string
	if c != 0 {
		parts = append(parts, fmt.Sprintf("%d", c))
	}

	if b > 0 {
		if b == 1 {
			if len(parts) > 0 {
				parts = append(parts, "+ x")
			} else {
				parts = append(parts, "x")
			}
		} else {
			if len(parts) > 0 {
				parts = append(parts, fmt.Sprintf("+ %d*x", b))
			} else {
				parts = append(parts, fmt.Sprintf("%d*x", b))
			}
		}
	} else if b < 0 {
		if b == -1 {
			parts = append(parts, "- x")
		} else {
			parts = append(parts, fmt.Sprintf("- %d*x", -b))
		}
	}

	if len(parts) > 0 {
		parts = append(parts, "+ x^2")
	} else {
		parts = append(parts, "x^2")
	}

	return strings.Join(parts, " ")
}
