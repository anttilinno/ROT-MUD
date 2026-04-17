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
      change: "Added rotmud/pkg/ai import; updated runFixture header comment (+mob templates); appended runMobTemplateSamples call after runSkillScenarios; appended 4 new functions (~190 lines)"
    - path: "go/pkg/golden/testdata/entities.golden"
      change: "Regenerated — 55 lines → 60 lines, 4465 bytes → 4808 bytes; MOB TEMPLATES section appended"
decisions:
  - id: D-01-04-A
    summary: "castFired=false on MobCast line is accepted as deterministic under seed 42"
    rationale: "specCastMage NumberBits(2)==0 guard does not fire under seed 42; victim is nil; function returns before ctx.CastSpell. Byte-identical across runs. Plan explicitly calls this acceptable."
  - id: D-01-04-B
    summary: "Inline SpecialContext struct literal used instead of aiSpecContext helper"
    rationale: "Plan flagged &aiSpecContext(...) as a compile error and recommended inline literal as the preferred path."
  - id: D-01-04-C
    summary: "aiSys local variable name used instead of ai to avoid package-name collision"
    rationale: "Plan explicitly required this. Applied as instructed."
metrics:
  duration_human: "complete"
  completed_at: "2026-04-17"
  tasks_total: 4
  tasks_completed: 4
  commits: 3
  files_changed: 2
  lines_added: 197
  lines_removed: 2
---

# Phase 01 Plan 04: Mob-Template Golden Coverage — Summary

Extended `pkg/golden/fixture.go` with a `runMobTemplateSamples` runner exercising three mob-template
behaviors (immunity flags, aggro trigger, caster special dispatch) and regenerated `testdata/entities.golden`
to include a new `=== MOB TEMPLATES (seed=42) ===` section. Closes VERIFICATION.md gap SC #3. MIGRATE-06 fully satisfied.

## What Was Built

### Task 1: fixture.go extension (commit `b93ae06`)

- **Import added:** `"rotmud/pkg/ai"`
- **runFixture updated:** header comment updated to `+ mob templates`; `runMobTemplateSamples(buf)` appended after `runSkillScenarios(buf)`
- **New functions:** `runMobTemplateSamples`, `emitMobImmunityScenario`, `emitMobAggroScenario`, `emitMobCasterScenario`

### Task 2: entities.golden regeneration

55 lines (4465 bytes) → 60 lines (4808 bytes). New section:

```
=== MOB TEMPLATES (seed=42) ===
MobImm   Lv=20  name=Mob        Imm=[fire,silver] Res=[charm] Vuln=[cold]
MobAggro Lv=20  name=AggroMob   Act=aggressive aggroFired=true  victim=AggroTarget
MobCast  Lv=22  name=CasterMob  Special=spec_cast_mage fighting=true  spellAttempted=none            castFired=false victimHp=1000->1000
```

### Task 3: Human checkpoint — approved

### Task 4: Commit (commit `45f2bb3`)

`testdata/entities.golden` committed after human approval.

## Verification

| Check | Result |
|-------|--------|
| `go vet ./pkg/golden/...` | clean |
| `go build ./...` | clean |
| `go test ./pkg/golden/ -run TestGolden -count=2` | ok |
| `go test ./...` | all 13 packages ok |
| Two `-update` runs byte-identical | confirmed |
| 19 Race=, 14 Class=, 7 Spell= lines preserved | confirmed |
| MobImm/MobAggro/MobCast lines present | confirmed |
| No pointer leaks / format-string bugs | confirmed |

## Deviations from Plan

- **castFired=false on MobCast:** specCastMage NumberBits guard doesn't fire under seed 42 — deterministic, explicitly acceptable per plan
- **Inline SpecialContext:** used per plan's own recommendation over the aiSpecContext helper
- **aiSys variable:** renamed per plan's requirement to avoid package collision
- **Separate commits for fixture.go and entities.golden:** fixture.go committed at Task 1; snapshot committed after human approval at Task 4

## Commits

| Hash | Message |
|------|---------|
| `b93ae06` | `feat(01-04): extend fixture.go with runMobTemplateSamples runner` |
| `be52e4c` | `docs(01-04): add partial plan summary (awaiting Task 3 human gate)` |
| `45f2bb3` | `test(01-04): close SC #3 gap with mob-template golden coverage` |
