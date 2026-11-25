# i-hate-decimal-calc / ihd

<p align="center">
  <img src="./assets/logo/ihd-logo.svg" alt="ihd (i-hate-decimal-calc) Logo" width="798" />
</p>

`ihd` is a command-line Computer Algebra System (CAS) calculator that excludes decimal approximations from input through evaluation. Every number and expression is computed strictly with arbitrary-precision rationals (`big.Rat`) and symbolic AST nodes, outputting exact results, derivation steps, and externally verifiable code (Lean 4 / Typst / certificate JSON).

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
- [Exam & Practical Problem Cookbook](#exam--practical-problem-cookbook)
- [LICENSE](#license)

## Why ihd? (Eliminating Decimals & When to Use It)

Standard programming languages and calculators rely on `float64` floating-point arithmetic, introducing rounding errors and truncating irrational or transcendental values into finite decimals.

Built around the principle of rejecting decimal approximations, `ihd` (i-hate-decimal-calc) converts decimal and repeating-decimal inputs into exact arbitrary-precision rational fractions (`big.Rat`) during lexing. Radicals, trigonometric functions, logarithms, transcendental constants, and imaginary units—as well as $`p`$-adic expansions, Puiseux series, closed rational intervals, and algebraic reals—remain first-class symbolic nodes throughout computation.

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
- **Full-width IME input & formal/typesetting projection**: Accepts full-width Japanese IME digits and operators directly in CLI/REPL, and projects derivations and plots directly to **Typst (`--typst`)**, **LaTeX (`--latex`)**, **SVG (`--format svg`)**, and **Lean 4 (`--lean`)**.
- **When other tools fit better**: General-purpose scripting workflows or large-scale floating-point numerical array simulations are better served by Python (SymPy / NumPy). `ihd` focuses specifically on standalone symbolic-rational calculation and externally checkable derivation artifacts.

<details>
<summary><b>Logo Trivia</b></summary>

- **Defying the Decimal Point**: In the top banner, the dot of the `i` in `ihd` shares the exact coordinates (`cx=125.2, cy=76.8`) with the decimal point `.` of `0.5477225575051662`, the approximation produced when the infinite irrational expansion `0.54772255750516611345...` of `sqrt(0.1 + 0.2)` is truncated at the 16th decimal place and rounded from `...1661` to `...1662`. That decimal point is struck out from below by a low-poly middle finger in the background layer, contrasted on the right with `ihd`'s exact rationalized result, `√30/10`.
- **Layout Coordinates Solved by `ihd` Itself**: Given the rendered bounding-box widths of the left typography (`164px`) and the right REPL output (`302px`), the four horizontal coordinates `[L_min, X_div, R_min, X_right]` that equalize both the outer frame margins (from left border `x=10`) and the inner margins around the central diagonal divider were computed directly using `ihd`'s own linear system solver `linsolve`:
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

### Closed-Domain First-Class AST Nodes, Certified Solvers & Exact Arithmetic
- **First-Class Closed Domains (`padic_expand` / `puiseux` / `interval` / `alg_real`)**: Truncated p-adic series, Puiseux series, closed rational intervals, and algebraic reals remain first-class nodes supporting closed arithmetic, integer powers, differentiation (Puiseux), and interval comparisons. Mixed-prime or symbolic-coefficient combinations remain symbolic or raise domain errors instead of approximating.
- **Rational SOS, Ideal Membership & Budgeted ECM Factorization**: Includes multivariate rational Sum-of-Squares decomposition (`sos_decompose`) with counterexample witnesses, Groebner-basis cofactor certificates (`ideal_member`), and multi-stage deterministic integer factorization (`factor`) combining trial division, perfect-power checks, Pollard-Brent rho, and Lenstra ECM Stage 1/2.
- **Physical Units, Formal Power Series & Discrete Math**: Combines exact dimensional analysis over the 7-dimensional rational vector space of SI base units (`convert_unit`) with formal power series (`fps_*`), finite-field Reed-Solomon codes (`rs_encode` / `rs_decode`), graph theory, discrete probability, and zero-sum game equilibria (`nash_zero_sum`).

### Visualization, Formal Projection & Safety
- **Derivation Trees & Single-DAG Formal Projection (`--explain` / `--typst` / `--verify` / `--lean`)**: Displays algebraic transformation trees across rationalization, factoring, limits, partial fractions (with automatic `NonZero` side conditions), CRT, and rational SOS, and projects from the same provenance DAG into Typst reports or Lean 4 formal proofs.
- **2D Typesetting & 2D/3D Braille/Vector Plotter (`--pretty` / `plot` / `plot3d`)**: Renders fractions, roots, and matrices as multi-line terminal box models, and outputs 2D curves (`plot` / `plot_svg`) as Unicode 2×4 Braille, SVG, and Typst (`--format svg` / `--typst` / `--out`). For 3D surfaces and space curves (`plot3d` / `plot3d_svg`), it generates hidden-line-removed wireframes and vector exports with viewpoint-adaptive 3D axes and exact CAS critical-point analysis.
- **REPL Assistance & Collection Control**: Zero-dependency FSM provides live signature guides and rainbow delimiters, alongside Python-style slicing (`[start:end:step]`), `range`, `lambda`, `map`, `filter`, and loop guards (`--warn-loop` / `--max-loop`).

Concrete CLI outputs, 2D/3D Braille and vector plots, derivation trees, closed-domain arithmetic, SOS, and Typst/Lean 4 artifacts are shown in **[Visual Samples (English)](./sample/sample.en.md)** / **[出力サンプル集（日本語）](./sample/sample.ja.md)**.

## Package & Repository Layout

Installed release bundles (`~/.ihd/`) and cloned source repositories contain different subsets of files:

| File / Directory | Release Archive Path | Installer Path (`~/.ihd/`) | Purpose |
| :--- | :--- | :--- | :--- |
| `ihd` / `micro-verifier` binaries | `./` (archive root) | `~/.ihd/bin/` | Main CAS binary and standalone certificate verifier |
| Reference manuals (MD / HTML) | `docs/references/` | `~/.ihd/references/` | Offline end-user syntax references |
| Live showcase & cookbook (`exam/`) | `exam/` | `~/.ihd/exam/` | Math benchmark suite and cookbook |
| Output samples (`sample/`) | `sample/` | `~/.ihd/sample/` | Concrete CLI, 2D/3D plot, and proof outputs |
| Verification helpers (`tools/`) | `tools/micro-verifier/`, `tools/sympy-oracle/` | Under `~/.ihd/tools/` | Standalone verifier source and SymPy differential oracle |
| Shell completion scripts | - | `~/.ihd/completions/` | Shell completion files generated during installation |
| `README.md`, `README.en.md`, `LICENSE.MIT`, `NOTICE.md` | `./` (archive root) | `~/.ihd/` root | Overview documentation and license notices |
| `docs/development.*.md`, `tools/e2e/`, `tools/repo-lint/`, `formal/`, `internal/` | Not included (repository only) | Not included (repository only) | Unit tests, E2E, repository linter, fuzzing, Lean 4 checks, and HTML builder |

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
| `-a`, `--approx` | Prints arbitrary-precision `big.Float` approximation and interval error bounds (`±`) beside the exact result (default 30 digits; `--digits <N>` and `--approx-digits <N>` accept positive integers, exit `1` on `<= 0`, and enable approximation even without `-a`).<br>`ihd -a "sqrt(2)"` |
| `--format <fmt>` | Sets output format (`typst`, `latex`, `ascii`, `pretty`, `svg`); individual flags `--typst`, `--latex`, `--ascii`, `--pretty` are also supported. `svg` applies only to plot expressions and exits `1` otherwise.<br>`ihd --format typst "1/2 + sqrt(2)"` |
| `--out <file>` | Saves one-shot formatted output (SVG / Typst / LaTeX / Explain) to the target path as UTF-8 without BOM. Combining with piped input, script execution, or `--step` exits `1`.<br>`ihd --format svg --out sin.svg "plot(sin(x), [-pi, pi])"` |
| `--explain` | Displays step-by-step algebraic derivation trees.<br>`ihd --explain "1 / (sqrt(2) + 1)"` |
| `--verify` | Reverse-verifies algebraic correctness by checking residual vanishing. Returns exit code `3` when a relational claim (`==`, etc.) is refuted.<br>`ihd --verify "integrate(1/(x^2 + 1), x)"` |
| `-i`, `--interval` | Evaluates expressions using adaptive rational interval enclosures (`--interval-eps` applies only with `-i`/`--interval`; non-positive or unparseable values fall back to default `1/1000000`).<br>`ihd -i "sin(1)"` |
| `--lean`, `--lean-file <path>` | Emits formal **Lean 4 (Mathlib4)** proof code to stdout or a `.lean` file.<br>`ihd --lean "factor(x^2 - 1)"` |
| `-h`, `--help` | Displays categorized, color-highlighted CLI help. |

<details>
<summary><strong>Expand Execution Control, Tracing & Certificate Options</strong></summary>

| Option | Description & Examples |
| :--- | :--- |
| `--step` | Interactively steps through algebraic derivation and proof stages one step at a time (cannot be combined with `--out`).<br>`ihd --step "expand((x+1)^3)"` |
| `--trace` | Emits internal solver search events (Buchberger, CAD, Risch) as structured JSON.<br>`ihd --trace "factor(x^2 - 1)"` |
| `--certificate <file>` | Saves calculation results and derivation steps as a JSON certificate (version `"1.0"`).<br>`ihd --certificate cert.json "factor(x^2 - 1)"` |
| `--replay <file>` | Reads a JSON certificate and verifies zero residuals without invoking CAS solvers.<br>`ihd --replay cert.json` |
| `--deg` | Evaluates trigonometric and inverse trigonometric functions in degrees (default: radians).<br>`ihd --deg "sin(30)"` |
| `--theme <dark\|light\|none>` | Temporarily sets the REPL color theme for the current session (default: `dark`; persist to `~/.ihd/config.json` via the REPL `theme <dark\|light\|none>` command).<br>`ihd --theme light` |
| `--lang <ja\|en>` | Sets display language for errors, derivations, and plot labels, persisting it to `~/.ihd/config.json` (defaults to saved config or OS locale).<br>`ihd --lang en "1/2 + 1/3"` |
| `--timeout <dur>` | Sets wall-clock timeout (default: unlimited).<br>`ihd --timeout 2s "factor(x^100 - 1)"` |
| `--budget-steps <N>` | Sets maximum rewrite steps per reduction pass (default: `0` = unlimited).<br>`ihd --budget-steps 5000 "expand((x+1)^20)"` |
| `--max-nodes <N>` | Sets maximum AST node count during expansion (default: `0` = unlimited).<br>`ihd --max-nodes 10000 "expand((x+1)^10)"` |
| `--max-depth <N>` | Sets maximum recursion depth for symbolic rewrites (default: `0` = unlimited).<br>`ihd --max-depth 200 "diff(x^10, x)"` |
| `--warn-loop <N>` | Sets the warning threshold for CAS loops (`table`, `for`, `product`, written to stderr; default: `100000`). |
| `--max-loop <N>` | Sets the hard cutoff limit for CAS loops (default: `-1` = unlimited). |
| `--deterministic` | Normalizes map iterations and polynomial ordering for byte-identical outputs.<br>`ihd --deterministic "factor(x^4 - 5*x^2 + 4)"` |
| `--no-color` | Disables ANSI color escape sequences (also disabled when `NO_COLOR` is set). |
| `--no-update-check` | Disables automatic release checks at startup (also disabled when `IHD_NO_UPDATE_CHECK=1`). |
| `-v`, `--version` | Displays version information. |

</details>

### Mathematical Domains & Reference Manuals

Built-in functions span elementary algebra and calculus through number theory, real algebraic geometry, local series analysis, discrete mathematics, and formal proof verification:

| Mathematical Domain | Representative Functions |
| :--- | :--- |
| **Basic Algebra, Number Theory, Equations & SOS** | `expand`, `factor`, `apart`, `together`, `gcd`, `lcm`, `mod`, `crt`, `is_prime`, `solve`, `cfrac`, `groebner`, `ideal_member`, `sos_decompose`, `sturm`, `isolate_roots_vas`, `alg_real`, `interval` |
| **Trigonometric, Logarithmic & Complex** | `sin`, `cos`, `tan`, `asin`, `acos`, `atan`, `trig_expand`, `trig_reduce`, `exp`, `log`, `ln`, `polar` |
| **Calculus, ODEs, Special Functions & Series Rings** | `diff`, `integrate`, `limit`, `taylor`, `puiseux`, `fps_inv`, `fps_exp`, `fps_log`, `fps_sqrt`, `dsolve`, `laplace`, `lambert_w`, `erf`, `gamma`, `sum` |
| **Linear Algebra, LP & Integer Normal Forms** | `det`, `inv`, `rank`, `rref`, `eigenvals`, `linsolve`, `lp_maximize`, `hnf`, `snf`, `lll` |
| **Computational Geometry, Constructibility & Proving** | `line_intersect`, `circle_intersect`, `triangle_area`, `geo_prove`, `is_constructible` |
| **Algebraic Structures, p-adic, Arithmetic Geometry & Quantum** | `padic_val`, `padic_norm`, `padic_expand`, `galois_group`, `is_solvable_by_radicals`, `ec_p_order`, `ec_rank_bound`, `clifford`, `quantum_equiv` |
| **Combinatorics, Error-Correcting Codes, Graphs, Probability & Games** | `partitions`, `bell`, `catalan`, `stirling1`, `stirling2`, `hamming_dist`, `rs_encode`, `rs_decode`, `graph_laplacian`, `spanning_trees`, `chromatic_poly`, `dice_prob`, `markov_stationary`, `nash_zero_sum`, `mex`, `nim_sum` |
| **Units, Assumptions & CAS Iteration** | `convert_unit`, `assume`, `unassume`, `table`, `range`, `map`, `filter` |
| **Visualization & 2D/3D Spatial Plotting** | `plot`, `plot_svg`, `plot3d`, `plot3d_svg` |

All function signatures, operator precedence rules, and CLI flags are documented in the syntax references and `ihd --help`:
- **Markdown Edition**: [Japanese Reference](docs/references/ja/reference.ja.md) / [English Reference](docs/references/en/reference.en.md)
- **Offline Single-File HTML Edition**: [Japanese Reference (HTML)](docs/references/ja/reference.ja.html) / [English Reference (HTML)](docs/references/en/reference.en.html) (installed to `~/.ihd/references/` via installers)

## Design Scope, Limitations & External Verification

In complex indefinite integration, higher-degree equations, limits, and symbolic simplification, algorithmic boundaries or implementation limits can produce incomplete reductions, dropped branches, or incorrect results. In addition, `factor` enforces deterministic work budgets on large composites to prevent hangs; when exhausted, it returns a partial factorization preserving discovered prime factors alongside any unfactored composite remainder.

This reflects a deliberate trade-off: because `ihd` ships as a single binary without bundling external CAS engines or language runtimes, it prioritizes independent external checkability over claiming universal coverage on its own.

- **Continuous CI checks**: GitHub Actions continuously runs repository linting, unit tests, the E2E test suite, reference audits, Lean 4 formal proofs, Typst compilation checks, SymPy differential and invariant fuzzing, and Bytecode VM verification (see the [Development Guide](docs/development.en.md#7-github-actions-ci-workflows) for the full workflow breakdown).
- **Local verification options**: For high-assurance research or engineering calculations, combine `--verify` (reverse residual checks), `--certificate` with `micro-verifier` (standalone zero-CAS-dependency replay), `--lean` (Lean 4 proof emission), and the bundled `tools/sympy-oracle`.

## Development & Testing

Core checks (unit tests, repository linting, and E2E tests) run with Go alone, and the Exam showcase runs with `pwsh` or `bash`:

```bash
# Run Go unit tests and repository linter
go test ./...
go run ./tools/repo-lint --all

# Run the Go-native E2E suite
go run ./tools/e2e

# Run the live math showcase (Windows / Linux & macOS)
pwsh exam/exam.ps1
./exam/exam.sh
```

Full contributor workflows are documented in **[Development Guide (English)](docs/development.en.md)** / **[開発・テスト・検証ガイド（日本語）](docs/development.ja.md)**. They cover enabling Git hooks (`.githooks/`), invariant fuzzing over $`\mathbb{Q}[i]`$, `tools/micro-verifier`, Lean 4 kernel checks (`formal/`), Bytecode VM benchmarks, `tools/sympy-oracle`, and reference auditing.

## Exam & Practical Problem Cookbook

Worked solutions and explanations for representative math problems (high school math, entrance exams, linear algebra, calculus, number theory, and geometry):
- [Japanese Edition Cookbook](./exam/problems.ja.md)
- [English Edition Cookbook](./exam/problems.en.md)

## LICENSE

[MIT](./LICENSE.MIT)

Third-Party → [NOTICE.md](./NOTICE.md)
