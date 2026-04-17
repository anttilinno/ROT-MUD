---
phase: 01-golden-master-safety-net
plan: 02
subsystem: golden/fixture
tags:
  - testing
  - golden-master
  - fixture
  - determinism
dependency_graph:
  requires:
    - "combat.SetRand (from Plan 01-01)"
    - "combat.CombatSystem{Output, SkillGetter, CheckDefenses, DoBackstab, DoKick, OneHit}"
    - "magic.NewMagicSystem / Registry.FindByName / Spell.Func"
    - "types.RaceTable (19 races) / types.ClassTable (14 classes)"
  provides:
    - "pkg/golden package: runFixture + four scenario runners + duplicated helpers"
    - "TestGolden driver with -update flag, seeded RNG install via combat.SetRand, t.Cleanup restore"
  affects:
    - "pkg/golden (new package)"
tech_stack:
  added: []
  patterns:
    - "Checked-in golden file + -update flag (Pattern 1)"
    - "bytes.Buffer Output capture via CombatSystem.Output / MagicSystem.Output callbacks (Pattern 2)"
    - "Package-scope seeded RNG via combat.SetRand + t.Cleanup(restore) (Pattern 3)"
    - "Scenario-function-per-axis output (Pattern 5)"
    - "Helper duplication (raceStatAtLevel, playerHP, etc.) decoupled from combat_sim_test.go"
key_files:
  created:
    - path: "go/pkg/golden/doc.go"
      purpose: "Package doc — seed rationale, -update usage, D-03 coverage scope, do-not list"
      lines: 62
    - path: "go/pkg/golden/fixture.go"
      purpose: "runFixture entry, four scenario runners (race x warrior, class x human, spells, skills), duplicated helpers, immFlagNames renderer"
      lines: 675
    - path: "go/pkg/golden/golden_test.go"
      purpose: "TestGolden driver with -update flag, seeded RNG, bytes.Equal diff vs testdata/entities.golden"
      lines: 71
  modified: []
decisions:
  - id: D-01-02-A
    summary: "Adjusted DoBackstab/DoKick call shape to match real (ch, victim *types.Character) signature"
    rationale: "Plan <interfaces> block documented these as (ch *types.Character, victimName string) — that is not the actual API in pkg/combat/skills.go (which takes *types.Character directly and returns SkillResult). Using the real signatures lets the code compile; the fixture still exercises the production skill-execution path. Tracked as Rule 3 (blocking) deviation."
  - id: D-01-02-B
    summary: "DefenseResult constants: DefenseNone/Dodged/Parried/Blocked (no DefenseHit/DefenseMissed)"
    rationale: "Plan <interfaces> listed DefenseHit and DefenseMissed constants. The real enum (combat/defense.go:11) is DefenseNone, DefenseDodged, DefenseParried, DefenseBlocked. Emit DefenseNone as the fallback 'hit' bucket — semantically, no defensive reaction fired means the normal THAC0 hit/miss pipeline runs. Reporting column renamed accordingly."
  - id: D-01-02-C
    summary: "Built local immFlagNames table instead of using a types.ImmFlagNames symbol"
    rationale: "Plan hinted types.ImmFlagNames might exist but allowed fallback. No such symbol is exported from pkg/types. Built a local slice-of-struct table (immFlagNames) covering all 21 ImmFlags bits. Preferred over a raw 0x%x dump because named flags produce more readable golden diffs when a race's immunity set changes."
  - id: D-01-02-D
    summary: "Duplicated the standard combat_sim_test.go helper set (10 helpers) verbatim"
    rationale: "Per Research Open Question #1 resolution — duplicate, don't extract. Copied raceStatAtLevel, playerHP, classEquipAC, weaponDice, weaponTypeForClass, makeWeapon, playerMana, makePlayer, mobHP, makeMob. Sim-specific extras (daggerDice, backstabMult, simWeaponDice, weaponSkillForClass, dodgeSkillForClass, parrySkillForClass, extraAttackSkillForClass, spellManaCost, castSpellDamage, casterMobHP, makeCasterMob, mobCastSpellDam) were NOT copied — they are not reachable from the fixture's call graph and omitting them keeps pkg/golden focused on the parity snapshot scope."
metrics:
  duration_seconds: 240
  duration_human: "~4 minutes"
  completed_at: "2026-04-17T09:00:00Z"
  tasks_total: 2
  tasks_completed: 2
  commits: 2
  files_changed: 3
  lines_added: 808
  lines_removed: 0
---

# Phase 01 Plan 02: Golden-Master Fixture Framework — Summary

Created the `go/pkg/golden/` package — scenario builders plus the `TestGolden`
driver — wired to the seeded RNG hook that Plan 01-01 landed. The fixture
compiles, vets cleanly, and fails in the expected "no snapshot yet" state.
Plan 03 will run it with `-update` to generate the first
`testdata/entities.golden`.

## What Shipped

### `go/pkg/golden/doc.go` (62 lines)

Package-level documentation covering:

