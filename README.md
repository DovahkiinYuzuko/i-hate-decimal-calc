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

| 入力数式 | 一般的な電卓・言語（浮動小数点数） | `ihd`（完全厳密計算） | 簡約・処理内容 |
| :--- | :--- | :--- | :--- |
| `0.1 + 0.2` | `0.30000000000000004` | `3/10` | 誤差ゼロの完全有理数約分 |
| `0.(3)` | `0.3333333333333333` | `1/3` | 循環小数をFSM字句解析により厳密分数化 |
| `sqrt(8)` | `2.8284271247461903` | `2*√2` | 平方因子の自動くくり出し |
| `sqrt(5 + 2*sqrt(6))` | `3.1462643699419726` | `√2 + √3` | Borodin (1985) 法による二重根号の自動分解 |
| `1 / (sqrt(2) + 1)` | `0.4142135623730951` | `-1 + √2` | 2項無理数分母の共役有理化 |
| `sin(pi/6)` | `0.49999999999999994` | `1/2` | 特殊角の代数的厳密値評価 |
| `１＋２×３` | エラー（全角未対応） | `7` | 日本語IMEの全角文字を透過的に半角ASCII正規化 |

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

#### 演算子と優先順位
乗算記号 `*` の省略（例: `2pi` や `(1+2)(3+4)`）は誤認識防止のため禁止されています。

| 順位 | 演算子 | 結合性 | 説明・使用例 |
| :---: | :---: | :---: | :--- |
| 1 | `()` | - | グループ化括弧 |
| 2 | `!` | 後置単項 | 非負整数の階乗（例: `5! = 120`） |
| 3 | `^` | **右結合** | べき乗（例: `2^3^2 = 2^(3^2) = 512`） |
| 4 | `-` | 前置単項 | 単項マイナス（`^` より低優先: `-3^2 = -(3^2) = -9`） |
| 5 | `*`, `/` | 左結合 | 乗算・除算（乗算記号 `*` 必須） |
| 6 | `+`, `-` | 左結合 | 加算・減算 |

#### 予約定数
| 定数名 | 記号 | 説明 |
| :--- | :---: | :--- |
| `pi`, `π` | $\pi$ | 円周率 |
| `e` | $e$ | 自然対数の底（ネイピア数） |
| `i` | $i$ | 虚数単位（$i^2 = -1$） |
| `inf`, `infinity` | $\infty$ | 無限大（極限計算 `limit` 等で使用） |
| `deg` | - | 度数法変換定数（$\pi/180$。`sin(30*deg)` などの記述が可能） |

#### カテゴリ別 関数一覧

##### 1. 基本代数・数論・方程式
| 関数 | 書式・例 | 説明 |
| :--- | :--- | :--- |
| `abs` | `abs(x)` | 絶対値（実数は符号反転、複素数は $|a+bi| = \sqrt{a^2+b^2}$） |
| `sqrt` / `√` | `sqrt(x)` | 平方根（平方因数のくくり出し、負数は複素数 $i$ へ自動昇格） |
| `cbrt` | `cbrt(x)` | 3乗根（立方因子のくくり出し、実数負数の符号抽出） |
| `gcd` / `lcm` | `gcd(a, b)`, `lcm(a, b)` | 最大公約数・最小公倍数（整数） |
| `mod` | `mod(a, b)` | 整数剰余 |
| `perm` / `comb` | `perm(n, r)`, `comb(n, r)` | 順列 $nPr$、組合せ $nCr$（非負整数） |
| `rand` | `rand(max)`, `rand(min, max)`, `rand(seed, min, max)` | PCGアルゴリズムによる厳密整数擬似乱数（小数を不使用、シード指定可） |
| `cfrac` | `cfrac(expr)` | 連分数展開（有理数は有限リスト `[a0, a1, ...]`, 平方根は周期リスト `[a0, [a1, ...]]`） |
| `from_cfrac` | `from_cfrac([3, 7, 16])` | 連分数リストから厳密有理数への復元（例: `355/113`） |
| `expand` | `expand((x+1)^3)` | 多項式展開（分配法則・二項定理） |
| `factor` | `factor(expr)` / `factor(expr, var)` | 整数素因数分解（ホイール法）および有理数係数多項式因数分解（無平方分解・有理根定理） |
| `apart` | `apart(expr)` / `apart(expr, var)` | 部分分数分解（Kung-TongおよびHenriciアルゴリズムによる有理式の単項有理式分解） |
| `together` | `together(expr)` | 有理式の通分・統合（最小公倍多項式による共通分母化と単一有理式への統合） |
| `inv_mod` | `inv_mod(a, m)` | 拡張ユークリッド互除法によるモジュラ逆数（$a x \equiv 1 \pmod m$） |
| `crt` | `crt([r1, r2], [m1, m2])` | 中国剰余定理（Garner法および非互いに素な合同式を解く一般化CRT拡張） |
| `totient` | `totient(n)` | オイラーのトーシェント関数 $\phi(n) = n \prod_{p \mid n} (1 - 1/p)$ |
| `is_prime` | `is_prime(n)` | 決定論的素数判定（64bitはSorenson-Websterの12基底完全決定論的判定、巨大数はBaillie-PSW） |
| `solve` | `solve(expr, var)` | 厳密代数方程式ソルバー（1次・2次方程式の根の公式求解、重解・複素数解対応） |

