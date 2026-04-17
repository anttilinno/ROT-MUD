---
phase: 01-golden-master-safety-net
reviewed: 2026-04-17T00:00:00Z
depth: standard
files_reviewed: 7
files_reviewed_list:
  - go/pkg/combat/combat.go
  - go/pkg/combat/dice.go
  - go/pkg/combat/dice_test.go
  - go/pkg/golden/doc.go
  - go/pkg/golden/fixture.go
  - go/pkg/golden/golden_test.go
  - go/pkg/golden/testdata/entities.golden
findings:
  critical: 0
  warning: 4
  info: 3
  total: 7
status: issues_found
---

# Phase 01: Code Review Report

**Reviewed:** 2026-04-17T00:00:00Z
**Depth:** standard
**Files Reviewed:** 7
**Status:** issues_found

## Summary

This phase establishes the golden-master parity gate for the ROT-MUD trait-migration project. The reviewed files cover the RNG abstraction (`dice.go`), a snapshot of the combat system's public surface (`combat.go`), the golden test infrastructure (`golden_test.go`), and the fixture that generates the snapshot (`fixture.go`).

The overall design is sound. `SetRand` / `randIntn` correctly routes all four rollers, and the golden-test architecture (seed-pinned, diff-on-mismatch, `-update` regeneration path) is a solid parity gate. No security vulnerabilities or data-loss risks were found.

Two correctness concerns stand out. First, `CheckImmune` in `combat.go` assigns the **same** flag constant to `immFlag`, `resFlag`, and `vulnFlag` for every damage type, making resistance and vulnerability lookups silently wrong. Second, the `MobCast` golden scenario records `castFired=false` because `specCastMage`'s probabilistic victim loop never fires under seed 42 — the committed snapshot encodes a no-op for that scenario rather than exercising the `CastSpell` dispatch path it was meant to gate.

---

## Warnings

### WR-01: `CheckImmune` assigns the same flag constant to all three of `immFlag`, `resFlag`, `vulnFlag`

**File:** `go/pkg/combat/combat.go:332-352`

**Issue:** Every `case` in the `switch damType` block assigns the same `ImmFlags` constant to all three variables:

```go
case types.DamFire:
    immFlag, resFlag, vulnFlag = types.ImmFire, types.ImmFire, types.ImmFire
```

The three variables exist to let the function check *different* bits on `Imm`, `Res`, and `Vuln`. With identical constants, `victim.Res.Has(resFlag)` checks the immunity bit, not the resistance bit. A character flagged with `ResFire` only (not `ImmFire`) will never be detected as resistant to fire, and similarly for vulnerability. The check silently collapses to immune-or-normal for every damage type.

The golden snapshot shows `Imm=- Res=- Vuln=-` for all 19 races (none have flags set), so the defect does not corrupt the current snapshot, but once races or mobs with resistance/vulnerability flags are added, the snapshot will encode the wrong behavior and serve as a false parity gate.

**Fix:** Use distinct constants per check. If the `types` package exposes `ResFire`, `VulnFire`, etc.:

```go
case types.DamFire:
    immFlag  = types.ImmFire
    resFlag  = types.ResFire
    vulnFlag = types.VulnFire
```

If `types` intentionally collapses all three into `ImmFlags` (sharing constants across `Imm`, `Res`, and `Vuln`), add a prominent comment explaining that design so readers know the assignment is intentional and not a copy-paste error.

---

### WR-02: MobCast golden scenario is a no-op — `specCastMage` never fires under seed 42

**File:** `go/pkg/golden/fixture.go:787-863`

**Issue:** `specCastMage` (in `go/pkg/ai/specials.go:329`) selects a victim by iterating `ch.InRoom.People` and accepting only when `combat.NumberBits(2) == 0` (1-in-4 chance per candidate). Under seed 42 the single victim in the room is never selected, so `victim == nil` and the function returns `false` without reaching `ctx.CastSpell`. The committed snapshot records:

```
MobCast  Lv=22  name=CasterMob  Special=spec_cast_mage fighting=true  spellAttempted=none  castFired=false victimHp=1000->1000
```

This line proves the registry lookup works (`specFn != nil`), but provides zero coverage of the `specCastMage → CastSpell` dispatch path that the scenario was designed to watch. Any future regression in that path will not be caught by the golden gate.

**Fix:** Pre-seed a deterministic inner RNG inside `emitMobCasterScenario` that produces `NumberBits(2) == 0` on the first call, then restore the outer seed. Alternatively, add a second victim candidate to the room to give the probabilistic check multiple chances:

