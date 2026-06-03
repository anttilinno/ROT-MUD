# Roadmap: ROT-MUD Data-Driven Trait System

## Overview

Transform the ROT-MUD entity systems (races, classes, skills, spells, mob templates, rooms, and items) from hardcoded Go `var` tables and scattered identity checks into a declarative, data-driven trait architecture. The journey starts with a safety net (golden-master tests that capture current behavior across combat, spells, skills, and mob encounters), then builds the trait type system in pure Go, then the TOML loaders for each entity domain (races/classes, skills/spells, mobs) with the Lua scripting host running in parallel, then swaps every race/class/skill/spell identity check in combat/magic/skill/game code for trait queries under CI lint enforcement, then migrates all 19 races, 14 classes, every skill, every spell, and every mob template to data files, then extends the existing area loader with trait parsing and annotates rooms and items (silver weapons, no-magic zones, etc.) with traits, and finally proves extensibility by adding a brand-new race via TOML only — zero Go changes, with a Lua behavior hook.

## Phases

**Phase Numbering:**

- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [ ] **Phase 1: Golden-Master Safety Net** - Capture current combat, spell, skill, and mob behavior in a deterministic parity suite before any migration starts
- [x] **Phase 2: Trait Type System** - Typed trait structs, additive composition with per-axis caps, and a trait query API in pure Go (completed 2026-06-01)
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
- [ ] **Phase 13: Economic Overhaul** - Add durability/repair, smith custom crafting, identify fees, and bank fees so the economy has real coin sinks; rebalance mob drops to a stable source/sink ratio (see `.planning/ECONOMY.md` for sub-phase detail)
- [ ] **Phase 14: LLM-Driven NPCs** - Local-LLM-backed dialog for shopkeepers/smiths/sages (Tier 1) and plan-once tactical combat for area bosses (Tier 2), with first-class scripted fallback, circuit breaker, and feature flag (see `.planning/LLM-NPC.md` for sub-phase detail) — *N1+N2 exploratory spike landed 2026-06-03 (`pkg/llm`, Otho live); not yet a formally planned/verified phase*

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

**Plans:** 4 plans
Plans:

- [x] 01-01-PLAN.md — Refactor pkg/combat/dice.go with package-scope seedable RNG (SetRand) and add CombatSystem.Rand field per D-02
- [x] 01-02-PLAN.md — Create pkg/golden/ package (doc.go, fixture.go scenario runners, golden_test.go driver with -update flag)
- [x] 01-03-PLAN.md — Generate and commit initial testdata/entities.golden snapshot (human-verify checkpoint)
- [x] 01-04-PLAN.md — Close VERIFICATION.md SC #3 gap: add mob-template coverage (immunity, aggro, caster special) to the golden fixture

### Phase 2: Trait Type System

**Goal**: A pure-Go trait system exists where entities (races, classes, skills, spells, mobs, rooms, items) can be composed from typed trait values and queried by combat/magic/skill code
**Depends on**: Phase 1
**Requirements**: TRAIT-01, TRAIT-02, TRAIT-03
**Success Criteria** (what must be TRUE):

  1. Typed trait structs exist for Vulnerability, Resistance, Immunity, StatModifier, CapabilityFlag, and BehaviorHook with parameterized fields and a closed `TraitKind` enum
  2. Traits from multiple sources (race + class, or skill + spell effect, or room + item) combine additively at the merge site, with per-axis caps preventing stacking blowup
  3. The trait query API (`HasTrait`, `HasCapability`, `GetModifier`, `ResolveImmunity`, `HooksFor`) is callable from any package and covered by unit tests
  4. A resolved-trait bitmask cache makes `HasCapability` O(1) with zero allocation per query

**Plans:** 2/2 plans complete
Plans:
**Wave 1**

- [x] 02-01-PLAN.md — Foundation: trait structs, TraitKind/HookEvent enums, capability string->bit registry + 256-bit CapBits primitive (TRAIT-01)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 02-02-PLAN.md — Composition + query API: TraitSet, Resolve() (clamped RIS/stat sums + bitset), HasTrait/HasCapability/GetModifier/ResolveImmunity/HooksFor (TRAIT-02, TRAIT-03)

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

### Phase 13: Economic Overhaul

