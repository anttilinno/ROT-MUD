---
phase: 01-golden-master-safety-net
reviewed: 2026-04-17T00:00:00Z
depth: standard
files_reviewed: 6
files_reviewed_list:
  - go/pkg/combat/combat.go
  - go/pkg/combat/dice.go
  - go/pkg/combat/dice_test.go
  - go/pkg/golden/doc.go
  - go/pkg/golden/fixture.go
  - go/pkg/golden/golden_test.go
findings:
  critical: 0
  warning: 4
  info: 4
  total: 8
status: issues_found
---

# Phase 01: Code Review Report

**Reviewed:** 2026-04-17
**Depth:** standard
**Files Reviewed:** 6
**Status:** issues_found

## Summary

This phase adds a seedable RNG hook to `pkg/combat/dice.go`, a `Rand *rand.Rand` field to `CombatSystem`, and a new `pkg/golden/` test package with scenario fixtures and the `TestGolden` parity gate. The overall design is sound. The `SetRand` / `randIntn` indirection correctly captures all four rollers, and the golden-test architecture (seed-pinned, diff-on-mismatch, `-update` regeneration path) is a solid parity gate for the upcoming trait migration.

The issues below are concentrated in two areas:

1. **`CheckImmune` in `combat.go`** has a logic defect that renders all three of `immFlag`, `resFlag`, and `vulnFlag` identical, meaning the immunity/resistance/vulnerability distinction is silently lost — the function always uses the same constant for all three checks.
2. **`fixture.go`** contains a `joinStrings` helper that goes unused (replaced by the standard `strings.Join`-equivalent iteration that is already inlined), the duplicate-from-sim comment leaves the snippet boundary unverified, and several minor robustness gaps exist in the golden output path.

No security vulnerabilities were found. The `SetRand` global-mutation approach is correctly documented as non-goroutine-safe for production; test usage is safe because Go test binary executes each `testing.T` sequentially by default (unless `-parallel` is used, which is not done here).

---

## Warnings

### WR-01: `CheckImmune` always assigns the same constant to all three of `immFlag`, `resFlag`, `vulnFlag`

**File:** `go/pkg/combat/combat.go:332-352`

**Issue:** Every `case` in the `switch damType` block assigns the same `ImmFlags` constant to all three variables. For example:

```go
case types.DamFire:
    immFlag, resFlag, vulnFlag = types.ImmFire, types.ImmFire, types.ImmFire
```

The three variables exist precisely to let the caller distinguish "immune to fire" from "resistant to fire" from "vulnerable to fire" via separate flag constants (e.g., `ImmFire`, `ResFire`, `VulnFire`). Assigning the same constant to all three means `victim.Res.Has(resFlag)` tests the immunity bit rather than the resistance bit, so a character flagged only with `ResFire` (but not `ImmFire`) will never be detected as resistant to fire damage. The same defect applies to vulnerability.

This is pre-existing code not introduced in this phase, but the golden fixture captures output from `CheckImmune` (via `formatImmBits`) and will silently bake the wrong behavior into `testdata/entities.golden`. Once that snapshot is committed, the fixture becomes a regression test for the wrong behavior.

**Fix:** Use distinct flag constants per check (adjust constant names to match whatever `types` exposes — if the types package only has `Imm*` constants and not `Res*`/`Vuln*`, that is a types-layer bug to address there first):

```go
case types.DamFire:
    immFlag  = types.ImmFire
    resFlag  = types.ResFire
    vulnFlag = types.VulnFire
```

If `types.ResFire` / `types.VulnFire` do not exist yet, they must be added before the golden snapshot is cut — otherwise the snapshot will encode the defective behavior and serve as a false parity gate for the trait migration.

---

### WR-02: `TestGolden` mismatch failure message dumps entire file contents twice

**File:** `go/pkg/golden/golden_test.go:61-70`

**Issue:** On mismatch, `t.Fatalf` prints both `want` and `got` in full as strings. For the intended use case (a multi-kilobyte `entities.golden`), this produces a wall of unreadable output. More importantly, embedding `%s` of potentially large byte slices can trigger extremely slow string formatting and fill CI logs.

**Fix:** Emit only a concise header and the first point of divergence. A practical alternative is to write `got` to a temp file and ask the developer to diff it manually:

```go
tmp, _ := os.CreateTemp("", "golden-got-*.txt")
_, _ = tmp.Write(got)
tmp.Close()
t.Fatalf("golden mismatch at %s\ngot written to %s\ndiff: diff %s %s",
    path, tmp.Name(), path, tmp.Name())
```

Or at minimum, cap the printed excerpt:

```go
const maxShow = 2000
excerpt := func(b []byte) string {
    if len(b) > maxShow {
        return string(b[:maxShow]) + "\n...(truncated)"
    }
    return string(b)
}
```

---

### WR-03: `SetRand` is not goroutine-safe; `TestSetRandDeterministic` calls it twice without `t.Parallel()` guard, but the package-level variable creates a race if any future test sets `-parallel`

**File:** `go/pkg/combat/dice.go:23-27`, `go/pkg/combat/dice_test.go:25-37`

**Issue:** `defaultRand` is a plain package-level `*rand.Rand` pointer with no mutex. `SetRand` mutates it directly. The doc comment correctly warns this is not goroutine-safe. However, `TestSetRandDeterministic` calls `SetRand` / `restore` twice in sequence without holding any lock. If Go's test runner ever executes these tests with `-parallel` (or if a future contributor adds `t.Parallel()` to these tests), the two `SetRand` calls race with each other and with `randIntn` in other concurrent tests.

The current code is safe as written, but the design is one `t.Parallel()` annotation away from a data race. The `TestSetRandRestore` test even checks `defaultRand != nil` which is a plain pointer read that would race.

