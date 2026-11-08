package calc

import (
	"math/big"
)

// TermGenerator is an interface for streaming continued fraction terms.
type TermGenerator interface {
	NextTerm() (*big.Int, bool)
}

// GosperCFState represents the lifecycle state of Gosper CF arithmetic FSM.
type GosperCFState int

const (
	GosperCFStateRunning GosperCFState = iota
	GosperCFStateTerminated
)

// GosperBilinearEngine executes Gosper (1972) continued fraction arithmetic
// for bilinear transformation z = (A*x*y + B*x + C*y + D) / (E*x*y + F*x + G*y + H).
type GosperBilinearEngine struct {
	A, B, C, D *big.Int
	E, F, G, H *big.Int

	GenX TermGenerator
	GenY TermGenerator

	XExhausted bool
	YExhausted bool

	IngestToggle bool
	State        GosperCFState
}

// NewGosperAddEngine creates an engine for x + y = (0*xy + 1*x + 1*y + 0) / (0*xy + 0*x + 0*y + 1).
func NewGosperAddEngine(genX, genY TermGenerator) *GosperBilinearEngine {
	e := &GosperBilinearEngine{
		A:     big.NewInt(0),
		B:     big.NewInt(1),
		C:     big.NewInt(1),
		D:     big.NewInt(0),
		E:     big.NewInt(0),
		F:     big.NewInt(0),
		G:     big.NewInt(0),
		H:     big.NewInt(1),
		GenX:  genX,
		GenY:  genY,
		State: GosperCFStateRunning,
	}
	e.reduceGCD()
	return e
}

// NewGosperMulEngine creates an engine for x * y = (1*xy + 0*x + 0*y + 0) / (0*xy + 0*x + 0*y + 1).
func NewGosperMulEngine(genX, genY TermGenerator) *GosperBilinearEngine {
	e := &GosperBilinearEngine{
		A:     big.NewInt(1),
		B:     big.NewInt(0),
		C:     big.NewInt(0),
		D:     big.NewInt(0),
		E:     big.NewInt(0),
		F:     big.NewInt(0),
		G:     big.NewInt(0),
		H:     big.NewInt(1),
		GenX:  genX,
		GenY:  genY,
		State: GosperCFStateRunning,
	}
	e.reduceGCD()
	return e
}

func (e *GosperBilinearEngine) reduceGCD() {
	coeffs := []*big.Int{e.A, e.B, e.C, e.D, e.E, e.F, e.G, e.H}
	gcdAll := big.NewInt(0)
	for _, c := range coeffs {
		if c != nil && c.Sign() != 0 {
			gcdAll.GCD(nil, nil, gcdAll, new(big.Int).Abs(c))
		}
	}
	if gcdAll.Cmp(big.NewInt(1)) > 0 {
		e.A.Div(e.A, gcdAll)
		e.B.Div(e.B, gcdAll)
		e.C.Div(e.C, gcdAll)
		e.D.Div(e.D, gcdAll)
		e.E.Div(e.E, gcdAll)
		e.F.Div(e.F, gcdAll)
		e.G.Div(e.G, gcdAll)
		e.H.Div(e.H, gcdAll)
	}
}

