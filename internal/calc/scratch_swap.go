package calc

import (
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"runtime/debug"
	"sync"


	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// GetAvailableDiskSpace returns available disk space in bytes for the specified path.
// If path is empty, os.TempDir() is used.
func GetAvailableDiskSpace(path string) (uint64, error) {
	if path == "" {
		path = os.TempDir()
	}
	absPath, err := filepath.Abs(path)
	if err == nil {
		path = absPath
	}
	return getDiskFreeSpace(path)
}

// ScratchSwapManager manages scratch/swap files on disk for out-of-core arithmetic.
type ScratchSwapManager struct {
	mu           sync.Mutex
	dir          string
	trackedFiles map[string]struct{}
}

// NewScratchSwapManager creates a new ScratchSwapManager.
// If baseDir is provided and not empty, the temporary directory is created inside baseDir.
// Otherwise, it is created under os.TempDir().
func NewScratchSwapManager(baseDir ...string) (*ScratchSwapManager, error) {
	var parentDir string
	if len(baseDir) > 0 && baseDir[0] != "" {
		parentDir = baseDir[0]
	} else {
		parentDir = os.TempDir()
	}

	tempDir, err := os.MkdirTemp(parentDir, "ihd_swap_*")
	if err != nil {
		return nil, fmt.Errorf("failed to create swap directory: %w", err)
	}

	return &ScratchSwapManager{
		dir:          tempDir,
		trackedFiles: make(map[string]struct{}),
	}, nil
}

// Dir returns the temporary swap directory path.
func (m *ScratchSwapManager) Dir() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.dir
}

// WriteBigInt writes val to a temporary swap file in Raw Bytes format:
// 1 byte sign (-1: 0xFF, 0: 0x00, 1: 0x01) followed by val.Bytes().
// It closes the file, calls debug.FreeOSMemory(), and records the filePath.
func (m *ScratchSwapManager) WriteBigInt(val *big.Int) (string, error) {
	if val == nil {
		return "", errors.New("val cannot be nil")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.dir == "" {
		return "", errors.New("swap manager is not initialized or has been cleaned up")
	}

	f, err := os.CreateTemp(m.dir, "swap_*.bin")
	if err != nil {
		return "", fmt.Errorf("failed to create swap file: %w", err)
	}
	filePath := f.Name()

	var signByte byte
	switch val.Sign() {
	case -1:
		signByte = 0xFF
	case 0:
		signByte = 0x00
	case 1:
		signByte = 0x01
	}

	if _, err := f.Write([]byte{signByte}); err != nil {
		_ = f.Close()
		_ = os.Remove(filePath)
		return "", fmt.Errorf("failed to write sign byte: %w", err)
	}

	if val.Sign() != 0 {
		bytes := val.Bytes()
		if len(bytes) > 0 {
			if _, err := f.Write(bytes); err != nil {
				_ = f.Close()
				_ = os.Remove(filePath)
				return "", fmt.Errorf("failed to write big.Int bytes: %w", err)
			}
		}
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(filePath)
		return "", fmt.Errorf("failed to close swap file: %w", err)
	}

	debug.FreeOSMemory()

	m.trackedFiles[filePath] = struct{}{}
	return filePath, nil
}

// ReadBigInt reads a big.Int from the given swap file.
// The file is closed after reading.
func (m *ScratchSwapManager) ReadBigInt(filePath string) (*big.Int, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open swap file: %w", err)
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat swap file: %w", err)
	}

	if fi.Size() == 0 {
		return nil, errors.New("swap file is empty")
	}

	buf := make([]byte, fi.Size())
	_, err = io.ReadFull(f, buf)
	if err != nil {
		return nil, fmt.Errorf("failed to read swap file: %w", err)
	}

	signByte := buf[0]
	rawBytes := buf[1:]

	res := new(big.Int)
	switch signByte {
	case 0x00:
		res.SetInt64(0)
	case 0x01:
		res.SetBytes(rawBytes)
	case 0xFF:
		res.SetBytes(rawBytes)
		res.Neg(res)
	default:
		return nil, fmt.Errorf("invalid sign byte in swap file: 0x%02X", signByte)
	}

	return res, nil
}

// Remove deletes the swap file and removes it from tracking.
func (m *ScratchSwapManager) Remove(filePath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.trackedFiles, filePath)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Cleanup removes all tracked temporary files and the swap directory.
func (m *ScratchSwapManager) Cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for f := range m.trackedFiles {
		_ = os.Remove(f)
	}
	m.trackedFiles = make(map[string]struct{})

	if m.dir != "" {
		_ = os.RemoveAll(m.dir)
		m.dir = ""
	}
}

// EstimateDiskRequired estimates the disk bytes required to store base^exp.
// Formula: estimatedBytes ≈ exp * bitLen(base) / 8
// Safely handles overflow.
func (m *ScratchSwapManager) EstimateDiskRequired(base *big.Int, exp *big.Int) uint64 {
	if base == nil || exp == nil || exp.Sign() <= 0 || base.Sign() == 0 {
		return 0
	}

	bitLen := base.BitLen()
	if bitLen == 0 {
		return 0
	}

	bitLenBig := big.NewInt(int64(bitLen))
	totalBits := new(big.Int).Mul(exp, bitLenBig)
	totalBytes := new(big.Int).Add(totalBits, big.NewInt(7))
	totalBytes.Div(totalBytes, big.NewInt(8))

	if totalBytes.IsUint64() {
		return totalBytes.Uint64()
	}
	return math.MaxUint64
}

