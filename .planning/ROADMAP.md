# Roadmap: ROT-MUD Data-Driven Trait System

## Overview

Transform the ROT-MUD entity systems (races, classes, skills, spells, mob templates, rooms, and items) from hardcoded Go `var` tables and scattered identity checks into a declarative, data-driven trait architecture. The journey starts with a safety net (golden-master tests that capture current behavior across combat, spells, skills, and mob encounters), then builds the trait type system in pure Go, then the TOML loaders for each entity domain (races/classes, skills/spells, mobs) with the Lua scripting host running in parallel, then swaps every race/class/skill/spell identity check in combat/magic/skill/game code for trait queries under CI lint enforcement, then migrates all 19 races, 14 classes, every skill, every spell, and every mob template to data files, then extends the existing area loader with trait parsing and annotates rooms and items (silver weapons, no-magic zones, etc.) with traits, and finally proves extensibility by adding a brand-new race via TOML only — zero Go changes, with a Lua behavior hook.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [ ] **Phase 1: Golden-Master Safety Net** - Capture current combat, spell, skill, and mob behavior in a deterministic parity suite before any migration starts
- [ ] **Phase 2: Trait Type System** - Typed trait structs, additive composition with per-axis caps, and a trait query API in pure Go
- [ ] **Phase 3: Race & Class Loaders** - Homogeneous-section TOML files for races and classes, read and batch-validated at startup
- [ ] **Phase 4: Lua Scripting Host** - Sandboxed gopher-lua VM running five named hook events with hand-written Go API (parallelizable with Phase 3/5/6)
- [ ] **Phase 5: Skills & Spells Loaders** - TOML data files for skills, skill groups, and spells with trait annotations, validated at startup
- [ ] **Phase 6: Mob Type Loaders** - TOML data files for mob type/templates with trait sections, validated at startup
- [ ] **Phase 7: Identity-Check Refactor** - Replace every race/class/skill/spell identity check in combat/magic/skills/game with trait queries; enforce via CI lint
- [ ] **Phase 8: Race & Class Migration** - All 19 races and 14 classes expressed in TOML with verified behavior parity
- [ ] **Phase 9: Skills & Spells Migration** - All skills, skill groups, and spells expressed in TOML with verified behavior parity
- [ ] **Phase 10: Mob Migration** - All mob type templates expressed in TOML with verified behavior parity
- [ ] **Phase 11: Area & Item Traits** - Extend the existing area loader with trait parsing for rooms and items; annotate existing area files with NoMagic zones, silver/fire weapons, etc.
- [ ] **Phase 12: Extensibility Proof** - New race (Lizardman) added by data file only, zero Go diff, with Lua behavior hook

## Phase Details

### Phase 1: Golden-Master Safety Net
**Goal**: Current entity behavior (races, classes, skills, spells, mobs) is captured in a deterministic test fixture that can detect any regression throughout the migration
**Depends on**: Nothing (first phase)
**Requirements**: MIGRATE-06
**Success Criteria** (what must be TRUE):
  1. A golden-master fixture covers all 19 races x 14 classes across representative combat events (hit, damage, resist, immunity, vulnerability)
  2. The fixture covers representative spell casts (damage spells, affect spells, healing) and skill executions (backstab, dodge, parry, kick) with deterministic seeded RNG
  3. The fixture covers mob-template behavior samples (aggro, assist, immunities, special attacks) at representative levels
  4. Running the fixture twice on the unmodified codebase produces byte-identical output; `combat_sim_test.go` integrates the fixture and runs in CI as the parity gate
  5. Any intentional change to entity behavior during later phases produces a visible, diffable fixture failure
**Plans**: TBD

### Phase 2: Trait Type System
**Goal**: A pure-Go trait system exists where entities (races, classes, skills, spells, mobs, rooms, items) can be composed from typed trait values and queried by combat/magic/skill code
**Depends on**: Phase 1
**Requirements**: TRAIT-01, TRAIT-02, TRAIT-03
**Success Criteria** (what must be TRUE):
  1. Typed trait structs exist for Vulnerability, Resistance, Immunity, StatModifier, CapabilityFlag, and BehaviorHook with parameterized fields and a closed `TraitKind` enum
  2. Traits from multiple sources (race + class, or skill + spell effect, or room + item) combine additively at the merge site, with per-axis caps preventing stacking blowup
  3. The trait query API (`HasTrait`, `HasCapability`, `GetModifier`, `ResolveImmunity`, `HooksFor`) is callable from any package and covered by unit tests
  4. A resolved-trait bitmask cache makes `HasCapability` O(1) with zero allocation per query
**Plans**: TBD

### Phase 3: Race & Class Loaders
**Goal**: Races and classes can be declared in TOML files with homogeneous trait sections and are validated and loaded at server startup
**Depends on**: Phase 2
**Requirements**: DATA-01, DATA-02, DATA-06
**Success Criteria** (what must be TRUE):
  1. Race TOML files parse with homogeneous sections (`[[vulnerabilities]]`, `[[resistances]]`, `[[immunities]]`, `[[capabilities]]`, `[[modifiers]]`, `[[hooks]]`) into trait structs
  2. Class TOML files use the same homogeneous section format and parse into the same trait structs; the remort-stacking policy is encoded and documented
  3. The shared loader runs at startup, accumulates every validation error across all entity files, and either prints a batch error report and aborts, or succeeds cleanly
  4. Invalid files (bad section name, unknown trait kind, out-of-range modifier, missing hook target) are rejected with a file+line error message
