# ihd Visual Samples & Showcases

This document provides verified CLI output samples for `ihd` (i-hate-decimal-calc), covering 2D/3D plots, derivation projections, proof certificates, closed-domain arithmetic, and specialized mathematical commands. Every example reflects actual output from the `ihd` binary; in color-capable terminals, plots and derivation trees also render with ANSI colors (disable with `--no-color`).

[Japanese Edition (sample.ja.md)](./sample.ja.md) | [Back to README](../README.en.md)

## Table of Contents

1. [2D Braille Terminal Plotter & SVG / Typst Vector Export (`plot` / `plot_svg`)](#1-2d-braille-terminal-plotter--svg--typst-vector-export-plot--plot_svg)
2. [3D Spatial Plotter & Exact CAS Critical-Point Analysis (`plot3d` / `plot3d_svg`)](#2-3d-spatial-plotter--exact-cas-critical-point-analysis-plot3d--plot3d_svg)
3. [Terminal 2D Pretty Printing & Typst / LaTeX Math Export](#3-terminal-2d-pretty-printing--typst--latex-math-export)
4. [Algebraic Derivation Trees & Formal Projections (`--explain` / `--typst` / `--lean`)](#4-algebraic-derivation-trees--formal-projections---explain----typst----lean)
5. [REPL Real-Time Signature Guide & Syntax Highlighting](#5-repl-real-time-signature-guide--syntax-highlighting)
6. [Physical Units, Collections & Semantic Diff](#6-physical-units-collections--semantic-diff)
7. [First-Class Closed-Domain AST Arithmetic (`padic_expand` / `puiseux` / `interval`)](#7-first-class-closed-domain-ast-arithmetic-padic_expand--puiseux--interval)
8. [Rational SOS, Ideal Membership & Algebraic Real Certificates](#8-rational-sos-ideal-membership--algebraic-real-certificates)
9. [Formal Power Series, Combinatorics, Error-Correcting Codes & Discrete Math](#9-formal-power-series-combinatorics-error-correcting-codes--discrete-math)
10. [Advanced & Specialized One-Shot Examples](#10-advanced--specialized-one-shot-examples)

## 1. 2D Braille Terminal Plotter & SVG / Typst Vector Export (`plot` / `plot_svg`)

Renders function curves in the terminal using 2×4 Unicode Braille characters (50×16 characters, 100×64 subpixel resolution). Symbolic feature detection identifies roots, extrema, removable holes, and asymptotes, and the same plot data can be exported as standalone SVG (`800x500`) or Typst (`cetz-plot`) vector code.

### 1.1 Basic Curve Plot with Explicit Y-Range

When the optional third argument `[y_min, y_max]` is omitted, `plot` auto-scales to a padded rational range. Supplying `[y_min, y_max]` pins exact rational bounds on the y-axis.

```bash
ihd --lang en "plot(sin(x), [-pi, pi], [-1, 1])"
```

```text
    1 ┼                          │       ⢀⠤⠒⠉⠉⠑⠒⢄⡀        
      │                          │      ⡔⠁       ⠈⢢       
      │                          │    ⢠⠊           ⠑⡄     
      │                          │   ⢠⠃             ⠘⡄    
      │                          │  ⡰⠁               ⠈⢆   
      │                          │ ⡰⠁                 ⠈⢆  
      │                          │⡠⠃                   ⠘⢄ 
      │                          ⡜                      ⠈⢆
      │ ⠱⡀──────────────────────⡜┼────────────────────────
      │  ⠱⡀                   ⢀⠎ │                        
      │   ⠱⡀                 ⢀⠎  │                        
      │    ⠱⡀               ⢀⠎   │                        
      │     ⠘⢄             ⢠⠃    │                        
      │       ⠣⡀          ⡠⠃     │                        
      │        ⠘⢄       ⢀⠜       │                        
   -1 ┼          ⠑⠢⢄⣀⡠⠤⠊⠁        │                        
      └──────────────────────────────────────────────────── x
       -π                                             π

[CAS Features Detected]
* No singularities or extrema (smooth monotonic curve)
* Domain:            x ∈ [-π, π], y ∈ [-1, 1]
```

### 1.2 Automatic Removable Singularity Detection (`sin(x)/x`)

Symbolic analysis detects removable singularities (holes) at common numerator-denominator zeros and marks them with `○` and exact coordinates.

```bash
ihd --lang en "plot(sin(x)/x, [-2*pi, 2*pi], [-1/2, 6/5])"
```

```text
  6/5 ┼                          │                        
      │                         ⣀○ [0, 1]                 
      │                      ⢀⠔⠊ ⠈⠑⠢⡀                     
      │                     ⡠⠊   │  ⠘⢄                    
      │                    ⡰⠁    │    ⠣⡀                  
      │                   ⡰⠁     │     ⠱⡀                 
      │                  ⡰⠁      │      ⠱⡀                
      │                 ⡰⠁       │       ⠱⡀               
      │                ⡰⠁        │        ⠱⡀              
      │               ⡰⠁         │         ⠱⡀             
      │              ⡰⠁          │          ⢣             
      │ ⠑⢄⡀────────⢀⠎────────────┼───────────⠣⡀────────⢀⡠⠊
      │   ⠈⠢⣀    ⣀⠔⠁             │            ⠈⠢⣀    ⣀⠔⠁  
      │      ⠉⠑⠒⠉                │               ⠉⠒⠊⠉     
      │                          │                        
 -1/2 ┼                          │                        
      └──────────────────────────────────────────────────── x
       -2*π                                         2*π

[CAS Features Detected]
* Removable Hole:    (0, 1)
* Domain:            x ∈ [-2*π, 2*π], y ∈ [-1/2, 6/5]
```

### 1.3 SVG / Typst Vector Output & File Export

Combining `plot` with `--format svg` or `--typst` outputs embeddable vector source code. Use `--out <path>` or the trailing path argument of `plot_svg` to write UTF-8 (no BOM) files directly.

`--format svg` applies only to plot expressions and exits with code `1` on non-plot expressions. To avoid mixing multi-line or interactive output into a single file, combining `--out` with piped stdin, script execution (`run`), or `--step` exits with code `1`; use standard shell redirection (`>`) to save those outputs instead.

```bash
# Save SVG directly to disk (--format svg + --out or plot_svg)
ihd --format svg --out sin.svg "plot(sin(x), [-pi, pi])"
ihd "plot_svg(sin(x), [-pi, pi], \"sin.svg\")"

# Export Typst (@preview/cetz-plot:0.1.4) vector code to a file
ihd --typst --out sin.typ "plot(sin(x), [-pi, pi])"
```

The Typst output generated by `--typst` (with intermediate coordinates abbreviated) is shown below:

```typst
#import "@preview/cetz:0.5.2"
#import "@preview/cetz-plot:0.1.4": plot

#cetz.canvas({
  plot.plot(
    size: (10, 6),
    x-min: -3.14159, x-max: 3.14159,
    y-min: -1.18806, y-max: 1.18806,
    x-label: [$x$],
    y-label: [$y$],
    {
      plot.add(((-3.14159, 0), (-2.60977, -0.507106), ... , (3.14159, 0)), label: [$sin(x)$], style: (stroke: (paint: blue, thickness: 1.5pt)))
    }
  )
})
```

---

## 2. 3D Spatial Plotter & Exact CAS Critical-Point Analysis (`plot3d` / `plot3d_svg`)

Renders 3D graphs in the terminal as Braille wireframes with continuous 2D hidden-line removal (HLR) and viewpoint-adaptive 3D axes. Call `plot3d(expr, x_range, y_range, [opts...])` for surfaces or `plot3d([x(t), y(t), z(t)], t_range, [opts...])` for parametric space curves. For surfaces, gradient and Hessian analysis classifies exact critical points (local min/max `●`, saddle point `◆`, degenerate critical point `◇`).

### 2.1 3D Surface, Hidden-Line Removal & Exact Saddle Point Detection

Renders the hyperbolic paraboloid $`z = x^2 - y^2`$ using the default orthographic camera (`az = 35°`, `el = 25°`) and marks the saddle point at `[0, 0, 0]` with `◆`.

```bash
ihd --lang en "plot3d(x^2 - y^2, [-2, 2], [-2, 2])"
```

```text
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣀⠔⠊⢹⠈⠉⠑⠒⣤⣄⣀⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⡠⠔⠊⠀⠀⠀⢸⠀⠀⡠⡺⢡⠃⡎⢨⠻⡒⠢⠤⢄⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢠⢖⣉⡀⠀⠀⠀⠀⠀⢸⢀⣎⢇⠷⡗⡲⠥⡇⢠⢋⣢⠒⠉⡇⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢸⠀⠀⠈⢉⣶⠦⣤⣄⣸⣳⡟⣏⡼⣀⡇⣼⠬⣏⠁⡇⠱⡀⡇⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢸⠀⡠⢎⠙⢄⡱⡼⣸⣹⡿⣻⣳⣣⡜⢺⠑⢼⠀⢹⡀⡰⠱⡇⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢸⠔⡅⢨⡖⠉⣦⠜⣞⣼⣿⣵⡧⢎⡿⠑⢤⠋⢣⡃⢘⡇⠀⡇⠀z⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢸⠀⡜⡅⡨⢪⣴⣽⠞⢹⠉⠻◆⡪⣏⡶⡁⢣⠊⠱⡸⠘⣤⡇⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢸⢏⢀⠎⡢⡧⣾⢎⡠⠜⠒⠤⠼⣯⡧⡣⡸⡊⠱⡸⠘⡄⡜⡇⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢸⠀⠱⢔⣕⡵⠛⠁⠀⠀⠀⠀⠀⠈⣿⡽⡣⡘⡴⠱⡀⣹⠀⡇⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢸⣀⠤⠊⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⡟⣿⣳⠻⡘⡄⡱⣁⠭⠃⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠑⠒⠢⠤⣀⡀⠀⠀⠀⠀⠀⠀⠀⡇⠈⠳⣣⡱⠞⠋⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠉⠑⠒⠤⢄⣀⡀⣇⡠⠔⠉⠀⠀x⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀y⠀⠀⠀⠀⠀⠀⠈⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀

* Domain:            x ∈ [-2, 2], y ∈ [-2, 2], z ∈ [-4, 4]
* Camera:            az = 35°, el = 25°
* Saddle Point:      [0, 0, 0]
```

### 2.2 Perspective Projection, Camera Parameters & Local Minimum Detection

Optional arguments can be passed in any order after the range arguments.

```bash
ihd --lang en "plot3d(x^2 + y^2, [-2, 2], [-2, 2], [az, 45], [el, 30], \"perspective\")"
```

```text
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣀⣀⠤⠤⢒⡖⠢⠤⢄⣀⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢠⣤⡒⠒⠉⠉⠁⠀⠀⠀⢀⡇⢸⡀⠀⠀⠀⠀⠉⠉⠒⢒⣤⡄⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⣿⠪⡑⠒⠤⣀⠀⠀⢀⢮⢑⡇⡵⡀⠀⠀⣀⡠⠔⢊⡕⣿⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢫⢷⣏⢢⢄⡀⣉⣲⠺⣌⡇⢸⣁⠷⣖⣉⢀⡠⡔⢹⡲⡟⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠸⡀⢫⣾⣜⢌⡇⢏⣢⢫⡜⣿⡝⣔⡹⢸⡡⣣⣷⡕⢁⠇⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢇⠀⢻⣾⣝⡽⡙⣌⡮⡀⣇⢵⣡⢋⢯⣫⣷⡟⠁⡸⠀⠀z⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢸⠀⠀⠘⣾⢽⢝⠫⡀⢑⡏⢀⠝⡫⡯⣷⠃⠀⠀⡇⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢧⠒⠉⠀⠑⠳⣕⣎⣣⣟⣱⣪⠖⠋⠀⠉⠒⡼⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠑⠤⡀⠀⠀⠀⠉⠉⡏⠉⠀⠀⠀⠀⡠⠊⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠢⡀⠀⠀⠀⡇⠀⠀⢀⠤⠊⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀y⠀⠀⠈⠢⢄⠀⡇⢀⠔⠁⠀⠀x⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠑⠗⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀

* Domain:            x ∈ [-2, 2], y ∈ [-2, 2], z ∈ [0, 8]
* Camera:            az = 45°, el = 30°
* Local Min:         [0, 0, 0]
```

| Option Syntax | Default | Allowed Range & Boundary Behavior |
| :--- | :--- | :--- |
| `"ortho"` / `"perspective"` | `"ortho"` | Selects orthographic (`"orthographic"`) or perspective (`"persp"`) projection |
| `"mono"` / `"no_color"` | ANSI color enabled | Disables depth-shaded ANSI 256-color rendering for monochrome output |
| `[az, deg]` / `[azimuth, deg]` | `35` | Finite real value in degrees; non-numeric or infinite values raise an error |
| `[el, deg]` / `[elevation, deg]` | `25` | Degrees; values outside `[-90, 90]` are clamped to the bounds |
| `[dist, val]` / `[distance, val]` | `4` | Perspective camera distance; `val > 1.8` is required (`<= 1.8` raises an error) |
| `[aspect, val]` | `1` | Positive real value (`val > 0`); `<= 0` raises an error |
| `[z, min, max]` | Auto (5th–95th percentile) | Explicit Z-axis bounds; `min < max` is required (`min >= max` raises an error) |

### 2.3 3D Space Curves & Vector Export

Passing a 3-element list `[x(t), y(t), z(t)]` as the first argument enables space-curve mode (any other list length raises an error).

```bash
ihd --lang en "plot3d([cos(t), sin(t), t/4], [0, 4*pi])"
```

```text
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣀⠔⠊⢹⠈⠉⠑⠒⠤⢄⣀⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⡠⠔⠊⠀⣀⣀⣸⣀⡀⠀⠀⠀⠀⠀⠈⠉⠒⠢⠤⢄⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢠⢖⣉⣠⠖⠉⠉⠀⠀⢸⠀⠈⠉⠉⠒⠒⠤⣀⠀⢀⡠⠒⠉⡇⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢸⠀⠀⠈⠛⠶⠦⠤⢄⣸⠀⠀⠀⠀⠀⠀⢀⠤⠛⢅⠀⠀⠀⡇⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢸⠀⠀⠀⠀⠀⠀⠀⠀⢸⠉⠑⠒⠢⡔⠊⠁⠀⠀⠈⡆⠀⠀⡇⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢸⠀⠀⠀⠀⣀⣀⠤⠤⢼⠤⢄⣀⣀⡇⠀⠀⠀⠀⡴⠁⠀⠀⡇⠀z⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢸⠀⠀⢰⡉⠀⠀⠀⠀⢸⠀⠀⠀⠀⡏⣉⡲⠴⡉⠀⠀⠀⠀⡇⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢸⠀⠀⠀⠉⠑⠒⢒⡲⠞⠲⠶⢖⣉⡏⠀⠀⠀⠈⢲⠀⠀⠀⡇⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢸⠀⠀⠀⢀⡠⠒⠁⠀⠀⠀⠀⠀⠀⡏⠑⠒⠢⠤⣀⡇⠀⠀⡇⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢸⣀⠤⠊⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⡇⠀⠀⠀⣀⠞⠈⣉⠵⠃⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠑⠒⠢⠤⣀⡀⠀⠀⠀⠀⠀⠀⣀⣇⠤⠒⠉⡠⠔⠊⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠉⠙⠛⠭⢍⣉⡀⣇⡠⠔⠉⠀⠀x⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀y⠀⠀⠀⠀⠀⠀⠈⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
  ⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀

* Domain:            t ∈ [0, 4*π], z ∈ [0, 1571/500]
* Camera:            az = 35°, el = 25°
```

Both surfaces and space curves export depth-sorted SVG, Typst (`@preview/cetz:0.5.2`), and `plot3d_svg` files using the same flags as 2D plots. The commands below export the saddle surface from Section 2.1:

```bash
ihd --format svg --out saddle.svg "plot3d(x^2 - y^2, [-2, 2], [-2, 2])"
ihd --typst --out saddle.typ "plot3d(x^2 - y^2, [-2, 2], [-2, 2])"
ihd "plot3d_svg(x^2 - y^2, [-2, 2], [-2, 2], \"saddle.svg\")"
```

---

## 3. Terminal 2D Pretty Printing & Typst / LaTeX Math Export

### 3.1 2D Pretty Printing (`--pretty`)

Renders fractions and square roots in a multi-line ASCII/Unicode box model.

```bash
ihd --pretty "1/2 + sqrt(2)/2"
```

```text
 1     √2 
--- + ----
 2     2  
```

### 3.2 Typst Math Export (`--typst` / `--format typst`)

Outputs expressions in Typst math syntax (`$ ... $`) ready for technical documents.

```bash
ihd --typst "1/2 + sqrt(2)/2"
# Output: $1 / 2 + (sqrt(2)) / 2$

ihd --format typst "[[1, 2], [3, 4]]"
# Output: $mat(1, 2; 3, 4)$
```

### 3.3 LaTeX Math Export (`--latex` / `--format latex`)

Outputs expressions in LaTeX display math syntax (`$$ ... $$`).

```bash
ihd --latex "1/2 + sqrt(2)"
# Output: $$ \frac{1}{2} + \sqrt{2} $$

ihd --format latex "[[1, 2], [3, 4]]"
# Output: $$ \begin{pmatrix} 1 & 2 \\ 3 & 4 \end{pmatrix} $$
```

---

## 4. Algebraic Derivation Trees & Formal Projections (`--explain` / `--typst` / `--lean`)

`ihd` records each algebraic transformation in a unified Provenance DAG and projects the same derivation graph into Unicode 2D trees, Typst reports, and Lean 4 theorem code.

### 4.1 Unicode 2D Derivation Tree

Displays differentiation rules and L'Hopital's rule steps hierarchically during limit evaluation.

```bash
ihd --lang en --explain "limit((sqrt(1 + x) - 1 - x/2) / x^2, x, 0)"
```

```text
Expression: limit((sqrt(1 + x) - 1 - x/2) / x^2, x, 0)
├── [Step 1: Power Rule of Differentiation]
│   Power rule of differentiation: d/dx [x^2] = 2*x
│   x^2  ──>  2*x
│
├── [Step 2: Limit Computation & Indeterminate Form Resolution]
│   L'Hopital's rule applied: (d/dx [-1 + sqrt(1 + x) + -x/2]) / (d/dx [x^2]) = (-1/2 + 1/(2 * sqrt(1 + x))) / (2 * x)
│   -x^2^-1 - 1/2*x*x^2^-1 + √(1 + x)*x^2^-1  ──>  -1/(4*x) + 1/4*√(1 + x)^-1*x^-1
│
├── [Step 3: Power Rule of Differentiation]
│   Power rule of differentiation: d/dx [√(1 + x)^-1] = -√(1 + x)^-3/2
│   √(1 + x)^-1  ──>  -√(1 + x)^-3/2
│
├── [Step 4: Limit Computation & Indeterminate Form Resolution]
│   L'Hopital's rule applied: (d/dx [-1/2 + 1/(2 * sqrt(1 + x))]) / (d/dx [2 * x]) = (-1/4 * sqrt(1 + x)^-3) / (2)
│   -1/(4*x) + 1/4*√(1 + x)^-1*x^-1  ──>  -√(1 + x)^-3/8
│
├── [Step 5: Limit Computation & Indeterminate Form Resolution]
│   Limit computation: lim_{x->0} -1 * x^2^-1 + sqrt(1 + x) * x^2^-1 + -1/2 * x * x^2^-1 = -1/8
│   -x^2^-1 - 1/2*x*x^2^-1 + √(1 + x)*x^2^-1  ──>  -1/8
└── [Result]
    = -1/8
```

### 4.2 Partial Fraction Decomposition with Nonzero Side Conditions & Lean 4 Proof

For rational partial fraction decomposition (`apart`), `ihd` automatically extracts `NonZero` denominator side conditions and injects them as hypotheses into the generated Lean 4 theorem.

```bash
ihd --lang en --explain --lean "apart(1/(x^2 - 1), x)"
```

```text
Expression: apart(1/(x^2 - 1), x)
├── [Step 1: Polynomial Division and Common-Denominator Identity (Layer 1)]
│   1 = 1
├── [Step 2: Partial Fraction Rational Identity with Nonzero Conditions (Layer 2)]
│   (-1 + x^2)^-1 = 1/(2 * (-1 + x)) + -1/(2 * (1 + x))
│   [Requires: NonZero: denominator factor must be non-zero (-1 + x)]
│   [Requires: NonZero: denominator factor must be non-zero (1 + x)]
│   [Requires: NonZero: denominator must be non-zero (-1 + x^2)]
└── [Result]
    = 1/(2 * (-1 + x)) + -1/(2 * (1 + x))
```

```lean
-- Automatically generated by ihd (i-hate-decimal-calc) Lean 4 Transpiler
-- Mathematical proof certificate certified with Skeptic's verification
import Mathlib.Tactic.FieldSimp
import Mathlib.Tactic.Ring

variable (x : ℚ)

theorem ihd_certified_derivation (h_nz_0 : ((-1) + x) ≠ 0) (h_nz_1 : (1 + x) ≠ 0) (h_nz_2 : ((-1) + x ^ 2) ≠ 0) : (1 = 1) ∧ ((((-1) + x ^ 2) ^ (-1 : ℤ)) = (((1 : ℚ) / 2) * ((-1) + x) ^ (-1 : ℤ) + ((-1 : ℚ) / 2) * (1 + x) ^ (-1 : ℤ))) := by
  constructor
  · ring
  · field_simp; try ring

#print axioms ihd_certified_derivation
```

### 4.3 Typst Report Output (`--explain --typst`)

Instead of a terminal tree, this option outputs a standalone A4 Typst report ready to compile to PDF via the Typst CLI.

```bash
ihd --lang en --explain --typst "factor(x^4 + 4)"
```

```typst
// Generated by ihd (Exact Arithmetic Calculator)
#set page(paper: "a4", margin: 2.5cm)
#set text(font: "Linux Libertine", size: 11pt)

= Mathematical Derivation Report

*Initial Expression:* $ factor(x^4 + 4) $

== Derivation Steps

=== Step 1: Factorization

$ 4 + x^4 = (x^2 + -2 * x + 2) * (x^2 + 2 * x + 2) $

== Final Result

#align(center)[#block(fill: luma(240), inset: 12pt, radius: 4pt)[*Result:* $ (x^2 + -2 * x + 2) * (x^2 + 2 * x + 2) $]]
```

### 4.4 Lean 4 Equational Chain Proof Output (`--explain --lean`)

Transpiles derivation steps into a Lean 4 `calc` equational proof (stdout prints the derivation tree followed by the Lean 4 source).

```bash
ihd --lang en --explain --lean "factor(x^4 + 4)"
```

```text
Expression: factor(x^4 + 4)
├── [Step 1: Factorization]
│   4 + x^4 = (x^2 + -2 * x + 2) * (x^2 + 2 * x + 2)
└── [Result]
    = (x^2 + -2 * x + 2) * (x^2 + 2 * x + 2)
```

```lean
-- Automatically generated by ihd (i-hate-decimal-calc) Lean 4 Transpiler
-- Mathematical proof certificate certified with Skeptic's verification
import Mathlib.Tactic.Ring

variable (x : ℚ)

theorem ihd_certified_derivation : (4 + x ^ 4) = ((x ^ 2 + (-2) * x + 2) * (x ^ 2 + 2 * x + 2)) := by
  calc
    (4 + x ^ 4) = ((x ^ 2 + (-2) * x + 2) * (x ^ 2 + 2 * x + 2)) := by ring

#print axioms ihd_certified_derivation
```

---

## 5. REPL Real-Time Signature Guide & Syntax Highlighting

Powered by a zero-dependency lexical finite-state machine (FSM) for interactive terminal sessions.

| Feature | Behavior & REPL Commands |
| :--- | :--- |
| **Real-Time Signature Guide** | Highlights the active argument index while typing and displays parameter names and localized docs below the prompt |
| **Rainbow Nested Delimiters** | Rotates `()` and `[]` nesting depths through 4 colors; unmatched brackets receive a red error background |
| **Custom Variable Highlighting** | Distinguishes user-defined session variables in pale cyan from unbound symbols |
| **Theme Selection** | Toggle with `theme dark` / `theme light` / `theme none` (persisted to `~/.ihd/config.json`) |
| **Session Save & Restore** | Save defined variables via `save <path.ihd>` and restore them with `load <path.ihd>` |
| **Structural Similarity Search** | Search history by AST structure similarity using `similar(<expr>)` |

---

## 6. Physical Units, Collections & Semantic Diff

### 6.1 Exact Rational Vector-Space Physical Units & Dimensional Analysis

Performs exact dimensional analysis over the 7-dimensional rational vector space $`\mathbb{Q}^7`$ of SI base units without floating-point approximations.

```bash
# Multiplication and unit cancellation
ihd "10 [m/s] * 5 [s]"
# Output: 50 [m]

# Prefix normalization in subtraction
ihd "1 [km] - 200 [m]"
# Output: 4/5 [km]

# Exact rational unit conversion
ihd "convert_unit((1/4) [h], min)"
# Output: 15 [min]

# Verified conversion certificate
ihd --verify "convert_unit(100 [km/h], m/s)"
# Output:
# 250/9 [m/s]
# [VERIFIED: convert_unit(100[km * h^-1], m * s^-1) == 250/9 [m/s]]
```

### 6.2 Collection Slicing & First-Class Lambda Expressions

Supports Python-style slice syntax (`[start:end:step]`), rational-step progressions `range`, first-class anonymous functions `lambda`, and higher-order functions `map` and `filter`.

```bash
ihd "map(lambda(x, x^2), range(1, 5))"
# Output: [1, 4, 9, 16]

ihd "[10, 20, 30, 40, 50][1:4]"
# Output: [20, 30, 40]
```

### 6.3 Semantic Diff Inspection (`diff`)

Compares two expressions or `.ihd` scripts to detect algebraic equivalence (`EQUIVALENT`) or mathematical differences (`CHANGED`).

```bash
ihd --lang en diff "x^2 - 1" "(x - 1)*(x + 1)"
# Output:
# Semantic Diff: x^2 - 1 <-> (x - 1)*(x + 1)
#   ~ [EQUIVALENT] L1: x^2 - 1 <=> (x - 1)*(x + 1)
```

---

## 7. First-Class Closed-Domain AST Arithmetic (`padic_expand` / `puiseux` / `interval`)

Truncated $`p`$-adic series, Puiseux series, and closed rational intervals remain first-class AST nodes rather than degenerating into strings. Downstream arithmetic, differentiation, and three-valued comparisons preserve precision, ramification order, and interval dependency directly.

### 7.1 Truncated p-adic Series (`padic_expand`)

Requires a prime $`p \ge 2`$ and positive integer precision `prec` (composite $`p`$ or division by truncated zero raises an error).

```bash
ihd "padic_expand(1/3, 5, 4)"
# Output: 2 + 3*5 + 5^2 + 3*5^3 + O(5^4)

ihd "padic_expand(1/3, 5, 4) * 3"
# Output: 1 + O(5^4)
```

### 7.2 Puiseux Series (`puiseux`)

Expands branches of an algebraic curve $`F(x, y) = 0`$ as a list of first-class series nodes that directly support differentiation and arithmetic.

```bash
ihd "puiseux(y^2 - x, y, x, 3)"
# Output: [-x^(1/2) + O(x^3), x^(1/2) + O(x^3)]

ihd "diff(puiseux(y^2 - x, y, x, 3)[1], x)"
# Output: (1/2)*x^(-1/2) + O(x^2)
```

### 7.3 Closed Rational Intervals (`interval`)

Performs dependency-preserving interval arithmetic and 3-valued ordering (`low > high` or negative powers of intervals containing `0` raise domain errors).

```bash
ihd "interval([1, 2]) - interval([1, 2])"
# Output: [-1, 1] (width: 2)

ihd "interval([1, 2]) < interval([3, 4])"
# Output: true

ihd "interval([1, 3]) < interval([2, 4])"
# Output: [1, 3] (width: 2) < [2, 4] (width: 2)
```

---

## 8. Rational SOS, Ideal Membership & Algebraic Real Certificates

### 8.1 Rational SOS Decomposition & Lean 4 Nonnegativity Proof (`sos_decompose`)

Searches for a rational Sum-of-Squares (SOS) certificate for multivariate polynomials and returns one of three statuses (`certified` / `refuted` / `no_certificate_found`). Certified results support `--verify` residual checking and `--lean` nonnegativity theorem generation.

```bash
# Rational SOS certificate with autonomous verification
ihd --verify "sos_decompose(x^4 + 2*x^2*y^2 + y^4 + 1)"
# Output:
# [certified, [[1, x^2 + y^2], [1, 1]]]
# [VERIFIED: x^4 + 2 * x^2 * y^2 + y^4 + 1 == (x^2 + y^2)^2 + (1)^2]

# Counterexample witness (refuted) for an indefinite polynomial
ihd "sos_decompose(x^2 - y^2)"
# Output: [refuted, [[x, 0], [y, 1]]]
```

```bash
# Emit Lean 4 nonnegativity proof code
ihd --lean "sos_decompose(x^4 + 2*x^2*y^2 + y^4 + 1)"
```

```lean
-- Automatically generated by ihd (i-hate-decimal-calc) Lean 4 Transpiler
-- Mathematical proof certificate certified with Skeptic's verification
import Mathlib.Data.Rat.Defs
import Mathlib.Tactic.Positivity
import Mathlib.Tactic.Ring

variable (x y : ℚ)

theorem ihd_certified_proof : 0 ≤ x ^ 4 + 2 * x ^ 2 * y ^ 2 + y ^ 4 + 1 := by
  have h_sos : x ^ 4 + 2 * x ^ 2 * y ^ 2 + y ^ 4 + 1 = (x ^ 2 + y ^ 2)^2 + (1)^2 := by ring
  rw [h_sos]
  positivity

#print axioms ihd_certified_proof
```

### 8.2 Ideal Membership Cofactor Certificate (`ideal_member`)

Decides polynomial ideal membership via a GRevLex Groebner basis and returns explicit cofactors when membership holds.

```bash
ihd --verify "ideal_member(x^2 - y^2, [x - y, x + y])"
# Output:
# [true, [x/2 + y/2, -y/2 + x/2]]
# [VERIFIED: x^2 + -y^2 == (x/2 + y/2) * (x + -y) + (x/2 + -y/2) * (x + y)]
```

### 8.3 VAS Root Isolation, Algebraic Reals & Constructibility

Supports Vincent-Akritas-Strzeboński (VAS) real root isolation, first-class algebraic real arithmetic, and Wantzel's straightedge-and-compass constructibility decision (`true` / `false`, or `undetermined` for degree 8 and higher powers of 2).

```bash
# VAS real root isolation (irrational roots returned as exact isolating intervals)
ihd "isolate_roots_vas(x^3 - 2)"
# Output: [[9/8, 3/2]]

# First-class algebraic real construction and addition (ascending coefficients from constant term, interval, root index)
ihd "alg_real([-2, 0, 1], [1, 2], 0) + alg_real([-2, 0, 1], [1, 2], 0)"
# Output: 2*AlgReal(x^2-2, [1, 2], 0)

# Doubling the cube (x^3 - 2 = 0) constructibility test
ihd "is_constructible(x^3 - 2)"
# Output: false
```

---

## 9. Formal Power Series, Combinatorics, Error-Correcting Codes & Discrete Math

### 9.1 Formal Power Series & Combinatorics

The truncation order `prec` takes a positive integer (`>= 1`); zero, negative, or non-integer values raise an error.

```bash
ihd "fps_exp(x, 5)"
# Output: 1 + x + x^2/2 + x^3/6 + x^4/24

ihd "fps_inv(1 - x, 6)"
# Output: 1 + x + x^2 + x^3 + x^4 + x^5

ihd "partitions(10)"
# Output: 42

ihd "catalan(5)"
# Output: 42
```

### 9.2 Finite-Field Error-Correcting Codes & Graph Theory

Reed-Solomon decoding via Berlekamp-Welch recovers up to $`\lfloor (n - k)/2 \rfloor`$ symbol errors over $`\mathbb{F}_p`$ and raises a decoding error when more symbols are corrupted.

```bash
# Reed-Solomon encoding over F_7, 1-symbol error injection, and decoding
ihd "rs_encode([1, 2], [0, 1, 2, 3, 4], 7)"
# Output: [1, 3, 5, 0, 2]

ihd "hamming_dist([1, 3, 5, 0, 2], [1, 3, 5, 6, 2])"
# Output: 1

ihd "rs_decode([1, 3, 5, 6, 2], [0, 1, 2, 3, 4], 2, 7)"
# Output: [1, 2]

# Spanning trees and chromatic polynomial of complete graph K_3
ihd "spanning_trees([[0,1,1],[1,0,1],[1,1,0]])"
# Output: 3

ihd "chromatic_poly([[0,1,1],[1,0,1],[1,1,0]])"
# Output: 2*k - 3*k^2 + k^3
```

### 9.3 Discrete Probability & Game Theory

```bash
# Dice sum probability and Markov chain stationary distribution
ihd "dice_prob(2, 6, 7)"
# Output: 1/6

ihd "markov_stationary([[1/2, 1/2], [1/3, 2/3]])"
# Output: [2/5, 3/5]

# Two-player zero-sum Nash equilibrium, mex, and Nim-sum
ihd "nash_zero_sum([[1, -1], [-1, 1]])"
# Output: [0, [1/2, 1/2], [1/2, 1/2]]

ihd "mex([0, 1, 2, 4])"
# Output: 3

ihd "nim_sum(3, 4, 5)"
# Output: 2
```

---

## 10. Advanced & Specialized One-Shot Examples

### 10.1 Analysis & Geometry Closed Forms and Theorem Proving

```bash
# Elementary transcendental equation solving (Lambert W function)
ihd "solve(x*exp(x) == 1, x)"
# Output: [lambert_w(1)]

# Gaussian integral closed form (Error function erf)
ihd "integrate(exp(-x^2), x)"
# Output: 1/2*√(π)*erf(x)

# Automated geometric theorem proving (Wu's method)
ihd "geo_prove([midpoint(M, A, B), midpoint(N, A, C)], parallel(M, N, B, C))"
# Output: true
```

### 10.2 Galois Theory, Arithmetic Geometry, Group Structure & CRT

```bash
# Abel-Ruffini algebraic insolvability verification
ihd --verify "is_solvable_by_radicals(x^5 - 4*x + 2)"
# Output:
# false
# [VERIFIED: AbelRuffini(2 + x^5 + -4 * x)]

# Elliptic curve point count over finite fields (Schoof's algorithm): ec_p_order([A, B], p)
ihd "ec_p_order([1, 1], 5)"
# Output: 9

# Integer matrix normal forms and finitely generated abelian group structure (HNF / SNF)
ihd "abelian_group_structure([[2, 4], [4, 2]])"
# Output: Z_2 + Z_6
```

For the Chinese Remainder Theorem (`crt`), `--explain` displays the congruence verification steps for the returned solution:

```bash
ihd --lang en --explain "crt([2, 3], [3, 5])"
```

```text
Expression: crt([2, 3], [3, 5])
├── [Step 1: Chinese Remainder Theorem Congruence Verification]
│   mod(8, 3) = 2
├── [Step 2: Chinese Remainder Theorem Congruence Verification]
│   mod(8, 5) = 3
└── [Result]
    = 8
```
