package calc

import (
	"fmt"
	"math/big"
	"sort"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// internalGraph represents a normalized undirected graph.
type internalGraph struct {
	N       int
	Edges   map[[2]int]int // (u, v) with u < v -> multiplicity
	HasLoop bool
}

// clone creates a deep copy of the internal graph.
func (g *internalGraph) clone() *internalGraph {
	cp := &internalGraph{
		N:       g.N,
		Edges:   make(map[[2]int]int, len(g.Edges)),
		HasLoop: g.HasLoop,
	}
	for k, v := range g.Edges {
		cp.Edges[k] = v
	}
	return cp
}

// parseUndirectedGraph converts a MatrixNode into an internalGraph.
func parseUndirectedGraph(mat *MatrixNode) (*internalGraph, error) {
	if mat.Rows != mat.Cols {
		return nil, fmt.Errorf("%s", i18n.T("graph.err_must_be_square", mat.Rows, mat.Cols))
	}
	n := mat.Rows
	if len(mat.Data) != n {
		return nil, fmt.Errorf("%s", i18n.T("graph.err_must_be_square", mat.Rows, mat.Cols))
	}
	for r := 0; r < n; r++ {
		if len(mat.Data[r]) != n {
			return nil, fmt.Errorf("%s", i18n.T("graph.err_must_be_square", mat.Rows, mat.Cols))
		}
	}

	g := &internalGraph{
		N:     n,
		Edges: make(map[[2]int]int),
	}

	for r := 0; r < n; r++ {
		for c := 0; c < n; c++ {
			valNode := mat.Data[r][c]
			rat, ok := valNode.(*RationalNode)
			if !ok || !rat.Val.IsInt() {
				return nil, fmt.Errorf("%s", i18n.T("graph.err_non_integer_entry", r, c))
			}
			num := rat.Val.Num()
			if num.Sign() < 0 {
				return nil, fmt.Errorf("%s", i18n.T("graph.err_negative_weight", r, c))
			}
			if !num.IsInt64() {
				return nil, fmt.Errorf("%s", i18n.T("graph.err_weight_too_large", r, c))
			}
			weight := int(num.Int64())

			if r == c {
				if weight > 0 {
					g.HasLoop = true
				}
				continue
			}

			// Symmetry check via exact big.Int comparison
			symNode := mat.Data[c][r]
			symRat, ok := symNode.(*RationalNode)
			if !ok || !symRat.Val.IsInt() || symRat.Val.Num().Cmp(num) != 0 {
				return nil, fmt.Errorf("%s", i18n.T("graph.err_asymmetric_matrix", r, c))
			}

			if r < c && weight > 0 {
				g.Edges[[2]int{r, c}] = weight
			}
		}
	}

	return g, nil
}

// EvalGraphLaplacian computes L = D - A for an undirected graph.
func EvalGraphLaplacian(args []Node) (Node, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("%s", i18n.T("graph.err_requires_1_matrix_arg", len(args)))
	}
	mat, ok := args[0].(*MatrixNode)
	if !ok {
		return nil, fmt.Errorf("%s", i18n.T("graph.err_expected_matrix"))
	}
	g, err := parseUndirectedGraph(mat)
	if err != nil {
		return nil, err
	}

	n := g.N
	// Compute degree ignoring self-loops
	deg := make([]int, n)
	for edge, mult := range g.Edges {
		deg[edge[0]] += mult
		deg[edge[1]] += mult
	}

	lData := make([][]Node, n)
	for i := 0; i < n; i++ {
		lData[i] = make([]Node, n)
		for j := 0; j < n; j++ {
			if i == j {
				lData[i][j] = mustRational(int64(deg[i]), 1)
			} else {
				u, v := i, j
				if u > v {
					u, v = v, u
				}
				mult := g.Edges[[2]int{u, v}]
				lData[i][j] = mustRational(int64(-mult), 1)
			}
		}
	}

	return NewMatrix(n, n, lData)
}

// EvalSpanningTrees calculates total spanning trees via Kirchhoff's Matrix Tree Theorem.
func EvalSpanningTrees(args []Node) (Node, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("%s", i18n.T("graph.err_requires_1_matrix_arg", len(args)))
	}
	mat, ok := args[0].(*MatrixNode)
	if !ok {
		return nil, fmt.Errorf("%s", i18n.T("graph.err_expected_matrix"))
	}
	g, err := parseUndirectedGraph(mat)
	if err != nil {
		return nil, err
	}

	if g.N == 0 {
		return mustRational(0, 1), nil
	}
	if g.N == 1 {
		return mustRational(1, 1), nil
	}

	lapNode, err := EvalGraphLaplacian(args)
	if err != nil {
		return nil, err
	}
	lapMat := lapNode.(*MatrixNode)

	// Remove row 0 and col 0 (cofactor C_00)
	sub := submatrix(lapMat, 0, 0)
	detVal, err := evalDet(sub)
	if err != nil {
		return nil, err
	}

	// Det can be slightly negative due to orientation; abs it
	if rat, ok := detVal.(*RationalNode); ok {
		absVal := new(big.Int).Abs(rat.Val.Num())
		return &RationalNode{Val: new(big.Rat).SetInt(absVal)}, nil
	}

	return detVal, nil
}

