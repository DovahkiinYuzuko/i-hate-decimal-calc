/-
Copyright (c) 2026 i-hate-decimal-calc (ihd) contributors. All rights reserved.
Released under Apache 2.0 license as described in the file LICENSE.
Authors: Yuzuko Underson & Antigravity
-/
import Lean
import Lean.Util.CollectAxioms

open Lean

/--
Approved standard axioms for verified mathematics in i-hate-decimal-calc:
1. `propext` (Propositional Extensionality)
2. `Classical.choice` (Axiom of Choice)
3. `Quot.sound` (Quotient Soundness)
-/
def allowedAxiomNames : NameSet :=
  let names : List Name := [
    `propext,
    `Classical.choice,
    `Quot.sound
  ]
  names.foldl (init := {}) fun acc n => acc.insert n

/--
Execute a CoreM action with a given Environment, converting it safely to IO.
-/
def runCoreMInEnv {α : Type} (env : Environment) (act : CoreM α) : IO α := do
  let ctx : Core.Context := { fileName := "<audit>", fileMap := FileMap.ofString "", options := {} }
  let state : Core.State := { env := env }
  let (val, _) ← act.toIO ctx state
  return val

/--
Check a single declaration `declName` in the environment.
Returns `(isSound, disallowedAxioms, errorMsg)`.
-/
def auditDeclaration (env : Environment) (declName : Name) : IO (Bool × List Name × Option String) := do
  -- 1. Check if declaration exists
  let some cinfo := env.find? declName
    | return (false, [], some s!"Declaration '{declName}' not found in environment")

  -- 2. Anti-cheat check: Reject axioms, opaque without values, unsafe, extern, implemented_by
  match cinfo with
  | .axiomInfo _ =>
    return (false, [declName], some s!"Declaration '{declName}' is defined as an unverified axiom!")
  | .opaqueInfo v =>
    if v.value.hasLooseBVars || v.value.isSort then
      return (false, [declName], some s!"Declaration '{declName}' is an opaque declaration without closed value!")
  | _ => pure ()

  if cinfo.isUnsafe then
    return (false, [], some s!"Declaration '{declName}' is marked unsafe!")

  if Lean.isExtern env declName then
    return (false, [declName], some s!"Declaration '{declName}' is an external definition (extern)!")

  if (Lean.Compiler.getImplementedBy? env declName).isSome then
    return (false, [declName], some s!"Declaration '{declName}' is overridden via implemented_by!")

  -- 3. Transitive axiom collection using Lean.collectAxioms inside CoreM
  let usedAxioms ← runCoreMInEnv env (Lean.collectAxioms declName)
  let mut disallowed : List Name := []
  for ax in usedAxioms do
    if !allowedAxiomNames.contains ax then
      disallowed := ax :: disallowed

  if disallowed.isEmpty then
    return (true, [], none)
  else
    return (false, disallowed, some s!"Declaration '{declName}' depends on disallowed axioms: {disallowed}")

/--
Main CLI entry point for `lake exe ihd-audit`.
Usage:
  ihd-audit [--self-test] <TheoremName> ...
-/
def main (args : List String) : IO UInt32 := do
  if args.isEmpty then
    IO.eprintln "Usage: ihd-audit [--self-test] <TheoremName> ..."
    return 1

  if args.head? == some "--self-test" then
    IO.println "=== Running Stage 0 Axiom Audit Canary Self-Test ==="
    -- Test allowlist contains standard axioms
    if !allowedAxiomNames.contains `propext then
      IO.eprintln "SELF-TEST FAIL: propext not in allowlist"
      return 1
    if !allowedAxiomNames.contains `Classical.choice then
      IO.eprintln "SELF-TEST FAIL: Classical.choice not in allowlist"
      return 1
    if !allowedAxiomNames.contains `Quot.sound then
      IO.eprintln "SELF-TEST FAIL: Quot.sound not in allowlist"
      return 1
    if allowedAxiomNames.contains `sorryAx then
      IO.eprintln "SELF-TEST FAIL: sorryAx improperly allowed"
      return 1

    -- Test empty environment execution through runCoreMInEnv
    let baseEnv ← mkEmptyEnvironment
    let testRes ← runCoreMInEnv baseEnv (pure 42)
    if testRes != 42 then
      IO.eprintln "SELF-TEST FAIL: runCoreMInEnv failed basic execution"
      return 1

    IO.println "[CANARY PASS] Allowlist contains standard axioms and CoreM runner is sound."
    return 0

  initSearchPath (← findSysroot)
  -- Hoist importModules out of the loop to prevent RSS memory leaks and OOM
  IO.println s!"=== Loading Proofs module environment for audit ==="
  let env ← importModules #[{ module := `Proofs }] {}
  IO.println s!"=== Auditing {args.length} declarations for axiom and soundness integrity ==="
  let mut exitCode : UInt32 := 0

  for arg in args do
    let declName := arg.toName
    let (isSound, _disallowed, errOpt) ← auditDeclaration env declName
    if isSound then
      IO.println s!"[PASS] '{declName}' transitively depends only on approved axioms."
    else
      IO.eprintln s!"[REJECT] '{declName}' AUDIT FAILED!"
      if let some msg := errOpt then
        IO.eprintln s!"  Reason: {msg}"
      exitCode := 1

  return exitCode