- **What this package is:** test-only parity gate; imports real combat/magic/skills.
- **Location rationale (D-05):** lives at `pkg/golden/` rather than inside
  `pkg/combat/` to avoid the magic→combat→magic import cycle.
- **Seed:** pinned to 42 via `combat.SetRand(rand.New(rand.NewSource(42)))`;
  one-line seed change invalidates the whole snapshot.
- **Usage:** CI default-path command, regenerate command, and the
  determinism `-count=2` check.
- **Coverage (D-03):** 19 races x warrior Lv20, 14 classes x human Lv20,
  representative spells (damage/affect/heal), representative skills
  (backstab, kick, dodge, parry). Mob aggro/AI deferred.
- **Do not:** reuse `*types.Character`, emit map order, print
  timestamps/pointer addresses.

### `go/pkg/golden/fixture.go` (675 lines)

- **`runFixture(buf *bytes.Buffer)`** — entry point; emits a header block and
  four scenario sections in stable order.
- **`runRaceWarriorCombos`** — iterates all 19 races (slice order, safe),
  each paired with a Lv20 warrior. Per combo: 30-round `OneHit` loop vs
  `makeMob(20)`, reports `HP / Str / Dex / Con / Hit% / Dam / Imm / Res / Vuln`.
- **`runClassHumanCombos`** — iterates all 14 classes with human race at
  Lv20. Reports `THAC0_00 / THAC0_32 / HP / Mana / HitRoll / DamRoll /
  Hit% / Dam`.
