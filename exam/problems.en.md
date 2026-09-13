# ihd Mathematical Practical Cookbook & Problems - English Edition

`ihd` (I Hate Decimal Calc) is an arbitrary-precision symbolic calculator that refuses premature decimal approximations, keeping expressions in exact fractions, radicals, complex numbers, polynomials, and symbolic algebraic forms.

This document showcases how to solve high school, entrance examination, undergraduate STEM (linear algebra, multivariable calculus, number theory), and mathematical competition problems using `ihd`.

---

## Part 1: High School Mathematics (Algebra, Calculus & Geometry)

### 01. [Algebra] 3rd-Degree Symmetric Polynomial Substitution
- **Problem**: Given $x + y = 3$ and $xy = 1$, find the exact value of $x^3 + y^3$.
- **Method**: Using symmetric identity $x^3 + y^3 = (x+y)^3 - 3xy(x+y)$.
- **ihd Command**:
  ```bash
  ihd "(3)^3 - 3*(1)*(3)"
  ```
- **Output**: `18`

---

### 02. [Algebra] Radical Denesting & Conjugate Rationalization
- **Problem**: Simplify $\frac{1}{\sqrt{5 - 2\sqrt{6}}}$.
- **Method**: The denominator denests to $\sqrt{(\sqrt{3}-\sqrt{2})^2} = \sqrt{3}-\sqrt{2}$. Rationalizing yields $\sqrt{2}+\sqrt{3}$.
- **ihd Command**:
  ```bash
  ihd "1 / sqrt(5 - 2*sqrt(6))"
  ```
- **Output**: `√2 + √3`
- **Step-by-Step Explanation (`--explain`)**:
  ```bash
  ihd --explain "1 / sqrt(5 - 2*sqrt(6))"
  ```
  ```text
  ├── [Step 1: Denest Radical]
  │   Borodin algorithm radical denesting
  │   √(5 - 2*√6)  ──>  -√2 + √3
  │
  ├── [Step 2: Rationalize Denominator]
  │   Multiply conjugate radical to rationalize
  │   (-√2 + √3)^-1  ──>  √2 + √3
  └── [Result]
      = √2 + √3
  ```

---

### 03. [Number Theory] GCD and LCM Fundamental Relation
- **Problem**: Verify $\gcd(a, b) \times \operatorname{lcm}(a, b) = a \times b$ for $a = 123456, b = 789012$.
- **ihd Command**:
  ```bash
  ihd "gcd(123456, 789012) * lcm(123456, 789012) - 123456 * 789012"
  ```
- **Output**: `0`

---

### 04. [Probability] Repeated Trials Probability
- **Problem**: Find the probability of getting exactly three 1s when rolling a fair die 10 times.
- **Formula**: $\binom{10}{3} \left(\frac{1}{6}\right)^3 \left(\frac{5}{6}\right)^7$
- **ihd Command**:
  ```bash
  ihd "comb(10, 3) * (1/6)^3 * (5/6)^7"
  ```
- **Output**: `390625/2519424`

---

### 05. [Complex Analysis] Complex Polar Form Representation
- **Problem**: Convert $z = 1 + i$ into polar form $r(\cos\theta + i\sin\theta)$.
- **ihd Command**:
  ```bash
  ihd "polar(1 + i)"
  ```
- **Output**: `√2*(cos(π/4) + i*sin(π/4))`

---

### 06. [Trigonometry] Angle Addition Formula Identity
- **Problem**: Verify $\sin\left(\frac{\pi}{3}\right) \cos\left(\frac{\pi}{6}\right) + \cos\left(\frac{\pi}{3}\right) \sin\left(\frac{\pi}{6}\right) = 1$.
- **ihd Command**:
  ```bash
  ihd "sin(pi/3) * cos(pi/6) + cos(pi/3) * sin(pi/6)"
  ```
- **Output**: `1`

---

### 07. [Logarithms] Multi-Base Logarithmic Identity
- **Problem**: Evaluate $\log_2(8) + \log_3(27) - \log_{10}(1000)$.
- **ihd Command**:
  ```bash
  ihd "log(2, 8) + log(3, 27) - log(10, 1000)"
  ```
- **Output**: `3`

---

### 08. [Series] Sum of Consecutive Cubes (Faulhaber's Formula)
- **Problem**: Find the closed-form polynomial for $\sum_{k=1}^n k^3$.
- **ihd Command**:
  ```bash
  ihd "sum(k^3, k, 1, n)"
  ```
- **Output**: `n^2/4 + n^3/2 + n^4/4`

---

### 09. [Calculus] Symbolic Product Rule Differentiation
- **Problem**: Compute the derivative of $f(x) = x \sin(x)$.
- **ihd Command**:
  ```bash
  ihd "diff(x * sin(x), x)"
  ```
- **Output**: `cos(x)*x + sin(x)`

---

### 10. [Calculus] Maclaurin Series Expansion
- **Problem**: Compute the 5th-order Maclaurin series of $\sin(x)$ around $x = 0$.
- **ihd Command**:
  ```bash
  ihd "taylor(sin(x), x, 0, 5)"
  ```
- **Output**: `x - x^3/6 + x^5/120`

---

