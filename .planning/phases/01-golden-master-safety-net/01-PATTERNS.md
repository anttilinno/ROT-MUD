# Phase 1: Golden-Master Safety Net - Pattern Map

**Mapped:** 2026-04-17
**Files analyzed:** 6 (4 new, 2 modified — plus a generated testdata artifact)
**Analogs found:** 6 / 6

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `go/pkg/golden/doc.go` (new) | package-doc | n/a | `go/pkg/combat/doc.go` | exact |
| `go/pkg/golden/golden_test.go` (new) | test-driver | request-response (read/write snapshot file) | `go/pkg/combat/combat_sim_test.go` (runSim + CombatSystem injection) | role-match |
| `go/pkg/golden/fixture.go` (new) | test-fixture / scenario builders | batch (iterate over 33 combos + spells/skills) | `go/pkg/combat/combat_sim_test.go` (makePlayer, makeMob, runSim) | exact |
| `go/pkg/golden/testdata/entities.golden` (generated) | test-fixture-data | file-I/O (read/compare, `-update` rewrites) | — (no existing golden files in the codebase) | none — bootstrap |
| `go/pkg/combat/dice.go` (modify) | utility / RNG primitive | transform (int→int, stateful via package var) | existing `dice.go` (self) + `combat.go` `CombatSystem` callback-field pattern | role-match (refactor in place) |
| `go/pkg/combat/combat.go` (modify: add `Rand *rand.Rand` field) | core system struct | request-response (method-receiver injection) | existing `CombatSystem` struct (self) — follows `Output`/`SkillGetter`/`RoomFinder` callback-field convention | exact |

## Pattern Assignments

### `go/pkg/golden/doc.go` (package-doc)

**Analog:** `go/pkg/combat/doc.go` (lines 1-76)

**Doc-file structure pattern** (lines 1-10):
```go
// Package combat implements the combat system for the ROT MUD.
//
// This package handles all aspects of combat including attack resolution,
// damage calculation, death handling, and experience rewards. It is ported
// from the original fight.c.
//
// # Combat System
//
// The [CombatSystem] manages combat operations:
package combat
```

**What to copy:**
- Package-level comment block immediately above `package` keyword.
- `# Section` markdown-style headings (rendered by `go doc`).
- `[TypeName]` linking to exported types.
- Final line = `package golden` (no imports).

