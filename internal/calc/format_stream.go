package calc

import (
	"bufio"
	"io"
	"math/big"
	"sync"
)

const (
	// leafCutoffDigits is the threshold below which numbers are converted directly
	// using temporary slice buffers without further recursive splitting.
	leafCutoffDigits = 8192

	// defaultBufferSize is the buffer size for streaming giant outputs (1MB).
	defaultBufferSize = 1024 * 1024
)

// zeroChunk is a pre-filled slice of '0' bytes for fast zero-padding.
var zeroChunk = make([]byte, 4096)

func init() {
	for i := range zeroChunk {
		zeroChunk[i] = '0'
	}
}

// powerTenCache caches precomputed powers of 10 for divide-and-conquer splits.
var (
	powerTenCacheMu sync.RWMutex
	powerTenCache   = make(map[int]*big.Int)
)

// getPowerOfTen returns 10^k from cache or computes and caches it.
func getPowerOfTen(k int) *big.Int {
	if k <= 0 {
		return big.NewInt(1)
	}

	powerTenCacheMu.RLock()
	cached, ok := powerTenCache[k]
	powerTenCacheMu.RUnlock()
	if ok {
		return cached
	}

	powerTenCacheMu.Lock()
	defer powerTenCacheMu.Unlock()
	if cached, ok = powerTenCache[k]; ok {
		return cached
	}

	val := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(k)), nil)
	powerTenCache[k] = val
	return val
}

// EstimateDecimalDigits estimates the upper bound of decimal digits of val.
// 2^bitLen has floor(bitLen * log10(2)) + 1 digits. log10(2) ~ 1233/4096.
func EstimateDecimalDigits(val *big.Int) int {
	if val == nil || val.Sign() == 0 {
		return 1
	}
	bitLen := val.BitLen()
	return (bitLen*1233)/4096 + 1
}

// ComputeExactDecimalDigits determines the exact number of decimal digits of val
// without allocating the full decimal string representation.
func ComputeExactDecimalDigits(val *big.Int) int {
	if val == nil || val.Sign() == 0 {
		return 1
	}

	absVal := val
	if val.Sign() < 0 {
		absVal = new(big.Int).Abs(val)
	}

	bitLen := absVal.BitLen()
	if bitLen <= 64 {
		return len(absVal.String())
	}

	// log10(2) bounds using high-precision rational with big.Int to avoid int64 overflow
	// log10(2) ~ 301029995663981195 / 10^18
	logNum := big.NewInt(301029995663981195)
	logDen := big.NewInt(1000000000000000000)

	bLenMinus1 := big.NewInt(int64(bitLen - 1))
	prodMin := new(big.Int).Mul(bLenMinus1, logNum)
	dMin := int(new(big.Int).Div(prodMin, logDen).Int64()) + 1

	bLen := big.NewInt(int64(bitLen))
	prodMax := new(big.Int).Mul(bLen, logNum)
	dMax := int(new(big.Int).Div(prodMax, logDen).Int64()) + 1

	if dMin == dMax {
		return dMin
	}

	// When dMin != dMax (difference is at most 1), test against 10^(dMax-1)
	threshold := getPowerOfTen(dMax - 1)
	if absVal.Cmp(threshold) >= 0 {
		return dMax
	}
	return dMin
}

// writeZeroes writes n '0' bytes to w in chunks.
func writeZeroes(w io.Writer, n int) error {
	for n > 0 {
		toWrite := len(zeroChunk)
		if toWrite > n {
			toWrite = n
		}
		if _, err := w.Write(zeroChunk[:toWrite]); err != nil {
			return err
		}
		n -= toWrite
	}
	return nil
}

// WriteIntDecimal streams the decimal string representation of val to w.
// It uses divide-and-conquer with StreamFSM to minimize memory allocation.
func WriteIntDecimal(w io.Writer, val *big.Int) error {
	if val == nil {
		_, err := w.Write([]byte("0"))
		return err
	}

	// Handle sign
	sign := val.Sign()
	if sign == 0 {
		_, err := w.Write([]byte("0"))
		return err
	}

	absVal := val
	if sign < 0 {
		if _, err := w.Write([]byte("-")); err != nil {
			return err
		}
		absVal = new(big.Int).Abs(val)
	}

	fsm := NewStreamFSM()
	return streamRecursive(w, absVal, 0, fsm, true)
}

// streamRecursive performs depth-first divide-and-conquer base conversion.
// If exactDigits > 0, the output is padded with leading zeros to width exactDigits.
func streamRecursive(w io.Writer, x *big.Int, exactDigits int, fsm *StreamFSM, isRoot bool) error {
	estDigits := EstimateDecimalDigits(x)

	// Leaf condition: small enough to convert directly
	if estDigits <= leafCutoffDigits {
		if isRoot {
			_ = fsm.Transition(StateDirect)
		}

		s := x.Text(10)
		if exactDigits > len(s) {
			if err := writeZeroes(w, exactDigits-len(s)); err != nil {
				fsm.Fail(err)
				return err
			}
		}
		if _, err := w.Write([]byte(s)); err != nil {
			fsm.Fail(err)
			return err
		}

		if isRoot {
			_ = fsm.Transition(StateDone)
		}
		return nil
	}

	// Split point: half of the estimated or exact digits
	k := estDigits / 2
	if exactDigits > 0 {
		k = exactDigits / 2
	}
	if k < 1 {
		k = 1
	}

	if isRoot {
		_ = fsm.Transition(StatePowerTableGen)
	}
	tenK := getPowerOfTen(k)

	if isRoot {
		_ = fsm.Transition(StateSplitting)
	}

	// X = Q * 10^K + R
	q := new(big.Int)
	r := new(big.Int)
	q.QuoRem(x, tenK, r)

	// Top half Q (depth-first)
	if isRoot {
		_ = fsm.Transition(StateStreamingUpper)
	}

	var upperPadding int
	if exactDigits > k {
		upperPadding = exactDigits - k
	}
	if err := streamRecursive(w, q, upperPadding, fsm, false); err != nil {
		fsm.Fail(err)
		return err
	}

	// Release reference to Q explicitly for GC
	if isRoot {
		_ = fsm.Transition(StateFreeUpper)
	}
	q = nil

	// Bottom half R (padded to exact k digits)
	if isRoot {
		_ = fsm.Transition(StateStreamingLower)
	}
	if err := streamRecursive(w, r, k, fsm, false); err != nil {
		fsm.Fail(err)
		return err
	}

	if isRoot {
		_ = fsm.Transition(StateDone)
	}
	return nil
}

// WriteNode formats node n and streams it directly to w.
// Giant integers are streamed with O(log N) memory allocation.
func WriteNode(w io.Writer, n Node, opts FormatOptions) error {
	if n == nil {
		return nil
	}

	switch v := n.(type) {
	case *RationalNode:
		if v.Val.IsInt() {
			return WriteIntDecimal(w, v.Val.Num())
		}
		// Giant fractions: stream num, "/", denom
		if err := WriteIntDecimal(w, v.Val.Num()); err != nil {
			return err
		}
		if _, err := w.Write([]byte("/")); err != nil {
			return err
		}
		return WriteIntDecimal(w, v.Val.Denom())

	default:
		// For standard expressions, format to string and write
		str := FormatWithOptions(n, opts)
		_, err := w.Write([]byte(str))
		return err
	}
}

// NewBufferedStreamWriter returns a bufio.Writer optimized for streaming giant integers.
func NewBufferedStreamWriter(w io.Writer) *bufio.Writer {
	return bufio.NewWriterSize(w, defaultBufferSize)
}
