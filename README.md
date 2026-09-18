# i-hate-decimal-calc / ihd

小数を一切許さず、すべての入力を即座に分数化し、平方根や定数をシンボルのまま厳密計算する電卓CLIツール / A strict CAS calculator CLI that completely rejects decimal approximations, converting all decimals to exact fractions and maintaining exact algebraic forms.

![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go&logoColor=white)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=flat-square&logo=opensourceinitiative&logoColor=white)](./LICENSE.MIT)

[日本語](#日本語) | [English](#english)

---

## 日本語

### なぜ ihd なのか？（小数の完全撲滅）

一般的なプログラミング言語や関数電卓では、小数を扱う際に `float64` などの浮動小数点数を使用するため、不可避の丸め誤差が生じます。また、無理数や超越数も適当な桁数で丸められてしまいます。

`ihd`（i-hate-decimal-calc）は、**「小数の存在を一切許さない」** という強い思想のもと設計された、コマンドライン向けの完全厳密数式処理システム（CAS）電卓です。入力された小数は字句解析の段階で即座に任意精度有理数（`big.Rat`）へ変換され、平方根・三角関数・対数・超越数・虚数単位はシンボルノードとして保持され、代数的簡約ルールによって厳密な形式のまま計算されます。

| 入力数式              | 一般的な電卓・言語（浮動小数点数） | `ihd`（完全厳密計算） | 簡約・処理内容                               |
| :-------------------- | :--------------------------------- | :-------------------- | :------------------------------------------- |
| `0.1 + 0.2`           | `0.30000000000000004`              | `3/10`                | 誤差ゼロの完全有理数約分                     |
| `0.(3)`               | `0.3333333333333333`               | `1/3`                 | 循環小数をFSM字句解析により厳密分数化        |
| `sqrt(8)`             | `2.8284271247461903`               | `2*√2`                | 平方因子の自動くくり出し                     |
| `sqrt(5 + 2*sqrt(6))` | `3.1462643699419726`               | `√2 + √3`             | Borodin (1985) 法による二重根号の自動分解    |
| `1 / (sqrt(2) + 1)`   | `0.4142135623730951`               | `-1 + √2`             | 2項無理数分母の共役有理化                    |
| `sin(pi/6)`           | `0.49999999999999994`              | `1/2`                 | 特殊角の代数的厳密値評価                     |
| `１＋２×３`           | エラー（全角未対応）               | `7`                   | 日本語IMEの全角文字を透過的に半角ASCII正規化 |

---

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
Go 1.22 以上がインストールされている環境で、リポジトリルートにてビルドを実行します。

```bash
go build -o ihd ./cmd/ihd
```

---

### クイックスタート & 実行形態

#### 1. ワンショット計算（CLI引数）
コマンドライン引数に数式を渡すだけで即座に厳密計算を実行します。

```bash
ihd "1/2 + 1/3"
# 出力: 5/6

ihd "sqrt(8) + sqrt(18)"
# 出力: 5*√2

ihd "0.(142857)"
# 出力: 1/7（循環小数の自動分数化）

ihd "abs(3 + 4*i)"
# 出力: 5（複素数の絶対値）

ihd "root_count(x^3 - 3*x + 1, x)"
# 出力: 3（スツルムの定理による実根の厳密個数算定）

ihd "isolate_roots(x^2 - 2, x)"
# 出力: [[-3/2, -3/4], [3/4, 3/2]]（実根の有理数区間完全分離）
```

#### 2. 対話モード（REPL）
引数なしで起動すると、履歴機能・カーソル移動・日本語IME対応を備えた対話型REPLが起動します。

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
ihd> 2*x - x
1 + √2
ihd> vars
ans = 1 + √2
x = 1 + √2
ihd> exit
Goodbye.
```

#### 3. 標準入力パイプ（バッチ・スクリプト連携）
他コマンドからのパイプ入力を処理できます。同一セッション内では変数が引き継がれ、空行や `#` から始まるコメント行は自動的に無視されます。

```bash
echo "sin(pi/6)^2 + cos(pi/6)^2" | ihd
# 出力: 1

printf "x = 1/2 + sqrt(2)\nx * 2\n" | ihd
# 出力:
# 1/2 + √2
# 1 + 2*√2
```

#### 4. スクリプトファイルのバッチ実行（`ihd run <file.ihd>`）
複数行の数式や変数定義を記述した `.ihd` テキストファイルを一括実行できます。末尾にセミコロン `;` を付与した行は評価・変数代入を行いつつ結果の出力を抑制します（MATLAB / Julia 等のCASスクリプト準拠）。エラー発生時はファイル名と行番号が表示されます。

```bash
# sample.ihd の内容:
# a = 1/2;   # セミコロンで行末出力を抑制
# b = sqrt(8);
# a + 1
# b * 3

ihd run sample.ihd
# または直接指定:
ihd sample.ihd

# 出力:
# 3/2
# 6*√2
```

#### 5. 日本語IMEの全角自動正規化
日本語IMEがONのまま入力された全角文字（数字、英字、`＋` `−` `×` `÷` `＾` `！` `＝`、全角括弧、全角スペース等）は自動的に半角ASCIIへ透過変換されるため、入力モードを切り替えるストレスなく計算できます。

```bash
ihd "１＋２×３"
# 出力: 7
```

---

### ビジュアル機能ハイライト

#### 1. 代数変換プロセスを可視化する途中式ツリー（`--explain`）
分母の有理化や二重根号外し、方程式の求解、微分の適用過程などを2Dツリー形式の途中式として詳細に出力します（REPLやパイプでも `explain <式>` または `steps <式>` で実行可能）。

```bash
ihd --explain "sqrt(5 + 2*sqrt(6))"
```
```text
式: sqrt(5 + 2*sqrt(6))
├── [Step 1: 二重根号の簡約]
│   Borodinアルゴリズムによる二重根号の簡約
│   √(5 + 2*√6)  ──>  √2 + √3
└── [Result]
    = √2 + √3
```

#### 2. ターミナル2D数式組版プリティプリント（`--pretty`）
分数や平方根を複数行のアスキーアートボックスモデルで美しく描画します。

```bash
ihd --pretty "1/2 + sqrt(2)/2"
```
```text
 1     √2 
--- + ----
 2     2  
```

#### 3. Braille 2×4 サブピクセル高解像度プロッタ（`plot`）
Unicode点字文字を用いた 2×4 サブピクセル描画により、ターミナル上で美麗に関数の形状をプロットします。CASエンジンが零点や極値を自動検出し、厳密代数ラベルを軸上にピン留めします。さらに Tupper (2001) の不連続判定により漸近線の偽結合を防止します。

```bash
ihd "plot(sin(x), [-pi, pi])"
```
```text
    1 ┼                          │                        
      │                          │        ⣀⠤⠔⠒⠢⠤⡀         
      │                          │      ⡠⠊      ⠈⠑⢄       
      │                          │    ⡠⠊           ⠑⡄     
      │                          │  ⢀⠔⠁             ⠈⢢    
      │                          │ ⢀⠎                 ⠱⡀  
      │                          │⡠⠃                   ⠘⡄ 
      │                          ⡰⠁                     ⠈⢆
      │ ⠑⡄──────────────────────⡜┼────────────────────────
      │  ⠈⢆                   ⢀⠎ │                        
      │   ⠈⢢                 ⢠⠃  │                        
      │     ⠱⡀              ⡔⠁   │                        
      │      ⠈⢆           ⡠⠊     │                        
      │        ⠑⢄⡀      ⡠⠊       │                        
      │          ⠈⠒⠢⠤⠔⠒⠉         │                        
   -1 ┼                          │                        
      └──────────────────────────────────────────────────── x
       -π                                             π

[CAS Features Detected]
* No singularities or extrema (smooth monotonic curve)
* Domain:            x ∈ [-π, π], y ∈ [-1, 1]
```

---

### 構文・演算子・関数リファレンス

`ihd` は微積分・線形代数・常微分方程式・ラプラス変換・多変数代数消去法・離散確率など、大学理工系レベルを網羅する 50 種類以上の組み込み関数を搭載しています。

> [!TIP]
> 演算子と優先順位、予約定数、およびカテゴリ別全関数の詳細仕様・書式・使用例は専用リファレンスに網羅されています。  
> **[完全構文・演算子・関数リファレンス (Japanese Edition)](docs/references/reference.ja.md)**

#### 主なカテゴリと代表関数
- **基本代数・数論・方程式**: `sqrt`, `cbrt`, `factor`, `apart`, `together`, `poly_gcd`, `poly_lcm`, `resultant`, `groebner`, `to_poly`, `to_alg`, `alg_inv`, `min_poly`, `crt`, `solve`
- **三角関数・対数・複素数**: `sin`, `cos`, `tan`, `trig_expand`, `trig_reduce`, `log`, `ln`, `polar`, `rect`
- **微積分・常微分方程式・特殊関数・級数**: `diff`, `integrate`, `risch_integrate`, `limit`, `residue`, `gamma`, `beta`, `bernoulli`, `zeta`, `dsolve`, `laplace`, `inv_laplace`, `taylor`, `fourier_series`, `sum`, `gosper_sum`, `wz_cert`
- **線形代数・3次元ベクトル解析**: `det`, `inv`, `rref`, `rank`, `trace`, `eigenvals`, `eigenvects`, `lu`, `qr`, `cholesky`, `ldlt`, `pinv`, `solve_linear`, `dot`, `cross`, `norm`, `grad`, `div`, `curl`
- **幾何学解析**: `line_intersect`, `circle_intersect`, `triangle_area`, `triangle_centers`
- **厳密離散確率・統計**: `binom`, `hyper`, `geom`, `bayes`, `expect`, `variance`, `stddev`
- **前提条件システム（仮定）**: `assume`, `unassume`, `assumptions`

---

---

### コマンドラインオプション

| オプション           | 説明・使用例                                                                                                                                                           |
| :------------------- | :--------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `--ascii`            | 数学記号（`√`, `π`）をASCII文字列（`sqrt`, `pi`）にフォールバックして出力。<br>`ihd --ascii "sqrt(2) + pi"` $\to$ `sqrt(2) + pi`                                       |
| `--approx`           | 厳密解の横に参考用の浮動小数点小数近似値（float64、15〜17桁）を併記。<br>`ihd --approx "sqrt(2)"` $\to$ `√2 (≈ 1.4142135623730951)`                                    |
| `--latex`            | MarkdownやTeX論文に貼り付け可能なLaTeX形式（`$$ ... $$`）で出力。<br>`ihd --latex "1/2 + sqrt(2)"` $\to$ `$$ \frac{1}{2} + \sqrt{2} $$`                                |
| `--pretty`           | 分数線や根号を複数行アスキーアートで組版表示する2Dプリティプリント。<br>`ihd --pretty "1/2 + sqrt(2)/2"`                                                               |
| `--deg`              | 三角関数および逆三角関数を度数法（Degree）として解釈・計算。<br>`ihd --deg "sin(30) + cos(60)"` $\to$ `1`                                                              |
| `--explain`          | 代数的項書き換え（有理化・二重根号・微分則等）を途中式ツリーとして詳細表示。<br>`ihd --explain "1 / (sqrt(2) + 1)"`                                                    |
| `--lang <code/auto>` | 表示言語（ロケール）を指定（`ja`, `en`, `auto`）。設定は `~/.ihd/config.json` に永続化され、カスタム辞書（`~/.ihd/locales/`）にも対応。<br>`ihd --lang en "1/2 + 1/3"` |
| `-h`, `--help`       | コマンドのヘルプメッセージを表示。                                                                                                                                     |
| `-v`, `--version`    | バージョン情報を表示（新リリースが存在する場合は更新案内を表示）。                                                                                                     |
| `--no-update-check`  | 新リリースの自動検知・更新チェックを無効化（環境変数 `IHD_NO_UPDATE_CHECK=1` でも設定可能）。                                                                          |

---

### 実演ショウケースと実践レシピ集（Exam & Cookbook）

`ihd` を使って、高校数学（数I・A・II・B・III・C）、難関大入試、大学教養〜理工系専門数学（線形代数・多変数微積分・高等整数論）、および数学オリンピック等の代表的な24問を実際に解く実演デモおよびレシピ集が `exam/` ディレクトリに用意されています。

#### 自動実演スクリプトの実行
端末で以下のコマンドを実行すると、全24問を自動で解き進めるデモショウケースが始まります（途中式 `--explain` も表示されます）。

**Windows (PowerShell):**
```powershell
pwsh exam/exam.ps1
```

**Linux / macOS (Bash):**
```bash
./exam/exam.sh
```

#### 実践レシピ集ドキュメント
各問題の数式、解法、`ihd` コマンド、および出力結果の解説は以下を参照してください。
- [日本語版レシピ集 (exam/problems.ja.md)](./exam/problems.ja.md)
- [英語版レシピ集 (exam/problems.en.md)](./exam/problems.en.md)

---

### LICENSE
[MIT License](./LICENSE.MIT)

---

## English

### Why ihd? (Total Elimination of Decimals)

Standard calculators and programming languages rely on `float64` floating-point representations, inevitably leading to rounding errors and truncating irrational or transcendental numbers into arbitrary decimals.

`ihd` (i-hate-decimal-calc) is a command-line Computer Algebra System (CAS) calculator built on the strict philosophy of **"Never tolerate decimals."** All decimal inputs are immediately converted into exact arbitrary-precision rational fractions (`big.Rat`) during lexical analysis. Radicals, trigonometric values, logarithms, transcendental constants, and imaginary units are maintained as symbolic AST nodes and simplified algebraically without precision loss.

| Input Expression      | Typical Calculators / Languages (`float64`) | `ihd` (Exact Calculation) | Simplification Behavior                                             |
| :-------------------- | :------------------------------------------ | :------------------------ | :------------------------------------------------------------------ |
| `0.1 + 0.2`           | `0.30000000000000004`                       | `3/10`                    | Zero-error exact rational fraction                                  |
| `0.(3)`               | `0.3333333333333333`                        | `1/3`                     | FSM lexical analysis converts repeating decimals to exact fractions |
| `sqrt(8)`             | `2.8284271247461903`                        | `2*√2`                    | Automatic extraction of perfect squares                             |
| `sqrt(5 + 2*sqrt(6))` | `3.1462643699419726`                        | `√2 + √3`                 | Automatic radical denesting via Borodin (1985) algorithm            |
| `1 / (sqrt(2) + 1)`   | `0.4142135623730951`                        | `-1 + √2`                 | Binomial conjugate radical rationalization                          |
| `sin(pi/6)`           | `0.49999999999999994`                       | `1/2`                     | Exact algebraic evaluation of special angles                        |
| `１＋２×３`           | Error (Unsupported)                         | `7`                       | Automatic normalization of Zenkaku characters to ASCII              |

---

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
go build -o ihd ./cmd/ihd
```

---

### Quick Start & Execution Modes

#### 1. One-shot Calculation
Provide expressions directly as command-line arguments for instant exact evaluation:

```bash
ihd "1/2 + 1/3"
# Output: 5/6

ihd "sqrt(8) + sqrt(18)"
# Output: 5*√2

ihd "0.(142857)"
# Output: 1/7 (Exact repeating decimal fraction)

ihd "abs(3 + 4*i)"
# Output: 5 (Complex modulus)

ihd "root_count(x^3 - 3*x + 1, x)"
# Output: 3 (Exact real root count via Sturm's theorem)

ihd "isolate_roots(x^2 - 2, x)"
# Output: [[-3/2, -3/4], [3/4, 3/2]] (Real root isolation into disjoint rational intervals)
```

#### 2. Interactive REPL
Running without arguments launches the interactive REPL with history navigation, cursor movement, and full IME support:

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
ihd> 2*x - x
1 + √2
ihd> vars
ans = 1 + √2
x = 1 + √2
ihd> exit
Goodbye.
```

#### 3. Piped Input (Batch & Script Integration)
Process expressions sequentially from standard input. Variable states persist across lines within the same piped session, while blank lines and comments starting with `#` are ignored:

```bash
echo "sin(pi/6)^2 + cos(pi/6)^2" | ihd
# Output: 1

printf "x = 1/2 + sqrt(2)\nx * 2\n" | ihd
# Output:
# 1/2 + √2
# 1 + 2*√2
```

#### 4. Script File Batch Execution (`ihd run <file.ihd>`)
Execute `.ihd` text files containing multi-line variable definitions and calculations. Lines ending with a semicolon `;` suppress output display while still persisting variables (standard in CAS environments like MATLAB / Julia). Runtime and domain errors report exact file names and line numbers.

```bash
# Content of sample.ihd:
# a = 1/2;   # Trailing semicolon suppresses line output
# b = sqrt(8);
# a + 1
# b * 3

ihd run sample.ihd
# Or direct file invocation:
ihd sample.ihd

# Output:
# 3/2
# 6*√2
```

#### 5. Full-width (Zenkaku) Normalization
Automatically normalizes full-width digits, letters, operators (`＋`, `−`, `×`, `÷`, `＾`, `！`, `＝`), parentheses, and spaces to half-width ASCII across all execution modes:

```bash
ihd "１＋２×３"
# Output: 7
```

---

### Visual Features Showcase

#### 1. Step-by-Step Derivation Trees (`--explain`)
Displays detailed educational step-by-step algebraic rewriting trees (rationalization, radical denesting, quadratic formula, differentiation rules, etc.):

```bash
ihd --explain "sqrt(5 + 2*sqrt(6))"
```
```text
式: sqrt(5 + 2*sqrt(6))
├── [Step 1: 二重根号の簡約]
│   Borodinアルゴリズムによる二重根号の簡約
│   √(5 + 2*√6)  ──>  √2 + √3
└── [Result]
    = √2 + √3
```

#### 2. Terminal 2D Pretty Printing (`--pretty`)
Renders fractions and square roots in multi-line ASCII/Unicode box-model typography:

```bash
ihd --pretty "1/2 + sqrt(2)/2"
```
```text
 1     √2 
--- + ----
 2     2  
```

#### 3. Unicode Braille High-Resolution Plotter (`plot`)
Renders smooth function graphs directly in your terminal using 2×4 Braille subpixel characters. Automatically detects roots and extrema with exact algebraic coordinate pinning, guarded against false asymptotic connections by Tupper (2001) discontinuity detection:

```bash
ihd "plot(sin(x), [-pi, pi])"
```
```text
    1 ┼                          │                        
      │                          │        ⣀⠤⠔⠒⠢⠤⡀         
      │                          │      ⡠⠊      ⠈⠑⢄       
      │                          │    ⡠⠊           ⠑⡄     
      │                          │  ⢀⠔⠁             ⠈⢢    
      │                          │ ⡀⠎                 ⠱⡀  
      │                          │⡠⠃                   ⠘⡄ 
      │                          ⡰⠁                     ⠈⢆
      │ ⠑⡄──────────────────────⡜┼────────────────────────
      │  ⠈⢆                   ⢀⠎ │                        
      │   ⠈⢢                 ⡠⠃  │                        
      │     ⠱⡀              ⡔⠁   │                        
      │      ⠈⢆           ⡠⠊     │                        
      │        ⠑⢄⡀      ⡠⠊       │                        
      │          ⠈⠒⠢⠤⠔⠒⠉         │                        
   -1 ┼                          │                        
      └──────────────────────────────────────────────────── x
       -π                                             π

[CAS Features Detected]
* No singularities or extrema (smooth monotonic curve)
* Domain:            x ∈ [-π, π], y ∈ [-1, 1]
```

---

### Syntax & Functions Reference

`ihd` is equipped with over 50 exact symbolic functions covering university-level STEM domains including calculus, linear algebra, ODEs, Laplace transforms, algebraic elimination, and discrete probability.

> [!TIP]
> For complete operator precedence, constants, and exhaustive function specifications with mathematical examples:  
> **[Complete Syntax, Operators & Functions Reference (English Edition)](docs/references/reference.en.md)**

#### Major Categories & Representative Functions
- **Basic Algebra, Number Theory & Equations**: `sqrt`, `cbrt`, `factor`, `apart`, `together`, `poly_gcd`, `poly_lcm`, `resultant`, `groebner`, `to_poly`, `to_alg`, `alg_inv`, `min_poly`, `crt`, `solve`
- **Trigonometric, Logarithmic & Complex Functions**: `sin`, `cos`, `tan`, `trig_expand`, `trig_reduce`, `log`, `ln`, `polar`, `rect`
- **Calculus, ODEs, Special Functions & Series**: `diff`, `integrate`, `risch_integrate`, `limit`, `residue`, `gamma`, `beta`, `bernoulli`, `zeta`, `dsolve`, `laplace`, `inv_laplace`, `taylor`, `fourier_series`, `sum`, `gosper_sum`, `wz_cert`
- **Linear Algebra & 3D Vector Calculus**: `det`, `inv`, `rref`, `rank`, `trace`, `eigenvals`, `eigenvects`, `lu`, `qr`, `cholesky`, `ldlt`, `pinv`, `solve_linear`, `dot`, `cross`, `norm`, `grad`, `div`, `curl`
- **Computational Geometry**: `line_intersect`, `circle_intersect`, `triangle_area`, `triangle_centers`
- **Exact Discrete Probability & Statistics**: `binom`, `hyper`, `geom`, `bayes`, `expect`, `variance`, `stddev`
- **Symbolic Assumptions System**: `assume`, `unassume`, `assumptions`

---

### Command-Line Options

| Option               | Description & Examples                                                                                                                            |
| :------------------- | :------------------------------------------------------------------------------------------------------------------------------------------------ |
| `--ascii`            | Output using standard ASCII strings (`sqrt`, `pi`) instead of Unicode symbols.<br>`ihd --ascii "sqrt(2) + pi"` $\to$ `sqrt(2) + pi`               |
| `--approx`           | Display approximate floating-point decimal value alongside the exact form.<br>`ihd --approx "sqrt(2)"` $\to$ `√2 (≈ 1.4142135623730951)`          |
| `--latex`            | Output expression in LaTeX format (`$$ ... $$`) ready to paste into papers.<br>`ihd --latex "1/2 + sqrt(2)"` $\to$ `$$ \frac{1}{2} + \sqrt{2} $$` |
| `--pretty`           | Output expression in multi-line 2D pretty-printed Unicode formatting.<br>`ihd --pretty "1/2 + sqrt(2)/2"`                                         |
| `--deg`              | Evaluate trigonometric and inverse trigonometric functions in degrees.<br>`ihd --deg "sin(30) + cos(60)"` $\to$ `1`                               |
| `--explain`          | Step-by-step educational explanations of algebraic derivations in a 2D tree.<br>`ihd --explain "1 / (sqrt(2) + 1)"`                               |
| `--lang <code/auto>` | Specify display language (`ja`, `en`, `auto`). Persisted to `~/.ihd/config.json`.<br>`ihd --lang en "1/2 + 1/3"`                                  |
| `-h`, `--help`       | Display command help message.                                                                                                                     |
| `-v`, `--version`    | Display version information (checks for newer releases if available).                                                                             |
| `--no-update-check`  | Disable checking for newer releases (can also be disabled via `IHD_NO_UPDATE_CHECK=1`).                                                           |

---

### Mathematical Exam & Cookbook Showcase

A live mathematical showcase and practical problem cookbook solving 24 benchmark problems (high school math, university STEM, and competitions) are provided in the `exam/` directory.

#### Running the Live Showcase
Run the showcase script in your terminal to see `ihd` automatically launch and solve all 24 problems with step-by-step `--explain` derivations:

**Windows (PowerShell):**
```powershell
pwsh exam/exam.ps1
```

**Linux / macOS (Bash):**
```bash
./exam/exam.sh
```

#### Practical Cookbook Recipes
For complete problem statements, derivations, `ihd` input commands, and outputs, see:
- [Japanese Edition Cookbook (exam/problems.ja.md)](./exam/problems.ja.md)
- [English Edition Cookbook (exam/problems.en.md)](./exam/problems.en.md)

---

### LICENSE
[MIT License](./LICENSE.MIT)
