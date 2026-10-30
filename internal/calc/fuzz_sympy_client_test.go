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

// SymPyClient manages a resident Python sympy_oracle.py worker communicating via unbuffered NDJSON.
type SymPyClient struct {
	mu           sync.Mutex
	cmd          *exec.Cmd
	stdin        io.WriteCloser
	reader       *bufio.Reader
	seq          atomic.Uint64
	available    bool
	unavailableR string
	pythonPath   string
	scriptPath   string
	version      string
	pid          int
	requestCount int
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
	c := &SymPyClient{}
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
	// Look up relative to project root or current working dir
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

	// Stderr must be drained continuously in background to prevent OS pipe buffer deadlocks
	go func() {
		_, _ = io.Copy(io.Discard, stderrPipe)
	}()

	c.cmd = cmd
	c.stdin = stdinPipe
	c.reader = bufio.NewReaderSize(stdoutPipe, 1024*1024) // 1MB buffer
	c.pid = cmd.Process.Pid
	c.requestCount = 0

	// Handshake verification
	reqID := c.seq.Add(1)
	handshakeReq := map[string]any{
		"id":     reqID,
		"action": "handshake",
	}
	handshakeBytes, _ := json.Marshal(handshakeReq)
	handshakeBytes = append(handshakeBytes, '\n')

	if _, err := c.stdin.Write(handshakeBytes); err != nil {
		c.stopWorkerLocked()
		return fmt.Errorf("failed to write handshake: %w", err)
	}

	line, err := c.reader.ReadBytes('\n')
	if err != nil {
		c.stopWorkerLocked()
		return fmt.Errorf("failed to read handshake: %w", err)
	}

	var res struct {
		ID      uint64 `json:"id"`
		Status  string `json:"status"`
		Version string `json:"sympy_version"`
		PID     int    `json:"pid"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(line, &res); err != nil {
		c.stopWorkerLocked()
		return fmt.Errorf("malformed handshake response: %w", err)
	}
	if res.Error != "" {
		c.stopWorkerLocked()
		return fmt.Errorf("handshake error from worker: %s", res.Error)
	}
	if res.Status != "ok" {
		c.stopWorkerLocked()
		return fmt.Errorf("worker status not ok: %s", res.Status)
	}

	c.version = res.Version
	return nil
}

func (c *SymPyClient) stopWorkerLocked() {
	if c.stdin != nil {
		_ = c.stdin.Close()
		c.stdin = nil
	}
	if c.cmd != nil && c.cmd.Process != nil {
		pid := c.cmd.Process.Pid
		if runtime.GOOS == "windows" {
			// Windows taskkill /F /T kills entire process tree
			_ = exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(pid)).Run()
		} else {
			_ = c.cmd.Process.Kill()
		}
		_ = c.cmd.Wait()
		c.cmd = nil
	}
	c.reader = nil
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

	// Recycle worker periodically (every 1000 queries) to prevent cache/memory leaks
	if c.requestCount >= 1000 {
		c.stopWorkerLocked()
		if err := c.startWorker(); err != nil {
			c.available = false
			c.unavailableR = fmt.Sprintf("recycle failed: %v", err)
			return nil, err
		}
	}
	c.requestCount++

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	reqBytes = append(reqBytes, '\n')

	type result struct {
		data []byte
		err  error
	}
	resChan := make(chan result, 1)

	go func() {
		if _, writeErr := c.stdin.Write(reqBytes); writeErr != nil {
			resChan <- result{nil, fmt.Errorf("stdin write error: %w", writeErr)}
			return
		}
		line, readErr := c.reader.ReadBytes('\n')
		if readErr != nil {
			resChan <- result{nil, fmt.Errorf("stdout read error: %w", readErr)}
			return
		}
		resChan <- result{line, nil}
	}()

	select {
	case <-time.After(timeout):
		// On timeout, kill process tree immediately to prevent orphaned SymPy CPU burn
		c.stopWorkerLocked()
		// Attempt to respawn worker for future requests
		_ = c.startWorker()
		return nil, fmt.Errorf("query timed out after %v", timeout)
	case res := <-resChan:
		if res.err != nil {
			// Restart worker on pipe/read failure
			c.stopWorkerLocked()
			_ = c.startWorker()
			return nil, res.err
		}
		return res.data, nil
	}
}

// PointEvaluation represents the result of evaluating an expression at a single point in SymPy.
type PointEvaluation struct {
	Undefined bool   `json:"undefined"`
	Reason    string `json:"reason"`
	Exact     bool   `json:"exact"`
	ReNum     string `json:"re_num"`
	ReDen     string `json:"re_den"`
	ImNum     string `json:"im_num"`
	ImDen     string `json:"im_den"`
	ApproxStr string `json:"approx_str"`
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

	reqID := c.seq.Add(1)
	req := map[string]any{
		"id":     reqID,
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

// SubtractiveZero checks if SymPy simplifies LHS - RHS to zero.
func (c *SymPyClient) SubtractiveZero(ctx context.Context, lhsAST, rhsAST map[string]any) (bool, string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	reqID := c.seq.Add(1)
	req := map[string]any{
		"id":     reqID,
		"action": "subtractive_zero",
		"lhs":    lhsAST,
		"rhs":    rhsAST,
	}

	timeout := 4 * time.Second
	data, err := c.sendRequestLocked(req, timeout)
	if err != nil {
		return false, "", err
	}

	var res struct {
		ID         uint64 `json:"id"`
		Status     string `json:"status"`
		IsZero     bool   `json:"is_zero"`
		Simplified string `json:"simplified"`
		Error      string `json:"error"`
	}
	if err := json.Unmarshal(data, &res); err != nil {
		return false, "", fmt.Errorf("unmarshal error: %w", err)
	}
	if res.Error != "" {
		return false, "", errors.New(res.Error)
	}
	return res.IsZero, res.Simplified, nil
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
			"assumptions": map[string]bool{
				"real": true,
			},
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

	t.Logf("SymPy version: %s (PID: %d)", client.Version(), client.pid)
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
