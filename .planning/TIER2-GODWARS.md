# Tier 2 as GodWars-Style Supernatural Classes

## Status

Design draft, 2026-09-18. Not yet on the roadmap. **T0 implemented** on branch `feat/tier2-godwars`. Reference codebase: GodWars Deluxe fork at
`~/Repos/Misc/GodWars` (github.com/benjamin-small/GodWars). Port **mechanics and ideas**, not code —
GodWars file headers still carry Richard Woolcock's "not to be copied without permission" notice.

## Decisions taken

| Decision | Choice |
|---|---|
| What tier 2 is | GodWars-style supernatural classes replace the unused ROM remort classes |
| Entry | Per-class rite in play; demon = **pact quest** with a demon-lord NPC |
| Sequencing | Hand-coded Go now on top of `pkg/traits`; Phase 8 (race/class migration) moves it to TOML later |
| First class | Demon (items → powers → economy) |
| Tier-1 vampire | Renamed **ghoul** (GodWars' vampire servant). Later, only ghouls can be embraced into tier-2 vampire |
| Level on ascension | **Reset to 1** (ROT 1.4 reroll semantics); skills carry over from the origin class |
| Old ROM tier-2 classes | **Deleted** (wizard, priest, mercenary, gladiator, strider, sage, lich) |
| Race interaction | Each tier-2 class declares the races allowed to take its rite |

## Current state (why tier 2 is free real estate)

- `types.ClassTable` has 7 tier-2 entries (wizard, priest, mercenary, gladiator, strider, sage, lich,
  indices 7–13) but **nothing reaches them**: no reroll/remort command, `PCData.Tier` is never read or
  written, login only offers `i < ClassWizard` (`pkg/server/login.go:309,905`).
- Original ROT 1.4 had `do_reroll` (`c_original` tarball, `act_comm.c:112`): hero-only, confirm twice,
  sets `tier = 1`, quits, re-runs creation offering tier-2 classes. The Go port never carried it over.
- `pkg/traits` exists (Phase 2) but is **not yet consulted by gameplay** — nothing outside the package
  imports it. Any tier-2 power expressed as traits needs a thin composition point wired in (see T0).
- No quest-point currency, no damage cap, no polymorph. Quest system (`pkg/game/quests.go`) supports
  kill / collect / deliver / explore — enough for a pact chain.

## GodWars model in one paragraph

GodWars classes are supernatural templates acquired in play (vampire by embrace, werewolf by bite, demon
by pact…), each with a private power tree and currency. Demons earn **power** from kills and from
sacrificing demonic items, and a demon lord spends it to *inpart* powers (body parts, stats, senses,
actives, demonform) onto champions. Power + quest points also craft a **12-slot demon armour set** in
five colour tiers; each worn piece raises the damage cap and higher tiers add a flat chance for hostile
spells to fizzle. Sacrificing a piece refunds exactly its power cost.

## Tier-2 roster (proposed)

| Tier-2 class | GodWars source | Entry rite | Currency | Notes |
|---|---|---|---|---|
| **Demon** | demon / champion | Pact quest | Power | First to ship. |
| Vampire | vampire | Embrace; ghouls only | Blood | Tier-1 vampire renamed ghoul |
| Werewolf | werewolf + tribes | Survive a bite quest | Rage / gnosis | |
| Magus | mage colours | Awakening trial | Quintessence | 5 colours ≈ spell schools |
| Highlander | highlander | Win a quickening duel | Quickening | Weapon-bound |
| Angel | angel | Mirror of demon | Grace | Same systems as demon, holy flavour |

Hybrids (abomination, lich-as-vamp+mage, baali, …) are **out of scope** — they multiply balance work.

## Phases

### T0 — Tier-2 foundation ✅

- `reroll` command: hero-only (`LevelHero` = 101), tier-1 class only, double-confirm, sets
  `PCData.Tier = 1` (eligible). **Unlike ROT 1.4**, the player does not re-create — the rite completes
  the ascension in place: origin class saved in `PCData.Classes`, class set, level reset to 1,
  `Tier = 2`. Until the pact quest lands (T3), the immortal `ascend <player> <class>` command runs it.
- Delete the ROM tier-2 `ClassTable` entries; add demon. Other GodWars classes are added in their own
  phase.
- Login/creation keeps offering tier-1 only; class names for tier 2 resolve for saved players.
- `PCData.Tier` and `PCData.Classes` (both already declared, unused) start being persisted.
- Deferred to the phase that first needs them: `ClassPower`/`ClassPowerTotal`/`Powers` (T2), and the
  `ch.ResolvedTraits()` composition point — race + class + bought powers + worn-item traits, cached and
  invalidated on equip/unequip/purchase (T1).

### T1 — Demon items (12-slot set)

- Slot → `WearLocation` mapping (no new wear locations):

  | Slot | WearLoc | Slot | WearLoc |
  |---|---|---|---|
  | ring | FingerL/R | gauntlets | Hands |
  | collar | Neck1/2 | sleeves | Arms |
  | plate | Body | cape | About |
  | helmet | Head | belt | Waist |
  | leggings | Legs | bracer | WristL/R |
  | boots | Feet | visor | Face |

- 12 object templates in a new `data/areas/relic/` (or similar) area, name/short-desc templated by tier
  (`a %s demonic ring`), matching GodWars' one-template-per-slot trick.
- Tiers: black → grey → purple → red → brass. Each piece: set tag `demonic`, tier, `Owner` = crafter
  (existing `Object.Owner`), anti-good, wear restricted to demons.
- Set bonus **recomputed from worn equipment** on every equip/unequip, not a running counter (GodWars'
  `pcdata->demonic` counter drifts across death/quit paths):
  - per piece: damage bonus (ROT has no damcap; use `DamRoll` modifier or % damage — tune in sim)
  - purple/red/brass: +3/+4/+5 % spell-fizzle chance per piece → new `spell_fizzle` modifier axis,
    checked once in the spell-hit path (`pkg/magic`)
- Timer decay message "vanishes in a blast of flames" when a demonic item decays.

### T2 — Demon powers

Each `inpart` power becomes a keyed entry in `PCData.Powers` that contributes a trait bundle.

| Power | GW cost | ROT effect |
|---|---|---|
| fangs, claws | 2500 | capability `natural_weapon` + bite/claw damage noun, unarmed damage bonus |
| hooves | 1500 | movement cost reduction |
| wings | 1000 | capability `fly` (permanent affect) |
| nightsight | 3000 | capability `infravision` + see in dark |
| might | 7500 | modifier: damage bonus |
| toughness | 7500 | resistances: bash/slash/pierce |
| speed | 7500 | extra attack |
| shadowsight, scry | 7500 | detect hidden / remote look skill |
| travel | 1500 | skill: go to a player's room |
| eyespy | 1000 | skill: summon a watcher mob |
| inferno, firewall | 10000 / 1000 | skills: room fire damage / blocking exit |
| imp | 10000 | skill: summon imp pet |
| demonform | 25000 | toggle: timed trait overlay (claws + nightsight + stat boost), morph name |

- `inpart <power>` buys for **self** (no player demon-lord hierarchy — the lord is an NPC; see T3).
- Deferred: head/tail mutations, weaponform (player-as-object), champion/prince hierarchy.

### T3 — Pact quest + power economy

- **Entry**: tier-1 hero with `Tier = 1` eligibility completes a pact chain given by a demon-lord NPC
  (LLM dialog tier fits here): e.g. kill 3 named good-aligned mobs → deliver a soul-gem → pact scene.
  Completion sets class = demon, grants starting power.
- **Faucets** (GodWars → ROT):
  - mob kill: +`victim.Level` power (GW `fight.c:5275`)
  - sacrifice demonic item: refund its power cost (GW recycle loop; maps to E3 `salvage`)
  - boss kill: flat bonus
  - player kill: none unless ROT gets PK
- **Sinks**: `inpart` powers; `demonarmour <tier> <slot>` crafting at the demon-lord.
- GodWars gates higher armour tiers on quest points; ROT has none. Replace with **boss materials**
  (E3 `ItemTypeMaterial`) + coin, so demon armour rides the E3 smith economy rather than inventing QP.
- Ledger (E1) txn types: `power_gain`, `power_spend`, `power_refund`.

### T4+ — Remaining tier-2 classes

One phase per class, each reusing T0's currency + powers + set machinery: new power table, new rite,
optionally a 12-slot set with its own tier colours (GodWars angels reuse the same templates).

## Relation to existing plans

- **ECONOMY.md E3** already specifies the same 12 shared slots and GodWars-style salvage. Demon armour
  should be the first concrete E3 "set", not a parallel system.
- **ECONOMY.md E8** (gods + favor) has the same shape as demon power (sacrifice → currency → boons).
  Keep separate currencies; a dark god may later grant demon power as a boon.
- **Roadmap Phase 7/8**: T0's composition point is a preview of Phase 7; tier-2 classes join the Phase 8
  TOML migration.

## Race limits (proposed defaults, tune freely)

Encoded as `Class.AllowedRaces` (empty = all races).

| Tier-2 class | Barred races | Reason |
|---|---|---|
| Demon | heucuva, avian, pixie | GodWars: "cannot make a pact with the undead"; sky and fey races as celestial counterweights |
| Vampire | heucuva, titan, giant | already undead; too large to pass unseen |
| Werewolf | heucuva, pixie, avian, kenku | undead / no wolf form for tiny or winged races |
| Magus | minotaur, gnoll, giant | low-Int brute races |
| Highlander | heucuva, pixie | undead; too small for the blade |
| Angel | heucuva, goblin, gnoll, halforc | undead and traditionally evil races |

## Skills after ascension

Skill and spell level requirements are keyed by class index. A tier-2 character looks skills up through
`ch.SkillClass()`, which returns the origin tier-1 class (`PCData.Classes[0]`). Tier-2 powers are separate
(bought, not levelled) so they never go through the skill table.

## Exit criteria (T0–T3)

- A hero can reroll, complete the pact quest, and become a demon; state survives save/load.
- Kills and sacrifices grant power; `inpart` and `demonarmour` spend it; ledger records all three.
- 12-slot set bonus computed from worn gear matches expected values for 0/1/6/12 pieces of each tier.
- Spell-fizzle chance applies only to demonic-set wearers and is capped.
- Golden-master suite unchanged for all non-demon encounters.
