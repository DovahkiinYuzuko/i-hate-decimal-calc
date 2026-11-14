package calc

import (
	"math/big"
	"sort"
)

// ecmTier defines a single Stage 1 / Stage 2 smoothness tier and curve allowance.
type ecmTier struct {
	B1        uint64
	B2        uint64
	MaxCurves int
}

// ecmWorkConfig bounds deterministic trial division, Pollard-Brent rho, and Lenstra ECM effort.
type ecmWorkConfig struct {
	SmallTrialLimit uint64
	BrentMaxSteps   int
	Tiers           []ecmTier
}

func defaultECMConfig() ecmWorkConfig {
	return ecmWorkConfig{
		SmallTrialLimit: 100000,
		BrentMaxSteps:   50000,
		Tiers: []ecmTier{
			{B1: 500, B2: 25000, MaxCurves: 12},
			{B1: 2000, B2: 100000, MaxCurves: 20},
			{B1: 11000, B2: 1000000, MaxCurves: 24},
		},
	}
}

type montgomeryPoint struct {
	X *big.Int
	Z *big.Int
}

func newMontgomeryPoint() montgomeryPoint {
	return montgomeryPoint{
		X: new(big.Int),
		Z: new(big.Int),
	}
}

func (p *montgomeryPoint) Set(src montgomeryPoint) {
	p.X.Set(src.X)
	p.Z.Set(src.Z)
}

// ecmScratch holds preallocated temporaries reused across Montgomery operations on a curve.
type ecmScratch struct {
	s, t, u, v, w, tmp1, tmp2, tmp3 *big.Int
}

func newECMScratch() *ecmScratch {
	return &ecmScratch{
		s:    new(big.Int),
		t:    new(big.Int),
		u:    new(big.Int),
		v:    new(big.Int),
		w:    new(big.Int),
		tmp1: new(big.Int),
		tmp2: new(big.Int),
		tmp3: new(big.Int),
	}
}

// xDBL computes [2]P on Montgomery curve with parameter a24 = (A+2)/4 mod n.
// dst must not alias p.
func xDBL(dst *montgomeryPoint, p montgomeryPoint, a24, n *big.Int, sc *ecmScratch) {
	// s = (X + Z) mod n, u = s^2 mod n
	sc.s.Add(p.X, p.Z)
	if sc.s.Cmp(n) >= 0 {
		sc.s.Sub(sc.s, n)
	}
	sc.u.Mul(sc.s, sc.s)
	sc.u.Mod(sc.u, n)

	// t = (X - Z) mod n, v = t^2 mod n
	sc.t.Sub(p.X, p.Z)
	if sc.t.Sign() < 0 {
		sc.t.Add(sc.t, n)
	}
	sc.v.Mul(sc.t, sc.t)
	sc.v.Mod(sc.v, n)

	// w = (u - v) mod n
	sc.w.Sub(sc.u, sc.v)
	if sc.w.Sign() < 0 {
		sc.w.Add(sc.w, n)
	}

	// X2 = (u * v) mod n
	dst.X.Mul(sc.u, sc.v)
	dst.X.Mod(dst.X, n)

	// Z2 = w * (v + a24 * w) mod n
	sc.tmp1.Mul(a24, sc.w)
	sc.tmp1.Mod(sc.tmp1, n)
	sc.tmp1.Add(sc.v, sc.tmp1)
	if sc.tmp1.Cmp(n) >= 0 {
		sc.tmp1.Sub(sc.tmp1, n)
	}
	dst.Z.Mul(sc.w, sc.tmp1)
	dst.Z.Mod(dst.Z, n)
}

