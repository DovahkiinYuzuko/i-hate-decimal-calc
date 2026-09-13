# ==============================================================================
# i-hate-decimal-calc (ihd) - Live Mathematical Exam & Showcase Runner (PowerShell)
# ==============================================================================

$ErrorActionPreference = "Stop"
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$OutputEncoding = [System.Text.Encoding]::UTF8
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = Split-Path -Parent $ScriptDir
$IhdBin = Join-Path $ProjectRoot "ihd.exe"

Write-Host ""
Write-Host "================================================================================" -ForegroundColor Cyan
Write-Host "   i-hate-decimal-calc (ihd) - Live Mathematical Exam & Showcase" -ForegroundColor Cyan
Write-Host "================================================================================" -ForegroundColor Cyan
Write-Host "  Demonstrating exact symbolic calculations across 24 benchmark problems:"
Write-Host "  - Part 1: High School Math (Algebra, Trigonometry, Calculus & Vectors)"
Write-Host "  - Part 2: Advanced Math (Pell Equation, Harmonics, Vandermonde & Olympiad)"
Write-Host "--------------------------------------------------------------------------------"

# Build ihd if binary not present
if (-not (Test-Path $IhdBin)) {
    Write-Host "[BUILD] Building ihd.exe from source..." -ForegroundColor Yellow
    Push-Location $ProjectRoot
    try {
        go build -o ihd.exe ./cmd/ihd
    } finally {
        Pop-Location
    }
    if (-not (Test-Path $IhdBin)) {
        Write-Error "Failed to build ihd.exe"
        exit 1
    }
    Write-Host "[BUILD] Built successfully: $IhdBin" -ForegroundColor Green
    Write-Host "--------------------------------------------------------------------------------"
}

