# i-hate-decimal-calc / ihd

<p align="center">
  <img src="./assets/logo/ihd-logo.svg" alt="ihd (i-hate-decimal-calc) Logo" width="798" />
</p>

A command-line Computer Algebra System (CAS) calculator that excludes decimal approximations throughout input and evaluation, computing strictly with arbitrary-precision rationals (`big.Rat`) and symbolic AST nodes to return exact results, derivation steps, and externally verifiable outputs (Lean 4 / Typst / certificate JSON).

![Go](https://img.shields.io/badge/Go-1.26.5-00ADD8?style=flat-square&logo=go&logoColor=white)
[![Lean 4 Verification](https://github.com/DovahkiinYuzuko/i-hate-decimal-calc/actions/workflows/lean-verify.yml/badge.svg)](https://github.com/DovahkiinYuzuko/i-hate-decimal-calc/actions/workflows/lean-verify.yml)
[![Typst Math Verification](https://github.com/DovahkiinYuzuko/i-hate-decimal-calc/actions/workflows/typst-verify.yml/badge.svg)](https://github.com/DovahkiinYuzuko/i-hate-decimal-calc/actions/workflows/typst-verify.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=flat-square&logo=opensourceinitiative&logoColor=white)](./LICENSE.MIT)

[日本語](README.md) | [English](README.en.md)

## Table of Contents

- [Why ihd? (Eliminating Decimals & When to Use It)](#why-ihd-eliminating-decimals--when-to-use-it)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Feature Highlights & Verification Ecosystem](#feature-highlights--verification-ecosystem)
- [Package & Repository Layout](#package--repository-layout)
- [Core Commands & Reference](#core-commands--reference)
- [Design Scope, Limitations & External Verification](#design-scope-limitations--external-verification)
- [Development & Testing](#development--testing)
- [Practical Problem Cookbook (Exam & Cookbook)](#practical-problem-cookbook-exam--cookbook)
- [LICENSE](#license)

## Why ihd? (Eliminating Decimals & When to Use It)

Standard programming languages and calculators rely on `float64` floating-point arithmetic, introducing rounding errors and truncating irrational or transcendental values into finite decimals.

`ihd` (i-hate-decimal-calc) is a command-line CAS calculator built around the design principle of rejecting decimal approximations. Decimal and repeating-decimal inputs are converted into exact arbitrary-precision rational fractions (`big.Rat`) during lexing, while radicals, trigonometric functions, logarithms, transcendental constants, imaginary units, $p$-adic expansions, Puiseux series, and closed rational intervals remain first-class symbolic AST nodes.

| Input Expression | Typical Calculators / Languages (`float64`) | `ihd` (Exact Calculation) | Simplification Behavior |
| :--- | :--- | :--- | :--- |
| `0.1 + 0.2` | `0.30000000000000004` | `3/10` | Zero-error exact rational fraction |
| `0.(3)` | `0.3333333333333333` | `1/3` | FSM lexical analysis converts repeating decimals to exact fractions |
| `sqrt(8)` | `2.8284271247461903` | `2*√2` | Automatic extraction of square factors |
| `sqrt(5 + 2*sqrt(6))` | `3.1462643699419726` | `√2 + √3` | Radical denesting via Borodin (1985) algorithm |
| `1 / (sqrt(2) + 1)` | `0.4142135623730951` | `-1 + √2` | Binomial conjugate radical rationalization |
| `sin(pi/6)` | `0.49999999999999994` | `1/2` | Exact algebraic evaluation of special angles |
| `１＋２×３` | Error (Unsupported) | `7` | Transparent normalization of full-width IME characters to ASCII |

### Where `ihd` Fits vs. Alternative Tools

- **Zero-dependency exact verification**: Deployable as a single static binary in CI pipelines, containers, or offline hosts where installing Python and SymPy is impractical or undesired, supporting symbolic reduction and algebraic equivalence diffing (`ihd diff`).
- **Full-width IME input & formal/typesetting projection**: Accepts full-width Japanese IME digits and operators directly in CLI/REPL, and projects derivation graphs directly to **Typst (`--typst`)**, **LaTeX (`--latex`)**, and **Lean 4 (`--lean`)**.
- **When other tools fit better**: General-purpose scripting workflows or large-scale floating-point numerical array simulations are better served by Python (SymPy / NumPy). `ihd` focuses specifically on standalone symbolic-rational calculation and externally checkable derivation artifacts.

<details>
<summary><b>Logo Trivia</b></summary>

- **Defying the Decimal Point**: In the top banner, the dot of the `i` in `ihd` shares the exact coordinates (`cx=125.2, cy=76.8`) with the decimal point `.` of `0.5477225575051662`—where the infinite irrational expansion `0.54772255750516611345...` of `sqrt(0.1 + 0.2)` is truncated at the 16th decimal place and rounded from `...1661` to `...1662`—and is struck out from below by a low-poly middle finger in the background layer. In stark contrast, the right side displays `ihd`'s exact rationalized result, `√30/10`.
- **Layout Coordinates Solved by `ihd` Itself**: Given the rendered bounding-box widths of the left typography (`164px`) and the right REPL output (`302px`), the four horizontal coordinates $[L_{\min}, X_{\text{div}}, R_{\min}, X_{\text{right}}]$ that equalize both the outer frame margins (from left border `x=10`) and the inner margins around the central diagonal divider were computed directly using `ihd`'s own linear system solver `linsolve`:
  ```text
  ihd> linsolve([[1, 0, 1, -1], [1, -2, 1, 0], [1, 0, 0, 0], [0, 1, 0, 0]], [-292, -164, 100, 330])
  [100, 330, 396, 788]
  ```
</details>

## Installation

### One-line Installer

**Linux / macOS (bash / zsh):**
```bash
curl -fsSL https://raw.githubusercontent.com/DovahkiinYuzuko/i-hate-decimal-calc/main/install.sh | bash
```

**Windows (PowerShell):**
```powershell
irm https://raw.githubusercontent.com/DovahkiinYuzuko/i-hate-decimal-calc/main/install.ps1 | iex
```

> [!NOTE]
> **Android (Termux) & Login Shell `PATH` Configuration**:  
> In environments such as Termux on Android, `~/.bash_profile` is evaluated at login instead of `~/.bashrc`. To persist `PATH`, append `export PATH="$HOME/.ihd/bin:$PATH"` to `~/.bash_profile` or source `~/.bashrc` inside it (`[ -f ~/.bashrc ] && . ~/.bashrc`).

### Building from Source

With Go 1.26.5 installed, build a standalone binary from the repository root:

```bash
go build -o ihd ./cmd/ihd
```

## Quick Start

Pass an expression as a CLI argument for immediate exact rational or symbolic evaluation:

```bash
# Exact rational arithmetic (zero rounding error)
ihd "0.1 + 0.2"
# Output: 3/10

# Radical denesting and algebraic equivalence
ihd "sqrt(2) + sqrt(3) == sqrt(5 + 2*sqrt(6))"
# Output: true

# Real algebraic inequality solving (CAD / Sturm root isolation)
ihd "solve(x^2 - 4 < 0)"
# Output: [[-2, 2]]
```

Four execution modes are available depending on the workflow:

<details>
<summary><strong>Interactive REPL, Piped Input, Script Files, and Full-width Normalization Examples</strong></summary>

#### 1. Interactive REPL
Running `ihd` without arguments starts the interactive REPL with history navigation, real-time argument hints, and syntax highlighting:

```text
$ ihd
ihd: Exact Arithmetic Calculator
Type 'exit' or 'quit' to exit.
ihd> 1/2 + 1/3
5/6
ihd> ans * 6
5
ihd> x = 1 + sqrt(2)
1 + √2
ihd> exit
Goodbye.
```

#### 2. Piped Input (Batch Integration)
Expressions read sequentially from standard input share variable bindings within the same session:

```bash
printf "x = 1/2 + sqrt(2)\nx * 2\n" | ihd
# Output:
# 1/2 + √2
# 1 + 2*√2
```

#### 3. Script Execution (`ihd run <file.ihd>`)
Runs multi-line `.ihd` files. Lines terminated with `;` evaluate and persist state while suppressing stdout output:

```bash
ihd run sample.ihd
# Direct file invocation is also accepted
ihd sample.ihd
```

#### 4. Full-width (Zenkaku) Normalization
Full-width digits, letters, operators (`＋`, `−`, `×`, `÷`, `＾`, `！`, `＝`), parentheses, and spaces are converted to half-width ASCII prior to lexing:

```bash
ihd "１＋２×３"
# Output: 7
```

</details>

## Feature Highlights & Verification Ecosystem

### Closed-Domain First-Class AST Nodes & Exact Arithmetic
- **First-Class Closed Domains (`PadicNode` / `PuiseuxNode` / `IntervalNode`)**: Truncated $p$-adic series (`padic_expand`), Puiseux series (`puiseux`), and closed rational intervals (`interval`) are represented as first-class AST nodes supporting closed arithmetic, integer powers, differentiation (Puiseux), and interval comparisons. Mixed-prime or symbolic-coefficient combinations remain symbolic or raise domain errors instead of approximating.
- **Budgeted Deterministic Integer Factorization (`factor`)**: Applies trial division, perfect-power detection, Pollard-Brent rho, and deterministic Lenstra Elliptic Curve Method (ECM Stage 1 / Stage 2) sequentially.
- **Rational Vector-Space Physical Units**: Performs exact dimensional analysis and conversions (`convert_unit`) over the 7-dimensional rational vector space $\mathbb{Q}^7$ of SI base units.

### Visualization, Formal Projection & Safety
- **Derivation Trees & Single-DAG Formal Projection (`--explain` / `--typst` / `--lean`)**: Displays algebraic transformation trees and projects from the same provenance DAG into Typst reports (`--explain --typst`) or Lean 4 transitive equational proofs (`--explain --lean`).
- **2D Typesetting & Braille Plotter (`--pretty` / `plot`)**: Renders fractions, roots, and matrices as multi-line terminal box models, and plots curves with Unicode 2×4 Braille characters.
- **REPL Assistance & Collection Control**: Zero-dependency FSM provides live signature guides and rainbow delimiters, alongside Python-style slicing (`[start:end:step]`), `range`, `lambda`, `map`, `filter`, and loop guards (`--warn-loop` / `--max-loop`).

Concrete CLI outputs, terminal plots, derivation trees, and Typst/Lean 4 artifacts are shown in **[Visual Samples (English)](./sample/sample.en.md)** / **[出力サンプル集（日本語）](./sample/sample.ja.md)**.

## Package & Repository Layout

Installed release bundles (`~/.ihd/`) and cloned source repositories contain different subsets of files:

| File / Directory | Release Archive Path | Installer Path (`~/.ihd/`) | Repository Only | Purpose |
| :--- | :--- | :--- | :--- | :--- |
| `ihd` / `micro-verifier` binaries | `./` (archive root) | `~/.ihd/bin/` | - | Main CAS binary and standalone certificate verifier |
| Reference manuals (MD / HTML / JSON) | `docs/references/` | `~/.ihd/references/` | - | Offline documentation and machine-audit specs |
| Live showcase & cookbook (`exam/`) | `exam/` | `~/.ihd/exam/` | - | 24-problem math benchmark suite and cookbook |
| Output samples (`sample/`) | `sample/` | `~/.ihd/sample/` | - | Concrete CLI, typesetting, and proof outputs |
| Verification helpers (`tools/`) | `tools/micro-verifier/`, `tools/sympy-oracle/` | Under `~/.ihd/tools/` | - | Standalone verifier source and SymPy differential oracle |
| `README.md`, `README.en.md`, `LICENSE.MIT`, `NOTICE.md` | `./` (archive root) | `~/.ihd/` root | - | Overview documentation and license notices |
| `docs/development.*.md`, `tools/`, `internal/` | Not included | Not included | Included | Unit tests, fuzzing, E2E suite, and HTML builder |

## Core Commands & Reference

### Subcommands

| Command | Description & Examples |
| :--- | :--- |
| `ihd diff <f1\|e1> <f2\|e2>` | Compares syntactic differences and algebraic equivalence (`EQUIVALENT` / `CHANGED`).<br>`ihd diff "x^2 - 1" "(x - 1)*(x + 1)"` |
| `ihd run <file.ihd>` | Executes multi-line `.ihd` calculation scripts.<br>`ihd run script.ihd` |
| `ihd completion <shell>` | Generates shell completion scripts (`bash`, `zsh`, `fish`, `powershell`).<br>`ihd completion powershell \| Out-String \| Invoke-Expression` |
| `ihd update` | Self-updates from GitHub Releases (supports `--check` / `-c` and `--force` / `-f`). |

### Major CLI Options

| Option | Description & Examples |
| :--- | :--- |
| `-a`, `--approx` | Prints arbitrary-precision `big.Float` approximation and interval error bounds (`±`) beside the exact result (default 30 digits; override via `--digits <N>`).<br>`ihd -a "sqrt(2)"` |
| `--format <fmt>` | Sets output format (`typst`, `latex`, `ascii`, `pretty`, `svg`); individual flags `--typst`, `--latex`, `--ascii`, `--pretty` are also supported.<br>`ihd --format typst "1/2 + sqrt(2)"` |
| `--explain` | Displays step-by-step algebraic derivation trees.<br>`ihd --explain "1 / (sqrt(2) + 1)"` |
| `--verify` | Reverse-verifies algebraic correctness by checking residual vanishing.<br>`ihd --verify "integrate(1/(x^2 + 1), x)"` |
| `-i`, `--interval` | Evaluates expressions using adaptive rational interval enclosures (tolerance configurable via `--interval-eps`).<br>`ihd -i "sin(1)"` |
| `--lean`, `--lean-file <path>` | Emits formal **Lean 4 (Mathlib4)** proof code to stdout or a `.lean` file.<br>`ihd --lean "factor(x^2 - 1)"` |
| `-h`, `--help` | Displays categorized, color-highlighted CLI help. |

<details>
<summary><strong>Expand Execution Control, Tracing & Certificate Options</strong></summary>

| Option | Description & Examples |
| :--- | :--- |
| `--step` | Prints intermediate rewrite rules and expressions step by step.<br>`ihd --step "expand((x+1)^3)"` |
| `--trace` | Emits structured rewrite and solver search logs.<br>`ihd --trace "factor(x^2 - 1)"` |
| `--certificate <file>` | Saves calculation results and derivation steps as a JSON certificate (version `"1.0"`).<br>`ihd --certificate cert.json "factor(x^2 - 1)"` |
| `--replay <file>` | Reads a JSON certificate and verifies zero residuals without invoking CAS solvers.<br>`ihd --replay cert.json` |
| `--deg` | Evaluates trigonometric and inverse trigonometric functions in degrees.<br>`ihd --deg "sin(30)"` |
| `--theme <dark\|light\|none>` | Sets the REPL color theme (persisted to `~/.ihd/config.json`).<br>`ihd --theme light` |
| `--lang <ja\|en\|auto>` | Sets display language.<br>`ihd --lang en "1/2 + 1/3"` |
| `--timeout <dur>` | Sets wall-clock timeout (default: unlimited).<br>`ihd --timeout 2s "factor(x^100 - 1)"` |
| `--budget-steps <N>` | Sets maximum rewrite steps per reduction pass (default: `0` = unlimited).<br>`ihd --budget-steps 5000 "expand((x+1)^20)"` |
| `--max-nodes <N>` | Sets maximum AST node count during expansion (default: `0` = unlimited).<br>`ihd --max-nodes 10000 "expand((x+1)^10)"` |
| `--max-depth <N>` | Sets maximum recursion depth for symbolic rewrites (default: `0` = unlimited).<br>`ihd --max-depth 200 "diff(x^10, x)"` |
| `--warn-loop <N>` | Warning threshold for CAS loops (`table`, `for`, `product`, written to stderr; default: `100000`). |
| `--max-loop <N>` | Hard cutoff limit for CAS loops (default: `-1` = unlimited). |
| `--deterministic` | Normalizes map iterations and polynomial ordering for byte-identical outputs.<br>`ihd --deterministic "factor(x^4 - 5*x^2 + 4)"` |
| `--no-color` | Disables ANSI color escape sequences. |
| `--no-update-check` | Disables automatic release checks at startup. |
| `-v`, `--version` | Displays version information. |

</details>

### Mathematical Domains & Reference Manuals

Built-in functions span elementary algebra and calculus through number theory, real algebraic geometry, local series analysis, and formal proof verification:

| Mathematical Domain | Representative Functions |
| :--- | :--- |
| **Basic Algebra, Number Theory & Equations** | `expand`, `factor`, `apart`, `together`, `gcd`, `lcm`, `mod`, `is_prime`, `solve`, `cfrac`, `groebner`, `sturm`, `interval` |
| **Trigonometric, Logarithmic & Complex** | `sin`, `cos`, `tan`, `asin`, `acos`, `atan`, `trig_expand`, `trig_reduce`, `exp`, `log`, `ln`, `polar` |
| **Calculus, ODEs, Special Functions & Local Series** | `diff`, `integrate`, `limit`, `taylor`, `puiseux`, `dsolve`, `laplace`, `lambert_w`, `erf`, `gamma`, `sum` |
| **Linear Algebra, LP & Integer Normal Forms** | `det`, `inv`, `rank`, `rref`, `eigenvals`, `linsolve`, `lp_maximize`, `hnf`, `snf`, `lll` |
| **Computational Geometry & Automated Proving** | `line_intersect`, `circle_intersect`, `triangle_area`, `geo_prove` |
| **Algebraic Structures, p-adic, Arithmetic Geometry & Quantum** | `padic_val`, `padic_norm`, `padic_expand`, `galois_group`, `is_solvable_by_radicals`, `ec_p_order`, `ec_rank_bound`, `clifford`, `quantum_equiv` |
| **Units, Assumptions & CAS Iteration** | `convert_unit`, `assume`, `unassume`, `table`, `range`, `map`, `filter` |

All function signatures, operator precedence rules, and CLI flags are documented in the syntax references and `ihd --help`:
- **Markdown Edition**: [Japanese Reference](docs/references/ja/reference.ja.md) / [English Reference](docs/references/en/reference.en.md)
- **Offline Single-File HTML Edition**: [Japanese Reference (HTML)](docs/references/ja/reference.ja.html) / [English Reference (HTML)](docs/references/en/reference.en.html) (installed to `~/.ihd/references/` via installers)

## Design Scope, Limitations & External Verification

- **No universal correctness guarantee across arbitrary inputs**: In complex indefinite integration, higher-degree equations, limits, and symbolic simplification, algorithmic boundaries or implementation limits can produce incomplete reductions, dropped branches, or incorrect results. In addition, `factor` enforces deterministic work budgets on large composites to prevent hangs; when exhausted, it returns a partial factorization preserving discovered prime factors alongside any unfactored composite remainder.
- **Trade-off for standalone binary distribution**: Because `ihd` avoids bundling external heavy runtimes or commercial CAS backends, it prioritizes independent external checkability over claiming universal coverage on its own.
- **Verification workflows**: GitHub Actions CI continuously runs unit tests (`unit-test.yml`), Lean 4 formal proofs (`lean-verify.yml`), Typst compilation (`typst-verify.yml`), and invariant fuzzing (`fuzz-smoke.yml`). For high-assurance research or engineering calculations, combine `--verify` (reverse residual checks), `--certificate` with `micro-verifier` (standalone zero-CAS-dependency replay), `--lean` (Lean 4 proof emission), or the bundled `tools/sympy-oracle`.

## Development & Testing

In a cloned repository, standard Go unit tests and the 24-problem showcase run directly:

```bash
# Run Go unit tests
go test ./...

# Run the 24-problem live showcase (Windows / Linux & macOS)
pwsh exam/exam.ps1
./exam/exam.sh
```

Procedures for invariant fuzzing (Schwartz-Zippel over $\mathbb{Q}[i]$), standalone verification (`tools/micro-verifier`), differential fuzzing (`tools/sympy-oracle`), and reference auditing are documented in **[Development Guide (English)](docs/development.en.md)** / **[開発・テスト・検証ガイド（日本語）](docs/development.ja.md)**.

## Practical Problem Cookbook (Exam & Cookbook)

Worked solutions and explanations for 24 representative problems (high school math, entrance exams, linear algebra, calculus, number theory, and geometry):
- [Japanese Edition Cookbook](./exam/problems.ja.md)
- [English Edition Cookbook](./exam/problems.en.md)

## LICENSE

[MIT](./LICENSE.MIT)

Third-Party → [NOTICE.md](./NOTICE.md)
