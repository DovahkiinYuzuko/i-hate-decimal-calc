package calc

import (
	"testing"
)

func TestQuantum_GateAxioms(t *testing.T) {
	// 1. H^2 == I
	qcH2, err := NewQuantumCircuit(1)
	if err != nil {
		t.Fatalf("NewQuantumCircuit failed: %v", err)
	}
	qcH2.AddGate(QuantumGate{Type: GateH, Target: 0, Control: -1})
	qcH2.AddGate(QuantumGate{Type: GateH, Target: 0, Control: -1})

	qcId, _ := NewQuantumCircuit(1)

	res, err := QuantumEquiv(qcH2, qcId, true)
	if err != nil || !res.IsEquivalent || !res.StrictMatch {
		t.Fatalf("H^2 should strictly equal I, got res: %v, err: %v", res, err)
	}

	// 2. S^2 == Z
	qcS2, _ := NewQuantumCircuit(1)
	qcS2.AddGate(QuantumGate{Type: GateS, Target: 0, Control: -1})
	qcS2.AddGate(QuantumGate{Type: GateS, Target: 0, Control: -1})

	qcZ, _ := NewQuantumCircuit(1)
	qcZ.AddGate(QuantumGate{Type: GateZ, Target: 0, Control: -1})

	res, err = QuantumEquiv(qcS2, qcZ, true)
	if err != nil || !res.IsEquivalent || !res.StrictMatch {
		t.Fatalf("S^2 should strictly equal Z, got res: %v, err: %v", res, err)
	}

	// 3. T^2 == S
	qcT2, _ := NewQuantumCircuit(1)
	qcT2.AddGate(QuantumGate{Type: GateT, Target: 0, Control: -1})
	qcT2.AddGate(QuantumGate{Type: GateT, Target: 0, Control: -1})

	qcS, _ := NewQuantumCircuit(1)
	qcS.AddGate(QuantumGate{Type: GateS, Target: 0, Control: -1})

	res, err = QuantumEquiv(qcT2, qcS, true)
	if err != nil || !res.IsEquivalent || !res.StrictMatch {
		t.Fatalf("T^2 should strictly equal S, got res: %v, err: %v", res, err)
	}

	// 4. T^8 == I
	qcT8, _ := NewQuantumCircuit(1)
	for i := 0; i < 8; i++ {
		qcT8.AddGate(QuantumGate{Type: GateT, Target: 0, Control: -1})
	}
	res, err = QuantumEquiv(qcT8, qcId, true)
	if err != nil || !res.IsEquivalent || !res.StrictMatch {
		t.Fatalf("T^8 should strictly equal I, got res: %v, err: %v", res, err)
	}

	// 5. CNOT^2 == I (2 qubits)
	qcCNOT2, _ := NewQuantumCircuit(2)
	qcCNOT2.AddGate(QuantumGate{Type: GateCNOT, Target: 1, Control: 0})
	qcCNOT2.AddGate(QuantumGate{Type: GateCNOT, Target: 1, Control: 0})

	qcId2, _ := NewQuantumCircuit(2)
	res, err = QuantumEquiv(qcCNOT2, qcId2, true)
	if err != nil || !res.IsEquivalent || !res.StrictMatch {
		t.Fatalf("CNOT^2 should strictly equal I, got res: %v, err: %v", res, err)
	}
}

