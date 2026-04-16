# Feature Landscape: Data-Driven Trait System

**Domain:** MUD race/class trait/decorator system (ROT-MUD, Go, ROM 2.4 lineage)
**Researched:** 2026-04-16
**Overall confidence:** HIGH for table stakes (verified in existing codebase and ROM/Diku tradition), MEDIUM for hook list and guard rails (MUD scripting norms + general RPG practice).

---

## Context from Codebase

The existing ROT-MUD codebase already has the raw materials for a trait system — they are simply tangled together:

- `types.Character` carries `Imm`, `Res`, `Vuln` bitflag fields (ImmFlags) — immunity/resist/vuln already data-driven **per-character**, just not per-race/class.
- `combat.CheckImmune` already does the immune → resist → vuln → normal resolution order, plus a hardcoded fallback `if isVampire(victim) { fire/silver → vulnerable }`. This is the canonical example of what the trait system must replace.
- Only **5 hardcoded Race/Class equality checks** exist outside `types/` (combat.go, login.go, commands_skills.go) — the refactor surface is narrow.
- `ActVampire`, `ActUndead`, `ActRanger`, `ActMage`, etc. on NPCs already function as capability flags — the trait system generalizes this for PCs too.
- `Affect` system (40+ spells, duration/tick) is a separate concern — traits are **permanent/innate** modifiers, Affects are **temporary** modifiers. Don't merge them.

This means the feature list below is mostly about **factoring** existing behaviour, not inventing new mechanics. That drives the categorization: table stakes = "what's in the code today, in a different shape."

---

## Table Stakes

Features that must exist in v1 or the system fails PROOF-01 ("add a Vampire via data file only").

### TS-1: Typed Vulnerability / Resistance / Immunity traits

**What:** Per-trait declarations of the form `{kind: "vulnerability", damage_type: "fire"}` that compose additively into `Character.Imm/Res/Vuln` bitflags at entity creation.
**Why expected:** ROM/Diku canon (`IMM_FIRE`, `RES_COLD`, `VULN_SILVER`) and already wired into `CheckImmune`. Without this, vampire/lich/heucuva are wrong.
**Complexity:** Low — flags already exist, just need a loader that ORs them in.
**Depends on:** None.
**Replaces:** The hardcoded `isVampire()` fire/silver branch in `combat.go:361-367`.

### TS-2: Passive stat modifiers

**What:** Additive modifiers to the six core stats (Str/Int/Wis/Dex/Con + derived: MaxHP, MaxMana, HitRoll, DamRoll, AC, SavingThrow). Trait form: `{kind: "stat_mod", stat: "str", value: 2}`.
**Why expected:** Every existing race has `BaseStats[]` and `MaxStats[]`; every class has `HPMin/HPMax/ManaGain`. Current system is "set absolute values"; trait system needs "apply deltas to a baseline." Giants need +Str, Pixies +Dex, etc.
**Complexity:** Medium — must decide the base (neutral human) and convert the 19 race tables to deltas. Easy mechanically, tedious in practice.
**Depends on:** None.
**Migration risk:** This is where MIGRATE-01/02 equivalence can drift; need a test that asserts loaded-race stats == current hardcoded stats.

### TS-3: Capability flags

**What:** Boolean properties that gate or enable game-system behaviour. Trait form: `{kind: "capability", name: "can_fly"}`.
**Why expected:** `AffFlying`, `AffSwim`, `ActRanger`, `ActVampire` all already serve this role. Codified capabilities stabilize the set.
**Baseline set for v1** (derived from current Act/Aff flags and known race needs):
| Flag | Meaning |
|------|---------|
| `can_fly` | Air sector passage, avoids ground hazards (pixie, avian, titan) |
| `can_swim` | Water sector passage (half racial) |
| `infravision` | Sees in dark rooms (dwarf, gnome, orc, vampire) |
| `detect_hidden` | Bonus to spotting hidden things (elf, half-elf) |
| `needs_blood` | Requires drinking blood for hunger (vampire, lich) |
| `no_sleep` | Cannot be put to sleep by spell (undead, certain races) |
| `no_charm` | Cannot be charmed |
| `regenerates` | Faster HP regen (giant, troll-like) |
| `undead` | Treated as undead for healing/turn undead |
| `corporeal_immune_normal` | Normal weapons pass through (ghost-like) |

