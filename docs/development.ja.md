# ihd 開発・テスト・検証ガイド

`ihd` リポジトリをクローンした開発者・コントリビュータ向けのテスト・検証手順書です。単体テスト、E2E テスト、統合 Linter、数学的不変条件ファジング、独立証明検証、Lean 4 カーネル検証、およびリファレンス監査の実行手順を収録しています。リリースアーカイブやインストーラ（`~/.ihd/references/`）に同梱される利用向けリファレンス（[reference.ja.md](references/ja/reference.ja.md)）とは役割が異なり、Git 追跡済みの公開ツールとテスト基盤のみを対象としています。

[日本語](development.ja.md) | [English](development.en.md) | [README に戻る](../README.md)

| 区分 | ツール・バージョン | 用途 |
| :--- | :--- | :--- |
| 必須 | Go 1.26.5 | 本体ビルド、単体テスト、E2E ランナー、統合 Linter、`micro-verifier`、`doc-auditor` |
| 任意 | PowerShell (`pwsh`) または `bash` | 実演ショウケース（Exam）の実行 |
| 任意 | Python 3 + SymPy | 常駐差分オラクル（`tools/sympy-oracle`）および Lean 証明カタログ集計 |
| 任意 | Node.js（または Bun） | オフライン Single-File HTML リファレンスと `doc_specs.json` の再生成 |
| 任意 | Lean 4 (`leanprover/lean4:v4.36.0-rc2`) | `formal/` 配下の Mathlib4 形式証明検証と公理監査 |

基本チェック（単体テスト・統合 Linter・E2E テスト）は Go 環境のみで完結します。

```bash
go test ./...
go run ./tools/repo-lint --all
go run ./tools/e2e
```

## 目次

