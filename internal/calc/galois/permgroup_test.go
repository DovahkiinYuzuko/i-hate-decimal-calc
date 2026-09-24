package galois

import (
	"reflect"
	"testing"
)

func TestPermutationBasics(t *testing.T) {
	p1, err := NewPermutation([]int{1, 2, 0})
	if err != nil {
		t.Fatalf("failed to create permutation: %v", err)
	}
	if p1.Degree() != 3 {
		t.Errorf("expected degree 3, got %d", p1.Degree())
	}
	if !p1.IsEven() {
		t.Errorf("expected (0 1 2) to be even")
	}

	p2, _ := NewPermutation([]int{1, 0, 2})
	if p2.IsEven() {
		t.Errorf("expected (0 1) to be odd")
	}

	// Inverse of (0 1 2) is (0 2 1)
	inv := p1.Inverse()
	expectedInv := Permutation{2, 0, 1}
	if !inv.Equals(expectedInv) {
		t.Errorf("expected inverse %v, got %v", expectedInv, inv)
	}

	// Composition p1 o inv = id
	id := p1.Compose(inv)
	if !id.Equals(Identity(3)) {
		t.Errorf("expected identity, got %v", id)
	}

	// Cycle type
	// (0 1)(2 3 4) in S5 has cycle type [3, 2]
	p5, _ := NewPermutation([]int{1, 0, 3, 4, 2})
	ct := p5.CycleType()
	expectedCT := []int{3, 2}
	if !reflect.DeepEqual(ct, expectedCT) {
		t.Errorf("expected cycle type %v, got %v", expectedCT, ct)
	}
}

func TestStandardGroupsOrdersAndSolvability(t *testing.T) {
	cases := []struct {
		group       *PermGroup
		expectedOrd int
		solvable    bool
	}{
		{StandardS3(), 6, true},
		{StandardA3(), 3, true},
		{StandardS4(), 24, true},
		{StandardA4(), 12, true},
		{StandardD4(), 8, true},
		{StandardV4(), 4, true},
		{StandardC4(), 4, true},
		{StandardS5(), 120, false},
		{StandardA5(), 60, false},
		{StandardF20(), 20, true},
		{StandardD5(), 10, true},
		{StandardC5(), 5, true},
	}

	for _, tc := range cases {
		t.Run(tc.group.Name, func(t *testing.T) {
			if tc.group.Order != tc.expectedOrd {
				t.Errorf("%s: expected order %d, got %d", tc.group.Name, tc.expectedOrd, tc.group.Order)
			}
			if !tc.group.IsTransitive() {
				t.Errorf("%s: expected group to be transitive", tc.group.Name)
			}
			solv := tc.group.IsSolvable()
			if solv != tc.solvable {
				t.Errorf("%s: expected IsSolvable=%v, got %v", tc.group.Name, tc.solvable, solv)
			}
		})
	}
}

func TestDerivedSeriesAbelRuffini(t *testing.T) {
	s5 := StandardS5()
	series := s5.DerivedSeries()
	// Derived series of S5: S5 -> A5 -> A5 (stops at A5 of order 60)
	if len(series) < 3 {
		t.Fatalf("expected at least 3 stages in S5 derived series, got %d", len(series))
	}
	if series[0].Order != 120 {
		t.Errorf("expected S5 order 120, got %d", series[0].Order)
	}
	if series[1].Order != 60 {
		t.Errorf("expected A5 order 60, got %d", series[1].Order)
	}
	if series[2].Order != 60 {
		t.Errorf("expected commutator of A5 to be A5 (order 60), got %d", series[2].Order)
	}
}