**Complexity:** Low for the flag, Medium for the wiring (each flag has a consumer in movement/sleep spell/charm spell/hunger tick, etc.).
**Depends on:** None, but consumers must be audited and refactored to query the trait.

### TS-4: Additive trait composition (race + class merge)

**What:** At character creation / login, merge race traits and class traits into the character's active trait set. Simple union for flags; sum for numeric modifiers.
**Why expected:** Stated in PROJECT.md as core requirement (TRAIT-02). Decision log says "additive stacking, no precedence rules."
**Complexity:** Low — one function called at `pcNewCharacter` and during player load.
**Depends on:** TS-1, TS-2, TS-3.

### TS-5: Trait query API

**What:** Three functions on `*types.Character`:
- `HasTrait(kind, name) bool` — for immunities/capabilities
- `GetModifier(stat) int` — for summed stat deltas
- `HasCapability(flag) bool` — for capability flag lookups
**Why expected:** TRAIT-03 in PROJECT.md. Combat/magic code must stop doing `if ch.Race == X`.
**Complexity:** Low — thin wrappers over an internal trait map.
**Depends on:** TS-1, TS-2, TS-3, TS-4.

### TS-6: TOML trait schema and loader

**What:** Extend the TOML loader (pkg/loader) to parse a `[[traits]]` array inside each race/class file. Validate at load time (unknown damage types, unknown capabilities fail startup).
**Why expected:** DATA-01/02/03 in PROJECT.md; TOML already used for rooms/mobs/objects.
**Complexity:** Medium — schema validation, error messages that point to file+line, registry of known trait kinds.
**Depends on:** Schema decisions from TS-1/2/3.

### TS-7: Lua behavior hooks — minimum viable hook set

**What:** A trait of the form `{kind: "hook", event: "on_death", script: "heucuva_phase_out"}` runs a named Lua script when the event fires on the character.
**Why expected:** LUA-01/02 in PROJECT.md explicitly list hooks: OnDeath, OnAttack, OnSpellCast.
**Minimum hook set for v1** (high-value, low-surface-area):
| Hook | Fires at | Used for |
|------|---------|----------|
| `on_death` | `combat.RawKill` before corpse | Heucuva phase-out, death rattle, undead rise, drop special loot |
| `on_before_damage` | `combat.Damage` before HP reduction | Custom resistance formulas, damage reflection, mitigation abilities |
| `on_after_damage` | `combat.Damage` after HP reduction | Counter-attack triggers, bloodlust, death-check reactions |
| `on_spell_cast` | `magic.Cast` before effect applies | Spell empowerment, mana discounts, silence-on-self |
| `on_level_up` | Player `gainExp` when level increments | Special bonuses, lore messages, stat adjustments |

**Justification for this set:**
- OTLand/Aardwolf Lua APIs expose effectively this same core set (`onAttack`, `onCast`, `onKill`, `onDeath`). [Source: Aardwolf Lua docs, OTLand scripting guide.]
- Each maps to exactly one existing Go call site — minimal surgical incisions.
- Splitting damage into `before`/`after` is the one non-obvious choice but it pays for itself: `before` is for modifying damage, `after` is for reacting to it. Collapsing them forces scripts to do both in one place and is a known source of MUD scripting bugs.

**Complexity:** Medium-High — Lua embedding (gopher-lua), context marshalling (Character → Lua table), error isolation so a script panic doesn't crash the game loop.
**Depends on:** TS-6 (trait schema must parse the hook trait kind).
**Consumers:** Combat (3 hooks), Magic (1 hook), Game loop (1 hook).

### TS-8: Migration equivalence

**What:** Automated test that loads the 19 races and 14 classes from the new TOML files and asserts every stat, flag, and bonus skill matches the current hardcoded tables.
**Why expected:** MIGRATE-01/02 requirement. Without this, silent drift is guaranteed.
**Complexity:** Medium — test harness compares loaded-trait result vs current `RaceTable[i]` / `ClassTable[i]`.
**Depends on:** TS-1 through TS-6.

---

## Differentiators

Features that make the system genuinely better than a copy-paste of ROM's flag system. Build if time allows.