// xADD computes P + Q given difference diff = P - Q on a Montgomery curve mod n.
// dst must not alias p, q, or diff.
func xADD(dst *montgomeryPoint, p, q, diff montgomeryPoint, n *big.Int, sc *ecmScratch) {
	// s = Xp - Zp, t = Xq + Zq => u = s * t mod n
	sc.s.Sub(p.X, p.Z)
	if sc.s.Sign() < 0 {
		sc.s.Add(sc.s, n)
	}
	sc.t.Add(q.X, q.Z)
	if sc.t.Cmp(n) >= 0 {
		sc.t.Sub(sc.t, n)
	}
	sc.u.Mul(sc.s, sc.t)
	sc.u.Mod(sc.u, n)

	// s = Xp + Zp, t = Xq - Zq => v = s * t mod n
	sc.s.Add(p.X, p.Z)
	if sc.s.Cmp(n) >= 0 {
		sc.s.Sub(sc.s, n)
	}
	sc.t.Sub(q.X, q.Z)
	if sc.t.Sign() < 0 {
		sc.t.Add(sc.t, n)
	}
	sc.v.Mul(sc.s, sc.t)
	sc.v.Mod(sc.v, n)

	// tmp1 = (u + v)^2 mod n
	sc.tmp1.Add(sc.u, sc.v)
	if sc.tmp1.Cmp(n) >= 0 {
		sc.tmp1.Sub(sc.tmp1, n)
	}
	sc.tmp1.Mul(sc.tmp1, sc.tmp1)
	sc.tmp1.Mod(sc.tmp1, n)

	// tmp2 = (u - v)^2 mod n
	sc.tmp2.Sub(sc.u, sc.v)
	if sc.tmp2.Sign() < 0 {
		sc.tmp2.Add(sc.tmp2, n)
	}
	sc.tmp2.Mul(sc.tmp2, sc.tmp2)
	sc.tmp2.Mod(sc.tmp2, n)

	dst.X.Mul(diff.Z, sc.tmp1)
	dst.X.Mod(dst.X, n)

	dst.Z.Mul(diff.X, sc.tmp2)
	dst.Z.Mod(dst.Z, n)
}

// xMUL computes [k]P via the constant-structure Montgomery ladder.
func xMUL(p montgomeryPoint, k uint64, a24, n *big.Int, sc *ecmScratch, r0, r1, scratchPt *montgomeryPoint) montgomeryPoint {
	if k == 0 {
		r0.X.SetInt64(1)
		r0.Z.SetInt64(0)
		return *r0
	}
	if k == 1 {
		r0.Set(p)
		return *r0
	}

	r0.Set(p)
	xDBL(r1, *r0, a24, n, sc)

	bitLen := 0
	for t := k; t > 0; t >>= 1 {
		bitLen++
	}
	for bit := bitLen - 2; bit >= 0; bit-- {
		if ((k >> uint(bit)) & 1) == 0 {
			xADD(scratchPt, *r0, *r1, p, n, sc)
			r1.Set(*scratchPt)
			xDBL(scratchPt, *r0, a24, n, sc)
			r0.Set(*scratchPt)
		} else {
			xADD(scratchPt, *r0, *r1, p, n, sc)
			r0.Set(*scratchPt)
			xDBL(scratchPt, *r1, a24, n, sc)
			r1.Set(*scratchPt)
		}
	}
	return *r0
}

// simplePrimesUpTo generates all primes <= limit using a basic sieve of Eratosthenes.
func simplePrimesUpTo(limit uint64) []uint64 {
	if limit < 2 {
		return nil
	}
	isComp := make([]bool, limit+1)
	var primes []uint64
	for i := uint64(2); i <= limit; i++ {
		if !isComp[i] {
			primes = append(primes, i)
			if i <= limit/i {
				for j := i * i; j <= limit; j += i {
					isComp[j] = true
				}
			}
		}
	}
	return primes
}

// exactNthRoot checks whether n == r^k for an integer r using monotone integer Newton iteration.
func exactNthRoot(n *big.Int, k int) (*big.Int, bool) {
	if n == nil || n.Sign() <= 0 || k < 2 {
		return nil, false
	}
	kBig := big.NewInt(int64(k))
	km1Big := big.NewInt(int64(k - 1))

	// Strict overestimate x_0 = 2^ceil(bitLen(n)/k)
	shift := uint((n.BitLen() + k - 1) / k)
	x := new(big.Int).Lsh(big.NewInt(1), shift)
	two := big.NewInt(2)
	if x.Cmp(two) < 0 {
		x.Set(two)
	}

	xPow := new(big.Int)
	quot := new(big.Int)
	term := new(big.Int)
	next := new(big.Int)

	for {
		xPow.Exp(x, km1Big, nil)
		if xPow.Sign() == 0 {
			return nil, false
		}
		quot.Quo(n, xPow)
		term.Mul(km1Big, x)
		term.Add(term, quot)
		next.Quo(term, kBig)
		if next.Cmp(x) >= 0 {
			break
		}
		x.Set(next)
	}

	check := new(big.Int).Exp(x, kBig, nil)
	if check.Cmp(n) == 0 {
		return x, true
	}
	return nil, false
}

