package calc_test

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
)

type workerResponse struct {
	generation uint64
	id         uint64
	data       []byte
	err        error
}

// SymPyClient manages a resident Python sympy_oracle.py worker communicating via unbuffered NDJSON.
// Features generation-tracked request correlation, single resident reader goroutine, and hybrid caching.
type SymPyClient struct {
	mu           sync.Mutex
	cmd          *exec.Cmd
	stdin        io.WriteCloser
	generation   atomic.Uint64
	seq          atomic.Uint64
	available    bool
	unavailableR string
	pythonPath   string
	scriptPath   string
	version      string
	mpmathVer    string
	pid          int
	requestCount int

	pendingMu  sync.Mutex
	pending    map[uint64]chan workerResponse
	readerDone chan struct{}
}

var (
	globalSymPyClient *SymPyClient
	globalSymPyOnce   sync.Once
)

// GetGlobalSymPyClient returns the singleton SymPyClient instance for differential testing.
func GetGlobalSymPyClient() *SymPyClient {
	globalSymPyOnce.Do(func() {
		globalSymPyClient = NewSymPyClient()
	})
	return globalSymPyClient
}

// NewSymPyClient initializes and starts a new resident SymPy worker.
func NewSymPyClient() *SymPyClient {
	c := &SymPyClient{
		pending: make(map[uint64]chan workerResponse),
	}
	pyPath, err := resolvePythonPath()
	if err != nil {
		c.unavailableR = fmt.Sprintf("python not found: %v", err)
		return c
	}
	c.pythonPath = pyPath

	script, err := resolveScriptPath()
	if err != nil {
		c.unavailableR = fmt.Sprintf("sympy_oracle.py not found: %v", err)
		return c
	}
	c.scriptPath = script

	if err := c.startWorker(); err != nil {
		c.unavailableR = fmt.Sprintf("failed to start worker: %v", err)
		return c
	}

	c.available = true
	return c
}

// Available returns whether the SymPy oracle is ready for queries.
func (c *SymPyClient) Available() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.available
}

// ReasonUnavailable returns why SymPy cannot be used.
func (c *SymPyClient) ReasonUnavailable() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.unavailableR
}

// Version returns the detected SymPy version.
func (c *SymPyClient) Version() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.version
}

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
	candidates := []string{
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
	return "", errors.New("cannot locate test-scripts/go/fuzz/sympy_oracle.py")
}

