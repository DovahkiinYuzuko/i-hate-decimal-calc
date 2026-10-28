# ihd 完全構文・演算子・関数リファレンス

小数を一切使用せず、任意精度有理数と代数的シンボル表現のみで厳密計算を行う CAS 電卓 `ihd`（i-hate-decimal-calc）の完全言語仕様および組み込み関数リファレンスです。

---

## 目次
- [ihd 完全構文・演算子・関数リファレンス](#ihd-完全構文演算子関数リファレンス)
  - [目次](#目次)
  - [演算子と優先順位](#演算子と優先順位)
    - [関係演算子と代入・等式の区別](#関係演算子と代入等式の区別)
  - [数値リテラル構文（複数進数・循環小数）](#数値リテラル構文複数進数循環小数)
  - [予約定数](#予約定数)
  - [カテゴリ別 組み込み関数一覧](#カテゴリ別-組み込み関数一覧)
    - [1. 基本代数・数論・方程式](#1-基本代数数論方程式)
    - [2. 三角関数・対数・複素数](#2-三角関数対数複素数)
    - [3. 微積分・常微分方程式・初等特殊関数・離散数列](#3-微積分常微分方程式初等特殊関数離散数列)
    - [4. 線形代数・3次元ベクトル解析](#4-線形代数3次元ベクトル解析)
    - [5. 幾何学解析](#5-幾何学解析)
    - [6. 厳密離散確率・統計](#6-厳密離散確率統計)
    - [7. 前提条件システム（仮定）](#7-前提条件システム仮定)
    - [8. 実代数幾何・数理論理・量化子消去](#8-実代数幾何数理論理量化子消去)
    - [9. 楕円曲線代数・数論幾何](#9-楕円曲線代数数論幾何)
    - [10. ビジュアル・検証ツール](#10-ビジュアル検証ツール)
    - [11. 格子基底縮小・代数的整数関係 (LLL)](#11-格子基底縮小代数的整数関係-lll)
    - [12. p進数演算・局所代数・ヘンゼル補題](#12-p進数演算局所代数ヘンゼル補題)
    - [13. 局所代数級数・特異点解析（ピュイズー級数）](#13-局所代数級数特異点解析ピュイズー級数)
    - [14. ガロア理論・代数的根号可解性・アーベル・ルフィニ代数的証明書](#14-ガロア理論代数的根号可解性アーベルルフィニ代数的証明書)
    - [15. 自己検証証明書 \& 形式証明基盤（ProofTrace / Explain / Verify / Lean 4）](#15-自己検証証明書--形式証明基盤prooftrace--explain--verify--lean-4)
    - [16. クリフォード幾何代数 \& 二進円分環・厳密量子回路等価性検証](#16-クリフォード幾何代数--二進円分環厳密量子回路等価性検証)
    - [17. CAS反復制御構文 \& 任意基数進数変換](#17-cas反復制御構文--任意基数進数変換)
    - [18. REPL 構文ハイライト・テーマ・対話環境コマンド](#18-repl-構文ハイライトテーマ対話環境コマンド)
    - [19. 多変数多項式イデアル・準素分解 (GTZ法)](#19-多変数多項式イデアル準素分解-gtz法)
    - [20. コレクション・リスト操作言語拡張 \& 第一級ラムダ式](#20-コレクションリスト操作言語拡張--第一級ラムダ式)
    - [21. 物理単位計算 \& 次元解析 (SI基本単位・組立単位・単位換算)](#21-物理単位計算--次元解析-si基本単位組立単位単位換算)
      - [対応単位・接頭辞](#対応単位接頭辞)

---

## 演算子と優先順位

乗算記号 `*` の省略（例: `2pi` や `(1+2)(3+4)`）は誤認識防止のため禁止されています。必ず明示的に `*` を記述してください。

| 順位 | 演算子 | 結合性 | 説明・使用例 |
| :---: | :---: | :---: | :--- |
| 1 | `()` | - | グループ化括弧 |
| 2 | `!` | 後置単項 | 非負整数の階乗（例: `5! = 120`。巨大階乗はルジャンドル素因数分解、二分木乗算、数論変換NTTにより高速計算） |
| 3 | `^` | **右結合** | べき乗（例: `2^3^2 = 2^(3^2) = 512`） |
| 4 | `-` | 前置単項 | 単項マイナス（`^` より低優先: `-3^2 = -(3^2) = -9`） |
| 5 | `*`, `/` | 左結合 | 乗算・除算（乗算記号 `*` 必須） |
| 6 | `+`, `-` | 左結合 | 加算・減算 |
| 7 | `<`, `<=`, `>`, `>=`, `==`, `!=` | なし | 関係演算・等式・不等式（有理区間論理および代数的同値証明による厳密判定） |

### 関係演算子と代入・等式の区別
- **代入演算子 (`=`)**: REPL やスクリプト環境における変数への束縛を行います（例: `x = 42`, `f = sin(x)`）。
- **等価比較・関係演算子 (`==`, `!=`)**:
  - トップレベル評価時: 代数的同値性（`lhs - rhs == 0` や恒等式証明）および有理区間解析に基づき、定数・同値式を `true` または `false` に厳密判定します（例: `1 == 2` $\to$ `false`、`0.(9) == 1` $\to$ `true`、`sqrt(2)+sqrt(3) == sqrt(5+2*sqrt(6))` $\to$ `true`、`1 != 2` $\to$ `true`）。
  - ソルバー引数内 (`solve`, `dsolve`, `rsolve`, `verify` など): 方程式・関係式オブジェクトとしてそのまま保持され、未知数の求解や等価性検証の入力として安全に渡されます（例: `solve(2 == 0, x)` は解なしエラーを報告し、`dsolve(y' == y, y, x)` は微分方程式として解かれます）。
- **階乗演算子 (`!`) との区別**: 階乗 `5!` の直後に関係演算子を続ける場合、`1!=2` は `1 != 2`（非等価比較）として正しくトークナイズされます。明示的に階乗と比較を組み合わせる場合は `(5!) == 120` または `5! != 120` と記述できます。

---

## 数値リテラル構文（複数進数・循環小数）

`ihd` は小数を排除した厳密有理数電卓であり、数値リテラルは構文解析時に即座に多倍長有理数（$\mathbb{Q}$）へ変換されます。

| リテラル形式 | 表記例 | 変換結果（10進有理数） | 説明 |
| :--- | :--- | :--- | :--- |
| **標準10進数** | `42`, `1/3` | `42`, `1/3` | 通常の多倍長整数および有理数除算 |
| **有限小数・循環小数** | `0.75`, `0.(3)`, `1.1(6)` | `3/4`, `1/3`, `7/6` | 小数点表記および循環節括弧表記。完全厳密な有理数に即時展開 |
| **2進数リテラル** | `0b1011`, `0B0` | `11`, `0` | 接頭辞 `0b` / `0B` による2進数表記 |
| **8進数リテラル** | `0o755`, `0O10` | `493`, `8` | 接頭辞 `0o` / `0O` による8進数表記 |
| **16進数リテラル** | `0xff`, `0X1A` | `255`, `26` | 接頭辞 `0x` / `0X` による16進数表記 |
| **任意進数リテラル** | `2^^101`, `16^^ff`, `64^^@_` | `5`, `255`, `4031` | Mathematica構文準拠の任意進数記法 `base^^digits`（$2 \le \text{base} \le 64$ 対応。Base 64はBash互換の `@`=62, `_`=63） |

---

## 予約定数

| 定数名 | 記号 | 説明 |
| :--- | :---: | :--- |
| `pi`, `π` | $\pi$ | 円周率（約 3.14159...。シンボルノードとして完全保持） |
| `e` | $e$ | 自然対数の底（ネイピア数 約 2.71828...） |
| `i` | $i$ | 虚数単位（$i^2 = -1$） |
| `inf`, `infinity` | $\infty$ | 正の無限大（極限計算 `limit`、実根分離区間等で使用） |
| `-inf` | $-\infty$ | 負の無限大 |
| `O` | $\mathcal{O}$ | 楕円曲線の群演算における単位元（無限遠点） |
| `deg` | - | 度数法変換定数（$\pi/180$。`sin(30*deg)` などの記述が可能） |
| `delta` | $\delta$ | ディラックのデルタ（ラプラス変換等の超関数） |
| `true`, `false` | - | ブール論理定数（関係演算やQE判定結果） |

---

## カテゴリ別 組み込み関数一覧

### 1. 基本代数・数論・方程式

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
| `poly_gcd` | `poly_gcd(p, q)` / `poly_gcd(p, q, var)` | 多変数多項式の厳密最大公約因子（Collins/BrownのSubresultant PRSアルゴリズムによる除算フリー・係数爆発防止型GCD） |
| `poly_lcm` | `poly_lcm(p, q)` / `poly_lcm(p, q, var)` | 多変数多項式の厳密最小公倍多項式（$\frac{p \cdot q}{\mathrm{GCD}(p, q)}$ による厳密多項式除算） |
| `resultant` | `resultant(p, q)` / `resultant(p, q, var)` | シルベスター終結式（Sylvester (1840) の Dialytic Method による行列式評価と多変数代数方程式消去法） |
| `groebner` | `groebner([p1, p2, ...], [x, y, ...])` / `groebner(..., [order])` | 多変数多項式イデアルの既約グレブナー基底（Buchberger (1965/1979/1985)、Gebauer-Möller (1988) 基準、Giovini (1991) Sugar 戦略による高速算出。順序 `lex`/`grevlex`） |
| `sturm` | `sturm(p)` / `sturm(p, var)` | 厳密スツルム列（Sturm chain）の生成（$\mathbb{Q}[x]$ 上のPrimitive PRSによる係数爆発防止型スツルム多項式剰余列） |
| `root_count` | `root_count(p)` / `root_count(p, var, [a, b])` | スツルムの定理による実根の厳密な個数算定（区間 $[a, b]$ または $(-\infty, \infty)$ 内の相異なる実根数を符号変化数から決定） |
| `isolate_roots` | `isolate_roots(p)` / `isolate_roots(p, var, [a, b])` | 厳密代数的実根分離（無平方分解と有理二分法探索により、各実根をちょうど1個含む互いに素な有理数区間を確定） |
| `inv_mod` | `inv_mod(a, m)` | 拡張ユークリッド互除法によるモジュラ逆数（$a x \equiv 1 \pmod m$） |
| `crt` | `crt([r1, r2], [m1, m2])` | 中国剰余定理（Garner法および非互いに素な合同式を解く一般化CRT拡張） |
| `totient` | `totient(n)` | オイラーのトーシェント関数 $\phi(n) = n \prod_{p \mid n} (1 - 1/p)$ |
| `is_prime` | `is_prime(n)` | 決定論的素数判定（64bitはSorenson-Websterの12基底完全決定論的判定、巨大数はBaillie-PSW） |
| `solve` | `solve(expr, var)` | 厳密代数・初等超越方程式ソルバー（1次・2次・3次・4次代数方程式の代数的根号求解に加え、ランバートW関数 `lambert_w` による初等超越方程式 $f(x)e^{f(x)}=c$, $ax+b\ln x=c$, $x^x=c$, $a^x=bx$ の完全閉形式解に対応。カルダノ公式・フェラーリ法・チルンハウス変換・分解3次方程式レゾルベントによる厳密解、重解・複素数解対応） |
| `to_poly` | `to_poly(expr, [x, y], "lex")` | 式を指定変数・単項式順序（`lex`, `grevlex`）の正規形多項式ノード `PolyNode` へ明示変換 |
| `to_alg` | `to_alg(rep, min_poly, [var])` | 代数拡大体 $\mathbb{Q}(\alpha) \cong \mathbb{Q}[x]/\langle m(x) \rangle$ の代数数ノード `AlgNode` を構築 |
| `alg_inv` | `alg_inv(rep, min_poly)` | 拡大ユークリッド互除法による代数数 $\beta \in \mathbb{Q}(\alpha)$ の乗法逆元 $\beta^{-1}$ 算出 |
| `min_poly` | `min_poly(rep, min_poly)` | 代数数の既約モニック最小多項式 $m(x) \in \mathbb{Q}[x]$ の導出 |
| `solve_diophantine` | `solve_diophantine(3*x + 5*y == 7, [x, y])` | 不定整数方程式（ディオファントス方程式）の完全厳密ソルバー。1次線形不定方程式（拡張ユークリッド互除法およびベズーの等式に基づく整数パラメータ $t$ による一般解系）およびピタゴラス方程式（$x^2+y^2=z^2$ 等の代数曲線有理点媒介変数表示）を判定・求解。`--verify` 時には代数的残差ゼロ証明書、`--lean` 時には Lean 4 形式証明を出力 |
| `pell_solve` | `pell_solve(D)` / `pell_solve(D, 1)` / `pell_solve(D, -1)` | ペル方程式 $x^2 - D y^2 = 1$（および負のペル方程式 $x^2 - D y^2 = -1$）の最小整数解（基本解 $(x_1, y_1)$）の完全厳密導出。非平方正整数 $D$ に対する連分数展開周期アルゴリズム（Continued Fraction Expansion）により巨大桁整数でも決定論的に求解。`--verify` および `--lean`（Mathlib4 `by decide` 定理コード生成）に対応 |
| `to_primitive_element` / `primitive_element` | `to_primitive_element([sqrt(2), sqrt(3)])` / `to_primitive_element([x^2 - 2, x^2 - 3])` | 原始元定理（Primitive Element Theorem）に基づき、多重代数拡大体 $\mathbb{Q}(\alpha_1, \dots, \alpha_k)$ を単一の代数的数 $\theta$ による単拡大 $\mathbb{Q}(\theta)$ へ同型変換し、原始元シンボル、合成最小多項式 $m_\theta(x)$、および各生成元の表現多項式リスト $[P_{\alpha_1}(\theta), \dots, P_{\alpha_k}(\theta)]$ を導出。シルベスター終結式と拡大体上多項式GCDにより決定論的に求解。`--verify` および `--lean` による代数関係式検証に対応 |
| `fib` / `fibonacci` | `fib(n)` / `fibonacci(n)` | フィボナッチ数 $F_n$。Fast Doubling アルゴリズム（$O(\log n)$ 準線形時間ステップ）により数万〜十万番台の巨大フィボナッチ数を高速算出。負の整数引数に対するネガフィボナッチ数 $F_{-n} = (-1)^{n+1} F_n$ にも完全対応 |
| `digit_count` | `digit_count(n)` / `digit_count(n, base)` | 巨大整数の厳密桁数算定。基数 `base`（省略時10）における桁数を高速に算定（例: `digit_count(fib(10000))` $\to$ `2090`） |
| `to_base` | `to_base(n, base)` | 任意基数進数変換（$2 \le \text{base} \le 64$）。整数の位取り記数法表現および有理数 $p/q$ の長除法余り追跡による循環小数展開（例: `to_base(255, 16)` $\to$ `ff`、`to_base(1/3, 2)` $\to$ `0.(01)_2`、`to_base(1/3, 16)` $\to$ `0.(5)_16`、`to_base(4031, 64)` $\to$ `@_`） |
| `from_base` | `from_base(str, base)` | 任意基数進数表現（$2 \le \text{base} \le 64$）から多倍長整数への復元。`0x`, `0b`, `0o` 接頭辞の自動処理、および Base 64 の `@`/`_` または `+`/`/` 表記に対応（例: `from_base("ff", 16)` $\to$ `255`、`from_base("@_", 64)` $\to$ `4031`） |
| `bin` | `bin(n)` | 2進数変換ショートカット（接頭辞 `0b` 付き、例: `bin(13)` $\to$ `0b1101`） |
| `oct` | `oct(n)` | 8進数変換ショートカット（接頭辞 `0o` 付き、例: `oct(63)` $\to$ `0o77`） |
| `hex` | `hex(n)` | 16進数変換ショートカット（接頭辞 `0x` 付き、例: `hex(255)` $\to$ `0xff`） |

---

### 2. 三角関数・対数・複素数

| 関数 | 書式・例 | 説明 |
| :--- | :--- | :--- |
| `sin`, `cos`, `tan` | `sin(pi/6)` | 三角関数（$\pi$ の有理数倍による特殊角を代数的に厳密簡約） |
| `asin`, `acos`, `atan` | `asin(1/2)` | 逆三角関数（特殊角を $\pi$ の有理数倍として代数簡約、主値管理） |
| `trig_expand` | `trig_expand(sin(x+y))` / `trig_expand(sin(2*x))` | 加法定理・多倍角公式による展開（和積展開、2倍角・3倍角等） |
| `trig_reduce` | `trig_reduce(sin(x)^2)` / `trig_reduce(sin(x)+cos(x))` | 三角関数の次数下げ（半角公式）、積和変換、および有名角調和合成 |
| `exp` | `exp(x)` | 自然指数関数（$e^x$ の厳密表現、指数法則積合成 $e^A \cdot e^B = e^{A+B}$） |
| `log` | `log(x)` / `log(base, x)` | 常用対数（底10）および任意底の対数 |
| `ln` | `ln(x)` | 自然対数（底 $e$） |
| `arg` | `arg(z)` | 複素数の厳密偏角（主値 $\theta \in (-\pi, \pi]$。代数的特殊角比率逆引き） |
| `polar` | `polar(1 + i)` | 複素数の極形式変換 $r(\cos\theta + i\sin\theta)$（例: `√2*(cos(π/4) + i*sin(π/4))`） |
| `polar_exp` | `polar_exp(1 + i)` | オイラーの公式による指数形式変換 $r e^{i\theta}$（例: `√2*e^i*1/4*π`） |
| `rect` | `rect(sqrt(2), pi/4)` | 極形式（動径 $r$, 偏角 $\theta$）から直交形式 $a + bi$ への厳密逆変換（例: `1 + i`） |

---

### 3. 微積分・常微分方程式・初等特殊関数・離散数列

| 関数 | 書式・例 | 説明 |
| :--- | :--- | :--- |
| `diff` | `diff(sin(x)*x, x)` / `diff(x^4, x, 2)` | 厳密記号微分（積の微分・商の微分・連鎖律、任意階数 $n$ 次微分対応） |
| `integrate` | `integrate(x^2, x)` / `integrate(sin(x), x, 0, pi)` | 厳密不定積分（原始関数導出、ガウス積分 $\int e^{-a x^2} dx \to \frac{\sqrt{\pi}}{2\sqrt{a}} \operatorname{erf}(\sqrt{a}x)$ の誤差関数閉形式化、$\int W(x) dx$ 原始関数対応）および区間 $[a, b]$ による厳密定積分 |
| `risch_integrate` | `risch_integrate(x*exp(x^2), x)` | 決定論的 Risch アルゴリズム（超越拡大体・Rothstein-Trager 法）による厳密不定積分 |
| `lambert_w` | `lambert_w(z)` / `lambert_w(z, k)` | ランバートのW関数 $W_k(z)$（$w e^w = z$ の多価逆関数。主枝 $k=0$ をデフォルトとし、特殊引数 $W(0)=0, W(e)=1, W(-1/e)=-1$ 等の即時簡約、導関数 $\frac{d}{dz}W(z) = \frac{W(z)}{z(1+W(z))}$、原始関数 $\int W(x) dx = x W(x) - x + e^{W(x)}$、代数的等値消去 $W(z)e^{W(z)} \to z$ を提供） |
| `erf` | `erf(z)` | ガウスの誤差関数 $\operatorname{erf}(z) = \frac{2}{\sqrt{\pi}}\int_0^z e^{-t^2} dt$（特殊値 $\operatorname{erf}(0)=0$、奇関数性 $\operatorname{erf}(-z) = -\operatorname{erf}(z)$、導関数 $\frac{d}{dz}\operatorname{erf}(z) = \frac{2}{\sqrt{\pi}} e^{-z^2}$、ガウス積分 $\int e^{-a x^2} dx$ の初等閉形式化基盤） |
| `limit` | `limit(sin(x)/x, x, 0)` / `limit(1/x, x, 0, 1)` | 厳密記号極限（$0/0$, $\infty/\infty$ の不定形解消、因数約分、ロピタルの定理、最高次数比較、片側極限） |
| `dsolve` | `dsolve(diff(y, x) == y, y, x)` / `dsolve(diff(y, x, 2) + 4*y == 0, y, x)` | 記号常微分方程式ソルバー（1階線形・変数分離形積分因子法、2階定数係数線形斉次・未定係数法特解による厳密求解） |
| `rsolve` | `rsolve(a(n+1) == 2*a(n) + 1, a(n), [a(1) == 1])` / `rsolve(a(n+3) - 6*a(n+2) + 11*a(n+1) - 6*a(n) == 0, a(n), [a(0) == 1, a(1) == 2, a(2) == 4])` | 線形漸化式ソルバー（次数 $d \ge 1$ 定数係数線形斉次・非同次漸化式、特性方程式求根・重根ジョルダン基底 $n^j r^n$・共鳴外力未定係数法・$d \times d$ 連立一次方程式による初期条件フィッティング、`--explain` トレース、代数・数値両立 `--verify`、Lean 4 `geom_solution_iff` 形式証明、Typst 数式出力連携） |
| `ztrans` | `ztrans(2^n, n, z)` / `ztrans(n, n, z)` / `ztrans(cos(w*n), n, z)` | 記号的片側Z変換 $\mathcal{Z}\{f[n]\} = \sum_{n=0}^{\infty} f[n] z^{-n}$（基本信号 $\delta[n], u[n], a^n, n^k, \sin, \cos$、線形性、周波数・時間シフト則、積則の厳密代数変換） |
| `iztrans` | `iztrans(z / (z - 2), z, n)` / `iztrans(z / (z - 1)^2, z, n)` / `iztrans(1/z, z, n)` | 記号的因果的逆Z変換 $\mathcal{Z}^{-1}\{X(z)\}$（$X(z)/z$ 部分分数分解 PFD、仮有理関数の多項式除算 $\delta[n+k]$、原点極遅延、二項定理重根極展開による厳密時間領域復元） |
| `laplace` | `laplace(sin(2*t), t, s)` / `laplace(exp(3*t)*t^2)` | 記号ラプラス変換 $\mathcal{L}\{f(t)\}$（線形性、多項式・指数・三角関数・減衰振動および周波数シフト則の厳密代数変換） |
| `inv_laplace` | `inv_laplace(1/(s-3), s, t)` / `inv_laplace(1/(s^2+4))` | 記号逆ラプラス変換 $\mathcal{L}^{-1}\{F(s)\}$（部分分数分解 `apart` と留数極マッチングによる厳密時間領域復元） |
| `taylor` | `taylor(f, x, a, n)` | テイラー展開・マクローリン展開（点 $x=a$ まわりで $n$ 次まで展開） |
| `fourier_series` | `fourier_series(f, [t, L, n])` | 記号フーリエ級数展開（オイラー・フーリエ公式による厳密定積分、有限次数 $n$ までの三角級数部分和導出） |
| `residue` | `residue(1/(z^2*(z+1)), z, 0)` / `residue(exp(z)/z^3, z, 0)` | 複素解析における孤立特異点 $z_0$ まわりの厳密留数 $\mathrm{Res}(f, z_0)$（位数判定、テイラー級数漸化式、コーシー留数公式） |
| `gamma` | `gamma(5)` / `gamma(1/2)` / `gamma(-1/2)` | ガンマ関数 $\Gamma(z)$（正の整数で階乗 $(n-1)!$、半整数で $\sqrt{\pi}$ 展開、非正整数で極特異点検出） |
| `beta` | `beta(2, 3)` / `beta(1/2, 1/2)` | ベータ関数 $\mathrm{B}(p, q) = \frac{\Gamma(p)\Gamma(q)}{\Gamma(p+q)}$（整数・半整数の厳密有理・代数展開、例: $\mathrm{B}(1/2, 1/2) = \pi$） |
| `bernoulli` | `bernoulli(4)` / `bernoulli(10)` | 第 $n$ ベルヌーイ数 $B_n$（秋山・谷川アルゴリズムによる多倍長有理数厳密出力、例: $B_{10} = 5/66$） |
| `zeta` | `zeta(2)` / `zeta(4)` / `zeta(6)` | リーマンゼータ関数 $\zeta(s)$（オイラーの公式による正の偶数値の完全厳密代数解、例: $\zeta(4) = \pi^4/90$、極 $s=1$ 検出） |
| `piecewise` | `piecewise([[expr1, cond1], ...], default)` | 区分定義関数（第一級 `PiecewiseNode`）。CADセル分解と連携し、条件領域ごとの微分（`diff`）、定積分（`integrate`）、方程式求解（`solve`）を厳密代数処理 |
| `sum` | `sum(expr, k, start, end)` | 離散和（有限整数範囲の合算、または Faulhaber 公式による $n$ に関する多項式閉形式） |
| `gosper_sum` | `gosper_sum(t_k, k)` | 超幾何級数に対する Gosper 不定和法による閉形式原始関数 $z_k$ 導出（$z_{k+1}-z_k=t_k$） |
| `wz_cert` | `wz_cert(F, n, k)` | Wilf-Zeilberger (WZ) 対による超幾何恒等式の有理関数証明書 $R(n, k)$ 生成 |

---

### 4. 線形代数・3次元ベクトル解析

| 関数 | 書式・例 | 説明 |
| :--- | :--- | :--- |
| `det` | `det(A)` | 厳密行列式（余因子展開法、または整数行列に対するHadamard上界と有限体ガウス消去＋Garner CRTモジュラー合成による中間係数爆発防止型完全厳密計算） |
| `inv` | `inv(A)` | 厳密逆行列（余因子行列法による算出、特異行列時はエラー検出） |
| `transpose` | `transpose(A)` | 行列の転置（行と列の反転） |
| `rref` | `rref(A)` | 行簡約階段形（Reduced Row Echelon Form。Bareiss整数除算アルゴリズム） |
| `rank` | `rank(A)` | 行列の厳密な階数（ピボット列数） |
| `trace` / `tr` | `trace(A)`, `tr(A)` | 行列のトレース（主対角成分の総和 $\sum_{i=1}^n A_{i,i}$） |
| `eigenvals` | `eigenvals(A)` | 厳密固有値解析（Faddeev-LeVerrier法による特性多項式導出と代数方程式ソルバーによる最大4×4行列の厳密根号解・固有値リスト） |
| `eigenvects` | `eigenvects(A)` | 厳密固有ベクトル解析（RREF零空間Null Space基底の厳密抽出による固有空間正規直交系リスト） |
| `lu` | `lu(A)` | 行ピボット付き厳密PLU分解 $PA=LU$（$P$ 置換行列、$L$ 単位下三角、$U$ 上三角） |
| `qr` | `qr(A)` | 2段階グラム・シュミット法による厳密QR分解 $A=QR$（$Q$ 正規直交、$R$ 上三角、$Q^T Q = I$） |
| `cholesky` | `cholesky(A)` | 実対称正定値行列に対する厳密コレスキー分解 $A=LL^T$（厳密根号代数表現） |
| `ldlt` | `ldlt(A)` | 平方根フリー厳密 $LDL^T$ 分解（$L$ 単位下三角、$D$ 正の有理数対角行列） |
| `pinv` | `pinv(A)` | RREFフルランク分解によるムーア・ペンローズ擬似逆行列 $A^+$（小数を一切使わない有理厳密解） |
| `solve_linear` / `linsolve` | `solve_linear(A, b)` | 厳密連立一次方程式系ソルバー $Ax=b$（唯一解、不能、不定パラメータ解の厳密判別） |
| `lp_maximize` | `lp_maximize(obj, constraints)` | 厳密有理数2段階改訂シンプレックス法による線形計画最大化問題ソルバー（Blandの最小添字規則、双対変数・Farkas証明書・後退方向 ray 返却） |
| `lp_minimize` | `lp_minimize(obj, constraints)` | 厳密有理数2段階改訂シンプレックス法による線形計画最小化問題ソルバー |
| `hnf` | `hnf(A)` | 整数行列のエルミート標準形（Hermite Normal Form: HNF）。小数を一切使わない整数初等行変形と剰余簡約により行階段形（上三角型）$H$ を導出 |
| `hnf_transform` | `hnf_transform(A)` | エルミート標準形と単模変換行列 $[H, U]$ を同時返却（$U \in \text{GL}_m(\mathbb{Z}), U A = H$） |
| `snf` | `snf(A)` | 整数行列のスミス標準形（Smith Normal Form: SNF）。整数初等行・列変形により主対角成分が整除鎖 $d_1 \mid d_2 \mid \dots \mid d_r$ を満たす対角行列 $D$ を導出 |
| `snf_transform` | `snf_transform(A)` | スミス標準形と両側単模変換行列 $[D, U, V]$ を同時返却（$U \in \text{GL}_m(\mathbb{Z}), V \in \text{GL}_n(\mathbb{Z}), U A V = D$） |
| `invariant_factors` | `invariant_factors(A)` | 整数行列の不変因子リスト $[d_1, d_2, \dots, d_r]$（スミス標準形の正の対角要素）を抽出 |
| `abelian_group_structure` | `abelian_group_structure(A)` | 整数関係行列 $A$ が定める有限生成アーベル群 $G = \mathbb{Z}^m / \operatorname{Im}(A)$ の不変因子分解直和表現（例: `Z_2 + Z_6`、自由部分 `Z`）を同定 |
| `dot`, `cross` | `dot(u, v)`, `cross(u, v)` | ベクトル内積および3次元外積（クロス積） |
| `norm` | `norm(v)` | ベクトルのユークリッドノルム（$\sqrt{\sum v_i^2}$、根号自動簡約） |
| `grad` | `grad(f, [x, y, z])` | スカラー場の勾配ベクトル（$\nabla f$） |
| `div`, `curl` | `div(F, [x, y, z])`, `curl(F, ...)` | ベクトル場の発散（$\nabla \cdot F$）・3次元ベクトル場の回転（$\nabla \times F$） |
| 行列・ベクトル記法 | `[[1, 2], [3, 4]]`, `[1, 2, 3]` | 行列リテラルおよびベクトルリテラル |

---

### 5. 幾何学解析

| 関数 | 書式・例 | 説明 |
| :--- | :--- | :--- |
| `line_intersect` | `line_intersect([A1, B1, C1], [A2, B2, C2])` | 2直線 $Ax+By+C=0$ の交点（クラメルの公式による厳密交点 `[x, y]`） |
| `circle_intersect` | `circle_intersect(c1, r1, c2, r2)` | 2円の交点（中心座標と半径から根軸次数下げにより交点座標リストを算出） |
| `triangle_area` | `triangle_area(p1, p2, p3)` | 3頂点座標からなる三角形の厳密面積（外積・Shoelace公式） |
| `triangle_centers` | `triangle_centers(p1, p2, p3)` | 三角形の五心解析（重心・外心・垂心・内心をリストで一括算出） |
| `geo_prove` | `geo_prove(hypotheses, conclusion [, var_order])` | 呉文俊（Wu Wen-tsun）の代数的幾何自動定理証明法（Wu's Method）。仮設多項式系から標数集合（Characteristic Set: 三角形式アセンダント列）を導出し、結論多項式に対する擬除算剰余（Pseudo-division remainder）が 0 となるかを判定。3段階NDG条件（Primary / Essential / Algebraic）導出および Lean 4 形式証明トランスパイル対応 |
| `midpoint` | `midpoint(M, A, B)` | 点 $M$ が線分 $AB$ の中点（$2 M_x = A_x + B_x, 2 M_y = A_y + B_y$） |
| `collinear` | `collinear(A, B, C)` | 3点 $A, B, C$ が同一直線上に存在（共線条件: 外積 $(B_x - A_x)(C_y - A_y) - (B_y - A_y)(C_x - A_x) = 0$） |
| `parallel` | `parallel(A, B, C, D)` | 直線 $AB$ と $CD$ が平行（方向ベクトルの外積 $= 0$） |
| `perpendicular` | `perpendicular(A, B, C, D)` | 直線 $AB$ と $CD$ が垂直直交（方向ベクトルの内積 $= 0$） |
| `equal_length_sq` | `equal_length_sq(A, B, C, D)` | 線分長の平方一致関係（距離自乗の等値関係: $AB^2 = CD^2$） |
| `circle_concyclic` | `circle_concyclic(A, B, C, D)` | 4点 $A, B, C, D$ が同一円周上に存在（共円条件） |

---

### 6. 厳密離散確率・統計

| 関数 | 書式・例 | 説明 |
| :--- | :--- | :--- |
| `binom` | `binom(n, k, p)` | 二項分布の確率質量関数（PMF） $P(X=k) = \binom{n}{k} p^k (1-p)^{n-k}$ |
| `hyper` | `hyper(N, K, n, k)` | 超幾何分布の確率質量関数（非復元抽出の厳密有理数） |
| `geom` | `geom(p, k)` | 幾何分布の確率質量関数 $P(X=k) = (1-p)^{k-1} p$ |
| `bayes` | `bayes(prior, likelihood, marginal)` | ベイズの定理による厳密事後確率 $P(A\|B) = \frac{P(B\|A)P(A)}{P(B)}$ |
| `expect` | `expect(binom, n, p)` / `expect([[x1, p1], ...])` | 離散確率変数の期待値 $E[X]$ |
| `variance` | `variance(...)` | 離散確率変数の分散 $V[X] = E[X^2] - (E[X])^2$ |
| `stddev` | `stddev(...)` | 離散確率変数の標準偏差 $\sigma = \sqrt{V[X]}$（根号形式代数簡約） |

---

### 7. 前提条件システム（仮定）

| コマンド | 例 | 説明 |
| :--- | :--- | :--- |
| `assume` | `assume(x > 0)`, `assume(n, integer)` | ドメイン制約を設定（`sqrt(x^2)` → `x`, `sin(n*pi)` → `0` 等の簡約が活性化） |
| `unassume` | `unassume(x)` | 指定した変数の仮定制約を解除 |
| `assumptions` | `assumptions()` | 現在設定されている前提条件の一覧を表示 |
| `clear_assumptions` | `clear_assumptions()` | 設定されているすべての前提条件（仮定）を一括クリア |

---

### 8. 実代数幾何・数理論理・量化子消去

| 関数 | 書式・例 | 説明 |
| :--- | :--- | :--- |
| `qe` | `qe(forall([x], x^2 + a*x + b > 0))` | 円筒代数分解（CAD）基盤の決定論的完全量化子消去（Hong (1992) 境界多項式による同値な量化子なしパラメータ条件式の導出） |
| `forall` | `forall([x], formula)` | 全称量化子式 $\forall x \, \Phi(x)$ の構築 |
| `exists` | `exists([x], formula)` | 存在量化子式 $\exists x \, \Phi(x)$ の構築 |
| `cad` | `cad([x^2 - 2], [x])` | 円筒代数分解（CAD: Cylindrical Algebraic Decomposition）の直接実行（1次元セル分割・多変数射影と実根分離による半代数的集合表現） |

---

### 9. 楕円曲線代数・数論幾何

| 関数 | 書式・例 | 説明 |
| :--- | :--- | :--- |
| `ec_add` | `ec_add([A, B], P1, P2)` | ワイエルシュトラス標準形 $y^2 = x^3 + Ax + B$ 上の有理点加算（Chord and Tangent 法、単位元・無限遠点 `O` 対応） |
| `ec_mul` | `ec_mul([A, B], n, P)` | 楕円曲線上の有理点のスカラー倍算 $n P$（バイナリ Double-and-Add アルゴリズム） |
| `ec_torsion` | `ec_torsion(A, B)` | Nagell-Lutz の定理および Mazur の定理に基づく有限位数有理点群（捩れ群 $E(\mathbb{Q})_{\text{tors}}$）の完全決定アルゴリズム |
| `ec_p_order` | `ec_p_order(A, B, p)` | 素有限体 $\mathbb{F}_p$ 上の非特異楕円曲線 $y^2 \equiv x^3 + Ax + B \pmod p$ の群位数 $\#E(\mathbb{F}_p)$ を決定論的 Schoof アルゴリズム（分割多項式・フロベニウス写像・CRT合成）により多項式時間で算定 |
| `ec_p_trace` | `ec_p_trace(A, B, p)` | 素有限体 $\mathbb{F}_p$ 上の楕円曲線のフロベニウストレース $a = p + 1 - \#E(\mathbb{F}_p)$ を決定論的に算出（Hasseの定理 $|a| \le 2\sqrt{p}$ を満たす整数） |
| `ec_p_add` | `ec_p_add(A, B, p, P, Q)` | 素有限体 $\mathbb{F}_p$ 上の楕円曲線群における点加算 $P + Q$（$\pmod p$ の有理逆元とChord and Tangent幾何演算、単位元・無限遠点 `O` 対応） |
| `ec_p_mul` | `ec_p_mul(A, B, p, n, P)` | 素有限体 $\mathbb{F}_p$ 上の楕円曲線群における点のスカラー倍算 $n P$（$\pmod p$ 上のDouble-and-Add高速累乗算） |
| `ec_rank_bound` | `ec_rank_bound(A, B)` | 有理数体上の楕円曲線 $y^2 = x^3 + Ax + B$ に対し、有理2等分点を用いた2降下法（2-isogeny descent）によりセルマー群次元を算定し、モーデル・ヴェイユ階数の保証区間 $[r_{\text{lower}}, r_{\text{upper}}]$ を決定論的に算出 |
| `ec_generators` | `ec_generators(A, B)` | 2降下法の等質空間有理点探索により得られた無限位数の独立有理点集合（階数下限を成す自由部分群の生成元）を抽出 |

---

### 10. ビジュアル・検証ツール

| 関数 | 書式・例 | 説明 |
| :--- | :--- | :--- |
| `verify` | `verify(integrate(1/(x^2+1), x), atan(x))` | 2つの式の代数的同値性を独立検証し、代数的証明書（Certificate）を発行 |
| `plot` | `plot(sin(x), [-pi, pi])` | Braille 2×4 サブピクセル高解像度ターミナルプロット（零点・極値自動検出・特異点漸近線偽結合防止） |

---

### 11. 格子基底縮小・代数的整数関係 (LLL)

| 関数 | 書式・例 | 説明 |
| :--- | :--- | :--- |
| `lll` | `lll([[1, -1, 3], [1, 0, 5], [1, 2, 6]])` | Lenstra–Lenstra–Lovász (1982) 格子基底縮小アルゴリズムによる簡約基底の厳密有理数算定 |
| `find_min_poly` | `find_min_poly(sqrt(2) + sqrt(3), 4)` | LLL格子縮小を用いた代数的数の有理数係数最小多項式（Minimal Polynomial）の完全逆算 |

---

### 12. p進数演算・局所代数・ヘンゼル補題

| 関数 | 書式・例 | 説明 |
| :--- | :--- | :--- |
| `padic_val` | `padic_val(50, 5)` | 有理数 $x$ に対する素数 $p$ を底とする $p$進付値 $v_p(x) \in \mathbb{Z}$ の完全厳密計算（$x=0$ は無限大） |
| `padic_norm` | `padic_norm(3/20, 2)` | 有理数 $x$ に対する超距離ノルム $|x|_p = p^{-v_p(x)}$ の完全厳密有理数（`big.Rat`）計算 |
| `padic_expand` | `padic_expand(2/3, 5, 4)` | 有理数 $x$ の指定精度 $k$ における $p$進切断級数展開 $a_0 + a_1 p + a_2 p^2 + \dots + O(p^k)$ の文字列生成 |

---

### 13. 局所代数級数・特異点解析（ピュイズー級数）

| 関数 | 書式・例 | 説明 |
| :--- | :--- | :--- |
| `puiseux` | `puiseux(y^2 - x^3, y, x, 3)` | 代数方程式 $F(x, y) = 0$ または多項式に対し、ニュートン多角形アルゴリズム（Newton Polygon）を用いて特異点・分岐点周りのピュイズー代数的分数冪級数解 $y(x) = \sum c_k x^{p_k/q_k}$ の局所分枝を完全決定論的導出（デフォルト次数3） |

---

### 14. ガロア理論・代数的根号可解性・アーベル・ルフィニ代数的証明書

| 関数 | 書式・例 | 説明 |
| :--- | :--- | :--- |
| `galois_group` | `galois_group(x^3 - 2)` / `galois_group(p, x)` | 有理数体 $\mathbb{Q}$ 上の次数5以下の多項式に対するガロア群 $\mathrm{Gal}(f/\mathbb{Q})$ の決定論的算定（無平方分解・判別式・3次/6次レゾルベント根探索・Dedekind-Frobenius巡回置換型正の証拠により、$S_n, A_n, D_n, C_n, V_4, F_{20}$ などの置換群構造を同型判定・特定） |
| `is_solvable_by_radicals` | `is_solvable_by_radicals(x^5 - 4*x + 2)` | ガロアの基本定理に基づく多項式方程式の代数的根号可解性の決定論的判定。可解群の場合は `true`、非可解（$A_5, S_5$）の場合は `false` を返し、`--verify` 時には独立検証可能なアーベル・ルフィニ代数的不可解性証明書（Abel-Ruffini Impossibility Certificate IR）、`--lean` 時には Lean 4（Mathlib4）形式の反証不可能定理コードを出力 |

---

### 15. 自己検証証明書 & 形式証明基盤（ProofTrace / Explain / Verify / Lean 4）

| 機能・オプション | 書式・例 | 説明 |
| :--- | :--- | :--- |
| `ProofTrace` | 内部データ構造 | 計算過程の各ステップ（Before/After、適用規則、残差式、検証状態）を保持するデータ構造。途中式表示、逆算検算、Lean 4 証明コード生成の共通表現として機能 |
| `--explain` / `explain` / `steps` | `ihd --explain "1/(sqrt(2)+1)"` | 導出ステップ列に基づき、代数的項書き換え（二重根号分解、有理化、因数分解、微分則等）をツリー形式で表示 |
| `--verify` / `verify` | `ihd --verify "factor(x^2 - 1)"` | 各ステップや結果に対し、定義式や導関数に基づく逆算検算を行い、残差が0であることを確認 |
| `--certificate` | `ihd --certificate [file] "factor(x^2 - 1)"` | 計算結果と導出ステップ列を証明書 JSON（スキーマ `ihd-proof-trace-v1`）として出力またはファイルへ保存 |
| `--replay` | `ihd --replay cert.json` | 保存された証明書 JSON を読み込み、各ステップの残差が0かを再検証 |
| `micro-verifier` | `micro-verifier cert.json` | 外部依存ゼロ・Go 標準ライブラリのみで実装された極小独立検証バイナリ。証明書 JSON（`ihd-proof-trace-v1`）を読み込み、CAS本体に依存せず残差ゼロを客観的に再検証 |
| `--lean` / `--lean-file` | `ihd --lean "factor(x^2 - 1)"` / `ihd --lean-file p.lean "factor(x^2 - 1)"` | 計算結果や導出ステップを **Lean 4（Mathlib4）** の形式証明コード（`by ring`, `by ext <;> fin_cases`, `by repeat constructor <;> norm_num`, `by simp` 等）として出力 |
| `--typst` / `--format typst` | `ihd --typst "1/2 + sqrt(2)"` / `ihd --format typst "[[1, 2], [3, 4]]"` | **Typst** の数式モード記法（`$ ... $`、分数 `/`、根号 `sqrt(...)`/`root(...)`、行列 `mat(...)` 等）で出力 |
| `--explain --lean` | `ihd --explain --lean "factor(x^4 + 4)"` | 単一の証明DAGから **Lean 4（Mathlib4）** の推移的等式連鎖 `calc` ブロック（`by ring`, `by norm_num` 等）を生成（Zero-sorry 契約遵守） |
| `--explain --typst` | `ihd --explain --typst "expand((x + 1)^2)"` | 単一の証明DAGから **Typst** 組版による数学導出レポートドキュメント（`= Mathematical Derivation Report`）を生成 |

---

### 16. クリフォード幾何代数 & 二進円分環・厳密量子回路等価性検証

直交対角計量 $(p, q, r)$（$p$ 個の正基底 $e_i^2 = +1$、$q$ 個の負基底 $e_i^2 = -1$、$r$ 個の退化・零基底 $e_i^2 = 0$）を持つ実クリフォード幾何代数 $\mathrm{Cl}(p, q, r)$ の演算系、および二進円分環 $\mathbb{Z}[1/\sqrt{2}, i]$ 上の厳密量子等価性判定を提供します。小数は一切介入せず、有理数係数および代数体上で厳密に計算されます。

| 関数 | 書式・例 | 説明 |
| :--- | :--- | :--- |
| `clifford` | `clifford(e0 * e1 + e1 * e0, 3, 0)` / `clifford(e0 * e0, 3, 0)` | 計量 $(p, q, [r])$ におけるクリフォード幾何積およびマルチベクトルの代数展開・正準簡約（基底自乗簡約・反交換関係整理）。基底記号は `e0`, `e1`, `e2` ... を使用 |
| `clifford_wedge` | `clifford_wedge(e0, e1, 3, 0)` | グラスマン外積（ウェッジ積） $A \wedge B$。計量 $(p, q, [r])$ 上で完全反对称化された次数ブレードを算出 |
| `clifford_contract` | `clifford_contract(e0, e0, "left", 3, 0)` | 左右縮約（内積拡張）。第3引数で方向（`"left"`: 左縮約 $\lrcorner$、`"right"`: 右縮約 $\llcorner$）を指定 |
| `clifford_dual` | `clifford_dual(e3, 3, 0, 1)` | ポアンカレ補集合双対（Poincaré Complement Dual）。擬スカラー単位ブレード $I$ の縮約による双対マルチベクトルの導出（退化計量 $r > 0$ にも完全対応） |
| `quantum_equiv` | `quantum_equiv(c1, c2, n, [k])` | 二進円分環 $\mathbb{Z}[1/\sqrt{2}, i]$ を用いた小数ゼロの Clifford+T 量子回路厳密等価性判定。$n$ 量子ビット回路 $c1, c2$ が大域位相 $\omega^k$（$\omega = e^{i\pi/4}$）を除いて等価であるかを判定。不等価時は反例基底状態（Witness State）を出力 |
| `quantum_eval` | `quantum_eval(circuit, numQubits)` | Clifford+T 量子回路の記号ユニタリ行列を二進円分環係数で厳密シミュレーション・導出 |

---

### 17. CAS反復制御構文 & 任意基数進数変換

リスト内包表記、総乗計算、手続き型反復、および任意基数（2〜64進法）リテラル・進数変換を提供します。

| 関数・構文 | 書式・例 | 説明 |
| :--- | :--- | :--- |
| `table` | `table(x^2, x, 1, 5)` / `table(i + j, i, 1, 3, 1, j, 1, 2)` | Mathematica 準拠のリスト内包表記・反復リスト生成。引数 `table(expr, var, start, end, [step])` による動的スコープ反復 |
| `product` | `product(k, k, 1, 5)` / `product(2*k + 1, k, 0, 4)` | 離散総乗（乗乗計算 $\prod_{k=\text{start}}^{\text{end}} \text{expr}$）。動的スコープ保護およびゼロ短絡評価対応 |
| `for` | `for(i, 1, 5, s = s + i)` | 手続き型反復ループ。引数 `for(var, start, end, [step], body)` による順次実行（返却値は最終式の評価結果） |
| `to_base` | `to_base(255, 16)` / `to_base(1/3, 2)` | 任意基数（2〜64）への基底変換文字列生成。有理数の場合は循環小数表記を出力 |
| `from_base` | `from_base("FF", 16)` / `from_base("101.1", 2)` | 任意基数（2〜64）の文字列から厳密有理数・整数への復元 |
| `bin`, `oct`, `hex` | `bin(42)`, `oct(42)`, `hex(255)` | 2進数、8進数、16進数への標準表記変換ショートカット |
| 基数リテラル | `16^^FF`, `2^^1011`, `0x1A`, `0b101` | 入力字句解析器による任意進数リテラル記法（$2 \le \text{base} \le 64$） |

---

### 18. REPL 構文ハイライト・テーマ・対話環境コマンド

対話型 REPL セッション内でリアルタイムに実行できる制御コマンド、カラーパレットテーマ、および構文ハイライト機能です。

| コマンド / 設定 | 書式・例 | 説明 |
| :--- | :--- | :--- |
| `theme` | `theme` | 現在適用されているカラーテーマ（`dark` / `light` / `none`）を表示 |
| `theme dark` | `theme dark` | 暗め背景端末向けカラーパレット（明るいシアン・マゼンタ・グリーン・ブルーのレインボー括弧、黄色数値、白演算子）へ切り替え、`~/.ihd/config.json` に永続化 |
| `theme light` | `theme light` | 明るめ・白背景端末向け高コントラストパレット（濃い青・マゼンタ・緑・シアンのレインボー括弧、赤数値、黒演算子）へ切り替え、永続化 |
| `highlight` | `highlight` / `color` | 現在の構文ハイライト有効/無効ステータスを表示 |
| `highlight on` / `off` | `highlight on` / `highlight off` | リアルタイム構文ハイライトおよびネスト深度別レインボー括弧の有効化・無効化を動的切り替え |
| `warn-loop` | `warn-loop <N>` / `warn-loop` | 反復ループの非破壊警告ステップ数を指定・表示し、`~/.ihd/config.json` に永続化（デフォルト: 100,000） |
| `max-loop` | `max-loop <N>` / `max-loop` | 反復ループの最大安全上限ステップ数を指定・表示し、`~/.ihd/config.json` に永続化（デフォルト: -1:無制限） |
| `config loop` | `config loop` | 現在の反復ループ警告閾値および最大安全上限の構成値を表示 |
| `vars` | `vars` | 現在の REPL セッション環境に束縛されている変数・定数の一覧を表示 |
| `help` | `help` / `help <関数名>` | REPL 内簡易ヘルプおよび指定した組み込み関数の完全引数シグネチャ・説明を表示 |
| `search` | `search <キーワード>` | 関数名または説明文からキーワード部分一致検索を行い、該当する関数とシグネチャを一覧表示 |
| `whatis` | `whatis <数式または概念>` | 指定された関数名・数学用語（楕円曲線、グレブナー基底など）や数式（方程式、多項式など）から適用可能な組み込み関数や数学概念を解説 |
| `similar` | `similar(<数式>)` / `similar <数式>` | 履歴内の過去の入力から AST 構造ハッシュに基づいて類似する数式を検索・類似度順に表示 |
| `save` | `save <ファイルパス>` | 現在の REPL セッション環境（変数および仮定 conditions）を `.ihd` 形式で永続化保存 |
| `load` | `load <ファイルパス>` | 保存された `.ihd` ファイルからセッション環境を再現復元 |
| `clear` / `reset` | `clear` / `reset` | セッション環境内のすべての変数および仮定を初期状態にリセット |
| `exit` / `quit` | `exit` / `quit` / `Ctrl+D` | REPL セッションを終了 |
| CLI オプション | `ihd --theme light` / `--no-color` / `--warn-loop <N>` / `--max-loop <N>` | 起動時のテーマ指定、カラー無効化、反復警告・最大上限指定フラグ |

---

### 19. 多変数多項式イデアル・準素分解 (GTZ法)

| 関数 | 書式・例 | 説明 |
| :--- | :--- | :--- |
| `primary_decomposition` | `primary_decomposition([x^2 - 1, y], [x, y])` | 多変数多項式イデアル $I = \langle p_1, \dots \rangle$ に対し、Gianni-Trager-Zacharias (GTZ) 法により準素分解 $I = \bigcap Q_i$ を計算し、各準素成分 $Q_i$ とその素イデアル $P_i = \sqrt{Q_i}$ のペアリスト `[[Q_1, P_1], ...]` を返却。第3引数で単項式順序（`"grevlex"` または `"lex"`）を指定可能 |
| `ideal_radical` | `ideal_radical([x^2, x*y], [x, y])` | イデアル $I$ の根基イデアル $\sqrt{I}$ の生成元リストを計算（零次元成分にはSeidenbergの補題を適用） |
| `ideal_dimension` | `ideal_dimension([x*y, x*z], [x, y, z])` | グレブナー基底の先頭単項式イデアルから最大独立変数集合を求め、代数多様体（イデアル）のKrull次元を整数で返却 |
| `ideal_intersection` | `ideal_intersection([x], [y], [x, y])` | 2つのイデアルの交わり $I \cap J$ を補助変数消去法により計算 |
| `ideal_colon` | `ideal_colon([x^2], x, [x])` | 多項式 $f$ による商イデアル（コロンイデアル） $I : f = \{ g \mid f g \in I \}$ を計算 |
| `ideal_saturation` | `ideal_saturation([x^2], x, [x])` | 多項式 $f$ による飽和イデアル $I : f^\infty$ を逐次計算し、安定化指数 $m$ で終端した生成元リストを返却 |

---

### 20. コレクション・リスト操作言語拡張 & 第一級ラムダ式

リスト・コレクションに対するシーケンス生成、高階関数適用、述語フィルタリング、第一級ラムダ式、およびインデックス・スライス構文を提供します。

| 関数・構文 | 書式・例 | 説明 |
| :--- | :--- | :--- |
| `range` | `range(5)` / `range(2, 6)` / `range(0, 1, 1/3)` | 等差数列のリストを生成。引数 `range(stop)` または `range(start, stop, [step])`。step には整数および有理数分数を指定可能（step = 0 はエラー） |
| `map` | `map(sin, [0, pi/2, pi])` / `map(lambda(x, x^2), range(5))` | コレクションの各要素に関数またはラムダ式を適用した新しいリストを返却。引数 `map(fn, list)`。走査時は親環境変数を保護するシャドウスコープを適用 |
| `filter` | `filter(is_prime, range(1, 11))` / `filter(lambda(x, x > 2), [1, 2, 3, 4])` | 述語関数または真偽値を返すラムダ式を満たす要素のみを抽出したリストを返却。引数 `filter(pred, list)` |
| `lambda` | `lambda(x, x^2)` / `lambda([x, y], x + y)` | 第一級無名関数（ラムダ式）を定義。引数 `lambda(param, body)` または `lambda([p1, p2, ...], body)`。即時呼び出し `lambda(x, x+1)(5)` や変数代入後の呼び出し `f = lambda(x, x^2); f(3)` に対応 |
| 単一インデックス | `list[0]`, `list[-1]` | 0-indexed による要素取得。負のインデックスは末尾からの相対位置（$-1$ は末尾要素）。範囲外アクセス時は IndexError を送出 |
| スライス記法 | `list[start:stop]`, `list[start:stop:step]` | CPython `PySlice_GetIndicesEx` 準拠の部分リスト抽出。全8パターン（`[:]`, `[:end]`, `[start:]`, `[start:end]`, `[::step]`, `[start::step]`, `[:end:step]`, `[start:end:step]`）に対応。負のインデックス・逆順ステップ（step < 0）および範囲外クリッピングに対応（step = 0 はエラー） |

---

### 21. 物理単位計算 & 次元解析 (SI基本単位・組立単位・単位換算)

有理数体 $\mathbb{Q}$ 上の7次元ベクトル空間（SI基本7単位）に基づき、浮動小数点数を使わずに物理単位の加減乗除、単位換算、および次元整合性検証を行います。

#### 対応単位・接頭辞
* **SI基本7単位**: `m` (長さ), `kg` (質量), `s` (時間), `A` (電流), `K` (温度), `mol` (物質量), `cd` (光度)
* **組立単位**: `N` (力), `J` (エネルギー), `W` (仕事率), `Pa` (圧力), `Hz` (周波数), `C` (電荷), `V` (電位差), `ohm` (電気抵抗), `F` (静電容量), `S` (コンダクタンス), `Wb` (磁束), `T` (磁束密度), `H` (インダクタンス)
* **SI併用単位**: `min` (分, $60\text{ s}$), `h` / `hour` (時, $3600\text{ s}$), `d` / `day` (日, $86400\text{ s}$), `g` (グラム, $1/1000\text{ kg}$), `t` (トン, $1000\text{ kg}$), `L` / `l` (リットル, $1/1000\text{ m}^3$)
* **SI接頭辞**: `Y` ($10^{24}$), `Z` ($10^{21}$), `E` ($10^{18}$), `P` ($10^{15}$), `T` ($10^{12}$), `G` ($10^9$), `M` ($10^6$), `k` ($10^3$), `h` ($10^2$), `da` ($10^1$), `d` ($10^{-1}$), `c` ($10^{-2}$), `m` ($10^{-3}$), `u` ($10^{-6}$), `n` ($10^{-9}$), `p` ($10^{-12}$), `f` ($10^{-15}$), `a` ($10^{-18}$)

| 関数・構文 | 書式・例 | 説明 |
| :--- | :--- | :--- |
| `[unit]` 記法 | `10 [m/s]`, `5 [kg*m/s^2]`, `(1/4) [h]` | 数値または数式に単位を付与。単位式内では乗算 `*`、除算 `/`、累乗 `^` が使用可能。分数の場合は `(1/4) [h]` のように括弧で括る |
| `unit` | `unit(m/s)`, `unit(N)` | 単位表現から値 1 の物理量を生成 |
| `convert_unit` | `convert_unit(100 [km/h], m/s)`, `convert_unit((1/4) [h], min)` | 物理量を同一の次元を持つ目標単位へ厳密有理数比率で換算。次元が異なる場合はエラー |
| 代数演算 | `10 [m/s] * 5 [s]` $\to$ `50 [m]`, `1 [km] - 200 [m]` $\to$ `4/5 [km]` | 加減算時は次元が一致していることを検証。乗除算時は有理数次数ベクトルを加減算。すべての次数が0となった場合は通常数値へ自動縮約 |
| 超越関数ガード | `sin(10 [m])` $\to$ エラー | `sin`, `cos`, `exp`, `ln` 等の初等超越関数に有次元の引数を渡すとエラーを返却 |
| 検証 (`--verify`) | `ihd --verify "convert_unit(100 [km/h], m/s)"` | 換算前後の物理量をSI基本単位に展開し、残差が0であることを独立検証 |






---

### 22. 代数パターンマッチング & 項書き換え規則 (TRS)

アンダースコア始まりのワイルドカード変数（_x, _a）を用いたパターンマッチング（非線形パターン検査、加算・乗算の結合・可換 AC 部分マッチング対応）および、ユーザー定義規則の適用・不動点反復エンジンを提供します。

| 関数・構文 | 形式・例 | 説明 |
| :--- | :--- | :--- |
| _var | _x, _a, _1 | 任意の式木にマッチするワイルドカード変数。同一ワイルドカードが複数箇所に出現する場合は構造等価性を検証（非線形パターン） |
| 
ule | 
ule(sin(_x)^2, 1 - cos(_x)^2) | パターン（左辺）と置換テンプレート（右辺）を保持する第一級ルールオブジェクトを作成 |
| 
eplace | 
eplace(sin(t)^2, rule(sin(_x)^2, 1 - cos(_x)^2)) | 式木をボトムアップ走査し、マッチした部分式を1回置換 |
| 
ewrite | 
ewrite(sin(sin(x)), rule(sin(_t), _t)) / 
ewrite(expr, [r1, r2], [max_steps]) | 不動点（どのルールも適用できなくなる状態）に達するまで繰り返し書き換えを適用。循環検出および ~/.ihd/config.json（
ewrite_max_steps、既定値1000）による安全上限ガード付き |