```go
// Option A: inner seed (find a seed where first NumberBits(2) == 0)
restoreInner := combat.SetRand(rand.New(rand.NewSource(seedThatFiresImmediately)))
specFn(mob, ctx)
restoreInner()

// Option B: add a second candidate so the loop is more likely to find one
victim2 := makePlayer(types.ClassWarrior, types.RaceHuman, level)
victim2.Name = "CasterTarget2"
victim2.Position = types.PosFighting
victim2.Fighting = mob
room.AddPerson(victim2)
```

After choosing a fix, regenerate the snapshot with `-update` and verify `castFired=true` appears.

---

### WR-03: `TestSetRandRestore` hard-asserts `defaultRand == nil` at test start — fragile against test ordering and `-count=2`

**File:** `go/pkg/combat/dice_test.go:40-43`

**Issue:**

```go
if defaultRand != nil {
    t.Fatalf("expected defaultRand nil at test start, got %v", defaultRand)
}
```

`defaultRand` is a package-level variable. If any other test in the binary installs `SetRand` and panics before its cleanup runs, `defaultRand` will be non-nil when this test executes, causing a misleading fatal. Running with `go test -count=2` in a future CI step would also hit this if the restore closure is not idempotent across runs.

**Fix:** Snapshot the value at entry and always restore rather than asserting nil:

```go
func TestSetRandRestore(t *testing.T) {
    prev := defaultRand
    t.Cleanup(func() { defaultRand = prev })

    restore := SetRand(rand.New(rand.NewSource(1)))
    if defaultRand == nil {
        t.Fatal("expected defaultRand set after SetRand")
    }
    restore()
    if defaultRand != nil {
        t.Fatalf("expected defaultRand nil after restore, got %v", defaultRand)
    }
}
```

---

### WR-04: `CombatSystem.Rand` field is never read by any receiver method — installing it has no effect

**File:** `go/pkg/combat/combat.go:49`

**Issue:** The `Rand *rand.Rand` field exists on `CombatSystem` but none of the receiver methods (`OneHit`, `MultiHit`, `DoBackstab`, `DoKick`, `CheckDefenses`) reference it. All dice calls go through the package-level `randIntn` / `SetRand` path. `TestCombatSystemRandField` only checks that the field exists and can be assigned, not that any method uses it. A caller that sets `cs.Rand` expecting per-instance determinism gets none — `combat.SetRand` is still required.

This is a confusion hazard: the field signals a test hook that does not work.

**Fix:** Either wire the field into dice calls in receiver methods, remove it and rely solely on `SetRand`, or add a `// TODO` making the placeholder status explicit:

```go
// Rand, when non-nil, will be used as the per-instance RNG source for
// receiver methods (OneHit, MultiHit, etc.) once wired in.
// TODO(phase-02): route OneHit/MultiHit dice calls through c.Rand.
// Until then, use the package-level combat.SetRand for deterministic tests.
Rand *rand.Rand
```

---

## Info

### IN-01: `joinStrings` reimplements `strings.Join`

**File:** `go/pkg/golden/fixture.go:392-401`

**Issue:** `joinStrings` is a hand-rolled string joiner called once from `formatImmBits`. `strings.Join` does the same thing.

**Fix:** Import `"strings"` and replace the call site:

```go
return "[" + strings.Join(names, ",") + "]"
```

Then remove the `joinStrings` function entirely (11 lines).

---

### IN-02: `sort.Strings(names)` in `formatImmBits` is redundant — order is already stable

**File:** `go/pkg/golden/fixture.go:374-385`

**Issue:** `formatImmBits` builds `names` by iterating `immFlagNames` (a `[]struct` in definition order), so `names` is already in a deterministic, stable order without the `sort.Strings` call. The sort is harmless but allocates unnecessarily for every invocation.

**Fix:** Either remove `sort.Strings(names)` and update the doc comment to "definition-ordered, comma-joined", or keep it as a defensive measure and add a comment explaining why (to guard against `immFlagNames` being reordered). If kept, no code change is needed — this is informational.

---

### IN-03: Global `mayorState` in `specials.go` is mutable package-level state, unsafe for parallel tests

**File:** `go/pkg/ai/specials.go:775-779`

**Issue:** `mayorState` is a module-level `*MayorState` pointer mutated by `specMayor`. Any test exercising `specMayor` (or `ProcessMobile` on a mayor mob) will mutate `mayorState.Pos` and `mayorState.Moving`, leaving dirty state for subsequent tests. The golden fixture does not exercise `specMayor` today, but if it or any future test does, parallel execution will produce a data race.

**Fix:** At minimum add a `// not goroutine-safe; do not call specMayor from parallel tests` comment next to the variable. For a production-quality fix, pass `MayorState` as a dependency through the `SpecialContext` or as a per-`AISystem` field so tests can supply a fresh instance.

---

_Reviewed: 2026-04-17T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