- **`runSpellScenarios`** — seven alphabetically-ordered spells (acid
  blast, bless, cure light, fireball, heal, magic missile, sanctuary)
  invoked via `spell.Func(caster, level, victim)` to bypass
  `CheckDefenses` per Pitfall #4. Reports `success / dHP / dMana /
  victimHp transition`.
- **`runSkillScenarios`** — backstab (thief → mob), kick (warrior → mob),
  and two defense trials (warrior defending vs thief, thief defending vs
  warrior) using 100-iteration `CheckDefenses` sampling. Reports
  `dodged / parried / blocked / hit` counts.
- **`safePct`, `immFlagNames`, `formatImmBits`, `joinStrings`** — local
  helpers for deterministic flag rendering.
- **Duplicated helpers** (from `combat_sim_test.go` lines ~43-362):
  `raceStatAtLevel`, `playerHP`, `classEquipAC`, `weaponDice`,
  `weaponTypeForClass`, `makeWeapon`, `playerMana`, `makePlayer`,
  `mobHP`, `makeMob`. Copied verbatim per Research Open Question #1.

### `go/pkg/golden/golden_test.go` (71 lines)

- **`var updateGolden = flag.Bool("update", ...)`** — `-update` flag
  registration; default false.
- **`const goldenSeed = 42`** — pinned seed constant.
- **`TestGolden(t *testing.T)`** — single parity-gate entry:
  1. `restore := combat.SetRand(rand.New(rand.NewSource(goldenSeed)))`
  2. `t.Cleanup(restore)` — ensures seed does not leak to other tests
  3. `runFixture(&buf)` — capture into `bytes.Buffer`
  4. If `-update`: `os.MkdirAll(testdata, 0o755)` + `os.WriteFile(..., 0o644)`.
  5. Else: `os.ReadFile` → `bytes.Equal` → `t.Fatalf` with "run ...
     `-update` ..." guidance on mismatch or missing file.

## Verification Results

| Check | Command | Result |
|-------|---------|--------|
| Golden package compiles | `go build ./pkg/golden/...` (from `go/`) | ok |
| Golden package vets | `go vet ./pkg/golden/...` (from `go/`) | ok |
| Full module still builds | `go build ./...` (from `go/`) | ok |
| Existing combat tests still pass | `go test ./pkg/combat/ -count=1 -timeout 180s` | ok (22s) |
| TestGolden expected-fail on missing snapshot | `go test ./pkg/golden/ -run TestGolden -timeout 30s` | **FAIL (expected)** with message `read golden: open testdata/entities.golden: no such file or directory (run \`go test ./pkg/golden/ -run TestGolden -update\` to create)` |
| `-update` flag accepted without "flag provided but not defined" | `go test ./pkg/golden/ -run TestGolden -update=false -timeout 30s` | flag registered (failure is the same expected no-snapshot diff error, proving the flag parsed) |

**Grep acceptance (Task 1 + Task 2):** all 15 grep checks from the plan's
`<acceptance_criteria>` blocks pass. Verified inline during execution.

## Expected Failure Message (Pre-Plan 03)

Running `go test ./pkg/golden/ -run TestGolden` before Plan 03 generates
the snapshot:

```
--- FAIL: TestGolden (0.00s)
    golden_test.go:58: read golden: open testdata/entities.golden: no such file or directory (run `go test ./pkg/golden/ -run TestGolden -update` to create)
FAIL
FAIL    rotmud/pkg/golden    0.008s
FAIL
```

This is the **correct** state at the end of Plan 02. Plan 03 runs the
fixture with `-update`, commits `testdata/entities.golden`, and the
default-path run flips to PASS.

## Deviations from Plan

Three compile-driven deviations, all documented in decisions D-01-02-A
through C above. Summary:

1. **[Rule 3 — Blocking] `DoBackstab`/`DoKick` signature mismatch.** Plan's
   `<interfaces>` block listed `DoBackstab(ch *types.Character, victimName
   string)` and `DoKick(ch *types.Character, victimName string)`. The real
   signatures at HEAD are `DoBackstab(ch, victim *types.Character)
   SkillResult` and `DoKick(ch, victim *types.Character) SkillResult`
   (`pkg/combat/skills.go:20` and `:219`). Fixed by passing the victim
   pointer directly; explanatory comment left in the fixture above each
   call. Same real production code is exercised — only the call shape
   changed from name-based to pointer-based.

2. **[Rule 3 — Blocking] `DefenseHit`/`DefenseMissed` constants do not
   exist.** Plan's `<interfaces>` listed a five-value `DefenseResult` enum
   including `DefenseHit` and `DefenseMissed`. The real enum
   (`pkg/combat/defense.go:11`) is four-valued: `DefenseNone, DefenseDodged,
   DefenseParried, DefenseBlocked`. Fixed by treating `DefenseNone` as the
   "no defensive reaction fired" bucket (i.e., the normal THAC0 path would
   run) and reporting it under the `hit` column. Semantic intent preserved;
   same counts would be seen under either naming.

3. **[Rule 2 — Missing infra] `types.ImmFlagNames` does not exist.** Plan's
   `formatImmBits` draft assumed a `map[string]types.ImmFlags` symbol named
   `types.ImmFlagNames`. No such export in `pkg/types/flags.go`. Fixed by
   building a local slice-of-struct table (`immFlagNames`) listing all 21
   `ImmFlags` bits with stable names (summon, charm, magic, weapon, bash,
   pierce, slash, fire, cold, lightning, acid, poison, negative, holy,
   energy, mental, disease, drowning, light, sound, silver). Fallback
   integer-dump path declined — named flags produce better diffs when a
   race's immunity set shifts during migration.

No architectural decisions were required. No auth gates encountered. No
Rule 4 escalations.

## Known Stubs

None. The fixture code is fully wired against real `pkg/combat`,
`pkg/magic`, and `pkg/types` APIs. No placeholder arrays, TODOs, or empty
data paths. The only "stub" is that `testdata/entities.golden` does not
exist yet — but that is by design: Plan 03 generates it from this fixture
and commits it. Plan 02 explicitly stops at the failing-test state.

## Commits

| # | Hash     | Message |
|---|----------|---------|
| 1 | `7140dd6` | `feat(01-02): add pkg/golden skeleton with fixture scenarios` |
| 2 | `4095712` | `test(01-02): add TestGolden driver with -update flag and seeded RNG` |

Both commits pass `go vet ./pkg/golden/...` and `go build ./...`. The
second commit is the boundary at which `go test ./pkg/golden/ -run
TestGolden` starts producing the expected no-snapshot failure message.

## Downstream Impact (for Plan 03)

Plan 03 can now execute:

```bash
cd go && go test ./pkg/golden/ -run TestGolden -update -timeout 30s
```

This will:
1. Install seed 42 via `combat.SetRand`.
2. Run `runFixture(&buf)` — the four scenario sections above.
3. Create `go/pkg/golden/testdata/` if missing.
4. Write `testdata/entities.golden` containing the captured buffer
   (deterministic under seed 42).
5. Emit `t.Logf("golden updated: testdata/entities.golden (N bytes)")`.

After that, the default-path `go test ./pkg/golden/ -run TestGolden`
(no flags) should pass, and `go test ./pkg/golden/ -run TestGolden
-count=2` must produce byte-identical output across both runs — the
determinism gate for MIGRATE-06 success criterion #4.

## Self-Check: PASSED

- `go/pkg/golden/doc.go` exists and contains `package golden` + `Seed 42`. ✓
- `go/pkg/golden/fixture.go` exists and exposes `runFixture`, four scenario
  runners, and duplicated `makePlayer`/`makeMob` helpers. ✓
- `go/pkg/golden/golden_test.go` exists with `TestGolden`, `flag.Bool("update", ...)`,
  `const goldenSeed = 42`, `combat.SetRand(rand.New(rand.NewSource(goldenSeed)))`,
  `t.Cleanup(restore)`, and the `testdata/entities.golden` diff path. ✓
- Commits `7140dd6` and `4095712` present in `git log --oneline`. ✓
- `go vet ./pkg/golden/...` clean. ✓
- `go build ./...` clean. ✓
- `go test ./pkg/combat/ -count=1` passes (22s, includes combat_sim_test.go). ✓
- `go test ./pkg/golden/ -run TestGolden` fails with the expected
  "no such file or directory (run ... `-update` to create)" message. ✓
