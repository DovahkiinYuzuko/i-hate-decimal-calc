# sympy-oracle (`tools/sympy-oracle`)

`ihd` の数学的不変条件検証および差分ファジング向け常駐 SymPy オラクル / Resident SymPy oracle for mathematical invariant verification and differential fuzzing with `ihd`

[![Python: >=3.10](https://img.shields.io/badge/python-%3E%3D3.10-blue.svg?style=flat-square&logo=python&logoColor=white)](https://www.python.org/)
[![SymPy](https://img.shields.io/badge/SymPy-%3E%3D1.12-brightgreen.svg?style=flat-square)](https://www.sympy.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=flat-square&logo=opensourceinitiative&logoColor=white)](../../LICENSE.MIT)

[日本語](#日本語) | [English](#english)

## 日本語

`tools/sympy-oracle` は、数式処理電卓 `ihd`（i-hate-decimal-calc）の内部簡約エンジン、代数的変形、微積分、および有理式演算の健全性を外部 CAS（Computer Algebra System）として第三者検証するための常駐型テストオラクルです。

標準入出力（`stdin` / `stdout`）を介した行指向 JSON（NDJSON: Newline Delimited JSON）プロトコルにより単一のワーカープロセスを常駐させることで、リクエストごとの Python インタプリタ起動や SymPy インポートのオーバーヘッドを排除しています。

### 主な機能とアーキテクチャ

1. **プロセス常駐型 IPC（NDJSON 通信）**:
   - 1プロセスを常駐させ、1行1リクエストの JSON を `stdin` に受け取り、1行1レスポンスの JSON を `stdout` へ書き出します。
   - ログや例外トレースバックはすべて `stderr` に隔離され、`stdout` のプロトコル同期破壊を防ぎます。
2. **安全な構造化 AST 構築**:
   - 任意の文字列に対する `eval()` や `sympify()` ではなく、構造化された JSON AST ノード（`Add`, `Mul`, `Pow`, `Diff`, `Rational`, `Symbol` 等）から SymPy のネイティブ構文木を決定論的に組み立てます（`sos_verify` で文字列を受ける場合を除く）。
3. **仮定（Assumptions）の忠実な伝播**:
   - 変数シンボルに対して `assumptions: {"real": true}` などのメタデータを明示的に伝播し、複素平面における非正則関数（`abs` 等）やブランチカットの不一致を制御・検証します。
4. **多層オラクル判定（3-Tier Verification）**:
   - **Tier 1 (代数的ゼロ判定)**: `subtractive_zero`（`equals(0)` および `simplify(lhs - rhs) == 0`）。
   - **Tier 2 (ガウス有理数体 $\mathbb{Q}[i]$ 厳密代入)**: `eval_points`（Schwartz-Zippel 補題に基づく多点代入一致）。
   - **Tier 3 (任意精度複素数値比較)**: `eval_numerical`（`mpmath` 80桁による適応的トレランス判定）。

### ディレクトリ構成

```text
tools/sympy-oracle/
├── sympy_oracle.py    # 常駐型 JSON-RPC NDJSON ワーカー本体
└── README.md          # 本仕様書
```

### 前提要件とインストール

Python 3.10 以上（`reconfigure` のチェック自体は Python 3.7+ 対応）、および `sympy`・`mpmath` パッケージが必要です。

```bash
# 依存ライブラリのインストール
python -m pip install --upgrade pip
pip install sympy mpmath
```

### プロトコル仕様

すべてのリクエストおよびレスポンスには、プロセス世代と対応関係を識別するための `id` と `generation`（整数）が含まれます。

| アクション (`action`) | 主な入力フィールド | 主なレスポンスフィールドと仕様 |
| :--- | :--- | :--- |
| `handshake` | なし | `sympy_version`, `mpmath_version`, `pid` を返却します。 |
| `subtractive_zero` | `lhs`, `rhs` (JSON AST) | `lhs - rhs == 0` を記号的に判定し、`verdict`（`equivalent` / `inequivalent` / `inconclusive` / `unknown`）, `reason`, `is_zero` (bool), `is_constant_nonzero` (bool), `simplified` (string) を返却します。 |
| `eval_points` | `expr` (JSON AST), `points` (`[{var_name: {"re_num", "re_den", "im_num", "im_den"}}]` 形式の辞書リスト) | 各サンプル点での厳密なガウス有理数評価結果リストを `evaluations` に返却します（極や未定義点は `undefined: true`）。 |
| `eval_numerical` | `lhs`, `rhs`, `points`, `atol`, `rtol` | 80 桁 `mpmath` で評価し、`all_agree`（`sample_count >= 5` かつ全有効点が許容誤差内）, `sample_count`, `max_residual`, `point_results` を返却します。 |
| `clear_cache` | なし | SymPy の内部キャッシュを消去し、`cleared: true` を返却します。 |
| `isolate_roots` | `poly` (低次からの整数係数配列) | `sp.Poly.real_roots()` で実根分離を行い、`roots`（有理根は `is_exact: true` と `num`/`den`、無理根は近似値）と `count` を返却します。 |
| `fps_expand` | `expr`, `var`, `order` | `x = 0` まわりのべき級数展開を実行し、各次の係数文字列配列 `coeffs` を返却します。 |
| `combinatorics` | `kind` (`partitions`, `bell`, `catalan`, `stirling1`, `stirling2`), `n`, `k` | 指定した組合せ論的数値の計算結果を `value`（文字列）として返却します。 |
| `canonical_str` | `expr` (JSON AST) | JSON AST から構築した SymPy 式の標準文字列表現を `canonical` に返却します。 |
| `sos_verify` | `poly`, `terms` (`[{"weight", "poly"}]`), `witness` (任意) | 多項式と二乗和項の残差ゼロ・重み正値性を検証し、`verified`, `all_positive_weights`, `residual`, `witness_negative` を返却します。 |

### 実行と動作確認

#### 1. 対話的テスト実行

ターミナルから直接スクリプトを起動し、JSON 行を入力して検証できます。

```bash
python tools/sympy-oracle/sympy_oracle.py
```

ハンドシェイクリクエスト（入力）:
```json
{"id": 1, "generation": 1, "action": "handshake"}
```

レスポンス（出力）:
```json
{"id": 1, "generation": 1, "status": "ok", "sympy_version": "1.13.3", "mpmath_version": "1.3.0", "pid": 12345}
```

代数的一致判定（入力例: $(x+1)^2 - (x^2+2x+1) \stackrel{?}{=} 0$）:
```json
{"id": 2, "generation": 1, "action": "subtractive_zero", "lhs": {"op": "Pow", "base": {"op": "Add", "args": [{"op": "Symbol", "name": "x"}, {"op": "Integer", "val": "1"}]}, "exp": {"op": "Integer", "val": "2"}}, "rhs": {"op": "Add", "args": [{"op": "Pow", "base": {"op": "Symbol", "name": "x"}, "exp": {"op": "Integer", "val": "2"}}, {"op": "Mul", "args": [{"op": "Integer", "val": "2"}, {"op": "Symbol", "name": "x"}]}, {"op": "Integer", "val": "1"}]}}
```

レスポンス:
```json
{"id": 2, "generation": 1, "status": "ok", "verdict": "equivalent", "reason": "diff_equals_zero", "is_zero": true, "is_constant_nonzero": false, "simplified": ""}
```

#### 2. Go 側テストスイートからの実行

`ihd` の単体テストおよびファジングハーネスは、自動的に本スクリプトを探索して起動します。

```bash
# SymPy オラクル連携単体テストの実行
go test -v ./internal/calc -run "^(TestSymPyClient_|TestDifferentialOracle_)"

# CI 環境など SymPy の存在を必須として検証する場合（PowerShell）
$env:IHD_REQUIRE_SYMPY_ORACLE="1"
go test -v ./internal/calc -run "^TestSymPyClient_"
```

### LICENSE
[MIT](../../LICENSE.MIT)

---

## English

`tools/sympy-oracle` is a resident test oracle used by the `ihd` (i-hate-decimal-calc) mathematical calculation engine to verify algebraic simplifications, derivatives, and rational operations against an authoritative third-party Computer Algebra System (CAS).

By keeping a single worker process resident and communicating via a line-delimited JSON (NDJSON) protocol over standard I/O (`stdin` / `stdout`), it eliminates per-request Python interpreter startup and SymPy import overhead.

### Features & Architecture

1. **Persistent IPC via NDJSON**:
   - Runs a single long-running process accepting newline-delimited JSON requests over `stdin` and writing one JSON response per line to `stdout`.
   - Warnings and tracebacks are routed to `stderr` to preserve `stdout` protocol synchronization.
2. **Safe Structured AST Construction**:
   - Reconstructs native SymPy expression trees deterministically from structured JSON AST nodes (`Add`, `Mul`, `Pow`, `Diff`, `Rational`, `Symbol`, etc.) rather than calling `eval()` or `sympify()` on arbitrary user strings (except optional string inputs in `sos_verify`).
3. **Assumption Metadata Propagation**:
   - Propagates explicit variable assumptions (e.g., `assumptions: {"real": true}`) to handle non-holomorphic operations (such as `Abs` in complex analysis) and branch cuts accurately.
4. **Multi-Tier Equivalence Verification**:
   - **Tier 1 (Algebraic Zero Reduction)**: `subtractive_zero` (`equals(0)` and `simplify(lhs - rhs) == 0`).
   - **Tier 2 (Gaussian Rational Evaluation)**: `eval_points` (multi-point exact evaluation over $\mathbb{Q}[i]$ via Schwartz-Zippel lemma).
   - **Tier 3 (Arbitrary-Precision Float Evaluation)**: `eval_numerical` (80-digit `mpmath` numerical evaluation with adaptive tolerance).

### Directory Structure

```text
tools/sympy-oracle/
├── sympy_oracle.py    # Persistent JSON-RPC NDJSON oracle worker
└── README.md          # Specification & usage guide
```

### Requirements & Setup

Requires Python 3.10+ (the internal `reconfigure` check targets 3.7+) and the `sympy` and `mpmath` packages:

```bash
python -m pip install --upgrade pip
pip install sympy mpmath
```

### Protocol Specification

Every request and response includes `id` and `generation` integers to match requests across process lifecycles.

| Action (`action`) | Primary Request Fields | Response Fields & Behavior |
| :--- | :--- | :--- |
| `handshake` | None | Returns `sympy_version`, `mpmath_version`, and `pid`. |
| `subtractive_zero` | `lhs`, `rhs` (JSON AST) | Symbolically tests `lhs - rhs == 0` and returns `verdict` (`equivalent` / `inequivalent` / `inconclusive` / `unknown`), `reason`, `is_zero` (bool), `is_constant_nonzero` (bool), and `simplified` (string). |
| `eval_points` | `expr` (JSON AST), `points` (list of dicts `[{var_name: {"re_num", "re_den", "im_num", "im_den"}}]`) | Evaluates the expression at each Gaussian rational point and returns the results list in `evaluations` (poles/undefined points yield `undefined: true`). |
| `eval_numerical` | `lhs`, `rhs`, `points`, `atol`, `rtol` | Evaluates at 80-digit `mpmath` precision and returns `all_agree` (requires `sample_count >= 5` and all valid points within tolerance), `sample_count`, `max_residual`, and `point_results`. |
| `clear_cache` | None | Clears SymPy internal caches and returns `cleared: true`. |
| `isolate_roots` | `poly` (integer coefficients from lowest degree) | Isolates real roots via `sp.Poly.real_roots()` and returns `roots` (`is_exact: true` with `num`/`den` for rationals, approximation otherwise) and `count`. |
| `fps_expand` | `expr`, `var`, `order` | Expands a formal power series around `var = 0` and returns coefficient strings in `coeffs`. |
| `combinatorics` | `kind` (`partitions`, `bell`, `catalan`, `stirling1`, `stirling2`), `n`, `k` | Evaluates the requested combinatorial sequence and returns `value` (string). |
| `canonical_str` | `expr` (JSON AST) | Reconstructs the SymPy expression and returns its canonical string representation in `canonical`. |
| `sos_verify` | `poly`, `terms` (`[{"weight", "poly"}]`), `witness` (optional) | Verifies SOS decomposition residual and positive weights, returning `verified`, `all_positive_weights`, `residual`, and `witness_negative`. |

### Running Tests & Verification

#### 1. Interactive Smoke Test

Launch the script directly from a terminal and feed NDJSON requests via `stdin`:

```bash
python tools/sympy-oracle/sympy_oracle.py
```

Handshake request (input):
```json
{"id": 1, "generation": 1, "action": "handshake"}
```

Response (output):
```json
{"id": 1, "generation": 1, "status": "ok", "sympy_version": "1.13.3", "mpmath_version": "1.3.0", "pid": 12345}
```

Algebraic equivalence check (example: $(x+1)^2 - (x^2+2x+1) \stackrel{?}{=} 0$):
```json
{"id": 2, "generation": 1, "action": "subtractive_zero", "lhs": {"op": "Pow", "base": {"op": "Add", "args": [{"op": "Symbol", "name": "x"}, {"op": "Integer", "val": "1"}]}, "exp": {"op": "Integer", "val": "2"}}, "rhs": {"op": "Add", "args": [{"op": "Pow", "base": {"op": "Symbol", "name": "x"}, "exp": {"op": "Integer", "val": "2"}}, {"op": "Mul", "args": [{"op": "Integer", "val": "2"}, {"op": "Symbol", "name": "x"}]}, {"op": "Integer", "val": "1"}]}}
```

Response:
```json
{"id": 2, "generation": 1, "status": "ok", "verdict": "equivalent", "reason": "diff_equals_zero", "is_zero": true, "is_constant_nonzero": false, "simplified": ""}
```

#### 2. Running via Go Test Suite

`ihd` unit tests and fuzz harnesses automatically locate and launch this script:

```bash
# Run SymPy oracle client integration tests
go test -v ./internal/calc -run "^(TestSymPyClient_|TestDifferentialOracle_)"

# Require SymPy availability (e.g., in CI environments, PowerShell)
$env:IHD_REQUIRE_SYMPY_ORACLE="1"
go test -v ./internal/calc -run "^TestSymPyClient_"
```

### LICENSE
[MIT](../../LICENSE.MIT)
