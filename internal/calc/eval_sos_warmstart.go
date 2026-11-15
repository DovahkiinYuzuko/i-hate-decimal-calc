package calc

import (
	"math"
	"math/big"
)

type sosWarmStartOutput struct {
	LambdaCandidates [][]*big.Rat
	KernelHints      []sosBlockCut
}

// runSOSWarmStartDR runs deterministic Douglas-Rachford splitting between the affine space
// Q(lambda) and the positive semidefinite cone S_+ to produce rational lambda candidates and
// near-kernel eigenvector hints. All outputs are strictly candidate hints verified by exact arithmetic.
func runSOSWarmStartDR(space *sosAffineSpace) sosWarmStartOutput {
	numBlocks := len(space.BlockBase)
	blockSizes := make([]int, numBlocks)
	for b := 0; b < numBlocks; b++ {
		blockSizes[b] = len(space.BlockBase[b])
	}

	// Initialize Z = Q(EqualLambda) in float64.
	Z := evalBlocksFloat(space, space.EqualLambda)
	X := allocBlocksFloat(blockSizes)
	R := allocBlocksFloat(blockSizes)
	Y := allocBlocksFloat(blockSizes)

	// Target floats and weight sums per group
	groupTargets := make([]float64, len(space.Groups))
	groupWeightSums := make([]float64, len(space.Groups))
	for gi, grp := range space.Groups {
		tf, _ := grp.TargetCoeff.Float64()
		groupTargets[gi] = tf
		var ws float64
		for _, pr := range grp.Pairs {
			ws += float64(pr.Weight)
		}
		groupWeightSums[gi] = ws
	}

	projectAffine := func(in, out [][][]float64) {
		copyBlocksFloat(in, out)
		for gi, grp := range space.Groups {
			ws := groupWeightSums[gi]
			if ws == 0 {
				continue
			}
			var curSum float64
			for _, pr := range grp.Pairs {
				curSum += float64(pr.Weight) * in[pr.Block][pr.LocalI][pr.LocalJ]
			}
			mu := (groupTargets[gi] - curSum) / ws
			for _, pr := range grp.Pairs {
				val := in[pr.Block][pr.LocalI][pr.LocalJ] + mu
				out[pr.Block][pr.LocalI][pr.LocalJ] = val
				if pr.LocalI != pr.LocalJ {
					out[pr.Block][pr.LocalJ][pr.LocalI] = val
				}
			}
		}
	}

	const maxIters = 240
	const relax = 1.5
	var eigVals [][](float64)
	var eigVecs [][][]float64

	for iter := 0; iter < maxIters; iter++ {
		projectAffine(Z, X)

		// R = 2*X - Z
		for b, sz := range blockSizes {
			for i := 0; i < sz; i++ {
				for j := 0; j < sz; j++ {
					R[b][i][j] = 2.0*X[b][i][j] - Z[b][i][j]
				}
			}
		}

		// Y = Proj_{S_+}(R)
		maxNegViol := 0.0
		for b, sz := range blockSizes {
			vals, vecs := jacobiSymEigen(R[b])
			for k := 0; k < sz; k++ {
				if vals[k] < 0 {
					if -vals[k] > maxNegViol {
						maxNegViol = -vals[k]
					}
					vals[k] = 0
				}
			}
			reconstructSymFromEigen(vals, vecs, Y[b])
		}

		// Z = Z + relax * (Y - X)
		stepNorm := 0.0
		for b, sz := range blockSizes {
			for i := 0; i < sz; i++ {
				for j := 0; j < sz; j++ {
					delta := Y[b][i][j] - X[b][i][j]
					stepNorm += delta * delta
					Z[b][i][j] += relax * delta
				}
			}
		}

		if iter >= 20 && stepNorm < 1e-22 {
			break
		}
	}

	// Final affine projection ensures X lies strictly on Q(lambda).
	projectAffine(Z, X)

	// Extract eigenvalues/vectors of final X for kernel hint discovery.
	eigVals = make([][]float64, numBlocks)
	eigVecs = make([][][]float64, numBlocks)
	for b := 0; b < numBlocks; b++ {
		eigVals[b], eigVecs[b] = jacobiSymEigen(X[b])
	}

	// Build rational lambda candidates at multiple continued-fraction precision caps.
	d := space.Dim
	rawLam := make([]float64, d)
	for k, dir := range space.Directions {
		rawLam[k] = X[dir.FreeBlock][dir.FreeLocalI][dir.FreeLocalJ]
	}

	var candidates [][]*big.Rat
	for _, maxDen := range []int64{16, 64, 256, 2048} {
		cand := make([]*big.Rat, d)
		for k := 0; k < d; k++ {
			cand[k] = floatToRatCF(rawLam[k], maxDen)
		}
		candidates = append(candidates, cand)
	}

	// Extract near-zero eigenvector hints for exact facial reduction soundness tests.
	var hints []sosBlockCut
	for b := 0; b < numBlocks; b++ {
		sz := blockSizes[b]
		for k := 0; k < sz; k++ {
			if eigVals[b][k] < 1e-4 {
				// Normalize by max abs entry before CF rounding so ratios like [1, 1, 1, 3] snap cleanly.
				vec := eigVecs[b][k]
				maxAbs := 0.0
				for _, x := range vec {
					if math.Abs(x) > maxAbs {
						maxAbs = math.Abs(x)
					}
				}
				if maxAbs < 1e-12 {
					continue
				}
				ratVec := make([]*big.Rat, sz)
				for i, x := range vec {
					ratVec[i] = floatToRatCF(x/maxAbs, 24)
				}
				hints = append(hints, sosBlockCut{
					Block:  b,
					Vector: ratVec,
					Key:    cutKey(b, ratVec),
				})
			}
		}
	}

	return sosWarmStartOutput{
		LambdaCandidates: candidates,
		KernelHints:      hints,
	}
}

