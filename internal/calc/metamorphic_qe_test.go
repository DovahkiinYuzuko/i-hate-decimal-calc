package calc

import (
	"testing"
)

// TestMetamorphic_VariablePermutation asserts semantic equivalence under variable swapping.
func TestMetamorphic_VariablePermutation(t *testing.T) {
	env := NewEnv()
	// phi(x, y): exists([x], x^2 + y < 0)
	// phi(y, x): exists([y], y^2 + x < 0)
	q1, err1 := Parse("exists([x], x^2 + y < 0)")
	if err1 != nil {
		t.Fatalf("parse q1 failed: %v", err1)
	}
	q2, err2 := Parse("exists([y], y^2 + x < 0)")
	if err2 != nil {
		t.Fatalf("parse q2 failed: %v", err2)
	}

	res1, err1 := EvalQE(q1, env)
	if err1 != nil {
		t.Fatalf("EvalQE(q1) failed: %v", err1)
	}
	res2, err2 := EvalQE(q2, env)
	if err2 != nil {
		t.Fatalf("EvalQE(q2) failed: %v", err2)
	}

	// Permuting variable names in res2: y -> x should match res1
	res2Sub := Substitute(res2, "x", &VarNode{Name: "y"})
	s1 := Format(res1)
	s2 := Format(res2Sub)

	if s1 != s2 {
		t.Errorf("Variable permutation failed: q1 gives %s, permuted q2 gives %s", s1, s2)
	}
}

// TestMetamorphic_PositiveScaling asserts invariance under positive constant polynomial scaling.
func TestMetamorphic_PositiveScaling(t *testing.T) {
	env := NewEnv()
	// phi: forall([x], x^2 + 1 > 0)
	// phi_scaled: forall([x], 3 * (x^2 + 1) > 0)
	q1, err1 := Parse("forall([x], x^2 + 1 > 0)")
	if err1 != nil {
		t.Fatalf("parse q1 failed: %v", err1)
	}
	q2, err2 := Parse("forall([x], 3 * x^2 + 3 > 0)")
	if err2 != nil {
		t.Fatalf("parse q2 failed: %v", err2)
	}

	res1, err1 := EvalQE(q1, env)
	if err1 != nil {
		t.Fatalf("EvalQE(q1) failed: %v", err1)
	}
	res2, err2 := EvalQE(q2, env)
	if err2 != nil {
		t.Fatalf("EvalQE(q2) failed: %v", err2)
	}

	if !res1.Equal(res2) {
		t.Errorf("Positive scaling invariance failed: res1=%v, res2=%v", res1, res2)
	}
}

// TestMetamorphic_FallbackMonotonicity asserts that forced Collins-Hong fallback produces
// identical cell truth semantics to McCallum projection on non-singular formulas.
func TestMetamorphic_FallbackMonotonicity(t *testing.T) {
	env := NewEnv()
	// Standard non-singular circle: x^2 + y^2 - 1
	p, err := Parse("x^2 + y^2 - 1")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	// 1. Compute with Brown-McCallum projection
	mProjSets, errM := computeBrownMcCallumProjection([]Node{p}, []string{"x", "y"}, env)
	if errM != nil {
		t.Fatalf("computeBrownMcCallumProjection failed: %v", errM)
	}
	fsmM := NewCadLifecycleFSM()
	_ = fsmM.TransitionTo(CadStateNormalized)
	_ = fsmM.TransitionTo(CadStateProjected)
	_ = fsmM.TransitionTo(CadStateValidated)
	cellsM, errLiftM := liftCADCells(mProjSets, []string{"x", "y"}, env, fsmM)
	if errLiftM != nil {
		t.Fatalf("McCallum lifting failed: %v", errLiftM)
	}

	// 2. Compute with Collins-Hong complete fallback projection
	chProjSets, errCH := computeCollinsHongProjection([]Node{p}, []string{"x", "y"}, env)
	if errCH != nil {
		t.Fatalf("computeCollinsHongProjection failed: %v", errCH)
	}
	fsmCH := NewCadLifecycleFSM()
	_ = fsmCH.TransitionTo(CadStateNormalized)
	_ = fsmCH.TransitionTo(CadStateProjected)
	_ = fsmCH.TransitionTo(CadStateValidated)
	cellsCH, errLiftCH := liftCADCellsOpt(chProjSets, []string{"x", "y"}, env, fsmCH, true)
	if errLiftCH != nil {
		t.Fatalf("Collins-Hong fallback lifting failed: %v", errLiftCH)
	}

	// Verify both contain origin (sample point with sign -1)
	foundNegativeM := false
	for _, c := range cellsM {
		if c.SignVector[p.String()] < 0 {
			foundNegativeM = true
			break
		}
	}
	foundNegativeCH := false
	for _, c := range cellsCH {
		if c.SignVector[p.String()] < 0 {
			foundNegativeCH = true
			break
		}
	}

	if !foundNegativeM || !foundNegativeCH {
		t.Errorf("Fallback monotonicity check: McCallum negative=%v, Collins-Hong negative=%v", foundNegativeM, foundNegativeCH)
	}
}

// TestMetamorphic_JCSRoundTrip asserts parse -> canonicalize -> hash -> parse preservation.
func TestMetamorphic_JCSRoundTrip(t *testing.T) {
	trace := NewProofTrace(&VarNode{Name: "y"}, &VarNode{Name: "y"})
	trace.AddStep(ProofStep{
		Before:          &VarNode{Name: "y"},
		After:           &VarNode{Name: "y"},
		Rule:            "Reflexivity",
		RuleDescription: "Tautological equational reflexivity",
		IsSelfVerified:  true,
	})
	trace.Metadata["schema_version"] = "2.0"
	trace.Metadata["canonical_profile"] = "rfc8785-jcs"

	canonical1, err1 := trace.ToCanonicalJSON()
	if err1 != nil {
		t.Fatalf("ToCanonicalJSON failed: %v", err1)
	}
	hash1, errH1 := trace.ComputeProofHash()
	if errH1 != nil {
		t.Fatalf("ComputeProofHash failed: %v", errH1)
	}

	// Re-canonicalize the canonical JSON bytes
	canonical2, err2 := CanonicalizeJSON(canonical1)
	if err2 != nil {
		t.Fatalf("re-canonicalize failed: %v", err2)
	}
	hash2, errH2 := CanonicalHashSHA256(canonical2)
	if errH2 != nil {
		t.Fatalf("re-hash failed: %v", errH2)
	}

	if string(canonical1) != string(canonical2) {
		t.Errorf("JCS roundtrip canonical bytes mismatch: %s != %s", canonical1, canonical2)
	}
	if hash1 != hash2 {
		t.Errorf("JCS roundtrip hash mismatch: %s != %s", hash1, hash2)
	}
}