// EvalChromaticPolynomial computes P(G, k) via deletion-contraction with structural simplifications.
func EvalChromaticPolynomial(args []Node) (Node, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("%s", i18n.T("graph.err_chromatic_args", len(args)))
	}
	mat, ok := args[0].(*MatrixNode)
	if !ok {
		return nil, fmt.Errorf("%s", i18n.T("graph.err_expected_matrix"))
	}
	g, err := parseUndirectedGraph(mat)
	if err != nil {
		return nil, err
	}

	var evalK Node
	if len(args) == 2 {
		evalK = args[1]
	}

	// Immediate zero for self-loop
	if g.HasLoop {
		if evalK != nil {
			return mustRational(0, 1), nil
		}
		return mustRational(0, 1), nil
	}

	// Normalize multi-edges to simple graph for coloring
	simpleG := &internalGraph{
		N:     g.N,
		Edges: make(map[[2]int]int, len(g.Edges)),
	}
	for k := range g.Edges {
		simpleG.Edges[k] = 1
	}

	stepBudget := 5000
	polyNode, err := computeChromaticRec(simpleG, &stepBudget)
	if err != nil {
		return nil, err
	}

	if evalK != nil {
		subEnv := NewEnv()
		subEnv.Set("k", evalK)
		return EvalWithEnv(polyNode, subEnv)
	}

	return polyNode, nil
}

// computeChromaticRec recursively calculates the chromatic polynomial.
func computeChromaticRec(g *internalGraph, budget *int) (Node, error) {
	if *budget <= 0 {
		return nil, fmt.Errorf("%s", i18n.T("graph.err_chromatic_poly_node_budget_exceeded", 5000))
	}
	*budget--

	if g.HasLoop {
		return mustRational(0, 1), nil
	}

	if g.N == 0 {
		return mustRational(1, 1), nil
	}

	// No edges: return k^N
	if len(g.Edges) == 0 {
		if g.N == 1 {
			return &VarNode{Name: "k"}, nil
		}
		return &PowNode{Base: &VarNode{Name: "k"}, Exp: mustRational(int64(g.N), 1)}, nil
	}

	// Connected components factorization
	components := findComponents(g)
	if len(components) > 1 {
		var compPolys []Node
		for _, comp := range components {
			cpNode, err := computeChromaticRec(comp, budget)
			if err != nil {
				return nil, err
			}
			compPolys = append(compPolys, cpNode)
		}
		return simplifyMul(compPolys)
	}

	// Structural Reduction 1: Isolated vertices (deg 0)
	deg := make([]int, g.N)
	for edge := range g.Edges {
		deg[edge[0]]++
		deg[edge[1]]++
	}
	for v := 0; v < g.N; v++ {
		if deg[v] == 0 {
			subG := removeVertex(g, v)
			subPoly, err := computeChromaticRec(subG, budget)
			if err != nil {
				return nil, err
			}
			return simplifyMul([]Node{&VarNode{Name: "k"}, subPoly})
		}
	}

	// Structural Reduction 2: Leaves (pendant vertices, deg 1)
	for v := 0; v < g.N; v++ {
		if deg[v] == 1 {
			subG := removeVertex(g, v)
			subPoly, err := computeChromaticRec(subG, budget)
			if err != nil {
				return nil, err
			}
			kMinus1, _ := simplifyAdd([]Node{&VarNode{Name: "k"}, mustRational(-1, 1)})
			return simplifyMul([]Node{kMinus1, subPoly})
		}
	}

	// Structural Reduction 3: Simplicial vertices (neighborhood is a clique)
	for v := 0; v < g.N; v++ {
		d := deg[v]
		if d >= 2 {
			nbrs := getNeighbors(g, v)
			if isClique(g, nbrs) {
				subG := removeVertex(g, v)
				subPoly, err := computeChromaticRec(subG, budget)
				if err != nil {
					return nil, err
				}
				kMinusD, _ := simplifyAdd([]Node{&VarNode{Name: "k"}, mustRational(int64(-d), 1)})
				return simplifyMul([]Node{kMinusD, subPoly})
			}
		}
	}

	// General Deletion-Contraction branch: P(G) = P(G - e) - P(G / e)
	// Pick an edge with highest degree endpoint
	var bestEdge [2]int
	maxDegSum := -1
	for edge := range g.Edges {
		dSum := deg[edge[0]] + deg[edge[1]]
		if dSum > maxDegSum {
			maxDegSum = dSum
			bestEdge = edge
		}
	}

	// G - e (deletion)
	gDel := g.clone()
	delete(gDel.Edges, bestEdge)
	polyDel, err := computeChromaticRec(gDel, budget)
	if err != nil {
		return nil, err
	}

	// G / e (contraction)
	gCont := contractEdge(g, bestEdge[0], bestEdge[1])
	polyCont, err := computeChromaticRec(gCont, budget)
	if err != nil {
		return nil, err
	}

	negPolyCont, _ := simplifyUnaryOp("-", polyCont)
	return simplifyAdd([]Node{polyDel, negPolyCont})
}

