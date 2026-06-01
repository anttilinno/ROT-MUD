# Trait System Migration — Dependency Trace

> Derived from the graphify knowledge graph (`graphify-out/graph.json`, 3016 nodes / 6411 edges).
> Traces the spine from the Lua concurrency constraint through the trait type system,
> the trait query API, and the Phase 08 migration parity gate.

## The spine

```
Lua goroutine constraint ──► Trait Type System (P02) ──► Trait Query API ──► Migration Parity Gate (P08)
```

Four findings, each grounded in graph edges and source locations.

---

## 1. Lua concurrency constraint shapes the behavior-hook contract

The extractor linked the existing mob-special context to the planned Lua VM safety note:

```
specCastMage / dragonBreath / specThief / ... (20 spec fns)
        │ all reference (EXTRACTED)
        ▼
SpecialContext ──semantically_similar_to (INFERRED 0.65)──► concept_lua_lstate_safety
(pkg/ai/specials.go:16)                                     (.planning/research/STACK.md)
        ▲ created by
AISystem.createContext (pkg/ai/system.go)
        ▲ called from
GameLoop.pulse → RunWithContext (pkg/game/loop.go:137, :749)  [single-threaded pulse loop]
```

- `SpecialContext` is the bundle every mob special receives (`Character`, `MagicSystem`, `Direction`, target), built once per call, consumed inside the pulse.
- gopher-lua `*LState` is **not goroutine-safe** — flagged as a Phase 04 blocker (`state_lstate_blocker`, `.planning/STATE.md`).
- **Implication:** Phase 04 Lua trait hooks must run inside the same single pulse-loop goroutine that `SpecialContext` already runs in. The codebase already solved the concurrency problem for specials by living entirely in the pulse loop; the Lua layer inherits that execution model. One LState pinned to the loop goroutine, synchronous mutation of `Character` in place.

---

## 2. Phase 02 → Phase 04: trait types feed the hooks via the trait query API

Roadmap dependency chain (each phase references its prerequisite):

```
P01 Golden Master ◄── P02 Trait Type System ◄──┬── P03 Race/Class Loaders ◄── P05 Skills/Spells, P06 Mob Loaders
                                                ├── P04 Lua Scripting Host
                                                └── P07 Identity-Check Refactor ◄── P08 Migration ◄── P12 Extensibility Proof
```

Requirements + rationale (the WHY nodes):

| Requirement | Target | Rationale node |
|---|---|---|
| `requirements_trait_01` | → P02 trait type system | `project_trait_system` |
| `requirements_trait_03` | trait query API | `project_trait_query_api` |
| `requirements_trait_02` | additive stacking | `project_additive_stacking` |
| `requirements_lua_01` | → P04 Lua host | `project_lua_behaviorhooks` |
| `requirements_combat_01` | → P07 identity refactor | — |

- **Trait query API is the universal read surface.** Combat, magic, and Lua hooks all query the *composed* trait set instead of doing identity checks.
- P04 references P02 → a hook with nothing to query is useless; trait types must exist first.
- `project_additive_stacking` → composition is additive so a hook *adds* a trait without clobbering the race/class base (mirrors the existing `mayorState` additive-mutation style).
- **Cross-cutting surprise:** `economy_e3_crafting → project_trait_query_api`. The economy overhaul (Phase 13) consumes the same API — crafting reads item/material traits through it. The trait query API is load-bearing well beyond races/classes.

Code that Phase 07 will replace (graph-confirmed identity checks):
- `combat.checkImmune → combat.isVampire` — hardcoded identity check `requirements_combat_01` removes.
- `magic.checkDamageResist → DamageType` — resistance logic Phase 02 makes data-driven.
- `types/races.go` (`Race`, `GetRace`, `RaceByName`) + `classes.go` — the hardcoded tables Phase 08 migrates to TOML.

---

## 3. Phase 08 migration — the golden parity contract

What the golden gate freezes:

```
TestGolden → runFixture → 9 scenario emitters
  (RaceWarriorCombo, ClassHumanCombo, MobImmunityScenario,
   Backstab, Kick, Defense, Spell, MobAggro, MobCaster)
        │ all call
   makePlayer                                  makeMob
     ├─ racestatatlevel  (HP/mana per race×level)   ├─ mobhp
     ├─ classequipac     (class starting AC)        └─ NewNPC
     ├─ playerhp / playermana → racestatatlevel
     ├─ makeweapon → weapontypeforclass, weapondice
     └─ NewCharacter
   formatImmBits → joinStrings   (immunity/resist/vuln bitstrings)
```

Must reproduce byte-identical after migration:

1. **`racestatatlevel`** — per-race HP/mana curve from `RaceTable`. Highest-risk node; every player fixture routes through it. Off-by-1 HP at any level → divergence → fail.
2. **`classequipac` + `weapontypeforclass` + `weapondice`** — class starting AC + weapon from `ClassTable`.
3. **`formatImmBits` / immunity scenario** — vuln/resist/immune bitfields. Phase 02 trait types replace the hardcoded bits; migration must compose to the *same* bitstring. Same output as the `checkImmune → isVampire` path, frozen before Phase 07 removes it.

**Determinism seam:** `pkg/golden → SetRand` (EXTRACTED). The 01-01 plan added an RNG seeding hook so combat/dice are reproducible — the load-bearing prerequisite that lets Phase 01 land before any migration phase (roadmap: P02 references P01).

**Migration trap:** `pkg/golden` calls into `combat_system`, `magic_system`, `skills_system` — fixtures exercise the real code, not stubs. Phase 08 can't just swap the data source; the composed trait set fed into combat must produce identical THAC0, AC, damage, and immunity results.

**Sequencing the graph enforces:**
- P08 references P07 → migrate data *after* combat queries traits, not before. Migrating into a system still doing `isVampire` checks guarantees a parity break.
- P12 (extensibility proof) references P08 → "add a race via TOML only" proof can't run until migration is green.

**Coverage gap:** `requirements_migrate_06` / `concept_migrate06` (mob-template golden coverage) is a separate snapshot from the player fixtures. `GetRace`/`GetClass` are shared by mobs (called from `cmdScore`, `cmdWho`, `manaGain`, `getThac0`), so mob fixtures can break even when player fixtures pass.

---

## Bottom line

Phase 08 must keep `racestatatlevel`, `classequipac`, weapon selection, and immunity bits producing identical values, with `SetRand` holding RNG fixed, **while** combat already reads through the trait query API (Phase 07). Break any one → golden diff → gate red.

The dependency order is not arbitrary: the Lua goroutine constraint forces single-loop execution, the trait type system must precede the hooks that query it, identity-check refactor must precede data migration, and the golden master must precede all of it to make parity testable.
