---
phase: 01-golden-master-safety-net
verified: 2026-04-17T10:00:00Z
status: gaps_found
score: 4/5
overrides_applied: 0
gaps:
  - truth: "Fixture covers mob-template behavior samples (aggro, assist, immunities, special attacks) at representative levels"
    status: failed
    reason: "No dedicated mob-template section in entities.golden. Mobs appear only as passive targets for player attacks (via makeMob()). Aggro triggers, assist behavior, mob immunities/vulnerabilities, and special attack functions (pkg/ai/ specials) are not exercised or captured in the snapshot. ROADMAP SC #3 explicitly requires these samples."
    artifacts:
      - path: "go/pkg/golden/fixture.go"
        issue: "runSkillScenarios and runRaceWarriorCombos use makeMob() as a passive combat target only; no runMobTemplateSamples or equivalent function exists"
      - path: "go/pkg/golden/testdata/entities.golden"
        issue: "No mob-template section — snapshot only has RACE x WARRIOR, CLASS x HUMAN, SPELL EXECUTIONS, SKILL EXECUTIONS sections; no MOB TEMPLATES section"
    missing:
      - "A runMobTemplateSamples(buf) scenario runner that exercises at least: (a) a mob with known immunity flags (ImmFlags) captures those in the snapshot, (b) an aggro mob triggers its fight response in a simple combat loop, (c) a caster mob (spec_cast_mage equivalent) executes at least one spell cast captured in the buffer"
      - "A new === MOB TEMPLATES === section in testdata/entities.golden produced by the above runner"
      - "Note: pkg/ai/ aggro/assist functions require wiring pkg/ai.AISystem into the fixture; mob special functions can be invoked directly via the specXxx pattern from pkg/ai/specials.go"
---

# Phase 1: Golden-Master Safety Net — Verification Report

