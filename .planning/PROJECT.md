# ROT-MUD: Data-Driven Trait System

## What This Is

A refactor of the ROT-MUD Go server's race and class system to replace hardcoded struct fields with a declarative, data-driven trait architecture. Races and classes are defined in TOML data files with typed trait annotations (vulnerabilities, resistances, immunities, stat modifiers, capability flags, Lua-scripted behavior hooks). Combat and spell code queries the composed trait set instead of doing race/class equality checks.

## Core Value

Any new race or class can be added by writing a data file — zero Go code changes required.

## Requirements

### Validated

- ✓ Race system with stats, size, XP multipliers, and bonus skills — existing
- ✓ Class system with THAC0, HP/mana gain, guilds, skill groups — existing
- ✓ Combat system with damage calculation and hit/miss checks — existing
- ✓ Magic system with 40+ spells and affect management — existing
- ✓ Skills system with proficiency tracking and improvement — existing

### Active

- [ ] **TRAIT-01**: Typed trait structs — Vulnerability, Resistance, Immunity, StatModifier, CapabilityFlag, BehaviorHook
- [ ] **TRAIT-02**: Additive trait composition — race traits + class traits merged at entity creation
- [ ] **TRAIT-03**: Trait query API — `HasTrait`, `GetModifier`, `HasCapability` on character
- [ ] **DATA-01**: TOML data files for races with traits section
- [ ] **DATA-02**: TOML data files for classes with traits section
- [ ] **DATA-03**: Loader reads trait definitions from files at startup
- [ ] **LUA-01**: Lua VM embedded; BehaviorHook trait runs named script on trigger event
- [ ] **LUA-02**: Hook events: OnDeath, OnAttack, OnSpellCast
- [ ] **COMBAT-01**: Combat damage code queries trait set (VulnerableToSilver, ResistFire, etc.) instead of race/class constants
- [ ] **MIGRATE-01**: All 19 existing races migrated to TOML data files with equivalent traits
- [ ] **MIGRATE-02**: All 14 existing classes migrated to TOML data files with equivalent traits
- [ ] **PROOF-01**: New race (e.g. Vampire) added via data file only — no Go changes

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
