package calc

import (
	"bytes"
	"math/big"
	"testing"
)

func TestWriteIntDecimalSmallAndMedium(t *testing.T) {
	testCases := []string{
		"0",
		"1",
		"-1",
		"42",
		"-42",
		"12345678901234567890",
		"-98765432109876543210",
	}

	for _, tc := range testCases {
		val, ok := new(big.Int).SetString(tc, 10)
		if !ok {
			t.Fatalf("failed to parse big.Int from %s", tc)
		}

		var buf bytes.Buffer
		if err := WriteIntDecimal(&buf, val); err != nil {
			t.Fatalf("WriteIntDecimal failed for %s: %v", tc, err)
		}

		got := buf.String()
		expected := val.String()
		if got != expected {
			t.Errorf("mismatch for %s:\ngot:      %s\nexpected: %s", tc, got, expected)
		}
	}
}

func TestWriteIntDecimalDivideAndConquerCrossingCutoff(t *testing.T) {
	// Generate numbers that cross the leaf threshold (e.g. 2^30000, 2^50000)
	exponents := []uint{100, 1000, 10000, 30000, 50000}

	for _, exp := range exponents {
		val := new(big.Int).Lsh(big.NewInt(1), exp)

		var buf bytes.Buffer
		if err := WriteIntDecimal(&buf, val); err != nil {
			t.Fatalf("WriteIntDecimal failed for 2^%d: %v", exp, err)
		}

		got := buf.String()
		expected := val.String()
		if got != expected {
			t.Errorf("mismatch for 2^%d (len got=%d, expected=%d)", exp, len(got), len(expected))
			if len(got) == len(expected) {
				// Find first mismatch
				for i := 0; i < len(got); i++ {
					if got[i] != expected[i] {
						t.Errorf("first mismatch at index %d: got %c, expected %c", i, got[i], expected[i])
						break
					}
				}
			}
		}
	}
}

func TestWriteNodeRational(t *testing.T) {
	node := &RationalNode{Val: big.NewRat(355, 113)}
	var buf bytes.Buffer
	if err := WriteNode(&buf, node, FormatOptions{}); err != nil {
		t.Fatalf("WriteNode failed: %v", err)
	}

	got := buf.String()
	expected := "355/113"
	if got != expected {
		t.Errorf("got %s, expected %s", got, expected)
	}
}

func TestStreamFSMTransitions(t *testing.T) {
	fsm := NewStreamFSM()
	if fsm.CurrentState != StateInit {
		t.Fatalf("expected StateInit, got %s", fsm.CurrentState)
	}

	if err := fsm.Transition(StateDirect); err != nil {
		t.Fatalf("transition to StateDirect failed: %v", err)
	}

	if err := fsm.Transition(StateDone); err != nil {
		t.Fatalf("transition to StateDone failed: %v", err)
	}

	// Invalid transition: StateDone -> StateSplitting
	if err := fsm.Transition(StateSplitting); err == nil {
		t.Fatalf("expected error for illegal transition from StateDone to StateSplitting, got nil")
	}
}

func TestComputeExactDecimalDigits(t *testing.T) {
	testValues := []int64{0, 1, 9, 10, 99, 100, 999, 1000, 1024, 999999999}
	for _, v := range testValues {
		b := big.NewInt(v)
		expected := len(b.String())
		got := ComputeExactDecimalDigits(b)
		if got != expected {
			t.Errorf("ComputeExactDecimalDigits(%d) = %d, expected %d", v, got, expected)
		}
	}

	// Test larger powers of two against string length
	for exp := uint(1); exp <= 2000; exp += 73 {
		b := new(big.Int).Lsh(big.NewInt(1), exp)
		expected := len(b.String())
		got := ComputeExactDecimalDigits(b)
		if got != expected {
			t.Errorf("ComputeExactDecimalDigits(2^%d) = %d, expected %d", exp, got, expected)
		}
	}

	// Test 2^(2^20) = 2^1048576 without materializing string
	// floor(1048576 * log10(2)) + 1 = 315653
	hugeExp := uint(1048576)
	hugeVal := new(big.Int).Lsh(big.NewInt(1), hugeExp)
	gotHuge := ComputeExactDecimalDigits(hugeVal)
	if gotHuge != 315653 {
		t.Errorf("ComputeExactDecimalDigits(2^1048576) = %d, expected 315653", gotHuge)
	}
}

func TestEvalDigitCountIntegration(t *testing.T) {
	env := NewEnv()
	expr, err := Parse("digit_count(10^5)")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	res, err := EvalWithEnv(expr, env)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}

	rat, ok := res.(*RationalNode)
	if !ok || !rat.Val.IsInt() {
		t.Fatalf("expected integer result, got %T (%v)", res, res)
	}

	if rat.Val.Num().Int64() != 6 {
		t.Errorf("expected 6 for digit_count(10^5), got %s", rat.String())
	}
}

func TestSplitSRT(t *testing.T) {
	// Test splitSRT across various powers of 10
	kValues := []int{10, 50, 100, 500, 1000, 4096}
	for _, k := range kValues {
		tenK := getPowerOfTen(k)

		// Test with x = 2 * 10^k + 12345
		x1 := new(big.Int).Mul(big.NewInt(2), tenK)
		x1.Add(x1, big.NewInt(12345))

		wantQ1 := new(big.Int)
		wantR1 := new(big.Int)
		wantQ1.QuoRem(x1, tenK, wantR1)

		gotQ1, gotR1 := splitSRT(x1, tenK, k)
		if gotQ1.Cmp(wantQ1) != 0 || gotR1.Cmp(wantR1) != 0 {
			t.Fatalf("splitSRT mismatch for x1 at k=%d:\ngot Q=%s, R=%s\nwant Q=%s, R=%s", k, gotQ1, gotR1, wantQ1, wantR1)
		}

		// Test with random large number x = 10^(2k) - 7
		x2 := new(big.Int).Sub(getPowerOfTen(2*k), big.NewInt(7))
		wantQ2 := new(big.Int)
		wantR2 := new(big.Int)
		wantQ2.QuoRem(x2, tenK, wantR2)

		gotQ2, gotR2 := splitSRT(x2, tenK, k)
		if gotQ2.Cmp(wantQ2) != 0 || gotR2.Cmp(wantR2) != 0 {
			t.Fatalf("splitSRT mismatch for x2 at k=%d:\ngot Q=%s, R=%s\nwant Q=%s, R=%s", k, gotQ2, gotR2, wantQ2, wantR2)
		}
	}
}


