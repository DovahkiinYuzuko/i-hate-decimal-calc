# i-hate-decimal-calc / ihd

小数を一切許さず、すべての入力を即座に分数化し、平方根や定数をシンボルのまま厳密計算する電卓CLIツール / A strict CAS calculator CLI that completely rejects decimal approximations, converting all decimals to exact fractions and maintaining exact algebraic forms.

![Go](https://img.shields.io/badge/Go-1.26.5-00ADD8?style=flat-square&logo=go&logoColor=white)
[![Lean 4 Verification](https://github.com/DovahkiinYuzuko/i-hate-decimal-calc/actions/workflows/lean-verify.yml/badge.svg)](https://github.com/DovahkiinYuzuko/i-hate-decimal-calc/actions/workflows/lean-verify.yml)
[![Typst Math Verification](https://github.com/DovahkiinYuzuko/i-hate-decimal-calc/actions/workflows/typst-verify.yml/badge.svg)](https://github.com/DovahkiinYuzuko/i-hate-decimal-calc/actions/workflows/typst-verify.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=flat-square&logo=opensourceinitiative&logoColor=white)](./LICENSE.MIT)

[日本語](#日本語) | [English](#english)

---

## 日本語

### なぜ ihd なのか？（小数の排除と厳密代数計算）

一般的なプログラミング言語や関数電卓では、小数を扱う際に `float64` などの浮動小数点数を使用するため、不可避の丸め誤差が生じます。また、無理数や超越数も適当な桁数で丸められてしまいます。

`ihd`（i-hate-decimal-calc）は、**「小数の存在を一切許さない」** という強い思想のもと設計された、コマンドライン向けの厳密数式処理システム（CAS）電卓です。入力された小数は字句解析の段階で即座に任意精度有理数（`big.Rat`）へ変換され、平方根・三角関数・対数・超越数・虚数単位はシンボルノードとして保持され、代数的簡約ルールによって厳密な形式のまま計算されます。

| 入力数式              | 一般的な電卓・言語（浮動小数点数） | `ihd`（厳密計算） | 簡約・処理内容                               |
| :-------------------- | :--------------------------------- | :---------------- | :------------------------------------------- |
| `0.1 + 0.2`           | `0.30000000000000004`              | `3/10`            | 誤差ゼロの有理数約分                         |
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
Go 1.26.5 がインストールされている環境で、リポジトリルートにて `go build -o ihd ./cmd/ihd` を実行してビルドします（開発者向けツールや検証手順は以下の折りたたみセクションを参照してください）。

<details>
<summary><b>開発者向けツール & 検証（独立検証器 <code>micro-verifier</code> / 数学的不変条件ファジング）</b></summary>

##### 1. 独立マイクロ検証器（`tools/micro-verifier`）
`ihd --certificate cert.json "..."` で発行された計算証明書（`ihd-proof-trace-v1`）を、`ihd` 本体のソルバロジックに依存せず独立して第三者検証するための軽量CLIツールです。

```bash
# 独立検証器のビルドと証明書の検証
go build -o micro-verifier ./tools/micro-verifier
./micro-verifier cert.json
# または go run で直接実行
go run ./tools/micro-verifier cert.json
```

##### 2. 数学的不変条件 Property-Based Testing & Fuzzing
`internal/calc` には、構造認識型AST生成器（Structure-Aware AST Generator）とガウス有理数体 $\mathbb{Q}[i]$ 上の厳密多点評価オラクル（Schwartz-Zippel補題）を組み合わせたファジングハーネスが組み込まれています。

```bash
# 全シードコーパスおよびオラクル単体テストの一括スモーク実行
go test -v ./internal/calc -run "^(Fuzz|TestOracle|TestProvider)"

# 特定の数学的不変条件ターゲットに対する連続ファジング探索（例: 30秒間）
go test ./internal/calc -run "^FuzzAlgebra_SimplifyIdempotence$" -fuzz "^FuzzAlgebra_SimplifyIdempotence$" -fuzztime 30s
```

主なファジングターゲット:
- `FuzzAlgebra_SimplifyIdempotence` / `FuzzAlgebra_ExpandFactorRoundTrip` / `FuzzAlgebra_GroebnerIdealMembership`
- `FuzzCalculus_IntegrateDiffRoundTrip` / `FuzzCalculus_DiffLeibnizLinearity` / `FuzzCalculus_ODESubstitution` / `FuzzCalculus_RSolveInvariance`
- `FuzzDomain_UnitsDimensionalConsistency` / `FuzzDomain_MatrixInvariants` / `FuzzDomain_SimplexDualityAndFarkas`

</details>

```bash
go build -o ihd ./cmd/ihd
```

---

### クイックスタート & 実行形態

#### 1. ワンショット計算（CLI引数）
コマンドライン引数に数式を渡すだけで即座に厳密計算を実行します。

```bash
# 基本有理数演算
ihd "1/2 + 1/3"
# 出力: 5/6

# 平方根の自動簡約と無理数同値性判定
ihd "sqrt(2) + sqrt(3) == sqrt(5 + 2*sqrt(6))"
# 出力: true（多重根号・無理数同値性の完全代数的判定）

# 実代数不等式の厳密解（CAD / Sturm根分離）
ihd "solve(x^2 - 4 < 0)"
# 出力: [[-2, 2]]
```

<details>
<summary><b>高等数学・専門分野のワンショット計算例（超越方程式・特殊積分・幾何自動証明・群論など）</b></summary>

```bash
# 初等超越方程式の閉形式求解（Lambert W関数）
ihd "solve(x*exp(x) == 1, x)"
# 出力: [lambert_w(1)]

# ガウス積分の初等閉形式化（誤差関数 erf）
ihd "integrate(exp(-x^2), x)"
# 出力: √pi/2 * erf(x)

# 初等幾何学の完全自動証明（呉の方法）
ihd "geo_prove([midpoint(M, A, B), midpoint(N, A, C)], parallel(M, N, B, C))"
# 出力: true（中点連結定理等の自動証明）

# ガロア群算定とアーベル・ルフィニ代数的不可解性証明書
ihd --verify "is_solvable_by_radicals(x^5 - 4*x + 2)"
# 出力: false（代数的不可解性証明書 [VERIFIED: expr == false] を発行）

# 素有限体上の楕円曲線位数決定（Schoof法）
ihd "ec_p_order(1, 1, 5)"
# 出力: 9

# 整数行列標準形と有限生成アーベル群構造同定（HNF / SNF）
ihd "abelian_group_structure([[2, 4], [4, 2]])"
# 出力: Z_2 + Z_6
```

</details>


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

#### 1. 代数変換プロセスを可視化する途中式ツリー & 4大プロジェクション（`--explain`）
ファイングレイン証明DAG（統一プロベナンスIR）に基づき、分母の有理化や二重根号外し、多項式因数分解、微積分、極限計算（ロピタル）、方程式求解の適用過程を詳細に出力します（REPLやパイプでも `explain <式>` または `steps <式>` で実行可能）。さらに、単一の導出ツリーから **Typst ドキュメント生成（`--typst`）** や **Lean 4 等式連鎖証明（`--lean`）** へのシームレスな射影（Projection）が可能です。

<details>
<summary><b>途中式ツリー実行例と出力（Unicode 2Dツリー / Typst / Lean 4 calc）</b></summary>

**基本途中式ツリー表示:**
```bash
ihd --explain "limit((sqrt(1 + x) - 1 - x/2) / x^2, x, 0)"
```

```text
式: limit((sqrt(1 + x) - 1 - x/2) / x^2, x, 0)
├── [Step 1: べき乗の微分公式の適用]
│   べき乗の微分公式: d/dx [x^2] = 2*x
│   x^2  ──>  2*x
│
├── [Step 2: 極限値計算・不定形の代数解消]
│   ロピタルの定理適用: (d/dx [-1 + sqrt(1 + x) + -x/2]) / (d/dx [x^2]) = (-1/2 + 1/(2 * sqrt(1 + x))) / (2 * x)
│   -x^2^-1 - 1/2*x*x^2^-1 + √(1 + x)*x^2^-1  ──>  -1/(4*x) + 1/4*√(1 + x)^-1*x^-1
│
├── [Step 3: べき乗の微分公式の適用]
│   べき乗の微分公式: d/dx [√(1 + x)^-1] = -√(1 + x)^-3/2
│   √(1 + x)^-1  ──>  -√(1 + x)^-3/2
│
├── [Step 4: 極限値計算・不定形の代数解消]
│   ロピタルの定理適用: (d/dx [-1/2 + 1/(2 * sqrt(1 + x))]) / (d/dx [2 * x]) = (-1/4 * sqrt(1 + x)^-3) / (2)
│   -1/(4*x) + 1/4*√(1 + x)^-1*x^-1  ──>  -√(1 + x)^-3/8
│
├── [Step 5: 極限値計算・不定形の代数解消]
│   極限値計算: lim_{x->0} -1 * x^2^-1 + sqrt(1 + x) * x^2^-1 + -1/2 * x * x^2^-1 = -1/8
│   -x^2^-1 - 1/2*x*x^2^-1 + √(1 + x)*x^2^-1  ──>  -1/8
└── [Result]
    = -1/8
```

**Typst 形式の数学導出レポート出力:**
```bash
ihd --explain --typst "factor(x^4 + 4)"
```

**Lean 4 Mathlib4 形式証明（推移的等式連鎖 `calc` ブロック）出力:**
```bash
ihd --explain --lean "factor(x^4 + 4)"
```

```lean
-- Automatically generated by ihd (i-hate-decimal-calc) Lean 4 Transpiler
-- Mathematical proof certificate certified with Skeptic's verification
import Mathlib.Tactic.Ring

variable (x : ℚ)

theorem ihd_certified_derivation : x ^ 4 + 4 = (x ^ 2 + (-2) * x + 2) * (x ^ 2 + 2 * x + 2) := by
  calc
    x ^ 4 + 4 = (x ^ 2 + (-2) * x + 2) * (x ^ 2 + 2 * x + 2) := by ring

#print axioms ihd_certified_derivation
```

</details>

#### 2. ターミナル2D数式組版プリティプリント（`--pretty`）
分数や平方根を複数行のアスキーアートボックスモデルで描画します。

<details>
<summary><b>ボックスモデル組版実行例と出力</b></summary>

```bash
ihd --pretty "1/2 + sqrt(2)/2"
```

```text
 1     √2 
--- + ----
 2     2  
```

</details>

#### 3. Braille 2×4 サブピクセル高解像度プロッタ（`plot`）
Unicode点字文字を用いた 2×4 サブピクセル描画により、ターミナル上に関数の形状をプロットします。CASエンジンが零点や極値を自動検出し、厳密代数ラベルを軸上にピン留めします。さらに Tupper (2001) の不連続判定により漸近線の偽結合を防止します。

<details>
<summary><b>ターミナルプロット描画実行例と出力（アスキーアート）</b></summary>

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

</details>

#### 4. REPL リアルタイム引数ガイド & 構文ハイライト
外部依存ライブラリを使用せず、単一レキシカル有限状態機械（FSM）により入力文字列の色分け表示およびリアルタイムな引数案内を行います。
- **リアルタイム引数ガイド & シグネチャヘルパー**: 関数入力中、現在カーソルがある引数を強調表示し、プロンプト直下に引数名・多言語解説を動的に表示します。
- **ネスト深度別レインボー括弧**: 括弧 `()` および角括弧 `[]` のネスト深度（4色循環）に応じたハイライト。括弧の不整合（ミスマッチ）はエラー警告色（赤背景）で表示。
- **カスタム変数ハイライト**: 未束縛の記号（$x, y$ 等）は標準色を維持し、現在のセッション環境に定義された変数のみを識別して着色（Darkテーマ時は淡いシアン）。
- **テーマ設定 & 永続化**: `dark`（暗め背景向け）および `light`（明るめ背景向け）テーマに対応。REPL 内コマンド（`theme dark` / `theme light`）で切り替え可能で、設定は `~/.ihd/config.json` に保存。

#### 5. 反復ループの非破壊安全性 & ユーザー主権制御（`--warn-loop` / `--max-loop`）
CAS反復構文（`table`, `for`, `product`）において、長時間の計算ループによる意図しない暴走を防止しつつ、大量サンプリングやモンテカルロ計算を阻害しないハイブリッド安全機構を搭載しています。
- **非破壊警告**: 警告閾値（デフォルト: 100,000ステップ）を超過した際、パイプライン（標準出力）を汚染せず `os.Stderr` へ一度だけ警告を発出し、計算を中断せず続行します。
- **最大安全上限ガード**: 指定した最大上限（デフォルト: 無制限）に到達した場合は安全に強制遮断します。
- **任意精度（`math/big.Int`）管理**: 32bit/64bit 環境での桁あふれや巨大数入力によるオーバーフローを防止します。
- **CLIワンショット & REPL対話コマンド**: ワンショット実行時は `--warn-loop <N>`, `--max-loop <N>` フラグで指定可能。REPL内では `warn-loop <N>`, `max-loop <N>`, `config loop` コマンドで設定確認・変更・永続化（`~/.ihd/config.json`）が可能です。

#### 6. Typst 数式エクスポート（`--typst` / `--format typst`）
Markdown や Typst ドキュメントへ貼り付け可能な Typst 数式モード記法（`$ ... $`）で出力します。分数・平方根・n乗根・行列・複素数・多項式などを Typst の簡潔な構文に正準変換します。

<details>
<summary><b>Typst 数式エクスポート実行例</b></summary>

```bash
ihd --typst "1/2 + sqrt(2)/2"
# 出力: $1 / 2 + sqrt(2) / 2$

ihd --format typst "[[1, 2], [3, 4]]"
# 出力: $mat(1, 2; 3, 4)$
```

</details>

エクスポートされる数式は GitHub Actions CI 上で Typst コンパイラにより自動コンパイル検証されています。

#### 7. コレクション・リスト操作 & 第一級ラムダ式
Python スタイルのスライス構文（`[start:end:step]`）、有理数刻み数列生成 `range`、第一級無名関数 `lambda`、および高階関数 `map`, `filter` をサポートしています。

<details>
<summary><b>コレクション・リスト操作 & ラムダ式実行例</b></summary>

```bash
ihd "map(lambda(x, x^2), range(1, 5))"
# 出力: [1, 4, 9, 16]

ihd "[10, 20, 30, 40, 50][1:4]"
# 出力: [20, 30, 40]
```

</details>

#### 8. 厳密有理数ベクトル空間による物理単位計算 & 次元解析
有理数体 $\mathbb{Q}$ 上の 7 次元ベクトル空間（SI基本7単位）に基づき、小数を一切使わずに物理単位の加減乗除、組立単位・接頭辞の厳密展開、次元整合性検証、および単位換算（`convert_unit`）を実行します。初等超越関数への有次元引数の投入は厳格に遮断され、無次元化された結果は自動的に通常数値へと縮約されます。

<details>
<summary><b>物理単位計算 & 次元解析実行例</b></summary>

```bash
# 単位の乗除算と約分
ihd "10 [m/s] * 5 [s]"
# 出力: 50 [m]

# 接頭辞と組立単位を含む減算（スケール正規化）
ihd "1 [km] - 200 [m]"
# 出力: 4/5 [km]

# 厳密分数による単位換算
ihd "convert_unit((1/4) [h], min)"
# 出力: 15 [min]

# 独立証明書付き単位換算自己検証
ihd --verify "convert_unit(100 [km/h], m/s)"
# 出力: 250/9 [m/s]
# [VERIFIED: convert_unit(100[km * h^-1], m * s^-1) == 250/9 [m/s]]
```

</details>

---

### 構文・演算子・関数リファレンス

`ihd` は微積分・線形代数・常微分方程式・ラプラス変換・多変数代数消去法・離散確率など、大学理工系レベルを網羅する 50 種類以上の組み込み関数を搭載しています。

> [!TIP]
> 演算子と優先順位、予約定数、およびカテゴリ別全関数の詳細仕様・書式・使用例は専用リファレンスに網羅されています。  
> - **Markdown版**: [日本語版 (docs/references/ja/reference.ja.md)](docs/references/ja/reference.ja.md) / [英語版 (docs/references/en/reference.en.md)](docs/references/en/reference.en.md)  
> - **オフラインSingle-File HTML版**: [日本語版 (docs/references/ja/reference.ja.html)](docs/references/ja/reference.ja.html) / [英語版 (docs/references/en/reference.en.html)](docs/references/en/reference.en.html)  
> - *※ HTML版の再ビルド手順やツールアーキテクチャの詳細は [tools/reference-builder/README.md](tools/reference-builder/README.md) を参照してください。*

#### 主なカテゴリと代表関数
- **基本代数・数論・方程式**: `sqrt`, `cbrt`, `expand`, `factor`, `apart`, `together`, `abs`, `gcd`, `lcm`, `mod`, `inv_mod`, `crt`, `totient`, `is_prime`, `cfrac`, `from_cfrac`, `rand`, `fib`, `fibonacci`, `digit_count`, `poly_gcd`, `poly_lcm`, `resultant`, `groebner`, `sturm`, `root_count`, `isolate_roots`, `to_poly`, `to_alg`, `alg_inv`, `min_poly`, `find_min_poly`, `to_primitive_element`, `primitive_element`, `padic_val`, `padic_norm`, `padic_expand`, `solve`, `solve_diophantine`, `pell`, `pell_solve`, `interval`
- **三角関数・対数・複素数**: `sin`, `cos`, `tan`, `asin`, `acos`, `atan`, `trig_expand`, `trig_reduce`, `exp`, `log`, `ln`, `arg`, `polar`, `polar_exp`, `rect`
- **微積分・常微分方程式・漸化式・特殊関数・級数**: `diff`, `integrate`, `piecewise`, `risch_integrate`, `limit`, `residue`, `lambert_w`, `erf`, `gamma`, `beta`, `bernoulli`, `zeta`, `delta`, `u`, `dsolve`, `rsolve`, `laplace`, `inv_laplace`, `ztrans`, `iztrans`, `taylor`, `fourier_series`, `sum`, `gosper_sum`, `wz_cert`, `puiseux`
- **線形代数・線形計画・整数標準形・3次元ベクトル解析**: `det`, `inv`, `transpose`, `rref`, `rank`, `trace`, `tr`, `eigenvals`, `eigenvects`, `lu`, `qr`, `cholesky`, `ldlt`, `pinv`, `solve_linear`, `linsolve`, `lp_maximize`, `lp_minimize`, `hnf`, `hnf_transform`, `snf`, `snf_transform`, `invariant_factors`, `abelian_group_structure`, `lll`, `dot`, `cross`, `norm`, `grad`, `div`, `curl`
- **幾何学解析**: `line_intersect`, `circle_intersect`, `triangle_area`, `triangle_centers`, `geo_prove`, `collinear`, `midpoint`, `parallel`, `perpendicular`, `equal_length`, `equal_length_sq`, `circle_concyclic`
- **厳密離散確率・統計**: `binom`, `comb`, `perm`, `hyper`, `geom`, `bayes`, `expect`, `variance`, `stddev`
- **実代数幾何・数理論理・量化子消去**: `qe`, `forall`, `exists`, `cad`
- **ガロア理論・代数的根号可解性**: `galois_group`, `is_solvable_by_radicals`
- **楕円曲線代数・数論幾何・2降下法**: `ec_add`, `ec_mul`, `ec_torsion`, `ec_p_order`, `ec_p_trace`, `ec_p_add`, `ec_p_mul`, `ec_rank_bound`, `ec_generators`
- **クリフォード代数・幾何代数 & 厳密量子回路等価性検証**: `clifford`, `clifford_wedge`, `clifford_contract`, `clifford_dual`, `quantum_equiv`, `quantum_eval`
- **多変数多項式イデアル・可換代数**: `primary_decomposition`, `ideal_radical`, `ideal_dimension`, `ideal_intersection`, `ideal_colon`, `ideal_saturation`
- **コレクション・リスト操作 & CAS反復制御**: `range`, `map`, `filter`, `lambda`, `table`, `product`, `for`, `to_base`, `from_base`, `bin`, `oct`, `hex`
- **物理単位計算・次元解析**: `convert_unit`, `unit`
- **前提条件システム（仮定）**: `assume`, `unassume`, `assumptions`, `clear_assumptions`
- **ビジュアル・自己検証**: `plot`, `verify`

> [!NOTE]
> **代数・超越方程式ソルバー、線形代数、楕円曲線2降下法、およびクリフォード代数・量子回路検証**  
> `solve` は1次・2次・3次・4次（Ferrari法）の代数方程式の根号による閉形式解に加え、ランバートのW関数（`lambert_w`）を用いた初等超越方程式（$x e^x = c$, $x + \ln x = c$, $x^x = c$ 等）の閉形式求解をサポートしています。また、`integrate` はガウス積分 $\int e^{-a x^2} dx$ の誤差関数 `erf` による閉形式化を完備し、線形代数の `eigenvals` は最大 4×4 行列の特性方程式から厳密固有値を導出可能です。さらに、`clifford` は直交対角計量 $(p, q, r)$ の幾何積・外積・縮約・双対ブレードを代数展開し、`quantum_equiv` は小数を一切排除した二進円分環 $\mathbb{Z}[1/\sqrt{2}, i]$ 上で Clifford+T 量子回路の厳密等価性判定および反例基底状態の抽出を実行します。有理数体上の楕円曲線に対しては、有理2等分点を用いた2降下法（2-isogeny descent）によりセルマー群の $\mathbb{F}_2$ 次元を解析し、モーデル・ヴェイユ群の階数（Rank）上界（`ec_rank_bound`）および無限位数の独立有理点生成元（`ec_generators`）の決定論的算定を提供します。

---

### コマンドラインオプション

| オプション           | 説明・使用例                                                                                                                                                           |
| :------------------- | :--------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `--theme <dark\|light\|none>` | REPLのカラーテーマを指定（`dark`: 暗め背景、`light`: 明るめ背景、`none`: ハイライト無効）。設定は `~/.ihd/config.json` に永続化。<br>`ihd --theme light`                 |
| `--no-color`         | ANSIエスケープシーケンスによるカラー出力を無効化（環境変数 `NO_COLOR=1` にも準拠）。                                                                                 |
| `--warn-loop <N>`    | 反復構文（`table`, `for`, `product`）の非破壊警告ステップ数を指定（デフォルト: 100000）。<br>`ihd --warn-loop 500 "table(k, k, 1, 1000)"`                                    |
| `--max-loop <N>`     | 反復構文の最大安全上限ステップ数を指定（超過時はエラー終了、デフォルト: -1:無制限）。<br>`ihd --max-loop 10000 "table(k, k, 1, 20000)"`                                |
| `--ascii`            | 数学記号（`√`, `π`）をASCII文字列（`sqrt`, `pi`）にフォールバックして出力。<br>`ihd --ascii "sqrt(2) + pi"` $\to$ `sqrt(2) + pi`                                       |
| `--approx`           | 厳密解の横に参考用の浮動小数点小数近似値（float64、15〜17桁）を併記。<br>`ihd --approx "sqrt(2)"` $\to$ `√2 (≈ 1.4142135623730951)`                                    |
| `--latex`            | MarkdownやTeX論文に貼り付け可能なLaTeX形式（`$$ ... $$`）で出力。<br>`ihd --latex "1/2 + sqrt(2)"` $\to$ `$$ \frac{1}{2} + \sqrt{2} $$`                                |
| `--typst`            | 組版システム Typst の数式モード記法（`$ ... $`）で出力。<br>`ihd --typst "1/2 + sqrt(2)"` $\to$ `$1 / 2 + sqrt(2)$`                                                   |
| `--format <fmt>`     | 出力フォーマットを指定（`typst`, `latex`, `ascii`, `pretty`）。<br>`ihd --format typst "[[1, 2], [3, 4]]"` $\to$ `$mat(1, 2; 3, 4)$`                                       |
| `--pretty`           | 分数線や根号を複数行アスキーアートで組版表示する2Dプリティプリント。<br>`ihd --pretty "1/2 + sqrt(2)/2"`                                                               |
| `--deg`              | 三角関数および逆三角関数を度数法（Degree）として解釈・計算。<br>`ihd --deg "sin(30) + cos(60)"` $\to$ `1`                                                              |
| `--explain`          | 計算過程（有理化・二重根号・微分則・行列分解等）をツリー形式の途中式として表示。<br>`ihd --explain "1 / (sqrt(2) + 1)"` |
| `--verify`           | 計算結果（因数分解、逆行列、微積分、ODE、幾何証明等）に対し、定義式や導関数に基づく逆算検算を行い、残差が0であることを確認。<br>`ihd --verify "integrate(1/(x^2 + 1), x)"` $\to$ `atan(x)`<br>`[VERIFIED: diff(F, x) - f == 0]` |
| `--certificate [file]` | 計算結果と導出ステップ列を証明書 JSON（`ihd-proof-trace-v1`）として出力またはファイルへ保存。<br>`ihd --certificate "factor(x^2 - 1)"`<br>`ihd --certificate cert.json "factor(x^2 - 1)"` |
| `--replay <file>`    | 保存された証明書 JSON ファイルを読み込み、各ステップの残差が0かを再検証（本体から分離された独立検証器 `go run ./tools/micro-verifier <file>` でも検証可能）。<br>`ihd --replay cert.json` |
| `-i`, `--interval`   | 適応的区間細分化（Rational Interval Arithmetic）による包含区間を算出して出力。<br>`ihd -i "sin(1)"`                                                                  |
| `--interval-eps <eps>` | 区間細分化の収束判定許容誤差を指定（有理数または小数、デフォルト: `1/1000000`）。<br>`ihd -i --interval-eps "1/1000000000" "sin(1)"`                                  |
| `--lean`             | 計算結果や導出ステップ列を定理証明支援系 **Lean 4（Mathlib4）** の形式証明コード（`by ring`, `by ext <;> ring`, `by repeat constructor <;> norm_num` 等）として出力。<br>`ihd --lean "factor(x^2 - 1)"` |
| `--lean-file <path>` | 生成された Lean 4 形式証明コードをスタンドアロンな `.lean` ファイルとして指定パスへ保存。<br>`ihd --lean-file proof.lean "factor(x^2 - 1)"`                                 |
| `--lang <code/auto>` | 表示言語（ロケール）を指定（`ja`, `en`, `auto`）。設定は `~/.ihd/config.json` に永続化され、カスタム辞書（`~/.ihd/locales/`）にも対応。<br>`ihd --lang en "1/2 + 1/3"` |
| `completion <shell>` | 主要4大シェル（`bash`, `zsh`, `fish`, `powershell`）向けのタブ自動補完スクリプトをゼロ依存で生成。<br>`ihd completion powershell \| Out-String \| Invoke-Expression` |
| `-h`, `--help`       | コマンドのヘルプメッセージを表示。                                                                                                                                     |
| `-v`, `--version`    | バージョン情報を表示（新リリースが存在する場合は更新案内を表示）。                                                                                                     |
| `--no-update-check`  | 新リリースの自動検知・更新チェックを無効化（環境変数 `IHD_NO_UPDATE_CHECK=1` でも設定可能）。                                                                          |

> [!NOTE]
> **Lean 4 / Mathlib4 形式証明エクスポートの仕様と対応範囲**  
> `ihd` の `--lean` オプションは、計算結果に対する証明コードを Lean 4（Mathlib4）形式で出力します。
> 
> **対応機能:**
> - 多項式の展開・因数分解・代数恒等式（`by ring`）
> - 初等関数の不定積分に対する導関数検証（`by simp`）
> - 行列の可逆性・連立1次方程式・整数標準形 HNF / SNF（`by ext <;> fin_cases`）
> - 多項式の有理根・ペル方程式の整数解（`by norm_num` / `by decide`）
> - 幾何定理の自動証明（Wuの代数消去法）
> 
> **非対応の機能と挙動:**
> - 超越関数（Lambert W、誤差関数 `erf` 等）、特殊関数、および一般の CAD / 半代数的集合の量化子消去は、自動証明生成に対応していません。
> - 未対応の構文や証明できない witness が含まれる場合、自明な命題（`True` 等）への置換は行わず、明示的なエラーを返します。
> 
> 生成された証明コードは GitHub Actions CI 上の Lean 4 環境でコンパイル検証されており、`#print axioms` において `sorry` を含まないことを確認しています。

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
[MIT](./LICENSE.MIT)

Third-Party → [NOTICE.md](./NOTICE.md)

---

## English

### Why ihd? (Elimination of Decimals & Exact Algebraic Computation)

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
Requires Go 1.26.5. Build directly from the root of the repository via `go build -o ihd ./cmd/ihd` (see the collapsible section below for standalone verification tools and developer fuzzing):

<details>
<summary><b>Developer Tools & Verification (Standalone <code>micro-verifier</code> / Mathematical Invariants Fuzzing)</b></summary>

##### 1. Standalone Micro-Verifier (`tools/micro-verifier`)
A lightweight, independent CLI tool that third-party verifies proof certificates (`ihd-proof-trace-v1`) generated via `ihd --certificate cert.json "..."` without relying on `ihd`'s internal solver logic:

```bash
# Build and run the standalone verifier
go build -o micro-verifier ./tools/micro-verifier
./micro-verifier cert.json
# Or run directly via go run
go run ./tools/micro-verifier cert.json
```

##### 2. Mathematical Invariants Property-Based Testing & Fuzzing
`internal/calc` includes a structure-aware AST generator and a Gaussian rational ($\mathbb{Q}[i]$) exact multi-point evaluation oracle (Schwartz-Zippel lemma) integrated with Go's native fuzzing engine (`testing.F`):

```bash
# Run all seed corpora and oracle unit tests (smoke test)
go test -v ./internal/calc -run "^(Fuzz|TestOracle|TestProvider)"

# Run continuous fuzzing exploration on a specific invariant target (e.g., for 30s)
go test ./internal/calc -run "^FuzzAlgebra_SimplifyIdempotence$" -fuzz "^FuzzAlgebra_SimplifyIdempotence$" -fuzztime 30s
```

Available fuzzing targets:
- `FuzzAlgebra_SimplifyIdempotence` / `FuzzAlgebra_ExpandFactorRoundTrip` / `FuzzAlgebra_GroebnerIdealMembership`
- `FuzzCalculus_IntegrateDiffRoundTrip` / `FuzzCalculus_DiffLeibnizLinearity` / `FuzzCalculus_ODESubstitution` / `FuzzCalculus_RSolveInvariance`
- `FuzzDomain_UnitsDimensionalConsistency` / `FuzzDomain_MatrixInvariants` / `FuzzDomain_SimplexDualityAndFarkas`

</details>

```bash
go build -o ihd ./cmd/ihd
```

---

### Quick Start & Execution Modes

#### 1. One-shot Calculation
Provide expressions directly as command-line arguments for instant exact evaluation:

```bash
# Exact rational arithmetic
ihd "1/2 + 1/3"
# Output: 5/6

# Exact algebraic equivalence & nested radicals
ihd "sqrt(2) + sqrt(3) == sqrt(5 + 2*sqrt(6))"
# Output: true

# Real algebraic inequality solving (CAD / Sturm root isolation)
ihd "solve(x^2 - 4 < 0)"
# Output: [[-2, 2]]
```

<details>
<summary><b>Advanced & Specialized One-Shot Examples (Transcendental equations, integration, automated geometry theorem proving, group theory, etc.)</b></summary>

```bash
# Elementary transcendental equation solving (Lambert W function)
ihd "solve(x*exp(x) == 1, x)"
# Output: [lambert_w(1)]

# Gaussian integral closed form (Error function erf)
ihd "integrate(exp(-x^2), x)"
# Output: √pi/2 * erf(x)

# Automated geometric theorem proving (Wu's method)
ihd "geo_prove([midpoint(M, A, B), midpoint(N, A, C)], parallel(M, N, B, C))"
# Output: true

# Abel-Ruffini algebraic insolvability verification
ihd --verify "is_solvable_by_radicals(x^5 - 4*x + 2)"
# Output: false (Emits Abel-Ruffini impossibility certificate [VERIFIED: expr == false])

# Elliptic curve point count over finite fields (Schoof's algorithm)
ihd "ec_p_order(1, 1, 5)"
# Output: 9

# Integer matrix normal forms and finitely generated abelian group structure (HNF / SNF)
ihd "abelian_group_structure([[2, 4], [4, 2]])"
# Output: Z_2 + Z_6
```

</details>


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

#### 1. Step-by-Step Derivation Trees & 4 Major Projections (`--explain`)
Powered by a fine-grained Single Source of Truth Derivation DAG (Provenance IR), `ihd` records each exact rewrite step across algebra, calculus, limits (L'Hopital), and solvers. From this single DAG, it can project into **Unicode 2D Trees**, **Typst Reports (`--typst`)**, or **Lean 4 Transitive `calc` Proofs (`--lean`)** with zero feature overhead.

<details>
<summary><b>Step-by-step derivation tree example & output (Unicode Tree / Typst / Lean 4 calc)</b></summary>

**Unicode Derivation Tree:**
```bash
ihd --explain "limit((sqrt(1 + x) - 1 - x/2) / x^2, x, 0)"
```

**Lean 4 Mathlib4 Transitive Equational Derivation (`calc` block):**
```bash
ihd --explain --lean "factor(x^4 + 4)"
```

```lean
-- Automatically generated by ihd (i-hate-decimal-calc) Lean 4 Transpiler
-- Mathematical proof certificate certified with Skeptic's verification
import Mathlib.Tactic.Ring

variable (x : ℚ)

theorem ihd_certified_derivation : x ^ 4 + 4 = (x ^ 2 + (-2) * x + 2) * (x ^ 2 + 2 * x + 2) := by
  calc
    x ^ 4 + 4 = (x ^ 2 + (-2) * x + 2) * (x ^ 2 + 2 * x + 2) := by ring

#print axioms ihd_certified_derivation
```

</details>

#### 2. Terminal 2D Pretty Printing (`--pretty`)
Renders fractions and square roots in multi-line ASCII/Unicode box-model typography:

<details>
<summary><b>2D box-model layout example & output</b></summary>

```bash
ihd --pretty "1/2 + sqrt(2)/2"
```

```text
 1     √2 
--- + ----
 2     2  
```

</details>

#### 3. Unicode Braille High-Resolution Plotter (`plot`)
Renders smooth function graphs directly in your terminal using 2×4 Braille subpixel characters. Automatically detects roots and extrema with exact algebraic coordinate pinning, guarded against false asymptotic connections by Tupper (2001) discontinuity detection:

<details>
<summary><b>Terminal plot rendering example & output (ASCII / Braille art)</b></summary>

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

</details>

#### 4. REPL Real-time Signature Guide & Syntax Highlighting
Lexical syntax highlighting and argument guidance powered by a zero-dependency finite-state machine (FSM):
- **Real-time Signature Guide**: As you type function calls, the REPL dynamically analyzes the active argument index and displays parameter names and localized documentation directly below the prompt.
- **Rainbow Nested Delimiters**: Delimiter depths across parentheses `()` and brackets `[]` rotate through 4 cyclic colors. Bracket mismatches are highlighted with an error background.
- **Custom Variable Highlighting**: Unbound symbols ($x, y$) retain the neutral base text color, while user-defined custom variables in the active session environment are highlighted.
- **Dual Themes & Persistence**: Supports `dark` and `light` themes. Configurable inside the REPL (`theme dark` / `theme light`), persisted to `~/.ihd/config.json`.

#### 5. Iteration Safety & Non-Destructive Loop Control (`--warn-loop` / `--max-loop`)
Protects against runaway CAS loops (`table`, `for`, `product`) without hindering heavy computations or Monte Carlo sampling:
- **Non-Destructive Warning**: Emits a single warning to `os.Stderr` when exceeding the warning threshold (default: 100,000 steps) without polluting `stdout` or breaking downstream pipes.
- **Maximum Safety Guard**: Stops execution safely when reaching the maximum iteration limit (default: unlimited).
- **Arbitrary-Precision (`math/big.Int`)**: Step counters and thresholds are managed via `big.Int` to prevent integer overflow across 32-bit and 64-bit architectures.
- **CLI Flags & REPL Commands**: Configurable per-run via `--warn-loop <N>` and `--max-loop <N>`, or interactively inside REPL via `warn-loop <N>`, `max-loop <N>`, and `config loop` commands with persistence in `~/.ihd/config.json`.

#### 6. Typst Math Export (`--typst` / `--format typst`)
Exports mathematical expressions directly into Typst math mode syntax (`$ ... $`) ready for technical documentation. Canonicalizes fractions, radicals, n-th roots, matrices, complex numbers, and polynomials into Typst-native representations.

<details>
<summary><b>Typst math export examples</b></summary>

```bash
ihd --typst "1/2 + sqrt(2)/2"
# Output: $1 / 2 + sqrt(2) / 2$

ihd --format typst "[[1, 2], [3, 4]]"
# Output: $mat(1, 2; 3, 4)$
```

</details>

Exported Typst expressions are continuously validated and compiled via Typst compiler in GitHub Actions CI.

#### 7. Collection Manipulation, Slicing & First-Class Lambda Expressions
Supports Python-style slice syntax (`[start:end:step]`), arithmetic progressions with rational step `range`, first-class anonymous functions `lambda`, and higher-order functions `map` and `filter`.

<details>
<summary><b>Collection manipulation & lambda expression examples</b></summary>

```bash
ihd "map(lambda(x, x^2), range(1, 5))"
# Output: [1, 4, 9, 16]

ihd "[10, 20, 30, 40, 50][1:4]"
# Output: [20, 30, 40]
```

</details>

#### 8. Exact Rational Vector Space Physical Units & Dimensional Analysis
Implements exact dimensional analysis over the 7-dimensional vector space $\mathbb{Q}^7$ of SI base units without floating-point approximations. Supports SI base and derived units (N, J, W, Pa, Hz, etc.), SI prefixes, dimensional consistency checks, transcendental function guards, and exact rational unit conversions (`convert_unit`):

<details>
<summary><b>Physical units & dimensional analysis examples</b></summary>

```bash
# Multiplication and unit cancellation
ihd "10 [m/s] * 5 [s]"
# Output: 50 [m]

# Prefix normalization in subtraction
ihd "1 [km] - 200 [m]"
# Output: 4/5 [km]

# Unit conversion with fractions
ihd "convert_unit((1/4) [h], min)"
# Output: 15 [min]

# Verified conversion certificate
ihd --verify "convert_unit(100 [km/h], m/s)"
# Output: 250/9 [m/s]
# [VERIFIED: convert_unit(100[km * h^-1], m * s^-1) == 250/9 [m/s]]
```

</details>

---

### Syntax & Functions Reference

`ihd` is equipped with over 50 exact symbolic functions covering university-level STEM domains including calculus, linear algebra, ODEs, Laplace transforms, algebraic elimination, and discrete probability.

> [!TIP]
> For complete operator precedence, constants, and exhaustive function specifications with mathematical examples:  
> **[Syntax, Operators & Functions Comprehensive Reference](docs/references/en/reference.en.md)**

#### Major Categories & Representative Functions
- **Basic Algebra, Number Theory & Equations**: `sqrt`, `cbrt`, `expand`, `factor`, `apart`, `together`, `abs`, `gcd`, `lcm`, `mod`, `inv_mod`, `crt`, `totient`, `is_prime`, `cfrac`, `from_cfrac`, `rand`, `fib`, `fibonacci`, `digit_count`, `poly_gcd`, `poly_lcm`, `resultant`, `groebner`, `sturm`, `root_count`, `isolate_roots`, `to_poly`, `to_alg`, `alg_inv`, `min_poly`, `find_min_poly`, `to_primitive_element`, `primitive_element`, `padic_val`, `padic_norm`, `padic_expand`, `solve`, `solve_diophantine`, `pell`, `pell_solve`, `interval`
- **Trigonometric, Logarithmic & Complex Functions**: `sin`, `cos`, `tan`, `asin`, `acos`, `atan`, `trig_expand`, `trig_reduce`, `exp`, `log`, `ln`, `arg`, `polar`, `polar_exp`, `rect`
- **Calculus, ODEs, Recurrences, Special Functions & Series**: `diff`, `integrate`, `piecewise`, `risch_integrate`, `limit`, `residue`, `lambert_w`, `erf`, `gamma`, `beta`, `bernoulli`, `zeta`, `delta`, `u`, `dsolve`, `rsolve`, `laplace`, `inv_laplace`, `ztrans`, `iztrans`, `taylor`, `fourier_series`, `sum`, `gosper_sum`, `wz_cert`, `puiseux`
- **Linear Algebra, Linear Programming, Integer Normal Forms & 3D Vector Calculus**: `det`, `inv`, `transpose`, `rref`, `rank`, `trace`, `tr`, `eigenvals`, `eigenvects`, `lu`, `qr`, `cholesky`, `ldlt`, `pinv`, `solve_linear`, `linsolve`, `lp_maximize`, `lp_minimize`, `hnf`, `hnf_transform`, `snf`, `snf_transform`, `invariant_factors`, `abelian_group_structure`, `lll`, `dot`, `cross`, `norm`, `grad`, `div`, `curl`
- **Computational Geometry**: `line_intersect`, `circle_intersect`, `triangle_area`, `triangle_centers`, `geo_prove`, `collinear`, `midpoint`, `parallel`, `perpendicular`, `equal_length`, `equal_length_sq`, `circle_concyclic`
- **Exact Discrete Probability & Statistics**: `binom`, `comb`, `perm`, `hyper`, `geom`, `bayes`, `expect`, `variance`, `stddev`
- **Real Algebraic Geometry & Quantifier Elimination**: `qe`, `forall`, `exists`, `cad`
- **Galois Theory & Solvability by Radicals**: `galois_group`, `is_solvable_by_radicals`
- **Elliptic Curves, Arithmetic Geometry & 2-Descent**: `ec_add`, `ec_mul`, `ec_torsion`, `ec_p_order`, `ec_p_trace`, `ec_p_add`, `ec_p_mul`, `ec_rank_bound`, `ec_generators`
- **Clifford Geometric Algebra & Quantum Circuit Equivalence**: `clifford`, `clifford_wedge`, `clifford_contract`, `clifford_dual`, `quantum_equiv`, `quantum_eval`
- **Multivariate Polynomial Ideals & Commutative Algebra**: `primary_decomposition`, `ideal_radical`, `ideal_dimension`, `ideal_intersection`, `ideal_colon`, `ideal_saturation`
- **Collection Manipulation & CAS Iteration Control**: `range`, `map`, `filter`, `lambda`, `table`, `product`, `for`, `to_base`, `from_base`, `bin`, `oct`, `hex`
- **Physical Units & Dimensional Analysis**: `convert_unit`, `unit`
- **Symbolic Assumptions System**: `assume`, `unassume`, `assumptions`, `clear_assumptions`
- **Visualization & Verification Tools**: `plot`, `verify`

> [!NOTE]
> **Algebraic & Transcendental Equation Solver, Linear Algebra, Elliptic Curve 2-Descent, Clifford Algebra & Quantum Verification**  
> `solve` provides exact radical solutions for linear, quadratic, cubic, and quartic polynomials (via Ferrari's method, Tschirnhaus transformation, and resolvent cubics), alongside exact closed-form solutions for elementary transcendental equations ($x e^x = c$, $x + \ln x = c$, $x^x = c$, etc.) via the Lambert W function (`lambert_w`). Furthermore, `integrate` incorporates Gaussian integrals $\int e^{-a x^2} dx$ in terms of the error function `erf`, and `eigenvals` derives exact radical eigenvalues for matrices up to 4×4. Additionally, `clifford` algebraically expands and canonicalizes multivector expressions (geometric product, exterior wedge, left/right contraction, Poincaré dual) under orthogonal diagonal signatures $(p, q, r)$, and `quantum_equiv` performs zero-decimal Clifford+T quantum circuit equivalence verification and witness extraction over the binary cyclotomic ring $\mathbb{Z}[1/\sqrt{2}, i]$. For elliptic curves over $\mathbb{Q}$ with rational 2-torsion, `ec_rank_bound` computes deterministic upper bounds on the Mordell-Weil rank via 2-isogeny descent and Selmer group dimension calculation, alongside searching for independent generators of infinite order (`ec_generators`).

---

### Command-Line Options

| Option               | Description & Examples                                                                                                                            |
| :------------------- | :------------------------------------------------------------------------------------------------------------------------------------------------ |
| `--theme <dark\|light\|none>` | Specify REPL color theme (`dark`, `light`, or `none`). Persisted to `~/.ihd/config.json`.<br>`ihd --theme light`                                  |
| `--no-color`         | Disables all ANSI terminal colors and escape sequences (respects `NO_COLOR=1`).                                                                   |
| `--warn-loop <N>`    | Step threshold before emitting non-destructive warning for loops (`table`, `for`, `product`, default: 100000).<br>`ihd --warn-loop 500 "table(k, k, 1, 1000)"` |
| `--max-loop <N>`     | Maximum iteration limit for loops before halting with an error (default: -1: unlimited).<br>`ihd --max-loop 10000 "table(k, k, 1, 20000)"`     |
| `--ascii`            | Output using standard ASCII strings (`sqrt`, `pi`) instead of Unicode symbols.<br>`ihd --ascii "sqrt(2) + pi"` $\to$ `sqrt(2) + pi`               |
| `--approx`           | Display approximate floating-point decimal value alongside the exact form.<br>`ihd --approx "sqrt(2)"` $\to$ `√2 (≈ 1.4142135623730951)`          |
| `--latex`            | Output expression in LaTeX format (`$$ ... $$`) ready to paste into papers.<br>`ihd --latex "1/2 + sqrt(2)"` $\to$ `$$ \frac{1}{2} + \sqrt{2} $$` |
| `--typst`            | Output expression in Typst math mode format (`$ ... $`).<br>`ihd --typst "1/2 + sqrt(2)"` $\to$ `$1 / 2 + sqrt(2)$`                                |
| `--format <fmt>`     | Specify output format (`typst`, `latex`, `ascii`, `pretty`).<br>`ihd --format typst "[[1, 2], [3, 4]]"` $\to$ `$mat(1, 2; 3, 4)$`                 |
| `--pretty`           | Output expression in multi-line 2D pretty-printed Unicode formatting.<br>`ihd --pretty "1/2 + sqrt(2)/2"`                                         |
| `--deg`              | Evaluate trigonometric and inverse trigonometric functions in degrees.<br>`ihd --deg "sin(30) + cos(60)"` $\to$ `1`                               |
| `--explain`          | Displays step-by-step algebraic rewriting derivation trees (radical denesting, rationalization, factoring, differentiation rules, etc.).<br>`ihd --explain "1 / (sqrt(2) + 1)"` |
| `--verify`           | Reverse-verifies algebraic correctness (factorization, matrix inverses, calculus, ODE, geometric proofs) by checking that residuals vanish.<br>`ihd --verify "integrate(1/(x^2 + 1), x)"` $\to$ `atan(x)`<br>`[VERIFIED: diff(F, x) - f == 0]` |
| `--certificate [file]` | Exports calculation steps and residuals into a certificate JSON file (`ihd-proof-trace-v1`).<br>`ihd --certificate "factor(x^2 - 1)"`<br>`ihd --certificate cert.json "factor(x^2 - 1)"` |
| `--replay <file>`    | Reads a certificate JSON file and verifies that residuals vanish without relying on solver logic (can also be verified via the standalone verifier `go run ./tools/micro-verifier <file>`).<br>`ihd --replay cert.json` |
| `-i`, `--interval`   | Evaluates expressions using adaptive rational interval arithmetic enclosures.<br>`ihd -i "sin(1)"`                                                |
| `--interval-eps <eps>` | Specifies the convergence tolerance epsilon for interval refinement (default: `1/1000000`).<br>`ihd -i --interval-eps "1/1000000000" "sin(1)"    |
| `--lean`             | Transpiles calculation results and rewriting steps into formal **Lean 4 (Mathlib4)** proof theorems (`by ring`, `by ext <;> ring`, `by repeat constructor <;> norm_num`, etc.).<br>`ihd --lean "factor(x^2 - 1)"` |
| `--lean-file <path>` | Save generated Lean 4 proof code as a standalone, runnable `.lean` file.<br>`ihd --lean-file proof.lean "factor(x^2 - 1)"`                         |
| `--lang <code/auto>` | Specify display language (`ja`, `en`, `auto`). Persisted to `~/.ihd/config.json`.<br>`ihd --lang en "1/2 + 1/3"`                                  |
| `completion <shell>` | Generate zero-dependency tab completion script for `bash`, `zsh`, `fish`, or `powershell`.<br>`ihd completion powershell \| Out-String \| Invoke-Expression` |
| `-h`, `--help`       | Display command help message.                                                                                                                     |
| `-v`, `--version`    | Display version information (checks for newer releases if available).                                                                             |
| `--no-update-check`  | Disable checking for newer releases (can also be disabled via `IHD_NO_UPDATE_CHECK=1`).                                                           |

> [!NOTE]  
> **Lean 4 / Mathlib4 Formal Proof Export Specification**  
> The `--lean` option exports formal proof code for calculation results in Lean 4 (Mathlib4) syntax.
> 
> **Supported:**
> - Polynomial expansion, factoring, and algebraic identities (`by ring`)
> - Indefinite integration verified via derivatives (`by simp`)
> - Matrix invertibility, linear systems, and HNF / SNF normal forms (`by ext <;> fin_cases`)
> - Rational roots of polynomials and integer solutions of Pell's equations (`by norm_num` / `by decide`)
> - Geometric automated theorem proving (Wu's method)
> 
> **Unsupported & Error Handling:**
> - Transcendental functions (Lambert W, `erf`), special functions, and general CAD quantifier elimination are not supported for automated proof generation.
> - When an expression or witness cannot be translated into a valid Lean theorem, the transpiler returns an explicit error rather than substituting a trivial proposition (such as `True`).
> 
> Proofs in `formal/test_cases.txt` are compiled and checked in GitHub Actions CI, with `#print axioms` confirming zero `sorry` usage.

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

### LICENSE
[MIT](./LICENSE.MIT)

Third-Party → [NOTICE.md](./NOTICE.md)