// detectPerfectPower tests if n >= 4 satisfies n == base^exp for some prime exp >= 2.
func detectPerfectPower(n *big.Int) (*big.Int, int) {
	four := big.NewInt(4)
	if n == nil || n.Cmp(four) < 0 {
		return nil, 1
	}
	maxExp := n.BitLen()
	for _, p := range simplePrimesUpTo(uint64(maxExp)) {
		if root, ok := exactNthRoot(n, int(p)); ok {
			return root, int(p)
		}
	}
	return nil, 1
}

// pollardBrentRho attempts deterministic factorization of odd composite n using Brent's cycle method.
func pollardBrentRho(n *big.Int, maxSteps int) *big.Int {
	if n == nil || n.Cmp(big.NewInt(3)) <= 0 {
		return nil
	}
	two := big.NewInt(2)
	if new(big.Int).Mod(n, two).Sign() == 0 {
		return big.NewInt(2)
	}

	constants := []int64{1, 2, 3, 5, 7}
	const m = 128

	y := new(big.Int)
	x := new(big.Int)
	ys := new(big.Int)
	q := new(big.Int)
	diff := new(big.Int)
	g := new(big.Int)
	cBig := new(big.Int)

	stepFn := func(v *big.Int) {
		v.Mul(v, v)
		v.Add(v, cBig)
		v.Mod(v, n)
	}

	for _, cVal := range constants {
		cBig.SetInt64(cVal)
		y.SetInt64(2)
		q.SetInt64(1)
		r := 1
		g.SetInt64(1)
		steps := 0

		for g.Cmp(big.NewInt(1)) == 0 && steps < maxSteps {
			x.Set(y)
			for i := 0; i < r; i++ {
				stepFn(y)
			}
			k := 0
			for k < r && g.Cmp(big.NewInt(1)) == 0 && steps < maxSteps {
				ys.Set(y)
				lim := m
				if r-k < lim {
					lim = r - k
				}
				for i := 0; i < lim; i++ {
					stepFn(y)
					diff.Sub(x, y)
					if diff.Sign() < 0 {
						diff.Add(diff, n)
					}
					q.Mul(q, diff)
					q.Mod(q, n)
					steps++
				}
				g.GCD(nil, nil, q, n)
				k += lim
			}
			r <<= 1
		}

		if g.Cmp(n) == 0 {
			// Backtrack single step from ys
			for steps < maxSteps*2 {
				stepFn(ys)
				diff.Sub(x, ys)
				if diff.Sign() < 0 {
					diff.Add(diff, n)
				}
				g.GCD(nil, nil, diff, n)
				steps++
				if g.Cmp(big.NewInt(1)) > 0 {
					break
				}
			}
		}

		if g.Cmp(big.NewInt(1)) > 0 && g.Cmp(n) < 0 {
			return new(big.Int).Set(g)
		}
	}
	return nil
}

// babyStepResidues returns r in [1, D/2] coprime to D=210.
var stage2Residues210 = func() []uint64 {
	const d = uint64(210)
	var res []uint64
	for r := uint64(1); r <= d/2; r++ {
		if r%2 != 0 && r%3 != 0 && r%5 != 0 && r%7 != 0 {
			res = append(res, r)
		}
	}
	return res
}()

