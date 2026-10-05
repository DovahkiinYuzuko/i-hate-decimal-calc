package calc

import (
	"testing"
)

func TestEvalLambertW_SpecialValues(t *testing.T) {
	// lambert_w(0) => 0
	res0, err := EvalString("lambert_w(0)")
	if err != nil {
		t.Fatalf("unexpected error for lambert_w(0): %v", err)
	}
	if Format(res0) != "0" {
		t.Errorf("expected 0, got %s", Format(res0))
	}

	// lambert_w(e) => 1
	resE, err := EvalString("lambert_w(e)")
	if err != nil {
		t.Fatalf("unexpected error for lambert_w(e): %v", err)
	}
	if Format(resE) != "1" {
		t.Errorf("expected 1, got %s", Format(resE))
	}

	// lambert_w(x) keeps symbolic
	resSym, err := EvalString("lambert_w(x)")
	if err != nil {
		t.Fatalf("unexpected error for lambert_w(x): %v", err)
	}
	if Format(resSym) != "lambert_w(x)" {
		t.Errorf("expected lambert_w(x), got %s", Format(resSym))
	}
}

func TestEvalErf_SpecialValues(t *testing.T) {
	// erf(0) => 0
	res0, err := EvalString("erf(0)")
	if err != nil {
		t.Fatalf("unexpected error for erf(0): %v", err)
	}
	if Format(res0) != "0" {
		t.Errorf("expected 0, got %s", Format(res0))
	}

	// erf(-x) => -erf(x)
	resOdd, err := EvalString("erf(-x)")
	if err != nil {
		t.Fatalf("unexpected error for erf(-x): %v", err)
	}
	if Format(resOdd) != "-erf(x)" {
		t.Errorf("expected -erf(x), got %s", Format(resOdd))
	}
}

func TestDifferentiateTranscendental(t *testing.T) {
	// diff(lambert_w(x), x)
	d1, err := EvalString("diff(lambert_w(x), x)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	formatted1 := Format(d1)
	if formatted1 == "" || formatted1 == "0" {
		t.Errorf("expected valid derivative, got %s", formatted1)
	}

	// diff(erf(x), x) => 2*exp(-x^2)/sqrt(pi) or equivalent
	d2, err := EvalString("diff(erf(x), x)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	formatted2 := Format(d2)
	if formatted2 == "" || formatted2 == "0" {
		t.Errorf("expected valid derivative for erf, got %s", formatted2)
	}
}

func TestIntegrateGaussianAndLambert(t *testing.T) {
	// integrate(exp(-x^2), x)
	g1, err := EvalString("integrate(exp(-x^2), x)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	fmtG1 := Format(g1)
	if fmtG1 == "" {
		t.Errorf("expected non-empty integral result for exp(-x^2)")
	}

	// integrate(lambert_w(x), x)
	w1, err := EvalString("integrate(lambert_w(x), x)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	fmtW1 := Format(w1)
	if fmtW1 == "" {
		t.Errorf("expected non-empty integral result for lambert_w(x)")
	}
}

func TestSolveTranscendental(t *testing.T) {
	// solve(x*exp(x) == 2, x) => [lambert_w(2)]
	s1, err := EvalString("solve(x*exp(x) == 2, x)")
	if err != nil {
		t.Fatalf("unexpected error for x*exp(x) == 2: %v", err)
	}
	if Format(s1) != "[lambert_w(2)]" {
		t.Errorf("expected [lambert_w(2)], got %s", Format(s1))
	}

	// solve(x + ln(x) == 0, x) => [lambert_w(1)]
	s2, err := EvalString("solve(x + ln(x) == 0, x)")
	if err != nil {
		t.Fatalf("unexpected error for x + ln(x) == 0: %v", err)
	}
	if Format(s2) != "[lambert_w(1)]" {
		t.Errorf("expected [lambert_w(1)], got %s", Format(s2))
	}

	// solve(x^x == 2, x) => [exp(lambert_w(ln(2)))]
	s3, err := EvalString("solve(x^x == 2, x)")
	if err != nil {
		t.Fatalf("unexpected error for x^x == 2: %v", err)
	}
	fmtS3 := Format(s3)
	if fmtS3 != "[exp(lambert_w(ln(2)))]" {
		t.Errorf("expected [exp(lambert_w(ln(2)))], got %s", fmtS3)
	}
}

func TestVerifyTranscendental(t *testing.T) {
	// Verify solve(x*exp(x) == 2, x)
	cert, err := EvalString("verify(solve(x*exp(x) == 2, x))")
	if err != nil {
		t.Fatalf("unexpected error for verify: %v", err)
	}
	fmtCert := Format(cert)
	if fmtCert == "" {
		t.Errorf("expected non-empty verify certificate, got %s", fmtCert)
	}

	// Verify integrate(exp(-x^2), x)
	certInt, err := EvalString("verify(integrate(exp(-x^2), x))")
	if err != nil {
		t.Fatalf("unexpected error for verify integrate: %v", err)
	}
	fmtCertInt := Format(certInt)
	if fmtCertInt == "" {
		t.Errorf("expected non-empty verify integrate certificate, got %s", fmtCertInt)
	}
}