##### 2. 三角関数・対数・複素数
| 関数 | 書式・例 | 説明 |
| :--- | :--- | :--- |
| `sin`, `cos`, `tan` | `sin(pi/6)` | 三角関数（$\pi$ の有理数倍による特殊角を代数的に厳密簡約） |
| `asin`, `acos`, `atan` | `asin(1/2)` | 逆三角関数（特殊角を $\pi$ の有理数倍として代数簡約、主値管理） |
| `trig_expand` | `trig_expand(sin(x+y))` / `trig_expand(sin(2*x))` | 加法定理・多倍角公式による展開（和積展開、2倍角・3倍角等） |
| `trig_reduce` | `trig_reduce(sin(x)^2)` / `trig_reduce(sin(x)+cos(x))` | 三角関数の次数下げ（半角公式）、積和変換、および有名角調和合成 |
| `log` | `log(x)` / `log(base, x)` | 常用対数（底10）および任意底の対数 |
| `ln` | `ln(x)` | 自然対数（底 $e$） |
| `arg` | `arg(z)` | 複素数の厳密偏角（主値 $\theta \in (-\pi, \pi]$。代数的特殊角比率逆引き） |
| `polar` | `polar(1 + i)` | 複素数の極形式変換 $r(\cos\theta + i\sin\theta)$（例: `√2*(cos(π/4) + i*sin(π/4))`） |
| `polar_exp` | `polar_exp(1 + i)` | オイラーの公式による指数形式変換 $r e^{i\theta}$（例: `√2*e^i*1/4*π`） |
| `rect` | `rect(sqrt(2), pi/4)` | 極形式（動径 $r$, 偏角 $\theta$）から直交形式 $a + bi$ への厳密逆変換（例: `1 + i`） |

##### 3. 微積分・離散数列
| 関数 | 書式・例 | 説明 |
| :--- | :--- | :--- |
| `diff` | `diff(sin(x)*x, x)` | 厳密記号微分（積の微分・商の微分・合成関数の連鎖律） |
| `integrate` | `integrate(x^2, x)` / `integrate(sin(x), x, 0, pi)` | 厳密不定積分（原始関数導出）および区間 $[a, b]$ による厳密定積分 |
| `limit` | `limit(sin(x)/x, x, 0)` / `limit(1/x, x, 0, 1)` | 厳密記号極限（$0/0$, $\infty/\infty$ の不定形解消、因数約分、ロピタルの定理、最高次数比較、片側極限） |
| `taylor` | `taylor(f, x, a, n)` | テイラー展開・マクローリン展開（点 $x=a$ まわりで $n$ 次まで展開） |
| `sum` | `sum(expr, k, start, end)` | 離散和（有限整数範囲の合算、または Faulhaber 公式による $n$ に関する多項式閉形式） |

