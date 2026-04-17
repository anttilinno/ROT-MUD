---
phase: 01-golden-master-safety-net
plan: 04
subsystem: testing
tags:
  - testing
  - golden-master
  - gap-closure
  - mob-templates
  - ai-specials
dependency_graph:
  requires:
    - phase: 01-01
      provides: seeded RNG hook (combat.SetRand) — determinism primitive
    - phase: 01-02
      provides: pkg/golden skeleton, fixture runners, TestGolden driver
    - phase: 01-03
      provides: committed baseline snapshot (55 lines, 4465 bytes)
  provides:
    - "runMobTemplateSamples scenario runner in pkg/golden/fixture.go"
    - "Extended parity snapshot at go/pkg/golden/testdata/entities.golden (60 lines, 4808 bytes)"
    - "Mob-template behavioral coverage: immunity flags, aggro trigger, caster special dispatch"
    - "VERIFICATION.md gap SC #3 closed"
    - "MIGRATE-06 fully satisfied"
  affects:
    - all-subsequent-trait-migration-phases
tech_stack:
  added:
    - "rotmud/pkg/ai imported by pkg/golden for the first time"
  patterns:
    - "Direct SpecialRegistry.Find + custom SpecialContext for isolated special-function invocation"
    - "Recording callback pattern: CastSpell stub captures spell name without side effects"
    - "Fresh AISystem per scenario — no cross-scenario state"
key_files:
  created:
    - path: ".planning/phases/01-golden-master-safety-net/01-04-SUMMARY.md"
      purpose: "Plan execution summary"
  modified:
    - path: "go/pkg/golden/fixture.go"
      change: "Added rotmud/pkg/ai import; updated runFixture header comment (+mob templates); appended runMobTemplateSamples call after runSkillScenarios; appended 4 new functions (runMobTemplateSamples + 3 emitters, ~190 lines)"
    - path: "go/pkg/golden/testdata/entities.golden"
      change: "Regenerated — 55 lines → 60 lines, 4465 bytes → 4808 bytes; header comment updated; MOB TEMPLATES section appended"
decisions:
  - id: D-01-04-A
    summary: "castFired=false on MobCast line is accepted as deterministic under seed 42"
    rationale: "specCastMage uses NumberBits(2)==0 guard in its victim-finding loop. Under seed 42, at the RNG state when emitMobCasterScenario executes, this rolls non-zero and victim is nil — the function returns false before reaching ctx.CastSpell. The snapshot is byte-identical across repeated runs (two consecutive -update runs produce the same file). The plan's how-to-verify section explicitly states this outcome is acceptable: 'If castFired=false on MobCast, the special ran but ctx.CastSpell was not reached — acceptable... the snapshot is still valid — the line is deterministic under seed 42.' The MobCast line is present and deterministic, which satisfies the parity-gate intent."
  - id: D-01-04-B
    summary: "Inline SpecialContext struct literal used instead of aiSpecContext helper"
    rationale: "Plan noted &aiSpecContext(...) is a compile error and recommended the inline struct literal as the preferred path. Inline literal is cleaner and eliminates an extra helper function."
  - id: D-01-04-C
    summary: "aiSys local variable name used instead of ai to avoid package-name collision"
    rationale: "Plan explicitly flagged this naming conflict and required aiSys. Applied as instructed."
metrics:
  duration_seconds: null
  duration_human: "in progress (awaiting human checkpoint approval)"
  completed_at: null
  tasks_total: 4
  tasks_completed: 2
  commits: 1
  files_changed: 1
  lines_added: 190
  lines_removed: 1
---

# Phase 01 Plan 04: Mob-Template Golden Coverage — Summary (partial)

**Awaiting Task 3 human checkpoint approval before Task 4 commit.**

Extended `pkg/golden/fixture.go` with a `runMobTemplateSamples` runner that exercises three
mob-template behaviors (immunity flags, aggro trigger, caster special dispatch) and regenerated
`testdata/entities.golden` to include a new `=== MOB TEMPLATES (seed=42) ===` section.

## What Was Built

### Task 1: fixture.go extension

Added to `go/pkg/golden/fixture.go`:

- **Import added:** `"rotmud/pkg/ai"` (line ~5, third-party group)
- **runFixture updated:** header comment line 3 changed from `+ spells + skills` to `+ spells + skills + mob templates`; `runMobTemplateSamples(buf)` call appended after `runSkillScenarios(buf)` with preceding blank line
- **Lines added:** ~190 (4 new functions appended after `makeMob`)