$problems = @(
    # --- Part 1: High School Math ---
    @{
        Num = 1; Category = "High School: Algebra"; Title = "3rd-Degree Symmetric Identity"
        Description = "Find x^3 + y^3 when x + y = 3 and xy = 1"
        Input = "(3)^3 - 3*(1)*(3)"
        Args = @()
        Expected = "18"
    },
    @{
        Num = 2; Category = "High School: Algebra"; Title = "Radical Denesting & Rationalization"
        Description = "Simplify 1 / sqrt(5 - 2*sqrt(6))"
        Input = "1 / sqrt(5 - 2*sqrt(6))"
        Args = @("--explain")
        Expected = "√2 + √3"
    },
    @{
        Num = 3; Category = "High School: Number Theory"; Title = "GCD & LCM Fundamental Relation"
        Description = "Verify gcd(a, b) * lcm(a, b) = a * b for 123456 and 789012"
        Input = "gcd(123456, 789012) * lcm(123456, 789012) - 123456 * 789012"
        Args = @()
        Expected = "0"
    },
    @{
        Num = 4; Category = "High School: Probability"; Title = "Repeated Trials Probability"
        Description = "Probability of exactly three 1s in 10 rolls of a fair die"
        Input = "comb(10, 3) * (1/6)^3 * (5/6)^7"
        Args = @()
        Expected = "390625/2519424"
    },
    @{
        Num = 5; Category = "High School: Complex Numbers"; Title = "Polar Form Representation"
        Description = "Convert z = 1 + i into exact polar form"
        Input = "polar(1 + i)"
        Args = @()
        Expected = "√2*(cos(π/4) + i*sin(π/4))"
    },
    @{
        Num = 6; Category = "High School: Trigonometry"; Title = "Exact Angle Addition Identity"
        Description = "Verify sin(pi/3)*cos(pi/6) + cos(pi/3)*sin(pi/6) = 1"
        Input = "sin(pi/3) * cos(pi/6) + cos(pi/3) * sin(pi/6)"
        Args = @()
        Expected = "1"
    },
    @{
        Num = 7; Category = "High School: Logarithms"; Title = "Multi-Base Logarithmic Identity"
        Description = "Evaluate log(2, 8) + log(3, 27) - log(10, 1000)"
        Input = "log(2, 8) + log(3, 27) - log(10, 1000)"
        Args = @()
        Expected = "3"
    },
    @{
        Num = 8; Category = "High School: Sequences"; Title = "Faulhaber Polynomial (Sum of Cubes)"
        Description = "Derive closed-form formula for sum(k^3, k=1..n)"
        Input = "sum(k^3, k, 1, n)"
        Args = @()
        Expected = "n^2/4 + n^3/2 + n^4/4"
    },
    @{
        Num = 9; Category = "High School: Calculus"; Title = "Symbolic Product Rule Derivative"
        Description = "Differentiate f(x) = x * sin(x)"
        Input = "diff(x * sin(x), x)"
        Args = @()
        Expected = "cos(x)*x + sin(x)"
    },
    @{
        Num = 10; Category = "High School: Calculus"; Title = "Maclaurin Series Expansion"
        Description = "5th-order Maclaurin series of sin(x) around 0"
        Input = "taylor(sin(x), x, 0, 5)"
        Args = @()
        Expected = "x - x^3/6 + x^5/120"
    },
    @{
        Num = 11; Category = "High School: Vectors"; Title = "Cross Product & Orthogonality"
        Description = "Compute [1,2,3] x [4,5,6] and verify orthogonality"
        Input = "dot([1, 2, 3], cross([1, 2, 3], [4, 5, 6]))"
        Args = @()
        Expected = "0"
    },
    @{
        Num = 12; Category = "High School: Geometry"; Title = "Line & Circle Intersections"
        Description = "Solve x-coordinates for y = x + 1 and x^2 + y^2 = 1"
        Input = "solve(2*x^2 + 2*x, x)"
        Args = @()
        Expected = "[-1, 0]"
    },

    # --- Part 2: Advanced, University & Competition Math ---
    @{
        Num = 13; Category = "Advanced: Number Theory"; Title = "Pell's Equation Minimal Solution"
        Description = "Verify minimal solution (1766319049, 226153980) for x^2 - 61*y^2 = 1"
        Input = "1766319049^2 - 61 * 226153980^2"
        Args = @()
        Expected = "1"
    },
    @{
        Num = 14; Category = "Advanced: Large Integers"; Title = "Mersenne Number M67 Factorization"
        Description = "Verify Cole (1903): 2^67 - 1 = 193707721 * 761838257287"
        Input = "2^67 - 1 - 193707721 * 761838257287"
        Args = @()
        Expected = "0"
    },
    @{
        Num = 15; Category = "Advanced: Number Theory"; Title = "Chinese Remainder Theorem System"
        Description = "Verify x = 23 satisfies x = 2 mod 3, 3 mod 5, 2 mod 7"
        Input = "[mod(23, 3), mod(23, 5), mod(23, 7)]"
        Args = @()
        Expected = "[2, 3, 2]"
    },
    @{
        Num = 16; Category = "Advanced: Complex Numbers"; Title = "Exact Principal Argument"
        Description = "Find principal argument arg(1 + i*sqrt(3))"
        Input = "arg(1 + i*sqrt(3))"
        Args = @()
        Expected = "π/3"
    },
    @{
        Num = 17; Category = "Advanced: Radicals"; Title = "Radical Denesting"
        Description = "Denest sqrt(7 + 2*sqrt(10))"
        Input = "sqrt(7 + 2*sqrt(10))"
        Args = @()
        Expected = "√2 + √5"
    },
    @{
        Num = 18; Category = "Advanced: Algebra"; Title = "Cubic Identity Polynomial Expansion"
        Description = "Verify (x + 1)^3 = x^3 + 3*x^2 + 3*x + 1"
        Input = "expand((x + 1)^3 - (x^3 + 3*x^2 + 3*x + 1))"
        Args = @()
        Expected = "0"
    },
    @{
        Num = 19; Category = "Advanced: Number Theory"; Title = "Continued Fraction Periodicity"
        Description = "Compute regular continued fraction expansion of sqrt(2)"
        Input = "cfrac(sqrt(2))"
        Args = @()
        Expected = "[1, [2]]"
    },
    @{
        Num = 20; Category = "University: Linear Algebra"; Title = "3x3 Vandermonde Determinant"
        Description = "Evaluate det([[1,1,1],[1,2,3],[1,4,9]])"
        Input = "det([[1, 1, 1], [1, 2, 3], [1^2, 2^2, 3^2]])"
        Args = @()
        Expected = "2"
    },
    @{
        Num = 21; Category = "University: Linear Algebra"; Title = "3x3 Circulant Matrix Inverse"
        Description = "Find exact rational inverse of circulant matrix C"
        Input = "inv([[1, 2, 3], [3, 1, 2], [2, 3, 1]])"
        Args = @()
        Expected = "[[-5/18, 7/18, 1/18], [1/18, -5/18, 7/18], [7/18, 1/18, -5/18]]"
    },
    @{
        Num = 22; Category = "University: Vector Analysis"; Title = "2D Harmonic Function Laplacian"
        Description = "Verify div(grad(x^3 - 3*x*y^2)) = 0"
        Input = "diff(diff(x^3 - 3*x*y^2, x), x) + diff(diff(x^3 - 3*x*y^2, y), y)"
        Args = @()
        Expected = "0"
    },
    @{
        Num = 23; Category = "Olympiad: Geometry"; Title = "Euler Line Triangle Centers"
        Description = "Compute centroid, circumcenter, orthocenter, incenter for A(0,0), B(4,0), C(0,3)"
        Input = "triangle_centers([0, 0], [4, 0], [0, 3])"
        Args = @()
        Expected = "[[4/3, 1], [2, 3/2], [0, 0], [1, 1]]"
    },
    @{
        Num = 24; Category = "Competition: Discrete Math"; Title = "5th Catalan Number"
        Description = "Evaluate C5 = 1/6 * comb(10, 5)"
        Input = "1/6 * comb(10, 5)"
        Args = @()
        Expected = "42"
    }
)

