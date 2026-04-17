# Phase 1: Golden-Master Safety Net - Context

**Gathered:** 2026-04-17
**Status:** Ready for planning

<domain>
## Phase Boundary

Capture current entity behavior (races, classes, skills, spells, mobs) in a deterministic test fixture before any migration starts. This fixture is the CI parity gate for every subsequent phase — any behavioral regression shows up as a visible, diffable failure.

No new game mechanics, no trait system work, no data file changes. This phase is purely observational: freeze what exists, make it verifiable.

</domain>

<decisions>
## Implementation Decisions

### Snapshot Format
- **D-01:** Checked-in `.golden` text file in `testdata/`. The test compares its output against the committed file. A `-update` flag (or `go test -run=TestGolden -update`) regenerates the file when behavior is intentionally changed. Regressions show as `git diff` line changes — exactly which race/class/spell behavior shifted and by how much.

### RNG Seeding
- **D-02:** Add a `Rand *rand.Rand` field to `CombatSystem`. When non-nil, combat rolls use this source instead of the global `math/rand`. The golden fixture injects a fixed-seed `rand.New(rand.NewSource(42))`. The existing `combat_sim_test.go` leaves the field nil (keeps using global rand as today). No changes to `Dice()` signature — combat system passes its `Rand` through internally.

### Coverage Scope
- **D-03:** Representative samples, not full matrix:
  - All 19 races × warrior class (captures race stat/trait/immunity/vulnerability differences)
  - All 14 classes × human race (captures class THAC0, HP gain, skill differences)
  - 33 combos total; fast enough for CI
- **D-04:** Real `pkg/magic` and `pkg/skills` API calls, not approximations. The fixture exercises actual `CastSpell()` and skill check paths so that spell/skill behavior changes during migration are caught. This requires the `pkg/golden/` package to import both magic and skills directly.

### Fixture Location
- **D-05:** Dedicated `go/pkg/golden/` package. Rationale: `pkg/magic` imports `pkg/combat`; a golden test inside `pkg/combat` that imports `pkg/magic` would create an import cycle. A standalone `pkg/golden/` package imports combat, magic, and skills without cycles. The `combat_sim_test.go` balance simulator stays untouched in `pkg/combat/`.

### Claude's Discretion
- How to structure the `.golden` file internally (one block per combo, or a table) — Claude decides based on what produces the clearest diffs
- Whether to cover mob behavior (aggro, assist, immunities) in Phase 1 or defer to a later pass — Claude can include a representative mob section if it fits cleanly, otherwise defer
- Exact fixture runner architecture (test helper struct vs. flat functions)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Core entity tables (what the fixture captures)
- `go/pkg/types/races.go` — all 19 race definitions (BaseStats, MaxStats, size, XP mult, bonus skills, immunities)
- `go/pkg/types/classes.go` — all 14 class definitions (THAC0, HPMin/HPMax, ManaGain, skill groups)

### Combat system (primary path under test)
- `go/pkg/combat/combat.go` — `CombatSystem` struct, `MultiHit`, callback fields
- `go/pkg/combat/hit.go` — `OneHit`, THAC0 calculation, hit/miss logic
- `go/pkg/combat/dice.go` — `Dice()`, `NumberPercent()` — the RNG functions that need seeding

### Magic and skills (real API calls per D-04)
- `go/pkg/magic/` — `CastSpell()` entry point, affect system, spell definitions
- `go/pkg/skills/` — `CheckImprove`, skill proficiency, defensive skill callbacks

### Existing simulation (context, not to be modified)
- `go/pkg/combat/combat_sim_test.go` — statistical balance simulator; fixture coexists with this, does not replace it

### Requirements
- `MIGRATE-06` in `.planning/REQUIREMENTS.md` — the single requirement this phase satisfies

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `CombatSystem` struct with injectable `Output`, `SkillGetter`, `RoomFinder`, `CharMover` callbacks — golden fixture follows the same injection pattern already used by `combat_sim_test.go`
- `types.NewCharacter()`, `types.NewNPC()`, `types.NewRoom()` — character/room builders used throughout tests; golden fixture uses these directly
- `makePlayer()`, `makeMob()` helpers in `combat_sim_test.go` — can be extracted to a shared `testutil` if needed, or duplicated in `pkg/golden/`

### Established Patterns
- Go golden-file pattern: write output to `bytes.Buffer`, compare against `testdata/*.golden`; regenerate with `-update` flag
- Test injection pattern: `CombatSystem.SkillGetter = func(...)` already used in `combat_sim_test.go` — same pattern for `Rand` injection
- `combat_sim_test.go` sets `cs.Output = func(_ *types.Character, _ string) {}` to suppress output — golden fixture will capture output instead

### Integration Points
- `pkg/golden/` imports `pkg/combat`, `pkg/magic`, `pkg/skills`, `pkg/types`
- `pkg/magic` requires a wired-up `MagicSystem` with room/character state — see `go/pkg/magic/magic_test.go` for setup patterns
- `pkg/skills` requires character state and a skill proficiency source — see `go/pkg/skills/` for test setup

</code_context>

<specifics>
## Specific Ideas

- The `.golden` file should read like a human-readable combat log, not just numbers — something a developer can scan and understand. For example: `Race=Vampire/Class=Warrior Lv20: hit=62% dam=avg34 outcome=win rounds=18`. Exact format is Claude's call, but clarity over compactness.
- `go test -update` pattern: standard in the Go ecosystem (`-run=TestGolden -update` regenerates `testdata/golden.txt`). Planner should include this in the test runner CLI notes.

</specifics>

<deferred>
## Deferred Ideas

- Full 19×14 matrix coverage — representative 33 combos is sufficient for Phase 1; full matrix can be added in Phase 8 (Race & Class Migration) when all combos are actively migrated
- Mob AI behavior coverage (aggro, assist triggers) — `pkg/ai/` is complex to wire up; defer unless it fits cleanly; mob stat/combat parity covered by makeMob()-style fixtures
- Statistical tolerance mode for flaky CI — not needed if RNG is seeded (D-02)

</deferred>

---

*Phase: 01-golden-master-safety-net*
*Context gathered: 2026-04-17*
