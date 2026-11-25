# ihd Development, Testing & Verification Guide

This guide is a test and verification manual for developers and contributors working in a cloned `ihd` repository. It covers unit tests, E2E tests, the unified repository linter, mathematical invariant fuzzing, standalone proof verification, Lean 4 kernel checks, and documentation audits. Unlike the end-user syntax reference bundled in release archives and `~/.ihd/references/` ([reference.en.md](references/en/reference.en.md)), this guide targets only Git-tracked tooling and test suites.

[日本語](development.ja.md) | [English](development.en.md) | [Back to README](../README.en.md)

| Requirement | Tool / Version | Purpose |
| :--- | :--- | :--- |
| Required | Go 1.26.5 | Core builds, unit tests, E2E runner, repository linter, `micro-verifier`, `doc-auditor` |
| Optional | PowerShell (`pwsh`) or `bash` | Running the Exam showcase |
| Optional | Python 3 + SymPy | Resident differential oracle (`tools/sympy-oracle`) and Lean proof catalog aggregation |
| Optional | Node.js (or Bun) | Rebuilding offline Single-File HTML references and `doc_specs.json` |
| Optional | Lean 4 (`leanprover/lean4:v4.36.0-rc2`) | Mathlib4 kernel verification and axiom audits under `formal/` |

Core checks (unit tests, repository lint, and E2E tests) run with Go alone:

```bash
go test ./...
go run ./tools/repo-lint --all
go run ./tools/e2e
```

## Table of Contents