func (c *SymPyClient) startWorker() error {
	gen := c.generation.Add(1)

	cmd := exec.Command(c.pythonPath, "-u", c.scriptPath)
	cmd.Env = append(os.Environ(),
		"PYTHONUNBUFFERED=1",
		"PYTHONIOENCODING=utf-8",
		"PYTHONUTF8=1",
		"PYTHONHASHSEED=0",
	)

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe error: %w", err)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdinPipe.Close()
		return fmt.Errorf("stdout pipe error: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		_ = stdinPipe.Close()
		_ = stdoutPipe.Close()
		return fmt.Errorf("stderr pipe error: %w", err)
	}

	if err := cmd.Start(); err != nil {
		_ = stdinPipe.Close()
		_ = stdoutPipe.Close()
		_ = stderrPipe.Close()
		return fmt.Errorf("cmd start error: %w", err)
	}

	// Drain stderr continuously in background to avoid pipe buffer deadlocks
	go func() {
		_, _ = io.Copy(io.Discard, stderrPipe)
	}()

	readerDone := make(chan struct{})
	c.cmd = cmd
	c.stdin = stdinPipe
	c.readerDone = readerDone
	c.pid = cmd.Process.Pid
	c.requestCount = 0

	c.pendingMu.Lock()
	c.pending = make(map[uint64]chan workerResponse)
	c.pendingMu.Unlock()

	// Dedicated single reader goroutine per process generation
	reader := bufio.NewReaderSize(stdoutPipe, 1024*1024)
	go func(currentGen uint64, r *bufio.Reader, done chan struct{}) {
		defer close(done)
		for {
			line, rErr := r.ReadBytes('\n')
			if rErr != nil {
				// Broadcast pipe termination to all active pending channels of this generation
				c.pendingMu.Lock()
				for id, ch := range c.pending {
					ch <- workerResponse{generation: currentGen, id: id, err: rErr}
				}
				c.pending = make(map[uint64]chan workerResponse)
				c.pendingMu.Unlock()
				return
			}

			var hdr struct {
				ID         uint64 `json:"id"`
				Generation uint64 `json:"generation"`
			}
			if jErr := json.Unmarshal(line, &hdr); jErr == nil {
				c.pendingMu.Lock()
				// Only dispatch if both generation and request ID match
				if ch, ok := c.pending[hdr.ID]; ok && hdr.Generation == currentGen {
					delete(c.pending, hdr.ID)
					ch <- workerResponse{generation: currentGen, id: hdr.ID, data: line, err: nil}
				}
				c.pendingMu.Unlock()
			}
		}
	}(gen, reader, readerDone)

	// Handshake verification
	reqID := c.seq.Add(1)
	handshakeReq := map[string]any{
		"id":         reqID,
		"generation": gen,
		"action":     "handshake",
	}
	handshakeBytes, _ := json.Marshal(handshakeReq)
	handshakeBytes = append(handshakeBytes, '\n')

	respCh := make(chan workerResponse, 1)
	c.pendingMu.Lock()
	c.pending[reqID] = respCh
	c.pendingMu.Unlock()

	if _, wErr := c.stdin.Write(handshakeBytes); wErr != nil {
		c.stopWorkerLocked()
		return fmt.Errorf("failed to write handshake: %w", wErr)
	}

	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()

	select {
	case <-timer.C:
		c.stopWorkerLocked()
		return errors.New("handshake timed out after 5s")
	case resp := <-respCh:
		if resp.err != nil {
			c.stopWorkerLocked()
			return fmt.Errorf("handshake read failed: %w", resp.err)
		}
		var res struct {
			ID            uint64 `json:"id"`
			Generation    uint64 `json:"generation"`
			Status        string `json:"status"`
			Version       string `json:"sympy_version"`
			MpmathVersion string `json:"mpmath_version"`
			Error         string `json:"error"`
		}
		if uErr := json.Unmarshal(resp.data, &res); uErr != nil {
			c.stopWorkerLocked()
			return fmt.Errorf("malformed handshake response: %w", uErr)
		}
		if res.Error != "" {
			c.stopWorkerLocked()
			return fmt.Errorf("handshake worker error: %s", res.Error)
		}
		if res.Status != "ok" {
			c.stopWorkerLocked()
			return fmt.Errorf("worker status not ok: %s", res.Status)
		}
		c.version = res.Version
		c.mpmathVer = res.MpmathVersion
		return nil
	}
}

func (c *SymPyClient) stopWorkerLocked() {
	if c.stdin != nil {
		_ = c.stdin.Close()
		c.stdin = nil
	}
	if c.cmd != nil && c.cmd.Process != nil {
		pid := c.cmd.Process.Pid
		if runtime.GOOS == "windows" {
			_ = exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(pid)).Run()
		} else {
			_ = c.cmd.Process.Kill()
		}
		_ = c.cmd.Wait()
		c.cmd = nil
	}
	if c.readerDone != nil {
		select {
		case <-c.readerDone:
		case <-time.After(500 * time.Millisecond):
		}
		c.readerDone = nil
	}
}

// Close terminates the worker process cleanly.
func (c *SymPyClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stopWorkerLocked()
	c.available = false
}