// NextTerm advances Gosper FSM and returns the next term, or false if stream ends.
func (e *GosperBilinearEngine) NextTerm() (*big.Int, bool) {
	if e.State == GosperCFStateTerminated {
		return nil, false
	}

	maxSteps := 5000 // Loop limiter guard
	for step := 0; step < maxSteps; step++ {
		// Both inputs exhausted: z = D / H
		if e.XExhausted && e.YExhausted {
			if e.H.Sign() == 0 {
				e.State = GosperCFStateTerminated
				return nil, false
			}
			q := new(big.Int).Div(e.D, e.H)
			rem := new(big.Int).Sub(e.D, new(big.Int).Mul(q, e.H))
			if rem.Sign() == 0 {
				e.State = GosperCFStateTerminated
				return q, true
			}
			// Invert D and H for next term
			e.D.Set(e.H)
			e.H.Set(rem)
			return q, true
		}

		// Check emission condition on domain [1, inf) x [1, inf)
		// 4 corner denominators:
		// d1: (1, 1) -> E + F + G + H
		// d2: (inf, 1) -> E + F
		// d3: (1, inf) -> E + G
		// d4: (inf, inf) -> E
		d1 := new(big.Int).Add(e.E, e.F)
		d1.Add(d1, e.G)
		d1.Add(d1, e.H)

		d2 := new(big.Int).Add(e.E, e.F)
		d3 := new(big.Int).Add(e.E, e.G)
		d4 := e.E

		// Corner numerators:
		n1 := new(big.Int).Add(e.A, e.B)
		n1.Add(n1, e.C)
		n1.Add(n1, e.D)

		n2 := new(big.Int).Add(e.A, e.B)
		n3 := new(big.Int).Add(e.A, e.C)
		n4 := e.A

		canEmit := false
		var emitQ *big.Int

		// Pole avoidance check: all denominators must be strictly of the same sign (and non-zero)
		if !e.XExhausted && !e.YExhausted {
			s1 := d1.Sign()
			s2 := d2.Sign()
			s3 := d3.Sign()
			s4 := d4.Sign()

			if s1 != 0 && s1 == s2 && s1 == s3 && s1 == s4 {
				q1 := floorDiv(n1, d1)
				q2 := floorDiv(n2, d2)
				q3 := floorDiv(n3, d3)
				q4 := floorDiv(n4, d4)

				if q1.Cmp(q2) == 0 && q1.Cmp(q3) == 0 && q1.Cmp(q4) == 0 {
					canEmit = true
					emitQ = q1
				}
			}
		} else if e.XExhausted {
			// Degenerated to y-transformation: z = (C*y + D) / (G*y + H)
			dy1 := new(big.Int).Add(e.G, e.H)
			dy2 := e.G
			ny1 := new(big.Int).Add(e.C, e.D)
			ny2 := e.C

			s1 := dy1.Sign()
			s2 := dy2.Sign()
			if s1 != 0 && s1 == s2 {
				q1 := floorDiv(ny1, dy1)
				q2 := floorDiv(ny2, dy2)
				if q1.Cmp(q2) == 0 {
					canEmit = true
					emitQ = q1
				}
			}
		} else if e.YExhausted {
			// Degenerated to x-transformation: z = (B*x + D) / (F*x + H)
			dx1 := new(big.Int).Add(e.F, e.H)
			dx2 := e.F
			nx1 := new(big.Int).Add(e.B, e.D)
			nx2 := e.B

			s1 := dx1.Sign()
			s2 := dx2.Sign()
			if s1 != 0 && s1 == s2 {
				q1 := floorDiv(nx1, dx1)
				q2 := floorDiv(nx2, dx2)
				if q1.Cmp(q2) == 0 {
					canEmit = true
					emitQ = q1
				}
			}
		}

		// Integer Ambiguity Resolution (HAKMEM 101B / Gosper 1972):
		// When true value is an exact integer k, approximations oscillate around k (k - eps and k + eps).
		// Consequently, corners never agree on floor (some corners floor to k-1, others to k).
		// For an irrational number, |z - k| > 0, so the interval eventually shrinks away from k.
		// Therefore, if corners bracket k with min(q_i) == k-1 and max(q_i) == k across multiple
		// ingest steps, the true value is mathematically guaranteed to be the exact integer k.
		if !canEmit && !e.XExhausted && !e.YExhausted {
			s1 := d1.Sign()
			s2 := d2.Sign()
			s3 := d3.Sign()
			s4 := d4.Sign()
			if s1 != 0 && s1 == s2 && s1 == s3 && s1 == s4 {
				qs := []*big.Int{floorDiv(n1, d1), floorDiv(n2, d2), floorDiv(n3, d3), floorDiv(n4, d4)}
				minQ, maxQ := qs[0], qs[0]
				for _, q := range qs[1:] {
					if q.Cmp(minQ) < 0 {
						minQ = q
					}
					if q.Cmp(maxQ) > 0 {
						maxQ = q
					}
				}
				diff := new(big.Int).Sub(maxQ, minQ)
				if diff.Cmp(big.NewInt(1)) == 0 && step >= 8 {
					e.State = GosperCFStateTerminated
					return maxQ, true
				}
			}
		}

		if canEmit {
			// Emit emitQ: z <- 1 / (z - q)
			q := emitQ
			newA := new(big.Int).Set(e.E)
			newB := new(big.Int).Set(e.F)
			newC := new(big.Int).Set(e.G)
			newD := new(big.Int).Set(e.H)

			newE := new(big.Int).Sub(e.A, new(big.Int).Mul(q, e.E))
			newF := new(big.Int).Sub(e.B, new(big.Int).Mul(q, e.F))
			newG := new(big.Int).Sub(e.C, new(big.Int).Mul(q, e.G))
			newH := new(big.Int).Sub(e.D, new(big.Int).Mul(q, e.H))

			e.A = newA
			e.B = newB
			e.C = newC
			e.D = newD
			e.E = newE
			e.F = newF
			e.G = newG
			e.H = newH

			e.reduceGCD()
			return q, true
		}

		// Ingest from X or Y
		ingestX := false
		if !e.XExhausted && !e.YExhausted {
			ingestX = e.IngestToggle
			e.IngestToggle = !e.IngestToggle
		} else if !e.XExhausted {
			ingestX = true
		} else if !e.YExhausted {
			ingestX = false
		}

		if ingestX {
			p, ok := e.GenX.NextTerm()
			if !ok {
				e.XExhausted = true
				e.C = new(big.Int).Set(e.A)
				e.D = new(big.Int).Set(e.B)
				e.G = new(big.Int).Set(e.E)
				e.H = new(big.Int).Set(e.F)
				e.A = big.NewInt(0)
				e.B = big.NewInt(0)
				e.E = big.NewInt(0)
				e.F = big.NewInt(0)
			} else {
				newA := new(big.Int).Add(new(big.Int).Mul(e.A, p), e.C)
				newB := new(big.Int).Add(new(big.Int).Mul(e.B, p), e.D)
				newC := new(big.Int).Set(e.A)
				newD := new(big.Int).Set(e.B)

				newE := new(big.Int).Add(new(big.Int).Mul(e.E, p), e.G)
				newF := new(big.Int).Add(new(big.Int).Mul(e.F, p), e.H)
				newG := new(big.Int).Set(e.E)
				newH := new(big.Int).Set(e.F)

				e.A = newA
				e.B = newB
				e.C = newC
				e.D = newD
				e.E = newE
				e.F = newF
				e.G = newG
				e.H = newH
			}
		} else {
			q, ok := e.GenY.NextTerm()
			if !ok {
				e.YExhausted = true
				e.B = new(big.Int).Set(e.A)
				e.D = new(big.Int).Set(e.C)
				e.F = new(big.Int).Set(e.E)
				e.H = new(big.Int).Set(e.G)
				e.A = big.NewInt(0)
				e.C = big.NewInt(0)
				e.E = big.NewInt(0)
				e.G = big.NewInt(0)
			} else {
				newA := new(big.Int).Add(new(big.Int).Mul(e.A, q), e.B)
				newB := new(big.Int).Set(e.A)
				newC := new(big.Int).Add(new(big.Int).Mul(e.C, q), e.D)
				newD := new(big.Int).Set(e.C)

				newE := new(big.Int).Add(new(big.Int).Mul(e.E, q), e.F)
				newF := new(big.Int).Set(e.E)
				newG := new(big.Int).Add(new(big.Int).Mul(e.G, q), e.H)
				newH := new(big.Int).Set(e.G)

				e.A = newA
				e.B = newB
				e.C = newC
				e.D = newD
				e.E = newE
				e.F = newF
				e.G = newG
				e.H = newH
			}
		}

		e.reduceGCD()
	}

	e.State = GosperCFStateTerminated
	return nil, false
}

// floorDiv computes the true floor division floor(n / d).
// If d is negative, both signs are flipped so d > 0, ensuring
// standard floor division semantics with big.Int.Div.
func floorDiv(n, d *big.Int) *big.Int {
	if d.Sign() < 0 {
		n = new(big.Int).Neg(n)
		d = new(big.Int).Neg(d)
	}
	return new(big.Int).Div(n, d)
}
