package calc_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
)

type docSpecsData struct {
	Version     string                     `json:"version"`
	GeneratedAt string                     `json:"generated_at"`
	Languages   map[string]langDocSpecData `json:"languages"`
}

type langDocSpecData struct {
	Functions []funcDocSpecData `json:"functions"`
}

type funcDocSpecData struct {
	Category    string   `json:"category"`
	Names       []string `json:"names"`
	Examples    []string `json:"examples"`
	Description string   `json:"description"`
}

var knownNonFuncItems = map[string]bool{
	"√":               true,
	"_var":            true,
	"[unit]":          true,
	"clear":           true,
	"exit":            true,
	"quit":            true,
	"help":            true,
	"highlight":       true,
	"highlight on":    true,
	"off":             true,
	"theme":           true,
	"theme dark":      true,
	"theme light":     true,
	"save":            true,
	"load":            true,
	"similar":         true,
	"vars":            true,
	"whatis":          true,
	"reset":           true,
	"config loop":     true,
	"warn-loop":       true,
	"max-loop":        true,
	"search":          true,
	"ProofTrace":      true,
	"micro-verifier":  true,
	"steps":           true,
	"explain":         true,
	"--verify":        true,
	"--explain":       true,
	"--explain --lean":  true,
	"--explain --typst": true,
	"--typst":         true,
	"--format typst":  true,
	"--lean":          true,
	"--lean-file":     true,
	"--replay":        true,
	"--certificate":   true,
}

var optArgRegexPattern = regexp.MustCompile(`,?\s*\[([a-zA-Z0-9_]+)\]`)

func TestReferenceDocumentationInvariants(t *testing.T) {
	// Locate doc_specs.json from either repo root or internal/calc directory
	specsPath := filepath.Join("..", "..", "docs", "references", "dist", "doc_specs.json")
	if _, err := os.Stat(specsPath); os.IsNotExist(err) {
		specsPath = filepath.Join("docs", "references", "dist", "doc_specs.json")
	}

	bytes, err := os.ReadFile(specsPath)
	if err != nil {
		t.Skipf("Skipping TestReferenceDocumentationInvariants: %s not found (run reference-builder first)", specsPath)
		return
	}

	var specs docSpecsData
	if err := json.Unmarshal(bytes, &specs); err != nil {
		t.Fatalf("Failed to parse %s: %v", specsPath, err)
	}

	allRegistered := calc.AllFunctionNames()
	regMap := make(map[string]calc.FunctionSpec, len(allRegistered))
	for _, name := range allRegistered {
		if spec, ok := calc.LookupFunction(name); ok {
			regMap[name] = spec
			regMap[strings.ToLower(name)] = spec
		}
	}

	for _, lang := range []string{"ja", "en"} {
		t.Run("Language_"+lang, func(t *testing.T) {
			lSpec, ok := specs.Languages[lang]
			if !ok {
				t.Fatalf("Language '%s' missing from doc_specs.json", lang)
			}

			docSymbols := make(map[string]bool)

			for _, fn := range lSpec.Functions {
				for _, name := range fn.Names {
					cleanName := strings.TrimSpace(name)
					docSymbols[cleanName] = true
					docSymbols[strings.ToLower(cleanName)] = true

					// Check for unreserved ghost functions
					if !knownNonFuncItems[cleanName] && !knownNonFuncItems[strings.ToLower(cleanName)] {
						if _, found := regMap[cleanName]; !found {
							t.Logf("[%s] Warning: symbol '%s' documented in table but not in registry.go", lang, cleanName)
						}
					}
				}

				isReplCategory := strings.Contains(fn.Category, "REPL") ||
					strings.Contains(fn.Category, "対話環境") ||
					strings.Contains(fn.Category, "Interactive Commands")

				// Validate syntax examples
				for _, example := range fn.Examples {
					ex := strings.TrimSpace(example)
					if ex == "" || strings.HasPrefix(ex, "ihd ") || strings.HasPrefix(ex, "--") || knownNonFuncItems[ex] {
						continue
					}
					if isReplCategory || strings.HasPrefix(ex, "micro-verifier ") {
						continue
					}

					normalized := ex
					if strings.Contains(normalized, "...") {
						normalized = strings.ReplaceAll(normalized, ", ...", "")
						normalized = strings.ReplaceAll(normalized, "...", "x")
					}

					node, err := calc.Parse(normalized)
					if err != nil {
						withoutOpt := optArgRegexPattern.ReplaceAllString(ex, "")
						node, err = calc.Parse(withoutOpt)
					}

					if err != nil {
						t.Errorf("[%s] Failed to parse syntax example '%s' for '%s': %v",
							lang, ex, fn.Names[0], err)
						continue
					}

					// Verify argument count
					if fnNode, ok := node.(*calc.FuncNode); ok {
						spec, found := regMap[fnNode.Name]
						if found {
							argCount := len(fnNode.Args)
							if spec.MinArgs == spec.MaxArgs {
								if argCount != spec.MinArgs {
									t.Errorf("[%s] Function '%s' argument mismatch in example '%s': got %d, expected exactly %d",
										lang, fnNode.Name, ex, argCount, spec.MinArgs)
								}
							} else {
								if argCount < spec.MinArgs || (spec.MaxArgs != -1 && argCount > spec.MaxArgs) {
									t.Errorf("[%s] Function '%s' argument mismatch in example '%s': got %d, expected between %d and %d",
										lang, fnNode.Name, ex, argCount, spec.MinArgs, spec.MaxArgs)
								}
							}
						}
					}

					// Smoke test evaluation
					env := calc.NewEnv()
					_, evalErr := calc.EvalWithEnv(node, env)
					if evalErr != nil {
						errMsg := evalErr.Error()
						if strings.Contains(errMsg, "imaginary unit") ||
							strings.Contains(errMsg, "invalid digit") {
							t.Errorf("[%s] Example '%s' failed evaluation: %v", lang, ex, evalErr)
						}
					}
				}
			}

			// Verify completeness: all registered functions must be documented
			for _, name := range allRegistered {
				if !docSymbols[name] && !docSymbols[strings.ToLower(name)] {
					t.Errorf("[%s] Registered function '%s' is not documented in reference", lang, name)
				}
			}
		})
	}
}
