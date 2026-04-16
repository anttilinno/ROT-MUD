# Architecture: Data-Driven Trait System for ROT-MUD

**Domain:** Brownfield refactor of hardcoded race/class tables in a Go MUD (ROM 2.4 lineage) into a declarative, trait-composed, TOML-driven system with Lua behavior hooks.
**Researched:** 2026-04-16
**Confidence:** HIGH on structural choices (evidenced by existing codebase patterns + Go idioms), MEDIUM on Lua integration specifics (verified via gopher-lua docs but untested in this codebase).

---

## Recommended Architecture

### One-Sentence Shape

Introduce a new `pkg/traits` package that owns trait types, composition, and queries. `pkg/loader` gains a TOML-to-traits decoder. `pkg/types.Character` gains a composed `Traits` slice (flat, not hierarchical). Combat/magic gain a thin query surface (`HasCapability`, `GetModifier`, `ResistanceFor`) they call instead of reading `ch.Race`/`ch.Class` directly. Lua is embedded via `gopher-lua` behind a `pkg/scripting` package with a per-goroutine LState pool; behavior-hook traits carry a script name that the scripting package invokes on named events.

### ASCII Diagram

```
                    STARTUP                                  RUNTIME (pulse)
                    =======                                  ==============

  data/races/*.toml                                  pkg/combat/hit.go
  data/classes/*.toml                                pkg/combat/damage.go
         |                                           pkg/magic/spells.go
         v                                                |
  +--------------+     +----------------+                 |  query
  | pkg/loader   |---->|  pkg/traits    |<----------------+
  | (decodes)    |     |  (registry,    |
  +--------------+     |   composer,    |
         |             |   query API)   |<------- types.Character.Traits
         |             +----------------+                 |  (composed at char init)
         |                    ^                           |
         v                    |                           v
  +---------------+      +-----------+              +-----------+
  | types.Race    |      | types.    |              | pkg/      |
  | types.Class   |----->| Character |------------->| scripting |
  | (thin wrapper)|      +-----------+   OnDeath,   | (gopher-  |
  +---------------+            |         OnAttack,  |  lua pool)|
                               |         OnSpellCast+-----------+
                               |              |
                               +<-------------+
                                  hook fires mutate character/combat state
```

### Why This Shape

- **Additive, not replacement (MIGRATE-01/02).** The existing `RaceTable` and `ClassTable` arrays are read by ~150 command handlers and login flows. Ripping them out is a high-churn change. Keeping `Race`/`Class` as thin structs that hold a `Traits` slice means everything that reads `ch.Race` keeps working; only the call sites that do *identity checks* (`== ClassVampire`) get rewritten.
- **Brownfield constraint.** `Character.Imm/Res/Vuln` bitsets already exist (`go/pkg/types/flags.go:361-381`), and `CheckImmune` already routes through them (`go/pkg/combat/combat.go:325-370`). The trait system layers *on top* of those bitsets — on character init, traits expand into the existing flag fields. No breaking change to the damage pipeline.
- **Single-threaded game loop.** The loop is already serialized ("no concurrent access to character/room/object state during game execution" — `codebase/ARCHITECTURE.md:249-252`). The trait query path can be lock-free. Only Lua state pools need concurrency discipline, and only if hooks ever run off the main goroutine (they should not, see Lua section).

---

## Component Boundaries

Each component has one responsibility. Arrows show *allowed* dependencies — anything not listed is forbidden.