**Goal**: The game economy has real coin sinks (durability/repair, race+class crafting, magical enchants, identify fees, bank fees, RNG loot lottery, gods + temples + faction hubs, player housing) and mob drops are rebalanced to a stable source/sink ratio near 1.0 per level bucket
**Depends on**: None (independent of trait system; can begin once the currency commit lands)
**Requirements**: ECON-01, ECON-02, ECON-03, ECON-04, ECON-05, ECON-06, ECON-07, ECON-08, ECON-09, ECON-10
**Success Criteria** (what must be TRUE):

  1. A coin ledger records every credit/debit on `ch.Coin` and `ch.PCData.BankCoin` with txn type, amount, source/target, and tick; sim tests produce a per-level-bucket source/sink ratio report
  2. Weapons and armor have hits-based durability that ticks down in combat; broken items wear-fail with halved stats but are not destroyed; `repair` command at smith NPCs restores durability at a cost scaled by item Cost and damage fraction
  3. Master smith NPCs craft race+class-aware gear from TOML recipes across 12 shared slots plus 1 race-signature slot; crafting is 3-tier level-gated (T1 L1+, T2 L31+, T3 L76+); T3 is best-in-slot at level cap and requires recipe-specific boss-drop materials; on-affinity (race+class match) wearers unlock 4/6/8/13-piece set bonuses; off-affinity wearers cap at T2; all crafted items are bind-on-pickup with a salvage path that refunds ~50% coin / ~80% materials / 100% craft-XP
  4. Items at vnum-level ≥ 20 drop unidentified; sage NPCs charge a flat fraction of `item.Cost` to identify; identify cost scales by rarity (Magic / Rare / Set / Unique) for lottery drops
  5. Bank deposits remain free; withdrawals at non-home bankers and player-to-player transfers charge a configurable fee
  6. After all sinks land, `mobCoinDrop` is rebalanced and the death-loss percentage reduced so the sim source/sink ratio per level bucket lands within ±10% of 1.0; the death penalty drops from 10% to 5% of carried coin
  7. Enchanter NPCs add one magical enchant slot to crafted T2+/T3 and lottery-rare-or-better found items; three difficulty tiers (Simple / Greater / Master) with surfaced odds (95% / 75% / 40%); Master-tier fail can brick the item (10% destroy chance); reagents are tradeable on the player market; `scour` strips an enchant for re-enchanting at a coin cost
  8. Mob-killed equipment drops at degraded durability (`max(0.10, 1 - dmg_taken_pct) * baseDurabilityMax`), with chest/quest/shop/bank items exempt; rarity tiers (Normal / Magic / Rare / Lottery-Set / Unique) roll affixes scoped by base type and ilvl; `MagicFind` stat shifts rarity weights with a soft cap; auto-loot filters honor rarity and durability thresholds
  9. Pantheon of 6-9 gods loads from TOML; players pick at L10+ via `pray <god>`; `sacrifice` / `tithe` / `offer` / `boon` / `atone` / `favor` / `pay_with_favor` / `bribe` commands implemented and persisted; each god has a temple shop with tiered access (open / worshipper / cleric / favored) and opposing-alignment refusal; cleric-of-this-god discount is the deepest; coin and favor are both valid currencies for temple purchases (favor-only items exist for iconic relics); tithe + decay loop creates recurring favor pressure on real-time clock. Three hub cities load in first pass: Midgaard (good/lawful), Shadowport (evil), Skullhold (chaotic) — each with bank, smiths, enchanters, sages, temple complex, housing market, and MUD school exit destination; symmetrical alignment-refusal at hub gates; wilderness shrines sit between hubs for one-shrine-per-god roadside coverage. City guards enforce alignment thresholds (newbie grace L<5, suspicious -350 to -700, hostile <-700); MUD school graduation offers a non-binding destination choice (good / neutral / evil / chaotic) with a guard-sergeant escort that one-way-teleports new grads to the appropriate hub plaza; cross-faction city entry is gated through atonement, disguise/polymorph, or per-real-day-limited gate bribes
  10. Innkeeper-run housing markets in each hub offer four rent tiers (Room 5g / Cottage 50g / Manor 500g / Fortress 2p per real-time week); rent in the player's registered home hub (shared E5 hometown field) is the listed price, cross-hub rent is 2× listed; weekly real-time auto-debit from bank then carried coin, with a 7-day grace + 30-day freeze + auto-downgrade-or-eviction ladder when balance runs out; `rent` / `unrent` / `upgrade` / `recall home` / `house` commands implemented; six one-time upgrades available (smith forge, workbench, alchemy bench, personal safe, altar, trophy hall) each adding a rent multiplier and replicating standard NPC behavior in-house (T3 craft still requires master-smith pilgrimage; enchant brick risk unchanged); `change_hometown <city>` costs 1p and 14-day real-time cooldown; storage room slot count scales with tier and is bank-grade (no decay, no carry-weight)

**Plans**: TBD
**Reference**: `.planning/ECONOMY.md` for sub-phase breakdown (E1 baseline → E2 durability → E3 race+class crafting → E3.5 enchants → E4 identify → E5 bank fees → E6 rebalance → E7 loot lottery + damaged drops → E8 gods + favor + temple shops + three-hub geography → E9 player housing + per-hub markets), open decisions, and risk register

### Phase 14: LLM-Driven NPCs

