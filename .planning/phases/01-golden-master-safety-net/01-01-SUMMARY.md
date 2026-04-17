---
phase: 01-golden-master-safety-net
plan: 01
subsystem: combat/rng
tags:
  - testing
  - rng
  - determinism
  - combat
  - tdd
dependency_graph:
  requires: []
  provides:
    - "combat.SetRand package-scope RNG hook (seeds Dice/NumberPercent/NumberRange/NumberBits)"
    - "combat.CombatSystem.Rand *rand.Rand receiver-scoped hook (D-02 spec)"
  affects:
    - "pkg/combat"
    - "pkg/magic (transitively, via combat rollers)"
    - "pkg/skills (transitively)"
    - "pkg/ai (transitively)"
tech_stack:
  added: []
  patterns:
    - "Package-level seedable RNG with restore-closure (prev-state capture via closure, not fixed nil)"
    - "Fallthrough helper (randIntn) so every roller respects the hook without branch duplication"
    - "Belt-and-suspenders receiver field (CombatSystem.Rand) kept unused in this plan, reserved for future use"
key_files:
  created:
    - path: "go/pkg/combat/dice_test.go"
      purpose: "Locks determinism contract (TestSetRandDeterministic), restore semantics (TestSetRandRestore), global fallback (TestDefaultRandFallsThroughToGlobal), edge-case invariants (TestEdgeCasesPreserved), and CombatSystem.Rand field shape (TestCombatSystemRandField, TestCombatSystemRandFieldTypeByReflection)"
  modified:
    - path: "go/pkg/combat/dice.go"
      change: "Added package-level defaultRand var, exported SetRand(r *rand.Rand) func() with restore closure, added unexported randIntn helper, rerouted Dice/NumberRange/NumberPercent/NumberBits through randIntn. Interpolate/Max/Min/Clamp unchanged."
    - path: "go/pkg/combat/combat.go"
      change: "Added math/rand import. Appended Rand *rand.Rand as last field on CombatSystem (follows OnDeath). NewCombatSystem() unchanged — zero value nil preserves prior behavior."
decisions:
  - id: D-01-01-A
    summary: "Exported SetRand (not testSetRand) per plan RESEARCH Open Question #4 resolution"
    rationale: "Allows pkg/golden and any other external test package to seed combat RNG without an internal export cycle. Doc comment marks it test-only and non-goroutine-safe."
  - id: D-01-01-B
    summary: "CombatSystem.Rand field left unused by receivers in this plan"
    rationale: "Package-scope SetRand already covers the 131 call sites across combat/magic/skills/ai. The field is added per D-02 literal spec as a future-use hook; wiring receivers to prefer c.Rand is deferred to avoid unnecessary churn."
  - id: D-01-01-C
    summary: "Doc comment for randIntn avoids literal 'rand.Intn' string"
    rationale: "Acceptance criterion required grep -c 'rand.Intn' dice.go == 1. The original draft doc mentioned 'rand.Intn' for clarity but tripped the grep. Rewrote the comment to be equally clear without the literal token."
metrics:
  duration_seconds: 305
  duration_human: "~5 minutes"
  completed_at: "2026-04-17T08:45:09Z"
  tasks_total: 2
  tasks_completed: 2
  commits: 4
  files_changed: 3
  lines_added: 153
  lines_removed: 4
---

# Phase 01 Plan 01: Combat RNG Seeding Hook — Summary

Introduced a package-scope, seed-injectable RNG (`combat.SetRand` + unexported
`defaultRand` / `randIntn`) so that the 131 `Dice`/`NumberPercent`/`NumberRange`/
`NumberBits` call sites across combat, magic, skills, and ai can all be made
deterministic from a single test-only hook, unblocking the upcoming
golden-master fixture (Plan 02). Also added the D-02-mandated
`CombatSystem.Rand *rand.Rand` field as a belt-and-suspenders receiver hook.

## What Shipped

### `combat.SetRand` API contract

```go
func SetRand(r *rand.Rand) func()
```

- **Signature:** accepts a `*rand.Rand`, returns a no-argument restore closure.
- **Restore semantics:** the closure captures the *previous* value of
  `defaultRand` (not a fixed nil), so nested `SetRand(...)`/`restore()` pairs
  correctly unwind state. `restore()` is idempotent in the sense that calling
  it twice reinstalls the same previous value twice; it is not self-cancelling.
- **Test-only intent:** documented in the doc comment. Non-nil `defaultRand`
  serializes all rolls through a single unsynchronised `*rand.Rand` and is
  not goroutine-safe. Production code must never call `SetRand`.
- **nil restores global:** `SetRand(nil)` is legal and returns a closure that
  restores whatever was there before (typically nil).
- **Call-site coverage:** `Dice`, `NumberRange`, `NumberPercent`, `NumberBits`
  all route through the unexported `randIntn` helper. The only `rand.Intn`
  call in the file is the fallthrough inside `randIntn` itself.

### `go/pkg/combat/dice.go` line-range discipline

