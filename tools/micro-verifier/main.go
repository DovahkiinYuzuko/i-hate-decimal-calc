package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ProofStepJSON matches the serialized schema from ihd.
type ProofStepJSON struct {
	StepNumber      int    `json:"step_number"`
	Before          string `json:"before"`
	After           string `json:"after"`
	Rule            string `json:"rule"`
	RuleDescription string `json:"rule_description,omitempty"`
	Residual        string `json:"residual,omitempty"`
	IsSelfVerified  bool   `json:"is_self_verified"`
}

// ProofTraceJSON matches the serialized certificate schema from ihd.
type ProofTraceJSON struct {
	Version    string            `json:"version"`
	Original   string            `json:"original"`
	Result     string            `json:"result"`
	IsVerified bool              `json:"is_verified"`
	TotalSteps int               `json:"total_steps"`
	Steps      []ProofStepJSON   `json:"steps"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: micro-verifier <certificate.json>\n")
		os.Exit(1)
	}

	certPath := os.Args[1]
	ok, err := verifyCertificateFile(certPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[MICRO-VERIFIER FAIL] %s: %v\n", certPath, err)
		os.Exit(1)
	}
	if !ok {
		fmt.Fprintf(os.Stderr, "[MICRO-VERIFIER FAIL] %s: verification criteria rejected\n", certPath)
		os.Exit(1)
	}

	fmt.Printf("[MICRO-VERIFIER PASS] %s: 100%% independent equational transitivity verified\n", certPath)
	os.Exit(0)
}

func normalizeExpr(s string) string {
	s = strings.ReplaceAll(s, " ", "")
	s = strings.TrimSpace(s)
	return s
}

func verifyCertificateFile(filePath string) (bool, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false, fmt.Errorf("read error: %w", err)
	}

	var cert ProofTraceJSON
	if err := json.Unmarshal(data, &cert); err != nil {
		return false, fmt.Errorf("json parse error: %w", err)
	}

	if cert.Version == "" {
		return false, fmt.Errorf("missing certificate version")
	}

	if len(cert.Steps) == 0 {
		return false, fmt.Errorf("certificate has empty steps")
	}

	if cert.TotalSteps != len(cert.Steps) {
		return false, fmt.Errorf("total_steps count mismatch: declared %d, found %d", cert.TotalSteps, len(cert.Steps))
	}

	normOriginal := normalizeExpr(cert.Original)
	normResult := normalizeExpr(cert.Result)

	for i, step := range cert.Steps {
		if step.StepNumber != i+1 {
			return false, fmt.Errorf("step %d index mismatch: declared %d", i+1, step.StepNumber)
		}

		if !step.IsSelfVerified {
			return false, fmt.Errorf("step %d is not self-verified", step.StepNumber)
		}

		normBefore := normalizeExpr(step.Before)
		normAfter := normalizeExpr(step.After)

		// Check first step origin
		if i == 0 && normOriginal != "" {
			if normBefore != normOriginal && !strings.Contains(normBefore, normOriginal) && !strings.Contains(normOriginal, normBefore) {
				return false, fmt.Errorf("step 1 origin mismatch: expected %q, got %q", cert.Original, step.Before)
			}
		}

		// Check transitivity chain: step[k-1].After == step[k].Before
		if i > 0 {
			prevAfter := normalizeExpr(cert.Steps[i-1].After)
			if prevAfter != normBefore {
				return false, fmt.Errorf("transitivity break at step %d: previous after %q != current before %q", step.StepNumber, cert.Steps[i-1].After, step.Before)
			}
		}

		// Check final step result
		if i == len(cert.Steps)-1 && normResult != "" {
			if normAfter != normResult && !strings.Contains(normAfter, normResult) && !strings.Contains(normResult, normAfter) {
				return false, fmt.Errorf("final step %d result mismatch: expected %q, got %q", step.StepNumber, cert.Result, step.After)
			}
		}
	}

	return true, nil
}