// suyamaCurveSetup initializes Montgomery base point P0 and curve parameter a24 for sigma >= 6.
// Returns (factorFound, P0, a24, okToProceed).
func suyamaCurveSetup(n *big.Int, sigma uint64) (*big.Int, montgomeryPoint, *big.Int, bool) {
	sig := new(big.Int).SetUint64(sigma)

	// u = (sigma^2 - 5) mod n
	u := new(big.Int).Mul(sig, sig)
	u.Sub(u, big.NewInt(5))
	u.Mod(u, n)

	// v = (4 * sigma) mod n
	v := new(big.Int).Lsh(sig, 2)
	v.Mod(v, n)

	// u3 = u^3 mod n, v3 = v^3 mod n
	u3 := new(big.Int).Mul(u, u)
	u3.Mod(u3, n)
	u3.Mul(u3, u)
	u3.Mod(u3, n)

	v3 := new(big.Int).Mul(v, v)
	v3.Mod(v3, n)
	v3.Mul(v3, v)
	v3.Mod(v3, n)

	// Harvest GCD(16 * u^3 * v^4, n) before inversion
	v4 := new(big.Int).Mul(v3, v)
	v4.Mod(v4, n)
	harvest := new(big.Int).Mul(big.NewInt(16), u3)
	harvest.Mod(harvest, n)
	harvest.Mul(harvest, v4)
	harvest.Mod(harvest, n)

	g := new(big.Int).GCD(nil, nil, harvest, n)
	if g.Cmp(big.NewInt(1)) > 0 && g.Cmp(n) < 0 {
		return g, montgomeryPoint{}, nil, false
	}
	if g.Cmp(n) == 0 {
		return nil, montgomeryPoint{}, nil, false
	}

	// d = 16 * u^3 * v mod n, compute inverse
	denom := new(big.Int).Mul(big.NewInt(16), u3)
	denom.Mod(denom, n)
	denom.Mul(denom, v)
	denom.Mod(denom, n)

	dInv := new(big.Int).ModInverse(denom, n)
	if dInv == nil {
		gd := new(big.Int).GCD(nil, nil, denom, n)
		if gd.Cmp(big.NewInt(1)) > 0 && gd.Cmp(n) < 0 {
			return gd, montgomeryPoint{}, nil, false
		}
		return nil, montgomeryPoint{}, nil, false
	}

	// a24 = (v - u)^3 * (3u + v) * dInv mod n
	vmu := new(big.Int).Sub(v, u)
	if vmu.Sign() < 0 {
		vmu.Add(vmu, n)
	}
	vmu3 := new(big.Int).Mul(vmu, vmu)
	vmu3.Mod(vmu3, n)
	vmu3.Mul(vmu3, vmu)
	vmu3.Mod(vmu3, n)

	u3pv := new(big.Int).Mul(big.NewInt(3), u)
	u3pv.Add(u3pv, v)
	u3pv.Mod(u3pv, n)

	a24 := new(big.Int).Mul(vmu3, u3pv)
	a24.Mod(a24, n)
	a24.Mul(a24, dInv)
	a24.Mod(a24, n)

	// Reject singular Montgomery curves: A^2 - 4 == 0 (mod n), A = 4*a24 - 2
	bigA := new(big.Int).Lsh(a24, 2)
	bigA.Sub(bigA, big.NewInt(2))
	bigA.Mod(bigA, n)
	disc := new(big.Int).Mul(bigA, bigA)
	disc.Sub(disc, big.NewInt(4))
	disc.Mod(disc, n)

	gDisc := new(big.Int).GCD(nil, nil, disc, n)
	if gDisc.Cmp(big.NewInt(1)) > 0 && gDisc.Cmp(n) < 0 {
		return gDisc, montgomeryPoint{}, nil, false
	}
	if gDisc.Cmp(n) == 0 {
		return nil, montgomeryPoint{}, nil, false
	}

	return nil, montgomeryPoint{X: u3, Z: v3}, a24, true
}

