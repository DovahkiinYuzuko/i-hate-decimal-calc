#!/usr/bin/env bash
# ==============================================================================
# i-hate-decimal-calc (ihd) - Live Mathematical Exam & Showcase Runner (Bash)
# ==============================================================================

set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
IHD_BIN="$PROJECT_ROOT/ihd"

echo ""
echo -e "\033[36m================================================================================\033[0m"
echo -e "\033[36m   i-hate-decimal-calc (ihd) - Live Mathematical Exam & Showcase\033[0m"
echo -e "\033[36m================================================================================\033[0m"
echo "  Demonstrating exact symbolic calculations across 24 benchmark problems:"
echo "  - Part 1: High School Math (Algebra, Trigonometry, Calculus & Vectors)"
echo "  - Part 2: Advanced Math (Pell Equation, Harmonics, Vandermonde & Olympiad)"
echo "--------------------------------------------------------------------------------"

# Build ihd if binary not present
if [ ! -f "$IHD_BIN" ]; then
    echo -e "\033[33m[BUILD] Building ihd binary from source...\033[0m"
    (cd "$PROJECT_ROOT" && go build -o ihd ./cmd/ihd)
    if [ ! -f "$IHD_BIN" ]; then
        echo "Failed to build ihd binary" >&2
        exit 1
    fi
    echo -e "\033[32m[BUILD] Built successfully: $IHD_BIN\033[0m"
    echo "--------------------------------------------------------------------------------"
fi

declare -a PROBLEMS=(
    "01|High School: Algebra|3rd-Degree Symmetric Identity|Find x^3 + y^3 when x + y = 3 and xy = 1|(3)^3 - 3*(1)*(3)||18"
    "02|High School: Algebra|Radical Denesting & Rationalization|Simplify 1 / sqrt(5 - 2*sqrt(6))|1 / sqrt(5 - 2*sqrt(6))|--explain|√2 + √3"
    "03|High School: Number Theory|GCD & LCM Fundamental Relation|Verify gcd(a, b) * lcm(a, b) = a * b|gcd(123456, 789012) * lcm(123456, 789012) - 123456 * 789012||0"
    "04|High School: Probability|Repeated Trials Probability|Probability of exactly three 1s in 10 rolls|comb(10, 3) * (1/6)^3 * (5/6)^7||390625/2519424"
    "05|High School: Complex Numbers|Polar Form Representation|Convert z = 1 + i into exact polar form|polar(1 + i)||√2*(cos(π/4) + i*sin(π/4))"
    "06|High School: Trigonometry|Exact Angle Addition Identity|Verify sin(pi/3)*cos(pi/6) + cos(pi/3)*sin(pi/6) = 1|sin(pi/3) * cos(pi/6) + cos(pi/3) * sin(pi/6)||1"
    "07|High School: Logarithms|Multi-Base Logarithmic Identity|Evaluate log(2, 8) + log(3, 27) - log(10, 1000)|log(2, 8) + log(3, 27) - log(10, 1000)||3"
    "08|High School: Sequences|Faulhaber Polynomial (Sum of Cubes)|Derive closed-form for sum(k^3)|sum(k^3, k, 1, n)||n^2/4 + n^3/2 + n^4/4"
    "09|High School: Calculus|Symbolic Product Rule Derivative|Differentiate f(x) = x * sin(x)|diff(x * sin(x), x)||cos(x)*x + sin(x)"
    "10|High School: Calculus|Maclaurin Series Expansion|5th-order Maclaurin series of sin(x)|taylor(sin(x), x, 0, 5)||x - x^3/6 + x^5/120"
    "11|High School: Vectors|Cross Product & Orthogonality|Compute [1,2,3] x [4,5,6] and orthogonality|dot([1, 2, 3], cross([1, 2, 3], [4, 5, 6]))||0"
    "12|High School: Geometry|Line & Circle Intersections|Solve x-coords for y = x + 1 and x^2 + y^2 = 1|solve(2*x^2 + 2*x, x)||[-1, 0]"
    "13|Advanced: Number Theory|Pell's Equation Minimal Solution|Verify minimal solution for x^2 - 61*y^2 = 1|1766319049^2 - 61 * 226153980^2||1"
    "14|Advanced: Large Integers|Mersenne Number M67 Factorization|Verify Cole (1903): 2^67 - 1 = factors|2^67 - 1 - 193707721 * 761838257287||0"
    "15|Advanced: Number Theory|Chinese Remainder Theorem System|Verify x = 23 satisfies mod 3, 5, 7|[mod(23, 3), mod(23, 5), mod(23, 7)]||[2, 3, 2]"
    "16|Advanced: Complex Numbers|Exact Principal Argument|Find arg(1 + i*sqrt(3))|arg(1 + i*sqrt(3))||π/3"
    "17|Advanced: Radicals|Radical Denesting|Denest sqrt(7 + 2*sqrt(10))|sqrt(7 + 2*sqrt(10))||√2 + √5"
    "18|Advanced: Algebra|Cubic Identity Polynomial Expansion|Verify (x+1)^3 identity|expand((x + 1)^3 - (x^3 + 3*x^2 + 3*x + 1))||0"
    "19|Advanced: Number Theory|Continued Fraction Periodicity|Periodic cfrac for sqrt(2)|cfrac(sqrt(2))||[1, [2]]"
    "20|University: Linear Algebra|3x3 Vandermonde Determinant|Evaluate det of Vandermonde matrix|det([[1, 1, 1], [1, 2, 3], [1^2, 2^2, 3^2]])||2"
    "21|University: Linear Algebra|3x3 Circulant Matrix Inverse|Find inverse of circulant matrix C|inv([[1, 2, 3], [3, 1, 2], [2, 3, 1]])||[[-5/18, 7/18, 1/18], [1/18, -5/18, 7/18], [7/18, 1/18, -5/18]]"
    "22|University: Vector Analysis|2D Harmonic Function Laplacian|Verify div(grad(x^3 - 3*x*y^2)) = 0|diff(diff(x^3 - 3*x*y^2, x), x) + diff(diff(x^3 - 3*x*y^2, y), y)||0"
    "23|Olympiad: Geometry|Euler Line Triangle Centers|Compute centers for triangle A(0,0), B(4,0), C(0,3)|triangle_centers([0, 0], [4, 0], [0, 3])||[[4/3, 1], [2, 3/2], [0, 0], [1, 1]]"
    "24|Competition: Discrete Math|5th Catalan Number|Evaluate C5 = 1/6 * comb(10, 5)|1/6 * comb(10, 5)||42"
)