##### 4. 線形代数・3次元ベクトル解析
| 関数 | 書式・例 | 説明 |
| :--- | :--- | :--- |
| `det` | `det(A)` | 厳密行列式（余因子展開法により除算を挟まない完全有理数計算） |
| `inv` | `inv(A)` | 厳密逆行列（余因子行列法による算出、特異行列時はエラー検出） |
| `transpose` | `transpose(A)` | 行列の転置（行と列の反転） |
| `rref` | `rref(A)` | 行簡約階段形（Reduced Row Echelon Form。Bareiss整数除算アルゴリズム） |
| `rank` | `rank(A)` | 行列の厳密な階数（ピボット列数） |
| `solve_linear` / `linsolve` | `solve_linear(A, b)` | 連立一次方程式 $Ax=b$ の厳密解（一意解、不能判定、自由変数 $x_i$ を含む一般解） |
| `dot`, `cross` | `dot(u, v)`, `cross(u, v)` | ベクトルの内積・3次元ベクトルの外積 |
| `norm` | `norm(v)` | ベクトルのユークリッドノルム（$\sqrt{\sum v_i^2}$、根号自動簡約） |
| `grad` | `grad(f, [x, y, z])` | スカラー場の勾配ベクトル（$\nabla f$） |
| `div`, `curl` | `div(F, [x, y, z])`, `curl(F, ...)` | ベクトル場の発散（$\nabla \cdot F$）・3次元ベクトル場の回転（$\nabla \times F$） |
| 行列・ベクトル記法 | `[[1, 2], [3, 4]]`, `[1, 2, 3]` | 行列リテラルおよびベクトルリテラル |

##### 5. 幾何学解析
| 関数 | 書式・例 | 説明 |
| :--- | :--- | :--- |
| `line_intersect` | `line_intersect([A1, B1, C1], [A2, B2, C2])` | 2直線 $Ax+By+C=0$ の交点（クラメルの公式による厳密交点 `[x, y]`） |
| `circle_intersect` | `circle_intersect(c1, r1, c2, r2)` | 2円の交点（中心座標と半径から根軸次数下げにより交点座標リストを算出） |
| `triangle_area` | `triangle_area(p1, p2, p3)` | 3頂点座標からなる三角形の厳密面積（外積・Shoelace公式） |
| `triangle_centers` | `triangle_centers(p1, p2, p3)` | 三角形の五心解析（重心・外心・垂心・内心をリストで一括算出） |

##### 6. 厳密離散確率・統計
| 関数 | 書式・例 | 説明 |
| :--- | :--- | :--- |
| `binom` | `binom(n, k, p)` | 二項分布の確率質量関数（PMF） $P(X=k) = \binom{n}{k} p^k (1-p)^{n-k}$ |
| `hyper` | `hyper(N, K, n, k)` | 超幾何分布の確率質量関数（非復元抽出の厳密有理数） |
| `geom` | `geom(p, k)` | 幾何分布の確率質量関数 $P(X=k) = (1-p)^{k-1} p$ |
| `bayes` | `bayes(prior, likelihood, marginal)` | ベイズの定理による厳密事後確率 $P(A\|B) = \frac{P(B\|A)P(A)}{P(B)}$ |
| `expect` | `expect(binom, n, p)` / `expect([[x1, p1], ...])` | 離散確率変数の期待値 $E[X]$ |
| `variance` | `variance(...)` | 離散確率変数の分散 $V[X] = E[X^2] - (E[X])^2$ |
| `stddev` | `stddev(...)` | 離散確率変数の標準偏差 $\sigma = \sqrt{V[X]}$（根号形式代数簡約） |

##### 7. 前提条件システム（仮定）
| コマンド | 例 | 説明 |
| :--- | :--- | :--- |
| `assume` | `assume(x > 0)`, `assume(n, integer)` | ドメイン制約を設定（`sqrt(x^2)` → `x`, `sin(n*pi)` → `0` 等の簡約が活性化） |
| `unassume` | `unassume(x)` | 指定した変数の仮定制約を解除 |
| `assumptions` | `assumptions()` | 現在設定されている前提条件の一覧を表示 |

---

### コマンドラインオプション

