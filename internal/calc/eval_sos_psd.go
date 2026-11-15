package calc

import (
	"math/big"
)

// sosLDLResult captures the outcome of an exact rational LDL^T factorization.
// When IsPSD is false, Witness holds an exact rational vector w such that w^T A w < 0.
type sosLDLResult struct {
	IsPSD   bool
	L       [][]*big.Rat
	D       []*big.Rat
	Witness []*big.Rat
}

// exactRationalLDLTWithWitness computes the symmetric factorization A = L * D * L^T over Q
// with exact zero-pivot rank-deficiency handling. When A is not positive semidefinite,
// it back-substitutes through L^T to construct a verified witness vector w with w^T A w < 0.
func exactRationalLDLTWithWitness(A [][]*big.Rat) sosLDLResult {
	m := len(A)
	L := make([][]*big.Rat, m)
	D := make([]*big.Rat, m)
	work := make([][]*big.Rat, m)
	for i := 0; i < m; i++ {
		L[i] = make([]*big.Rat, m)
		work[i] = make([]*big.Rat, m)
		D[i] = big.NewRat(0, 1)
		for j := 0; j < m; j++ {
			if i == j {
				L[i][j] = big.NewRat(1, 1)
			} else {
				L[i][j] = big.NewRat(0, 1)
			}
			work[i][j] = new(big.Rat).Set(A[i][j])
		}
	}

	for k := 0; k < m; k++ {
		d := work[k][k]
		if d.Sign() < 0 {
			wTilde := make([]*big.Rat, m)
			for i := range wTilde {
				wTilde[i] = big.NewRat(0, 1)
			}
			wTilde[k] = big.NewRat(1, 1)
			w := solveUpperUnitBackSub(L, wTilde, k)
			return sosLDLResult{
				IsPSD:   false,
				Witness: simplifyWitnessForQuadForm(A, w),
			}
		}

		if d.Sign() == 0 {
			// A zero pivot is PSD only if the remaining row/column k in work is identically zero.
			for j := k + 1; j < m; j++ {
				if work[k][j].Sign() != 0 {
					// Construct wTilde = alpha*e_k + e_j such that wTilde^T work wTilde = -1 < 0:
					// 2 * alpha * work[k][j] + work[j][j] = -1  =>  alpha = (-1 - work[j][j]) / (2 * work[k][j])
					wTilde := make([]*big.Rat, m)
					for i := range wTilde {
						wTilde[i] = big.NewRat(0, 1)
					}
					num := new(big.Rat).Add(big.NewRat(-1, 1), new(big.Rat).Neg(work[j][j]))
					den := new(big.Rat).Mul(big.NewRat(2, 1), work[k][j])
					wTilde[k] = new(big.Rat).Quo(num, den)
					wTilde[j] = big.NewRat(1, 1)
					w := solveUpperUnitBackSub(L, wTilde, j)
					return sosLDLResult{
						IsPSD:   false,
						Witness: simplifyWitnessForQuadForm(A, w),
					}
				}
			}
			D[k] = big.NewRat(0, 1)
			continue
		}

		D[k] = new(big.Rat).Set(d)
		for i := k + 1; i < m; i++ {
			if work[i][k].Sign() == 0 {
				continue
			}
			L[i][k] = new(big.Rat).Quo(work[i][k], d)
		}

		for i := k + 1; i < m; i++ {
			lik := L[i][k]
			if lik.Sign() == 0 {
				continue
			}
			for j := i; j < m; j++ {
				wkj := work[k][j]
				if wkj.Sign() == 0 {
					continue
				}
				sub := new(big.Rat).Mul(lik, wkj)
				work[i][j] = new(big.Rat).Sub(work[i][j], sub)
				if i != j {
					work[j][i] = new(big.Rat).Set(work[i][j])
				}
			}
		}
	}

	return sosLDLResult{
		IsPSD: true,
		L:     L,
		D:     D,
	}
}