PASSED=0
FAILED=0

for item in "${PROBLEMS[@]}"; do
    IFS='|' read -r NUM CAT TITLE DESC EXPR EXTRA_ARG EXPECTED <<< "$item"
    echo ""
    echo -e "\033[33m[$NUM/24] [$CAT] $TITLE\033[0m"
    echo "  Problem: $DESC"
    
    CMD_ARGS=()
    if [ -n "$EXTRA_ARG" ]; then
        CMD_ARGS+=("$EXTRA_ARG")
    fi
    CMD_ARGS+=("$EXPR")

    echo -e "  \033[36m> ihd ${EXTRA_ARG:+$EXTRA_ARG }\"$EXPR\"\033[0m"
    OUTPUT=$("$IHD_BIN" "${CMD_ARGS[@]}" 2>&1)
    
    echo "  < ihd output:"
    echo "$OUTPUT" | sed 's/^/     /'

    if echo "$OUTPUT" | grep -F -q "$EXPECTED"; then
        echo -e "  \033[32m[PASS] Matches expected exact solution: $EXPECTED\033[0m"
        PASSED=$((PASSED + 1))
    else
        echo -e "  \033[31m[FAIL] Expected: $EXPECTED\033[0m"
        FAILED=$((FAILED + 1))
    fi
done

echo ""
echo -e "\033[36m================================================================================\033[0m"
if [ "$FAILED" -eq 0 ]; then
    echo -e "\033[32m  EXAM SHOWCASE RESULT: ALL $PASSED PROBLEMS SOLVED SUCCESSFULLY! (100% PASS)\033[0m"
else
    echo -e "\033[31m  EXAM SHOWCASE RESULT: $PASSED PASSED, $FAILED FAILED\033[0m"
fi
echo -e "\033[36m================================================================================\033[0m"
echo ""

if [ "$FAILED" -gt 0 ]; then
    exit 1
fi