**Plans**: TBD

### Phase 4: Lua Scripting Host
**Goal**: BehaviorHook traits execute sandboxed Lua scripts on five named game events with bounded CPU and a hand-written Go API
**Depends on**: Phase 2
**Requirements**: LUA-01, LUA-02
**Success Criteria** (what must be TRUE):
  1. `gopher-lua` is embedded with a single game-loop LState (not called from connection goroutines); scripts are pre-compiled to `FunctionProto` at startup, not re-parsed per call
  2. Five hook events fire on the correct triggers: OnBeforeDamage, OnAfterDamage, OnDeath, OnSpellCast, OnLevelUp
  3. Every hook invocation runs inside `pcall` with an instruction-count limit and a context timeout; runaway or erroring scripts are killed without crashing the server
  4. Scripts see only the hand-written Go API surface (no `gopher-luar`, stripped stdlib); a malicious script cannot reach unrelated game state
**Plans**: TBD

### Phase 5: Skills & Spells Loaders
**Goal**: Skills, skill groups, and spells can be declared in TOML files with trait annotations and are validated and loaded at server startup
**Depends on**: Phase 3
**Requirements**: DATA-03, DATA-04
**Success Criteria** (what must be TRUE):
  1. Skill TOML files define name, learn percentages per class, group membership, and trait annotations (capabilities, modifiers, hooks); skill groups are declared separately and referenced by name
  2. Spell TOML files define damage type, target type, effects, magnitude, mana cost, and trait annotations; affects are expressed as structured data, not strings
  3. The loader batch-validates all skill and spell files, rejects unknown trait kinds or bad class/group references, and surfaces file+line error messages
  4. An entity that is both a skill and a spell slot (e.g. combat spells like harm/heal) resolves to a single authoritative definition without duplication
**Plans**: TBD

### Phase 6: Mob Type Loaders
**Goal**: Mob type templates can be declared in TOML files with trait sections and are validated and loaded at server startup
**Depends on**: Phase 3
**Requirements**: DATA-05
**Success Criteria** (what must be TRUE):
  1. Mob-template TOML files define base stats (HP, mana, hit/dam, AC, align, level, race), behavior flags (aggro, assist, sentinel, scavenger), and trait sections
  2. The boundary between mob-template trait data (in scope) and per-mob AI special functions in `pkg/ai/` (out of scope) is documented and enforced; no duplication
  3. The loader batch-validates all mob-template files, rejects unknown trait kinds or bad references, and surfaces file+line error messages
  4. Existing area/room TOML files continue to reference mob templates correctly after the template data source moves from Go tables to TOML
**Plans**: TBD

### Phase 7: Identity-Check Refactor
**Goal**: All race/class/skill/spell identity checks in combat, magic, skill, and game code paths are replaced with trait queries, and a CI lint prevents new ones
**Depends on**: Phase 2
**Requirements**: COMBAT-01, COMBAT-02
**Success Criteria** (what must be TRUE):
  1. Every `ch.Race == RaceX`, `ch.Class == ClassX`, `skill == SkillX`, and `spell == SpellX` identity check in `pkg/combat/`, `pkg/magic/`, `pkg/skills/`, and `pkg/game/` is replaced with a trait query call
  2. The Phase 1 golden-master fixture still passes against the refactored codebase (byte-identical combat/spell/skill simulation output)
  3. A forbidden-pattern lint step in CI fails the build if `ch.Race ==` / `ch.Class ==` / equivalent spell/skill identity checks appear outside `pkg/types/` and `pkg/loader/`
  4. The lint allowlist contains only pre-approved integer-identity call sites with documented justification and shrinks to zero by end of Phase 10
**Plans**: TBD

### Phase 8: Race & Class Migration
**Goal**: All 19 races and 14 classes are defined in TOML data files with behavior identical to the previous hardcoded tables
**Depends on**: Phase 3, Phase 7
**Requirements**: MIGRATE-01, MIGRATE-02
**Success Criteria** (what must be TRUE):
  1. All 19 races (human, elf, dwarf, vampire, etc.) load from TOML at startup with no hardcoded race tables remaining in `races.go`
  2. All 14 classes (mage, cleric, thief, warrior, ranger, and remort classes like Lich and Wizard) load from TOML; remort classes stack their traits additively on tier-1 class traits per the documented policy
  3. The Phase 1 golden-master fixture passes against the fully migrated race/class codebase (byte-identical combat-simulation output)
  4. Player saves continue to load correctly after race/class data moves to TOML (name-keyed save format, not integer-ordinal)
**Plans**: TBD

