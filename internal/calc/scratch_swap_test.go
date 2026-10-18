package calc

import (
	"crypto/rand"
	"math/big"
	"os"
	"testing"
)

func TestGetAvailableDiskSpace(t *testing.T) {
	space, err := GetAvailableDiskSpace("")
	if err != nil {
		t.Fatalf("GetAvailableDiskSpace(\"\") failed: %v", err)
	}
	if space == 0 {
		t.Errorf("expected available disk space to be > 0, got %d", space)
	}
	t.Logf("Available disk space in temp dir: %d bytes (%.2f GB)", space, float64(space)/(1024*1024*1024))
}

func TestScratchSwapManager_WriteReadBigInt(t *testing.T) {
	mgr, err := NewScratchSwapManager()
	if err != nil {
		t.Fatalf("failed to create ScratchSwapManager: %v", err)
	}
	defer mgr.Cleanup()

	// Generate a 1,000,000-bit giant integer
	giantInt, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 1000000))
	if err != nil {
		t.Fatalf("failed to generate giant int: %v", err)
	}

	testCases := []struct {
		name string
		val  *big.Int
	}{
		{name: "zero", val: big.NewInt(0)},
		{name: "positive small", val: big.NewInt(123456789)},
		{name: "negative small", val: big.NewInt(-987654321)},
		{name: "positive giant (1M bits)", val: giantInt},
		{name: "negative giant (1M bits)", val: new(big.Int).Neg(giantInt)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path, err := mgr.WriteBigInt(tc.val)
			if err != nil {
				t.Fatalf("WriteBigInt failed: %v", err)
			}

			// File should exist
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("swap file does not exist: %v", err)
			}

			readVal, err := mgr.ReadBigInt(path)
			if err != nil {
				t.Fatalf("ReadBigInt failed: %v", err)
			}

			if tc.val.Cmp(readVal) != 0 {
				t.Errorf("read value does not match written value for %s", tc.name)
			}

			// Test Remove
			if err := mgr.Remove(path); err != nil {
				t.Errorf("Remove failed: %v", err)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Errorf("file should have been deleted: %s", path)
			}
		})
	}
}

func TestScratchSwapManager_Cleanup(t *testing.T) {
	mgr, err := NewScratchSwapManager()
	if err != nil {
		t.Fatalf("failed to create ScratchSwapManager: %v", err)
	}

	paths := make([]string, 0, 5)
	for i := 0; i < 5; i++ {
		p, err := mgr.WriteBigInt(big.NewInt(int64(i * 1000)))
		if err != nil {
			t.Fatalf("WriteBigInt failed: %v", err)
		}
		paths = append(paths, p)
	}

	dir := mgr.Dir()
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("swap dir should exist before cleanup: %v", err)
	}

	mgr.Cleanup()

	// All files and directory should be removed
	for _, p := range paths {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("file %s was not cleaned up", p)
		}
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("dir %s was not cleaned up", dir)
	}
}

func TestScratchSwapManager_DiskSpaceCheck(t *testing.T) {
	mgr, err := NewScratchSwapManager()
	if err != nil {
		t.Fatalf("failed to create ScratchSwapManager: %v", err)
	}
	defer mgr.Cleanup()

	// 1. Estimation logic check
	// 2^10 = 1024 -> base 2 (bitLen 2), exp 10 -> 2 * 10 / 8 = ~3 bytes
	base := big.NewInt(2)
	exp := big.NewInt(10)
	est := mgr.EstimateDiskRequired(base, exp)
	if est < 2 || est > 4 {
		t.Errorf("unexpected disk estimate for 2^10: %d", est)
	}

	// 0 base or exp
	if mgr.EstimateDiskRequired(big.NewInt(0), exp) != 0 {
		t.Errorf("expected 0 estimate for base 0")
	}
	if mgr.EstimateDiskRequired(base, big.NewInt(0)) != 0 {
		t.Errorf("expected 0 estimate for exp 0")
	}

	// Safe overflow check: huge exp
	hugeExp := new(big.Int).Lsh(big.NewInt(1), 100)
	estHuge := mgr.EstimateDiskRequired(big.NewInt(10), hugeExp)
	if estHuge == 0 {
		t.Errorf("expected huge estimate for huge exponent, got 0")
	}

	// 2. CheckDiskSpaceForPow
	// Moderate exponent should pass
	err = mgr.CheckDiskSpaceForPow(big.NewInt(2), big.NewInt(1000))
	if err != nil {
		t.Errorf("expected CheckDiskSpaceForPow to succeed for small power, got: %v", err)
	}

	// Astronomical exponent should fail due to exceeding 80% disk space
	astronomicalExp := new(big.Int).Lsh(big.NewInt(1), 60)
	err = mgr.CheckDiskSpaceForPow(big.NewInt(10), astronomicalExp)
	if err == nil {
		t.Errorf("expected CheckDiskSpaceForPow to fail for astronomical exponent")
	} else {
		t.Logf("correctly caught insufficient space: %v", err)
	}
}