**Goal**: Selected NPCs (shopkeepers, smiths, sages, area bosses) are driven by a local LLM for dialog and tactical combat planning, with a first-class scripted fallback that runs identically when the LLM is unavailable
**Depends on**: None — independent of trait system and economy overhaul; benefits from Phases 4 (Lua hook taxonomy) and 8-12 (data-driven mob kits) but does not block on them
**Requirements**: LLM-01, LLM-02, LLM-03, LLM-04, LLM-05, LLM-06
**Success Criteria** (what must be TRUE):

  1. Async LLM worker pool calls a local endpoint (Ollama / llama.cpp / vLLM); requests are non-blocking, results dispatch back through the game loop; per-mob serial with overflow drop
  2. Scripted fallback runs identically to current behavior when the LLM is off, unreachable, slow, or returns invalid output; a per-endpoint circuit breaker opens after 5/10 failures and recovers via half-open probing; `llmstat` immortal command exposes queue depth, breaker state, p50/p95 latency, and failure rate
  3. Tier 1 (dialog) NPCs flagged with `llm_enabled = true` emit tool calls (`say`, `emote`, `set_price`, `offer_item`, `refuse`) which the server validates and executes; persona + per-(mob, player) dialog memory persisted to JSONL
  4. Tier 2 (combat) bosses flagged with `llm_combat = true` produce a structured battle plan on aggro (opener, HP-phase rotations, replan triggers, taunt lines, exploit notes); per-round combat is a cheap scripted FSM walking the plan; replan triggers fire async without blocking the round
  5. Post-fight post-mortem stores structured lessons keyed by `(mob_vnum, player_id)`; rematches load lessons into the plan-call context, producing visibly escalating mob tactics
  6. Phase 1 golden-master combat parity still passes with the LLM feature flag off; an `llm_smoke_test.go` mocks the endpoint and verifies tool validation, fallback paths, circuit breaker transitions, and overflow handling

**Plans**: TBD

**Spike progress (2026-06-03, exploratory — not a formally planned/verified phase):**

- **N1 — worker pool + endpoint client: DONE.** `pkg/llm` ships async worker pool (non-blocking, per-mob in-flight dedup, drop-on-overflow), llama.cpp OpenAI-compatible client with grammar-constrained `json_schema` sampling, circuit breaker (opens 5/10, half-open probe, 60s cooldown), Tier-1 tool-call validation, and the `llmstat` immortal command. Mock-endpoint unit tests cover validation, fallback, breaker transitions, and overflow (criterion 6's `llm_smoke_test` intent).
- **N2 — Tier 1 dialog on one mob: DONE.** Otho the money changer (vnum 3162) flagged `llm_enabled` with a persona in TOML; player `say` in-room triggers an LLM dialog turn delivered back through the game loop; live-verified end-to-end against a local llama.cpp server.
- **Gaps vs full phase:** feature gated by env var (`ROTMUD_LLM`), not `config.toml` (server has no config loader yet); tool surface is `say`/`emote`/`refuse` only (no `set_price`/`offer_item`); no per-(mob,player) JSONL dialog memory; `llmstat` lacks p50/p95 latency; Tier 2 combat (N4–N6, criteria 4–5) not started; golden-master parity (criterion 6) deferred until Phase 1 exists.

**Reference**: `.planning/LLM-NPC.md` for sub-phase breakdown (N1 worker pool → N6 Tier 2 rollout), tool surface, schema definitions, and risk register

## Progress

**Execution Order:**
Phases execute in numeric order: 1 -> 2 -> 3 -> 4 -> 5 -> 6 -> 7 -> 8 -> 9 -> 10 -> 11 -> 12 -> 13 -> 14

**Parallelization opportunities** (parallelization=true in config):

- Phase 4 (Lua Scripting Host) is independent of Phases 3/5/6 and can run in parallel once Phase 2 lands
- Phase 7 (Identity-Check Refactor) is independent of Phases 3/4/5/6 and can start in parallel once Phase 2 lands; it must complete before any migration phase
- Phases 5 (skills/spells loaders) and 6 (mob loaders) can run in parallel after Phase 3 lands
- Phases 8, 9, 10 (migration) can run in parallel after their respective loader phases and Phase 7 land
- Phase 11 (area/item traits) can run in parallel with Phases 8/9/10 once Phase 7 lands (it extends the existing area loader, so it does not need the race/class/skill/spell/mob loader phases)
- Phase 13 (economic overhaul) is independent of the trait system and can run in parallel with Phases 2–12 once the currency commit lands
- Phase 14 (LLM-driven NPCs) is independent of all other phases; can run anytime in parallel. Benefits from Phase 4 (Lua hook taxonomy) and the data-driven mob kits from Phases 8/10 landing first, but stub events suffice

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Golden-Master Safety Net | 0/3 | Not started | - |
| 2. Trait Type System | 2/2 | Complete   | 2026-06-01 |
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
| 13. Economic Overhaul | 0/TBD | Not started | - |
| 14. LLM-Driven NPCs | 0/TBD | Spike (N1+N2 landed) | - |