// ecmStage2 performs baby-step giant-step Stage 2 continuation on Montgomery point p after Stage 1.
func ecmStage2(n, a24 *big.Int, p montgomeryPoint, b1, b2 uint64, sc *ecmScratch) *big.Int {
	if b2 <= b1 {
		return nil
	}

	const D = uint64(210)
	maxR := D / 2

	// 1. Precompute baby steps kP for 1 <= k <= maxR and keep residues coprime to 210
	babies := make([]montgomeryPoint, len(stage2Residues210))
	pPrev := newMontgomeryPoint()
	pCurr := newMontgomeryPoint()
	pNext := newMontgomeryPoint()
	p1 := newMontgomeryPoint()
	p1.Set(p)

	pCurr.Set(p1) // 1P
	babies[0] = newMontgomeryPoint()
	babies[0].Set(pCurr) // r = 1 is always stage2Residues210[0]
	babyIdx := 1

	g := new(big.Int).GCD(nil, nil, pCurr.Z, n)
	if g.Cmp(big.NewInt(1)) > 0 && g.Cmp(n) < 0 {
		return g
	}
	if g.Cmp(n) == 0 {
		return nil
	}

	pPrev.Set(pCurr)
	xDBL(&pCurr, p1, a24, n, sc) // 2P

	for k := uint64(3); k <= maxR; k++ {
		xADD(&pNext, pCurr, p1, pPrev, n, sc)
		pPrev.Set(pCurr)
		pCurr.Set(pNext)

		if babyIdx < len(stage2Residues210) && stage2Residues210[babyIdx] == k {
			g.GCD(nil, nil, pCurr.Z, n)
			if g.Cmp(big.NewInt(1)) > 0 && g.Cmp(n) < 0 {
				return g
			}
			if g.Cmp(n) == 0 {
				return nil
			}
			babies[babyIdx] = newMontgomeryPoint()
			babies[babyIdx].Set(pCurr)
			babyIdx++
		}
	}

	// 2. Giant steps m * (D * P)
	r0 := newMontgomeryPoint()
	r1 := newMontgomeryPoint()
	scratchPt := newMontgomeryPoint()
	dp := newMontgomeryPoint()
	dp.Set(xMUL(p1, D, a24, n, sc, &r0, &r1, &scratchPt))

	g.GCD(nil, nil, dp.Z, n)
	if g.Cmp(big.NewInt(1)) > 0 && g.Cmp(n) < 0 {
		return g
	}
	if g.Cmp(n) == 0 {
		return nil
	}

	mMin := b1 / D
	if mMin < 1 {
		mMin = 1
	}
	mMax := (b2 + maxR) / D

	qPrev := newMontgomeryPoint()
	qCurr := newMontgomeryPoint()
	qNext := newMontgomeryPoint()
	qPrevCkpt := newMontgomeryPoint()
	qCurrCkpt := newMontgomeryPoint()

	if mMin == 1 {
		qCurr.Set(dp)
	} else {
		qPrev.Set(xMUL(dp, mMin-1, a24, n, sc, &r0, &r1, &scratchPt))
		qCurr.Set(xMUL(dp, mMin, a24, n, sc, &r0, &r1, &scratchPt))
	}

	prod := big.NewInt(1)
	diff := new(big.Int)
	t1 := new(big.Int)
	t2 := new(big.Int)
	one := big.NewInt(1)

	qPrevCkpt.Set(qPrev)
	qCurrCkpt.Set(qCurr)
	mCkpt := mMin
	stepsSinceCkpt := 0

	advanceGiant := func(m uint64, qp, qc, qn *montgomeryPoint) {
		if m == 1 {
			xDBL(qn, *qc, a24, n, sc)
			qp.Set(*qc)
			qc.Set(*qn)
			return
		}
		xADD(qn, *qc, dp, *qp, n, sc)
		qp.Set(*qc)
		qc.Set(*qn)
	}

	rollbackCheck := func(startM, endM uint64) *big.Int {
		rbPrev := newMontgomeryPoint()
		rbCurr := newMontgomeryPoint()
		rbNext := newMontgomeryPoint()
		rbPrev.Set(qPrevCkpt)
		rbCurr.Set(qCurrCkpt)

		for mm := startM; mm <= endM; mm++ {
			for _, b := range babies {
				t1.Mul(rbCurr.X, b.Z)
				t1.Mod(t1, n)
				t2.Mul(b.X, rbCurr.Z)
				t2.Mod(t2, n)
				diff.Sub(t1, t2)
				if diff.Sign() < 0 {
					diff.Add(diff, n)
				}
				g.GCD(nil, nil, diff, n)
				if g.Cmp(one) > 0 && g.Cmp(n) < 0 {
					return new(big.Int).Set(g)
				}
			}
			advanceGiant(mm, &rbPrev, &rbCurr, &rbNext)
		}
		return nil
	}

	for m := mMin; m <= mMax; m++ {
		for _, b := range babies {
			t1.Mul(qCurr.X, b.Z)
			t1.Mod(t1, n)
			t2.Mul(b.X, qCurr.Z)
			t2.Mod(t2, n)
			diff.Sub(t1, t2)
			if diff.Sign() < 0 {
				diff.Add(diff, n)
			}
			prod.Mul(prod, diff)
			prod.Mod(prod, n)
		}

		stepsSinceCkpt++
		if stepsSinceCkpt >= 16 || m == mMax {
			g.GCD(nil, nil, prod, n)
			if g.Cmp(one) > 0 && g.Cmp(n) < 0 {
				return new(big.Int).Set(g)
			}
			if g.Cmp(n) == 0 {
				if factor := rollbackCheck(mCkpt, m); factor != nil {
					return factor
				}
				return nil
			}
			prod.Set(one)
			stepsSinceCkpt = 0
		}

		if m < mMax {
			advanceGiant(m, &qPrev, &qCurr, &qNext)
			if stepsSinceCkpt == 0 {
				qPrevCkpt.Set(qPrev)
				qCurrCkpt.Set(qCurr)
				mCkpt = m + 1
			}
		}
	}

	return nil
}

