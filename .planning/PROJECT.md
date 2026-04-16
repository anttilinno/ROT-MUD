# ROT-MUD: Data-Driven Trait System

## What This Is

A full data-driven overhaul of the ROT-MUD Go server's game entity systems. Races, classes, skills, spells, and mobs are all moved from hardcoded Go struct tables to declarative TOML data files with typed trait annotations (vulnerabilities, resistances, immunities, stat modifiers, capability flags, Lua-scripted behavior hooks). Combat, magic, and skill code queries the composed trait set instead of doing identity checks. The game becomes a data-driven engine where content lives in files, not code.

## Core Value

Any new race, class, skill, spell, or mob type can be added by writing a data file — zero Go code changes required.

## Requirements

### Validated

- ✓ Race system with stats, size, XP multipliers, and bonus skills — existing
- ✓ Class system with THAC0, HP/mana gain, guilds, skill groups — existing
- ✓ Combat system with damage calculation and hit/miss checks — existing
- ✓ Magic system with 40+ spells and affect management — existing
- ✓ Skills system with proficiency tracking and improvement — existing

### Active

- [ ] **TRAIT-01**: Typed trait structs — Vulnerability, Resistance, Immunity, StatModifier, CapabilityFlag, BehaviorHook
- [ ] **TRAIT-02**: Additive trait composition — entity traits merged at runtime; per-axis caps prevent blowup
- [ ] **TRAIT-03**: Trait query API — `HasTrait`, `GetModifier`, `HasCapability`, `ResolveImmunity`, `HooksFor`
- [ ] **DATA-01**: TOML data files for races with homogeneous trait sections
- [ ] **DATA-02**: TOML data files for classes with homogeneous trait sections; remort classes stack additively on tier-1
- [ ] **DATA-03**: TOML data files for skills and skill groups with trait sections
- [ ] **DATA-04**: TOML data files for spells with trait sections (damage type, target, effects)
- [ ] **DATA-05**: TOML data files for mob types/templates with trait sections
- [ ] **DATA-06**: Loader validates and reads all entity TOML files at startup with batch error reporting
- [ ] **LUA-01**: gopher-lua VM embedded; BehaviorHook runs sandboxed script with instruction-count + context timeout
- [ ] **LUA-02**: Five hook events: OnBeforeDamage, OnAfterDamage, OnDeath, OnSpellCast, OnLevelUp
- [ ] **COMBAT-01**: All identity checks (`ch.Race ==`, `ch.Class ==`, etc.) in combat/magic/game replaced with trait queries
- [ ] **COMBAT-02**: Forbidden-pattern lint (CI) prevents new identity checks outside `pkg/types/` and `pkg/loader/`
- [ ] **MIGRATE-01**: All 19 races migrated to TOML; parity verified by golden-master suite
- [ ] **MIGRATE-02**: All 14 classes migrated to TOML; parity verified
- [ ] **MIGRATE-03**: All skills and skill groups migrated to TOML; parity verified
- [ ] **MIGRATE-04**: All spells migrated to TOML; parity verified
- [ ] **MIGRATE-05**: Mob type templates migrated to TOML; parity verified
- [ ] **MIGRATE-06**: Golden-master test suite captures all entity behaviors before migration starts
- [ ] **PROOF-01**: New race added via TOML only — zero Go changes; includes Lua behavior hook

### Out of Scope

- Hot-reloading trait files at runtime — startup-only loading keeps it simple
- Player-created races/classes — admin-defined data files only
- Trait inheritance hierarchies — flat additive composition only
- Visual scripting or GUI for trait authoring — text TOML files only

## Context

ROT-MUD is a Go port of a classic C MUD server (ROM 2.4 lineage). The existing race and class definitions live in `go/pkg/types/races.go` and `go/pkg/types/classes.go` as Go `var` tables with hardcoded struct fields. Combat checks like `if ch.Race == RaceVampire` are scattered across `pkg/combat/` and `pkg/magic/`.

The codebase already uses TOML for world data (rooms, mobs, objects). Extending that pattern to races/classes with a trait section is consistent with the existing data-loading architecture in `pkg/loader/`.

Lua is the standard MUD scripting language and fits the hook use case — small scripts with access to character/combat context, not full game logic.

## Constraints

- **Tech Stack**: Go — no new compiled language dependencies except a Lua VM library (gopher-lua or goja)
- **Compatibility**: Migrated races/classes must behave identically to current hardcoded definitions
- **Data Format**: TOML — consistent with existing world data files
- **Scope**: Trait system covers races and classes only; mob special functions stay in `pkg/ai/`

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Additive trait stacking (race + class both apply) | Simplest model; no precedence rules needed | — Pending |
| BehaviorHooks scripted in Lua | Standard MUD scripting; separates logic from Go code | — Pending |
| TOML data files for trait definitions | Consistent with existing world data loading pattern | — Pending |
| Trait query API instead of race/class constant checks | Eliminates scattered identity checks throughout combat/magic | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-04-16 after initialization*