**Phase Goal:** Establish a deterministic golden-master test harness that captures current entity behavior (races, classes, spells, skills) as a committed snapshot, providing a parity gate for all subsequent migration phases.
**Verified:** 2026-04-17T10:00:00Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (from ROADMAP.md Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Golden-master fixture covers all 19 races x 14 classes across representative combat events (hit, damage, resist, immunity, vulnerability) | VERIFIED | entities.golden has 19 Race= lines and 14 Class= lines; each row captures HP, Str/Dex/Con, Hit%, damage, and Imm/Res/Vuln flags; `grep -c '^Race='` returns 19, `grep -c '^Class='` returns 14 |
| 2 | Fixture covers representative spell casts (damage, affect, healing) and skill executions (backstab, dodge, parry, kick) with deterministic seeded RNG | VERIFIED | entities.golden has 7 Spell= lines (acid blast, bless, cure light, fireball, heal, magic missile, sanctuary) and 4 skill lines (Backstab, Kick, Defense x2); `go test ./pkg/golden/ -run TestGolden -count=2` exits 0 confirming byte-identical output |
| 3 | Fixture covers mob-template behavior samples (aggro, assist, immunities, special attacks) at representative levels | FAILED | entities.golden has no mob-template section; mobs appear only as passive combat targets in other scenario sections; no aggro, assist, mob immunity, or special-attack coverage present |
| 4 | Running the fixture twice produces byte-identical output; parity gate runs in CI | VERIFIED | `go test ./pkg/golden/ -run TestGolden -count=2 -timeout 60s` exits 0; `go test ./...` passes cleanly across all 14 packages; TestGolden lives in pkg/golden (dedicated package, not combat_sim_test.go — see note below) |
| 5 | Any intentional change to entity behavior during later phases produces a visible, diffable fixture failure | VERIFIED | entities.golden is a committed byte-for-byte snapshot; `TestGolden` does `bytes.Equal(got, want)` against the file; any behavioral drift will produce a `t.Fatalf` with the full diff; confirmed by code inspection of golden_test.go:60-70 |

**Score: 4/5 truths verified**

Note on SC #4 wording: ROADMAP says "combat_sim_test.go integrates the fixture." In practice, TestGolden lives in the standalone `pkg/golden/` package per D-05 (avoiding the magic->combat->magic import cycle). The intent — a CI parity gate that `go test ./...` picks up — is fully satisfied. This is an alternative implementation that achieves the same outcome.

### Deferred Items

None — SC #3 failure is not addressed in any later milestone phase at the golden-fixture level.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `go/pkg/combat/dice.go` | Seed-injectable package-scope RNG (defaultRand + SetRand + randIntn) | VERIFIED | Contains `var defaultRand *rand.Rand`, `func SetRand(r *rand.Rand) func()`, `func randIntn(n int) int`; all four rollers route through randIntn; only 1 `rand.Intn` call in the file (the fallthrough inside randIntn) |
| `go/pkg/combat/combat.go` | CombatSystem.Rand field for receiver-scoped seeding | VERIFIED | Contains `Rand *rand.Rand` as last struct field; `"math/rand"` import present; NewCombatSystem() zero-value nil preserved |
| `go/pkg/combat/dice_test.go` | Determinism guard for SetRand seeded source | VERIFIED | Contains TestSetRandDeterministic, TestSetRandRestore, TestDefaultRandFallsThroughToGlobal, TestEdgeCasesPreserved, TestCombatSystemRandField, TestCombatSystemRandFieldTypeByReflection; all pass |
| `go/pkg/golden/doc.go` | Package-level doc with seed rationale and -update usage | VERIFIED | Contains `package golden`, seed rationale, D-03 coverage scope, do-not list |
| `go/pkg/golden/fixture.go` | Scenario builders (runRaceWarriorCombos, runClassHumanCombos, runSpellScenarios, runSkillScenarios, plus helpers) | VERIFIED (partial) | 675 lines; all four scenario runners present; makePlayer/makeMob helpers duplicated; mob coverage limited to passive combat target only |
| `go/pkg/golden/golden_test.go` | TestGolden entry with -update flag, seeded RNG, bytes.Equal diff | VERIFIED | Contains TestGolden, flag.Bool("update"), const goldenSeed = 42, combat.SetRand call, t.Cleanup(restore), testdata/entities.golden path |
| `go/pkg/golden/testdata/entities.golden` | Committed parity snapshot, min 50 lines | VERIFIED | 55 lines, 4465 bytes; committed in git at a7f4089; contains all required === sections except mob templates |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| golden_test.go:TestGolden | combat.SetRand | rand.NewSource(42) + t.Cleanup(restore) | VERIFIED | `combat.SetRand(rand.New(rand.NewSource(goldenSeed)))` at line 36; `t.Cleanup(restore)` at line 37; seed install precedes runFixture call |
| fixture.go:runFixture | combat.NewCombatSystem, magic.NewMagicSystem | real-API construction | VERIFIED | grep confirms both NewCombatSystem and NewMagicSystem called in fixture.go |
| fixture.go | types.RaceTable, types.ClassTable | iterate all 19 races and 14 classes | VERIFIED | `for raceIdx := 0; raceIdx < len(types.RaceTable)` and `for classIdx := 0; classIdx < len(types.ClassTable)` present |
| golden_test.go | testdata/entities.golden | os.ReadFile + bytes.Equal OR os.WriteFile on -update | VERIFIED | filepath.Join("testdata", "entities.golden") at line 42; ReadFile + bytes.Equal at lines 56-70; WriteFile behind *updateGolden at lines 46-53 |
| dice.go:Dice | defaultRand (package var) | randIntn helper | VERIFIED | `total += randIntn(size) + 1` in Dice loop |
| dice.go:NumberPercent | defaultRand (package var) | randIntn helper | VERIFIED | `return randIntn(100) + 1` |
| dice.go:NumberRange | defaultRand (package var) | randIntn helper | VERIFIED | `return low + randIntn(high-low+1)` |
| dice.go:NumberBits | defaultRand (package var) | randIntn helper | VERIFIED | `return randIntn(1 << bits)` |
| combat.go:CombatSystem | math/rand.Rand | Rand *rand.Rand field | VERIFIED | `Rand *rand.Rand` at line 49 of combat.go |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| fixture.go:runFixture | buf *bytes.Buffer | CombatSystem.OneHit, Spell.Func, CheckDefenses results | Yes — real combat API calls with seeded RNG produce deterministic non-zero output (e.g., Race=human HP=343, dHP=137 for acid blast) | FLOWING |
| entities.golden | snapshot bytes | runFixture output via os.WriteFile | Yes — 55 non-trivial lines with populated numeric values | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| TestGolden passes with committed snapshot | `go test ./pkg/golden/ -run TestGolden -count=2 -timeout 60s` | ok rotmud/pkg/golden 0.004s | PASS |
| Determinism: dice package under seed | `go test ./pkg/combat/ -run TestSetRandDeterministic -count=2 -timeout 30s` | ok rotmud/pkg/combat 0.003s | PASS |
| Full test suite | `go test ./... -timeout 180s` | all 14 packages ok | PASS |
| go vet clean | `go vet ./pkg/golden/... ./pkg/combat/...` | no output (clean) | PASS |
| Snapshot has 19 race lines | `grep -c '^Race=' pkg/golden/testdata/entities.golden` | 19 | PASS |
| Snapshot has 14 class lines | `grep -c '^Class=' pkg/golden/testdata/entities.golden` | 14 | PASS |
| Snapshot has 7 spell lines | `grep -c '^Spell=' pkg/golden/testdata/entities.golden` | 7 | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|---------|
| MIGRATE-06 | 01-01, 01-02, 01-03 | Golden-master test suite captures all entity behaviors (combat, spell, skill) before migration starts; used as CI parity gate throughout | PARTIAL | Races, classes, spells, and skills are captured and functional as a CI parity gate. Mob-template behavior (aggro, assist, immunities, special attacks) per SC #3 is not captured. Core parity gate is operational; mob coverage gap exists. |

### Anti-Patterns Found

No TODO/FIXME/placeholder markers found in any modified files. No empty return stubs. No hardcoded-empty-data paths. Code is fully wired against real APIs.

### Human Verification Required

No human verification items. All automated checks are complete.

### Gaps Summary

One gap blocks full MIGRATE-06 satisfaction:

**SC #3 — Mob-template behavior not captured.** ROADMAP.md Phase 1 success criterion #3 states the fixture must cover "mob-template behavior samples (aggro, assist, immunities, special attacks) at representative levels." The committed entities.golden has no mob-template section. Mobs appear only as passive combat targets for the race/class/skill scenarios.

This was a scoping decision during research (CONTEXT.md: "include a representative mob section if it fits cleanly, otherwise defer"; RESEARCH.md: "mob aggro/assist AI to a later phase") but the ROADMAP contract still lists it as a Phase 1 success criterion and no later phase explicitly schedules extending the golden fixture with mob behavior coverage.

The gap requires adding a `runMobTemplateSamples` scenario runner that captures: mob immunity/vulnerability flags at stat construction time, an aggro trigger test using a mob with ActAggressive set, and at least one special-function invocation via the pkg/ai spec path. This is a contained addition to fixture.go that does not require architectural changes.

---

_Verified: 2026-04-17T10:00:00Z_
_Verifier: Claude (gsd-verifier)_