| Component | Package | Responsibility | May Import | Must NOT Import |
|-----------|---------|----------------|------------|-----------------|
| **Trait types** | `pkg/traits` | Define `Trait` interface + concrete trait structs (Vulnerability, Resistance, Immunity, StatModifier, CapabilityFlag, BehaviorHook). Expose query API. Compose race+class trait slices into per-character slice. | `pkg/types` (only for `DamageType`, `ImmFlags`, stat indices) | `pkg/combat`, `pkg/magic`, `pkg/loader`, `pkg/scripting` |
| **Trait loader** | `pkg/loader` (extension) | Decode `[[traits]]` TOML tables into concrete `Trait` values via a tag-discriminated union. Validate unknown trait kinds as errors at startup. | `pkg/traits`, `pkg/types`, `go-toml/v2` | `pkg/combat`, `pkg/magic`, `pkg/scripting` |
| **Race/class defs** | `pkg/types` (thin wrappers) | Hold trait slices loaded at startup. `Race.Traits []traits.Trait`, `Class.Traits []traits.Trait`. Keep existing public fields (Name, ShortName, PrimeStat, Thac0_00, etc.) unchanged. | `pkg/traits` | `pkg/loader` (types is foundational — loader populates types, not the reverse) |
| **Character binding** | `pkg/types` (Character extension) | `Character.Traits []traits.Trait` — the *composed* list (race + class, and later affects). Populated by `NewCharacter` / login. Serves as the single query target. | `pkg/traits` | N/A |
| **Scripting host** | `pkg/scripting` (new) | Own `*lua.LState` pool. Register Go API (getCh, dealDamage, sendMsg). Dispatch `Fire(hookName, ctx)` to named Lua script. Swallow Lua errors with slog + continue (MUD must not crash). | `pkg/types`, `gopher-lua` | `pkg/combat`, `pkg/magic` directly — they call *out* to scripting, not the reverse |
| **Combat consumer** | `pkg/combat` | Call `traits.CheckImmunity(ch, damType)`, `traits.GetStatModifier(ch, StatStr)`, `scripting.Fire("OnAttack", ctx)`. Remove `isVampire()`-style race/class switches. | `pkg/traits`, `pkg/scripting` | — |
| **Magic consumer** | `pkg/magic` | Same query surface as combat. | `pkg/traits`, `pkg/scripting` | — |

**The key rule:** `pkg/traits` imports nothing game-specific except `pkg/types` foundational constants. Combat and magic depend on `pkg/traits`, never the reverse. This prevents the accidental cycle that would otherwise appear when a trait needs to "know about" damage resolution.

---

## Data Flow

### Startup (once)

1. `main.go` calls `loader.LoadWorld(cfg.DataPath)`.
2. Loader walks `data/races/*.toml` and `data/classes/*.toml`.
3. For each file, `go-toml/v2` decodes into an intermediate struct with `Traits []rawTrait` where `rawTrait` has `Kind string` plus loose fields.
4. Loader switches on `Kind` and constructs the concrete trait struct (polymorphic TOML pattern — see below).
5. Loader writes results into `types.RaceTable[i].Traits` and `types.ClassTable[i].Traits`.
6. Unknown `kind=` values are startup errors (fail-loud — MUDs cannot afford silent data bugs).

### Character creation / login

1. `server/login.go` or `NewCharacter()` creates a `Character`.
2. After race/class are chosen, call `ch.Traits = traits.Compose(race.Traits, class.Traits)`.
3. `Compose` is a pure function: concatenates both slices, resolves duplicates (last-writer-wins for `StatModifier` of the same stat, union for `CapabilityFlag`, max-severity for immunity/resistance/vulnerability of the same damage type).
4. The composer also *expands* traits into the existing `Character.Imm/Res/Vuln` bitsets so `CheckImmune` keeps working unchanged during migration.

### Runtime query (hot path)

```go
// pkg/combat/combat.go
if traits.HasCapability(victim, traits.CapVulnerableToSilver) && damType == types.DamSilver {
    return ImmVulnerable
}
// replaces the current `isVampire(victim)` block at combat.go:363-367
```

Query functions are O(n) over the trait slice. With ~5-15 traits per character and ~100 queries per combat round across 100 players, this is comfortably below a millisecond. No indexing needed for v1; add a bitmask cache in the composer if profiling flags it later.

### Runtime event (Lua hook)

1. Combat/magic reaches a hook point (e.g. `scripting.Fire(ch, "OnAttack", &Context{Victim: v, Damage: d})`).
2. `scripting.Fire` iterates `ch.Traits`, filters for `BehaviorHook` whose `Event == "OnAttack"`.
3. For each matching hook, borrow an LState from the pool (or, for single-threaded loop, keep one shared LState — see Lua section), load the script by name, push context userdata, `PCall`.
4. On Lua error: log at `slog.Error`, continue. Never propagate the panic.

### Runtime event chain (decoupling)

Combat does not know Lua exists. It only knows `scripting.Fire`. `scripting.Fire` does not know about traits — it receives the event name and the character, then queries `traits.BehaviorHooksFor(ch, event)`. Neither traits nor combat imports scripting internals. This is the "thin interface between stable cores" pattern from the existing callback-driven game loop (`OnViolence`, `OnMobile`, `OnCommand`).

---

## Key Design Decisions

