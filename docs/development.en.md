# ihd Development, Testing & Verification Guide

[日本語](development.ja.md) | [English](development.en.md) | [Back to README](../README.en.md)

This guide covers how to run Go unit tests, mathematical invariant fuzzing, standalone proof verification, differential SymPy oracles, and documentation audits in a cloned `ihd` repository. Unlike the end-user syntax reference bundled in release archives and `~/.ihd/references/` ([reference.en.md](references/en/reference.en.md)), this guide targets contributors working on the repository source and test suite.

- **Base environment**: Go 1.26.5 (`pwsh` or `bash` for Exam runners)
- **Optional tools (for specific checks)**: Python with SymPy (differential oracle), Node.js (or Bun: reference HTML rebuild)

## Table of Contents

- [1. Unit Tests](#1-unit-tests)
- [2. Mathematical Invariants Property-Based Testing & Fuzzing](#2-mathematical-invariants-property-based-testing--fuzzing)
- [3. Standalone Proof Verifier (`tools/micro-verifier`)](#3-standalone-proof-verifier-toolsmicro-verifier)
- [4. Live Exam Showcase](#4-live-exam-showcase)
- [5. Resident SymPy Differential Fuzzing Oracle (`tools/sympy-oracle`)](#5-resident-sympy-differential-fuzzing-oracle-toolssympy-oracle)
- [6. Reference HTML Build & Documentation Audit](#6-reference-html-build--documentation-audit)

## 1. Unit Tests

Run standard Go unit tests across all packages:

```bash
go test ./...
```

## 2. Mathematical Invariants Property-Based Testing & Fuzzing

Run the fuzzing harness backed by exact multi-point evaluation oracles over Gaussian rationals $\mathbb{Q}[i]$ (Schwartz-Zippel lemma) and algebraic/calculus invariants:

```bash
# Smoke test all Fuzz target seed corpora, oracle, and provider tests
go test -v ./internal/calc -run "^(Fuzz|TestThreeTierOracle_|TestByteStreamProvider_)"

# Continuous fuzzing exploration on a specific invariant target (e.g., 30s)
go test ./internal/calc -run "^FuzzAlgebra_SimplifyIdempotence$" -fuzz "^FuzzAlgebra_SimplifyIdempotence$" -fuzztime 30s
```

Main fuzz targets are defined in `fuzz_*_test.go` under [internal/calc/](../internal/calc), covering simplification idempotence (`FuzzAlgebra_SimplifyIdempotence`), factor-expand round trips (`FuzzAlgebra_FactorExpandRoundTrip`), differentiation-integration inversion (`FuzzCalculus_DiffIntegrateInversion`), and physical unit dimensional consistency (`FuzzDomain_UnitsDimensionalConsistency`).

## 3. Standalone Proof Verifier (`tools/micro-verifier`)

A standalone CLI tool that independently verifies certificate JSON files (version `"1.0"`) emitted by `ihd --certificate <file>` without relying on `ihd` solver logic:

```bash
# Unit tests for verifier
go test ./tools/micro-verifier

# Standalone verification of a certificate file
go run ./tools/micro-verifier cert.json
```

## 4. Live Exam Showcase

An automated showcase solving 24 benchmark problems ranging from high school math to competitions and university STEM. Problem statements and solutions are documented in the [English Edition Cookbook](../exam/problems.en.md).

```bash
# Windows (PowerShell)
pwsh exam/exam.ps1

# Linux / macOS (Bash)
./exam/exam.sh
```

## 5. Resident SymPy Differential Fuzzing Oracle (`tools/sympy-oracle`)

In environments with Python and SymPy installed, the script `tools/sympy-oracle/sympy_oracle.py` provides a resident JSON-RPC oracle for differential equivalence testing against SymPy (calculus, algebraic simplification, rationals, and combinatorics).

```bash
# Run SymPy differential oracle unit tests
go test -v ./internal/calc -run "^(TestSymPyClient_|TestDifferentialOracle_)"
```

## 6. Reference HTML Build & Documentation Audit

After editing `docs/references/{ja,en}/reference.{ja,en}.md`, rebuild the offline Single-File HTML manuals and `doc_specs.json`, then verify registry consistency via `tools/doc-auditor` (run `npm ci` in `tools/reference-builder` on first setup):

```bash
# Rebuild Single-File HTML references and doc_specs.json
node tools/reference-builder/index.js

# Run doc-auditor (exits 0 when no discrepancies remain)
go run ./tools/doc-auditor
```
