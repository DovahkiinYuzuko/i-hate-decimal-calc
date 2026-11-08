package calc

import (
	"bufio"
	"encoding/json"
	"math/big"
	"os/exec"
	"testing"
)

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
	// Launch sympy_oracle.py
	cmd := exec.Command("python", "../../tools/sympy-oracle/sympy_oracle.py")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Skipf("cannot start python sympy_oracle: %v", err)
		return
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Skipf("cannot start python sympy_oracle: %v", err)
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
