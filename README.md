# i-hate-decimal-calc / ihd

<p align="center">
  <img src="./assets/logo/ihd-logo.svg" alt="ihd (i-hate-decimal-calc) Logo" width="798" />
</p>

小数を入力・計算の途中で一切使わず、すべての数式と数値を任意精度有理数（`big.Rat`）および記号 AST ノードのまま厳密評価し、計算結果・途中式・外部検証用コード（Lean 4 / Typst / 証明書 JSON）を返すコマンドライン数式処理（CAS）電卓です。

![Go](https://img.shields.io/badge/Go-1.26.5-00ADD8?style=flat-square&logo=go&logoColor=white)
[![Lean 4 Verification](https://github.com/DovahkiinYuzuko/i-hate-decimal-calc/actions/workflows/lean-verify.yml/badge.svg)](https://github.com/DovahkiinYuzuko/i-hate-decimal-calc/actions/workflows/lean-verify.yml)
[![Typst Math Verification](https://github.com/DovahkiinYuzuko/i-hate-decimal-calc/actions/workflows/typst-verify.yml/badge.svg)](https://github.com/DovahkiinYuzuko/i-hate-decimal-calc/actions/workflows/typst-verify.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=flat-square&logo=opensourceinitiative&logoColor=white)](./LICENSE.MIT)

[日本語](README.md) | [English](README.en.md)

## 目次

- [なぜ ihd なのか（小数の排除と使いどころ）](#なぜ-ihd-なのか小数の排除と使いどころ)
- [インストール](#インストール)
- [クイックスタート](#クイックスタート)
- [主な機能と検証・出力エコシステム](#主な機能と検証出力エコシステム)
- [入手形態別のファイル構成](#入手形態別のファイル構成)
- [主なコマンドとリファレンス](#主なコマンドとリファレンス)
- [設計上の制約と外部検証](#設計上の制約と外部検証)
- [開発とテスト](#開発とテスト)
- [実践レシピ集（Exam & Cookbook）](#実践レシピ集exam--cookbook)
- [LICENSE](#license)

## なぜ ihd なのか（小数の排除と使いどころ）

一般的なプログラミング言語や関数電卓では、小数を扱う際に `float64` などの浮動小数点数を使用するため丸め誤差が生じ、無理数や超越数も一定の桁数で近似値に打ち切られます。

`ihd`（i-hate-decimal-calc）は、**「小数の存在を一切許さない」** という設計思想のもとで作られたコマンドライン CAS 電卓です。入力された小数・循環小数は字句解析の段階で即座に任意精度有理数（`big.Rat`）へ変換され、平方根・三角関数・対数・超越数・虚数単位・$p$進展開・ピュイズー級数・有理閉区間は第一級シンボルノードとして保持されます。

| 入力数式 | 一般的な電卓・言語（浮動小数点数） | `ihd`（厳密計算） | 処理・簡約内容 |
| :--- | :--- | :--- | :--- |
| `0.1 + 0.2` | `0.30000000000000004` | `3/10` | 誤差のない有理数約分 |
| `0.(3)` | `0.3333333333333333` | `1/3` | 有限状態機械（FSM）字句解析による循環小数の分数化 |
| `sqrt(8)` | `2.8284271247461903` | `2*√2` | 平方因子の自動くくり出し |
| `sqrt(5 + 2*sqrt(6))` | `3.1462643699419726` | `√2 + √3` | Borodin (1985) 法による二重根号の分解 |
| `1 / (sqrt(2) + 1)` | `0.4142135623730951` | `-1 + √2` | 2項無理数分母の共役有理化 |
| `sin(pi/6)` | `0.49999999999999994` | `1/2` | 特殊角の代数的評価 |
| `１＋２×３` | エラー（全角未対応） | `7` | 日本語 IME の全角文字を半角 ASCII へ透過変換 |

### 向いている用途と他ツールとの棲み分け

- **ランタイム依存のない厳密検算環境**: Python や SymPy を導入できない、または要求したくない環境（CI パイプライン、オフライン端末、コンテナ等）へ単一バイナリのみで配置し、数式の簡約や `ihd diff` による代数的同値性判定を実行できます。
- **日本語 IME のままの対話計算と組版連携**: 全角数字・全角演算子・全角括弧をそのまま受け付ける REPL に加え、導出過程を **Typst（`--typst`）**、**LaTeX（`--latex`）**、**Lean 4（`--lean`）** へ直接出力できます。
- **他ツールが適する領域**: 汎用プログラミング言語からのライブラリ組み込みや、大規模な数値配列・行列の浮動小数点シミュレーションには Python（SymPy / NumPy）などのエコシステムが適しています。`ihd` は端末・スクリプト単体で完結する記号計算と、導出過程・計算証明書の外部検証に特化しています。

<details>
<summary><b>ロゴの小話</b></summary>

- **小数点へのアンチテーゼ**: トップバナーの `ihd` の `i` の上のドットは、`sqrt(0.1 + 0.2)` の真の値 `0.54772255750516611345...`（無限に続く無理数）が小数第16位で打ち切られ末尾が `...1661` から `...1662` へと丸め込まれた近似小数値 `0.5477225575051662` の小数点 `.` と座標（`cx=125.2, cy=76.8`）を共有しており、背面レイヤーの多面体（ローポリゴン）の中指によって下から突き上げられ打ち消されています。その対比として、右側には `ihd` による有理化・平方因子抽出済みの厳密解 `√30/10` が配置されています。
- **`ihd` 自身によるレイアウト余白の連立方程式求解**: バナー内の左タイポグラフィ（実測幅 `164px`）と右REPL出力（実測幅 `302px`）について、左右の外枠（左端 `x=10`）からの外側余白および中央の斜め分割線からの内側余白が対称となる4つのX座標 $[L_{\min}, X_{\text{div}}, R_{\min}, X_{\text{right}}]$ は、`ihd` 自身の連立一次方程式ソルバ `linsolve` を用いて算出・配置されています。
  ```text
  ihd> linsolve([[1, 0, 1, -1], [1, -2, 1, 0], [1, 0, 0, 0], [0, 1, 0, 0]], [-292, -164, 100, 330])
  [100, 330, 396, 788]
  ```
</details>

## インストール

### ワンライナーインストール

**Linux / macOS (bash / zsh):**
```bash
curl -fsSL https://raw.githubusercontent.com/DovahkiinYuzuko/i-hate-decimal-calc/main/install.sh | bash
```

**Windows (PowerShell):**
```powershell
irm https://raw.githubusercontent.com/DovahkiinYuzuko/i-hate-decimal-calc/main/install.ps1 | iex
```

> [!NOTE]
> **Android (Termux) やログインシェル環境での `PATH` 設定**:  
> Termux などの環境では起動時に `~/.bashrc` ではなく `~/.bash_profile` が優先評価されます。`PATH` を永続化する場合は `~/.bash_profile` に `export PATH="$HOME/.ihd/bin:$PATH"` を追記するか、`~/.bash_profile` 内で `[ -f ~/.bashrc ] && . ~/.bashrc` を読み込みます。

### ソースコードからのビルド

Go 1.26.5 が導入されている環境では、リポジトリルートから単一バイナリをビルドできます。

```bash
go build -o ihd ./cmd/ihd
```

## クイックスタート

コマンドライン引数に数式を渡すと、即座に有理数・記号形式で評価結果を返します。

```bash
# 有理数演算（丸め誤差なし）
ihd "0.1 + 0.2"
# 出力: 3/10

# 二重根号の分解と同値性判定
ihd "sqrt(2) + sqrt(3) == sqrt(5 + 2*sqrt(6))"
# 出力: true

# 実代数不等式の求解（CAD / Sturm根分離）
ihd "solve(x^2 - 4 < 0)"
# 出力: [[-2, 2]]
```

用途に応じて、以下の4つの入力形態を使い分けられます。

<details>
<summary><strong>対話モード（REPL）・標準入力パイプ・スクリプト実行・全角正規化の実行例</strong></summary>

#### 1. 対話モード（REPL）
引数なしで `ihd` を起動すると、履歴管理・リアルタイム引数ガイド・構文ハイライト付きの REPL が起動します。

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

#### 2. 標準入力パイプ（バッチ連携）
標準入力から複数行を受け取ると、同一セッション内の変数束縛を引き継いで順次評価します。

```bash
printf "x = 1/2 + sqrt(2)\nx * 2\n" | ihd
# 出力:
# 1/2 + √2
# 1 + 2*√2
```

#### 3. スクリプトファイルの実行（`ihd run <file.ihd>`）
`.ihd` テキストファイルに記述した計算を一括実行します。行末に `;` を付けた行は評価のみ行い標準出力を抑制します。

```bash
ihd run sample.ihd
# ファイル拡張子による直接指定にも対応
ihd sample.ihd
```

#### 4. 日本語 IME の全角自動正規化
全角の英数字、演算子（`＋` `−` `×` `÷` `＾` `！` `＝`）、括弧、空白は字句解析前に半角 ASCII へ自動変換されます。

```bash
ihd "１＋２×３"
# 出力: 7
```

</details>

## 主な機能と検証・出力エコシステム

### 閉領域第一級ノードと厳密演算
- **閉領域の第一級 AST ノード（`PadicNode` / `PuiseuxNode` / `IntervalNode`）**: `padic_expand` の $p$進切断級数、`puiseux` のピュイズー級数、`interval` の有理閉区間を文字列へ退化させず第一級ノードとして保持し、閉じた四則演算・整数べき乗・微分・順序比較を実行します。異なる素数や記号係数が混在する場合は暗黙の近似を行わず、記号式または定義域エラーとして扱います。
- **多段階・予算制御付き整数素因数分解（`factor`）**: 小素数の試し割り、完全べき乗判定、Pollard-Brent rho 法、決定論的パラメータの Lenstra 楕円曲線法（ECM Stage 1 / Stage 2）を順に適用します。
- **有理数ベクトル空間による物理単位計算**: SI 基本7単位に対応する有理数体 $\mathbb{Q}^7$ 上の次元解析により、小数を介さず単位乗除算と換算（`convert_unit`）を行います。

### 可視化・形式出力・対話支援
- **途中式ツリーと単一 DAG からの形式出力射影（`--explain` / `--typst` / `--lean`）**: 有理化・二重根号分解・因数分解・極限等の導出過程を 2D ツリーで表示し、同じ導出グラフから Typst レポート（`--explain --typst`）および Lean 4 等式連鎖証明（`--explain --lean`）を生成します。
- **端末組版と 2D/3D 点字・ベクタープロッタ（`--pretty` / `plot` / `plot3d`）**: 複数行ボックスモデルによる数式組版に加え、Unicode Braille 2×4 点字文字および SVG / Typst による 2D 曲線プロット（`plot` / `plot_svg`）と 3D 曲面・空間曲線ワイヤーフレーム描画（`plot3d` / `plot3d_svg`、隠線処理・視点適応 3D 座標軸・CAS 厳密特徴点検出対応）を備えています。
- **REPL 入力支援とコレクション制御**: 外部依存ゼロの FSM によるリアルタイム引数ガイド・レインボー括弧に加え、Python 互換スライス（`[start:end:step]`）、`range`、`lambda`、`map`、`filter`、反復安全ガード（`--warn-loop` / `--max-loop`）を搭載しています。

ターミナル組版、点字プロット、途中式ツリー、Typst / Lean 4 の実出力例は **[出力サンプル集（日本語）](./sample/sample.ja.md)** / **[Visual Samples（English）](./sample/sample.en.md)** に収録しています。

## 入手形態別のファイル構成

リリースアーカイブ・インストーラ経由で導入した場合と、リポジトリをクローンした場合とで手元に配置されるファイル群が異なります。

| ファイル・ディレクトリ | リリースアーカイブ展開先 | インストーラ配置先（`~/.ihd/`） | リポジトリ限定 | 役割 |
| :--- | :--- | :--- | :--- | :--- |
| `ihd` / `micro-verifier` バイナリ | `./`（アーカイブ直下） | `~/.ihd/bin/` | - | 本体 CLI と独立証明書検証器 |
| リファレンス（MD / HTML / JSON） | `docs/references/` | `~/.ihd/references/` | - | オフライン仕様書と機械監査用メタデータ |
| 実演ショウケース・問題集（`exam/`） | `exam/` | `~/.ihd/exam/` | - | 24問の数学ベンチマークと解説 |
| 出力サンプル集（`sample/`） | `sample/` | `~/.ihd/sample/` | - | CLI 出力・組版・証明コードの実例集 |
| 検証補助ソース（`tools/` 配下） | `tools/micro-verifier/`, `tools/sympy-oracle/` | `~/.ihd/tools/` 配下 | - | 独立検証器ソースと SymPy 差分オラクル |
| `README.md`, `README.en.md`, `LICENSE.MIT`, `NOTICE.md` | `./`（アーカイブ直下） | `~/.ihd/` 直下 | - | 概要ドキュメントおよびライセンス表記 |
| `docs/development.*.md`, `tools/`, `internal/` 等 | 含まれない | 含まれない | 同梱 | 単体テスト・Fuzz・E2E・HTML ビルド環境 |

## 主なコマンドとリファレンス

### サブコマンド

| コマンド | 説明・使用例 |
| :--- | :--- |
| `ihd diff <f1\|e1> <f2\|e2>` | 2つの数式またはスクリプトの構文差分と代数的同値性（`EQUIVALENT` / `CHANGED`）を判定。<br>`ihd diff "x^2 - 1" "(x - 1)*(x + 1)"` |
| `ihd run <file.ihd>` | `.ihd` スクリプトファイルの一括実行。<br>`ihd run script.ihd` |
| `ihd completion <shell>` | `bash`, `zsh`, `fish`, `powershell` 向けの補完スクリプト生成。<br>`ihd completion powershell \| Out-String \| Invoke-Expression` |
| `ihd update` | GitHub Releases からの自己アップデート（確認のみ `--check` / `-c`、強制再導入 `--force` / `-f`）。 |

### 主な CLI オプション

| オプション | 説明・使用例 |
| :--- | :--- |
| `-a`, `--approx` | 厳密解の横に有理数区間演算に基づく任意精度 `big.Float` 近似値と誤差上限 `±` を併記（既定 30 桁、`--digits <N>` で変更）。<br>`ihd -a "sqrt(2)"` |
| `--format <fmt>` | 出力形式を指定（`typst`, `latex`, `ascii`, `pretty`, `svg`）。個別フラグ `--typst`, `--latex`, `--ascii`, `--pretty` も指定可能。<br>`ihd --format typst "1/2 + sqrt(2)"` |
| `--explain` | 有理化・因数分解・微分・極限等の計算過程を途中式ツリーで表示。<br>`ihd --explain "1 / (sqrt(2) + 1)"` |
| `--verify` | 定義式や導関数に基づき計算結果の残差が 0 になるかを逆算検算。<br>`ihd --verify "integrate(1/(x^2 + 1), x)"` |
| `-i`, `--interval` | 適応的有理数区間演算による真値包含区間を算出（許容幅は `--interval-eps` で指定）。<br>`ihd -i "sin(1)"` |
| `--lean`, `--lean-file <path>` | **Lean 4（Mathlib4）** の形式証明コードを標準出力または `.lean` ファイルへ出力。<br>`ihd --lean "factor(x^2 - 1)"` |
| `-h`, `--help` | カテゴリ別に色分けされたコマンドヘルプを表示。 |

<details>
<summary><strong>実行制御・トレース・証明書関連のオプション一覧を展開</strong></summary>

| オプション | 説明・使用例 |
| :--- | :--- |
| `--step` | 各簡約ステップの適用ルールと中間式を標準出力へ順次出力。<br>`ihd --step "expand((x+1)^3)"` |
| `--trace` | 式木の書き換え過程や内部探索イベントを構造化ログとして出力。<br>`ihd --trace "factor(x^2 - 1)"` |
| `--certificate <file>` | 計算結果と導出ステップ列を証明書 JSON（version `"1.0"`）として保存。<br>`ihd --certificate cert.json "factor(x^2 - 1)"` |
| `--replay <file>` | 証明書 JSON ファイルを読み込み、ソルバを再実行せず各ステップの残差 0 を再検証。<br>`ihd --replay cert.json` |
| `--deg` | 三角関数・逆三角関数を度数法（Degree）として解釈。<br>`ihd --deg "sin(30)"` |
| `--theme <dark\|light\|none>` | REPL のカラーテーマを指定（`~/.ihd/config.json` に保存）。<br>`ihd --theme light` |
| `--lang <ja\|en\|auto>` | 表示言語を指定。<br>`ihd --lang en "1/2 + 1/3"` |
| `--timeout <dur>` | 壁時計時間のタイムアウトを指定（既定: 制限なし）。<br>`ihd --timeout 2s "factor(x^100 - 1)"` |
| `--budget-steps <N>` | 単一簡約パスあたりの最大書き換えステップ予算を指定（既定: `0` = 無制限）。<br>`ihd --budget-steps 5000 "expand((x+1)^20)"` |
| `--max-nodes <N>` | 式木展開時の最大許容 AST ノード数を指定（既定: `0` = 無制限）。<br>`ihd --max-nodes 10000 "expand((x+1)^10)"` |
| `--max-depth <N>` | 再帰的簡約処理の最大深度上限を指定（既定: `0` = 無制限）。<br>`ihd --max-depth 200 "diff(x^10, x)"` |
| `--warn-loop <N>` | 反復構文（`table`, `for`, `product`）で標準エラー出力へ警告を出す回数閾値（既定: `100000`）。 |
| `--max-loop <N>` | 反復構文の実行を強制遮断する最大回数上限（既定: `-1` = 無制限）。 |
| `--deterministic` | マップ反復や多項式順序を正規化し、バイト単位で再現可能な出力を生成。<br>`ihd --deterministic "factor(x^4 - 5*x^2 + 4)"` |
| `--no-color` | ANSI カラーシーケンス出力を無効化。 |
| `--no-update-check` | 起動時の新バージョン確認を無効化。 |
| `-v`, `--version` | バージョン情報を表示。 |

</details>

### 搭載領域とリファレンスへの案内

`ihd` の組み込み関数は、初等代数から数論・実代数幾何・局所解析・形式証明までの領域をカバーしています。

| 領域 | 主な関数 |
| :--- | :--- |
| **基本代数・数論・方程式** | `expand`, `factor`, `apart`, `together`, `gcd`, `lcm`, `mod`, `is_prime`, `solve`, `cfrac`, `groebner`, `sturm`, `interval` |
| **三角関数・対数・複素数** | `sin`, `cos`, `tan`, `asin`, `acos`, `atan`, `trig_expand`, `trig_reduce`, `exp`, `log`, `ln`, `polar` |
| **微積分・ODE・特殊関数・局所級数** | `diff`, `integrate`, `limit`, `taylor`, `puiseux`, `dsolve`, `laplace`, `lambert_w`, `erf`, `gamma`, `sum` |
| **線形代数・線形計画・整数標準形** | `det`, `inv`, `rank`, `rref`, `eigenvals`, `linsolve`, `lp_maximize`, `hnf`, `snf`, `lll` |
| **幾何学解析・自動証明** | `line_intersect`, `circle_intersect`, `triangle_area`, `geo_prove` |
| **代数構造・p進数・数論幾何・量子** | `padic_val`, `padic_norm`, `padic_expand`, `galois_group`, `is_solvable_by_radicals`, `ec_p_order`, `ec_rank_bound`, `clifford`, `quantum_equiv` |
| **物理単位・仮定・反復** | `convert_unit`, `assume`, `unassume`, `table`, `range`, `map`, `filter` |
| **可視化・2D/3D 空間プロット** | `plot`, `plot_svg`, `plot3d`, `plot3d_svg` |

全関数・引数仕様・優先順位・全 CLI フラグ一覧は、専用リファレンスまたは `ihd --help` で確認できます。
- **Markdown 版**: [日本語版リファレンス](docs/references/ja/reference.ja.md) / [英語版リファレンス](docs/references/en/reference.en.md)
- **オフライン Single-File HTML 版**: [日本語版リファレンス (HTML)](docs/references/ja/reference.ja.html) / [英語版リファレンス (HTML)](docs/references/en/reference.en.html)（※インストーラ配置先は `~/.ihd/references/`）

## 設計上の制約と外部検証

- **全入力域での無謬性は保証されない（仕様事実）**: 複雑な不定積分、高次代数方程式、極限、記号簡約において、アルゴリズムの適用範囲や実装上の制限により誤った結果や未簡約・一部解の欠落が生じる場合があります。また、`factor` による巨大合成数の素因数分解には、計算ハングを防ぐため決定論的な作業予算上限を設けており、上限到達時は発見済みの素因数と未分解の合成数因子を残した部分分解を返します。
- **外部ツール非依存と交換したトレードオフ（理由）**: 外部の巨大 CAS やランタイムを抱えずに単一バイナリで完結させる設計を採っているため、すべての数学領域を単体で網羅する代わりに「計算結果を無条件に信用せず外から検証できる仕組み」を優先しています。
- **検証手段の併用（対処法）**: GitHub Actions CI 上では単体テスト（`unit-test.yml`）、Lean 4 形式検証（`lean-verify.yml`）、Typst 数式コンパイル（`typst-verify.yml`）、不変条件ファジング（`fuzz-smoke.yml`）を常時実行しています。研究・業務等で確証が必要な計算では、`--verify`（逆算検算）、`--certificate` + `micro-verifier`（CAS 本体に依存しない独立再検証）、`--lean`（Lean 4 定理コード出力）、または同梱の `tools/sympy-oracle` を組み合わせて検証できます。

## 開発とテスト

リポジトリをクローンした環境では、Go 単体テストと実演ショウケースをすぐに実行できます。

```bash
# Go 単体テストの実行
go test ./...

# 高校〜大学数学 24 問の実演ショウケース（Windows / Linux・macOS）
pwsh exam/exam.ps1
./exam/exam.sh
```

ファジング（$\mathbb{Q}[i]$ 上の Schwartz-Zippel オラクル）、独立証明検証器 `tools/micro-verifier`、SymPy 常駐差分オラクル `tools/sympy-oracle`、およびドキュメント監査の詳しい手順は **[開発・テスト・検証ガイド（日本語）](docs/development.ja.md)** / **[Development Guide（English）](docs/development.en.md)** にまとめています。

## 実践レシピ集（Exam & Cookbook）

24問の代表的な数学問題（高校数学・難関大入試・線形代数・微積分・数論・幾何）に対する `ihd` の解法と解説を収録しています。
- [日本語版レシピ集](./exam/problems.ja.md)
- [英語版レシピ集](./exam/problems.en.md)

## LICENSE

[MIT](./LICENSE.MIT)

Third-Party → [NOTICE.md](./NOTICE.md)