func (c *SymPyClient) sendRequestLocked(req map[string]any, timeout time.Duration) ([]byte, error) {
	if !c.available {
		return nil, errors.New("sympy worker not available")
	}

	// Hybrid cache lifecycle: clear_cache every 200 requests, hard restart every 5000 requests
	c.requestCount++
	if c.requestCount >= 5000 {
		c.stopWorkerLocked()
		if err := c.startWorker(); err != nil {
			c.available = false
			c.unavailableR = fmt.Sprintf("hard recycle failed: %v", err)
			return nil, err
		}
	} else if c.requestCount%200 == 0 {
		// Lightweight clear_cache RPC
		clearReq := map[string]any{
			"id":         c.seq.Add(1),
			"generation": c.generation.Load(),
			"action":     "clear_cache",
		}
		clearBytes, _ := json.Marshal(clearReq)
		clearBytes = append(clearBytes, '\n')
		_, _ = c.stdin.Write(clearBytes)
	}

	reqID := c.seq.Add(1)
	gen := c.generation.Load()
	req["id"] = reqID
	req["generation"] = gen

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	reqBytes = append(reqBytes, '\n')

	respCh := make(chan workerResponse, 1)
	c.pendingMu.Lock()
	c.pending[reqID] = respCh
	c.pendingMu.Unlock()

	if _, wErr := c.stdin.Write(reqBytes); wErr != nil {
		c.pendingMu.Lock()
		delete(c.pending, reqID)
		c.pendingMu.Unlock()
		c.stopWorkerLocked()
		if rErr := c.startWorker(); rErr != nil {
			c.available = false
			c.unavailableR = fmt.Sprintf("respawn after write failure failed: %v", rErr)
		}
		return nil, fmt.Errorf("stdin write error: %w", wErr)
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-timer.C:
		c.pendingMu.Lock()
		delete(c.pending, reqID)
		c.pendingMu.Unlock()

		c.stopWorkerLocked()
		if rErr := c.startWorker(); rErr != nil {
			c.available = false
			c.unavailableR = fmt.Sprintf("respawn after timeout failed: %v", rErr)
			return nil, fmt.Errorf("query timed out after %v, respawn failed: %w", timeout, rErr)
		}
		return nil, fmt.Errorf("query timed out after %v", timeout)

	case resp := <-respCh:
		if resp.err != nil {
			c.stopWorkerLocked()
			if rErr := c.startWorker(); rErr != nil {
				c.available = false
				c.unavailableR = fmt.Sprintf("respawn after pipe failure failed: %v", rErr)
				return nil, fmt.Errorf("pipe error: %v, respawn failed: %w", resp.err, rErr)
			}
			return nil, resp.err
		}
		return resp.data, nil
	}
}

// PointEvaluation represents the result of evaluating an expression at a single point in SymPy.
type PointEvaluation struct {
	Undefined bool    `json:"undefined"`
	Reason    string  `json:"reason"`
	Exact     bool    `json:"exact"`
	ReNum     string  `json:"re_num"`
	ReDen     string  `json:"re_den"`
	ImNum     string  `json:"im_num"`
	ImDen     string  `json:"im_den"`
	ApproxStr string  `json:"approx_str"`
	ReFloat   float64 `json:"re_float"`
	ImFloat   float64 `json:"im_float"`
}

// ToGaussianRat converts an exact PointEvaluation to a gaussianRat in Q[i].
func (p PointEvaluation) ToGaussianRat() (gaussianRat, bool) {
	if p.Undefined || !p.Exact {
		return gaussianRat{}, false
	}
	rn, ok1 := new(big.Int).SetString(p.ReNum, 10)
	rd, ok2 := new(big.Int).SetString(p.ReDen, 10)
	in, ok3 := new(big.Int).SetString(p.ImNum, 10)
	id, ok4 := new(big.Int).SetString(p.ImDen, 10)
	if !ok1 || !ok2 || !ok3 || !ok4 || rd.Sign() == 0 || id.Sign() == 0 {
		return gaussianRat{}, false
	}
	return newGaussianRat(new(big.Rat).SetFrac(rn, rd), new(big.Rat).SetFrac(in, id)), true
}

