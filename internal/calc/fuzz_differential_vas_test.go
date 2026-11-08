package calc

import (
	"bufio"
	"encoding/json"
	"errors"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func resolvePythonPath() (string, error) {
	if custom := os.Getenv("IHD_SYMPY_PATH"); custom != "" {
		if _, err := os.Stat(custom); err == nil {
			return custom, nil
		}
	}
	for _, name := range []string{"python", "python3", "py"} {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", errors.New("no python executable found in PATH or IHD_SYMPY_PATH")
}

func resolveScriptPath() (string, error) {
	if envPath := os.Getenv("IHD_SYMPY_ORACLE_PATH"); envPath != "" {
		if abs, err := filepath.Abs(envPath); err == nil {
			if _, statErr := os.Stat(abs); statErr == nil {
				return abs, nil
			}
		}
	}
	candidates := []string{
		filepath.Join("tools", "sympy-oracle", "sympy_oracle.py"),
		filepath.Join("..", "..", "tools", "sympy-oracle", "sympy_oracle.py"),
		filepath.Join("test-scripts", "go", "fuzz", "sympy_oracle.py"),
		filepath.Join("..", "..", "test-scripts", "go", "fuzz", "sympy_oracle.py"),
	}
	for _, p := range candidates {
		abs, err := filepath.Abs(p)
		if err == nil {
			if _, statErr := os.Stat(abs); statErr == nil {
				return abs, nil
			}
		}
	}
	return "", errors.New("cannot locate tools/sympy-oracle/sympy_oracle.py")
}

type sympyRequest struct {
	ID     int      `json:"id"`
	Action string   `json:"action"`
	Poly   []string `json:"poly"`
}

type sympyResponse struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
	Count  int    `json:"count"`
	Error  string `json:"error,omitempty"`
}

func TestDifferential_VAS_SymPy(t *testing.T) {
	pyPath, err := resolvePythonPath()
	if err != nil {
		if os.Getenv("IHD_REQUIRE_SYMPY_ORACLE") != "" {
			t.Fatalf("CI CRITICAL: Python executable not found: %v", err)
		}
		t.Skipf("cannot find python executable: %v", err)
		return
	}

	out, err := exec.Command(pyPath, "-c", "import sympy; print('SYMPY_OK')").Output()
	if err != nil || !strings.Contains(string(out), "SYMPY_OK") {
		if os.Getenv("IHD_REQUIRE_SYMPY_ORACLE") != "" {
			t.Fatalf("CI CRITICAL: SymPy oracle is strictly required in CI environment but unavailable: %v", err)
		}
		t.Skipf("SymPy is not installed or importable: %v", err)
		return
	}

	scriptPath, err := resolveScriptPath()
	if err != nil {
		if os.Getenv("IHD_REQUIRE_SYMPY_ORACLE") != "" {
			t.Fatalf("CI CRITICAL: sympy_oracle.py not found: %v", err)
		}
		t.Skipf("cannot find sympy_oracle.py: %v", err)
		return
	}

	// Launch sympy_oracle.py
	cmd := exec.Command(pyPath, "-u", scriptPath)
	cmd.Env = append(os.Environ(),
		"PYTHONUNBUFFERED=1",
		"PYTHONIOENCODING=utf-8",
		"PYTHONUTF8=1",
		"PYTHONHASHSEED=0",
	)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Skipf("cannot start python sympy_oracle stdin: %v", err)
		return
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Skipf("cannot start python sympy_oracle stdout: %v", err)
		return
	}
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot run python: %v", err)
		return
	}
	defer func() {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
	}()

	scanner := bufio.NewScanner(stdout)

	testPolys := [][]*big.Int{
		{big.NewInt(-2), big.NewInt(0), big.NewInt(1)},                         // x^2 - 2
		{big.NewInt(-2), big.NewInt(0), big.NewInt(0), big.NewInt(1)},          // x^3 - 2
		{big.NewInt(1), big.NewInt(-3), big.NewInt(0), big.NewInt(1)},          // x^3 - 3x + 1
		{big.NewInt(-1), big.NewInt(-1), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(1)}, // x^5 - x - 1
		{big.NewInt(2), big.NewInt(0), big.NewInt(-4), big.NewInt(0), big.NewInt(1)}, // x^4 - 4x^2 + 2
	}

	for i, poly := range testPolys {
		// 1. Compute VAS
		vasRoots := IsolateRealRootsVAS(poly)

		// 2. Query SymPy
		polyStr := make([]string, len(poly))
		for j, c := range poly {
			polyStr[j] = c.String()
		}
		req := sympyRequest{
			ID:     i + 1,
			Action: "isolate_roots",
			Poly:   polyStr,
		}
		reqBytes, _ := json.Marshal(req)
		_, _ = stdin.Write(append(reqBytes, '\n'))

		if !scanner.Scan() {
			t.Fatalf("sympy_oracle crashed on poly %v", poly)
		}

		var res sympyResponse
		if err := json.Unmarshal(scanner.Bytes(), &res); err != nil {
			t.Fatalf("invalid json response from sympy: %v", err)
		}
		if res.Status != "ok" {
			t.Fatalf("sympy error on poly %v: %s", poly, res.Error)
		}

		if len(vasRoots) != res.Count {
			t.Errorf("poly %v: VAS count %d != SymPy count %d", poly, len(vasRoots), res.Count)
		}
	}
}
