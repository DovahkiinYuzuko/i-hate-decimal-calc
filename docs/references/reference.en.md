# ihd Complete Syntax, Operators & Functions Reference (English Edition)

Complete language specification and built-in function reference for `ihd` (i-hate-decimal-calc), a strict CAS calculator that rejects all decimal approximations and computes strictly using arbitrary-precision rational numbers and algebraic symbols.

---

## Table of Contents
- [Operators & Precedence](#operators--precedence)
- [Reserved Constants](#reserved-constants)
- [Built-in Functions by Category](#built-in-functions-by-category)
  - [1. Basic Algebra, Number Theory & Equations](#1-basic-algebra-number-theory--equations)
  - [2. Trigonometric, Logarithmic & Complex Functions](#2-trigonometric-logarithmic--complex-functions)
  - [3. Calculus, Differential Equations & Discrete Summation](#3-calculus-differential-equations--discrete-summation)
  - [4. Linear Algebra & 3D Vector Calculus](#4-linear-algebra--3d-vector-calculus)
  - [5. Computational Geometry](#5-computational-geometry)
  - [6. Exact Discrete Probability & Statistics](#6-exact-discrete-probability--statistics)
  - [7. Symbolic Assumptions System](#7-symbolic-assumptions-system)
  - [8. Real Algebraic Geometry, Mathematical Logic & Quantifier Elimination](#8-real-algebraic-geometry-mathematical-logic--quantifier-elimination)
  - [9. Elliptic Curves & Arithmetic Geometry](#9-elliptic-curves--arithmetic-geometry)
  - [10. Visualization & Verification Tools](#10-visualization--verification-tools)

---

## Operators & Precedence

Implicit multiplication (e.g., `2pi` or `(1+2)(3+4)`) is strictly prohibited to prevent syntactic ambiguity. Explicit `*` is required.

| Rank | Operator | Associativity | Description & Examples |
| :---: | :---: | :---: | :--- |
| 1 | `()` | - | Grouping parentheses |
| 2 | `!` | Postfix Unary | Factorial of non-negative integers (e.g., `5! = 120`) |
| 3 | `^` | **Right-associative** | Exponentiation (e.g., `2^3^2 = 2^(3^2) = 512`) |
| 4 | `-` | Prefix Unary | Unary negation (Lower precedence than `^`: `-3^2 = -(3^2) = -9`) |
| 5 | `*`, `/` | Left-associative | Multiplication and division (Explicit `*` required) |
| 6 | `+`, `-` | Left-associative | Addition and subtraction |
| 7 | `<`, `<=`, `>`, `>=`, `==`, `!=` | None | Relational operations & inequalities (Rigorous evaluation via rational interval arithmetic) |

---

## Reserved Constants

| Identifier | Symbol | Description |
| :--- | :---: | :--- |
| `pi`, `π` | $\pi$ | Archimedes' constant (Ratio of circumference to diameter) |
| `e` | $e$ | Euler's number (Base of natural logarithm) |
| `i` | $i$ | Imaginary unit ($i^2 = -1$) |
| `inf`, `infinity` | $\infty$ | Positive infinity (Used in limits `limit`, root isolation bounds, etc.) |
| `-inf` | $-\infty$ | Negative infinity |
| `O` | $\mathcal{O}$ | Group identity / point at infinity for elliptic curve operations |
| `deg` | - | Degree conversion constant ($\pi/180$. Allows `sin(30*deg)`) |
| `delta` | $\delta$ | Dirac delta generalized function (Laplace transforms) |
| `true`, `false` | - | Boolean logical constants (Relational operations, QE outcomes) |

---

## Built-in Functions by Category

### 1. Basic Algebra, Number Theory & Equations

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
| `apart` | `apart(expr)` / `apart(expr, var)` | Partial fraction decomposition via Kung-Tong and Henrici algorithms |
| `together` | `together(expr)` | Rational fraction common denominator consolidation |
| `poly_gcd` | `poly_gcd(p, q)` / `poly_gcd(p, q, var)` | Multivariate polynomial exact GCD via Collins/Brown Subresultant PRS algorithm |
| `poly_lcm` | `poly_lcm(p, q)` / `poly_lcm(p, q, var)` | Multivariate polynomial exact LCM via $\frac{p \cdot q}{\mathrm{GCD}(p, q)}$ |
| `resultant` | `resultant(p, q)` / `resultant(p, q, var)` | Sylvester resultant and algebraic elimination via Sylvester's Dialytic Method |
| `groebner` | `groebner([p1, p2, ...], [x, y, ...])` / `groebner(..., [order])` | Reduced Gröbner basis for polynomial ideals (Buchberger 1965/1979/1985, Gebauer-Möller 1988 criteria, Giovini 1991 Sugar strategy. Orders: `lex`/`grevlex`) |
| `sturm` | `sturm(p)` / `sturm(p, var)` | Exact Sturm sequence (chain) via Primitive PRS over $\mathbb{Q}[x]$ |
| `root_count` | `root_count(p)` / `root_count(p, var, [a, b])` | Exact number of distinct real roots in interval $[a, b]$ or $(-\infty, \infty)$ via Sturm's Theorem |
| `isolate_roots` | `isolate_roots(p)` / `isolate_roots(p, var, [a, b])` | Exact real root isolation into disjoint rational intervals via square-free bisection |
| `inv_mod` | `inv_mod(a, m)` | Modular inverse via Extended Euclidean Algorithm ($a x \equiv 1 \pmod m$) |
| `crt` | `crt([r1, r2], [m1, m2])` | Chinese Remainder Theorem (Garner's algorithm & generalized non-coprime CRT) |
| `totient` | `totient(n)` | Euler's totient function $\phi(n) = n \prod_{p \mid n} (1 - 1/p)$ |
| `is_prime` | `is_prime(n)` | Deterministic primality test (Sorenson & Webster 12 bases for 64-bit, Baillie-PSW for big ints) |
| `solve` | `solve(expr, var)` | Exact algebraic equation solver for linear, quadratic, and cubic equations |
| `to_poly` | `to_poly(expr, [x, y], "lex")` | Explicitly converts an expression to a canonical `PolyNode` under specified variables and monomial order (`lex`, `grevlex`) |
| `to_alg` | `to_alg(rep, min_poly, [var])` | Constructs an algebraic number node `AlgNode` in field extension $\mathbb{Q}(\alpha) \cong \mathbb{Q}[x]/\langle m(x) \rangle$ |
| `alg_inv` | `alg_inv(rep, min_poly)` | Computes multiplicative inverse $\beta^{-1} \in \mathbb{Q}(\alpha)$ via Extended Euclidean Algorithm |
| `min_poly` | `min_poly(rep, min_poly)` | Derives the irreducible monic minimal polynomial $m(x) \in \mathbb{Q}[x]$ of an algebraic element |

---

### 2. Trigonometric, Logarithmic & Complex Functions

| Function | Syntax & Example | Description |
| :--- | :--- | :--- |
| `sin`, `cos`, `tan` | `sin(pi/6)` | Trigonometric functions (exact algebraic evaluation for rational multiples of $\pi$) |
| `asin`, `acos`, `atan` | `asin(1/2)` | Inverse trigonometric functions (exact values for special angles, principal branch) |
| `trig_expand` | `trig_expand(sin(x+y))` / `trig_expand(sin(2*x))` | Expands trigonometric functions using addition theorems and multiple-angle formulas |
| `trig_reduce` | `trig_reduce(sin(x)^2)` / `trig_reduce(sin(x)+cos(x))` | Reduces powers (half-angle), products to sums, and harmonic linear combinations |
| `exp` | `exp(x)` | Natural exponential function ($e^x$ exact symbolic representation with exponential multiplication rules) |
| `log` | `log(x)` / `log(base, x)` | Common logarithm (base 10) and arbitrary base logarithm |
| `ln` | `ln(x)` | Natural logarithm (base $e$) |
| `arg` | `arg(z)` | Exact principal complex argument $\theta \in (-\pi, \pi]$ |
| `polar` | `polar(1 + i)` | Polar form conversion $r(\cos\theta + i\sin\theta)$ (e.g., `√2*(cos(π/4) + i*sin(π/4))`) |
| `polar_exp` | `polar_exp(1 + i)` | Exponential polar conversion $r e^{i\theta}$ via Euler's formula |
| `rect` | `rect(sqrt(2), pi/4)` | Converts polar form $(r, \theta)$ to rectangular form $a + bi$ |

---

### 3. Calculus, Differential Equations & Discrete Summation

| Function | Syntax & Example | Description |
| :--- | :--- | :--- |
| `diff` | `diff(sin(x)*x, x)` / `diff(x^4, x, 2)` | Exact symbolic differentiation (Product, Quotient, Chain rules, and higher-order derivatives) |
| `integrate` | `integrate(x^2, x)` / `integrate(sin(x), x, 0, pi)` | Exact symbolic indefinite integration and definite integration over $[a, b]$ |
| `risch_integrate` | `risch_integrate(x*exp(x^2), x)` | Deterministic Risch algorithm (transcendental extension & Rothstein-Trager method) for exact indefinite integration |
| `limit` | `limit(sin(x)/x, x, 0)` / `limit(1/x, x, 0, 1)` | Exact symbolic limit computation (indeterminate forms, factoring, L'Hopital's rule, one-sided limits) |
| `dsolve` | `dsolve(diff(y, x) == y, y, x)` / `dsolve(diff(y, x, 2) + 4*y == 0, y, x)` | Symbolic ODE solver (1st-order linear via integrating factor, 2nd-order linear with constant coefficients and undetermined coefficients) |
| `rsolve` | `rsolve(a(n+1) == 2*a(n) + 1, a(n), [a(1) == 1])` / `rsolve(a(n+2) == a(n+1) + a(n), a(n), [a(0) == 0, a(1) == 1])` | Linear recurrence relation solver (1st and 2nd order linear recurrences with constant coefficients, characteristic roots analysis, undetermined coefficients, and linear initial condition fitting) |
| `laplace` | `laplace(sin(2*t), t, s)` / `laplace(exp(3*t)*t^2)` | Exact symbolic Laplace transform $\mathcal{L}\{f(t)\}$ (linearity, polynomial, exponential, trigonometric, damped oscillation & frequency shifting) |
| `inv_laplace` | `inv_laplace(1/(s-3), s, t)` / `inv_laplace(1/(s^2+4))` | Exact symbolic inverse Laplace transform $\mathcal{L}^{-1}\{F(s)\}$ (via partial fraction decomposition `apart` and pole matching) |
| `taylor` | `taylor(f, x, a, n)` | Taylor / Maclaurin series expansion around $x=a$ up to order $n$ |
| `fourier_series` | `fourier_series(f, [t, L, n])` | Symbolic Fourier series expansion (computes exact definite integrals via Euler-Fourier formulas up to order $n$) |
| `residue` | `residue(1/(z^2*(z+1)), z, 0)` / `residue(exp(z)/z^3, z, 0)` | Exact algebraic residue $\mathrm{Res}(f, z_0)$ at isolated singularity $z_0$ in complex analysis (order determination, Taylor recurrence & Cauchy formula) |
| `gamma` | `gamma(5)` / `gamma(1/2)` / `gamma(-1/2)` | Gamma function $\Gamma(z)$ (factorial $(n-1)!$ for positive integers, $\sqrt{\pi}$ expansion for half-integers, pole detection for non-positive integers) |
| `beta` | `beta(2, 3)` / `beta(1/2, 1/2)` | Beta function $\mathrm{B}(p, q) = \frac{\Gamma(p)\Gamma(q)}{\Gamma(p+q)}$ (exact rational or algebraic expansion for integers/half-integers, e.g. $\mathrm{B}(1/2, 1/2) = \pi$) |
| `bernoulli` | `bernoulli(4)` / `bernoulli(10)` | $n$-th Bernoulli number $B_n$ (exact arbitrary-precision rational via Akiyama-Tanigawa algorithm, e.g. $B_{10} = 5/66$) |
| `zeta` | `zeta(2)` / `zeta(4)` / `zeta(6)` | Riemann zeta function $\zeta(s)$ (exact algebraic closed forms for positive even integers via Euler formula, e.g. $\zeta(4) = \pi^4/90$, pole detection at $s=1$) |
| `sum` | `sum(expr, k, start, end)` | Discrete summation (finite sum or exact polynomial closed form via Faulhaber formula) |
| `gosper_sum` | `gosper_sum(t_k, k)` | Gosper's algorithm for hypergeometric summation: finds closed-form antiderivative $z_k$ such that $z_{k+1}-z_k=t_k$ |
| `wz_cert` | `wz_cert(F, n, k)` | Generates rational function certificate $R(n, k)$ for hypergeometric identity verification via Wilf-Zeilberger (WZ) theory |

---

### 4. Linear Algebra & 3D Vector Calculus

| Function | Syntax & Example | Description |
| :--- | :--- | :--- |
| `det` | `det(A)` | Exact determinant of a matrix via division-free Laplace expansion |
| `inv` | `inv(A)` | Exact inverse matrix via adjugate matrix method |
| `transpose` | `transpose(A)` | Matrix transpose (swaps rows and columns) |
| `rref` | `rref(A)` | Reduced Row Echelon Form via Bareiss fraction-free elimination |
| `rank` | `rank(A)` | Exact matrix rank (number of pivot columns) |
| `trace` / `tr` | `trace(A)`, `tr(A)` | Matrix trace (sum of main diagonal entries $\sum_{i=1}^n A_{i,i}$) |
| `eigenvals` | `eigenvals(A)` | Exact eigenvalue analysis via Faddeev-LeVerrier characteristic polynomial and algebraic solver |
| `eigenvects` | `eigenvects(A)` | Exact eigenvector analysis via RREF nullspace basis extraction |
| `lu` | `lu(A)` | Exact PLU decomposition $PA=LU$ with row pivoting ($P$ permutation, $L$ unit lower, $U$ upper) |
| `qr` | `qr(A)` | Exact QR decomposition $A=QR$ via Two-Stage Gram-Schmidt with exact radicals ($Q^T Q = I$) |
| `cholesky` | `cholesky(A)` | Exact Cholesky decomposition $A=LL^T$ for symmetric positive-definite matrices with exact radicals |
| `ldlt` | `ldlt(A)` | Exact square-root-free $LDL^T$ decomposition ($L$ unit lower, $D$ positive diagonal rational matrix) |
| `pinv` | `pinv(A)` | Exact Moore-Penrose pseudoinverse $A^+$ via RREF full-rank factorization |
| `solve_linear` / `linsolve` | `solve_linear(A, b)` | Exact linear system solver $Ax=b$ (unique, inconsistent, or parametric solutions) |
| `dot`, `cross` | `dot(u, v)`, `cross(u, v)` | Vector dot product and 3D vector cross product |
| `norm` | `norm(v)` | Euclidean vector norm ($\sqrt{\sum v_i^2}$, with radical simplification) |
| `grad` | `grad(f, [x, y, z])` | Gradient vector field ($\nabla f$) |
| `div`, `curl` | `div(F, [x, y, z])`, `curl(F, ...)` | Divergence ($\nabla \cdot F$) and 3D curl ($\nabla \times F$) |
| Matrix / Vector Syntax | `[[1, 2], [3, 4]]`, `[1, 2, 3]` | Matrix and vector literal syntax |

---

### 5. Computational Geometry

| Function | Syntax & Example | Description |
| :--- | :--- | :--- |
| `line_intersect` | `line_intersect([A1, B1, C1], [A2, B2, C2])` | Exact intersection point of two lines $Ax+By+C=0$ via Cramer's rule |
| `circle_intersect` | `circle_intersect(c1, r1, c2, r2)` | Exact intersection points of two circles via radical axis order reduction |
| `triangle_area` | `triangle_area(p1, p2, p3)` | Exact area of a triangle given three vertices via Shoelace formula |
| `triangle_centers` | `triangle_centers(p1, p2, p3)` | Triangle centers (centroid, circumcenter, orthocenter, incenter) |

---

### 6. Exact Discrete Probability & Statistics

| Function | Syntax & Example | Description |
| :--- | :--- | :--- |
| `binom` | `binom(n, k, p)` | Binomial distribution PMF $P(X=k) = \binom{n}{k} p^k (1-p)^{n-k}$ |
| `hyper` | `hyper(N, K, n, k)` | Hypergeometric distribution PMF (without replacement) |
| `geom` | `geom(p, k)` | Geometric distribution PMF $P(X=k) = (1-p)^{k-1} p$ |
| `bayes` | `bayes(prior, likelihood, marginal)` | Exact posterior probability $P(A\|B) = \frac{P(B\|A)P(A)}{P(B)}$ via Bayes' theorem |
| `expect` | `expect(binom, n, p)` / `expect([[x1, p1], ...])` | Expected value $E[X]$ of discrete distributions |
| `variance` | `variance(...)` | Variance $V[X] = E[X^2] - (E[X])^2$ |
| `stddev` | `stddev(...)` | Standard deviation $\sigma = \sqrt{V[X]}$ (in exact radical form) |

---

### 7. Symbolic Assumptions System

| Command | Example | Description |
| :--- | :--- | :--- |
| `assume` | `assume(x > 0)`, `assume(n, integer)` | Sets domain constraints (`sqrt(x^2)` → `x`, `sin(n*pi)` → `0`, etc.) |
| `unassume` | `unassume(x)` | Clears assumptions for the specified variable |
| `assumptions` | `assumptions()` | Displays all active domain constraints |
| `clear_assumptions` | `clear_assumptions()` | Clears all active domain constraints simultaneously |

---

### 8. Real Algebraic Geometry, Mathematical Logic & Quantifier Elimination

| Function | Syntax & Example | Description |
| :--- | :--- | :--- |
| `qe` | `qe(forall([x], x^2 + a*x + b > 0))` | Full Quantifier Elimination via Cylindrical Algebraic Decomposition (CAD) (constructs equivalent quantifier-free conditions via Hong (1992) boundary polynomials) |
| `forall` | `forall([x], formula)` | Universal quantifier formula constructor $\forall x \, \Phi(x)$ |
| `exists` | `exists([x], formula)` | Existential quantifier formula constructor $\exists x \, \Phi(x)$ |
| `cad` | `cad([x^2 - 2], [x])` | Direct execution of Cylindrical Algebraic Decomposition (1D cell decomposition and sample point derivation for semi-algebraic sets) |

---

### 9. Elliptic Curves & Arithmetic Geometry

| Function | Syntax & Example | Description |
| :--- | :--- | :--- |
| `ec_add` | `ec_add([A, B], P1, P2)` | Rational point addition on Weierstrass elliptic curve $y^2 = x^3 + Ax + B$ via Chord and Tangent method (handles point at infinity `O`) |
| `ec_mul` | `ec_mul([A, B], n, P)` | Scalar multiplication $n P$ of rational points on elliptic curves via binary Double-and-Add algorithm |
| `ec_torsion` | `ec_torsion(A, B)` | Full determination of the rational torsion subgroup $E(\mathbb{Q})_{\text{tors}}$ based on the Nagell-Lutz and Mazur theorems |

---

### 10. Visualization & Verification Tools

| Function | Syntax & Example | Description |
| :--- | :--- | :--- |
| `verify` | `verify(integrate(1/(x^2+1), x), atan(x))` | Independently verifies algebraic equivalence of two expressions and issues an algebraic verification certificate |
| `plot` | `plot(sin(x), [-pi, pi])` | High-resolution terminal curve plotter using Unicode 2×4 Braille characters (automatic root/extrema detection, singularity discontinuity handling) |

---

### 11. Lattice Basis Reduction & Integer Relations (LLL)

| Function | Syntax & Example | Description |
| :--- | :--- | :--- |
| `lll` | `lll([[1, -1, 3], [1, 0, 5], [1, 2, 6]])` | Lenstra–Lenstra–Lovász (1982) lattice basis reduction algorithm over exact rationals |
| `find_min_poly` | `find_min_poly(sqrt(2) + sqrt(3), 4)` | Exact minimal polynomial reconstruction for algebraic numbers using LLL lattice reduction |

---

### 12. p-adic Arithmetic, Local Algebra & Hensel Lifting

| Function | Syntax & Example | Description |
| :--- | :--- | :--- |
| `padic_val` | `padic_val(50, 5)` | Exact p-adic valuation $v_p(x) \in \mathbb{Z}$ for rational $x$ and prime base $p$ ($x=0$ yields infinity) |
| `padic_norm` | `padic_norm(3/20, 2)` | Exact ultrametric norm $|x|_p = p^{-v_p(x)}$ as an exact rational (`big.Rat`) |
| `padic_expand` | `padic_expand(2/3, 5, 4)` | Truncated p-adic power series expansion string $a_0 + a_1 p + a_2 p^2 + \dots + O(p^k)$ |

---

### 13. Local Algebraic Series & Singularities (Puiseux Series)

| Function | Syntax & Example | Description |
| :--- | :--- | :--- |
| `puiseux` | `puiseux(y^2 - x^3, y, x, 3)` | Deterministically solves for local fractional power Puiseux series branches $y(x) = \sum c_k x^{p_k/q_k}$ around singularities/branch points of $F(x, y) = 0$ via the Newton polygon algorithm (default order 3) |