// EvalPoints requests SymPy to evaluate a structured AST expression across sample points.
func (c *SymPyClient) EvalPoints(ctx context.Context, exprAST map[string]any, points []map[string]map[string]string) ([]PointEvaluation, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	req := map[string]any{
		"action": "eval_points",
		"expr":   exprAST,
		"points": points,
	}

	timeout := 5 * time.Second
	if dl, ok := ctx.Deadline(); ok {
		if remaining := time.Until(dl); remaining > 0 && remaining < timeout {
			timeout = remaining
		}
	}

	data, err := c.sendRequestLocked(req, timeout)
	if err != nil {
		return nil, err
	}

	var res struct {
		ID          uint64            `json:"id"`
		Generation  uint64            `json:"generation"`
		Status      string            `json:"status"`
		Evaluations []PointEvaluation `json:"evaluations"`
		Error       string            `json:"error"`
	}
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}
	if res.Error != "" {
		return nil, errors.New(res.Error)
	}
	return res.Evaluations, nil
}

// NumericalPointResult represents the dual-tolerance check result at a single point.
type NumericalPointResult struct {
	Valid     bool    `json:"valid"`
	Agree     bool    `json:"agree"`
	Residual  float64 `json:"residual"`
	Tolerance float64 `json:"tolerance"`
	Reason    string  `json:"reason"`
}

// NumericalResult holds the aggregate outcome of Tier-3 numerical verification.
type NumericalResult struct {
	AllAgree     bool                   `json:"all_agree"`
	SampleCount  int                    `json:"sample_count"`
	MaxResidual  float64                `json:"max_residual"`
	PointResults []NumericalPointResult `json:"point_results"`
}

// EvalNumerical evaluates LHS and RHS at high precision with dual tolerance.
func (c *SymPyClient) EvalNumerical(ctx context.Context, lhsAST, rhsAST map[string]any, points []map[string]map[string]string, atol, rtol float64) (NumericalResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	req := map[string]any{
		"action": "eval_numerical",
		"lhs":    lhsAST,
		"rhs":    rhsAST,
		"points": points,
		"atol":   atol,
		"rtol":   rtol,
	}

	timeout := 5 * time.Second
	if dl, ok := ctx.Deadline(); ok {
		if remaining := time.Until(dl); remaining > 0 && remaining < timeout {
			timeout = remaining
		}
	}

	data, err := c.sendRequestLocked(req, timeout)
	if err != nil {
		return NumericalResult{}, err
	}

	var res struct {
		ID          uint64                 `json:"id"`
		Generation  uint64                 `json:"generation"`
		Status      string                 `json:"status"`
		AllAgree    bool                   `json:"all_agree"`
		SampleCount int                    `json:"sample_count"`
		MaxResidual float64                `json:"max_residual"`
		Points      []NumericalPointResult `json:"point_results"`
		Error       string                 `json:"error"`
	}
	if err := json.Unmarshal(data, &res); err != nil {
		return NumericalResult{}, fmt.Errorf("unmarshal error: %w", err)
	}
	if res.Error != "" {
		return NumericalResult{}, errors.New(res.Error)
	}
	return NumericalResult{
		AllAgree:     res.AllAgree,
		SampleCount:  res.SampleCount,
		MaxResidual:  res.MaxResidual,
		PointResults: res.Points,
	}, nil
}

// SubtractiveZero checks if SymPy simplifies LHS - RHS to zero.
func (c *SymPyClient) SubtractiveZero(ctx context.Context, lhsAST, rhsAST map[string]any) (bool, bool, string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	req := map[string]any{
		"action": "subtractive_zero",
		"lhs":    lhsAST,
		"rhs":    rhsAST,
	}

	timeout := 4 * time.Second
	if dl, ok := ctx.Deadline(); ok {
		if remaining := time.Until(dl); remaining > 0 && remaining < timeout {
			timeout = remaining
		}
	}

	data, err := c.sendRequestLocked(req, timeout)
	if err != nil {
		return false, false, "", err
	}

	var res struct {
		ID                 uint64 `json:"id"`
		Generation         uint64 `json:"generation"`
		Status             string `json:"status"`
		IsZero             bool   `json:"is_zero"`
		IsConstantNonzero  bool   `json:"is_constant_nonzero"`
		Simplified         string `json:"simplified"`
		Error              string `json:"error"`
	}
	if err := json.Unmarshal(data, &res); err != nil {
		return false, false, "", fmt.Errorf("unmarshal error: %w", err)
	}
	if res.Error != "" {
		return false, false, "", errors.New(res.Error)
	}
	return res.IsZero, res.IsConstantNonzero, res.Simplified, nil
}