| オプション | 説明・使用例 |
| :--- | :--- |
| `--ascii` | 数学記号（`√`, `π`）をASCII文字列（`sqrt`, `pi`）にフォールバックして出力。<br>`ihd --ascii "sqrt(2) + pi"` $\to$ `sqrt(2) + pi` |
| `--approx` | 厳密解の横に参考用の浮動小数点小数近似値（float64、15〜17桁）を併記。<br>`ihd --approx "sqrt(2)"` $\to$ `√2 (≈ 1.4142135623730951)` |
| `--latex` | MarkdownやTeX論文に貼り付け可能なLaTeX形式（`$$ ... $$`）で出力。<br>`ihd --latex "1/2 + sqrt(2)"` $\to$ `$$ \frac{1}{2} + \sqrt{2} $$` |
| `--pretty` | 分数線や根号を複数行アスキーアートで組版表示する2Dプリティプリント。<br>`ihd --pretty "1/2 + sqrt(2)/2"` |
| `--deg` | 三角関数および逆三角関数を度数法（Degree）として解釈・計算。<br>`ihd --deg "sin(30) + cos(60)"` $\to$ `1` |
| `--explain` | 代数的項書き換え（有理化・二重根号・微分則等）を途中式ツリーとして詳細表示。<br>`ihd --explain "1 / (sqrt(2) + 1)"` |
| `--lang <code/auto>` | 表示言語（ロケール）を指定（`ja`, `en`, `auto`）。設定は `~/.ihd/config.json` に永続化され、カスタム辞書（`~/.ihd/locales/`）にも対応。<br>`ihd --lang en "1/2 + 1/3"` |
| `-h`, `--help` | コマンドのヘルプメッセージを表示。 |

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

| Input Expression | Typical Calculators / Languages (`float64`) | `ihd` (Exact Calculation) | Simplification Behavior |
| :--- | :--- | :--- | :--- |
| `0.1 + 0.2` | `0.30000000000000004` | `3/10` | Zero-error exact rational fraction |
| `0.(3)` | `0.3333333333333333` | `1/3` | FSM lexical analysis converts repeating decimals to exact fractions |
| `sqrt(8)` | `2.8284271247461903` | `2*√2` | Automatic extraction of perfect squares |
| `sqrt(5 + 2*sqrt(6))` | `3.1462643699419726` | `√2 + √3` | Automatic radical denesting via Borodin (1985) algorithm |
| `1 / (sqrt(2) + 1)` | `0.4142135623730951` | `-1 + √2` | Binomial conjugate radical rationalization |
| `sin(pi/6)` | `0.49999999999999994` | `1/2` | Exact algebraic evaluation of special angles |
| `１＋２×３` | Error (Unsupported) | `7` | Automatic normalization of Zenkaku characters to ASCII |

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

### Syntax & Functions Reference

#### Operators & Precedence
Implicit multiplication (e.g., `2pi` or `(1+2)(3+4)`) is strictly prohibited to prevent syntactic ambiguity.

| Rank | Operator | Associativity | Description & Examples |
| :---: | :---: | :---: | :--- |
| 1 | `()` | - | Grouping parentheses |
| 2 | `!` | Postfix Unary | Factorial of non-negative integers (e.g., `5! = 120`) |
| 3 | `^` | **Right-associative** | Exponentiation (e.g., `2^3^2 = 2^(3^2) = 512`) |
| 4 | `-` | Prefix Unary | Unary negation (Lower precedence than `^`: `-3^2 = -(3^2) = -9`) |
| 5 | `*`, `/` | Left-associative | Multiplication and division (Explicit `*` required) |
| 6 | `+`, `-` | Left-associative | Addition and subtraction |

#### Constants
| Identifier | Symbol | Description |
| :--- | :---: | :--- |
| `pi`, `π` | $\pi$ | Archimedes' constant (Ratio of circumference to diameter) |
| `e` | $e$ | Euler's number (Base of natural logarithm) |
| `i` | $i$ | Imaginary unit ($i^2 = -1$) |
| `deg` | - | Degree conversion constant ($\pi/180$. Allows `sin(30*deg)`) |

#### Functions by Category