**Fix:** Either document the tests explicitly as serial-only (add a comment warning against `t.Parallel()`) or protect `defaultRand` with a `sync/atomic.Pointer[rand.Rand]`:

```go
import "sync/atomic"

var defaultRand atomic.Pointer[rand.Rand]

func SetRand(r *rand.Rand) func() {
    prev := defaultRand.Load()
    defaultRand.Store(r)
    return func() { defaultRand.Store(prev) }
}

func randIntn(n int) int {
    if r := defaultRand.Load(); r != nil {
        return r.Intn(n)
    }
    return rand.Intn(n)
}
```

Note that `rand.Rand.Intn` itself is still not goroutine-safe, so if concurrent tests each install a *different* `*rand.Rand` there is still a window; the atomic pointer only protects the install/read of the pointer itself. For fully isolated concurrent tests, each test would need its own `rand.Rand` passed via context rather than a global.

---

### WR-04: `fixture.go` `emitRaceWarriorCombo` / `emitClassHumanCombo` share room vnums 1 and 2, so `AddPerson` may accumulate stale entries if the room struct is reused

**File:** `go/pkg/golden/fixture.go:50-51`, `107-108`

**Issue:** Each iteration in `runRaceWarriorCombos` calls `types.NewRoom(1, ...)` and `types.NewRoom(2, ...)`, creating fresh room values, so there is no actual reuse. This is fine. However, the combat loop at lines 64-77 may leave `mob.Position <= types.PosDead` mid-loop and `break`, but `ch` and `mob` remain `AddPerson`'d to `room` which then goes out of scope. This is not a memory correctness issue (GC handles it), but it means the golden output line for a mob that died before round 30 reflects fewer than 30 rounds, silently. This is likely intentional, but the doc comment says "30-round combat log header" — add an explicit note that the count may be less on early kill.

**Fix:** Minor: update the comment at line 38 to say "up to 30 rounds" to avoid future confusion:

```go
// Captures race stat distribution, HP, immunity/vulnerability/resistance flags, and a
// deterministic up-to-30-round combat log header vs a standard warrior mob.
```

---

## Info

### IN-01: `joinStrings` helper in `fixture.go` is redundant — `strings.Join` is not imported but is available

**File:** `go/pkg/golden/fixture.go:388-398`

**Issue:** `joinStrings` is a hand-rolled string joiner. The standard `strings.Join` function does the same thing and is already in the standard library. The helper is only ever called from `formatImmBits` (line 385). Using `strings.Join` would remove 11 lines of code.

**Fix:**
```go
import "strings"

// in formatImmBits:
return "[" + strings.Join(names, ",") + "]"
```

---

### IN-02: `sort.Strings(names)` in `formatImmBits` sorts an already-stable list

**File:** `go/pkg/golden/fixture.go:374-380`

**Issue:** `immFlagNames` is declared in definition order (lines 346-370) and `formatImmBits` appends to `names` by iterating that slice in order. Because the slice is ordered, `names` will always come out in the same order without the `sort.Strings` call. The sort is harmless but adds a small allocation (sort needs a `[]string` of up to 21 elements). The doc comment at line 372 says "sorted, comma-joined" which is accurate; just note that the order is guaranteed by construction, not by sort.

**Fix:** Either remove the `sort.Strings` call and change the comment to "definition-ordered, comma-joined", or keep it and note it is defensive. If the `immFlagNames` slice order is ever rearranged, the sort would become load-bearing — keeping it is the safer choice. No change required; this is purely informational.

---

### IN-03: `CombatSystem.Rand` field is never used by any method in the reviewed files

**File:** `go/pkg/combat/combat.go:49`

**Issue:** The `Rand *rand.Rand` field is added to `CombatSystem` but none of the receiver methods (`OneHit`, `MultiHit`, `DoBackstab`, `DoKick`, `CheckDefenses`, etc.) reference it. All dice calls in those methods go through the package-level `randIntn` / `SetRand` path. The field exists as a test hook (documented in the comment), but because it is never read, installing it has no effect. If a test sets `cs.Rand` expecting deterministic results, it will get none — `SetRand` is still required.

This is potentially a confusion hazard: the `dice_test.go` test `TestCombatSystemRandField` only checks that the field exists and can be assigned, not that any method uses it.

**Fix:** Either wire the field into the dice calls (e.g., `OneHit` checks `c.Rand` before falling back to `randIntn`), or remove the field and rely solely on `SetRand`. If the field is intentionally a placeholder for a future per-instance RNG path, add a `// TODO: wire into OneHit/MultiHit` comment so the intent is clear:

```go
// Rand, when non-nil, will be used as the per-instance RNG source for
// receiver methods (OneHit, MultiHit, etc.) once wired in.
// TODO(phase-02): route OneHit/MultiHit dice calls through c.Rand.
// For now, use the package-level combat.SetRand instead.
Rand *rand.Rand
```

---

### IN-04: Duplicated helper block in `fixture.go` is not verified against the source it was copied from

**File:** `go/pkg/golden/fixture.go:400-415` (comment block)

**Issue:** The comment says the helpers were "copied verbatim from combat_sim_test.go lines 43-362 (at commit 6e0810e)" and "must remain byte-for-byte stable". There is no automated check that enforces this stability — if `combat_sim_test.go` is updated (e.g., during combat tuning), the golden fixture silently diverges from the simulation used to tune balance, defeating the parity guarantee.

**Fix:** Add a Go `TestFixtureHelperParity` test that re-runs a known scenario with both `makePlayer`/`makeMob` from this package and the equivalents from `combat` (via a shared internal testutil if created later), or at minimum add a CI step that warns when the two files diverge. For now, a comment pointing to the verification step in the phase checklist is sufficient.

---

_Reviewed: 2026-04-17_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