// ASTToSymPyJSON converts an ihd AST Node into a structured JSON dictionary for SymPy.
func ASTToSymPyJSON(node ast.Node) (map[string]any, error) {
	if node == nil {
		return map[string]any{"op": "Integer", "val": 0}, nil
	}

	switch n := node.(type) {
	case *ast.RationalNode:
		if n.Val == nil {
			return map[string]any{"op": "Integer", "val": 0}, nil
		}
		return map[string]any{
			"op":  "Rational",
			"num": n.Val.Num().String(),
			"den": n.Val.Denom().String(),
		}, nil

	case *ast.VarNode:
		return map[string]any{
			"op":   "Symbol",
			"name": n.Name,
		}, nil

	case *ast.AddNode:
		args := make([]map[string]any, 0, len(n.Terms))
		for _, a := range n.Terms {
			j, err := ASTToSymPyJSON(a)
			if err != nil {
				return nil, err
			}
			args = append(args, j)
		}
		return map[string]any{"op": "Add", "args": args}, nil

	case *ast.MulNode:
		args := make([]map[string]any, 0, len(n.Factors))
		for _, a := range n.Factors {
			j, err := ASTToSymPyJSON(a)
			if err != nil {
				return nil, err
			}
			args = append(args, j)
		}
		return map[string]any{"op": "Mul", "args": args}, nil

	case *ast.PowNode:
		base, err := ASTToSymPyJSON(n.Base)
		if err != nil {
			return nil, err
		}
		exp, err := ASTToSymPyJSON(n.Exp)
		if err != nil {
			return nil, err
		}
		return map[string]any{"op": "Pow", "base": base, "exp": exp}, nil

	case *ast.UnaryOpNode:
		arg, err := ASTToSymPyJSON(n.Expr)
		if err != nil {
			return nil, err
		}
		if n.Op == "-" {
			return map[string]any{"op": "Neg", "arg": arg}, nil
		}
		return arg, nil

	case *ast.SqrtNode:
		arg, err := ASTToSymPyJSON(n.Radicand)
		if err != nil {
			return nil, err
		}
		return map[string]any{"op": "Sqrt", "arg": arg}, nil

	case *ast.ConstNode:
		return map[string]any{"op": "Constant", "name": n.Name}, nil

	case *ast.FuncNode:
		switch strings.ToLower(n.Name) {
		case "sin":
			if len(n.Args) != 1 {
				return nil, fmt.Errorf("sin expects 1 arg, got %d", len(n.Args))
			}
			arg, err := ASTToSymPyJSON(n.Args[0])
			if err != nil {
				return nil, err
			}
			return map[string]any{"op": "Sin", "arg": arg}, nil
		case "cos":
			if len(n.Args) != 1 {
				return nil, fmt.Errorf("cos expects 1 arg, got %d", len(n.Args))
			}
			arg, err := ASTToSymPyJSON(n.Args[0])
			if err != nil {
				return nil, err
			}
			return map[string]any{"op": "Cos", "arg": arg}, nil
		case "tan":
			if len(n.Args) != 1 {
				return nil, fmt.Errorf("tan expects 1 arg, got %d", len(n.Args))
			}
			arg, err := ASTToSymPyJSON(n.Args[0])
			if err != nil {
				return nil, err
			}
			return map[string]any{"op": "Tan", "arg": arg}, nil
		case "exp":
			if len(n.Args) != 1 {
				return nil, fmt.Errorf("exp expects 1 arg, got %d", len(n.Args))
			}
			arg, err := ASTToSymPyJSON(n.Args[0])
			if err != nil {
				return nil, err
			}
			return map[string]any{"op": "Exp", "arg": arg}, nil
		case "log", "ln":
			if len(n.Args) != 1 {
				return nil, fmt.Errorf("log expects 1 arg, got %d", len(n.Args))
			}
			arg, err := ASTToSymPyJSON(n.Args[0])
			if err != nil {
				return nil, err
			}
			return map[string]any{"op": "Log", "arg": arg}, nil
		case "abs":
			if len(n.Args) != 1 {
				return nil, fmt.Errorf("abs expects 1 arg, got %d", len(n.Args))
			}
			arg, err := ASTToSymPyJSON(n.Args[0])
			if err != nil {
				return nil, err
			}
			return map[string]any{"op": "Abs", "arg": arg}, nil
		case "diff":
			if len(n.Args) < 2 {
				return nil, fmt.Errorf("diff expects at least 2 args, got %d", len(n.Args))
			}
			arg, err := ASTToSymPyJSON(n.Args[0])
			if err != nil {
				return nil, err
			}
			varName := "x"
			if v, ok := n.Args[1].(*ast.VarNode); ok {
				varName = v.Name
			}
			order := 1
			if len(n.Args) >= 3 {
				if r, ok := n.Args[2].(*ast.RationalNode); ok && r.Val != nil && r.Val.IsInt() {
					order = int(r.Val.Num().Int64())
				}
			}
			return map[string]any{"op": "Diff", "arg": arg, "var": varName, "order": order}, nil
		case "expand":
			if len(n.Args) != 1 {
				return nil, fmt.Errorf("expand expects 1 arg, got %d", len(n.Args))
			}
			arg, err := ASTToSymPyJSON(n.Args[0])
			if err != nil {
				return nil, err
			}
			return map[string]any{"op": "Expand", "arg": arg}, nil
		case "factor":
			if len(n.Args) != 1 {
				return nil, fmt.Errorf("factor expects 1 arg, got %d", len(n.Args))
			}
			arg, err := ASTToSymPyJSON(n.Args[0])
			if err != nil {
				return nil, err
			}
			return map[string]any{"op": "Factor", "arg": arg}, nil
		default:
			return nil, fmt.Errorf("unsupported FuncNode: %s", n.Name)
		}

	default:
		return nil, fmt.Errorf("unsupported AST node type for SymPy serialization: %T", node)
	}
}