### Decision 1: Slice of Tagged Traits — NOT ECS, NOT Bitset

**Recommendation:** `Character.Traits []traits.Trait` where `Trait` is an interface.

**Options compared:**

| Approach | Query Speed | Extensibility | Memory | Fit for MUD |
|----------|-------------|---------------|--------|-------------|
| Full ECS (archetype/SoA) | Best | Best | Largest | **Overkill.** ECS pays off at 10K+ entities iterated per frame; a MUD has ~100 players + ~1K mobs queried at human reaction speeds (250ms pulses). [Go ECS libraries](https://github.com/kjkrol/goke) target hot inner loops that don't exist here. |
| Bitset flags only | Fastest | Worst | Smallest | **Doesn't fit.** Bitsets model binary presence. `StatModifier{Stat: StatStr, Delta: +2}` and `BehaviorHook{Script: "vampire_drain"}` carry *payloads* — not expressible as bits. The existing `ImmFlags` bitset is the right shape for *one* trait kind (binary resistances) but cannot represent the full system. |
| Slice of `Trait` interface | Good | Best | Small | **Right fit.** Polymorphic, queryable, easy to add new trait kinds (new struct + loader case). Matches the existing `Affected AffectList` pattern on `Character` — affects are already a heterogeneous list handled identically. |

**Evidence:**
- Character already has `Affected AffectList` — a flat list of heterogeneous modifiers iterated per-tick. Traits mirror this exactly, differing only in lifetime (permanent vs. timed).
- Benchmarks on Go ECS libraries ([GOKe](https://github.com/kjkrol/goke)) emphasize cache-friendly iteration at scale — irrelevant at MUD scale and inconsistent with brownfield integration.
- A MUD pulse is 250ms. Even a deliberately slow query is invisible here.

**Trait interface sketch:**

```go
// pkg/traits/trait.go
type Trait interface {
    Kind() TraitKind  // sentinel for fast filtering; not strictly required
}

type Vulnerability struct {
    DamType types.DamageType
    Factor  int  // 150 = +50% damage; leaves room for graduated vulnerability
}
func (v Vulnerability) Kind() TraitKind { return KindVulnerability }

type StatModifier struct {
    Stat  int  // StatStr, StatInt, ...
    Delta int
}
// ... Immunity, Resistance, CapabilityFlag, BehaviorHook similarly
```

### Decision 2: Polymorphic TOML via Tag-Discriminated Decode

**Recommendation:** Two-pass decoding. Pass 1: decode `[[traits]]` into a `rawTrait` with `Kind string` and generic fields. Pass 2: a switch on `Kind` constructs the concrete trait.

**Why not other approaches:**

- `go-toml/v2` does not support `UnmarshalTOML`-style hooks with sufficient power for true polymorphism. Trying to force one `Trait` interface through TOML's reflective decoder requires reinventing `encoding/json`'s `json.RawMessage` pattern ([similar problem in JSON discussed here](https://alexkappa.medium.com/json-polymorphism-in-go-4cade1e58ed1)).
- Separate sections per trait kind (`[vulnerabilities]`, `[resistances]`, `[stat_modifiers]`) works but scatters a single race's definition across many named tables. Poor ergonomics for authors.
- One `[[traits]]` array with a `kind` discriminator is the idiomatic shape — same pattern TOML-consuming game tools use ([discriminated union pattern in Go](https://danielmschmidt.de/posts/2024-07-22-discriminated-union-pattern-in-go/)).

**Example TOML (vampire race):**

```toml
name = "vampire"
short_name = "Vampr"
points = 10
base_stats = [15, 13, 10, 14, 16]
max_stats = [20, 18, 15, 19, 22]
size = "medium"

[[traits]]
kind = "vulnerability"
dam_type = "fire"
factor = 200

[[traits]]
kind = "vulnerability"
dam_type = "silver"
factor = 150

[[traits]]
kind = "resistance"
dam_type = "cold"

[[traits]]
kind = "capability"
flag = "see_in_dark"

[[traits]]
kind = "stat_modifier"
stat = "str"
delta = 2

[[traits]]
kind = "behavior_hook"
event = "OnDeath"
script = "vampire_ashes"
```

**Loader sketch:**

```go
type rawTrait struct {
    Kind    string `toml:"kind"`
    DamType string `toml:"dam_type,omitempty"`
    Factor  int    `toml:"factor,omitempty"`
    Flag    string `toml:"flag,omitempty"`
    Stat    string `toml:"stat,omitempty"`
    Delta   int    `toml:"delta,omitempty"`
    Event   string `toml:"event,omitempty"`
    Script  string `toml:"script,omitempty"`
}

func decodeTrait(r rawTrait) (traits.Trait, error) {
    switch r.Kind {
    case "vulnerability":
        return traits.Vulnerability{DamType: parseDamType(r.DamType), Factor: r.Factor}, nil
    case "resistance":
        return traits.Resistance{DamType: parseDamType(r.DamType)}, nil
    // ... etc
    default:
        return nil, fmt.Errorf("unknown trait kind %q", r.Kind)
    }
}
```

### Decision 3: Lua via `gopher-lua` — Single-Loop LState (Not Pool)

**Recommendation:** Use [`github.com/yuin/gopher-lua`](https://github.com/yuin/gopher-lua). Keep *one* `*lua.LState` owned by the game loop goroutine. Do not pool until you have a concrete reason.

**Evidence:**
- The game loop is single-threaded by design. Commands, combat, magic, AI all execute on the loop goroutine. A hook fired from combat runs on the loop goroutine. One LState is sufficient and eliminates the "LState is not goroutine-safe" failure mode called out [by the gopher-lua author](https://github.com/yuin/gopher-lua/issues/5).
- `gopher-lua` outperforms `goja` (JavaScript) on equivalent microbenchmarks (~95µs vs ~281µs on Fibonacci — [gopher-lua README](https://github.com/yuin/gopher-lua)). Lua is also the genre-standard scripting language for MUDs, matching what content authors will expect.
- Compile each hook script *once* at startup into a `*lua.FunctionProto`, then `L.Push(L.NewFunctionFromProto(fp))` per invocation. This avoids re-parsing on every hook fire — the main performance trap with embedded Lua.

**Lua API surface (minimal, v1):**

```
ch = scripting.getCharacter(id)          -- userdata wrapper around *types.Character
ch.hp, ch.max_hp, ch.name, ch.level      -- readable fields
ch:send("You burst into ash.")           -- output
ch:damage(amount, "fire")                -- delegated back to combat via callback
room = ch.room
room:broadcast("...")
```

The Go-side of this API is a handful of registered functions in `pkg/scripting/lua_bindings.go`. Keep it *deliberately small* in v1 — expand only when a hook cannot be written without it. This prevents the scripting layer from accidentally becoming a second command system.

### Decision 4: Keep `Race` / `Class` — Do Not Replace

**Recommendation:** Keep `types.Race` and `types.Class` structs. Add `Traits []traits.Trait` field. Eventually, migrate fields like `ClassMultiplier`, `Thac0_00`, `HPMin` into `StatModifier`-style traits, but *not in the first phase*.

**Why:**
- `ch.Race` (an `int` index) is read in login (`server/login.go:767,790`), combat tests, skill gates (`commands_skills.go:646,694`), and the simulation test suite. These are *identity* lookups, not trait queries. Breaking the index model cascades into invalidating every player save file (`Race` is persisted as an integer).
- The project's stated value (`PROJECT.md:9`) is "Any new race or class can be added by writing a data file." This requires the *trait* layer be data-driven, not that the `Race` Go struct disappear. Keeping the struct preserves the stable `int`-index identity while the trait slice carries the behavior.
- Incremental migration: phase 1 adds `Traits []Trait` and *leaves* `ClassMultiplier`, `Thac0_00`, etc. where they are. Phase N (post-milestone) can fold those into traits once combat queries are fluent.

### Decision 5: Additive Composition with Deterministic Merge Rules

Specified in `PROJECT.md:61` but architectural implications:

| Trait kind | Duplicate resolution |
|------------|----------------------|
| `Vulnerability`/`Resistance`/`Immunity` on same `DamType` | Take highest severity (Immune > Resistant > Normal > Vulnerable). This matches existing `CheckImmune` ordering. |
| `StatModifier` on same `Stat` | Sum deltas. |
| `CapabilityFlag` | Set union (presence is boolean). |
| `BehaviorHook` on same `Event` | Keep both — fire in order: race hooks then class hooks. |

These rules live in `traits.Compose()` — one place, tested exhaustively in unit tests.

---

## Integration with Existing Combat/Magic

### Pattern: Query at Decision Points, Don't Walk Traits Inline

**Good:**
```go
// pkg/combat/combat.go - refactored CheckImmune
immStatus := traits.ResolveImmunity(victim, damType)  // single call
if immStatus == traits.Immune { return ImmImmune }
```

**Bad:**
```go
// DO NOT scatter trait iteration through combat
for _, t := range victim.Traits {
    if v, ok := t.(traits.Vulnerability); ok && v.DamType == damType { ... }
}
```

The query API (`HasTrait`, `GetModifier`, `HasCapability`, `ResolveImmunity`, `HooksFor`) is the *entire contract* between combat/magic and the trait system. Combat imports `pkg/traits` and calls these functions; it never type-asserts on trait values.

### Migration Sites (concrete, from grep)

| File:Line | Current code | Replace with |
|-----------|--------------|--------------|
| `combat/combat.go:321` | `ch.Class == types.ClassVampire \|\| ch.Class == types.ClassLich` | `traits.HasCapability(ch, traits.CapUndead)` |
| `combat/combat.go:363-367` | `if isVampire(victim) && damType == DamFire/DamSilver` | Remove entirely — vampire traits declare `Vulnerability{DamFire}`, `Vulnerability{DamSilver}` |
| `combat/hit.go:36-39` | silver weapon vampire-vuln check | Same — handled by trait resolution |
| `game/commands_skills.go:646,694` | `ch.Class == Warrior \|\| Thief` | `traits.HasCapability(ch, traits.CapMartial)` |
| `server/login.go:767,790` | class index comparison | *Keep as-is* — identity, not behavior |

All MOB-side resistances (`Imm`/`Res`/`Vuln` bitsets populated by `MobileData.ImmFlags`/`ResFlags`/`VulnFlags` at `loader/schema.go:120-122`) keep working unchanged. This is important — trait system covers *races and classes* only per `PROJECT.md:56`, and mob flags stay bitset.

---

## Suggested Build Order (Dependency Graph)

Each arrow is "must exist before." Phases in the roadmap should follow this topology.

```
(1) Trait types + query API        (pure Go, no I/O)
        |
        v
(2) Composition + merge rules      (depends on (1); unit-testable in isolation)
        |
        v
(3) TOML loader extension          (depends on (1)+(2); reads data/races/*.toml,
        |                            classes/*.toml)
        |
        v
(4) Character binding              (ch.Traits populated on creation/login;
        |                            composer expands to existing Imm/Res/Vuln)
        |
        v
(5) Migrate ONE race + ONE class   (PROOF point: vampire race + warrior class in
        |                            TOML; existing behavior preserved; test parity
        |                            against current hardcoded tables via the
        |                            combat_sim_test.go harness)
        |
        v
(6) Refactor combat/magic call     (swap the ~5-8 identity-check sites identified
        | sites to trait queries    above; combat_sim_test.go must still pass with
        |                            same win%/damage curves from MEMORY.md)
        |
        v
(7) Scripting host (gopher-lua)    (independent track — can start in parallel with
        |                            (1-4), merge before (8))
        |
        v
(8) BehaviorHook trait + events    (OnDeath, OnAttack, OnSpellCast wired into
        |                            combat/magic; one real scripted behavior like
        |                            vampire ashes-on-death as proof)
        |
        v
(9) Migrate remaining 18 races +   (data-only work; no Go changes expected —
         13 classes                  this is the test of the whole system)
        |
        v
(10) PROOF-01: add new race from   (success criterion: zero Go diff)
     TOML alone
```

### Critical Ordering Notes

- **(2) must come before (3).** The loader needs to call `Compose` on the loaded traits to produce the final race/class `Traits` slice. Composition rules must be locked in before data authors see the format — changing merge semantics later invalidates data files.
- **(5) is the linchpin.** Running the existing `combat_sim_test.go` (mentioned in `MEMORY.md` as the tuning harness) against a TOML-driven vampire and getting identical damage curves is the gate for the whole refactor. If parity fails, the composer or the `Imm/Res/Vuln` expansion is wrong. Do not proceed to (6) until parity holds.
- **(7) can parallelize with (1-6).** Scripting does not touch traits until (8). A separate phase can establish `pkg/scripting` with a test script (e.g. `hello.lua` triggered from a debug command) independent of the trait work.
- **(9) is data-only.** If any of the 18 remaining races cannot be expressed as traits, the trait types in (1) are incomplete — reopen (1), extend, retest. This is the "new trait kind" feedback loop and should be expected for ~1-2 races (Heucuva and Titan are likely candidates given their unusual stat ranges).
- **(10) has no code deliverable.** It is a *test* of the system: can a developer add a "lizardman" race with `[[traits]]` entries and nothing else and have it appear in character creation with correct mechanics? If yes, the system meets its core value.

### What Must NOT Be Done Early

- Do not touch `login.go`'s class-index switches. They are identity lookups, not behavior queries. Leave them alone through phase (6).
- Do not migrate `ClassMultiplier` (XP multipliers) into traits in this milestone. That is a second-order refactor — the trait system must prove itself on the damage pipeline first.
- Do not try to hot-reload traits. Explicitly out of scope (`PROJECT.md:38`). Startup-only loading simplifies (4) enormously — no invalidation, no cache consistency.

---

## Data Flow Direction — Explicit Summary

1. **Source of truth → memory.** TOML files → loader → `RaceTable[i].Traits` + `ClassTable[i].Traits`. Read-only after startup.
2. **Template → instance.** At character creation, `Compose(race.Traits, class.Traits)` → `Character.Traits` + populated `Character.Imm/Res/Vuln` bitsets.
3. **Runtime query is read-only.** Combat/magic/skills call `traits.HasX(ch, ...)`. Nothing writes back to `ch.Traits` in v1. (Future dynamic traits from spells would go via affects, which already exist.)
4. **Hook dispatch is push.** Combat/magic → `scripting.Fire(ch, event, ctx)` → iterate `BehaviorHook` traits → Lua. Lua writes back to character state only through the registered Go API, which reuses the same combat/magic entrypoints.

No circular dependencies. No mutable shared state beyond the character itself. Matches the existing "modify in place, single-threaded game loop" pattern from `codebase/ARCHITECTURE.md:249-252`.

---

## Sources

- Existing codebase: `go/pkg/types/races.go`, `go/pkg/types/classes.go`, `go/pkg/combat/combat.go:315-370`, `go/pkg/combat/hit.go:36-39`, `go/pkg/magic/spells.go:850-865`, `go/pkg/loader/schema.go:120-140`, `go/pkg/types/flags.go:361-381`. **HIGH** — inspected directly.
- [gopher-lua README and goroutine safety issue](https://github.com/yuin/gopher-lua) — LState is not goroutine-safe; use one per goroutine or pool. **HIGH**.
- [gopher-lua issue #5 on concurrency](https://github.com/yuin/gopher-lua/issues/5) — confirms LState pooling pattern. **HIGH**.
- [gopher-lua vs goja Fibonacci benchmark](https://scriggo.com/benchmarks) — gopher-lua ~3x faster on microbenchmark. **MEDIUM** (benchmark, not MUD-specific).
- [Daniel Schmidt: Discriminated Union Pattern in Go](https://danielmschmidt.de/posts/2024-07-22-discriminated-union-pattern-in-go/) — two-pass decode via `type` tag. **MEDIUM**.
- [Alex Kalyvitis: JSON polymorphism in Go](https://alexkappa.medium.com/json-polymorphism-in-go-4cade1e58ed1) — the pattern translates directly to TOML. **MEDIUM**.
- [GOKe — zero-allocation ECS for Go](https://github.com/kjkrol/goke) — ECS targets scales (10K+ entities) not relevant here. **HIGH** confidence on "don't use ECS" conclusion.
- `PROJECT.md`, `codebase/ARCHITECTURE.md`, `codebase/STACK.md` — authoritative for project scope and existing patterns. **HIGH**.

---

## Open Questions for Downstream Phases

- **Affect integration.** When a spell temporarily grants `Resistance{Fire}`, does it add a `Trait` to `ch.Traits`, or does it go through the existing `Affected` list? Recommendation: existing `Affected` list, since it has timing and removal infrastructure. Trait slice stays immutable per-character. *Defer decision to spell-system phase.*
- **Trait query caching.** At scale, may want a precomputed capability bitmask in the character alongside `Traits`. Recommendation: profile first, optimize second. *Defer until observed hot path.*
- **Lua script storage.** One file per hook (`data/scripts/vampire_ashes.lua`) or bundled per race? Recommendation: one-file-per-hook for editability. *Decide during phase (7).*
- **Mob trait support.** Out of scope per `PROJECT.md:56`, but the trait types are reusable. *Revisit in a later milestone.*