// lenstraECM attempts to find a non-trivial factor of composite odd integer n using deterministic Suyama curves.
func lenstraECM(n *big.Int, tiers []ecmTier) *big.Int {
	sigma := uint64(6)
	sc := newECMScratch()
	p := newMontgomeryPoint()
	pCkpt := newMontgomeryPoint()
	r0 := newMontgomeryPoint()
	r1 := newMontgomeryPoint()
	scratchPt := newMontgomeryPoint()
	g := new(big.Int)
	one := big.NewInt(1)

	for _, tier := range tiers {
		primes := simplePrimesUpTo(tier.B1)
		for curve := 0; curve < tier.MaxCurves; curve++ {
			curSigma := sigma
			sigma++

			factor, p0, a24, ok := suyamaCurveSetup(n, curSigma)
			if factor != nil {
				return factor
			}
			if !ok {
				continue
			}

			p.Set(p0)
			pCkpt.Set(p0)
			stage1Failed := false
			ckptStart := 0

			for idx, pr := range primes {
				pe := pr
				for pe <= tier.B1/pr {
					pe *= pr
				}
				p.Set(xMUL(p, pe, a24, n, sc, &r0, &r1, &scratchPt))

				if (idx+1)%32 == 0 || idx == len(primes)-1 {
					g.GCD(nil, nil, p.Z, n)
					if g.Cmp(one) > 0 && g.Cmp(n) < 0 {
						return new(big.Int).Set(g)
					}
					if g.Cmp(n) == 0 {
						// Backtrack prime-by-prime from checkpoint
						p.Set(pCkpt)
						for j := ckptStart; j <= idx; j++ {
							prj := primes[j]
							pej := prj
							for pej <= tier.B1/prj {
								pej *= prj
							}
							p.Set(xMUL(p, pej, a24, n, sc, &r0, &r1, &scratchPt))
							g.GCD(nil, nil, p.Z, n)
							if g.Cmp(one) > 0 && g.Cmp(n) < 0 {
								return new(big.Int).Set(g)
							}
							if g.Cmp(n) == 0 {
								stage1Failed = true
								break
							}
						}
						if stage1Failed {
							break
						}
					}
					pCkpt.Set(p)
					ckptStart = idx + 1
				}
			}

			if stage1Failed {
				continue
			}

			if f2 := ecmStage2(n, a24, p, tier.B1, tier.B2, sc); f2 != nil {
				return f2
			}
		}
	}
	return nil
}

// factorCompositeWithECM recursively splits positive integer val >= 2 using perfect power check,
// Pollard-Brent rho, and deterministic Lenstra ECM, returning sorted prime factors plus any
// unfactored composite factors (preserving multiplicity from perfect-power peeling) if budget is exceeded.
func factorCompositeWithECM(val *big.Int, cfg ecmWorkConfig) ([]primeFactor, []primeFactor) {
	if val == nil || val.Cmp(big.NewInt(1)) <= 0 {
		return nil, nil
	}

	primeMap := make(map[string]primeFactor)
	compMap := make(map[string]primeFactor)
	one := big.NewInt(1)

	addFactor := func(target map[string]primeFactor, base *big.Int, mult int) {
		k := base.String()
		if existing, ok := target[k]; ok {
			existing.exp += mult
			target[k] = existing
		} else {
			target[k] = primeFactor{prime: new(big.Int).Set(base), exp: mult}
		}
	}

	splitWithPower := func(m *big.Int) (*big.Int, int) {
		totalMult := 1
		curr := new(big.Int).Set(m)
		for {
			root, exp := detectPerfectPower(curr)
			if root == nil || exp <= 1 {
				break
			}
			totalMult *= exp
			curr = root
		}
		return curr, totalMult
	}

	type workItem struct {
		num  *big.Int
		mult int
	}
	stack := []workItem{{num: new(big.Int).Set(val), mult: 1}}

	for len(stack) > 0 {
		item := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		n := item.num
		if n.Cmp(one) <= 0 {
			continue
		}

		// Peel powers first
		base, pwr := splitWithPower(n)
		mult := item.mult * pwr

		if IsDeterministicPrime(base) {
			addFactor(primeMap, base, mult)
			continue
		}

		// Try fast deterministic Pollard-Brent rho
		var divisor *big.Int
		if cfg.BrentMaxSteps > 0 {
			divisor = pollardBrentRho(base, cfg.BrentMaxSteps)
		}
		// Fallback to deterministic Lenstra ECM (Stage 1 & Stage 2)
		if divisor == nil && len(cfg.Tiers) > 0 {
			divisor = lenstraECM(base, cfg.Tiers)
		}

		if divisor == nil || divisor.Cmp(one) <= 0 || divisor.Cmp(base) >= 0 {
			addFactor(compMap, base, mult)
			continue
		}

		quot := new(big.Int).Quo(base, divisor)
		stack = append(stack, workItem{num: divisor, mult: mult})
		stack = append(stack, workItem{num: quot, mult: mult})
	}

	toSortedSlice := func(m map[string]primeFactor) []primeFactor {
		out := make([]primeFactor, 0, len(m))
		for _, pf := range m {
			out = append(out, pf)
		}
		sort.Slice(out, func(i, j int) bool {
			return out[i].prime.Cmp(out[j].prime) < 0
		})
		return out
	}

	return toSortedSlice(primeMap), toSortedSlice(compMap)
}

