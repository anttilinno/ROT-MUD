# Requirements: ROT-MUD Trait System

**Defined:** 2026-04-16
**Core Value:** Any new race, class, skill, spell, or mob type can be added by writing a data file — zero Go code changes required.

## v1 Requirements

### Trait Foundation

- [ ] **TRAIT-01**: Typed trait structs exist for Vulnerability, Resistance, Immunity, StatModifier, CapabilityFlag, and BehaviorHook with parameterized fields (e.g. `Vulnerability{DamageType: Silver}` not `VulnerableToSilver`)
- [ ] **TRAIT-02**: Trait composition merges entity traits additively at runtime; per-axis caps prevent stacking blowup
- [ ] **TRAIT-03**: Trait query API — `HasCapability`, `GetModifier`, `ResolveImmunity`, `HooksFor`, `HasTrait` — callable from combat/magic/skill code

### Data Files

- [ ] **DATA-01**: TOML data files define races with homogeneous trait sections (`[[vulnerabilities]]`, `[[resistances]]`, `[[capabilities]]`, `[[hooks]]`, etc.)
- [ ] **DATA-02**: TOML data files define classes with the same trait section format; remort classes stack their traits additively on tier-1 class traits
- [ ] **DATA-03**: TOML data files define skills and skill groups (name, learn percentages, group membership, trait annotations)
- [ ] **DATA-04**: TOML data files define spells (damage type, target type, effects, magnitude, trait annotations)
- [ ] **DATA-05**: TOML data files define mob types/templates (base stats, behavior, trait annotations)
- [ ] **DATA-06**: Loader validates and reads all entity TOML files at startup with batch error reporting; invalid files prevent startup

### Scripting

- [ ] **LUA-01**: `gopher-lua` VM embedded; BehaviorHook trait runs a named Lua script file on trigger; scripts compile to FunctionProto at startup; sandboxed with instruction-count limit + pcall + context timeout; hand-written Go API surface (no gopher-luar)
- [ ] **LUA-02**: Five hook events supported: OnBeforeDamage, OnAfterDamage, OnDeath, OnSpellCast, OnLevelUp

### Combat Integration

- [ ] **COMBAT-01**: All `ch.Race == RaceX`, `ch.Class == ClassX`, and equivalent identity checks in `pkg/combat/`, `pkg/magic/`, `pkg/skills/`, and `pkg/game/` replaced with trait query API calls
- [ ] **COMBAT-02**: Forbidden-pattern lint (CI check) prevents new identity checks from being added outside `pkg/types/` and `pkg/loader/`

### Migration

- [ ] **MIGRATE-01**: All 19 existing races migrated to TOML data files; golden-master suite verifies behavior parity
- [ ] **MIGRATE-02**: All 14 existing classes migrated to TOML data files; remort stacking policy applied; parity verified
- [ ] **MIGRATE-03**: All skills and skill groups migrated to TOML data files; parity verified
- [ ] **MIGRATE-04**: All spells migrated to TOML data files; parity verified
- [ ] **MIGRATE-05**: Mob type templates migrated to TOML data files; parity verified
- [ ] **MIGRATE-06**: Golden-master test suite captures all entity behaviors (combat, spell, skill) before migration starts; used as CI parity gate throughout

### Area Traits

- [ ] **AREA-01**: Room definitions in area TOML files support trait annotations (e.g. `[[traits]]` section with NoMagic, Underwater, Dark capability flags) that affect combat/magic code
- [ ] **AREA-02**: Object/item definitions in area TOML files support trait annotations (e.g. SilverWeapon, FireDamage) that the combat system checks against character vulnerabilities/resistances
- [ ] **AREA-03**: Existing area files annotated with relevant room and item traits (NoMagic zones, silver/fire/etc. weapons); parity with current hardcoded checks verified

### Extensibility Proof

- [ ] **PROOF-01**: A new race (e.g. Lizardman) added to the game via TOML data file only — zero changes to Go source; includes at least one Lua behavior hook

## v2 Requirements

### Enhanced Scripting

- **SCRIPT-01**: Hot-reload of Lua hook scripts without server restart
- **SCRIPT-02**: Lua debug/trace mode for hook development
- **SCRIPT-03**: Additional hook events (OnLogin, OnLogout, OnEquip)

### Extended Stacking

- **STACK-01**: Item enchantments can contribute traits to the composed trait set
- **STACK-02**: Spell affects can temporarily modify the trait set

## Out of Scope

| Feature | Reason |
|---------|--------|
| Runtime hot-reload of TOML files | Startup-only keeps it simple and safe; hot-reload is v2 |
| Player-created races/classes | Admin-defined data files only |
| Trait inheritance hierarchies | Flat additive composition only; inheritance is an LPMud-style footgun |
| GUI/visual trait authoring | Text TOML files only |
| Item enchantment traits | Complex stacking surface; deferred to STACK-01 |
| Individual area mob placements | Already in TOML area files; only mob type templates are in scope |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| MIGRATE-06 | Phase 1 (Golden-Master Safety Net) | Pending |
| TRAIT-01 | Phase 2 (Trait Type System) | Pending |
| TRAIT-02 | Phase 2 (Trait Type System) | Pending |
| TRAIT-03 | Phase 2 (Trait Type System) | Pending |
| DATA-01 | Phase 3 (Race & Class Loaders) | Pending |
| DATA-02 | Phase 3 (Race & Class Loaders) | Pending |
| DATA-06 | Phase 3 (Race & Class Loaders) | Pending |
| LUA-01 | Phase 4 (Lua Scripting Host) | Pending |
| LUA-02 | Phase 4 (Lua Scripting Host) | Pending |
| DATA-03 | Phase 5 (Skills & Spells Loaders) | Pending |
| DATA-04 | Phase 5 (Skills & Spells Loaders) | Pending |
| DATA-05 | Phase 6 (Mob Type Loaders) | Pending |
| COMBAT-01 | Phase 7 (Identity-Check Refactor) | Pending |
| COMBAT-02 | Phase 7 (Identity-Check Refactor) | Pending |
| MIGRATE-01 | Phase 8 (Race & Class Migration) | Pending |
| MIGRATE-02 | Phase 8 (Race & Class Migration) | Pending |
| MIGRATE-03 | Phase 9 (Skills & Spells Migration) | Pending |
| MIGRATE-04 | Phase 9 (Skills & Spells Migration) | Pending |
| MIGRATE-05 | Phase 10 (Mob Migration) | Pending |
| AREA-01 | Phase 11 (Area & Item Traits) | Pending |
| AREA-02 | Phase 11 (Area & Item Traits) | Pending |
| AREA-03 | Phase 11 (Area & Item Traits) | Pending |
| PROOF-01 | Phase 12 (Extensibility Proof) | Pending |

**Coverage:**
- v1 requirements: 23 total
- Mapped to phases: 23
- Unmapped: 0 ✓

---
*Requirements defined: 2026-04-16*
*Last updated: 2026-04-16 — roadmap finalized at 12 phases with Phase 11 (Area & Item Traits) inserted between mob migration and the extensibility proof*