$passed = 0
$failed = 0

foreach ($p in $problems) {
    Write-Host ""
    Write-Host ("[{0:D2}/24] [{1}] {2}" -f $p.Num, $p.Category, $p.Title) -ForegroundColor Yellow
    Write-Host ("  Problem: {0}" -f $p.Description) -ForegroundColor Gray

    $cmdArgs = @()
    if ($p.Args) {
        $cmdArgs += $p.Args
    }
    $cmdArgs += $p.Input

    Write-Host ("  > ihd {0} `"{1}`"" -f ($p.Args -join " "), $p.Input) -ForegroundColor Cyan

    $output = & $IhdBin @cmdArgs 2>&1 | Out-String
    $cleanOutput = $output.Trim()

    # Display output
    Write-Host ("  < ihd output:") -ForegroundColor White
    $cleanOutput -split "`r?`n" | ForEach-Object {
        Write-Host ("     " + $_) -ForegroundColor White
    }

    if ($cleanOutput -match [regex]::Escape($p.Expected)) {
        Write-Host ("  [PASS] Matches expected exact solution: {0}" -f $p.Expected) -ForegroundColor Green
        $passed++
    } else {
        Write-Host ("  [FAIL] Expected: {0}" -f $p.Expected) -ForegroundColor Red
        $failed++
    }
}

Write-Host ""
Write-Host "================================================================================" -ForegroundColor Cyan
if ($failed -eq 0) {
    Write-Host ("  EXAM SHOWCASE RESULT: ALL {0} PROBLEMS SOLVED SUCCESSFULLY! (100% PASS)" -f $passed) -ForegroundColor Green
} else {
    Write-Host ("  EXAM SHOWCASE RESULT: {0} PASSED, {1} FAILED" -f $passed, $failed) -ForegroundColor Red
}
Write-Host "================================================================================" -ForegroundColor Cyan
Write-Host ""

if ($failed -gt 0) {
    exit 1
}