func allocBlocksFloat(sizes []int) [][][]float64 {
	out := make([][][]float64, len(sizes))
	for b, sz := range sizes {
		out[b] = make([][]float64, sz)
		for i := 0; i < sz; i++ {
			out[b][i] = make([]float64, sz)
		}
	}
	return out
}

func copyBlocksFloat(src, dst [][][]float64) {
	for b := range src {
		for i := range src[b] {
			copy(dst[b][i], src[b][i])
		}
	}
}

func evalBlocksFloat(space *sosAffineSpace, lam []*big.Rat) [][][]float64 {
	numBlocks := len(space.BlockBase)
	out := make([][][]float64, numBlocks)
	for b := 0; b < numBlocks; b++ {
		Qb := space.EvaluateBlock(b, lam)
		sz := len(Qb)
		mat := make([][]float64, sz)
		for i := 0; i < sz; i++ {
			mat[i] = make([]float64, sz)
			for j := 0; j < sz; j++ {
				f, _ := Qb[i][j].Float64()
				mat[i][j] = f
			}
		}
		out[b] = mat
	}
	return out
}

// jacobiSymEigen computes eigenvalues and orthonormal eigenvectors (as rows vecs[k][i])
// of a real symmetric matrix A via cyclic Jacobi rotations.
func jacobiSymEigen(A [][]float64) ([]float64, [][]float64) {
	m := len(A)
	vals := make([]float64, m)
	vecs := make([][]float64, m)
	work := make([][]float64, m)
	for i := 0; i < m; i++ {
		vecs[i] = make([]float64, m)
		vecs[i][i] = 1.0
		work[i] = make([]float64, m)
		copy(work[i], A[i])
	}
	if m <= 1 {
		if m == 1 {
			vals[0] = work[0][0]
		}
		return vals, vecs
	}

	const maxSweeps = 40
	for sweep := 0; sweep < maxSweeps; sweep++ {
		offNorm := 0.0
		for p := 0; p < m; p++ {
			for q := p + 1; q < m; q++ {
				offNorm += math.Abs(work[p][q])
			}
		}
		if offNorm < 1e-13 {
			break
		}
		for p := 0; p < m; p++ {
			for q := p + 1; q < m; q++ {
				apq := work[p][q]
				if math.Abs(apq) < 1e-15 {
					continue
				}
				tau := (work[q][q] - work[p][p]) / (2.0 * apq)
				var t float64
				if tau >= 0 {
					t = 1.0 / (tau + math.Sqrt(1.0+tau*tau))
				} else {
					t = -1.0 / (-tau + math.Sqrt(1.0+tau*tau))
				}
				c := 1.0 / math.Sqrt(1.0+t*t)
				s := t * c

				app := work[p][p]
				aqq := work[q][q]
				work[p][p] = app - t*apq
				work[q][q] = aqq + t*apq
				work[p][q] = 0
				work[q][p] = 0

				for r := 0; r < m; r++ {
					if r == p || r == q {
						continue
					}
					arp := work[r][p]
					arq := work[r][q]
					nrp := c*arp - s*arq
					nrq := s*arp + c*arq
					work[r][p] = nrp
					work[p][r] = nrp
					work[r][q] = nrq
					work[q][r] = nrq
				}

				for r := 0; r < m; r++ {
					vp := vecs[p][r]
					vq := vecs[q][r]
					vecs[p][r] = c*vp - s*vq
					vecs[q][r] = s*vp + c*vq
				}
			}
		}
	}

	for i := 0; i < m; i++ {
		vals[i] = work[i][i]
	}
	return vals, vecs
}

func reconstructSymFromEigen(vals []float64, vecs [][]float64, out [][]float64) {
	m := len(vals)
	for i := 0; i < m; i++ {
		for j := i; j < m; j++ {
			var sum float64
			for k := 0; k < m; k++ {
				if vals[k] != 0 {
					sum += vals[k] * vecs[k][i] * vecs[k][j]
				}
			}
			out[i][j] = sum
			out[j][i] = sum
		}
	}
}

// floatToRatCF computes the best rational approximation to x with denominator <= maxDenom
// via standard continued fraction expansion.
func floatToRatCF(x float64, maxDenom int64) *big.Rat {
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return big.NewRat(0, 1)
	}
	if math.Abs(x) < 1e-11 {
		return big.NewRat(0, 1)
	}
	neg := x < 0
	if neg {
		x = -x
	}

	h0, k0 := int64(0), int64(1)
	h1, k1 := int64(1), int64(0)
	rem := x
	for iter := 0; iter < 32; iter++ {
		aFloat := math.Floor(rem)
		if aFloat > 1e15 {
			break
		}
		a := int64(aFloat)
		h2 := a*h1 + h0
		k2 := a*k1 + k0
		if k2 > maxDenom {
			break
		}
		h0, k0 = h1, k1
		h1, k1 = h2, k2
		frac := rem - aFloat
		if frac < 1e-11 {
			break
		}
		rem = 1.0 / frac
	}
	if k1 == 0 {
		return big.NewRat(0, 1)
	}
	if neg {
		h1 = -h1
	}
	return big.NewRat(h1, k1)
}
