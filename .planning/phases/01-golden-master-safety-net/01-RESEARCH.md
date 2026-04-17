# Phase 1: Golden-Master Safety Net - Research

**Researched:** 2026-04-17
**Domain:** Go golden-file testing; deterministic RNG for ROM-derived combat/magic/skills
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01 — Snapshot Format:** Checked-in `.golden` text file in `testdata/`. The test compares its output against the committed file. A `-update` flag (or `go test -run=TestGolden -update`) regenerates the file when behavior is intentionally changed. Regressions show as `git diff` line changes — exactly which race/class/spell behavior shifted and by how much.
- **D-02 — RNG Seeding:** Add a `Rand *rand.Rand` field to `CombatSystem`. When non-nil, combat rolls use this source instead of the global `math/rand`. The golden fixture injects a fixed-seed `rand.New(rand.NewSource(42))`. The existing `combat_sim_test.go` leaves the field nil (keeps using global rand as today). No changes to `Dice()` signature — combat system passes its `Rand` through internally.
- **D-03 — Coverage Scope:** Representative samples, not full matrix:
  - All 19 races × warrior class (captures race stat/trait/immunity/vulnerability differences)
  - All 14 classes × human race (captures class THAC0, HP gain, skill differences)
  - 33 combos total; fast enough for CI
- **D-04 — Real APIs:** Real `pkg/magic` and `pkg/skills` API calls, not approximations. The fixture exercises actual `CastSpell()` and skill check paths so that spell/skill behavior changes during migration are caught. Requires the `pkg/golden/` package to import both magic and skills directly.
- **D-05 — Fixture Location:** Dedicated `go/pkg/golden/` package. Rationale: `pkg/magic` imports `pkg/combat`; a golden test inside `pkg/combat` that imports `pkg/magic` would create an import cycle. A standalone `pkg/golden/` package imports combat, magic, and skills without cycles. The `combat_sim_test.go` balance simulator stays untouched in `pkg/combat/`.

### Claude's Discretion

- How to structure the `.golden` file internally (one block per combo, or a table) — clarity over compactness
- Whether to cover mob behavior (aggro, assist, immunities) in Phase 1 or defer to a later pass — include a representative mob section if it fits cleanly, otherwise defer
- Exact fixture runner architecture (test helper struct vs. flat functions)

### Deferred Ideas (OUT OF SCOPE)

- Full 19×14 matrix coverage — defer to Phase 8 (Race & Class Migration)
- Mob AI behavior coverage (aggro, assist triggers) — `pkg/ai/` wiring complexity; defer unless it fits cleanly
- Statistical tolerance mode for flaky CI — not needed if RNG is seeded (D-02)
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| MIGRATE-06 | Golden-master test suite captures all entity behaviors (combat, spell, skill) before migration starts; used as CI parity gate throughout | This research specifies: `pkg/golden/` location (D-05), `.golden` file format and `-update` pattern (D-01), seeded-RNG architecture covering the 80+ RNG call sites across combat/magic/skills/ai (D-02 + extension), API exercise points for real `Cast` and defensive-skill paths (D-04), and a 33-combo coverage matrix (D-03). Sections below map each sub-requirement to files and APIs. |
</phase_requirements>

## Summary

This phase builds a **golden-master fixture in a new `go/pkg/golden/` package** that captures the current runtime behavior of races, classes, skills, and spells into a checked-in `testdata/*.golden` text file. The fixture is deterministic (fixed seed), exercises real `pkg/combat`, `pkg/magic`, and `pkg/skills` APIs (not reimplementations), and runs in CI as the migration parity gate.

The critical architectural finding is that **ROM's RNG is package-global**. All of `combat.Dice`, `combat.NumberPercent`, `combat.NumberRange`, and `combat.NumberBits` call `math/rand` directly, and they are invoked from **80+ call sites across `pkg/combat/`, `pkg/magic/`, `pkg/skills/`, and `pkg/ai/`** [VERIFIED: `grep` count of the four function names]. D-02 (adding a `Rand` field on `CombatSystem`) only fully achieves determinism if the package-level functions in `pkg/combat/dice.go` route through a shared source — otherwise any spell that calls `combat.Dice(...)` will still hit the global RNG. The planner must treat "seed the RNG" as a sub-task that refactors `dice.go` to consult an injectable source, not just a `CombatSystem` field.

Go's `testing` package has native golden-file support via the convention `testdata/` (ignored by build tools) + a custom `-update` flag parsed at test init. `testing.T` exposes `t.TempDir()` for scratch work; the standard pattern is `os.WriteFile` on `-update`, `os.ReadFile` + diff assertion otherwise [VERIFIED: Go testing docs]. No external dependency is needed — `github.com/stretchr/testify` is already available for diff prettification but is optional.

