# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-04-16)

**Core value:** Any new race, class, skill, spell, or mob type can be added by writing a data file — zero Go code changes required.
**Current focus:** Phase 1 - Golden-Master Safety Net

## Current Position

Phase: 1 of 12 (Golden-Master Safety Net)
Plan: 0 of TBD in current phase
Status: Ready to plan
Last activity: 2026-04-16 — Roadmap finalized at 12 phases covering 23 v1 requirements (races, classes, skills, spells, mobs, rooms, items)

Progress: [░░░░░░░░░░] 0%

## Performance Metrics

**Velocity:**
- Total plans completed: 0
- Average duration: n/a
- Total execution time: 0.0 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| - | - | - | - |

**Recent Trend:**
- Last 5 plans: n/a
- Trend: n/a

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- Additive trait stacking (race + class + skill/spell effects + room + item all apply) — pending validation during Phase 2
- BehaviorHooks scripted in Lua (gopher-lua, hand-written API, single game-loop LState) — pending validation during Phase 4
- TOML data files with homogeneous sections (no polymorphic unmarshal via pelletier/go-toml/v2) — locked by Phase 3
- Trait query API replaces identity checks across combat/magic/skills/game; CI lint enforces — locked by Phase 7
- Area/item traits extend the existing area loader rather than introducing a new loader — locked by Phase 11

### Pending Todos

None yet.

### Blockers/Concerns

- Save-file format: integer race/class/skill/spell ordinals in JSON saves will silently corrupt on reorder. Research flagged migrating saves to name-keyed format as pre-migration work. Resolve before Phase 8/9 reorders any data.
- Remort class trait stacking policy (Lich, Wizard, etc.): explicit decision required before MIGRATE-02 (Phase 8).
- gopher-lua LState is not goroutine-safe: enforce single game-loop LState or LStatePool during Phase 4.
- Mob-template boundary vs `pkg/ai/` special functions: boundary rule must be documented during Phase 6 to prevent drift.
- Room/item trait boundary: Phase 11 must inventory which current hardcoded room/item flag checks become trait queries vs. which stay in the core engine (e.g. light/dark rendering is not a trait).

## Deferred Items

Items acknowledged and carried forward from previous milestone close:

| Category | Item | Status | Deferred At |
|----------|------|--------|-------------|
| *(none)* | | | |

## Session Continuity

Last session: 2026-04-16
Stopped at: Roadmap finalized at 12 phases with Phase 11 (Area & Item Traits) inserted; STATE.md and REQUIREMENTS.md traceability updated for full-scope overhaul including rooms and items
Resume file: None