### D-1: Trait source tracking

**What:** Each applied trait records its source (race name or class name). The query API can answer "where did this +2 Str come from?"
**Why valuable:** Debug-ability, which is the single most common complaint about opaque buff systems in live MUDs. Also enables future features (remove-on-class-change, trait list display).
**Complexity:** Low — one extra field per trait instance.
**Pitfall it prevents:** Untraceable bugs when a race and class accidentally stack to give 30 Str.

### D-2: Named modifier types with stacking rules

**What:** Extend stat modifier traits with a `source_type` field: `racial`, `class`, `item`, `spell`. Stacking rule: sum within a type, **cap** across types (or PF2e-style: highest wins within same `source_type`, stack across).
**Why valuable:** Prevents the exact exploit surfaced in ResearchQ4. Pathfinder 2e codified this (3 bonus types) precisely because uncapped additive stacking in PF1e was unbalanced. [Source: Pathfinder 2e bonus rules.]
**Complexity:** Medium — requires a resolution step at trait composition time.
**Trade-off:** Pure additive is simpler (PROJECT.md explicitly chose it) but it relies on trust in authored data. A middle ground is worth considering: keep pure-additive for race+class (internal, high-trust) but add caps for any future item/spell integration.
**Note:** May be flagged as anti-feature by the decision log in PROJECT.md; include this as a "differentiator worth re-evaluating" rather than a must-have.

### D-3: Hook priority / ordering

**What:** When multiple traits register for the same event (e.g. race has `on_death`, class has `on_death`), specify order: `priority: 100` field, higher runs first; allow `stop_propagation` return value from Lua.
**Why valuable:** Two races or a race+class with `on_death` hooks are a realistic scenario (vampire class + heucuva race → both have interesting death behaviour). Without ordering, it's undefined which runs first.
**Complexity:** Low — sort hooks before dispatch.
**Depends on:** TS-7.

### D-4: Behavior hook read-only context vs mutating context

**What:** Distinguish between hooks that can mutate (`on_before_damage` can change damage value) and hooks that are advisory (`on_after_damage` cannot rewrite history). Reflect this in the Lua API — advisory hooks return nothing, mutating hooks return a modified value/table.
**Why valuable:** Prevents a whole class of "scripter writes to read-only context and is silently ignored" bugs. OpenMW Lua spell system explicitly does this. [Source: OpenMW dehardcoding spellcasting merge request.]
**Complexity:** Medium — two Lua binding patterns instead of one.

### D-5: Hook sandboxing with per-hook CPU/memory budget

**What:** Each Lua hook has an instruction count limit; exceeding aborts with a logged error and the character continues without the hook effect.
**Why valuable:** One bad `while true do end` takes down the whole MUD otherwise. Single-threaded game loop means game-wide impact. gopher-lua supports a `SetMx` instruction limit.
**Complexity:** Medium — wiring the limit and handling abort cleanly.
**Note:** gopher-lua explicitly does not support debug hooks, which is the preferred mechanism in reference Lua. Instruction limits via `Options.CallStackSize` / panic recovery is the available fallback. Verify capability before depending on it.

### D-6: Declarative damage-type resistance formulas

**What:** Instead of three discrete flags (immune/resist/vuln) per damage type, allow `{kind: "damage_scaling", damage_type: "fire", multiplier: 0.5}`. 0.0 = immune, 0.5 = resist, 1.5 = vuln, 2.0 = double-vuln.
**Why valuable:** Richer expression (e.g. "water elemental takes 75% from cold" — no flag exists for that today). Future-proofs for environmental damage.
**Complexity:** Medium — requires modifying `CheckImmune` from a tri-state enum to a float multiplier; affects damage math in `damage.go`.
**Trade-off:** Diverges from ROM convention. Mark as "v1.1 evolution" not v1.

### D-7: Capability flag → gameplay system registry

**What:** A central registry that maps each capability flag to the system(s) that consume it, with docs. Validating that a declared capability actually has a consumer prevents data-file typos like `cant_fly` silently having no effect.
**Why valuable:** The exact same reason SQL schemas enforce foreign keys. In a flag-based system, a typo produces no error and no effect — silent data drift.
**Complexity:** Low — a map literal in code and a load-time validation pass.