func TestQuantum_GlobalPhaseExhaustive(t *testing.T) {
	// X Y Z = i * I = omega^2 * I
	// (X Y Z)^2 = -I = omega^4 * I
	// (X Y Z)^3 = -i * I = omega^6 * I
	// (X Y Z)^4 = I = omega^0 * I
	qcXYZ, _ := NewQuantumCircuit(1)
	qcXYZ.AddGate(QuantumGate{Type: GateX, Target: 0, Control: -1})
	qcXYZ.AddGate(QuantumGate{Type: GateY, Target: 0, Control: -1})
	qcXYZ.AddGate(QuantumGate{Type: GateZ, Target: 0, Control: -1})

	qcId, _ := NewQuantumCircuit(1)

	// Step 1: XYZ vs I (circuit applies X then Y then Z, resulting in U = Z Y X = -i I = omega^6 I)
	resStrict, _ := QuantumEquiv(qcXYZ, qcId, true)
	if resStrict.IsEquivalent {
		t.Fatalf("XYZ has global phase omega^6, should fail strict equality")
	}

	resProj, err := QuantumEquiv(qcXYZ, qcId, false)
	if err != nil || !resProj.IsEquivalent || resProj.PhasePower != 6 {
		t.Fatalf("XYZ should have phase power 6, got: %v, err: %v", resProj, err)
	}

	// Step 2: Test powers m = 0, 1, 2, 3 corresponding to phase omega^(6*m)
	for m := 0; m < 4; m++ {
		expectedK := (6 * m) % 8
		qcM, _ := NewQuantumCircuit(1)
		for step := 0; step < m; step++ {
			for g := 0; g < len(qcXYZ.Gates); g++ {
				qcM.AddGate(qcXYZ.Gates[g])
			}
		}

		resM, err := QuantumEquiv(qcM, qcId, false)
		if err != nil || !resM.IsEquivalent || resM.PhasePower != expectedK {
			t.Fatalf("(XYZ)^%d failed projective check: expected phase power %d, got %v, err: %v", m, expectedK, resM, err)
		}

		resMStrict, _ := QuantumEquiv(qcM, qcId, true)
		if expectedK == 0 {
			if !resMStrict.IsEquivalent || !resMStrict.StrictMatch {
				t.Fatalf("(XYZ)^%d (k=0) should match strictly", m)
			}
		} else {
			if resMStrict.IsEquivalent {
				t.Fatalf("(XYZ)^%d (k=%d) should NOT match strictly", m, expectedK)
			}
		}
	}
}

func TestQuantum_NearEquivalentMismatch_Witness(t *testing.T) {
	// Asymmetric CNOT: CNOT(0, 1) vs CNOT(1, 0)
	qc1, _ := NewQuantumCircuit(2)
	qc1.AddGate(QuantumGate{Type: GateCNOT, Target: 1, Control: 0})

	qc2, _ := NewQuantumCircuit(2)
	qc2.AddGate(QuantumGate{Type: GateCNOT, Target: 0, Control: 1})

	res, err := QuantumEquiv(qc1, qc2, false)
	if err != nil {
		t.Fatalf("QuantumEquiv error: %v", err)
	}
	if res.IsEquivalent {
		t.Fatalf("CNOT(0,1) and CNOT(1,0) should NOT be equivalent")
	}
	if res.CounterexampleState < 0 {
		t.Fatalf("Expected valid counterexample state index, got %d", res.CounterexampleState)
	}

	// Circuit differing only by T vs S gate
	qcA, _ := NewQuantumCircuit(1)
	qcA.AddGate(QuantumGate{Type: GateH, Target: 0, Control: -1})
	qcA.AddGate(QuantumGate{Type: GateT, Target: 0, Control: -1})

	qcB, _ := NewQuantumCircuit(1)
	qcB.AddGate(QuantumGate{Type: GateH, Target: 0, Control: -1})
	qcB.AddGate(QuantumGate{Type: GateS, Target: 0, Control: -1})

	resAB, _ := QuantumEquiv(qcA, qcB, false)
	if resAB.IsEquivalent {
		t.Fatalf("H-T and H-S circuits should not be equivalent under projective check")
	}
	if resAB.CounterexampleState < 0 {
		t.Fatalf("Expected witness state for H-T vs H-S")
	}
}

func TestQuantum_BellStateSimulation(t *testing.T) {
	// Prepare Bell State |Phi+> = (|00> + |11>) / sqrt(2)
	qc, _ := NewQuantumCircuit(2)
	qc.AddGate(QuantumGate{Type: GateH, Target: 0, Control: -1})
	qc.AddGate(QuantumGate{Type: GateCNOT, Target: 1, Control: 0})

	state, err := qc.ApplyToStateVector(nil)
	if err != nil {
		t.Fatalf("ApplyToStateVector failed: %v", err)
	}

	// Dimension 4: state[0] = 1/sqrt(2), state[1] = 0, state[2] = 0, state[3] = 1/sqrt(2)
	invSqrt2 := Cyclotomic8InvSqrt2()
	zero := Cyclotomic8Zero()

	if !state[0].Equals(invSqrt2) {
		t.Fatalf("state[0] should be 1/sqrt(2), got: %v", state[0])
	}
	if !state[1].Equals(zero) {
		t.Fatalf("state[1] should be 0, got: %v", state[1])
	}
	if !state[2].Equals(zero) {
		t.Fatalf("state[2] should be 0, got: %v", state[2])
	}
	if !state[3].Equals(invSqrt2) {
		t.Fatalf("state[3] should be 1/sqrt(2), got: %v", state[3])
	}
}
