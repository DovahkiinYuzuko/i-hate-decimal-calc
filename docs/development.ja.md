# ihd 開発・テスト・検証ガイド

[日本語](development.ja.md) | [English](development.en.md) | [README に戻る](../README.md)

このガイドは、`ihd` のリポジトリをクローンして Go 単体テスト、数学的不変条件ファジング、独立証明検証、SymPy 差分オラクル、およびドキュメント監査を実行する開発者・コントリビュータ向けの手順書です。リリースアーカイブやインストーラ（`~/.ihd/references/`）に同梱される利用向けリファレンス（[reference.ja.md](references/ja/reference.ja.md)）とは役割が異なり、リポジトリ直下のソース・テスト基盤を対象としています。

- **基本前提環境**: Go 1.26.5（Exam ランナーは `pwsh` または `bash` を使用）
- **追加ツール（該当テスト実行時のみ）**: Python + SymPy（差分オラクル）、Node.js（または Bun：リファレンス HTML 再生成）

## 目次

- [1. 単体テスト](#1-単体テスト)
- [2. 数学的不変条件 Property-Based Testing & Fuzzing](#2-数学的不変条件-property-based-testing--fuzzing)
- [3. 独立証明検証器 (`tools/micro-verifier`)](#3-独立証明検証器-toolsmicro-verifier)
- [4. 実演ショウケース (Exam)](#4-実演ショウケース-exam)
- [5. SymPy 常駐差分ファジングオラクル (`tools/sympy-oracle`)](#5-sympy-常駐差分ファジングオラクル-toolssympy-oracle)
- [6. リファレンス HTML ビルドとドキュメント監査](#6-リファレンス-html-ビルドとドキュメント監査)

## 1. 単体テスト

全パッケージの Go 単体テストを実行します。

```bash
go test ./...
```

## 2. 数学的不変条件 Property-Based Testing & Fuzzing

ガウス有理数体 $\mathbb{Q}[i]$ 上の多点評価オラクル（Schwartz-Zippel 補題）および代数・微積分の不変条件を用いたファジングハーネスを実行します。

```bash
# 全 Fuzz ターゲットのシードコーパスおよびオラクル・プロバイダテストのスモーク実行
go test -v ./internal/calc -run "^(Fuzz|TestThreeTierOracle_|TestByteStreamProvider_)"

# 特定ターゲットに対する連続ファジング探索（例: 30秒間）
go test ./internal/calc -run "^FuzzAlgebra_SimplifyIdempotence$" -fuzz "^FuzzAlgebra_SimplifyIdempotence$" -fuzztime 30s
```

主な Fuzz ターゲットは [internal/calc/](../internal/calc) 配下の `fuzz_*_test.go` に定義されており、簡約べき等性（`FuzzAlgebra_SimplifyIdempotence`）、因数分解と展開の往復（`FuzzAlgebra_FactorExpandRoundTrip`）、微分と積分の逆演算（`FuzzCalculus_DiffIntegrateInversion`）、物理単位の次元整合性（`FuzzDomain_UnitsDimensionalConsistency`）を検証します。

## 3. 独立証明検証器 (`tools/micro-verifier`)

本体ソルバの簡約ロジックに依存せず、`ihd --certificate <file>` で出力された証明書 JSON（version `"1.0"`）を独立して再検証するスタンドアロン CLI ツールです。

```bash
# 独立検証器の単体テスト
go test ./tools/micro-verifier

# 証明書ファイルの独立検証
go run ./tools/micro-verifier cert.json
```

## 4. 実演ショウケース (Exam)

高校数学から数学オリンピック・大学理工系分野までの代表 24 問を自動実行するデモです。問題と解説は [日本語版レシピ集](../exam/problems.ja.md) に収録しています。

```bash
# Windows (PowerShell)
pwsh exam/exam.ps1

# Linux / macOS (Bash)
./exam/exam.sh
```

## 5. SymPy 常駐差分ファジングオラクル (`tools/sympy-oracle`)

Python と SymPy が導入された環境では、同梱スクリプト（`tools/sympy-oracle/sympy_oracle.py`）による常駐 JSON-RPC プロセス経由で、SymPy との差分検証（微分・代数簡約・有理式・組合せ論など）を実行できます。

```bash
# SymPy オラクル単体テストおよび差分検証
go test -v ./internal/calc -run "^(TestSymPyClient_|TestDifferentialOracle_)"
```

## 6. リファレンス HTML ビルドとドキュメント監査

リファレンス Markdown（`docs/references/{ja,en}/reference.{ja,en}.md`）を更新した際は、オフライン Single-File HTML と `doc_specs.json` を再生成した上で、`tools/doc-auditor` による関数レジストリ・構文例整合性チェックを実行できます（初回のみ `tools/reference-builder` で `npm ci` が必要です）。

```bash
# Single-File HTML と doc_specs.json の再生成
node tools/reference-builder/index.js

# doc-auditor による整合性監査（不整合ゼロで終了コード 0）
go run ./tools/doc-auditor
```