### Phase 9: Skills & Spells Migration
**Goal**: All skills, skill groups, and spells are defined in TOML data files with behavior identical to the previous hardcoded tables
**Depends on**: Phase 5, Phase 7
**Requirements**: MIGRATE-03, MIGRATE-04
**Success Criteria** (what must be TRUE):
  1. All skills and skill groups load from TOML at startup with no hardcoded skill tables remaining in `pkg/skills/`
  2. All 40+ spells load from TOML at startup with no hardcoded spell tables remaining in `pkg/magic/`
  3. The Phase 1 golden-master fixture passes against the fully migrated skill/spell codebase for all sampled skills, spells, and affect chains
  4. Player skill proficiency and mana/mana-cost tracking continue to work correctly after migration (no save corruption)
**Plans**: TBD

### Phase 10: Mob Migration
**Goal**: All mob type templates are defined in TOML data files with behavior identical to the previous hardcoded tables
**Depends on**: Phase 6, Phase 7
**Requirements**: MIGRATE-05
**Success Criteria** (what must be TRUE):
  1. All mob type templates load from TOML at startup with no hardcoded mob tables remaining outside `pkg/loader/`
  2. The Phase 1 golden-master fixture passes against the fully migrated mob codebase (combat parity for mob encounters at representative levels)
  3. The forbidden-pattern lint allowlist from Phase 7 is empty; zero residual identity checks remain in combat/magic/skills/game
  4. Area/room TOML references to mob templates continue to resolve correctly at world load
**Plans**: TBD

### Phase 11: Area & Item Traits
**Goal**: Room and item definitions in area TOML files support trait annotations, and existing area files are annotated so combat/magic checks (NoMagic zones, silver/fire weapons, etc.) flow through the trait query API instead of hardcoded room/item flag tests
**Depends on**: Phase 7
**Requirements**: AREA-01, AREA-02, AREA-03
**Success Criteria** (what must be TRUE):
  1. The existing area loader parses `[[traits]]`/homogeneous trait sections on room definitions, producing the same trait structs as race/class/skill/spell/mob loaders
  2. The existing area loader parses trait sections on object/item definitions (silver weapon, fire-damage weapon, etc.), composed into the wielder's resolved trait set during combat
  3. Combat and magic code consult the trait query API for room and item checks (NoMagic, Underwater, Dark, SilverWeapon, FireDamage) with zero hardcoded room/item flag identity checks remaining in `pkg/combat/`, `pkg/magic/`, or `pkg/game/`
  4. Existing area files are annotated with the room and item traits that current hardcoded checks rely on; the Phase 1 golden-master fixture passes with byte-identical output for encounters that exercise those rooms and items
**Plans**: TBD
**UI hint**: no

### Phase 12: Extensibility Proof
**Goal**: A brand-new race is added to the game solely by writing a TOML file, proving the core value of the project
**Depends on**: Phase 4, Phase 8, Phase 9, Phase 10, Phase 11
**Requirements**: PROOF-01
**Success Criteria** (what must be TRUE):
  1. A new race (e.g. Lizardman) is added by creating a single TOML file with no Go code changes (`git diff --stat -- '*.go'` is empty for this change)
  2. The new race includes at least one Lua BehaviorHook that fires correctly on its trigger event during live play
  3. A character of the new race can be created, enters combat in a NoMagic room while wielding a silver weapon, and has race/class/skill/spell/room/item traits all compose correctly through the trait query API
  4. The full test suite (including the Phase 1 golden-master parity for existing races, classes, skills, spells, mobs, rooms, and items) still passes after the new race is added
**Plans**: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 1 -> 2 -> 3 -> 4 -> 5 -> 6 -> 7 -> 8 -> 9 -> 10 -> 11 -> 12

**Parallelization opportunities** (parallelization=true in config):
- Phase 4 (Lua Scripting Host) is independent of Phases 3/5/6 and can run in parallel once Phase 2 lands
- Phase 7 (Identity-Check Refactor) is independent of Phases 3/4/5/6 and can start in parallel once Phase 2 lands; it must complete before any migration phase
- Phases 5 (skills/spells loaders) and 6 (mob loaders) can run in parallel after Phase 3 lands
- Phases 8, 9, 10 (migration) can run in parallel after their respective loader phases and Phase 7 land
- Phase 11 (area/item traits) can run in parallel with Phases 8/9/10 once Phase 7 lands (it extends the existing area loader, so it does not need the race/class/skill/spell/mob loader phases)

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Golden-Master Safety Net | 0/TBD | Not started | - |
| 2. Trait Type System | 0/TBD | Not started | - |
| 3. Race & Class Loaders | 0/TBD | Not started | - |
| 4. Lua Scripting Host | 0/TBD | Not started | - |
| 5. Skills & Spells Loaders | 0/TBD | Not started | - |
| 6. Mob Type Loaders | 0/TBD | Not started | - |
| 7. Identity-Check Refactor | 0/TBD | Not started | - |
| 8. Race & Class Migration | 0/TBD | Not started | - |
| 9. Skills & Spells Migration | 0/TBD | Not started | - |
| 10. Mob Migration | 0/TBD | Not started | - |
| 11. Area & Item Traits | 0/TBD | Not started | - |
| 12. Extensibility Proof | 0/TBD | Not started | - |