// factorBigIntWithConfig runs bounded small trial division followed by ECM/Brent factorization
// and returns (sorted prime factors, sorted unfactored composite remainders).
func factorBigIntWithConfig(n *big.Int, cfg ecmWorkConfig) ([]primeFactor, []primeFactor) {
	if n == nil || n.Sign() <= 0 {
		return nil, nil
	}
	val := new(big.Int).Set(n)
	one := big.NewInt(1)
	if val.Cmp(one) <= 0 {
		return nil, nil
	}

	var factors []primeFactor
	two := big.NewInt(2)
	three := big.NewInt(3)
	rem := new(big.Int)

	// Factor 2
	count2 := 0
	for {
		rem.Mod(val, two)
		if rem.Sign() == 0 {
			count2++
			val.Div(val, two)
		} else {
			break
		}
	}
	if count2 > 0 {
		factors = append(factors, primeFactor{prime: big.NewInt(2), exp: count2})
	}

	// Factor 3
	count3 := 0
	for {
		rem.Mod(val, three)
		if rem.Sign() == 0 {
			count3++
			val.Div(val, three)
		} else {
			break
		}
	}
	if count3 > 0 {
		factors = append(factors, primeFactor{prime: big.NewInt(3), exp: count3})
	}

	if val.Cmp(one) > 0 && IsDeterministicPrime(val) {
		factors = append(factors, primeFactor{prime: new(big.Int).Set(val), exp: 1})
		return factors, nil
	}

	// Bounded 6k +/- 1 trial division wheel up to cfg.SmallTrialLimit
	trialLimit := new(big.Int).SetUint64(cfg.SmallTrialLimit)
	d := big.NewInt(5)
	step := big.NewInt(2)
	d2 := new(big.Int).Mul(d, d)

	for d2.Cmp(val) <= 0 && (cfg.SmallTrialLimit == 0 || d.Cmp(trialLimit) <= 0) {
		count := 0
		for {
			rem.Mod(val, d)
			if rem.Sign() == 0 {
				count++
				val.Div(val, d)
			} else {
				break
			}
		}
		if count > 0 {
			factors = append(factors, primeFactor{prime: new(big.Int).Set(d), exp: count})
			if val.Cmp(one) <= 0 {
				break
			}
			if IsDeterministicPrime(val) {
				break
			}
			d2.Mul(d, d)
		}

		d.Add(d, step)
		if step.Cmp(two) == 0 {
			step.SetInt64(4)
		} else {
			step.SetInt64(2)
		}
		d2.Mul(d, d)
	}

	if val.Cmp(one) <= 0 {
		return factors, nil
	}
	if IsDeterministicPrime(val) {
		factors = append(factors, primeFactor{prime: new(big.Int).Set(val), exp: 1})
		return factors, nil
	}

	ecmPrimes, unfactored := factorCompositeWithECM(val, cfg)
	factors = append(factors, ecmPrimes...)
	sort.Slice(factors, func(i, j int) bool {
		return factors[i].prime.Cmp(factors[j].prime) < 0
	})
	return factors, unfactored
}