**Primary recommendation:** Create `go/pkg/golden/` containing `golden_test.go` (driver), `fixture.go` (scenario builders), and `testdata/entities.golden`. Refactor `pkg/combat/dice.go` to introduce an injectable `rand.Source`-style variable (`defaultRand`) that `Dice`/`NumberPercent`/`NumberRange`/`NumberBits` all consult — with a package-level `SetRand(*rand.Rand)` hook the fixture calls at test start and restores in `t.Cleanup`. This makes D-02 work for the whole RNG surface, not just combat's own calls.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Scenario construction (build character, equip weapon, place in room) | `pkg/golden/` | `pkg/types` (NewCharacter, NewRoom, NewNPC, NewObject) | The fixture is a test-only consumer; builders already exist and should be reused, not duplicated. |
| Deterministic RNG injection | `pkg/combat/dice.go` (refactor) | `pkg/golden/` (caller) | RNG is called from `combat`, `magic`, `skills`, `ai` — the seed must live in the lowest common package (combat). |
| Combat execution under test | `pkg/combat` (MultiHit, OneHit, Damage, defense) | `pkg/golden/` (asserts) | Fixture calls real `CombatSystem` methods; no reimplementation. |
| Spell casting under test | `pkg/magic` (MagicSystem.Cast and direct spellXxx calls) | `pkg/golden/` | Cast mana/proficiency path is required for parity; direct `spellFunc(...)` calls also valid where side-effect output is the target. |
| Skill execution under test | `pkg/combat` (DoBackstab/DoKick/DoBash/DoAssassinate) + `pkg/skills` (CheckImprove) | `pkg/golden/` | Defensive skills (dodge/parry/shield block) run inside `combat.CheckDefenses`; offensive skills have explicit Do* entry points. |
| Output capture (combat/spell messages) | `pkg/golden/` (via `CombatSystem.Output` and `MagicSystem.Output` callback injection) | — | Identical pattern to `combat_sim_test.go`; a `bytes.Buffer`-backed callback captures the textual log. |
| Snapshot read/write/diff | `pkg/golden/golden_test.go` | `testing` stdlib | No extra libs; `os.ReadFile` + `os.WriteFile` + `flag`-based `-update`. |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go `testing` | 1.23+ (project uses 1.23.12) | Test runner, `t.Fatal`, `t.Cleanup` | Native; golden-file pattern is idiomatic here [VERIFIED: `go.mod` + `go version`]. |
| Go `math/rand` v1 | stdlib | `rand.New(rand.NewSource(seed))` for deterministic RNG | The existing `pkg/combat/dice.go` already imports `math/rand`; staying on v1 avoids a migration [VERIFIED: `combat/dice.go:4`]. |
| Go `flag` | stdlib | `-update` flag for regenerating golden files | Standard Go golden-file idiom [CITED: pkg.go.dev testing/fstest and numerous stdlib examples]. |
| Go `os` / `bytes` | stdlib | Read/write `testdata/*.golden`; buffer output | `os.ReadFile`, `os.WriteFile`, `bytes.Buffer` for capturing `CombatSystem.Output` writes. |
| Go `diff` (in-test) | stdlib `strings` + manual | Line-by-line diff on mismatch | Plain strings diff suffices; if nicer output is wanted, `testify/assert.Equal` renders a diff. |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/stretchr/testify` | v1.11.1 (already in go.sum indirectly via transitive or `go get`) | Cleaner assertion diffs if desired | Optional — only if a raw string compare produces hard-to-read failures. Note: `go.mod` does not currently list testify as a direct dependency [VERIFIED: `go.mod`]. Adding it is one line. |
| `github.com/sebdah/goldie/v2` | n/a | Third-party helper for golden-file tests | **NOT recommended** — D-01 specifies the manual `-update` flag pattern, and the stdlib idiom is short enough that an extra dependency is not worth it. |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `math/rand` v1 `rand.New(rand.NewSource(42))` | `math/rand/v2` `rand.New(rand.NewPCG(1, 2))` | v2 is newer (Go 1.22+) and has a better statistical profile, but `pkg/combat/dice.go` already uses v1 [VERIFIED: `combat/dice.go`]. Migrating the whole package in this phase is out of scope; the sim tests depend on v1 global state. Stay on v1 for parity and lowest-friction refactor. [CITED: pkg.go.dev/math/rand/v2] |
| Checked-in golden file (D-01) | Generate fixture at test start, compare against a checksum | Loses the "diff the file in a PR" benefit; the whole point of a golden file is the diff is human-readable. D-01 explicitly rules this out. |
| `pkg/golden/` location (D-05) | Put fixture inside `pkg/combat/` with build-tag exclusion | Build tags complicate CI; D-05 analysis (magic imports combat → import cycle) is correct. Confirmed: `pkg/magic/system.go:6` imports `rotmud/pkg/combat` [VERIFIED: grep]. |

**Installation:**

No new Go dependencies are strictly required. If optional `testify/assert` is desired:
```bash
cd /home/antti/Repos/Misc/ROT-MUD/go && go get github.com/stretchr/testify/assert@v1.11.1
```

**Version verification:** All tools are stdlib. Project Go version verified: `go version go1.23.12 linux/amd64` [VERIFIED: `go version` output 2026-04-17].

## Architecture Patterns

### System Architecture Diagram

```
                   golden_test.go (TestGolden)
                           │
              ┌────────────┴────────────┐
              │                          │
          -update?                   default
              │                          │
   runFixture→writeFile         runFixture→compare
              │                          │
              └────────────┬─────────────┘
                           ▼
                    runFixture(seed=42)
                           │
     ┌─────────────────────┼─────────────────────┐
     │                     │                     │
     ▼                     ▼                     ▼
 pkg/golden/          pkg/golden/           pkg/golden/
 fixture.go           fixture.go            fixture.go
 (races×warrior)      (classes×human)       (spells+skills)
     │                     │                     │
     └──────────┬──────────┴──────────┬──────────┘
                ▼                     ▼
       pkg/combat (real)      pkg/magic (real)
       MultiHit, Damage,      DefaultSpells(),
       CheckDefenses,         Cast() or spellXxx()
       DoBackstab/Bash/Kick,  AddAffect, IsAffectedBy
       CheckImmune                       │
                ▼                        ▼
          pkg/combat/dice.go ◄── shared seeded RNG ──► pkg/combat/dice.go
                                        ▲
                                        │
                                pkg/skills (real)
                                CheckImprove,
                                GetSkillByIndex
                                        │
                                        ▼
                              bytes.Buffer (Output capture)
                                        │
                                        ▼
                            Rendered log → compared/written
