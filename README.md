# i-hate-decimal-calc / ihd

小数を一切許さず、すべての入力を即座に分数化し、平方根や定数をシンボルのまま厳密計算する電卓CLIツール / A strict CAS calculator CLI that completely rejects decimal approximations, converting all decimals to exact fractions and maintaining exact algebraic forms.

![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go&logoColor=white)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=flat-square&logo=opensourceinitiative&logoColor=white)](./LICENSE.MIT)

[日本語](#日本語) | [English](#english)

## 日本語

### 概要
`ihd`（i-hate-decimal-calc）は、浮動小数点数による丸め誤差や小数を一切排除した、コマンドライン向けの厳密計算機（CAS: 数式処理システム）です。
入力されたすべての小数は内部で即座に有理数（`big.Rat`）へ変換され、平方根・三角関数・対数・超越数・虚数単位はシンボルノードとして保持されます。

### 特徴
- **完全厳密計算**: 内部計算に `float64` などの浮動小数点型を使用せず、有理数演算および代数的簡約のみで計算を実行します。
- **自動数式簡約**:
  - 平方数のくくり出し（例: `sqrt(8)` → `2*√2`）
  - 単項ルートの自動有理化（例: `1/sqrt(2)` → `√2/2`）
  - 三角関数の特殊値評価（例: `sin(pi/6)` → `1/2`）
  - 複素数への自動昇格と代数計算（例: `sqrt(-4)` → `2*i`, `(1+2*i)*(1-2*i)` → `5`）
  - 同類項の集約と分配法則による展開
- **厳格な構文解析**: 乗算記号の省略（例: `2pi` や `(1+2)(3+4)`）を禁止し、誤認識による計算ミスを防ぎます。
- **多彩な実行形態**: コマンドライン引数によるワンショット実行、対話型REPL、および標準入力パイプに対応しています。
- **表示モード切替**: 人間が読みやすいUnicode記号表示を標準としつつ、スクリプト連携用の `--ascii` フラグや、参考値としての `--approx`（小数近似値併記）フラグを備えています。

### インストール

#### ワンライナーインストール（推奨）

**Linux / macOS (bash / zsh):**
```bash
curl -fsSL https://raw.githubusercontent.com/DovahkiinYuzuko/i-hate-decimal-calc/main/install.sh | bash
```

**Windows (PowerShell):**
```powershell
irm https://raw.githubusercontent.com/DovahkiinYuzuko/i-hate-decimal-calc/main/install.ps1 | iex
```

#### ソースコードからのビルド
Go 1.22 以上がインストールされている環境で、リポジトリのルートディレクトリにて以下を実行してください。

```bash
go build -o ihd .
```

### 使い方

#### 1. ワンショット計算
コマンドライン引数に数式を指定して実行します。

```bash
ihd "1/2 + 1/3"
# 出力: 5/6

ihd "sqrt(2) + sqrt(8)"
# 出力: 3*√2
```

#### 2. 対話モード（REPL）
引数を指定せずに実行すると、対話型REPLが起動します。

```text
$ ihd
ihd: Exact Arithmetic Calculator
Type 'exit' or 'quit' to exit.
ihd> 4 + 4 * 6.441
7441/250
ihd> (1 + sqrt(2)) * (1 - sqrt(2))
-1
ihd> exit
Goodbye.
```

#### 3. パイプ入力
他のコマンドからの出力をパイプ経由で一括計算します。空行および `#` で始まるコメント行は自動的に無視されます。

```bash
echo "sin(pi/6)^2 + cos(pi/6)^2" | ihd
# 出力: 1
```

#### 4. コマンドラインオプション
- `--ascii`: Unicode記号（`√`, `π`）の代わりにASCII文字列（`sqrt`, `pi`）で出力します。
  ```bash
  ihd --ascii "sqrt(2) + pi"
  # 出力: sqrt(2) + pi
  ```
- `--approx`: 厳密解の横に参考用の浮動小数点小数近似値を併記します。
  ```bash
  ihd --approx "sqrt(2)"
  # 出力: √2 (≈ 1.4142135623730951)
  ```
- `-h`, `--help`: コマンドの使用方法を表示します。

### サポート構文と演算子

#### 演算子（優先順位順）
1. `()`: 括弧
2. `!`: 階乗（後置単項演算子、非負整数のみ対応）
3. `^`: べき乗（**右結合**: `2^3^2 = 2^(3^2) = 512`）
4. 単項マイナス `-`（`^` より低優先: `-3^2 = -(3^2) = -9`）
5. `*`, `/`: 乗算・除算（乗算記号 `*` の省略は不可）
6. `+`, `-`: 加算・減算

#### 定数
- `pi` または `π`: 円周率
- `e`: 自然対数の底
- `i`: 虚数単位（$i^2 = -1$）

#### 関数
- `sqrt(x)` または `√(x)`: 平方根（中身が負の場合は複素数へ自動昇格）
- `sin(x)`, `cos(x)`, `tan(x)`: 三角関数（$\pi$ の有理数倍による特殊角を代数的に簡約）
- `log(x)`: 常用対数（底10）
- `log(base, x)`: 任意の底を指定する対数
- `ln(x)`: 自然対数（底 $e$）

### LICENSE
[MIT](./LICENSE.MIT)

---

## English

### Overview
`ihd` (i-hate-decimal-calc) is a command-line Computer Algebra System (CAS) calculator that strictly eliminates all floating-point rounding errors and decimal representations.
All decimal inputs are converted immediately into exact rational fractions (`big.Rat`), while roots, trigonometric functions, logarithms, transcendental constants, and imaginary units are preserved as exact symbolic nodes.

### Features
- **Completely Exact Calculation**: Does not utilize `float64` or floating-point types for internal simplification, performing all operations via rational arithmetic and algebraic rules.
- **Automated Mathematical Simplification**:
  - Factoring out square components (e.g., `sqrt(8)` → `2*√2`)
  - Monomial root rationalization (e.g., `1/sqrt(2)` → `√2/2`)
  - Evaluation of trigonometric special values (e.g., `sin(pi/6)` → `1/2`)
  - Promotion to complex numbers and algebraic operations (e.g., `sqrt(-4)` → `2*i`, `(1+2*i)*(1-2*i)` → `5`)
  - Like-term aggregation and distributive expansion
- **Strict Parsing Rules**: Prohibits implicit multiplication (e.g., `2pi` or `(1+2)(3+4)`) to eliminate parse ambiguities.
- **Multiple Execution Modes**: Supports one-shot execution via CLI arguments, interactive REPL, and standard input piping.
- **Configurable Output**: Defaults to Unicode mathematical symbols, with `--ascii` for integration scripts and `--approx` for displaying reference floating-point approximations.

### Installation

#### One-line Installer (Recommended)

**Linux / macOS (bash / zsh):**
```bash
curl -fsSL https://raw.githubusercontent.com/DovahkiinYuzuko/i-hate-decimal-calc/main/install.sh | bash
```

**Windows (PowerShell):**
```powershell
irm https://raw.githubusercontent.com/DovahkiinYuzuko/i-hate-decimal-calc/main/install.ps1 | iex
```

#### Building from Source
Requires Go 1.22 or higher. Build directly from the root of the repository:

```bash
go build -o ihd .
```

### Usage

#### 1. One-shot Calculation
Provide mathematical expressions directly as command-line arguments:

```bash
ihd "1/2 + 1/3"
# Output: 5/6

ihd "sqrt(2) + sqrt(8)"
# Output: 3*√2
```

#### 2. Interactive REPL
Running without arguments launches the interactive REPL:

```text
$ ihd
ihd: Exact Arithmetic Calculator
Type 'exit' or 'quit' to exit.
ihd> 4 + 4 * 6.441
7441/250
ihd> (1 + sqrt(2)) * (1 - sqrt(2))
-1
ihd> exit
Goodbye.
```

#### 3. Piped Input
Process expressions sequentially from standard input. Empty lines and lines starting with `#` are ignored:

```bash
echo "sin(pi/6)^2 + cos(pi/6)^2" | ihd
# Output: 1
```

#### 4. Command-Line Options
- `--ascii`: Output using ASCII characters (`sqrt`, `pi`) instead of Unicode symbols.
  ```bash
  ihd --ascii "sqrt(2) + pi"
  # Output: sqrt(2) + pi
  ```
- `--approx`: Display an approximate decimal value alongside the exact form.
  ```bash
  ihd --approx "sqrt(2)"
  # Output: √2 (≈ 1.4142135623730951)
  ```
- `-h`, `--help`: Display the help message.

### Supported Syntax & Operators

#### Operators (in order of precedence)
1. `()`: Parentheses
2. `!`: Factorial (postfix unary operator, non-negative integers only)
3. `^`: Exponentiation (**Right-associative**: `2^3^2 = 2^(3^2) = 512`)
4. Unary minus `-` (Lower precedence than `^`: `-3^2 = -(3^2) = -9`)
5. `*`, `/`: Multiplication and division (explicit `*` required)
6. `+`, `-`: Addition and subtraction

#### Constants
- `pi` or `π`: The ratio of a circle's circumference to its diameter
- `e`: Euler's number (base of the natural logarithm)
- `i`: The imaginary unit ($i^2 = -1$)

#### Functions
- `sqrt(x)` or `√(x)`: Square root (promotes to complex numbers if radicand is negative)
- `sin(x)`, `cos(x)`, `tan(x)`: Trigonometric functions (exact values for rational multiples of $\pi$)
- `log(x)`: Common logarithm (base 10)
- `log(base, x)`: Logarithm with an arbitrary base
- `ln(x)`: Natural logarithm (base $e$)

### LICENSE
[MIT](./LICENSE.MIT)