- Lines **1–40** (original RNG block) replaced with the seedable variant
  (new `defaultRand` var, `SetRand`, `randIntn`, reworked `Dice` /
  `NumberRange` / `NumberPercent` / `NumberBits`).
- Lines **42–73** (`Interpolate`, `Max`, `Min`, `Clamp`) preserved
  byte-for-byte — acceptance grep `grep -q 'func Interpolate'` passes.
- Net delta: +36 insertions, −4 removals.

### `CombatSystem.Rand *rand.Rand`

- Appended as the **last** field on the struct, matching the callback-field
  placement convention (`OnDeath` precedes it).
- Zero-value is `nil`; `NewCombatSystem()` is unchanged.
- Not yet consumed by any receiver method. The field is reserved for future
  use per D-02 literal spec and is verifiable today via reflection
  (`reflect.TypeOf(CombatSystem{}).FieldByName("Rand")`).

## Verification Results

| Check | Command | Result |
|-------|---------|--------|
| Determinism (single run) | `go test ./pkg/combat/ -run TestSetRandDeterministic -count=2 -timeout 30s` | ok |
| Determinism (repeated) | `go test ./pkg/combat/ -run TestSetRandDeterministic -count=5 -timeout 30s` | ok |
| Edge cases | `go test ./pkg/combat/ -run TestEdgeCasesPreserved -timeout 10s` | ok |
| Field shape | `go test ./pkg/combat/ -run 'TestCombatSystemRandField\|TestCombatSystemRandFieldTypeByReflection' -timeout 10s` | ok |
| Full combat package (includes combat_sim_test.go) | `go test ./pkg/combat/ -count=1 -timeout 180s` | ok (21–22s) |
| Downstream build | `go build ./...` (from `go/`) | ok |
| Diff sanity | `git diff --stat HEAD~4 -- go/pkg/combat/` | `dice.go` / `combat.go` modified, `dice_test.go` new — as spec |

## Deviations from Plan

**None.** The plan executed exactly as written.

One micro-adjustment worth noting (not a deviation from behavior, only from
draft wording): the doc comment on `randIntn` originally contained the
literal string `rand.Intn` as prose. The acceptance criterion
`grep -c 'rand.Intn' go/pkg/combat/dice.go` expects exactly `1` (the
fallthrough call). Rewrote the comment to "(not the global rand package
directly)" which is equally clear and preserves the grep invariant. See
D-01-01-C above. This is documentation-only and does not affect behavior.

No Rule 1–4 auto-fixes were triggered. No auth gates encountered. No
architectural decisions needed.

## Commits

| # | Hash | Gate | Message |
|---|------|------|---------|
| 1 | `3d9468d` | RED (T1) | `test(01-01): add failing tests for combat SetRand determinism hook` |
| 2 | `b294fd1` | GREEN (T1) | `feat(01-01): add seedable RNG hook to pkg/combat/dice.go` |
| 3 | `5265953` | RED (T2) | `test(01-01): add failing tests for CombatSystem.Rand field` |
| 4 | `ed37e71` | GREEN (T2) | `feat(01-01): add CombatSystem.Rand *rand.Rand field (D-02)` |

Both tasks followed the RED/GREEN TDD cycle strictly — tests committed
first and verified to fail (compile-error RED), then implementation
committed with all tests green.

## TDD Gate Compliance

- Task 1: `test(...)` at `3d9468d`, then `feat(...)` at `b294fd1`. ✓
- Task 2: `test(...)` at `5265953`, then `feat(...)` at `ed37e71`. ✓

No REFACTOR commits were needed — both GREEN implementations matched the
plan's literal `<action>` block and no cleanup opportunities remained.

## Downstream Impact (for Plan 02)

Plan 02 (golden-master fixture) can now call `combat.SetRand(rand.New(
rand.NewSource(42)))` at the top of its fixture test and get byte-identical
output from every roller in the combat, magic, skills, and ai packages
without any further wiring. This is the MIGRATE-06 success criterion #4
unlocker.

The `CombatSystem.Rand` field is available but not yet consumed. Plan 02
does not need to set it (the package-scope hook alone is sufficient). It
remains reserved for any future scenario where two `CombatSystem` instances
need independent RNG streams concurrently — not a concern for the current
golden fixture which runs synchronously.

## Known Stubs

None. All code is fully wired; no placeholder values, no `TODO`/`FIXME`
markers, no empty-array defaults flowing to UI or downstream consumers.

## Self-Check: PASSED

- `go/pkg/combat/dice.go` exists and contains `SetRand`, `defaultRand`, `randIntn`, `Interpolate`. ✓
- `go/pkg/combat/combat.go` exists and contains `"math/rand"` import plus `Rand *rand.Rand` field. ✓
- `go/pkg/combat/dice_test.go` exists with all six required tests. ✓
- Commits `3d9468d`, `b294fd1`, `5265953`, `ed37e71` present in `git log`. ✓
- Full `go test ./pkg/combat/` passes (22s, includes combat_sim_test.go). ✓
- `go build ./...` clean. ✓