// findComponents returns independent connected components as subgraphs.
func findComponents(g *internalGraph) []*internalGraph {
	visited := make([]bool, g.N)
	adjList := make([][]int, g.N)
	for edge := range g.Edges {
		adjList[edge[0]] = append(adjList[edge[0]], edge[1])
		adjList[edge[1]] = append(adjList[edge[1]], edge[0])
	}

	var components []*internalGraph
	for i := 0; i < g.N; i++ {
		if visited[i] {
			continue
		}
		var compVerts []int
		queue := []int{i}
		visited[i] = true
		for len(queue) > 0 {
			curr := queue[0]
			queue = queue[1:]
			compVerts = append(compVerts, curr)
			for _, nbr := range adjList[curr] {
				if !visited[nbr] {
					visited[nbr] = true
					queue = append(queue, nbr)
				}
			}
		}

		if len(compVerts) == g.N {
			return []*internalGraph{g}
		}

		// Build subgraph
		vertMap := make(map[int]int, len(compVerts))
		for newIdx, oldIdx := range compVerts {
			vertMap[oldIdx] = newIdx
		}
		subG := &internalGraph{
			N:     len(compVerts),
			Edges: make(map[[2]int]int),
		}
		for edge, mult := range g.Edges {
			uNew, okU := vertMap[edge[0]]
			vNew, okV := vertMap[edge[1]]
			if okU && okV {
				if uNew > vNew {
					uNew, vNew = vNew, uNew
				}
				subG.Edges[[2]int{uNew, vNew}] = mult
			}
		}
		components = append(components, subG)
	}
	return components
}

// removeVertex creates a subgraph with vertex v deleted and vertices renumbered.
func removeVertex(g *internalGraph, vDel int) *internalGraph {
	subG := &internalGraph{
		N:     g.N - 1,
		Edges: make(map[[2]int]int),
	}
	for edge, mult := range g.Edges {
		u, v := edge[0], edge[1]
		if u == vDel || v == vDel {
			continue
		}
		if u > vDel {
			u--
		}
		if v > vDel {
			v--
		}
		if u > v {
			u, v = v, u
		}
		subG.Edges[[2]int{u, v}] = mult
	}
	return subG
}

// getNeighbors returns all adjacent vertices to v.
func getNeighbors(g *internalGraph, v int) []int {
	var nbrs []int
	for edge := range g.Edges {
		if edge[0] == v {
			nbrs = append(nbrs, edge[1])
		} else if edge[1] == v {
			nbrs = append(nbrs, edge[0])
		}
	}
	sort.Ints(nbrs)
	return nbrs
}

// isClique checks if a set of vertices forms a complete graph.
func isClique(g *internalGraph, verts []int) bool {
	for i := 0; i < len(verts); i++ {
		for j := i + 1; j < len(verts); j++ {
			u, v := verts[i], verts[j]
			if u > v {
				u, v = v, u
			}
			if _, ok := g.Edges[[2]int{u, v}]; !ok {
				return false
			}
		}
	}
	return true
}

// contractEdge contracts edge (u, v) into a single vertex.
func contractEdge(g *internalGraph, uContract, vContract int) *internalGraph {
	if uContract > vContract {
		uContract, vContract = vContract, uContract
	}

	newN := g.N - 1
	subG := &internalGraph{
		N:     newN,
		Edges: make(map[[2]int]int),
	}

	for edge := range g.Edges {
		u, v := edge[0], edge[1]
		if u == uContract && v == vContract {
			continue // contracted edge disappears
		}

		// Map vertices
		mapVert := func(vert int) int {
			if vert == vContract {
				return uContract
			}
			if vert > vContract {
				return vert - 1
			}
			return vert
		}

		uNew := mapVert(u)
		vNew := mapVert(v)

		if uNew == vNew {
			subG.HasLoop = true
			continue
		}
		if uNew > vNew {
			uNew, vNew = vNew, uNew
		}
		subG.Edges[[2]int{uNew, vNew}] = 1 // collapse multi-edges
	}

	return subG
}