### D-8: Trait introspection commands

**What:** Immortal command `traits <player>` that dumps the composed trait set with sources (pairs with D-1). Players get a trimmed version via `traits` / `score`.
**Why valuable:** A non-trivial feature ROM never had cleanly. Aardwolf's `help race` / `consider` are popular for exactly this reason. Trivial once D-1 is built.
**Complexity:** Low (given D-1).

---

## Anti-Features

Features to explicitly NOT build in v1. Each has a specific "what to do instead" so the reasoning doesn't evaporate.

### AF-1: Trait inheritance hierarchies

**Why avoid:** Out-of-scope in PROJECT.md. Diamond-inheritance bugs are the classic LPMud pitfall; CS-grads reach for inheritance reflexively and pay for it forever.
**What to do instead:** Keep flat additive composition. If "vampire lord extends vampire" is needed, author the composed trait list directly in the vampire-lord file. Copy-paste > inheritance for 20 tables.

### AF-2: Hot-reloading trait files at runtime

**Why avoid:** Out-of-scope in PROJECT.md. Mid-combat reload of a fire-immunity trait means the damage formula switches mid-calculation. Cache invalidation is its own research project.
**What to do instead:** Graceful restart. Startup-only loading.

### AF-3: Player-authored traits / in-game trait builder

**Why avoid:** Out-of-scope in PROJECT.md. Arbitrary Lua scripts from untrusted input is a server exploit waiting to happen.
**What to do instead:** Admin-only TOML file editing with git review.

### AF-4: Hooks for every conceivable event

**Why avoid:** Every hook is a permanent API surface. Aardwolf's Lua API has dozens of hooks accumulated over 20 years, and some are undocumented or unused. Define only the five hooks in TS-7 for v1; add new hooks only on demand with a concrete use case.
**What to do instead:** Start with the five-hook minimum; document a process for adding hooks (new trait kind, Go call site, one test, docs).

### AF-5: Merging Traits and Affects into one system

**Why avoid:** Traits are permanent/innate structural properties. Affects are temporary spell/item effects with durations. They have different lifecycles (loaded-at-startup vs tick-decremented), different serialization (TOML file vs player save), and different consumers. A unified system ends up with a "permanent" flag on every Affect and worse ergonomics both sides.
**What to do instead:** Keep Affects (`pkg/magic` affect system) untouched. Traits live in a separate field on Character. Combat code consults both via the trait query API plus existing Affect queries.

### AF-6: Cross-character trait interactions in Lua

**Why avoid:** Scenarios like "my on_attack hook modifies the attacker's stats for one round" require exposing the attacker's mutable state to victim-owned scripts. This creates an implicit trust boundary inside the scripting layer. Known pain point in Aardwolf and similar MUDs.
**What to do instead:** Hooks operate on self only. Effect on another character must go through an existing Go-side system (damage, affect apply, etc.) as an intermediary.

### AF-7: Trait prerequisites / "if you have X you also get Y"

**Why avoid:** The "flat additive" rule is a feature, not a limitation. Prerequisites reintroduce order-of-composition bugs. D&D 5e racial/feat prerequisites are a documented source of character-builder bugs.
**What to do instead:** Duplicate the trait in both files. 19 races × 14 classes is small; don't optimize for the wrong scale.

### AF-8: Trait "removal" at runtime

**Why avoid:** Asymmetric with the additive composition model. Removing a capability flag that was never actually used is a no-op; removing one that IS used (in the middle of combat) is a race condition.
**What to do instead:** Affects cover temporary removal (e.g. `weaken` lowers Str). Permanent removal = reroll character.

### AF-9: Non-deterministic / random traits