// CheckDiskSpaceForPow checks if the estimated disk requirement for base^exp
// exceeds 80% of available disk space in the swap directory.
func (m *ScratchSwapManager) CheckDiskSpaceForPow(base *big.Int, exp *big.Int) error {
	reqBytes := m.EstimateDiskRequired(base, exp)

	m.mu.Lock()
	targetDir := m.dir
	m.mu.Unlock()

	if targetDir == "" {
		targetDir = os.TempDir()
	}

	availBytes, err := GetAvailableDiskSpace(targetDir)
	if err != nil {
		return fmt.Errorf("failed to check available disk space: %w", err)
	}

	// 80% limit: reqBytes > availBytes * 0.8
	availBig := new(big.Int).SetUint64(availBytes)
	limitBig := new(big.Int).Mul(availBig, big.NewInt(80))
	limitBig.Div(limitBig, big.NewInt(100))

	reqBig := new(big.Int).SetUint64(reqBytes)
	if reqBig.Cmp(limitBig) > 0 {
		return fmt.Errorf("%s", i18n.T("err_disk_space_exceeded", fmt.Sprintf("%d bytes", reqBytes), fmt.Sprintf("%d bytes", availBytes)))
	}

	return nil
}

// PowBigIntWithSwap computes base^exp using repeated squaring and disk-swap if threshold exceeded.
// PowBigIntWithSwap computes base^exp using repeated squaring and disk-swap if threshold exceeded.
func PowBigIntWithSwap(base *big.Int, exp *big.Int) (*big.Int, error) {
	return PowBigIntWithSwapAndBudget(base, exp, nil)
}

// PowBigIntWithSwapAndBudget computes base^exp using repeated squaring and disk-swap while ticking execution budget.
func PowBigIntWithSwapAndBudget(base *big.Int, exp *big.Int, b *ExecutionBudget) (*big.Int, error) {
	if exp.Sign() < 0 {
		return nil, errors.New("negative exponent requires rational inversion")
	}
	if exp.Sign() == 0 {
		return big.NewInt(1), nil
	}
	if base.Sign() == 0 {
		return big.NewInt(0), nil
	}
	one := big.NewInt(1)
	if base.Cmp(one) == 0 {
		return big.NewInt(1), nil
	}
	minusOne := big.NewInt(-1)
	if base.Cmp(minusOne) == 0 {
		if exp.Bit(0) == 0 {
			return big.NewInt(1), nil
		}
		return big.NewInt(-1), nil
	}

	mgr, err := NewScratchSwapManager()
	if err != nil {
		return nil, err
	}
	defer mgr.Cleanup()

	if err := mgr.CheckDiskSpaceForPow(base, exp); err != nil {
		return nil, err
	}

	// 基数 2 の高速パス: 1 << exp
	two := big.NewInt(2)
	if base.Cmp(two) == 0 && exp.IsInt64() && exp.Int64() <= int64(math.MaxInt32) {
		return new(big.Int).Lsh(big.NewInt(1), uint(exp.Int64())), nil
	}

	// 閾値: 推定ビット長が 10MB (約 8388万ビット) 未満ならインメモリ Exp
	reqBytes := mgr.EstimateDiskRequired(base, exp)
	const inMemoryThreshold = 10 * 1024 * 1024 // 10 MB
	if reqBytes < inMemoryThreshold {
		if b != nil {
			if err := b.Tick(); err != nil {
				return nil, err
			}
		}
		return new(big.Int).Exp(base, exp, nil), nil
	}

	// アウト・オブ・コア Repeated Squaring
	e := new(big.Int).Set(exp)
	curr := new(big.Int).Set(base)
	res := big.NewInt(1)

	var currFile string
	var resFile string

	for e.Sign() > 0 {
		if b != nil {
			if err := b.Tick(); err != nil {
				return nil, err
			}
		}
		if e.Bit(0) == 1 {
			// res = res * curr
			if resFile != "" {
				loadedRes, err := mgr.ReadBigInt(resFile)
				if err != nil {
					return nil, err
				}
				_ = mgr.Remove(resFile)
				res = loadedRes
			}
			if currFile != "" {
				loadedCurr, err := mgr.ReadBigInt(currFile)
				if err != nil {
					return nil, err
				}
				curr = loadedCurr
			}
			res.Mul(res, curr)
			if uint64(len(res.Bytes())) > inMemoryThreshold {
				resFile, err = mgr.WriteBigInt(res)
				if err != nil {
					return nil, err
				}
				res = nil
			}
		}

		e.Rsh(e, 1)
		if e.Sign() > 0 {
			// curr = curr * curr
			if currFile != "" {
				loadedCurr, err := mgr.ReadBigInt(currFile)
				if err != nil {
					return nil, err
				}
				_ = mgr.Remove(currFile)
				curr = loadedCurr
			}
			curr.Mul(curr, curr)
			if uint64(len(curr.Bytes())) > inMemoryThreshold {
				currFile, err = mgr.WriteBigInt(curr)
				if err != nil {
					return nil, err
				}
				curr = nil
			}
		}
	}

	if resFile != "" {
		loadedRes, err := mgr.ReadBigInt(resFile)
		if err != nil {
			return nil, err
		}
		_ = mgr.Remove(resFile)
		res = loadedRes
	}
	if currFile != "" {
		_ = mgr.Remove(currFile)
	}

	return res, nil
}