```

Data flow: the test reads `-update`; the fixture runs 33 combos + spell + skill scenarios against real subsystems while capturing their output into a buffer; the buffer becomes `testdata/entities.golden` on update, or is diffed against it otherwise.

### Recommended Project Structure

```
go/pkg/golden/
├── doc.go                  # Package documentation
├── fixture.go              # Scenario builders (runRaceCombo, runClassCombo, runSpell, runSkill)
├── golden_test.go          # TestGolden entry point with -update flag
├── rng.go                  # (optional) Helpers to install/restore seeded RNG
└── testdata/
    └── entities.golden     # The checked-in snapshot
```

Related change in `go/pkg/combat/`:
```
go/pkg/combat/
├── dice.go                 # REFACTOR: add package-level `defaultRand *rand.Rand`; SetRand()/ResetRand()
└── combat.go               # ADD: `Rand *rand.Rand` field on CombatSystem (D-02) - routes to dice.go setter
```

### Pattern 1: Checked-in Golden File with `-update` Flag
**What:** Standard Go golden-file idiom. A `flag.Bool("update", false, ...)` parsed via `init()` or inside the test controls whether the file is (re)written or diffed.
**When to use:** Any test whose expected output is large enough that inlining it as a string literal is unreadable (50+ lines). This fixture will be hundreds of lines.
**Example:**
```go
// Source: stdlib idiom (see pkg.go.dev/testing, and e.g. go/src/cmd/go/internal/* patterns)
var update = flag.Bool("update", false, "update golden files")

func TestGolden(t *testing.T) {
    got := runFixture()
    path := filepath.Join("testdata", "entities.golden")

    if *update {
        if err := os.WriteFile(path, got, 0o644); err != nil {
            t.Fatal(err)
        }
        return
    }

    want, err := os.ReadFile(path)
    if err != nil {
        t.Fatalf("read golden: %v (run with -update to create)", err)
    }
    if !bytes.Equal(got, want) {
        t.Fatalf("golden mismatch:\n--- want\n%s\n--- got\n%s", want, got)
    }
}
```

### Pattern 2: Callback-Injected Output Capture
**What:** `CombatSystem.Output` and `MagicSystem.Output` are `func(ch *types.Character, msg string)` callbacks. Wire them to a `bytes.Buffer` to capture the textual log deterministically.
**When to use:** Whenever narrative output contributes to the snapshot (combat messages, "You parry ...", spell effect lines). Reuses the existing pattern from `combat_sim_test.go:700` which sets `cs.Output = func(_ *types.Character, _ string) {}` for suppression [VERIFIED: `combat/combat_sim_test.go:700`].
**Example:**
```go
// Source: established pattern in pkg/combat/combat_sim_test.go:700
var buf bytes.Buffer
cs := combat.NewCombatSystem()
cs.Output = func(ch *types.Character, msg string) {
    fmt.Fprintf(&buf, "[%s] %s", ch.Name, msg)
}
// ... run combat ...
// buf.String() now contains deterministic output
```

### Pattern 3: Seeded RNG Hook at Package Scope
**What:** Because `combat.Dice`/`NumberPercent`/`NumberRange`/`NumberBits` are package-level functions called globally (including by `pkg/magic` and `pkg/skills`), the seed must be installed at package scope — not just on the `CombatSystem` struct.
**When to use:** Mandatory for this phase. D-02's `Rand` field alone will not seed spell damage rolls inside `pkg/magic/spells.go` [VERIFIED: 80 total call sites to `Dice`/`NumberPercent`/`NumberRange`/`NumberBits` across combat/magic/skills/ai via grep].
**Example:**
```go
// combat/dice.go (proposed refactor)
var defaultRand *rand.Rand // nil means use global math/rand

// SetRand installs a deterministic source for all dice rolls.
// Returns a restore function (call in t.Cleanup).
func SetRand(r *rand.Rand) func() {
    prev := defaultRand
    defaultRand = r
    return func() { defaultRand = prev }
}

