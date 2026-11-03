#!/usr/bin/env python3
"""Proof Debt & Tactic Coverage Catalog Generator for i-hate-decimal-calc (ihd).

Scans test cases in formal/test_cases.txt, generated Lean proofs in formal/Proofs,
and outputs a structured debt ledger (formal/proof_debt.json) and markdown summary.
Enforces a ratchet policy: CI fails if total uncertified/sorry debt increases.
"""

import os
import sys
import json
import re
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent.parent
TEST_CASES_FILE = REPO_ROOT / "formal" / "test_cases.txt"
PROOFS_DIR = REPO_ROOT / "formal" / "Proofs"
CATALOG_OUTPUT = REPO_ROOT / "formal" / "proof_debt.json"


def safe_relpath(p: Path) -> str:
    try:
        return str(p.relative_to(REPO_ROOT))
    except ValueError:
        return str(p)


def scan_test_cases():
    cases = []
    if not TEST_CASES_FILE.exists():
        return cases
    with open(TEST_CASES_FILE, "r", encoding="utf-8") as f:
        case_id = 1
        for line in f:
            line = line.strip()
            if not line or line.startswith("#"):
                continue
            cases.append({"id": case_id, "expr": line})
            case_id += 1
    return cases


def analyze_proofs(cases):
    catalog = []
    tactics_distribution = {
        "ring": 0,
        "norm_num": 0,
        "linear_combination": 0,
        "positivity": 0,
        "linarith": 0,
        "omega": 0,
        "calc": 0,
        "sorry": 0,
    }

    total_theorems = 0
    full_coverage = 0
    partial_coverage = 0
    zero_coverage = 0

    for case in cases:
        idx = case["id"]
        expr = case["expr"]
        proof_file = PROOFS_DIR / f"Proof_{idx}.lean"

        entry = {
            "id": idx,
            "expr": expr,
            "proof_file": str(proof_file.relative_to(REPO_ROOT)) if proof_file.exists() else None,
            "status": "None",
            "tactics": [],
            "line_count": 0,
            "has_sorry": False,
        }

        if proof_file.exists():
            content = proof_file.read_text(encoding="utf-8")
            entry["line_count"] = len(content.splitlines())

            # Detect tactics
            found_tactics = []
            for tac in tactics_distribution.keys():
                if re.search(r"\b" + re.escape(tac) + r"\b", content):
                    tactics_distribution[tac] += 1
                    found_tactics.append(tac)

            entry["tactics"] = found_tactics
            if "sorry" in found_tactics or "sorryAx" in content:
                entry["has_sorry"] = True
                entry["status"] = "Partial"
                partial_coverage += 1
            else:
                entry["status"] = "Full"
                full_coverage += 1
        else:
            zero_coverage += 1

        total_theorems += 1
        catalog.append(entry)

    summary = {
        "total_test_cases": total_theorems,
        "full_coverage": full_coverage,
        "partial_coverage": partial_coverage,
        "none_coverage": zero_coverage,
        "coverage_rate_percent": round((full_coverage / total_theorems * 100), 2) if total_theorems > 0 else 0.0,
        "tactics_distribution": tactics_distribution,
    }

    return {"summary": summary, "entries": catalog}


def main():
    print("=== Scanning ihd Formal Proof Debt and Tactic Coverage ===")
    cases = scan_test_cases()
    print(f"Loaded {len(cases)} test cases from {safe_relpath(TEST_CASES_FILE)}")

    data = analyze_proofs(cases)
    summary = data["summary"]

    print(f"Summary:")
    print(f"  Total Test Cases: {summary['total_test_cases']}")
    print(f"  Full Verified   : {summary['full_coverage']} ({summary['coverage_rate_percent']}%)")
    print(f"  Partial (Sorry) : {summary['partial_coverage']}")
    print(f"  Unverified      : {summary['none_coverage']}")
    print(f"  Tactic Distribution: {summary['tactics_distribution']}")

    # Check ratchet policy against existing debt file
    if CATALOG_OUTPUT.exists():
        try:
            old_data = json.loads(CATALOG_OUTPUT.read_text(encoding="utf-8"))
            old_unverified = old_data.get("summary", {}).get("none_coverage", 0) + old_data.get("summary", {}).get("partial_coverage", 0)
            cur_unverified = summary["partial_coverage"] + summary["none_coverage"]
            if cur_unverified > old_unverified:
                print(f"[RATCHET FAIL] Proof debt increased! Old: {old_unverified}, Current: {cur_unverified}", file=sys.stderr)
                return 1
        except Exception as e:
            print(f"Notice: could not read old debt file: {e}")

    with open(CATALOG_OUTPUT, "w", encoding="utf-8") as f:
        json.dump(data, f, indent=2, ensure_ascii=False)
    print(f"Successfully generated catalog: {safe_relpath(CATALOG_OUTPUT)}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
