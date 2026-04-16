# Research Summary: ROT-MUD Data-Driven Trait System

## Recommended Stack

| Component | Choice | Rationale |
|-----------|--------|-----------|
| Lua VM | `yuin/gopher-lua` v1.1.2 | Pure Go (no cgo), Lua 5.1 (MUD standard), active maintenance, context cancellation for timeouts |
| TOML loader | `pelletier/go-toml/v2` v2.3.0 | Already in go.mod; bump version. Fastest Go TOML lib. |
| Trait storage | Three orthogonal stdlib stores | `uint64` bitmask for capability flags (O(1), zero-alloc); `map[ModifierKey]int` for resist/vuln/stat amounts; `map[HookEvent][]Script` for hooks |
| ECS framework | None | Overkill for ~100 players at 250ms pulses. Slice-of-traits interface matches existing `AffectList` pattern. |

**Critical TOML constraint:** `pelletier/go-toml/v2` does NOT support polymorphic unmarshal. Use homogeneous sections (`[[vulnerabilities]]`, `[[resistances]]`, `[[hooks]]`) NOT a single `[[traits]]` with a `kind` discriminator.

## Table Stakes Features

1. Typed trait structs: Vulnerability, Resistance, Immunity, StatModifier, CapabilityFlag, BehaviorHook
2. Additive composition of race + class traits at entity creation
3. Trait query API: `HasCapability`, `GetModifier`, `ResolveImmunity`, `HooksFor`, `HasTrait`
4. TOML data files for races and classes with traits sections
5. Loader reads and validates trait definitions at startup
6. Combat/magic code queries trait set instead of race/class constant checks
7. Lua VM with OnDeath, OnAttack, OnSpellCast, OnBeforeDamage, OnAfterDamage hooks (5, not 3)
8. Migration parity: existing 19 races + 14 classes expressed as TOML with identical behavior

## Suggested Build Order

1. **Pre-migration plumbing** — Switch player save to name-keyed race/class (CRITICAL: prevents save corruption); build golden-master fixture generator; add forbidden-pattern lint for `ch.Race ==` / `ch.Class ==` outside `pkg/types`
2. **Trait type system** — Closed `TraitKind` enum, parameterized traits, per-axis caps, resolved-trait bitmask cache (TRAIT-01/02/03)
3. **TOML loaders** — Data files for races and classes with validation (DATA-01/02/03)
4. **Character binding** — `ch.Traits []Trait` populated at creation; bitsets expanded from traits (COMBAT-01 partial)
5. **Migration canary** — Vampire + Warrior migrated; `combat_sim_test.go` must pass with identical results (MIGRATE-01 partial)
6. **Full combat/magic refactor** — All ~8 identity-check call sites replaced with trait queries
7. **Lua scripting host** — gopher-lua, single game-loop LState, pcall + instruction limit + context timeout, hand-written API surface (LUA-01/02)
8. **Migrate remaining races + classes** — Data-only; forbidden-pattern lint allowlist shrinks to zero
9. **PROOF-01** — New race added via TOML only, zero Go diff

Phases 1-6 and Phase 7 (Lua) are independent and can run in parallel.

## Top Pitfalls

| # | Pitfall | Phase | Prevention |
|---|---------|-------|------------|
| 1 | **Save-file index shift** — integer race/class ordinals in JSON saves will silently corrupt on reorder | Pre-migration | Migrate saves to name-keyed format BEFORE any data-file reordering |
| 2 | **gopher-lua LState is not goroutine-safe** — game loop has separate connection goroutines | LUA-01 | Single game-loop LState or LStatePool; never call from connection goroutines |
| 3 | **Identity-check leakage** — PROOF-01 fails if `ch.Class == ClassVampire` survives anywhere | All | Forbidden-pattern lint from day 1; shrink allowlist over phases |
| 4 | **Additive stacking blowup** — same risk as acid blast cap; every new trait axis reopens it | TRAIT-02 | Per-axis caps built into composition rules; document limits in data schema |
| 5 | **No golden-master = invisible regressions** — "behavior identical" can't be verified without fixtures | Pre-migration | Build characterization test suite before any migration starts |
| 6 | **Lua sandbox escape** — scripts accessing restricted Go state | LUA-01 | Hand-written API surface only (no gopher-luar); strip stdlib; instruction-count hook |

## Key Decisions for Roadmap

- `ch.Race` / `ch.Class` int indices are **kept** (identity stays integer-indexed, behavior becomes data-driven)
- Traits layer on top of existing Race/Class structs — no struct replacement needed
- Remort class (Lich) trait stacking policy needs explicit decision before MIGRATE-02
- Mob special functions in `pkg/ai/` stay out of scope; need a clear boundary rule to prevent drift
- Hook set: 5 events (OnBeforeDamage, OnAfterDamage, OnDeath, OnSpellCast, OnLevelUp) not 3
