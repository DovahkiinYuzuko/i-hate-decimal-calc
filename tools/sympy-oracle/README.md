# sympy-oracle (`tools/sympy-oracle`)

`ihd` の数学的不変条件検証および差分ファジング向け常駐 SymPy オラクル / Resident SymPy oracle for mathematical invariant verification and differential fuzzing with `ihd`

[![Python: >=3.10](https://img.shields.io/badge/python-%3E%3D3.10-blue.svg?style=flat-square&logo=python&logoColor=white)](https://www.python.org/)
[![SymPy](https://img.shields.io/badge/SymPy-%3E%3D1.12-brightgreen.svg?style=flat-square)](https://www.sympy.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=flat-square&logo=opensourceinitiative&logoColor=white)](../../LICENSE.MIT)

[日本語](#日本語) | [English](#english)

---

## 日本語

`tools/sympy-oracle` は、数式処理電卓 `ihd`（i-hate-decimal-calc）の内部簡約エンジン、代数的変形、微積分、および有理式演算の健全性を外部CAS（Computer Algebra System）として第三者検証するための常駐型テストオラクルです。

プロセス起動コスト（Python インタプリタおよび SymPy インポートのオーバーヘッド）を排除するため、標準入出力（`stdin` / `stdout`）を介した行指向 JSON（NDJSON: Newline Delimited JSON）プロトコルにより常駐ワーカーとして高速に双方向通信を行います。

### 主な機能とアーキテクチャ

1. **プロセス常駐型 IPC（NDJSON 通信）**:
   - 1プロセスを立ち上げ続け、1行1リクエストの JSON を `stdin` に流し込み、1行1レスポンスの JSON を `stdout` から即座に受け取る高スループット設計です。
   - ログや例外トレースバックはすべて `stderr` に隔離され、`stdout` のプロトコル同期破壊を防ぎます。
2. **安全な構造化 AST 構築**:
   - 任意の文字列に対する `eval()` や `sympify()` を一切行わず、構造化された JSON AST ノード（`Add`, `Mul`, `Pow`, `Diff`, `Rational`, `Symbol` 等）から SymPy のネイティブ構文木を決定論的に組み立てます。
3. **仮定（Assumptions）の忠実な伝播**:
   - 変数シンボルに対して `assumptions: {"real": true}` などのメタデータを明示的に伝播可能。複素平面における非正則関数（`abs` 等）やブランチカットの不一致を正確に制御・検証します。
4. **多層オラクル判定（3-Tier Verification）**:
   - **Tier 1 (代数的ゼロ判定)**: `subtractive_zero`（`equals(0)` および `simplify(lhs - rhs) == 0`）。
   - **Tier 2 (ガウス有理数体 $\mathbb{Q}[i]$ 厳密代入)**: `eval_points`（Schwartz-Zippel 補題に基づく多点点代入一致）。
   - **Tier 3 (任意精度複素数値比較)**: `eval_numerical`（`mpmath` 80桁による適応的トレランス判定）。

### ディレクトリ構成

```text
tools/sympy-oracle/
├── sympy_oracle.py    # 常駐型 JSON-RPC NDJSON ワーカー本体
└── README.md          # 本仕様書
```

### 前提要件とインストール

Python 3.10 以上、および `sympy` パッケージが必要です。

```bash
# 依存ライブラリのインストール
python -m pip install --upgrade pip
pip install sympy mpmath
```

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
{"id": 2, "generation": 1, "status": "ok", "is_zero": true, "is_constant_nonzero": false, "simplified": ""}
```

#### 2. Go 側テストスイートからの実行

`ihd` の単体テストおよびファジングハーネスは、自動的に本スクリプトを探索して起動します。

```bash
# SymPy オラクル連携単体テストの実行
go test -v ./internal/calc -run "^(TestSymPyClient_|TestDifferentialOracle_)"

# CI 環境など SymPy の存在を必須として検証する場合
$env:IHD_REQUIRE_SYMPY_ORACLE="1"
go test -v ./internal/calc -run "^TestSymPyClient_"
```

---

## English

`tools/sympy-oracle` is a resident test oracle used by the `ihd` (i-hate-decimal-calc) mathematical calculation engine to verify algebraic simplifications, derivatives, and rational operations against an authoritative third-party Computer Algebra System (CAS).

To eliminate process spawn overhead (Python startup and SymPy import latency), it runs as a persistent background worker communicating via a high-throughput line-delimited JSON (NDJSON) protocol over standard I/O (`stdin` / `stdout`).

### Features & Architecture

1. **Persistent IPC via NDJSON**:
   - Single long-running process accepting newline-delimited JSON requests over `stdin` and returning atomic JSON responses over `stdout`.
   - Warnings and tracebacks are strictly routed to `stderr` to preserve protocol synchronization.
2. **Safe Structured AST Construction**:
   - Rejects arbitrary string `eval()` or `sympify()`. Instead, native SymPy expression trees are reconstructed deterministically from structured JSON AST nodes (`Add`, `Mul`, `Pow`, `Diff`, `Rational`, `Symbol`, etc.).
3. **Assumption Metadata Propagation**:
   - Supports explicit variable assumptions (e.g., `assumptions: {"real": true}`). This ensures proper handling of non-holomorphic operations (such as `Abs` in complex analysis) and branch cuts.
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

Requires Python 3.10+ and the `sympy` package:

```bash
python -m pip install --upgrade pip
pip install sympy mpmath
```

### Protocol Specification

Every request and response includes an `id` and `generation` integer to ensure request/response matching across process lifecycles.

#### 1. Handshake (`action: "handshake"`)
- Request: `{"id": 1, "generation": 1, "action": "handshake"}`
- Response: `{"id": 1, "generation": 1, "status": "ok", "sympy_version": "1.13.3", "mpmath_version": "1.3.0", "pid": 12345}`

#### 2. Subtractive Zero Check (`action: "subtractive_zero"`)
- Tests whether `lhs - rhs == 0` symbolically.
- Fields: `is_zero` (bool), `is_constant_nonzero` (bool), `simplified` (string).

#### 3. Exact Point Evaluation (`action: "eval_points"`)
- Evaluates expression at Gaussian rational points: `{"re_num": "...", "re_den": "...", "im_num": "...", "im_den": "..."}`.
- Response: Exact rational numerator/denominator components for each sample point.

#### 4. Numerical Evaluation (`action: "eval_numerical"`)
- Dual-tolerance 80-digit numerical evaluation (`atol`, `rtol`).
- Returns: `all_agree` (bool), `max_residual` (float), `point_results` (array).

### Running Tests

```bash
# Run SymPy oracle client integration tests
go test -v ./internal/calc -run "^(TestSymPyClient_|TestDifferentialOracle_)"
```