1. [Git フックの有効化 (`.githooks/`)](#1-git-フックの有効化-githooks)
2. [単体テストと Go ネイティブ E2E テスト (`tools/e2e`)](#2-単体テストと-go-ネイティブ-e2e-テスト-toolse2e)
3. [統合 Linter (`tools/repo-lint`)](#3-統合-linter-toolsrepo-lint)
4. [数学的不変条件 Fuzzing・SymPy 差分オラクル・Bytecode VM 検証](#4-数学的不変条件-fuzzingsympy-差分オラクルbytecode-vm-検証)
5. [独立証明検証器 (`tools/micro-verifier`) と Lean 4 形式検証 (`formal/`)](#5-独立証明検証器-toolsmicro-verifier-と-lean-4-形式検証-formal)
6. [実演ショウケース (Exam) とリファレンス HTML ビルド・監査](#6-実演ショウケース-exam-とリファレンス-html-ビルド監査)
7. [GitHub Actions CI ワークフロー一覧](#7-github-actions-ci-ワークフロー一覧)

---

## 1. Git フックの有効化 (`.githooks/`)

リポジトリには共有用の Git フック（`.githooks/pre-commit`, `.githooks/pre-push`）が含まれています。クローン直後は Git のフックパスを切り替えることで有効になります。

```bash
git config core.hooksPath .githooks
```

- **`.githooks/pre-commit`**: `docs/references/` 配下の Markdown がステージされている場合、`node tools/reference-builder/index.js` で HTML を再生成し、`tools/reference-builder/package.json` のバージョンを最新 Git タグと同期します。続いて `go run ./tools/repo-lint --staged` を実行し、違反があれば終了コード `1` でコミットを中断します。
- **`.githooks/pre-push`**: PowerShell が存在する環境ではローカル専用の差分 Fuzz スモークスクリプトの実行を試みます（PowerShell 不在時のみ警告を出して終了コード `0` で通過します）。当該スクリプトが未配置のクローン環境では push が失敗します。

---

## 2. 単体テストと Go ネイティブ E2E テスト (`tools/e2e`)

### Go 単体テスト

全公開パッケージ（`cmd/ihd`, `internal/...`, `tools/...`）の単体テストを実行します。起動時の更新確認を省いて再現性を揃える場合、`IHD_NO_UPDATE_CHECK=1` を指定します。

```bash
IHD_NO_UPDATE_CHECK=1 go test -v -timeout 5m ./...
```

### Go ネイティブ E2E テストランナー (`tools/e2e`)

一時バイナリ（`ihd` 本体と `micro-verifier`）を自動ビルドし、[tools/e2e/test-cases/](../tools/e2e/test-cases) 配下の CSV テストケース群を並列実行します。失敗したケースがある場合は `tools/e2e/logs/e2e_failures_<timestamp>.log` に詳細を出力して終了コード `1` で終了します（フィルタ一致 0 件時は警告を出して終了コード `0` で終了します）。

```bash
# 全 E2E ケースを実行
go run ./tools/e2e

# カテゴリ・フィルタ・並列数を絞って実行
go run ./tools/e2e -c plotter -v
go run ./tools/e2e -f "SOS" -j 4
```

| フラグ | 説明 | 既定値 |
| :--- | :--- | :--- |
| `-c`, `--category <str>` | カテゴリ名またはファイルパスの部分一致で絞り込み（大文字・小文字を区別） | 全カテゴリ |
| `-f`, `--filter <str>` | ケース ID・説明の部分一致で絞り込み（大文字・小文字を区別） | 制限なし |
| `-j`, `--jobs <int>` | 並列ワーカー数（`0` 以下または省略時は CPU コア数） | CPU コア数 |
| `-v`, `--verbose` | 個別ケースの実行ログを表示 | `false` |

CSV テストケースは 10 列ヘッダー（`id,category,flags,pipe_input,match_mode,expected_stdout,expected_stderr,expected_exit_code,deduction_method,description`）で管理されています。統合 Linter は、`match_mode`（`exact` / `contains` / `regex` / `error`）、`deduction_method`（空または `deductive` / `contradiction` / `contrapositive` / `inductive` / `metamorphic` / `roundtrip`）、および全 CSV 横断の `id` 一意性を検証します。

---

## 3. 統合 Linter (`tools/repo-lint`)

リポジトリ固有の規約、i18n 整合性、ライセンス表記、Lean ピン留め、Go 実装規約、および E2E CSV スキーマを一括検査するツールです。

```bash
# リポジトリ全体を検査（既定）
go run ./tools/repo-lint --all

# ステージ済みファイルのみ検査（pre-commit 相当）
go run ./tools/repo-lint --staged

# ルール一覧の確認・個別実行・JSON 出力
go run ./tools/repo-lint --list-rules
go run ./tools/repo-lint --only i18n-ast,e2e-csv --format json
```

終了コードは、問題なしが `0`、エラー検出（または `--warn-as-error` 指定時の警告検出）が `1`、未定義フラグやフラグ構文エラーまたは内部実行・JSON シリアライズ失敗時は `2` です。

| フラグ | 説明 | 既定値・優先規則 |
| :--- | :--- | :--- |
| `--all` | リポジトリ全体を検査 | 既定（`--staged` 同時指定時は `--all` 優先） |
| `--staged` | Git ステージ済みファイルのみ検査 | `false` |
| `--only <csv>` | 実行するチェッカー／アナライザをカンマ区切りで限定 | 全ルール |
| `--skip <csv>` | 除外するチェッカー／アナライザをカンマ区切りで指定 | なし |
| `--root <dir>` | 検査対象のリポジトリルートパスを指定 | `.` |
| `--format <text\|json>` | 診断結果の出力形式を指定（`json` 以外はテキスト出力） | `text` |
| `--warn-as-error` | 警告を終了コード `1` として扱う | `false` |
| `--list-rules` | 登録済み全チェッカー・AST アナライザ一覧を表示して終了 | `false` |

| チェッカー / アナライザ | 検査内容と主な制約 |
| :--- | :--- |
| `readme-quickstart` | `--staged` 時のみ発動。`README.md` / `README.en.md` のクイックスタート節への変更を検知してブロック（承認済み変更時は環境変数 `IHD_ALLOW_QUICKSTART_EDIT=1` で解除） |
| `git-whitelist` | 禁止アーティファクト（`.env`, `.lake/`, ローカル作業ディレクトリ等）の混入と、公開許可リスト外 `.md` ファイルの追跡を禁止 |
| `i18n-ast` | `internal/i18n/locales/{ja,en}.json` のキー一致・`%w` 禁止・AST 上の `i18n.T` 呼び出しキーとフォーマット引数数の一致を検査 |
| `notice-sync` | `go.mod` および `tools/reference-builder/package.json` の依存が `NOTICE.md` に網羅されているか検査 |
| `lean-pinning` | `formal/lean-toolchain` のバージョン固定、`formal/lake-manifest.json` の 40 桁 SHA 固定、および CI での `lake update` 禁止を検査 |
| `go-hygiene` | 空の `if err != nil {}`、文字列リテラルおよびロケール JSON 内の絵文字（数学記号・罫線・点字を除く）、理由なき `_ = err`、文字列連結パス構築を検出 |
| `e2e-csv` | `tools/e2e/test-cases/**/*.csv` の 10 列スキーマ、列挙値、必須列、および `id` 一意性を検査 |

正当な理由で特定の `go-hygiene` 指摘を抑制する場合は、理由文を添えた `// repo-lint:ignore <rule-id|all> <concrete reason>` を対象行の直前または行末に記述します（理由文が欠けている場合はエラーになります）。

---

## 4. 数学的不変条件 Fuzzing・SymPy 差分オラクル・Bytecode VM 検証

### Property-Based Testing & Fuzzing

ガウス有理数体 $`\mathbb{Q}[i]`$ 上の多点評価オラクル（Schwartz-Zippel 補題）および代数・微積分の不変条件を用いたファジングハーネスを実行します。主なターゲットは [internal/calc/](../internal/calc) 配下の `fuzz_*_test.go` に定義されています。

- `FuzzAlgebra_SimplifyIdempotence`: 簡約べき等性
- `FuzzAlgebra_FactorExpandRoundTrip`: 因数分解と展開の往復
- `FuzzCalculus_DiffIntegrateInversion`: 微分と積分の逆演算
- `FuzzDomain_UnitsDimensionalConsistency`: 物理単位の次元整合性

```bash
# 全 Fuzz ターゲットのシードコーパスおよびオラクル・プロバイダテストのスモーク実行
go test -v ./internal/calc -run "^(Fuzz|TestThreeTierOracle_|TestByteStreamProvider_)"

# 特定ターゲットに対する連続ファジング探索（例: 30 秒間）
go test ./internal/calc -run "^FuzzAlgebra_SimplifyIdempotence$" -fuzz "^FuzzAlgebra_SimplifyIdempotence$" -fuzztime 30s
```

### SymPy 常駐差分ファジングオラクル (`tools/sympy-oracle`)

Python 3 と SymPy が導入された環境では、`tools/sympy-oracle/sympy_oracle.py`（標準入出力 NDJSON 常駐プロセス）経由で SymPy との差分検証を実行できます。SymPy が未導入の環境では関連テストが自動スキップされますが、環境変数 `IHD_REQUIRE_SYMPY_ORACLE=1` を指定すると不在時にスキップせず失敗として検出します。

```bash
# CI（fuzz-smoke.yml）と同等の SymPy 必須スモーク検証
IHD_REQUIRE_SYMPY_ORACLE=1 GOMEMLIMIT=2GiB IHD_NO_UPDATE_CHECK=1 \
  go test -v -count=1 -timeout 3m ./internal/calc \
  -run "^(Fuzz|TestThreeTierOracle|TestI18nErrorIntegrity|TestByteStreamProvider|TestGuard|TestDifferentialOracle|TestASTShrinker|TestSymPyClient)"
```

### Bytecode VM 不変条件テストとベンチマーク

バイトコード VM と AST 評価器の一致性および相対速度比を検証します。

```bash
# VM 不変条件と差分テスト
go test -v -count=1 -run "TestVM_Invariants|TestVM_Differential" ./internal/calc/...

# AST 評価器との相対速度ベンチマーク
go test -v -run "^$" -bench "BenchmarkVM_RelativeSpeedup" -benchmem -benchtime 1s ./internal/calc/...
```

---

## 5. 独立証明検証器 (`tools/micro-verifier`) と Lean 4 形式検証 (`formal/`)

### 独立証明検証器 (`tools/micro-verifier`)

本体ソルバの簡約ロジックに依存せず、`ihd --certificate <file>` で出力された証明書 JSON（version `"1.0"`）を再検証するスタンドアロン CLI ツールです。

```bash
# 独立検証器の単体テスト
go test ./tools/micro-verifier

# 証明書ファイルの独立検証
go run ./tools/micro-verifier cert.json
```

### Lean 4 形式証明検証 (`formal/`)

`ihd --lean` / `--lean-file` が生成する定理証明コードは、[formal/](../formal) 配下の Lean 4 プロジェクトでカーネル検証できます。再現性を維持するため、`formal/lean-toolchain` は `leanprover/lean4:v4.36.0-rc2`、`formal/lake-manifest.json` は 40 桁 SHA コミットで固定されており、`lake update` の実行は `lean-pinning` ルールによって禁止されています。

```bash
# 例: Lean 4 証明コードを生成し、Mathlib キャッシュ取得後にカーネル検証
go run ./cmd/ihd --lean-file formal/Proofs/Sample.lean "factor(x^4 + 4)"
cd formal
lake exe cache get
lake env lean Proofs/Sample.lean

# 公理監査バイナリ（sorryAx 等が混入していないかの自己テスト）
lake build ihd-audit
lake exe ihd-audit --self-test

# 証明カタログと負債ラチェットの集計（リポジトリルートから実行）
cd ..
python3 tools/lean/proof_catalog.py
```

---

## 6. 実演ショウケース (Exam) とリファレンス HTML ビルド・監査

### 実演ショウケース (Exam)

高校数学から数学オリンピック・大学理工系分野までの代表的な数学問題を自動実行します。問題と解説は [日本語版レシピ集](../exam/problems.ja.md) に収録しています。

```bash
# Windows (PowerShell)
pwsh exam/exam.ps1

# Linux / macOS (Bash)
./exam/exam.sh
```

### リファレンス HTML ビルドとドキュメント監査

`tools/reference-builder/node_modules` が未作成の初回環境では、先に `tools/reference-builder` で `npm ci` を実行します。リファレンス Markdown（`docs/references/{ja,en}/reference.{ja,en}.md`）を更新した際は、オフライン Single-File HTML とビルド生成物 `docs/references/dist/doc_specs.json`（Git 未追跡）を再生成したうえで、`tools/doc-auditor` を実行します。

```bash
# Single-File HTML と doc_specs.json の再生成
node tools/reference-builder/index.js

# 関数レジストリ・引数数・構文例の一致監査
go run ./tools/doc-auditor
```

不整合が 0 件なら既存の監査レポートを削除して終了コード `0` で終了します。1 件でも不整合がある場合は `docs/reference_audit_report.md` を生成して終了コード `1` で終了します。また、`doc_specs.json` が生成済みの環境では `go test ./internal/calc -run TestReferenceDocumentationInvariants` でも同じ不変条件を検証できます。

---

## 7. GitHub Actions CI ワークフロー一覧

公開リポジトリでは以下のワークフローが自動検証およびリリースを担っています。

| ワークフロー | 主な実行内容 |
| :--- | :--- |
| `unit-test.yml` | `go run ./tools/repo-lint --all` と全パッケージ単体テスト（`go test -v -timeout 5m ./...`）を実行 |
| `e2e-test.yml` | `go run ./tools/e2e` による CSV テストスイートの E2E 検証を実行 |
| `reference-sync.yml` | `node tools/reference-builder/index.js` の差分同期確認と `go run ./tools/doc-auditor` を実行 |
| `fuzz-smoke.yml` | Python 3.12 + SymPy 環境での Fuzz・PBT・SymPy 差分スモーク検証を実行 |
| `lean-verify.yml` | `formal/test_cases.txt` の Lean 4 カーネル検証、`ihd-audit --self-test`、`proof_catalog.py` を実行 |
| `typst-verify.yml` | `ihd --typst` 出力の Typst CLI による PDF / SVG コンパイル検証を実行 |
| `vm-perf.yml` | Bytecode VM の不変条件・差分テストおよび相対速度ベンチマークを実行 |
| `release.yml` | `v*` タグ push 時の複数 OS・アーキテクチャ向けビルド、アーカイブ同梱、`SHA256SUMS` 生成とリリース公開を実行 |
