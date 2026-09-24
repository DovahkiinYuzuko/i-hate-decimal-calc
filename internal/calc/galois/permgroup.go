package galois

import (
	"fmt"
	"sort"
	"strings"
)

// Permutation represents a bijection on {0, 1, ..., n-1}.
// p[i] is the image of i under the permutation.
type Permutation []int

// NewPermutation validates and creates a new Permutation from a mapping.
func NewPermutation(mapping []int) (Permutation, error) {
	n := len(mapping)
	seen := make([]bool, n)
	for i, val := range mapping {
		if val < 0 || val >= n {
			return nil, fmt.Errorf("permutation element out of range [0, %d): %d at index %d", n, val, i)
		}
		if seen[val] {
			return nil, fmt.Errorf("duplicate element in permutation: %d", val)
		}
		seen[val] = true
	}
	res := make(Permutation, n)
	copy(res, mapping)
	return res, nil
}

// Identity returns the identity permutation of degree n.
func Identity(n int) Permutation {
	p := make(Permutation, n)
	for i := 0; i < n; i++ {
		p[i] = i
	}
	return p
}

// Degree returns the number of elements permuted.
func (p Permutation) Degree() int {
	return len(p)
}

// Compose computes (p ∘ q)(i) = p[q[i]].
func (p Permutation) Compose(q Permutation) Permutation {
	n := len(p)
	res := make(Permutation, n)
	for i := 0; i < n; i++ {
		res[i] = p[q[i]]
	}
	return res
}

// Inverse returns the inverse permutation p^-1.
func (p Permutation) Inverse() Permutation {
	n := len(p)
	res := make(Permutation, n)
	for i := 0; i < n; i++ {
		res[p[i]] = i
	}
	return res
}

// Equals checks if two permutations are identical.
func (p Permutation) Equals(q Permutation) bool {
	if len(p) != len(q) {
		return false
	}
	for i := range p {
		if p[i] != q[i] {
			return false
		}
	}
	return true
}

// Hash returns a unique uint64 for small permutations (n <= 10).
func (p Permutation) Hash() uint64 {
	var h uint64
	for _, val := range p {
		h = h*16 + uint64(val)
	}
	return h
}

// CycleDecomposition decomposes the permutation into disjoint cycles.
func (p Permutation) CycleDecomposition() [][]int {
	n := len(p)
	visited := make([]bool, n)
	var cycles [][]int
	for i := 0; i < n; i++ {
		if !visited[i] {
			var cycle []int
			curr := i
			for !visited[curr] {
				visited[curr] = true
				cycle = append(cycle, curr)
				curr = p[curr]
			}
			cycles = append(cycles, cycle)
		}
	}
	return cycles
}

// CycleType returns the lengths of disjoint cycles in descending order (e.g. [2, 1, 1, 1] or [3, 2]).
func (p Permutation) CycleType() []int {
	cycles := p.CycleDecomposition()
	var lengths []int
	for _, c := range cycles {
		lengths = append(lengths, len(c))
	}
	sort.Sort(sort.Reverse(sort.IntSlice(lengths)))
	return lengths
}

// IsEven returns true if the permutation is even (sign +1).
func (p Permutation) IsEven() bool {
	cycles := p.CycleDecomposition()
	inversions := 0
	for _, c := range cycles {
		inversions += len(c) - 1
	}
	return inversions%2 == 0
}

// String outputs the permutation in standard cycle notation.
func (p Permutation) String() string {
	cycles := p.CycleDecomposition()
	var b strings.Builder
	hasCycles := false
	for _, c := range cycles {
		if len(c) > 1 {
			hasCycles = true
			b.WriteString("(")
			for i, v := range c {
				if i > 0 {
					b.WriteString(" ")
				}
				b.WriteString(fmt.Sprintf("%d", v))
			}
			b.WriteString(")")
		}
	}
	if !hasCycles {
		return "()"
	}
	return b.String()
}

// PermGroup represents a permutation group acting on {0, ..., Degree-1}.
type PermGroup struct {
	Degree     int
	Name       string
	Generators []Permutation
	Elements   []Permutation
	Order      int
}

// NewPermGroup generates a permutation group from generators by closing under composition.
func NewPermGroup(name string, degree int, generators []Permutation) (*PermGroup, error) {
	id := Identity(degree)
	elements := []Permutation{id}
	elementMap := map[uint64]bool{id.Hash(): true}

	queue := []Permutation{id}
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		for _, gen := range generators {
			next := curr.Compose(gen)
			h := next.Hash()
			if !elementMap[h] {
				elementMap[h] = true
				elements = append(elements, next)
				queue = append(queue, next)
			}
		}
	}

	g := &PermGroup{
		Degree:     degree,
		Name:       name,
		Generators: generators,
		Elements:   elements,
		Order:      len(elements),
	}
	return g, nil
}

// Contains checks if permutation p belongs to group g.
func (g *PermGroup) Contains(p Permutation) bool {
	if p.Degree() != g.Degree {
		return false
	}
	h := p.Hash()
	for _, elem := range g.Elements {
		if elem.Hash() == h && elem.Equals(p) {
			return true
		}
	}
	return false
}

// IsTransitive checks if the group action on {0, ..., Degree-1} is transitive.
func (g *PermGroup) IsTransitive() bool {
	if g.Degree <= 1 {
		return true
	}
	reached := make([]bool, g.Degree)
	for _, p := range g.Elements {
		reached[p[0]] = true
	}
	for _, r := range reached {
		if !r {
			return false
		}
	}
	return true
}

// CommutatorSubgroup computes [G, H] = <g h g^-1 h^-1 | g in G, h in H>.
func (g *PermGroup) CommutatorSubgroup(h *PermGroup) *PermGroup {
	degree := g.Degree
	var commGens []Permutation
	seen := make(map[uint64]bool)

	for _, gElem := range g.Elements {
		gInv := gElem.Inverse()
		for _, hElem := range h.Elements {
			hInv := hElem.Inverse()
			// c = g * h * g^-1 * h^-1
			comm := gElem.Compose(hElem).Compose(gInv).Compose(hInv)
			hVal := comm.Hash()
			if !seen[hVal] {
				seen[hVal] = true
				commGens = append(commGens, comm)
			}
		}
	}

	name := fmt.Sprintf("[%s, %s]", g.Name, h.Name)
	res, _ := NewPermGroup(name, degree, commGens)
	return res
}

// DerivedSeries computes the derived series G = G^(0) ⊵ G^(1) ⊵ G^(2) ⊵ ...
// where G^(k+1) = [G^(k), G^(k)].
func (g *PermGroup) DerivedSeries() []*PermGroup {
	series := []*PermGroup{g}
	curr := g
	for {
		next := curr.CommutatorSubgroup(curr)
		series = append(series, next)
		if next.Order == curr.Order {
			// Commutator subgroup stabilized
			break
		}
		curr = next
	}
	return series
}

// IsSolvable returns true if the derived series reaches the trivial group {id}.
func (g *PermGroup) IsSolvable() bool {
	series := g.DerivedSeries()
	last := series[len(series)-1]
	return last.Order == 1
}
