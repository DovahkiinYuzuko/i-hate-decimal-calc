package calc

import (
	"fmt"
	"math/big"
)

// GateType enumerates supported quantum gate primitives in Clifford+T.
type GateType int

const (
	GateH GateType = iota
	GateX
	GateY
	GateZ
	GateS
	GateT
	GateCNOT
)

func (gt GateType) String() string {
	switch gt {
	case GateH:
		return "H"
	case GateX:
		return "X"
	case GateY:
		return "Y"
	case GateZ:
		return "Z"
	case GateS:
		return "S"
	case GateT:
		return "T"
	case GateCNOT:
		return "CNOT"
	default:
		return "Unknown"
	}
}

// QuantumGate represents a gate operation.
type QuantumGate struct {
	Type    GateType
	Target  int
	Control int // -1 if not applicable
}

// QuantumCircuit represents an n-qubit quantum circuit.
type QuantumCircuit struct {
	NumQubits int
	Gates     []QuantumGate
}

// EquivResult holds the result of quantum circuit equivalence checking.
type EquivResult struct {
	IsEquivalent         bool
	PhasePower           int // 0..7 if equivalent, -1 if not
	CounterexampleState  int // Basis index where outputs differ, -1 if equivalent
	StrictMatch          bool
}

// NewQuantumCircuit creates an empty quantum circuit with the given number of qubits.
func NewQuantumCircuit(numQubits int) (*QuantumCircuit, error) {
	if numQubits < 1 || numQubits > 6 {
		return nil, fmt.Errorf("numQubits must be between 1 and 6 for exact CAS verification, got %d", numQubits)
	}
	return &QuantumCircuit{
		NumQubits: numQubits,
		Gates:     make([]QuantumGate, 0),
	}, nil
}

// AddGate adds a quantum gate after validating qubit indices.
func (qc *QuantumCircuit) AddGate(g QuantumGate) error {
	if g.Target < 0 || g.Target >= qc.NumQubits {
		return fmt.Errorf("target qubit %d out of bounds for %d-qubit circuit", g.Target, qc.NumQubits)
	}
	if g.Type == GateCNOT {
		if g.Control < 0 || g.Control >= qc.NumQubits {
			return fmt.Errorf("control qubit %d out of bounds for %d-qubit circuit", g.Control, qc.NumQubits)
		}
		if g.Target == g.Control {
			return fmt.Errorf("CNOT target and control qubits must be distinct, got %d", g.Target)
		}
	}
	qc.Gates = append(qc.Gates, g)
	return nil
}

// ApplyToStateVector applies the circuit's gates sequentially to a state vector of length 2^n.
func (qc *QuantumCircuit) ApplyToStateVector(initialState []Cyclotomic8) ([]Cyclotomic8, error) {
	dim := 1 << qc.NumQubits
	state := make([]Cyclotomic8, dim)

	if initialState == nil {
		// Default to |0...0>
		state[0] = Cyclotomic8One()
		for i := 1; i < dim; i++ {
			state[i] = Cyclotomic8Zero()
		}
	} else {
		if len(initialState) != dim {
			return nil, fmt.Errorf("initialState length %d does not match 2^%d = %d", len(initialState), qc.NumQubits, dim)
		}
		for i := 0; i < dim; i++ {
			state[i] = initialState[i]
		}
	}

	fsm := NewQuantumCircuitFSM(qc.NumQubits)
	if err := fsm.Transition(QuantumStateValidating); err != nil {
		return nil, err
	}
	if err := fsm.Transition(QuantumStateSimulating); err != nil {
		return nil, err
	}

	invSqrt2 := Cyclotomic8InvSqrt2()
	omega := Cyclotomic8Omega()
	iUnit := Cyclotomic8{A0: big.NewInt(0), A1: big.NewInt(0), A2: big.NewInt(1), A3: big.NewInt(0), K: 0}
	negIUnit := iUnit.Neg()

	for _, g := range qc.Gates {
		targetMask := 1 << g.Target

		if g.Type == GateCNOT {
			controlMask := 1 << g.Control
			for idx := 0; idx < dim; idx++ {
				// Only apply when control bit is 1 and target bit is 0
				if (idx&controlMask) != 0 && (idx&targetMask) == 0 {
					partner := idx | targetMask
					state[idx], state[partner] = state[partner], state[idx]
				}
			}
			continue
		}

		// Single qubit gates
		for idx := 0; idx < dim; idx++ {
			if (idx & targetMask) == 0 {
				partner := idx | targetMask
				v0 := state[idx]
				v1 := state[partner]

				switch g.Type {
				case GateH:
					// v0' = (v0 + v1) / sqrt(2)
					// v1' = (v0 - v1) / sqrt(2)
					state[idx] = (v0.Add(v1)).Mul(invSqrt2)
					state[partner] = (v0.Sub(v1)).Mul(invSqrt2)

				case GateX:
					state[idx] = v1
					state[partner] = v0

				case GateY:
					// [ 0  -i ] [ v0 ] = [ -i * v1 ]
					// [ i   0 ] [ v1 ]   [  i * v0 ]
					state[idx] = v1.Mul(negIUnit)
					state[partner] = v0.Mul(iUnit)

				case GateZ:
					state[partner] = v1.Neg()

				case GateS:
					state[partner] = v1.Mul(iUnit)

				case GateT:
					state[partner] = v1.Mul(omega)
				}
			}
		}
	}

	_ = fsm.Transition(QuantumStateComplete)
	return state, nil
}