// solveUpperUnitBackSub solves L^T w = wTilde where wTilde[i] = 0 for all i > maxActiveIdx.
func solveUpperUnitBackSub(L [][]*big.Rat, wTilde []*big.Rat, maxActiveIdx int) []*big.Rat {
	m := len(wTilde)
	w := make([]*big.Rat, m)
	for i := range w {
		w[i] = big.NewRat(0, 1)
	}
	for r := maxActiveIdx; r >= 0; r-- {
		val := new(big.Rat).Set(wTilde[r])
		for i := r + 1; i <= maxActiveIdx; i++ {
			if L[i][r].Sign() == 0 || w[i].Sign() == 0 {
				continue
			}
			prod := new(big.Rat).Mul(L[i][r], w[i])
			val.Sub(val, prod)
		}
		w[r] = val
	}
	return w
}

// extractKernelBasisFromLDLT computes an exact rational basis for ker(A) when A = L*D*L^T >= 0
// by solving L^T w = e_k for every index k where D[k] == 0.
func extractKernelBasisFromLDLT(L [][]*big.Rat, D []*big.Rat) [][]*big.Rat {
	m := len(D)
	var basis [][]*big.Rat
	for k := 0; k < m; k++ {
		if D[k].Sign() != 0 {
			continue
		}
		ek := make([]*big.Rat, m)
		for i := range ek {
			ek[i] = big.NewRat(0, 1)
		}
		ek[k] = big.NewRat(1, 1)
		w := solveUpperUnitBackSub(L, ek, k)
		basis = append(basis, normalizeWitnessVector(w))
	}
	return basis
}

// normalizeWitnessVector scales vector w by 1 / max_i |w_i| so cutting-plane coefficients stay bounded.
func normalizeWitnessVector(w []*big.Rat) []*big.Rat {
	maxAbs := big.NewRat(0, 1)
	for _, x := range w {
		if x == nil {
			continue
		}
		ax := new(big.Rat).Abs(x)
		if ax.Cmp(maxAbs) > 0 {
			maxAbs = ax
		}
	}
	out := make([]*big.Rat, len(w))
	if maxAbs.Sign() == 0 {
		for i := range out {
			out[i] = big.NewRat(0, 1)
		}
		return out
	}
	for i, x := range w {
		if x == nil {
			out[i] = big.NewRat(0, 1)
		} else {
			out[i] = new(big.Rat).Quo(x, maxAbs)
		}
	}
	return out
}

// simplifyWitnessForQuadForm rounds normalized rational witness w to small coprime integers
// that still strictly satisfy w_int^T A w_int < 0, preventing big.Rat denominator growth in Simplex pivots.
func simplifyWitnessForQuadForm(A [][]*big.Rat, w []*big.Rat) []*big.Rat {
	norm := normalizeWitnessVector(w)
	scales := []int64{2, 4, 8, 16, 32, 64, 128, 256, 1024, 4096}
	for _, scale := range scales {
		scaleRat := big.NewRat(scale, 1)
		cand := make([]*big.Rat, len(norm))
		ints := make([]*big.Int, len(norm))
		gcdAll := big.NewInt(0)
		for i, x := range norm {
			if x.Sign() == 0 {
				ints[i] = big.NewInt(0)
				continue
			}
			scaled := new(big.Rat).Mul(x, scaleRat)
			q, r := new(big.Int), new(big.Int)
			q.QuoRem(scaled.Num(), scaled.Denom(), r)
			absR := new(big.Int).Abs(r)
			twoAbsR := new(big.Int).Lsh(absR, 1)
			if twoAbsR.Cmp(scaled.Denom()) >= 0 {
				if scaled.Sign() >= 0 {
					q.Add(q, big.NewInt(1))
				} else {
					q.Sub(q, big.NewInt(1))
				}
			}
			ints[i] = q
			if q.Sign() != 0 {
				absQ := new(big.Int).Abs(q)
				if gcdAll.Sign() == 0 {
					gcdAll.Set(absQ)
				} else {
					gcdAll.GCD(nil, nil, gcdAll, absQ)
				}
			}
		}
		if gcdAll.Sign() == 0 {
			continue
		}
		for i, q := range ints {
			reduced := new(big.Int).Quo(q, gcdAll)
			cand[i] = new(big.Rat).SetInt(reduced)
		}
		if evalSymQuadForm(A, cand).Sign() < 0 {
			return cand
		}
	}
	return norm
}
