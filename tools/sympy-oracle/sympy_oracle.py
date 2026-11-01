#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
sympy_oracle.py
Resident SymPy oracle worker communicating via unbuffered NDJSON over stdin/stdout.
Designed for high-throughput differential fuzzing with i-hate-decimal-calc (ihd).

Key Safety & Architecture Invariants:
1. Pure NDJSON stdout: stdout is strictly reserved for single-line JSON responses with write_through=True.
   All warnings, logs, and tracebacks are redirected to stderr.
2. Generation & Request Tracking: Every response echoes the caller's generation ID and request ID
   to prevent pipeline desynchronization after process recycles.
3. Structured AST Construction: No raw eval() or sympify() on arbitrary strings.
   Expressions are built using explicit SymPy AST node constructors.
4. Exception Isolation: Unexpected evaluation errors return structured JSON errors without process crash.
5. Assumption Neutrality: Generic complex symbols without forced real=True to avoid false-positive
   simplifications when sampling over Q[i].
"""

import sys
import os
import json
import traceback

# Force UTF-8 and unbuffered line-level writing on standard streams
if sys.version_info >= (3, 7):
    sys.stdin.reconfigure(encoding="utf-8", newline="\n")
    sys.stdout.reconfigure(encoding="utf-8", newline="\n", write_through=True)
    sys.stderr.reconfigure(encoding="utf-8", newline="\n", write_through=True)

try:
    import sympy as sp
    import mpmath as mp
    from sympy.core.cache import clear_cache
except ImportError as e:
    sys.stderr.write(f"[sympy_oracle] Fatal: failed to import sympy/mpmath: {e}\n")
    sys.stderr.flush()
    sys.exit(1)


def build_ast(node):
    """
    Recursively builds a native SymPy expression from a structured JSON AST.
    Guarantees unambiguous operator precedence and eliminates symbol shadowing hazards.
    """
    if node is None:
        return sp.Integer(0)

    if isinstance(node, (int, float)):
        return sp.Integer(int(node))

    if isinstance(node, str):
        return sp.Symbol(node)

    if not isinstance(node, dict):
        raise ValueError(f"Invalid AST node format: expected dict, got {type(node)}")

    op = node.get("op")
    if not op:
        raise ValueError(f"Missing 'op' field in node: {node}")

    if op == "Rational":
        num = int(node.get("num", "0"))
        den = int(node.get("den", "1"))
        return sp.Rational(num, den)

    elif op == "Integer":
        val = int(node.get("val", "0"))
        return sp.Integer(val)

    elif op == "Symbol":
        name = node.get("name", "x")
        # Apply explicit assumption metadata if provided; default to neutral complex symbol
        assumptions = node.get("assumptions") or {}
        kwargs = {}
        for k, v in assumptions.items():
            if isinstance(k, str) and isinstance(v, bool):
                kwargs[k] = v
        return sp.Symbol(name, **kwargs)

    elif op == "Constant":
        name = node.get("name", "")
        if name in ("pi", "Pi"):
            return sp.pi
        elif name in ("e", "E"):
            return sp.E
        elif name in ("i", "I"):
            return sp.I
        elif name in ("infinity", "oo"):
            return sp.oo
        raise ValueError(f"Unknown constant: {name}")

    elif op == "Add":
        args = [build_ast(a) for a in node.get("args", [])]
        return sp.Add(*args) if args else sp.Integer(0)

    elif op == "Mul":
        args = [build_ast(a) for a in node.get("args", [])]
        return sp.Mul(*args) if args else sp.Integer(1)

    elif op == "Pow":
        base = build_ast(node.get("base"))
        exp = build_ast(node.get("exp"))
        return sp.Pow(base, exp)

    elif op == "Neg":
        arg = build_ast(node.get("arg"))
        return -arg

    elif op == "Sqrt":
        arg = build_ast(node.get("arg"))
        return sp.sqrt(arg)

    elif op == "Sin":
        return sp.sin(build_ast(node.get("arg")))
    elif op == "Cos":
        return sp.cos(build_ast(node.get("arg")))
    elif op == "Tan":
        return sp.tan(build_ast(node.get("arg")))
    elif op == "Asin":
        return sp.asin(build_ast(node.get("arg")))
    elif op == "Acos":
        return sp.acos(build_ast(node.get("arg")))
    elif op == "Atan":
        return sp.atan(build_ast(node.get("arg")))
    elif op == "Exp":
        return sp.exp(build_ast(node.get("arg")))
    elif op == "Log":
        return sp.log(build_ast(node.get("arg")))
    elif op == "Abs":
        return sp.Abs(build_ast(node.get("arg")))

    elif op == "Diff":
        arg = build_ast(node.get("arg"))
        var_name = node.get("var", "x")
        order = int(node.get("order", 1))
        # Match symbol in arg.free_symbols by name to preserve assumption metadata (e.g., real=True)
        var_symbol = None
        for sym in arg.free_symbols:
            if sym.name == var_name:
                var_symbol = sym
                break
        if var_symbol is None:
            assumptions = node.get("assumptions") or {}
            kwargs = {k: v for k, v in assumptions.items() if isinstance(k, str) and isinstance(v, bool)}
            var_symbol = sp.Symbol(var_name, **kwargs)
        return sp.diff(arg, var_symbol, order)

    elif op == "Expand":
        arg = build_ast(node.get("arg"))
        return sp.expand(arg)

    elif op == "Factor":
        arg = build_ast(node.get("arg"))
        return sp.factor(arg)

    elif op == "Simplify":
        arg = build_ast(node.get("arg"))
        return sp.simplify(arg)

    else:
        raise ValueError(f"Unsupported AST operation: {op}")


def format_gaussian_rat(val):
    """
    Converts a SymPy evaluated number into a structured Gaussian rational {re_num, re_den, im_num, im_den}
    or returns high-precision floating string if transcendental.
    """
    if val is None or val in (sp.zoo, sp.nan, sp.oo, -sp.oo):
        return {"undefined": True, "reason": str(val)}

    # Separate real and imaginary components
    re_part = sp.re(val)
    im_part = sp.im(val)

    # Check if exact rationals
    if isinstance(re_part, sp.Rational) and isinstance(im_part, sp.Rational):
        return {
            "undefined": False,
            "exact": True,
            "re_num": str(re_part.p),
            "re_den": str(re_part.q),
            "im_num": str(im_part.p),
            "im_den": str(im_part.q),
        }

    # Try exact conversion via as_numer_denom if integer/fractional
    try:
        p_re, q_re = re_part.as_numer_denom()
        p_im, q_im = im_part.as_numer_denom()
        if p_re.is_Integer and q_re.is_Integer and p_im.is_Integer and q_im.is_Integer:
            return {
                "undefined": False,
                "exact": True,
                "re_num": str(p_re),
                "re_den": str(q_re),
                "im_num": str(p_im),
                "im_den": str(q_im),
            }
    except Exception:
        pass

    # Tier-2/3 Fallback: High-precision mpmath evaluation (80 digits)
    try:
        val_eval = val.evalf(80)
        re_f = float(sp.re(val_eval))
        im_f = float(sp.im(val_eval))
        return {
            "undefined": False,
            "exact": False,
            "approx_str": str(val_eval),
            "re_float": re_f,
            "im_float": im_f,
        }
    except Exception as e:
        return {"undefined": True, "reason": f"Evaluation error: {e}"}


def handle_request(req):
    """Processes a single JSON-RPC style request dictionary."""
    req_id = req.get("id")
    generation = req.get("generation", 0)
    action = req.get("action")

    base_res = {
        "id": req_id,
        "generation": generation,
    }

    if action == "handshake":
        base_res.update({
            "status": "ok",
            "sympy_version": sp.__version__,
            "mpmath_version": mp.__version__,
            "pid": os.getpid(),
        })
        return base_res

    elif action == "clear_cache":
        clear_cache()
        base_res.update({"status": "ok", "cleared": True})
        return base_res

    elif action == "eval_points":
        expr_ast = req.get("expr")
        points = req.get("points", [])  # list of dict: {var: {re_num, re_den, im_num, im_den}}

        try:
            expr = build_ast(expr_ast)
        except Exception as e:
            base_res.update({"status": "error", "error": f"build_ast error: {e}"})
            return base_res

        results = []
        for pt in points:
            subs_map = {}
            for var_name, coord in pt.items():
                re_num = int(coord.get("re_num", "0"))
                re_den = int(coord.get("re_den", "1"))
                im_num = int(coord.get("im_num", "0"))
                im_den = int(coord.get("im_den", "1"))

                q_val = sp.Rational(re_num, re_den) + sp.Rational(im_num, im_den) * sp.I
                # Substitute all symbols matching the variable name regardless of assumption differences
                matched = False
                for sym in expr.free_symbols:
                    if sym.name == var_name:
                        subs_map[sym] = q_val
                        matched = True
                if not matched:
                    subs_map[sp.Symbol(var_name)] = q_val
                    subs_map[sp.Symbol(var_name, real=True)] = q_val

            try:
                val = expr.subs(subs_map)
                formatted = format_gaussian_rat(val)
                results.append(formatted)
            except ZeroDivisionError:
                results.append({"undefined": True, "reason": "division by zero (pole)"})
            except Exception as e:
                results.append({"undefined": True, "reason": str(e)})

        base_res.update({"status": "ok", "evaluations": results})
        return base_res

    elif action == "eval_numerical":
        # Tier-3 arbitrary-precision numerical equivalence checking with dual tolerance
        lhs_ast = req.get("lhs")
        rhs_ast = req.get("rhs")
        points = req.get("points", [])
        atol = float(req.get("atol", 1e-12))
        rtol = float(req.get("rtol", 1e-9))

        try:
            lhs = build_ast(lhs_ast)
            rhs = build_ast(rhs_ast)
        except Exception as e:
            base_res.update({"status": "error", "error": f"build_ast error: {e}"})
            return base_res

        all_agree = True
        max_residual = 0.0
        sample_count = 0
        point_results = []

        old_dps = mp.mp.dps
        mp.mp.dps = 80
        try:
            for pt in points:
                subs_map = {}
                for var_name, coord in pt.items():
                    re_num = int(coord.get("re_num", "0"))
                    re_den = int(coord.get("re_den", "1"))
                    im_num = int(coord.get("im_num", "0"))
                    im_den = int(coord.get("im_den", "1"))

                    q_val = sp.Rational(re_num, re_den) + sp.Rational(im_num, im_den) * sp.I
                    # Substitute all matching symbols in both LHS and RHS free symbols
                    matched = False
                    for sym in (lhs.free_symbols | rhs.free_symbols):
                        if sym.name == var_name:
                            subs_map[sym] = q_val
                            matched = True
                    if not matched:
                        subs_map[sp.Symbol(var_name)] = q_val
                        subs_map[sp.Symbol(var_name, real=True)] = q_val

                try:
                    val_l = lhs.subs(subs_map).evalf(80)
                    val_r = rhs.subs(subs_map).evalf(80)

                    if any(v in (sp.zoo, sp.nan, sp.oo, -sp.oo) for v in (val_l, val_r)):
                        point_results.append({"valid": False, "reason": "pole or undefined"})
                        continue

                    # Complex evaluation difference
                    c_l = complex(val_l)
                    c_r = complex(val_r)
                    diff = abs(c_l - c_r)
                    max_mag = max(abs(c_l), abs(c_r))
                    tol = atol + rtol * max_mag

                    agree = (diff <= tol)
                    if diff > max_residual:
                        max_residual = diff

                    if not agree:
                        all_agree = False

                    sample_count += 1
                    point_results.append({
                        "valid": True,
                        "agree": agree,
                        "residual": diff,
                        "tolerance": tol,
                    })
                except Exception as eval_err:
                    point_results.append({"valid": False, "reason": str(eval_err)})

        finally:
            mp.mp.dps = old_dps

        base_res.update({
            "status": "ok",
            "all_agree": all_agree and (sample_count >= 5),
            "sample_count": sample_count,
            "max_residual": max_residual,
            "point_results": point_results,
        })
        return base_res

    elif action == "subtractive_zero":
        lhs_ast = req.get("lhs")
        rhs_ast = req.get("rhs")

        try:
            lhs = build_ast(lhs_ast)
            rhs = build_ast(rhs_ast)
        except Exception as e:
            base_res.update({"status": "error", "error": f"build_ast error: {e}"})
            return base_res

        diff = lhs - rhs
        is_zero = diff.equals(0)
        simplified_str = ""
        is_constant_nonzero = False

        if not is_zero:
            try:
                simp = sp.simplify(diff)
                is_zero = (simp == 0)
                simplified_str = str(simp)
                if simp.is_number and simp != 0:
                    is_constant_nonzero = True
            except Exception as e:
                simplified_str = f"simplify_failed: {e}"

        base_res.update({
            "status": "ok",
            "is_zero": bool(is_zero),
            "is_constant_nonzero": bool(is_constant_nonzero),
            "simplified": simplified_str,
        })
        return base_res

    elif action == "canonical_str":
        expr_ast = req.get("expr")
        try:
            expr = build_ast(expr_ast)
            base_res.update({"status": "ok", "canonical": str(expr)})
        except Exception as e:
            base_res.update({"status": "error", "error": f"build_ast error: {e}"})
        return base_res

    else:
        base_res.update({"status": "error", "error": f"Unknown action: '{action}'"})
        return base_res


def main():
    # Process NDJSON line by line from stdin
    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue

        try:
            req = json.loads(line)
            res = handle_request(req)
        except json.JSONDecodeError as e:
            res = {"id": None, "generation": 0, "status": "error", "error": f"JSON decode error: {e}"}
        except Exception as e:
            res = {
                "id": None,
                "generation": 0,
                "status": "error",
                "error": f"Unhandled worker exception: {e}\n{traceback.format_exc()}",
            }

        # Emit strictly single-line NDJSON to stdout with explicit write_through
        try:
            out_line = json.dumps(res, ensure_ascii=True)
            sys.stdout.write(out_line + "\n")
            sys.stdout.flush()
        except Exception as e:
            sys.stderr.write(f"[sympy_oracle] stdout write error: {e}\n")
            sys.stderr.flush()


if __name__ == "__main__":
    main()