// ToDenseUnitary computes the full 2^n x 2^n unitary matrix of the circuit.
func (qc *QuantumCircuit) ToDenseUnitary() ([][]Cyclotomic8, error) {
	dim := 1 << qc.NumQubits
	unitary := make([][]Cyclotomic8, dim)
	for r := 0; r < dim; r++ {
		unitary[r] = make([]Cyclotomic8, dim)
	}

	for col := 0; col < dim; col++ {
		basisIn := make([]Cyclotomic8, dim)
		for r := 0; r < dim; r++ {
			if r == col {
				basisIn[r] = Cyclotomic8One()
			} else {
				basisIn[r] = Cyclotomic8Zero()
			}
		}

		colOut, err := qc.ApplyToStateVector(basisIn)
		if err != nil {
			return nil, err
		}

		for r := 0; r < dim; r++ {
			unitary[r][col] = colOut[r]
		}
	}

	return unitary, nil
}

// QuantumEquiv tests whether two quantum circuits are equivalent.
// If strict is true, tests U1 == U2.
// If strict is false, tests U1 == omega^k * U2 for some k in {0, ..., 7}.
func QuantumEquiv(qc1, qc2 *QuantumCircuit, strict bool) (*EquivResult, error) {
	if qc1.NumQubits != qc2.NumQubits {
		return &EquivResult{
			IsEquivalent:        false,
			PhasePower:          -1,
			CounterexampleState: 0,
			StrictMatch:         false,
		}, nil
	}

	fsm := NewQuantumCircuitFSM(qc1.NumQubits)
	_ = fsm.Transition(QuantumStateValidating)
	_ = fsm.Transition(QuantumStateSimulating)

	u1, err := qc1.ToDenseUnitary()
	if err != nil {
		return nil, err
	}
	u2, err := qc2.ToDenseUnitary()
	if err != nil {
		return nil, err
	}

	_ = fsm.Transition(QuantumStateComparing)
	dim := 1 << qc1.NumQubits

	// Step 1: Compute candidate phase from (U1^dagger * U2)[0][0] = sum_r conj(U1[r][0]) * U2[r][0]
	// Or more directly: find the first non-zero entry of U1 or basis output.
	// In unitary matrices, column 0 is guaranteed to have non-zero norm 1.
	var phase Cyclotomic8
	phaseFound := false

	for r := 0; r < dim; r++ {
		if !u1[r][0].IsZero() && !u2[r][0].IsZero() {
			// Candidate: phase = U1[r][0] / U2[r][0]
			// Since entries are in unitary matrix, U1[r][0] * conj(U2[r][0]) gives candidate phase if columns match
			// Or check candidate k in 0..7 directly:
			for k := 0; k < 8; k++ {
				omegaK := getOmegaPower(k)
				// Test if u1[r][0] == omegaK * u2[r][0]
				if u1[r][0].Equals(omegaK.Mul(u2[r][0])) {
					phase = omegaK
					phaseFound = true
					break
				}
			}
			if phaseFound {
				break
			}
		}
	}

	if !phaseFound {
		// Could be that col 0 has orthogonal non-zero positions, check all k=0..7
		for k := 0; k < 8; k++ {
			omegaK := getOmegaPower(k)
			match := true
			for r := 0; r < dim; r++ {
				if !u1[r][0].Equals(omegaK.Mul(u2[r][0])) {
					match = false
					break
				}
			}
			if match {
				phase = omegaK
				phaseFound = true
				break
			}
		}
	}

	if !phaseFound {
		// Definitely not equivalent up to omega^k
		_ = fsm.Transition(QuantumStateComplete)
		return &EquivResult{
			IsEquivalent:        false,
			PhasePower:          -1,
			CounterexampleState: 0,
			StrictMatch:         false,
		}, nil
	}

	phasePower, isRoot := phase.IsRootOfUnity()
	if !isRoot {
		_ = fsm.Transition(QuantumStateComplete)
		return &EquivResult{
			IsEquivalent:        false,
			PhasePower:          -1,
			CounterexampleState: 0,
			StrictMatch:         false,
		}, nil
	}

	if strict && phasePower != 0 {
		_ = fsm.Transition(QuantumStateComplete)
		return &EquivResult{
			IsEquivalent:        false,
			PhasePower:          phasePower,
			CounterexampleState: 0,
			StrictMatch:         false,
		}, nil
	}

	// Step 2: Verify entrywise U1[r][c] == phase * U2[r][c] for all r, c
	for col := 0; col < dim; col++ {
		for r := 0; r < dim; r++ {
			expected := phase.Mul(u2[r][col])
			if !u1[r][col].Equals(expected) {
				_ = fsm.Transition(QuantumStateComplete)
				return &EquivResult{
					IsEquivalent:        false,
					PhasePower:          -1,
					CounterexampleState: col,
					StrictMatch:         false,
				}, nil
			}
		}
	}

	_ = fsm.Transition(QuantumStateComplete)
	return &EquivResult{
		IsEquivalent:        true,
		PhasePower:          phasePower,
		CounterexampleState: -1,
		StrictMatch:         (phasePower == 0),
	}, nil
}

func getOmegaPower(k int) Cyclotomic8 {
	k = ((k % 8) + 8) % 8
	curr := Cyclotomic8One()
	omega := Cyclotomic8Omega()
	for i := 0; i < k; i++ {
		curr = curr.Mul(omega)
	}
	return curr
}