// TestSymPyClient_Handshake validates worker startup, version reporting, and roundtrip JSON communication.
func TestSymPyClient_Handshake(t *testing.T) {
	client := GetGlobalSymPyClient()
	if !client.Available() {
		t.Skipf("SymPy not available: %s", client.ReasonUnavailable())
	}

	t.Logf("SymPy version: %s, mpmath version: %s (PID: %d)", client.Version(), client.mpmathVer, client.pid)
	if client.Version() == "" {
		t.Fatalf("Expected non-empty SymPy version")
	}

	// Test point evaluation of (x + 1)^2 at x = 3
	addExpr := ast.NewAdd([]ast.Node{ast.NewVar("x"), ast.NewRationalFromBigRat(big.NewRat(1, 1))})
	twoExpr := ast.NewRationalFromBigRat(big.NewRat(2, 1))
	expr, err := ast.NewPow(addExpr, twoExpr)
	if err != nil {
		t.Fatalf("NewPow error: %v", err)
	}
	jsonAST, err := ASTToSymPyJSON(expr)
	if err != nil {
		t.Fatalf("ASTToSymPyJSON error: %v", err)
	}

	pt := map[string]map[string]string{
		"x": {
			"re_num": "3", "re_den": "1",
			"im_num": "0", "im_den": "1",
		},
	}
	evals, err := client.EvalPoints(context.Background(), jsonAST, []map[string]map[string]string{pt})
	if err != nil {
		t.Fatalf("EvalPoints error: %v", err)
	}
	if len(evals) != 1 {
		t.Fatalf("Expected 1 evaluation, got %d", len(evals))
	}
	gRat, ok := evals[0].ToGaussianRat()
	if !ok {
		t.Fatalf("Expected exact Gaussian rational evaluation, got: %+v", evals[0])
	}
	expected := big.NewRat(16, 1)
	if gRat.re.Cmp(expected) != 0 || gRat.im.Sign() != 0 {
		t.Fatalf("Expected 16, got %v + %vi", gRat.re, gRat.im)
	}
}