func Dice(number, size int) int {
    if number < 1 || size < 1 { return 0 }
    total := 0
    for i := 0; i < number; i++ {
        if defaultRand != nil {
            total += defaultRand.Intn(size) + 1
        } else {
            total += rand.Intn(size) + 1
        }
    }
    return total
}
// Same pattern for NumberRange, NumberPercent, NumberBits.
```
Fixture usage:
```go
// pkg/golden/golden_test.go
restore := combat.SetRand(rand.New(rand.NewSource(42)))
t.Cleanup(restore)
```

### Pattern 4: `CombatSystem.Rand` Field (D-02 as stated)
**What:** D-02 specifies adding `Rand *rand.Rand` to `CombatSystem`. This is still valuable: it lets the *method receivers* on `CombatSystem` (OneHit, Damage, MultiHit, defense checks, DoBackstab, etc.) prefer the struct's source when set.
**When to use:** As a complementary hook alongside Pattern 3. The struct field handles `cs`-routed calls cleanly; the package-scope hook covers free-function callers in magic/skills/ai.
**Implementation note:** Inside `CombatSystem` methods, the pattern becomes:
```go
func (c *CombatSystem) rollPercent() int {
    if c.Rand != nil { return c.Rand.Intn(100) + 1 }
    return NumberPercent() // falls through to defaultRand or global
}
```
If Pattern 3 is adopted, Pattern 4's field arguably becomes redundant (all calls route through the package). The planner should decide whether to implement both (defense-in-depth) or just Pattern 3 (simpler).

### Pattern 5: Scenario Function per Axis
**What:** One helper per axis of variation, producing labeled output blocks. Matches D-03 structure.
**Example output format (Claude's discretion per CONTEXT):**
```
=== RACE × WARRIOR (Lv 20, seed=42) ===
Race=Human      HP=231  Str=18 Dex=18 Con=18  Hit%=61.2  Dam/rnd=22.3  Won=24/30
Race=Elf        HP=197  Str=16 Dex=21 Con=15  Hit%=58.0  Dam/rnd=19.8  Won=18/30
...
=== CLASS × HUMAN (Lv 20, seed=42) ===
Class=Warrior   THAC0=8   HP=231  Won=26/30  Avg rounds=22.4
Class=Mage      THAC0=16  HP=135  Won=29/30  Avg rounds=14.1  (acid/fireball)
...
=== SKILL EXECUTIONS (seed=42) ===
Backstab Lv20 thief: success=true  damage=47  victim_hp=53→6
Dodge Lv20 thief vs warrior Lv20: 6/10 dodges
Parry Lv20 warrior vs thief Lv20: 4/10 parries
Kick Lv20 warrior: success=true damage=18
...
=== SPELL EXECUTIONS (seed=42) ===
Cast 'magic missile' Lv10 mage→Lv10 warrior: damage=14 (ROM dice(level,4))
Cast 'heal' Lv20 cleric→self (Hit=50/200): Hit=199/200
Cast 'sanctuary' Lv20 cleric→self: AffSanctuary=true duration=20
Cast 'fireball' Lv22 mage→Lv22 warrior: damage=65 (ROM dice(level,6)+40)
...
```
Exact format is Claude's discretion; the principle is: one line per event, stable ordering, include the knobs (level, class, race, skill) so diffs immediately identify the drifted behavior.

### Anti-Patterns to Avoid

- **Re-deriving expected numbers in assertions** (e.g. `if dam != 14 { fail }`). The whole point of a golden file is the *file* is the expectation; hand-computed expected values are what we're trying to eliminate (they rot during migration).
- **Hidden dependencies on map iteration order.** Go's map iteration is randomized. Every loop over a map in `fixture.go` must explicitly sort keys before emitting, or the golden will be flaky even with a seeded RNG.
- **Unseeded `time.Now()` calls.** Do not include timestamps, durations measured from `time.Now()`, or goroutine IDs in the output. Stick to pure pure values.
- **Printing pointer addresses.** `fmt.Sprintf("%v", someStruct)` can include pointer addresses for unexported fields. Use explicit format strings.
- **Cross-test state leakage.** If `TestGolden` installs a seeded RNG via package-scope hook (Pattern 3) and does not `t.Cleanup` to restore, other tests in the same `go test ./...` run become deterministic in unintended ways. Always restore.
- **Running spells through real `MagicSystem.Cast` for damage parity when only the `spellFunc` is needed.** `Cast` includes mana deduction, proficiency rolls (via `combat.NumberPercent`), and combat start — great for integration, but if the goal is "does fireball still do X damage at level Y," call `spellFireball(caster, level, victim)` directly (same pattern used by `pkg/magic/magic_test.go:516`). Use both: one representative scenario through `Cast` (exercises the full path), and individual `spellXxx` calls for damage parity per spell family.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Seeded RNG for combat | A new RNG layer | Go stdlib `math/rand.New(rand.NewSource(42))` plugged into existing `combat.Dice` | Existing combat uses v1; adding a second RNG API creates drift. |
| Character/room/NPC builders | `goldenNewCharacter(...)` | `types.NewCharacter`, `types.NewNPC`, `types.NewRoom`, `types.NewObject` | Already exist; used by `combat_sim_test.go` and `magic_test.go` [VERIFIED: `combat/combat_sim_test.go:258`, `magic/magic_test.go:72`]. |
| Class-appropriate weapon/HP for a combo | Rewrite the class/race math | Import `makePlayer`/`makeMob`-style helpers from `combat_sim_test.go`, or extract to `testutil` | Per CONTEXT code context: these helpers can be extracted to a shared `testutil` if needed, or duplicated in `pkg/golden/`. Duplication is acceptable for Phase 1 — it keeps `pkg/golden/` self-contained and avoids touching the sim. |
| Spell registry setup | Construct spells by hand | `magic.DefaultSpells()` + `magic.NewMagicSystem()` | `NewMagicSystem()` returns a registry populated with all default spells [VERIFIED: `magic/system.go:72`]. |
| Skill definitions | Build a learn map from scratch | `skills.DefaultSkills()` + `skills.NewSkillSystem()` | Same pattern [VERIFIED: `skills/system.go:14`]. |
| Output capture | Manual `StringWriter` wrappers | `bytes.Buffer` + inject as `OutputFunc` callback | Pattern 2 above; trivial. |
| `-update` flag parsing | Own env var or filesystem trick | `flag.Bool("update", false, ...)` + `flag.Parse()` (or rely on `testing.Main` pre-parsing) | Canonical Go idiom. |
| Diff rendering on mismatch | Manual byte-level diff | `t.Errorf(...)` with both strings; use `testify/assert.Equal` if the built-in diff is too terse | Project likely already has testify available indirectly. |

**Key insight:** Everything the fixture needs (builders, registries, callback hooks) already exists in the codebase. The only code that must be *written* is: the driver, the scenario list, and the RNG-seeding refactor in `dice.go`. The only file that must be *created as content* is `testdata/entities.golden` — generated by running the test once with `-update`.

## Runtime State Inventory

This phase is **greenfield** (adds a new package and one refactor hook). The following categories apply minimally:

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | None — the golden file is checked into `go/pkg/golden/testdata/` as a new file; no external datastore. | None. |
| Live service config | None — phase adds a test, not a running service change. | None. |
| OS-registered state | None — no cron, systemd, or scheduled tasks are added. | None. |
| Secrets/env vars | None — the seed `42` is a literal in code; no secret material. | None. |
| Build artifacts / installed packages | The `go/rotmud` binary is unaffected (test package is not linked into main). No `go generate` artifacts. | None. |

**Nothing found in any category beyond the new test file and the `testdata/entities.golden` snapshot it produces.**

## Common Pitfalls

### Pitfall 1: Seeding only `CombatSystem.Rand` leaves magic/skills non-deterministic
**What goes wrong:** Adding `Rand *rand.Rand` to `CombatSystem` (D-02 literal spec) does not cover `pkg/magic` and `pkg/skills`, which call package-level `combat.Dice`/`NumberPercent` directly. The golden file becomes flaky on magic and skill-improvement scenarios.
**Why it happens:** Package-global RNG is the ROM heritage; the test only knows to seed what it holds a reference to.
**How to avoid:** Add package-scope `SetRand()` in `combat/dice.go` (Pattern 3). Install it in the test; restore in `t.Cleanup`.
**Warning signs:** Running `go test ./pkg/golden/...` twice produces different output on runs that include spell damage or skill-improvement checks. Fix before the first `.golden` is committed — a flaky seed is much easier to catch at inception than after a week of plan commits have baselined against it.

### Pitfall 2: Map iteration order pollutes the golden file
**What goes wrong:** A loop like `for k, v := range someMap { emit(k, v) }` produces different key orderings between runs.
**Why it happens:** Go deliberately randomizes map iteration.
**How to avoid:** Always `keys := slices.Sorted(maps.Keys(m))` (Go 1.23+) or manual sort before emitting. This applies to: `types.ClassTable` (already a slice — safe), `types.RaceTable` (slice — safe), but any `map[string]int` like `PCData.Learned` must be sorted before printing.
**Warning signs:** The first few `-update` runs produce subtly different outputs on the same seed.

### Pitfall 3: Character state leaks between scenarios
**What goes wrong:** `makePlayer(...)` returns a `*types.Character`. If the same character is reused across scenarios, post-combat state (reduced HP, depleted mana, added affects, equipment stripped into a corpse after death) contaminates the next scenario.
**Why it happens:** ROM death handling calls `victim.Unequip(i)` for every slot (`damage.go:487`) and resets armor to 100 (`damage.go:322`).
**How to avoid:** Build a fresh character per scenario (the existing `makePlayer` pattern already does this inside `runSim`'s loop). Do not share `*types.Character` across scenario functions.
**Warning signs:** A scenario run in isolation passes; running it as the N-th scenario in a sequence fails with different numbers.

### Pitfall 4: Spell damage needs explicit no-defense scenarios for parity
**What goes wrong:** `magic.MagicSystem.Cast` starts combat and eventually routes damage through `combat.Damage` which runs `CheckDefenses` — so fireball at level 22 may `dodge` or `parry` and emit zero damage, masking actual spell-damage drift during migration.
**Why it happens:** `Damage()` calls `CheckDefenses()` when `dam > 0 && ch != victim` (damage.go:49). Spells going through the damage pipeline get defended against.
**How to avoid:** For damage-parity snapshots, call `spellXxx(caster, level, victim)` directly (as `magic_test.go:500` does) and inspect the victim's HP delta. This bypasses `CheckDefenses` and isolates "what does this spell produce at this level." Use full `Cast` for a smaller set of integration scenarios.
**Warning signs:** Spell damage appears as zero in the golden file for reasons unrelated to spell mechanics.

### Pitfall 5: `-update` flag collides with `testing.T.Run` subtests
**What goes wrong:** If `-update` is defined at package scope and subtests branch on it, a partial run (`go test -run TestGolden/Races`) only regenerates that slice of the golden file and breaks other sections.
**Why it happens:** The golden file is one atomic document; partial updates leave it inconsistent.
**How to avoid:** Either (a) one monolithic `TestGolden` that always regenerates the full file on `-update`, or (b) one golden file per subtest with `testdata/races.golden`, `testdata/classes.golden`, etc. Option (a) is simpler and matches D-01 ("A `-update` flag ... regenerates the file"). Option (b) scales better if the file grows past a few thousand lines.
**Warning signs:** Developer runs `-update` on one subtest, commits, and the next CI run sees byte mismatches in the sections they didn't mean to touch.

### Pitfall 6: Parity gate doesn't fail the way we want in CI
**What goes wrong:** The golden file is committed. A developer runs tests locally with `-update` before committing (habit from other projects), the golden silently regenerates, `git add .` picks it up, and the parity violation slips into the commit.
**Why it happens:** `-update` is a footgun; the default should be "diff" and `-update` should be opt-in.
**How to avoid:** CI must run `go test ./...` without any flags — this guarantees the default-path code that diffs rather than writes. Consider adding a pre-commit check (separate phase) that fails if `testdata/entities.golden` is modified in the same commit as `.go` files unless the commit message includes `[golden-update]`.
**Warning signs:** A migration plan lands with the golden file quietly changed and nobody notices. Phase 2+ loses the parity guarantee.

### Pitfall 7: Vampire fire/silver innate vulnerability is magic, not a flag
**What goes wrong:** Coverage assumes "immunities = `victim.Imm.Has(flag)`". Vampires have an innate fire+silver vulnerability baked into `CheckImmune` via the `isVampire()` helper (combat.go:317-369) that applies regardless of `Vuln` flags.
**Why it happens:** `combat.go:364-369` adds innate vampire vulnerability after all explicit flag checks. This is hardcoded class-identity logic — exactly the kind of thing trait migration must preserve.
**How to avoid:** The fixture MUST include scenarios that exercise vampire vs fire and vampire vs silver weapons, so that when Phase 7 (identity-check refactor) or Phase 8 (class migration) rewrites this, the golden catches the regression.
**Warning signs:** Phase 7/8 plans refactor `isVampire()` and the golden passes — that's a false negative; the fixture didn't cover the innate branch.

## Code Examples

### Example A: Minimal seeded fixture driver

```go
// Source: idiomatic Go + established CombatSystem/MagicSystem injection from
//         pkg/combat/combat_sim_test.go:700 and pkg/magic/magic_test.go
package golden

