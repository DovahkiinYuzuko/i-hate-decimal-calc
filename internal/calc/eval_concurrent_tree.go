package calc

import (
	"math/big"
	"sync"
)

// DefaultProductTreeCutoff is the threshold below which sequential multiplication is preferred.
const DefaultProductTreeCutoff int64 = 64

// ParallelProductTree computes the product of all integers in [low, high] inclusive:
// low * (low + 1) * ... * high.
// It uses binary splitting (divide-and-conquer) and parallel goroutines up to an optimal tree depth.
func ParallelProductTree(low, high int64, cutoff int64) *big.Int {
	if low > high {
		return big.NewInt(1)
	}
	if low == high {
		return big.NewInt(low)
	}
	if cutoff <= 0 {
		cutoff = DefaultProductTreeCutoff
	}

	maxDepth := GetMaxTreeDepth()
	return productTreeRec(low, high, cutoff, 0, maxDepth)
}

func productTreeRec(low, high int64, cutoff int64, depth int, maxDepth int) *big.Int {
	count := high - low + 1
	if count <= 0 {
		return big.NewInt(1)
	}
	if count == 1 {
		return big.NewInt(low)
	}
	if count == 2 {
		return new(big.Int).Mul(big.NewInt(low), big.NewInt(high))
	}
	if count <= cutoff {
		// Base case: iterative sequential multiplication
		res := big.NewInt(low)
		cur := new(big.Int)
		for k := low + 1; k <= high; k++ {
			cur.SetInt64(k)
			res.Mul(res, cur)
		}
		return res
	}

	mid := low + (count / 2) - 1

	if depth < maxDepth {
		// Parallel branch: spawn goroutine for left half, compute right half on current goroutine
		var left *big.Int
		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			defer wg.Done()
			left = productTreeRec(low, mid, cutoff, depth+1, maxDepth)
		}()

		right := productTreeRec(mid+1, high, cutoff, depth+1, maxDepth)
		wg.Wait()

		return new(big.Int).Mul(left, right)
	}

	// Sequential binary splitting (preserves balanced operand lengths for fast Karatsuba/Toom-Cook)
	left := productTreeRec(low, mid, cutoff, depth+1, maxDepth)
	right := productTreeRec(mid+1, high, cutoff, depth+1, maxDepth)
	return new(big.Int).Mul(left, right)
}
