package calc

import (
	"math/big"
	"testing"
)

// simpleSliceGenerator streams terms from a finite slice.
type simpleSliceGenerator struct {
	terms []*big.Int
	idx   int
}

func newSliceGen(vals ...int64) *simpleSliceGenerator {
	terms := make([]*big.Int, len(vals))
	for i, v := range vals {
		terms[i] = big.NewInt(v)
	}
	return &simpleSliceGenerator{terms: terms}
}

func (s *simpleSliceGenerator) NextTerm() (*big.Int, bool) {
	if s.idx >= len(s.terms) {
		return nil, false
	}
	t := s.terms[s.idx]
	s.idx++
	return t, true
}

// periodicGenerator streams terms of a periodic continued fraction.
// e.g. sqrt(2) = [1; 2, 2, 2, ...]
type periodicGenerator struct {
	first    *big.Int
	periodic []*big.Int
	idx      int
}

func newSqrt2Gen() *periodicGenerator {
	return &periodicGenerator{
		first:    big.NewInt(1),
		periodic: []*big.Int{big.NewInt(2)},
		idx:      0,
	}
}

func (p *periodicGenerator) NextTerm() (*big.Int, bool) {
	if p.idx == 0 {
		p.idx++
		return p.first, true
	}
	t := p.periodic[(p.idx-1)%len(p.periodic)]
	p.idx++
	return t, true
}

func TestGosper_RationalAdd(t *testing.T) {
	// 1/2 = [0; 2]
	// 1/3 = [0; 3]
	// 1/2 + 1/3 = 5/6 = [0; 1, 5]
	genX := newSliceGen(0, 2)
	genY := newSliceGen(0, 3)

	engine := NewGosperAddEngine(genX, genY)
	if engine == nil {
		t.Fatalf("failed to create GosperAddEngine")
	}

	var terms []int64
	for i := 0; i < 10; i++ {
		term, ok := engine.NextTerm()
		if !ok {
			break
		}
		terms = append(terms, term.Int64())
	}

	// Expected: [0, 1, 5]
	if len(terms) != 3 || terms[0] != 0 || terms[1] != 1 || terms[2] != 5 {
		t.Errorf("expected [0, 1, 5], got %v", terms)
	}
}

func TestGosper_Sqrt2AddSqrt2(t *testing.T) {
	// sqrt(2) + sqrt(2) = 2*sqrt(2) = sqrt(8) = [2; 1, 4, 1, 4, ...]
	genX := newSqrt2Gen()
	genY := newSqrt2Gen()

	engine := NewGosperAddEngine(genX, genY)
	var terms []int64
	for i := 0; i < 5; i++ {
		term, ok := engine.NextTerm()
		if !ok {
			break
		}
		terms = append(terms, term.Int64())
	}

	// Check first 5 terms: [2, 1, 4, 1, 4]
	expected := []int64{2, 1, 4, 1, 4}
	for i, exp := range expected {
		if i >= len(terms) || terms[i] != exp {
			t.Fatalf("term %d mismatch: got %v, expected %v", i, terms, expected)
		}
	}
}

func TestGosper_Sqrt2MulSqrt2(t *testing.T) {
	// sqrt(2) * sqrt(2) = 2 = [2]
	genX := newSqrt2Gen()
	genY := newSqrt2Gen()

	engine := NewGosperMulEngine(genX, genY)
	var terms []int64
	for i := 0; i < 5; i++ {
		term, ok := engine.NextTerm()
		if !ok {
			break
		}
		terms = append(terms, term.Int64())
	}

	if len(terms) != 1 || terms[0] != 2 {
		t.Errorf("expected [2], got %v", terms)
	}
}