New functions:
- `runMobTemplateSamples(buf *bytes.Buffer)` — banner + 3 emitter calls
- `emitMobImmunityScenario(buf *bytes.Buffer, level int)` — sets ImmFire|ImmSilver on mob.Imm, ImmCharm on mob.Res, ImmCold on mob.Vuln, renders via formatImmBits
- `emitMobAggroScenario(buf *bytes.Buffer, level int)` — creates aggressive mob + player in shared room, wires ai.AISystem with recording StartCombat, invokes ProcessMobile
- `emitMobCasterScenario(buf *bytes.Buffer, level int)` — resolves spec_cast_mage via Registry.Find, invokes with recording SpecialContext.CastSpell

### Task 2: entities.golden regeneration

File extended from 55 lines (4465 bytes) to 60 lines (4808 bytes).

New section verbatim:
```
=== MOB TEMPLATES (seed=42) ===
MobImm   Lv=20  name=Mob        Imm=[fire,silver] Res=[charm] Vuln=[cold]
MobAggro Lv=20  name=AggroMob   Act=aggressive aggroFired=true  victim=AggroTarget
MobCast  Lv=22  name=CasterMob  Special=spec_cast_mage fighting=true  spellAttempted=none            castFired=false victimHp=1000->1000
```

Header line 3 change:
```
- # Coverage: 19 races x warrior Lv20 + 14 classes x human Lv20 + spells + skills
+ # Coverage: 19 races x warrior Lv20 + 14 classes x human Lv20 + spells + skills + mob templates
```

## Verification Results (Tasks 1-2)

| Check | Result |
|-------|--------|
| `go vet ./pkg/golden/...` | clean |
| `go build ./...` | clean |
| `go test ./pkg/golden/ -run TestGolden -count=2` | ok |
| `go test ./...` | all 13 packages ok |
| Two consecutive `-update` runs byte-identical | confirmed (diff /tmp/golden_run{1,2}.txt empty) |
| No pointer addresses in snapshot | confirmed |
| No format-string bugs in snapshot | confirmed |
| REGISTRY_MISSING sentinel absent | confirmed |
| NOT_FOUND sentinel absent | confirmed |
| 19 Race= lines | confirmed |
| 14 Class= lines | confirmed |
| 7 Spell= lines | confirmed |
| MobImm line present | confirmed |
| MobAggro line present, aggroFired=true | confirmed |
| MobCast line present, deterministic | confirmed (castFired=false — see D-01-04-A) |

## Known Notes for Human Reviewer

**MobCast castFired=false:** Under seed 42, the `specCastMage` victim-finding loop's `NumberBits(2)==0` guard does not fire, so the function returns early before reaching `ctx.CastSpell`. This is deterministic — both runs produce the same output. The plan's `how-to-verify` section explicitly calls this acceptable. The parity-gate intent is preserved: any future change to the specCastMage dispatch path that makes it fire under seed 42 will produce a visible diff.

**aggroFired=true, victim=AggroTarget:** The aggro branch fired correctly.

**ImmFire, ImmSilver on MobImm:** Rendered as `[fire,silver]` in sorted order via formatImmBits — correct.

## Task 4 (Pending Human Approval)

Task 4 commits both `fixture.go` and the regenerated `entities.golden` in a single atomic commit. This task is gated on human approval of Task 3.

## Deviations from Plan

**1. [Rule 1 - Note] castFired=false on MobCast**
- **Found during:** Task 2 (snapshot inspection)
- **Issue:** specCastMage's NumberBits(2)==0 guard does not fire under seed 42 at the RNG state during emitMobCasterScenario execution; victim is nil; function returns false before CastSpell
- **Fix:** None applied — plan explicitly designates this as acceptable outcome; snapshot is deterministic
- **Files modified:** none
- **Commit:** n/a

**2. Inline SpecialContext used (per plan's own recommendation)**
- Changed `&aiSpecContext(...)` approach to inline `&ai.SpecialContext{...}` literal as instructed in the plan's action notes

**3. aiSys variable name used (per plan's own requirement)**
- Local variable named `aiSys` to avoid collision with imported package `ai`

## Phase 1 Closing Notes

After Task 4 approval and commit:
- Phase 1 complete; 01-golden-master-safety-net shipped with gap closure
- MIGRATE-06 fully satisfied; CI parity gate live with mob-template coverage
- The committed `entities.golden` is the frozen behavioral baseline for all Phase 2+ migration work

## Task 1 Commit

| # | Hash | Message |
|---|------|---------|
| 1 | `b93ae06` | `feat(01-04): extend fixture.go with runMobTemplateSamples runner` |

Task 2 not yet committed (awaiting Task 3 human gate per plan design).
Task 4 commit hash: TBD after human approval.

## Known Stubs

None. All scenario runners are fully wired against real APIs. The `castFired=false` outcome is a deterministic property of the seeded RNG, not a stub or placeholder.