**Why avoid:** E.g. "on creation, randomly gain one of these three resistances." Breaks player-save reproducibility and regression tests (TS-8 migration equivalence can't assert equality against an RNG). If stochastic mechanics are wanted, they belong in spells/affects, not race definition.
**What to do instead:** Deterministic data files. Players pick their race; variety comes from race × class combinatorics (19 × 14 = 266 builds).

---

## Feature Dependencies

```
TS-1 (Vuln/Res/Imm traits) ──┐
TS-2 (Stat modifiers) ───────┼──► TS-4 (Additive composition) ──► TS-5 (Query API) ──► TS-8 (Migration test)
TS-3 (Capability flags) ─────┘                                                                  ▲
                                                                                                │
TS-6 (TOML schema/loader) ────────────────────────────────────────────────────────────────── feeds
                    │
                    └──► TS-7 (Lua hooks)

Differentiators:
D-1 (Source tracking) ──► D-8 (Introspection cmd)
TS-7 ──► D-3 (Hook priority), D-4 (RO vs mutating), D-5 (Sandboxing)
TS-1 ──► D-6 (Scaled resistance) [evolution, not v1]
TS-3 ──► D-7 (Capability registry)
```

**Critical path for PROOF-01 (add Vampire via data file):** TS-1 → TS-2 → TS-3 → TS-4 → TS-5 → TS-6 → TS-7 → TS-8. All eight table stakes are on the critical path. Skipping any one of them breaks the proof.

---

## Ordering Recommendation for Roadmap

If phases emerge, the natural order is:

1. **Foundation:** TS-1, TS-2, TS-3, TS-5 (trait types + query API, all operating on in-memory traits with no loader yet; hardcode a vampire trait list in Go to prove the query API works against existing combat code).
2. **Composition:** TS-4, TS-8 (migration-equivalence test with still-hardcoded traits, proving the composition merge is equivalent to the current `RaceTable` behavior).
3. **Data files:** TS-6 (TOML loader; move all 19 races + 14 classes out of Go tables into data files; re-run TS-8 migration test).
4. **Scripting:** TS-7 + D-5 (Lua hooks; do not ship without D-5 sandboxing — one runaway script kills the server).
5. **Proof:** Add Vampire as a new race entirely via data file, including blood-drinking on_attack hook. Run TS-8 migration test one more time to confirm existing races still match.
6. **Polish (differentiators):** D-1, D-3, D-7, D-8 — cheap and high-value once the foundation is there.

---

## Sources

- [ROM | Muds Wiki](https://muds.fandom.com/wiki/ROM) — ROM 2.4 / Diku lineage context.
- [BaseMUD (Synival)](https://github.com/Synival/BaseMUD) — modernized ROM reference for flag systems.
- [dnd-5e-srd races.json](https://github.com/BTMorton/dnd-5e-srd/blob/master/json/01%20races.json) — racial trait data model example; subrace additive inheritance pattern.
- [dnd-5e-srd feats.json](https://github.com/BTMorton/dnd-5e-srd/blob/master/json/05%20feats.json) — feat schema.
- [PF2e Feats and Features rules](https://2e.aonprd.com/Rules.aspx?ID=1327) — class feature / feat taxonomy.
- [Aardwolf Lua docs](https://www.aardwolf.com/lua.html) and [Lua triggers](https://www.aardwolf.com/lua/lua-mud-triggers.html) — event hook conventions in a large live MUD.
- [Aardwolf Races](https://www.aardwolf.com/race.html) / [Classes](https://www.aardwolf.com/class.html) — race and class data structure in a production MUD.
- [OTLand Lua scripting guide](https://otland.net/threads/scripting-guide.74030/) — onAttack/onCast/onKill/onDeath hook conventions.
- [OpenMW dehardcoding spellcasting MR](https://gitlab.com/OpenMW/openmw/-/merge_requests/3029) — handler-chain approach for spell events with read-only vs mutating contexts.
- [Design Insights: Static Bonuses](https://apothecary.press/2021/11/27/design-insights-static-bonuses/) — stat modifier stacking design.
- [ModifierManager (Roblox)](https://devforum.roblox.com/t/modifiermanager-a-stat-modifier-system-with-type-safety-stacking-rules-and-client-sync/4338060) — runtime modifier system with stacking rules.
- [gopher-lua README](https://github.com/yuin/gopher-lua) — embedding constraints (no debug hooks, instruction limits via options).
- [LPMud FAQ](https://github.com/maldorne/awesome-muds/blob/master/docs/lpmuds/lpmud-faq.md) — mudlib modularity lessons.
- Codebase references: `go/pkg/combat/combat.go` (CheckImmune, isVampire), `go/pkg/types/flags.go` (ActFlags, AffectFlags, ImmFlags), `go/pkg/types/races.go`, `go/pkg/types/classes.go`.