##### 1. Basic Algebra, Number Theory & Equations
| Function | Syntax & Example | Description |
| :--- | :--- | :--- |
| `abs` | `abs(x)` | Absolute value (Sign removal for reals; $|a+bi| = \sqrt{a^2+b^2}$ for complex) |
| `sqrt` / `√` | `sqrt(x)` | Square root (Square-free factoring, promotes negative numbers to complex $i$) |
| `cbrt` | `cbrt(x)` | Cube root (Cube-free factoring, real sign extraction) |
| `gcd` / `lcm` | `gcd(a, b)`, `lcm(a, b)` | Greatest common divisor & least common multiple (integers) |
| `mod` | `mod(a, b)` | Integer modulo |
| `perm` / `comb` | `perm(n, r)`, `comb(n, r)` | Permutations $nPr$ and combinations $nCr$ (non-negative integers) |
| `rand` | `rand(max)`, `rand(min, max)`, `rand(seed, min, max)` | Integer pseudo-random generation via PCG algorithm (no floats, seedable) |
| `cfrac` | `cfrac(expr)` | Regular continued fraction expansion (finite list for rationals, periodic for roots) |
| `from_cfrac` | `from_cfrac([3, 7, 16])` | Reconstruct exact rational fraction from continued fraction list (`355/113`) |
| `expand` | `expand((x+1)^3)` | Polynomial expansion using distributive law and binomial expansion |
| `factor` | `factor(expr)` / `factor(expr, var)` | Integer prime factorization and univariate polynomial factorization |
| `inv_mod` | `inv_mod(a, m)` | Modular inverse via Extended Euclidean Algorithm ($a x \equiv 1 \pmod m$) |
| `crt` | `crt([r1, r2], [m1, m2])` | Chinese Remainder Theorem (Garner's algorithm & generalized non-coprime CRT) |
| `totient` | `totient(n)` | Euler's totient function $\phi(n) = n \prod_{p \mid n} (1 - 1/p)$ |
| `is_prime` | `is_prime(n)` | Deterministic primality test (Sorenson & Webster 12 bases for 64-bit, Baillie-PSW for big ints) |
| `solve` | `solve(expr, var)` | Exact algebraic equation solver for linear and quadratic equations |

##### 2. Trigonometric, Logarithmic & Complex Functions
| Function | Syntax & Example | Description |
| :--- | :--- | :--- |
| `sin`, `cos`, `tan` | `sin(pi/6)` | Trigonometric functions (exact algebraic evaluation for rational multiples of $\pi$) |
| `asin`, `acos`, `atan` | `asin(1/2)` | Inverse trigonometric functions (exact values for special angles, principal branch) |
| `trig_expand` | `trig_expand(sin(x+y))` / `trig_expand(sin(2*x))` | Expands trigonometric functions using addition theorems and multiple-angle formulas |
| `trig_reduce` | `trig_reduce(sin(x)^2)` / `trig_reduce(sin(x)+cos(x))` | Reduces powers (half-angle), products to sums, and harmonic linear combinations |
| `log` | `log(x)` / `log(base, x)` | Common logarithm (base 10) and arbitrary base logarithm |
| `ln` | `ln(x)` | Natural logarithm (base $e$) |
| `arg` | `arg(z)` | Exact principal complex argument $\theta \in (-\pi, \pi]$ |
| `polar` | `polar(1 + i)` | Polar form conversion $r(\cos\theta + i\sin\theta)$ (e.g., `√2*(cos(π/4) + i*sin(π/4))`) |
| `polar_exp` | `polar_exp(1 + i)` | Exponential polar conversion $r e^{i\theta}$ via Euler's formula |
| `rect` | `rect(sqrt(2), pi/4)` | Converts polar form $(r, \theta)$ to rectangular form $a + bi$ |

##### 3. Calculus & Discrete Summation
| Function | Syntax & Example | Description |
| :--- | :--- | :--- |
| `diff` | `diff(sin(x)*x, x)` | Exact symbolic differentiation (Product, Quotient, and Chain rules) |
| `integrate` | `integrate(x^2, x)` / `integrate(sin(x), x, 0, pi)` | Exact symbolic indefinite integration and definite integration over $[a, b]$ |
| `taylor` | `taylor(f, x, a, n)` | Taylor / Maclaurin series expansion around $x=a$ up to order $n$ |
| `sum` | `sum(expr, k, start, end)` | Discrete summation (finite sum or exact polynomial closed form via Faulhaber formula) |

##### 4. Linear Algebra & 3D Vector Calculus
| Function | Syntax & Example | Description |
| :--- | :--- | :--- |
| `det` | `det(A)` | Exact determinant of a matrix via division-free Laplace expansion |
| `inv` | `inv(A)` | Exact inverse matrix via adjugate matrix method |
| `transpose` | `transpose(A)` | Matrix transpose (swaps rows and columns) |
| `rref` | `rref(A)` | Reduced Row Echelon Form via Bareiss fraction-free elimination |
| `rank` | `rank(A)` | Exact matrix rank (number of pivot columns) |
| `solve_linear` / `linsolve` | `solve_linear(A, b)` | Exact linear system solver $Ax=b$ (unique, inconsistent, or parametric solutions) |
| `dot`, `cross` | `dot(u, v)`, `cross(u, v)` | Vector dot product and 3D vector cross product |
| `norm` | `norm(v)` | Euclidean vector norm ($\sqrt{\sum v_i^2}$, with radical simplification) |
| `grad` | `grad(f, [x, y, z])` | Gradient vector field ($\nabla f$) |
| `div`, `curl` | `div(F, [x, y, z])`, `curl(F, ...)` | Divergence ($\nabla \cdot F$) and 3D curl ($\nabla \times F$) |
| Matrix / Vector Syntax | `[[1, 2], [3, 4]]`, `[1, 2, 3]` | Matrix and vector literal syntax |

##### 5. Computational Geometry
| Function | Syntax & Example | Description |
| :--- | :--- | :--- |
| `line_intersect` | `line_intersect([A1, B1, C1], [A2, B2, C2])` | Exact intersection point of two lines $Ax+By+C=0$ via Cramer's rule |
| `circle_intersect` | `circle_intersect(c1, r1, c2, r2)` | Exact intersection points of two circles via radical axis order reduction |
| `triangle_area` | `triangle_area(p1, p2, p3)` | Exact area of a triangle given three vertices via Shoelace formula |
| `triangle_centers` | `triangle_centers(p1, p2, p3)` | Triangle centers (centroid, circumcenter, orthocenter, incenter) |

##### 6. Exact Discrete Probability & Statistics
| Function | Syntax & Example | Description |
| :--- | :--- | :--- |
| `binom` | `binom(n, k, p)` | Binomial distribution PMF $P(X=k) = \binom{n}{k} p^k (1-p)^{n-k}$ |
| `hyper` | `hyper(N, K, n, k)` | Hypergeometric distribution PMF (without replacement) |
| `geom` | `geom(p, k)` | Geometric distribution PMF $P(X=k) = (1-p)^{k-1} p$ |
| `bayes` | `bayes(prior, likelihood, marginal)` | Exact posterior probability $P(A\|B) = \frac{P(B\|A)P(A)}{P(B)}$ via Bayes' theorem |
| `expect` | `expect(binom, n, p)` / `expect([[x1, p1], ...])` | Expected value $E[X]$ of discrete distributions |
| `variance` | `variance(...)` | Variance $V[X] = E[X^2] - (E[X])^2$ |
| `stddev` | `stddev(...)` | Standard deviation $\sigma = \sqrt{V[X]}$ (in exact radical form) |

##### 7. Symbolic Assumptions System
| Command | Example | Description |
| :--- | :--- | :--- |
| `assume` | `assume(x > 0)`, `assume(n, integer)` | Sets domain constraints (`sqrt(x^2)` → `x`, `sin(n*pi)` → `0`, etc.) |
| `unassume` | `unassume(x)` | Clears assumptions for the specified variable |
| `assumptions` | `assumptions()` | Displays all active domain constraints |

---

### Command-Line Options

| Option | Description & Examples |
| :--- | :--- |
| `--ascii` | Output using standard ASCII strings (`sqrt`, `pi`) instead of Unicode symbols.<br>`ihd --ascii "sqrt(2) + pi"` $\to$ `sqrt(2) + pi` |
| `--approx` | Display approximate floating-point decimal value alongside the exact form.<br>`ihd --approx "sqrt(2)"` $\to$ `√2 (≈ 1.4142135623730951)` |
| `--latex` | Output expression in LaTeX format (`$$ ... $$`) ready to paste into papers.<br>`ihd --latex "1/2 + sqrt(2)"` $\to$ `$$ \frac{1}{2} + \sqrt{2} $$` |
| `--pretty` | Output expression in multi-line 2D pretty-printed Unicode formatting.<br>`ihd --pretty "1/2 + sqrt(2)/2"` |
| `--deg` | Evaluate trigonometric and inverse trigonometric functions in degrees.<br>`ihd --deg "sin(30) + cos(60)"` $\to$ `1` |
| `--explain` | Step-by-step educational explanations of algebraic derivations in a 2D tree.<br>`ihd --explain "1 / (sqrt(2) + 1)"` |
| `--lang <code/auto>` | Specify display language (`ja`, `en`, `auto`). Persisted to `~/.ihd/config.json`.<br>`ihd --lang en "1/2 + 1/3"` |
| `-h`, `--help` | Display command help message. |

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