func TestPowBigIntWithSwap(t *testing.T) {
	// 2^10 = 1024
	res, err := PowBigIntWithSwap(big.NewInt(2), big.NewInt(10))
	if err != nil {
		t.Fatalf("PowBigIntWithSwap(2, 10) failed: %v", err)
	}
	if res.Cmp(big.NewInt(1024)) != 0 {
		t.Errorf("expected 1024, got %s", res.String())
	}

	// 3^5 = 243
	res3, err := PowBigIntWithSwap(big.NewInt(3), big.NewInt(5))
	if err != nil {
		t.Fatalf("PowBigIntWithSwap(3, 5) failed: %v", err)
	}
	if res3.Cmp(big.NewInt(243)) != 0 {
		t.Errorf("expected 243, got %s", res3.String())
	}

	// (-1)^1000 = 1
	huge1000 := big.NewInt(1000)
	resNeg1, err := PowBigIntWithSwap(big.NewInt(-1), huge1000)
	if err != nil {
		t.Fatalf("PowBigIntWithSwap(-1, 1000) failed: %v", err)
	}
	if resNeg1.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("expected 1, got %s", resNeg1.String())
	}

	// (-1)^1001 = -1
	resNeg2, err := PowBigIntWithSwap(big.NewInt(-1), big.NewInt(1001))
	if err != nil {
		t.Fatalf("PowBigIntWithSwap(-1, 1001) failed: %v", err)
	}
	if resNeg2.Cmp(big.NewInt(-1)) != 0 {
		t.Errorf("expected -1, got %s", resNeg2.String())
	}
}

func TestSimplifyPow_Issue135(t *testing.T) {
	// (1 + i)^8 = 16
	one := mustRational(1, 1)
	c := NewComplex(one, one)
	exp8 := mustRational(8, 1)
	res, err := simplifyPow(c, exp8)
	if err != nil {
		t.Fatalf("simplifyPow((1+i), 8) failed: %v", err)
	}
	ratRes, ok := res.(*RationalNode)
	if !ok {
		t.Fatalf("expected RationalNode, got %T (%s)", res, res.String())
	}
	if ratRes.Val.Cmp(big.NewRat(16, 1)) != 0 {
		t.Errorf("expected 16, got %s", ratRes.Val.String())
	}

	// (-x)^100 where 100 is even
	uop := &UnaryOpNode{Op: "-", Expr: &VarNode{Name: "x"}}
	expHuge := &RationalNode{Val: new(big.Rat).SetInt(new(big.Int).Lsh(big.NewInt(1), 70))} // 2^70 (exceeds int64!)
	resEven, err := simplifyPow(uop, expHuge)
	if err != nil {
		t.Fatalf("simplifyPow(-x, 2^70) failed: %v", err)
	}
	// Since 2^70 is even, result should be x^(2^70) without leading "-"
	if _, isUop := resEven.(*UnaryOpNode); isUop {
		t.Errorf("expected positive expression, got unary minus: %s", resEven.String())
	}
}