1. [Enabling Git Hooks (`.githooks/`)](#1-enabling-git-hooks-githooks)
2. [Unit Tests & Go-Native E2E Test Runner (`tools/e2e`)](#2-unit-tests--go-native-e2e-test-runner-toolse2e)
3. [Unified Repository Linter (`tools/repo-lint`)](#3-unified-repository-linter-toolsrepo-lint)
4. [Mathematical Invariant Fuzzing, SymPy Oracle & Bytecode VM Verification](#4-mathematical-invariant-fuzzing-sympy-oracle--bytecode-vm-verification)
5. [Standalone Proof Verifier (`tools/micro-verifier`) & Lean 4 Verification (`formal/`)](#5-standalone-proof-verifier-toolsmicro-verifier--lean-4-verification-formal)
6. [Live Exam Showcase & Reference HTML Build / Audit](#6-live-exam-showcase--reference-html-build--audit)
7. [GitHub Actions CI Workflows](#7-github-actions-ci-workflows)

---

## 1. Enabling Git Hooks (`.githooks/`)

Shared Git hooks are tracked in `.githooks/pre-commit` and `.githooks/pre-push`. Enable them after cloning by configuring `core.hooksPath`:

```bash
git config core.hooksPath .githooks
```

- **`.githooks/pre-commit`**: When Markdown files under `docs/references/` are staged, rebuilds the offline HTML manuals via `node tools/reference-builder/index.js` and syncs `tools/reference-builder/package.json` with the latest Git tag. It then runs `go run ./tools/repo-lint --staged` and blocks the commit with exit code `1` on violations.
- **`.githooks/pre-push`**: When PowerShell is available, attempts to run a local differential fuzz smoke script (skips with a warning and exit code `0` only when PowerShell is absent). In a fresh clone without that local script, `pre-push` fails and blocks the push.

---

## 2. Unit Tests & Go-Native E2E Test Runner (`tools/e2e`)

### Go Unit Tests

Run unit tests across all public packages (`cmd/ihd`, `internal/...`, `tools/...`). Set `IHD_NO_UPDATE_CHECK=1` to skip release checks during test runs:

```bash
IHD_NO_UPDATE_CHECK=1 go test -v -timeout 5m ./...
```

### Go-Native E2E Test Runner (`tools/e2e`)

Automatically builds temporary binaries (`ihd` and `micro-verifier`) and executes the CSV test suites under [tools/e2e/test-cases/](../tools/e2e/test-cases) in parallel. On failure, detailed diagnostics are written to `tools/e2e/logs/e2e_failures_<timestamp>.log` and the runner exits with code `1` (if zero cases match the filter, it prints a warning and exits `0`).

```bash
# Run the full E2E suite
go run ./tools/e2e

# Filter by category, substring, or worker count
go run ./tools/e2e -c plotter -v
go run ./tools/e2e -f "SOS" -j 4
```

| Flag | Description | Default |
| :--- | :--- | :--- |
| `-c`, `--category <str>` | Filter by category name or file path substring (case-sensitive) | All categories |
| `-f`, `--filter <str>` | Filter by case ID or description substring (case-sensitive) | Unfiltered |
| `-j`, `--jobs <int>` | Number of parallel workers (`<= 0` or omitted falls back to CPU core count) | CPU core count |
| `-v`, `--verbose` | Print per-case execution output | `false` |

Each CSV file follows a 10-column schema (`id,category,flags,pipe_input,match_mode,expected_stdout,expected_stderr,expected_exit_code,deduction_method,description`). The repository linter validates `match_mode` (`exact` / `contains` / `regex` / `error`), `deduction_method` (empty or `deductive` / `contradiction` / `contrapositive` / `inductive` / `metamorphic` / `roundtrip`), and cross-file `id` uniqueness.

---

## 3. Unified Repository Linter (`tools/repo-lint`)

Checks repository rules, i18n parity, third-party notice sync, Lean toolchain pinning, Go code hygiene, and E2E CSV integrity in a single command.

```bash
# Check the entire repository (default)
go run ./tools/repo-lint --all

# Check staged files only (same mode used by pre-commit)
go run ./tools/repo-lint --staged

# List rules, run selected checkers, or emit JSON
go run ./tools/repo-lint --list-rules
go run ./tools/repo-lint --only i18n-ast,e2e-csv --format json
```

Exit codes are `0` when clean, `1` when errors are reported (or warnings with `--warn-as-error`), and `2` on unknown/malformed flags or internal/JSON serialization errors.

| Flag | Description | Default & Precedence |
| :--- | :--- | :--- |
| `--all` | Check the entire repository | Default (takes precedence when combined with `--staged`) |
| `--staged` | Check Git-staged files only | `false` |
| `--only <csv>` | Restrict execution to a comma-separated list of checkers/analyzers | All rules |
| `--skip <csv>` | Exclude a comma-separated list of checkers/analyzers | None |
| `--root <dir>` | Set the target repository root directory | `.` |
| `--format <text\|json>` | Set diagnostic output format (any non-`json` value emits text) | `text` |
| `--warn-as-error` | Exit with code `1` on warnings | `false` |
| `--list-rules` | Print all registered checkers and AST analyzers, then exit | `false` |

| Checker / Analyzer | Checked Rules & Boundaries |
| :--- | :--- |
| `readme-quickstart` | Active only under `--staged`. Blocks edits inside the Quick Start section of `README.md` / `README.en.md` unless explicitly approved via `IHD_ALLOW_QUICKSTART_EDIT=1` |
| `git-whitelist` | Rejects forbidden artifacts (`.env`, `.lake/`, local working directories) and Markdown files outside the public allowlist |
| `i18n-ast` | Checks key parity between `internal/i18n/locales/{ja,en}.json`, forbids `%w`, and verifies `i18n.T` keys and format argument counts via Go AST analysis |
| `notice-sync` | Ensures all dependencies in `go.mod` and `tools/reference-builder/package.json` appear in `NOTICE.md` |
| `lean-pinning` | Enforces pinned versions in `formal/lean-toolchain`, 40-char hex SHAs in `formal/lake-manifest.json`, and forbids `lake update` in CI workflows |
| `go-hygiene` | Flags empty `if err != nil {}`, emoji in Go string literals or locale JSON (excluding math symbols, box drawing, and Braille), uncommented `_ = err`, and string-concatenated file paths |
| `e2e-csv` | Validates the 10-column schema, allowed enum values, required fields, and unique IDs across `tools/e2e/test-cases/**/*.csv` |

When suppressing a specific `go-hygiene` diagnostic with justification, add `// repo-lint:ignore <rule-id|all> <concrete reason>` on the line immediately preceding the target line or inline on the same line (omitting the reason fails the check).

---

## 4. Mathematical Invariant Fuzzing, SymPy Oracle & Bytecode VM Verification

### Property-Based Testing & Fuzzing

Run the fuzzing harness backed by exact multi-point evaluation oracles over Gaussian rationals $`\mathbb{Q}[i]`$ (Schwartz-Zippel lemma) and algebraic/calculus invariants. Main targets are defined in `fuzz_*_test.go` under [internal/calc/](../internal/calc):

- `FuzzAlgebra_SimplifyIdempotence`: simplification idempotence
- `FuzzAlgebra_FactorExpandRoundTrip`: factor-expand round trips
- `FuzzCalculus_DiffIntegrateInversion`: differentiation-integration inversion
- `FuzzDomain_UnitsDimensionalConsistency`: physical unit dimensional consistency

```bash
# Smoke test all Fuzz target seed corpora, oracle, and provider tests
go test -v ./internal/calc -run "^(Fuzz|TestThreeTierOracle_|TestByteStreamProvider_)"

# Continuous fuzzing exploration on a specific invariant target (e.g., 30s)
go test ./internal/calc -run "^FuzzAlgebra_SimplifyIdempotence$" -fuzz "^FuzzAlgebra_SimplifyIdempotence$" -fuzztime 30s
```

### Resident SymPy Differential Fuzzing Oracle (`tools/sympy-oracle`)

When Python 3 and SymPy are installed, `tools/sympy-oracle/sympy_oracle.py` runs as a resident NDJSON process over stdin/stdout for differential equivalence testing against SymPy. Differential tests skip automatically when SymPy is absent; set `IHD_REQUIRE_SYMPY_ORACLE=1` to fail instead of skipping.

```bash
# Match the CI fuzz-smoke workflow with SymPy required
IHD_REQUIRE_SYMPY_ORACLE=1 GOMEMLIMIT=2GiB IHD_NO_UPDATE_CHECK=1 \
  go test -v -count=1 -timeout 3m ./internal/calc \
  -run "^(Fuzz|TestThreeTierOracle|TestI18nErrorIntegrity|TestByteStreamProvider|TestGuard|TestDifferentialOracle|TestASTShrinker|TestSymPyClient)"
```

### Bytecode VM Invariants & Benchmarks

Verify equivalence between the bytecode VM and the AST evaluator, and measure relative throughput:

```bash
# VM invariant and differential tests
go test -v -count=1 -run "TestVM_Invariants|TestVM_Differential" ./internal/calc/...

# Relative speedup benchmark against AST evaluation
go test -v -run "^$" -bench "BenchmarkVM_RelativeSpeedup" -benchmem -benchtime 1s ./internal/calc/...
```

---

## 5. Standalone Proof Verifier (`tools/micro-verifier`) & Lean 4 Verification (`formal/`)

### Standalone Proof Verifier (`tools/micro-verifier`)

A standalone CLI tool that independently verifies certificate JSON files (version `"1.0"`) emitted by `ihd --certificate <file>` without relying on `ihd` solver logic:

```bash
# Unit tests for verifier
go test ./tools/micro-verifier

# Standalone verification of a certificate file
go run ./tools/micro-verifier cert.json
```

### Lean 4 Formal Verification (`formal/`)

Theorem files generated by `ihd --lean` / `--lean-file` can be kernel-checked using the Lean 4 project in [formal/](../formal). To preserve reproducibility, `formal/lean-toolchain` is pinned to `leanprover/lean4:v4.36.0-rc2`, `formal/lake-manifest.json` pins 40-character commit SHAs, and running `lake update` is disallowed by `lean-pinning`.

```bash
# Example: emit a Lean 4 proof, fetch Mathlib cache, and run kernel verification
go run ./cmd/ihd --lean-file formal/Proofs/Sample.lean "factor(x^4 + 4)"
cd formal
lake exe cache get
lake env lean Proofs/Sample.lean

# Build and self-test the axiom auditor (rejects sorryAx and unexpected axioms)
lake build ihd-audit
lake exe ihd-audit --self-test

# Update proof catalog and check ratchet constraints (from repository root)
cd ..
python3 tools/lean/proof_catalog.py
```

---

## 6. Live Exam Showcase & Reference HTML Build / Audit

### Live Exam Showcase

An automated showcase solving representative math problems ranging from high school math to competitions and university STEM. Problem statements and solutions are documented in the [English Edition Cookbook](../exam/problems.en.md).

```bash
# Windows (PowerShell)
pwsh exam/exam.ps1

# Linux / macOS (Bash)
./exam/exam.sh
```

### Reference HTML Build & Documentation Audit

On a fresh checkout where `tools/reference-builder/node_modules` does not yet exist, run `npm ci` inside `tools/reference-builder` first. After editing `docs/references/{ja,en}/reference.{ja,en}.md`, rebuild the offline Single-File HTML manuals and the untracked build artifact `docs/references/dist/doc_specs.json`, then run `tools/doc-auditor`:

```bash
# Rebuild Single-File HTML references and doc_specs.json
node tools/reference-builder/index.js

# Verify function registry, arity invariants, and syntax examples
go run ./tools/doc-auditor
```

When zero discrepancies remain, `doc-auditor` deletes any stale report and exits `0`. If any mismatch is detected, it writes `docs/reference_audit_report.md` and exits `1`. Once `doc_specs.json` is built, `go test ./internal/calc -run TestReferenceDocumentationInvariants` runs the same invariants inside `go test`.

---

## 7. GitHub Actions CI Workflows

The following GitHub Actions workflows enforce verification and release packaging:

| Workflow | Responsibility |
| :--- | :--- |
| `unit-test.yml` | Runs `go run ./tools/repo-lint --all` and full Go unit tests (`go test -v -timeout 5m ./...`) |
| `e2e-test.yml` | Runs `go run ./tools/e2e` across the full E2E CSV suite |
| `reference-sync.yml` | Verifies `node tools/reference-builder/index.js` HTML sync and runs `go run ./tools/doc-auditor` |
| `fuzz-smoke.yml` | Runs Fuzz, PBT, and SymPy differential smoke tests under Python 3.12 + SymPy |
| `lean-verify.yml` | Generates and kernel-checks `formal/test_cases.txt`, and runs `ihd-audit --self-test` and `proof_catalog.py` |
| `typst-verify.yml` | Compiles `ihd --typst` outputs to PDF / SVG via the Typst CLI |
| `vm-perf.yml` | Runs Bytecode VM invariant/differential tests and `BenchmarkVM_RelativeSpeedup` |
| `release.yml` | Builds multi-platform OS/arch targets on `v*` tags, packages archives, writes `SHA256SUMS`, and publishes GitHub Releases |
