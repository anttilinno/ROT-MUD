# Requirements: ROT-MUD Trait System

**Defined:** 2026-04-16
**Core Value:** Any new race or class can be added by writing a data file — zero Go code changes required.

## v1 Requirements

### Trait Types

- [ ] **TRAIT-01**: Typed trait structs exist for Vulnerability, Resistance, Immunity, StatModifier, CapabilityFlag, and BehaviorHook with parameterized fields (e.g. `Vulnerability{DamageType: Silver}` not `VulnerableToSilver`)
- [ ] **TRAIT-02**: Trait composition merges race traits + class traits additively at character creation; per-axis caps prevent stacking blowup
- [ ] **TRAIT-03**: Trait query API — `HasCapability`, `GetModifier`, `ResolveImmunity`, `HooksFor`, `HasTrait` — callable from combat/magic code

### Data Files

- [ ] **DATA-01**: TOML data files define races with homogeneous trait sections (`[[vulnerabilities]]`, `[[resistances]]`, `[[capabilities]]`, `[[hooks]]`)
- [ ] **DATA-02**: TOML data files define classes with the same trait section format
- [ ] **DATA-03**: Loader validates and reads all race/class TOML files at startup with batch error reporting; invalid files prevent startup

### Scripting

- [ ] **LUA-01**: `gopher-lua` VM embedded; BehaviorHook trait runs a named Lua script file on trigger; scripts compile to FunctionProto at startup (not re-parsed per call); sandboxed with instruction-count limit + pcall + context timeout; hand-written Go API surface (no gopher-luar)
- [ ] **LUA-02**: Five hook events supported: OnBeforeDamage, OnAfterDamage, OnDeath, OnSpellCast, OnLevelUp

### Combat Integration

- [ ] **COMBAT-01**: All `ch.Race == RaceX` and `ch.Class == ClassX` identity checks in `pkg/combat/`, `pkg/magic/`, and `pkg/game/` replaced with trait query API calls
- [ ] **COMBAT-02**: Forbidden-pattern lint (CI check) prevents new `ch.Race ==` / `ch.Class ==` checks from being added outside `pkg/types/` and `pkg/loader/`

### Migration

- [ ] **MIGRATE-01**: All 19 existing races migrated to TOML data files; `combat_sim_test.go` passes with identical results (behavior parity)
- [ ] **MIGRATE-02**: All 14 existing classes migrated to TOML data files; remort classes (Lich, Wizard, etc.) stack their traits additively on the tier-1 class traits; parity verified
- [ ] **MIGRATE-03**: Golden-master test suite captures all race × class trait behaviors before migration starts; used to verify parity throughout

### Extensibility Proof

- [ ] **PROOF-01**: A new race (e.g. Lizardman) added to the game via TOML data file only — zero changes to Go source; includes at least one Lua behavior hook

## v2 Requirements

### Enhanced Scripting

- **SCRIPT-01**: Hot-reload of Lua hook scripts without server restart
- **SCRIPT-02**: Lua debug/trace mode for hook development
- **SCRIPT-03**: Hook events for OnLogin, OnLogout, OnLevelUp (second wave)

### Mob Traits

- **MOB-01**: Mob definitions support the same trait system as races/classes
- **MOB-02**: Mob special functions in `pkg/ai/` can be replaced by trait-defined Lua hooks

### Extended Stacking

- **STACK-01**: Item enchantments can contribute traits to the composed trait set
- **STACK-02**: Spell affects can temporarily modify the trait set

## Out of Scope

| Feature | Reason |
|---------|--------|
| Runtime hot-reload of TOML race/class files | Startup-only keeps it simple and safe; hot-reload is v2 |
| Player-created races/classes | Admin-defined data files only; player creation out of scope |
| Trait inheritance hierarchies | Flat additive composition only; inheritance is an LPMud-style footgun |
| GUI/visual trait authoring | Text TOML files only |
| Mob special functions migration | `pkg/ai/` out of scope for this milestone; deferred to MOB-01 |
| Item/spell trait contributions | Complex stacking surface; deferred to STACK-01/02 |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| MIGRATE-03 | Phase 1 | Pending |
| TRAIT-01 | Phase 2 | Pending |
| TRAIT-02 | Phase 2 | Pending |
| TRAIT-03 | Phase 2 | Pending |
| DATA-01 | Phase 3 | Pending |
| DATA-02 | Phase 3 | Pending |
| DATA-03 | Phase 3 | Pending |
| LUA-01 | Phase 4 | Pending |
| LUA-02 | Phase 4 | Pending |
| COMBAT-01 | Phase 5 | Pending |
| COMBAT-02 | Phase 5 | Pending |
| MIGRATE-01 | Phase 6 | Pending |
| MIGRATE-02 | Phase 6 | Pending |
| PROOF-01 | Phase 7 | Pending |

**Coverage:**
- v1 requirements: 14 total
- Mapped to phases: 14
- Unmapped: 0 ✓

---
*Requirements defined: 2026-04-16*
*Last updated: 2026-04-16 after initial definition*
