package calc

import (
	"strings"
	"testing"
)

func TestApprox_Basic(t *testing.T) {
	testCases := []struct {
		input    string
		contains string
	}{
		{"1/2", "0.5"},
		{"1/3", "0.333333333333333333333333333333"},
		{"sqrt(2)", "1.414213562373095048801688724210"},
		{"pi", "3.141592653589793238462643383280"},
		{"e", "2.718281828459045235360287471353"},
		{"sin(pi/2)", "1"},
		{"cos(0)", "1"},
		{"ln(e)", "1"},
		{"2^10", "1024"},
		{"5!", "120"},
		{"3 + 4*i", "3 + 4i"},
		{"3 - 4*i", "3 - 4i"},
		{"cbrt(2)", "1.259921049894873164767210607278"},
	}

	for _, tc := range testCases {
		node, err := EvalString(tc.input)
		if err != nil {
			t.Fatalf("eval error for %q: %v", tc.input, err)
		}
		app, err := Approx(node)
		if err != nil {
			t.Fatalf("approx error for %q: %v", tc.input, err)
		}
		if !strings.Contains(app, tc.contains) {
			t.Errorf("input %q: expected approx to contain %q, got %q", tc.input, tc.contains, app)
		}
	}
}

func TestApproxWithDigits_HighPrecision(t *testing.T) {
	node, err := EvalString("sqrt(2)")
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}

	// 50 digits test
	app50, err := ApproxWithDigits(node, 50)
	if err != nil {
		t.Fatalf("approx error 50 digits: %v", err)
	}
	expected50 := "1.41421356237309504880168872420969807856967187537695"
	if !strings.Contains(app50, expected50) {
		t.Errorf("expected 50 digits of sqrt(2) to contain %s, got %s", expected50, app50)
	}
	if !strings.Contains(app50, "(± ") {
		t.Errorf("expected certified error bound (± ...), got %s", app50)
	}
}

func TestApprox_ArbitraryPrecisionOverflowUnderflow(t *testing.T) {
	// 171! overflows float64 to +Inf, but must succeed in arbitrary precision
	node171, err := EvalString("171!")
	if err != nil {
		t.Fatalf("eval error for 171!: %v", err)
	}
	app171, err := Approx(node171)
	if err != nil {
		t.Fatalf("approx error for 171!: %v", err)
	}
	if strings.Contains(app171, "+Inf") || strings.Contains(app171, "NaN") {
		t.Errorf("171! must not overflow to +Inf or NaN in arbitrary precision, got %s", app171)
	}
	// 171! exact value prefix
	if !strings.HasPrefix(app171, "1241018070217667823424840524103103992616605577501693185388951803611996075221691752992751978120487585576464959501670387052809889858690710767331242032218484364310473577889968548278290754541561964852153468318044293239598173696899657235903947616152278558180061176365108428800000000000000000000000000000000000000000") {
		t.Errorf("171! unexpected value: %s", app171)
	}

	// 2^-1000 underflows float64 to 0, but must not be 0 in arbitrary precision
	nodeUnder, err := EvalString("2^(-1000)")
	if err != nil {
		t.Fatalf("eval error for 2^-1000: %v", err)
	}
	appUnder, err := Approx(nodeUnder)
	if err != nil {
		t.Fatalf("approx error for 2^-1000: %v", err)
	}
	if strings.HasPrefix(appUnder, "0.") && !strings.Contains(appUnder, "e-") && strings.Trim(appUnder, "0. (±)") == "" {
		t.Errorf("2^-1000 must not underflow to zero: %s", appUnder)
	}
}

func TestApprox_CollectionsAndMatrix(t *testing.T) {
	// List of solutions
	nodeList, err := EvalString("[sqrt(2), sqrt(3)]")
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	appList, err := Approx(nodeList)
	if err != nil {
		t.Fatalf("approx error list: %v", err)
	}
	if !strings.Contains(appList, "1.41421356") || !strings.Contains(appList, "1.73205080") {
		t.Errorf("list approx unexpected: %s", appList)
	}
}