import (
    "bytes"
    "flag"
    "math/rand"
    "os"
    "path/filepath"
    "testing"

    "rotmud/pkg/combat"
    "rotmud/pkg/magic"
    "rotmud/pkg/skills"
)

var updateGolden = flag.Bool("update", false, "regenerate testdata/entities.golden")

func TestGolden(t *testing.T) {
    // Install deterministic RNG for the whole call tree (combat/magic/skills/ai).
    restore := combat.SetRand(rand.New(rand.NewSource(42)))
    t.Cleanup(restore)

    var buf bytes.Buffer
    runFixture(&buf)

    path := filepath.Join("testdata", "entities.golden")
    got := buf.Bytes()

    if *updateGolden {
        if err := os.WriteFile(path, got, 0o644); err != nil {
            t.Fatalf("write golden: %v", err)
        }
        t.Logf("golden updated: %s (%d bytes)", path, len(got))
        return
    }

    want, err := os.ReadFile(path)
    if err != nil {
        t.Fatalf("read golden: %v (run `go test -run TestGolden -update` to create)", err)
    }
    if !bytes.Equal(got, want) {
        t.Fatalf("golden mismatch at %s\nrun `go test -run TestGolden -update` if the behavior change is intentional\n\n--- want (%d bytes)\n%s\n--- got (%d bytes)\n%s",
            path, len(want), want, len(got), got)
    }
}
```

### Example B: Race × warrior scenario

```go
// Source: pattern derived from pkg/combat/combat_sim_test.go:693 (runSim)
func runRaceWarriorCombo(buf *bytes.Buffer, raceIdx int) {
    race := &types.RaceTable[raceIdx]
    cl := &types.ClassTable[types.ClassWarrior]

    ch := types.NewCharacter(cl.Name + "/" + race.Name)
    ch.Level = 20
    ch.Class = types.ClassWarrior
    ch.Race = raceIdx
    // ... set stats/HP/armor using existing helpers ...

    mob := types.NewNPC(1, "Mob", 20)
    mob.Act.Set(types.ActWarrior)
    // ... standard mob setup ...

    room := types.NewRoom(1, "Arena", "Arena.")
    ch.InRoom = room
    mob.InRoom = room
    room.AddPerson(ch)
    room.AddPerson(mob)

    cs := combat.NewCombatSystem()
    // Output intentionally collected via buf, or discarded if we summarize stats
    cs.Output = func(_ *types.Character, _ string) {}
    cs.SkillGetter = func(ch *types.Character, name string) int { return 75 } // fixed for parity

    combat.SetFighting(ch, mob)
    combat.SetFighting(mob, ch)

    // Run N deterministic rounds, aggregate
    var pDmg, pHits, pMiss int
    for i := 0; i < 30; i++ {
        before := mob.Hit
        cs.OneHit(ch, mob, false)
        dealt := before - mob.Hit
        if dealt > 0 {
            pDmg += dealt
            pHits++
        } else {
            pMiss++
        }
    }

    fmt.Fprintf(buf, "Race=%-10s  HP=%-4d  Str=%-2d Dex=%-2d Con=%-2d  Hit%%=%5.1f  Dam=%3d\n",
        race.Name, ch.MaxHit,
        ch.PermStats[types.StatStr], ch.PermStats[types.StatDex], ch.PermStats[types.StatCon],
        100*float64(pHits)/float64(pHits+pMiss), pDmg)
}
```

### Example C: Spell parity scenario

```go
// Source: pattern from pkg/magic/magic_test.go:494 (TestDamageSpells)
func runSpellCombo(buf *bytes.Buffer, spellName string, casterLevel int) {
    ms := magic.NewMagicSystem()
    spell := ms.Registry.FindByName(spellName)
    if spell == nil {
        fmt.Fprintf(buf, "Spell=%-20s NOT_FOUND\n", spellName)
        return
    }

    caster := types.NewCharacter("Caster")
    caster.Level = casterLevel
    caster.Class = types.ClassMage
    caster.Mana = 1000; caster.MaxMana = 1000

    victim := types.NewCharacter("Victim")
    victim.Level = casterLevel
    victim.Hit = 1000; victim.MaxHit = 1000

    before := victim.Hit
    success := spell.Func(caster, casterLevel, victim)
    after := victim.Hit
    fmt.Fprintf(buf, "Spell=%-20s Lv%-3d  success=%-5v  damage=%-4d  target_hp=%d→%d\n",
        spellName, casterLevel, success, before-after, before, after)
}
```

### Example D: Defensive skill parity (dodge/parry)

```go
// Source: dodge/parry live inside CombatSystem.CheckDefenses (pkg/combat/defense.go)
func runDefenseTrial(buf *bytes.Buffer, defenderClass, attackerClass, level int) {
    cs := combat.NewCombatSystem()
    cs.Output = func(_ *types.Character, _ string) {}
    cs.SkillGetter = func(ch *types.Character, name string) int {
        // Fixed high skill so each decision is driven purely by the seeded RNG
        if name == "dodge" || name == "parry" || name == "shield block" { return 80 }
        return 75
    }

    def := makePlayer(defenderClass, types.RaceHuman, level) // or local helper
    atk := makePlayer(attackerClass, types.RaceHuman, level)
    // ... room setup, SetFighting(atk, def) ...

    var dodged, parried, blocked, hit int
    for i := 0; i < 100; i++ {
        r := cs.CheckDefenses(atk, def)
        switch r {
        case combat.DefenseDodged:  dodged++
        case combat.DefenseParried: parried++
        case combat.DefenseBlocked: blocked++
        default:                    hit++
        }
    }
    fmt.Fprintf(buf, "Defense Lv%d  %s vs %s:  dodged=%d  parried=%d  blocked=%d  hit=%d\n",
        level,
        types.ClassTable[defenderClass].Name,
        types.ClassTable[attackerClass].Name,
        dodged, parried, blocked, hit)
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `math/rand.Seed(42)` global reseed | `math/rand.New(rand.NewSource(42))` instance | Go 1.20 deprecated global `Seed()` | Use instance-style or v2 `NewPCG`; the package stays on v1 here for consistency with existing `dice.go`. |
| Third-party golden libs (`goldie`, etc.) | Stdlib `-update` flag pattern | Always the idiomatic Go choice | No extra dep; diffs are readable from `git diff`. |
| `assert.Equal(t, got, want)` one-liner | Manual `bytes.Equal` + `t.Fatalf` with both sides | Stdlib sufficient; testify is optional | Avoids adding a direct dep for one assertion. |

**Deprecated / outdated:**
- `rand.Seed(time.Now().UnixNano())` — the `Seed()` top-level function is deprecated in Go 1.20+. The existing code in `combat/dice.go` calls `rand.Intn(...)` directly on the global source (which is auto-seeded); this is still valid for runtime, but for tests the new pattern is always `rand.New(rand.NewSource(n))`.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Extracting `makePlayer`/`makeMob`-style helpers is acceptable to duplicate in `pkg/golden/` rather than moved to a shared `testutil`. | Don't Hand-Roll | LOW — CONTEXT.md code insights explicitly notes this choice: "extracted to a shared `testutil` if needed, or duplicated in `pkg/golden/`". Planner can decide. |
| A2 | CI runs `go test ./...` without `-update`. | Pitfall 6 | LOW — standard Go convention; verified by `.mise.toml` task `test: "go test ./..."`. |
| A3 | 30-round per-combo sample size (Example B) is sufficient to produce stable aggregated numbers under a fixed seed. | Example B | MEDIUM — with a fixed seed, any N is deterministic, but very small N may mask variance. If the golden ends up noisy in practice (e.g. one roll flipping a hit/miss across a boundary), increase N. Easy to tune after first `-update` run. |
| A4 | Adding `testify/assert` is optional and not required. | Standard Stack | LOW — stdlib comparison works; testify is a nice-to-have for diff rendering. |
| A5 | The existing fallback `GetSkill` in `CombatSystem` (returns `20 + ch.Level*2` when `SkillGetter == nil`) is acceptable for the fixture's baseline scenarios. | Example D | LOW — the fixture should inject `SkillGetter` explicitly for reproducibility anyway (Example D does). |

## Open Questions (RESOLVED)

1. **Extract shared `testutil` vs. duplicate helpers?**
   - What we know: `combat_sim_test.go` has `makePlayer`, `makeMob`, `raceStatAtLevel`, `classEquipAC`, `weaponDice`, etc. — about 400 lines of useful builders.
   - What's unclear: whether refactoring these into `pkg/testutil/` as part of this phase is in scope, or whether duplication is preferred to keep this phase minimal.
   - RESOLVED: **duplicate for Phase 1.** The sim file is a moving target (recent commits have tuned it heavily); decoupling the golden fixture from sim-tuning churn is valuable. A later cleanup phase can merge them if the duplication pain grows.

2. **Include a representative mob section or defer entirely?** (per Claude's discretion)
   - What we know: `makeMob` and `makeCasterMob` exist. Aggro/assist behavior lives in `pkg/ai/` and requires a `GameHandlers` struct to wire up fully.
   - What's unclear: whether a *stat-and-immunity* mob snapshot (without AI behavior) counts as "representative mob behavior" per success criterion #3.
   - RESOLVED: **include stat/immunity coverage for warrior + caster mob templates (matches what `combat_sim_test.go` uses); explicitly defer aggro/assist AI to a later phase, and note this in the fixture comment so Phase 6 (Mob Type Loaders) knows to extend it.**

3. **One golden file or multiple?**
   - What we know: D-01 says "the file" (singular).
   - What's unclear: whether growing the file past ~5000 lines becomes a review burden.
   - RESOLVED: **start with one `testdata/entities.golden`.** If it crosses ~10k lines during future expansion (Phase 8+ adds the full 19×14 matrix), split then. Premature splitting adds coordination cost.

4. **Should `SetRand` be exported from `pkg/combat` or hidden behind a test-only build tag?**
   - What we know: `pkg/golden/` imports `pkg/combat`; an exported `SetRand` is reachable by any caller.
   - What's unclear: whether test-only hooks leak into production binaries.
   - RESOLVED: **export it publicly.** The function is harmless (passing `nil` restores global). Build tags would complicate testing and IDE tooling. A doc comment `// SetRand installs a deterministic RNG source; intended for tests.` is sufficient.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | All of Go build/test | ✓ | 1.23.12 | — |
| `git` | `git diff` review of `.golden` regressions | ✓ | system | — |
| `mise` | Optional task runner (`mise run test`) | Project assumes available | — | `go test ./...` direct |

**Missing dependencies with no fallback:** None.
**Missing dependencies with fallback:** None.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` (Go 1.23.12) |
| Config file | None — Go tests are zero-config |
| Quick run command | `go test ./pkg/golden/ -run TestGolden` (from `go/` directory) |
| Full suite command | `go test ./...` (from `go/` directory) — matches existing `mise run test` |
| Regenerate golden | `go test ./pkg/golden/ -run TestGolden -update` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| MIGRATE-06 (success #1) | 19 races × warrior + 14 classes × human covered with combat events | golden-fixture unit | `go test ./pkg/golden/ -run TestGolden` | ❌ Wave 0 |
| MIGRATE-06 (success #2) | Damage spells, affect spells, healing spells, backstab/dodge/parry/kick with seeded RNG | golden-fixture unit | `go test ./pkg/golden/ -run TestGolden` | ❌ Wave 0 |
| MIGRATE-06 (success #3) | Representative mob templates (aggro omitted by open-question #2; stats/immunity/special attacks included) | golden-fixture unit | `go test ./pkg/golden/ -run TestGolden` | ❌ Wave 0 |
| MIGRATE-06 (success #4) | Running fixture twice produces byte-identical output | determinism check | `go test ./pkg/golden/ -run TestGolden -count=2` | ❌ Wave 0 |
| MIGRATE-06 (success #5) | Intentional behavior change produces a visible, diffable failure | CI parity gate | `go test ./...` (no flags) | Existing CI pipeline |

### Sampling Rate
- **Per task commit:** `go test ./pkg/golden/ -run TestGolden` (quick, ~seconds)
- **Per wave merge:** `go test ./...` (full suite — includes `combat_sim_test.go` and golden; combined runtime ~30s based on existing sim test sizes)
- **Phase gate:** `go test ./... -count=1` passes on a clean tree; `go test ./... -count=2` passes on the golden test specifically to prove determinism.

### Wave 0 Gaps
- [ ] `go/pkg/golden/doc.go` — package documentation
- [ ] `go/pkg/golden/golden_test.go` — entry test with `-update` flag
- [ ] `go/pkg/golden/fixture.go` — scenario builders (race, class, spell, skill)
- [ ] `go/pkg/golden/testdata/entities.golden` — initial snapshot (produced by `-update` run)
- [ ] `go/pkg/combat/dice.go` — refactor to add `defaultRand` + `SetRand()` hook
- [ ] Optional: extract `makePlayer`/`makeMob` helpers to a shared testutil (see Open Question #1; recommendation: skip for this phase)

*No framework install required — all tooling is stdlib.*

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | Phase adds test code; no auth surface. |
| V3 Session Management | no | No session handling. |
| V4 Access Control | no | No authz surface. |
| V5 Input Validation | no | Fixture input is code-internal; no untrusted data. |
| V6 Cryptography | no | RNG is deterministic-for-test only. `math/rand` (v1) is NOT a CSPRNG; this is intentional. No secret material. |
| V11 Business Logic | yes | The fixture IS the business-logic parity gate — its whole job is to detect logic drift. Control: the fixture exists and runs in CI (success criteria #4, #5). |
| V14 Configuration | marginal | Seed value `42` is a configuration constant. Control: document it in `pkg/golden/doc.go` so nobody "improves" it to a random value. |

### Known Threat Patterns for Go test infrastructure

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Non-deterministic test leaks into prod (e.g., `SetRand` called in production code path) | Tampering | `SetRand(nil)` is a no-op restore; `combat.Dice` falls through to global `rand.Intn` when `defaultRand == nil` — verify no production code calls `SetRand`. |
| Golden file becomes a "silent approval" — devs update without understanding the change | Repudiation (process-level) | Require the golden diff to be reviewed in the PR; consider a CODEOWNERS entry on `pkg/golden/testdata/`. (Process control, not code.) |
| Seed collision with a deliberately adversarial data file (e.g., if Phase 8 TOML files had content derived from seed-42 output) | Tampering | Seed is a constant and the fixture output is derived from it; no input channel exists for adversarial influence. N/A. |
| Flaky test masks real regressions | Denial-of-service (test suite utility) | Phase 1 explicitly eliminates flakiness via seeded RNG (D-02); determinism check on `-count=2` is the guard. |

No new network surface, no new data ingress, no new parsing of untrusted input. This phase is isolated test infrastructure.

## Sources

### Primary (HIGH confidence)
- **Codebase** — `/home/antti/Repos/Misc/ROT-MUD/go/pkg/combat/dice.go` (RNG implementation), `combat/combat.go` (CombatSystem struct + callback fields), `combat/hit.go` (OneHit), `combat/damage.go` (Damage pipeline + defense hook), `combat/defense.go` (dodge/parry/shield block), `combat/skills.go` (DoBackstab/DoBash/DoKick/DoAssassinate), `combat/combat_sim_test.go` (makePlayer/makeMob/runSim patterns), `magic/system.go` (MagicSystem.Cast), `magic/magic_test.go` (direct spellXxx calls), `magic/spells.go` (80+ `combat.Dice`/`NumberPercent` call sites), `skills/system.go` (GetSkill, CheckImprove), `skills/skill.go` (Skill/SkillRegistry), `types/races.go` (19 races + stats), `types/classes.go` (14 classes + THAC0/HP curves), `go.mod` (Go 1.25.5 module, 1.23 runtime).
- **CONTEXT.md** — `/home/antti/Repos/Misc/ROT-MUD/.planning/phases/01-golden-master-safety-net/01-CONTEXT.md` — locked decisions D-01 through D-05, canonical refs, code insights.
- **REQUIREMENTS.md** — MIGRATE-06 definition.
- **CLAUDE.md** — Go stdlib conventions, `tabs+gofmt` style, `gsd`-workflow enforcement.

### Secondary (MEDIUM confidence)
- **pkg.go.dev/math/rand/v2** — https://pkg.go.dev/math/rand/v2 — confirms v1 vs v2 PRNG semantics; verified via WebFetch 2026-04-17.
- **pkg.go.dev/testing** — Go testing package docs — `t.Cleanup`, `-update` flag idiom.

### Tertiary (LOW confidence)
- None — every claim in this document is either code-verified in the ROT-MUD repo or cited from Go stdlib docs.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — every library is stdlib and already in use.
- Architecture: HIGH — file locations, imports, and refactor site all verified by direct code inspection.
- Pitfalls: HIGH (1, 2, 3, 4, 7) / MEDIUM (5, 6) — pitfalls 5/6 are procedural/CI-process risks, not code risks.
- RNG plan: HIGH — 80-call-site count verified by grep; ROM heritage of global RNG confirmed.

**Research date:** 2026-04-17
**Valid until:** 2026-05-17 (30 days; the ROT-MUD Go code and plans are stable, and stdlib idioms change slowly).

---
*Phase: 01-golden-master-safety-net*
*Research completed: 2026-04-17*