**What to add that's specific to `pkg/golden`:**
- Seed constant documentation: `// Seed 42 is chosen by fiat; do not change lightly — every existing golden line was produced under this seed.` (addresses Research Open Question #4 / Pitfall 6).
- A `# Usage` section with `go test ./pkg/golden/ -run TestGolden` and `-update` invocations.

---

### `go/pkg/golden/golden_test.go` (test-driver, request-response)

**Analog:** `go/pkg/combat/combat_sim_test.go` (lines 1-29, 697-744, 941-999)

**Imports pattern** (combat_sim_test.go lines 23-29):
```go
import (
	"fmt"
	"strings"
	"testing"

	"rotmud/pkg/types"
)
```

**What to copy:** The `rotmud/pkg/*` import-path style with stdlib grouped separately (blank-line-separated groups).

**What to add for golden driver:**
```go
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
	"rotmud/pkg/types"
)
```

**CombatSystem injection pattern** (combat_sim_test.go lines 698-727):
```go
cs := NewCombatSystem()
cs.Output = func(_ *types.Character, _ string) {}
cs.SkillGetter = func(ch *types.Character, skillName string) int {
	if ch.IsNPC() {
		s := 20 + ch.Level*2
		if s > 80 {
			s = 80
		}
		return s
	}
	switch skillName {
	case "dodge":
		return dodgeSkillForClass(ch.Class, ch.Level)
	case "parry":
		return parrySkillForClass(ch.Class, ch.Level)
	// ...
	}
	return weaponSkillForClass(ch.Class, ch.Level)
}
```

**What to copy:** The `cs.Output = func(_ *types.Character, _ string) {}` suppression idiom when not capturing, OR the `fmt.Fprintf(&buf, ...)` variant when capturing. The `SkillGetter` callback for deterministic skill levels.

**What to change for golden test:** Route `cs.Output` to a `bytes.Buffer` instead of suppressing, since the golden file is the output.

**Test-entry-point pattern** (combat_sim_test.go lines 941-944):
```go
func TestCombatSimByClass(t *testing.T) {
	const raceIdx = types.RaceHuman
	const n = 1000
	levels := []int{1, 10, 20, 30, 40, 50, 60, 75, 100}
```

**What to add that doesn't exist anywhere in the repo** (bootstrap — from Research Example A):
```go
var updateGolden = flag.Bool("update", false, "regenerate testdata/entities.golden")

func TestGolden(t *testing.T) {
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
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden: %v (run `go test -run TestGolden -update` to create)", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("golden mismatch at %s\n--- want (%d)\n%s\n--- got (%d)\n%s",
			path, len(want), want, len(got), got)
	}
}
```

**Error-handling pattern** (combat_test.go lines 19-26):
```go
if Dice(0, 6) != 0 {
	t.Error("Dice(0,6) should return 0")
}
```

**What to copy:** Plain `t.Fatal`/`t.Fatalf`/`t.Errorf` from stdlib testing — no testify. Same style used across every `_test.go` in the repo.

---

### `go/pkg/golden/fixture.go` (test-fixture, batch)

**Analog:** `go/pkg/combat/combat_sim_test.go` (lines 258-302, 325-363, 693-935)

**Character/NPC/Room builder pattern** (combat_sim_test.go lines 258-302):
```go
func makePlayer(classIdx, raceIdx, level int) *types.Character {
	cl := &types.ClassTable[classIdx]
	race := &types.RaceTable[raceIdx]

	ch := types.NewCharacter(cl.Name + "/" + race.Name)
	ch.Level = level
	ch.Class = classIdx
	ch.Race = raceIdx

	ch.MaxHit = playerHP(cl, race, level)
	ch.Hit = ch.MaxHit
	ch.MaxMana = playerMana(classIdx, race, level)
	ch.Mana = ch.MaxMana

	for stat := 0; stat < types.MaxStats; stat++ {
		ch.PermStats[stat] = raceStatAtLevel(race, stat, level)
	}

	ac := classEquipAC(classIdx, level)
	for i := range ch.Armor {
		ch.Armor[i] = ac
	}

	ch.HitRoll = level / 3
	ch.DamRoll = level / 4

	weapon := makeWeapon(classIdx, level)
	ch.Equip(weapon, types.WearLocWield)

	ch.Position = types.PosStanding
	return ch
}
```

**Recommendation per Research Open Question #1:** duplicate `makePlayer`, `makeMob`, `raceStatAtLevel`, `playerHP`, `classEquipAC`, `weaponDice`, `makeWeapon`, `playerMana` into `pkg/golden/` (do NOT extract to `pkg/testutil/` in this phase — the sim is a moving target; decoupling keeps the golden stable).

**Mob builder pattern** (combat_sim_test.go lines 325-363):
```go
func makeMob(level int) *types.Character {
	mob := types.NewNPC(1, "Mob", level)
	mob.Act.Set(types.ActWarrior)
	mob.MaxHit = mobHP(level)
	mob.Hit = mob.MaxHit
	mob.PermStats[types.StatStr] = 15 + level/6
	mob.PermStats[types.StatDex] = 14
	// ...
	mob.Position = types.PosStanding
	return mob
}
```

**Room-and-combat-setup pattern** (combat_sim_test.go lines 737-744):
```go
room := types.NewRoom(1, "Arena", "Arena.")
p.InRoom = room
m.InRoom = room
room.AddPerson(p)
room.AddPerson(m)

SetFighting(p, m)
SetFighting(m, p)
```

**What to copy:** Fresh character per scenario (do not reuse across iterations — Research Pitfall #3). `SetFighting` both directions to activate combat.

**Output-capture variant for golden** (adapted from combat_sim_test.go line 700):
```go
// combat_sim_test.go suppresses:
cs.Output = func(_ *types.Character, _ string) {}

// golden fixture captures:
cs.Output = func(ch *types.Character, msg string) {
	fmt.Fprintf(buf, "[%s] %s", ch.Name, msg)
}
```

**Spell direct-call pattern** (from `pkg/magic/magic_test.go` lines 494-523):
```go
func TestDamageSpells(t *testing.T) {
	t.Run("magic missile", func(t *testing.T) {
		victim := types.NewCharacter("Target")
		victim.Hit = 100
		victim.MaxHit = 100

		success := spellMagicMissile(nil, 10, victim)
		if !success {
			t.Error("magic missile should succeed")
		}
		if victim.Hit >= 100 {
			t.Error("victim should take damage")
		}
	})
}
```

**What to copy:** `spellXxx(caster, level, victim)` calls bypass `CheckDefenses` and isolate per-spell damage parity (Research Pitfall #4). These are package-private in `pkg/magic/`, so from `pkg/golden/` the fixture must route through `MagicSystem.Cast` OR through the `Spell.Func` obtained from `Registry.FindByName(...)`. Prefer the `Registry` lookup because it mirrors how `magic_test.go` operates without access to unexported helpers:

```go
ms := magic.NewMagicSystem()
spell := ms.Registry.FindByName("magic missile")
success := spell.Func(caster, casterLevel, victim) // bypasses Cast/mana/defense
```

**Aggregate-loop pattern for combat combos** (combat_sim_test.go lines 732-935):
```go
for i := 0; i < n; i++ {
	p := makePlayer(classIdx, raceIdx, level)  // FRESH per iteration
	m := mobFn(level)
	// ... room setup ...
	SetFighting(p, m); SetFighting(m, p)
	for rounds < maxRounds {
		if p.Position <= types.PosDead || m.Position <= types.PosDead { break }
		rounds++
		cs.MultiHit(p, m)
		// ... mob attack ...
	}
	// aggregate into simResult
}
```

**What to copy:** Bounded round loop (`maxRounds = 200` or smaller for fixtures), early termination on death, accumulation into a struct, cleanup at end of iteration (`room.RemovePerson(p)`, `p.Fighting = nil`).

**Output format (per Claude's discretion, CONTEXT D-01 / Research Example B+E)**:
```go
fmt.Fprintf(buf, "Race=%-10s  HP=%-4d  Str=%-2d Dex=%-2d Con=%-2d  Hit%%=%5.1f  Dam=%3d\n",
	race.Name, ch.MaxHit,
	ch.PermStats[types.StatStr], ch.PermStats[types.StatDex], ch.PermStats[types.StatCon],
	100*float64(pHits)/float64(pHits+pMiss), pDmg)
```

**Error handling:**
- No errors expected from fixture builders. If `types.RaceTable[idx]` is out of range, that's a coding bug — let it panic; the test will fail loudly.
- Spell-not-found case: emit `"Spell=%s NOT_FOUND\n"` (Research Example C) rather than fatal — keeps the fixture resilient to Phase 8 renames.

---

### `go/pkg/golden/testdata/entities.golden` (test-fixture-data, file-I/O)

**Analog:** none — this codebase has no existing `testdata/*.golden` files (verified by `Glob go/pkg/**/testdata/**` returning no results, and `Grep` for `flag.Bool("update"` returning zero matches).

**Pattern source:** Go stdlib idiom (cited in RESEARCH.md Pattern 1 / Example A).

**Generation path:**
1. `fixture.go` + `golden_test.go` land first without this file present.
2. Developer runs `go test ./pkg/golden/ -run TestGolden -update` from `go/` directory.
3. `os.WriteFile("testdata/entities.golden", buf.Bytes(), 0o644)` produces the file.
4. Developer inspects the file (must be human-readable — CONTEXT specifics), then `git add` commits it.

**No code excerpt to copy** — the file is generated, not authored.

**Planner note:** The Phase-1 PLAN must include an explicit "generate the initial snapshot" task after the code tasks. The task description should emphasize manual review of the generated file before committing (defense against Pitfall #6: silent regen).

---

### `go/pkg/combat/dice.go` (modify: add seeded-RNG hook)

**Analog:** `go/pkg/combat/dice.go` itself (self-referential refactor) + the callback-field pattern already used in `go/pkg/combat/combat.go` lines 38-47.

**Current dice.go** (lines 1-40):
```go
package combat

import (
	"math/rand"
)

// Dice rolls a number of dice with a given size
// e.g., Dice(2, 6) rolls 2d6
func Dice(number, size int) int {
	if number < 1 || size < 1 {
		return 0
	}

	total := 0
	for i := 0; i < number; i++ {
		total += rand.Intn(size) + 1
	}
	return total
}

func NumberRange(low, high int) int {
	if low >= high {
		return low
	}
	return low + rand.Intn(high-low+1)
}

func NumberPercent() int {
	return rand.Intn(100) + 1
}

func NumberBits(bits int) int {
	if bits <= 0 {
		return 0
	}
	return rand.Intn(1 << bits)
}
```

**Proposed refactor pattern (from Research Pattern 3):**
```go
// Unexported package-level source. nil means use global math/rand (production default).
var defaultRand *rand.Rand

// SetRand installs a deterministic RNG source for all dice rolls in this package.
// Intended for tests only. Returns a restore function; call it in t.Cleanup.
// Passing nil restores the global math/rand source.
func SetRand(r *rand.Rand) func() {
	prev := defaultRand
	defaultRand = r
	return func() { defaultRand = prev }
}

// randIntn routes to defaultRand when set, else falls through to global math/rand.
func randIntn(n int) int {
	if defaultRand != nil {
		return defaultRand.Intn(n)
	}
	return rand.Intn(n)
}

func Dice(number, size int) int {
	if number < 1 || size < 1 { return 0 }
	total := 0
	for i := 0; i < number; i++ {
		total += randIntn(size) + 1
	}
	return total
}
// Identical shape for NumberRange, NumberPercent, NumberBits.
```

**Why this pattern (not just D-02's `CombatSystem.Rand`):** Research verified 131 call sites of `combat.Dice`/`NumberPercent`/`NumberRange`/`NumberBits` across 13 files including `pkg/magic/spells.go` (65), `pkg/ai/`, `pkg/skills/`. Seeding only `CombatSystem.Rand` leaves magic+skills+ai non-deterministic. Package-scope hook covers every caller.

**Error handling:** None needed. `SetRand(nil)` restores global; race-free for single-test use (no mutex — Research notes tests are sequential within a package by default and `SetRand` is the only writer).

**Coding convention compliance** (per CLAUDE.md):
- Public functions use PascalCase → `SetRand`.
- Private helpers use camelCase → `randIntn`, `defaultRand`.
- Doc comment on every exported identifier.
- Error-free functions return a single value (or restore-closure).

---

### `go/pkg/combat/combat.go` (modify: add `Rand *rand.Rand` field per D-02)

**Analog:** `go/pkg/combat/combat.go` itself (the existing `CombatSystem` callback-field convention).

**Existing callback-field pattern** (combat.go lines 38-47):
```go
type CombatSystem struct {
	Output      OutputFunc
	RoomFinder  RoomFinderFunc  // For finding recall room on death
	CharMover   CharMoverFunc   // For moving characters to rooms
	SkillGetter SkillGetterFunc // For checking skill levels
	OnLevelUp   OnLevelUpFunc   // Called when a character levels up
	OnDamage    OnDamageFunc    // Called when damage is dealt (for metrics)
	OnKill      OnKillFunc      // Called when a character is killed (for quests)
	OnDeath     OnDeathFunc     // Called after death processing (for autoloot/autosac)
}
```

**Proposed addition:**
```go
type CombatSystem struct {
	Output      OutputFunc
	RoomFinder  RoomFinderFunc
	CharMover   CharMoverFunc
	SkillGetter SkillGetterFunc
	OnLevelUp   OnLevelUpFunc
	OnDamage    OnDamageFunc
	OnKill      OnKillFunc
	OnDeath     OnDeathFunc
	Rand        *rand.Rand // When non-nil, combat rolls use this source instead of package RNG (test hook).
}
```

**Fallback-when-nil pattern** (existing GetSkill in combat.go lines 50-56):
```go
func (c *CombatSystem) GetSkill(ch *types.Character, skillName string) int {
	if c.SkillGetter != nil {
		return c.SkillGetter(ch, skillName)
	}
	return 20 + ch.Level*2  // fallback
}
```

**What to copy:** The `if c.Field != nil { use it } else { fallback }` guard pattern. Applied to Rand:
```go
func (c *CombatSystem) rollPercent() int {
	if c.Rand != nil {
		return c.Rand.Intn(100) + 1
	}
	return NumberPercent() // routes through package-level defaultRand, else global
}
```

**Planner decision (per Research Pattern 4 final note):** Since Pattern 3 (`SetRand` at package scope) already covers every call site, the `CombatSystem.Rand` field becomes largely redundant — a belt-and-suspenders hook. Implement it per D-02 literal spec; use it inside the few `CombatSystem`-receiver methods that could benefit (hit.go, damage.go `CheckDefenses` entry). The package-scope hook is still mandatory — the struct field alone fails Pitfall #1.

---

## Shared Patterns

### Package-level Doc Convention

**Source:** Every `doc.go` in the repo — consistent style across `combat`, `magic`, `skills`, `help`, `server`, `loader`, `types`, `builder`, `ai`, `game`, `persistence`, `shops`.

**Apply to:** `go/pkg/golden/doc.go`

**Excerpt** (combat/doc.go lines 1-10):
```go
// Package combat implements the combat system for the ROT MUD.
//
// This package handles all aspects of combat including attack resolution,
// damage calculation, death handling, and experience rewards.
// ...
// # Usage Example
//
//	cs := combat.NewCombatSystem()
//	cs.Output = sendToPlayer
package combat
```

### Test Import & Naming Convention

**Source:** `go/pkg/combat/combat_test.go` lines 1-7; `go/pkg/magic/magic_test.go` lines 1-7; `go/pkg/skills/skills_test.go` lines 1-7 — every test file in the repo follows this.

**Apply to:** `go/pkg/golden/golden_test.go`

**Excerpt:**
```go
package golden    // same package as production code (allows touching unexported helpers)

import (
	"testing"

	"rotmud/pkg/types"
)

func TestGolden(t *testing.T) {
	// use t.Fatal / t.Errorf / t.Run — NO testify
}
```

### Error Handling Style

**Source:** Per CLAUDE.md project conventions + `go/pkg/loader/loader.go` and `go/pkg/combat/combat_test.go` usage.

**Apply to:** Every new file.

**Conventions:**
- `(result, error)` tuple for fallible I/O (`os.ReadFile` → check `err`).
- `fmt.Errorf("context: %w", err)` for wrapping.
- Lowercase error messages: `"read golden: %v"`, not `"Read Golden: %v"`.
- In tests, prefer `t.Fatalf("read golden: %v (run with -update to create)", err)` — one line, context-prefixed, actionable.

### Naming Conventions (CLAUDE.md-enforced)

**Apply to:** All code in `pkg/golden/` and the dice.go modifications.

- Public identifiers PascalCase: `SetRand`, `TestGolden`, `Rand`.
- Private identifiers camelCase: `defaultRand`, `randIntn`, `runFixture`, `updateGolden`.
- Constructor `New*` prefix: not applicable here (no new system struct) — `NewCombatSystem`/`NewMagicSystem`/`NewSkillSystem` already exist and are reused.
- Character aliases `ch`, `victim`, `room`, `mob` — match existing combat_sim_test.go usage.
- Tabs for indentation (gofmt default).

### CombatSystem Injection Convention

**Source:** `go/pkg/combat/combat_sim_test.go:698-727`, `go/pkg/combat/combat.go:38-47`.

**Apply to:** `go/pkg/golden/fixture.go` when instantiating `CombatSystem` per scenario.

**Pattern:** Build `*CombatSystem` via `NewCombatSystem()`, then assign callback fields directly on the struct (`cs.Output = ...`, `cs.SkillGetter = ...`). Never mutate from inside running combat code.

### Fresh-Character-Per-Scenario Invariant

**Source:** `combat_sim_test.go:732-744` (inside the `for i := 0; i < n; i++` loop, `makePlayer`/`mobFn` are called fresh every iteration).

**Apply to:** Every scenario iteration in `fixture.go`.

**Why:** ROM death handling strips equipment (`damage.go:487`) and mutates armor (`damage.go:322`). Reusing a `*types.Character` across scenarios contaminates subsequent iterations (Pitfall #3).

### Deterministic Map Iteration (Golden-Specific)

**Source:** Research Pitfall #2. Not present in existing codebase (the sim uses `ClassTable` and `RaceTable` which are slices, so ordering is already deterministic).

**Apply to:** Any map emitted into the golden buffer in `fixture.go`. `PCData.Learned` is `map[string]int`.

**Convention:** `keys := slices.Sorted(maps.Keys(m))` (Go 1.23+) or `sort.Strings(keys)` before emitting.

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `go/pkg/golden/testdata/entities.golden` | test-fixture-data | file-I/O | No existing golden/snapshot files in the repo; this phase bootstraps the pattern. Subsequent phases (2-12) can reference THIS file as the canonical analog for any future snapshot testing. |

The `flag.Bool("update", ...)` / golden-file / `testdata/` pattern is **absent from the codebase**. Verified via:
- `Glob go/pkg/**/testdata/**` → no files
- `Grep` for `flag.Bool("update"` → no matches
- `Grep` for `testdata` in Go files → no matches

For this file, planner should reference RESEARCH.md Example A (and Pattern 1) directly — there is no in-repo precedent to copy from. Research cites this as stdlib idiom (pkg.go.dev/testing), so the canonical source is external docs, not codebase code.

## Metadata

**Analog search scope:** `go/pkg/combat/`, `go/pkg/magic/`, `go/pkg/skills/`, `go/pkg/types/`, `go/pkg/help/`, `go/pkg/loader/`, `go/pkg/persistence/`, `go/pkg/ai/`, `go/pkg/game/`, `go/pkg/server/`, `go/pkg/builder/`, `go/pkg/shops/`.

**Files scanned:** 26 `_test.go` files + 12 `doc.go` files + all sources under `pkg/combat/`, `pkg/magic/`, `pkg/skills/`.

**Key verified facts carried into pattern assignments:**
- 131 `combat.Dice|NumberPercent|NumberRange|NumberBits` call sites across 13 files (Grep count).
- 91 such calls within `pkg/combat/` itself (confirms the package-scope hook covers combat internals; struct field covers receiver methods).
- Zero existing `testdata/*.golden` files — this phase establishes the pattern.
- Every test file uses stdlib `testing` (no `testify` import in any `_test.go`).
- `CombatSystem` callback-field convention is the universal injection pattern — re-apply for the new `Rand` field without creating a new pattern.

**Pattern extraction date:** 2026-04-17

---

*Phase: 01-golden-master-safety-net*
*Patterns mapped: 2026-04-17*
