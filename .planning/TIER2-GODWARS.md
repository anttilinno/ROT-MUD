# Tier 2 as GodWars-Style Supernatural Classes

## Status

Design draft, 2026-09-18. Not yet on the roadmap. **T0–T4 implemented** on branch `feat/tier2-godwars`. Reference codebase: GodWars Deluxe fork at
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

## Tier-2 roster

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

### T1 — Demon items (12-slot set) ✅

- Slot → `WearLocation` mapping (no new wear locations):

  | Slot | WearLoc | Slot | WearLoc |
  |---|---|---|---|
  | ring | FingerL/R | gauntlets | Hands |
  | collar | Neck1/2 | sleeves | Arms |
  | plate | Body | cape | About |
  | helmet | Head | belt | Waist |
  | leggings | Legs | bracer | WristL/R |
  | boots | Feet | visor | Face |

- As built:
  - `data/areas/relic`: 12 templates at vnums 29650–29661 (GodWars' relic range), level 1, named
    "a demonic <slot>". `types.ForgeDemonic(obj, tier, owner)` inserts the tier colour
    ("a purple demonic ring") and binds the piece; T3's `demonarmour` command calls it. `oload`
    gives an unforged, unbound piece (counts as black).
  - `Object.Tier` added and persisted. Tiers: black, grey, purple, red, brass.
  - `ch.DemonicSet()` recomputes pieces and fizzle chance from worn equipment on every call: no
    running counter (GodWars' drifted), no cache. Non-demons get nothing.
  - Per piece: +2 damroll (`types.DemonDamrollPerPiece`, applied in `combat.GetDamroll`).
  - Purple/red/brass: +3/+4/+5 % hostile-spell fizzle per piece (full brass = 60 %), checked by
    `Spell.Fizzles` at every offensive-spell call site (cast, object cast, wand/scroll/etc.).
  - Wear: demons only; a bound piece only by its owner.
- Dropped from the original plan:
  - Traits composition point: `DemonicSet` is the only consumer, so there's nothing to compose yet.
    Revisit with T2 powers.
  - Decay message: forged pieces have no timer.
- Known gap (pre-existing, not T1's): armour `Values` AC is never applied on equip anywhere in ROT,
  so the templates' `ac_*` values are inert. Fixing it rebalances all armour; track separately.

### T2 — Powers (all tier-2 classes) ✅

- `types.PowerTable`: 45 powers across the six classes (GodWars inpart, disciplines, gifts, spheres,
  katas). Owned powers are keys in `PCData.Powers`; `powers` lists, `powers <key>` buys (with
  prerequisites and costs).
- **Passive effects are derived at query time, never stored as affects**, so nothing drifts across
  save/load. There are four choke points:
  - `Character.IsAffected`: power affect flags (flying, haste, detect, infravision, sneak, regeneration)
  - `Character.GetStat`: stat bonuses
  - `combat.GetHitroll`/`GetDamroll`: hitroll and damroll bonuses
  - `combat.CheckImmune`: `InnateRIS()`, i.e. class innate resistance/vulnerability plus power resistances
- Active powers unlock commands:
  - `travel <player>` (demon travel, vampire mist, werewolf moonbridge, magus correspondence, angel
    travel)
  - room blasts `inferno`/`forcebolt`/`smite` (fire/energy/holy)
  - battle forms `demonform`/`bloodrage`/`crinos`/`quickening`/`angelform`
- A form's affect is only a timer (`ApplyNone`); its +5/+10 bonus is derived in `PowerBonus`.
  Wear-off text for non-spell affects goes through `magic.WearOffMessages`.
- Deferred: head/tail mutations, weaponform (player-as-object), champion/prince hierarchy, hooves,
  scry/eyespy, imp summon, firewall.

### T3 — Rites + power economy ✅

- **Hall of Rites** (`data/areas/relic`, rooms 29600–29607): down the stair behind the Temple of
  Thoth (3001). There is one master NPC per class (29601–29606) in safe, no_mob rooms.
- **Rite flow**: `reroll confirm` (hero) → at a master: `rite` (terms and progress) → `rite begin` →
  complete the deed → `rite complete` → `Character.Ascend`. Each rite is data (`Class.Tier2.Rite`):
  kill N NPCs with minimum level, alignment, night-only and act-flag filters.
- **Faucet**: `Combat.OnKill` was declared but never wired. It is now wired to `onKill`:
  - tier-2 players earn the victim's level in currency (pets earn for their owner)
  - rerolled heroes advance their rite
- **Demon armour**: `demonarmour <tier> <slot>` at the demon lord costs 2000 power plus coin
  (black free; grey 10g, purple 30g, red 60g, brass 100g). GodWars' quest points became coin because
  ROT has none.
- **Recycle loop**: sacrificing a demonic piece refunds its 2000 power to a demon (the coin is lost).
- Not done: the E1 ledger (not built yet), boss-material gating (E3 materials don't exist yet).

### T4 — Remaining tier-2 classes ✅

- Vampire (ghouls only, via `Tier2.OriginClasses`), werewolf, magus, highlander and angel are in
  `ClassTable` with race bars, innate RIS, currency, master and rite.
- Innate RIS:
  - demon: fire res, holy vuln
  - vampire: negative/poison res, fire/silver vuln
  - werewolf: silver vuln
  - angel: holy res, negative vuln

### Fixes found along the way

- **Saving**: ascended characters restart at level 1, but quit/disconnect skipped saving every
  level-1 character. Now `Character.Saveable()` means level > 1 **or** rerolled/ascended.
- **Affects ticked twice per tick** (server `tickUpdate` and `GameLoop.processAffectDecay`), halving
  every spell duration. The server copy is removed.
- **Still open (pre-existing, not fixed):**
  - HP/mana/move regen also runs twice per tick (server `tickUpdate` and
    `GameLoop.processRegeneration`). Fixing it halves regen game-wide, which is a balance decision.
  - Armour AC values are never applied on equip.
  - Buff affects are re-added on load without their stat modifiers but reversed on expiry, so a
    save/load mid-buff permanently lowers the stat.

## Relation to existing plans

- **ECONOMY.md E3** already specifies the same 12 shared slots and GodWars-style salvage. Demon armour
  should be the first concrete E3 "set", not a parallel system.
- **ECONOMY.md E8** (gods + favor) has the same shape as demon power (sacrifice → currency → boons).
  Keep separate currencies; a dark god may later grant demon power as a boon.
- **Roadmap Phase 7/8**: T0's composition point is a preview of Phase 7; tier-2 classes join the Phase 8
  TOML migration.

## Race limits (as built; tune in `ClassTable`)

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