### 11. [Vector Algebra] 3D Vector Cross Product & Orthogonality
- **Problem**: Compute $(1, 2, 3) \times (4, 5, 6)$ and verify it is perpendicular to $(1, 2, 3)$.
- **ihd Command**:
  ```bash
  ihd "dot([1, 2, 3], cross([1, 2, 3], [4, 5, 6]))"
  ```
- **Output**: `0`

---

### 12. [Geometry] Line and Circle Intersection
- **Problem**: Find the $x$-coordinates of the intersections between $y = x + 1$ and $x^2 + y^2 = 1$.
- **ihd Command**:
  ```bash
  ihd "solve(2*x^2 + 2*x, x)"
  ```
- **Output**: `[-1, 0]`

---

## Part 2: Advanced, University & Competition Problems

### 13. [Number Theory: Wester CAS Benchmark] Pell's Equation Large Solutions
- **Problem**: Verify minimal solution $(x, y) = (1766319049, 226153980)$ to $x^2 - 61y^2 = 1$.
- **ihd Command**:
  ```bash
  ihd "1766319049^2 - 61 * 226153980^2"
  ```
- **Output**: `1`

---

### 14. [Large Integers: Cole 1903] Mersenne Number $M_{67}$ Factorization
- **Problem**: Verify $M_{67} = 2^{67} - 1 = 193707721 \times 761838257287$.
- **ihd Command**:
  ```bash
  ihd "2^67 - 1 - 193707721 * 761838257287"
  ```
- **Output**: `0`

---

### 15. [Number Theory] Chinese Remainder Theorem (CRT) System
- **Problem**: Verify $x = 23$ satisfies $x \equiv 2 \pmod 3, x \equiv 3 \pmod 5, x \equiv 2 \pmod 7$.
- **ihd Command**:
  ```bash
  ihd "[mod(23, 3), mod(23, 5), mod(23, 7)]"
  ```
- **Output**: `[2, 3, 2]`

---

### 16. [Complex Analysis] Exact Principal Argument
- **Problem**: Find principal argument $\operatorname{arg}(1 + \sqrt{3}i)$.
- **ihd Command**:
  ```bash
  ihd "arg(1 + i*sqrt(3))"
  ```
- **Output**: `π/3`

---

### 17. [Algebra] Radical Denesting
- **Problem**: Denest $\sqrt{7 + 2\sqrt{10}}$.
- **ihd Command**:
  ```bash
  ihd "sqrt(7 + 2*sqrt(10))"
  ```
- **Output**: `√2 + √5`

---

### 18. [Algebra] Cubic Identity Polynomial Expansion
- **Problem**: Verify $(x + 1)^3 = x^3 + 3x^2 + 3x + 1$.
- **ihd Command**:
  ```bash
  ihd "expand((x + 1)^3 - (x^3 + 3*x^2 + 3*x + 1))"
  ```
- **Output**: `0`

---

### 19. [Number Theory] Continued Fraction Periodicity
- **Problem**: Compute regular continued fraction expansion of $\sqrt{2}$.
- **ihd Command**:
  ```bash
  ihd "cfrac(sqrt(2))"
  ```
- **Output**: `[1, [2]]`

---

### 20. [MIT 18.06] $3 \times 3$ Vandermonde Determinant Evaluation
- **Problem**: Compute $\det \begin{pmatrix} 1 & 1 & 1 \\ 1 & 2 & 3 \\ 1 & 4 & 9 \end{pmatrix}$.
- **ihd Command**:
  ```bash
  ihd "det([[1, 1, 1], [1, 2, 3], [1^2, 2^2, 3^2]])"
  ```
- **Output**: `2`

---

### 21. [MIT 18.06] $3 \times 3$ Circulant Matrix Inverse
- **Problem**: Find exact rational inverse of $C = \begin{pmatrix} 1 & 2 & 3 \\ 3 & 1 & 2 \\ 2 & 3 & 1 \end{pmatrix}$.
- **ihd Command**:
  ```bash
  ihd "inv([[1, 2, 3], [3, 1, 2], [2, 3, 1]])"
  ```
- **Output**: `[[-5/18, 7/18, 1/18], [1/18, -5/18, 7/18], [7/18, 1/18, -5/18]]`

---

### 22. [MIT 18.02] 2D Harmonic Function Laplacian
- **Problem**: Verify $\nabla^2 (x^3 - 3xy^2) = 0$.
- **ihd Command**:
  ```bash
  ihd "diff(diff(x^3 - 3*x*y^2, x), x) + diff(diff(x^3 - 3*x*y^2, y), y)"
  ```
- **Output**: `0`

---

### 23. [Olympiad Geometry] Triangle Centers (Euler Line Components)
- **Problem**: Simultaneously compute Centroid, Circumcenter, Orthocenter, and Incenter for triangle $A(0,0), B(4,0), C(0,3)$.
- **ihd Command**:
  ```bash
  ihd "triangle_centers([0, 0], [4, 0], [0, 3])"
  ```
- **Output**: `[[4/3, 1], [2, 3/2], [0, 0], [1, 1]]`

---

### 24. [Discrete Math] 5th Catalan Number
- **Problem**: Compute $C_5 = \frac{1}{6}\binom{10}{5}$.
- **ihd Command**:
  ```bash
  ihd "1/6 * comb(10, 5)"
  ```
- **Output**: `42`
