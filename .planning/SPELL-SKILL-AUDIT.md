# Spell & skill audit — 2026-10-29

Read-only audit of master at `6efa742` (after the item-spells, mob-assist,
furniture and d20 merges). Every registered and table-listed spell and skill
was checked for: learnable by some class (or cast by items/mobs), really
implemented, and its effect read somewhere. Checked by reading code, not by
play-testing each one.

**Status: awaiting decision.** Nothing below has been changed yet.

## Proposed plan

- **Cut** the dead/pointless group.
- **Fix** the bug group.
- **Needs-system** group: fix pass door, cut farsight and infravision; leave
  hunger, sneak, animate, feed and resurrect until it's decided whether those
  systems are coming.

## Cut — dead or pointless

| name | type | classes:level | evidence | note |
|---|---|---|---|---|
| ventriloquate | spell | none (unregistered) | function unreferenced (spells.go:1916); still in mage "illusion" group (commands_skills.go:837); gaining a group adds names without checking they exist (commands_skills.go:757) | drop from group, AllSpells and login.go table too |
| create rose | spell | none (unregistered) | spells.go:2369 dead | cosmetic |
| transport | spell | 103 for every class | spells.go:2252 dead | |
| floating disc | spell | none (unregistered) | spells.go:2355 and hook system.go:710 dead | |
| continual light | spell | none (unregistered) | spells.go:1935 dead; `Room.IsDark` has no callers (types/room.go:164) | needs darkness |
| voodoo, protection voodoo | spell | none | levels only for "necromancer"/"warlock", not classes (spells.go:564, 584); voodoo needs body parts; `voodoo` command applies the protection affect itself (commands_misc.go:383) | |
| implode, disintegrate | spell | Mag40, Mag45 | 20d10+2L/30 mana and 30d10+3L/40 mana (spells.go:2491, 2503) vs acid blast dice(L,12)/20 mana at Mag30 (spells.go:1667); acid blast wins at every level (~260 vs 190/220 at L40) | not ROT spells; or retune |
| control weather | spell | Cle17 Ran16 Dru10 | weather state never read outside weather.go | cosmetic |
| fireproof | spell | Mag13 Cle16 Dru6 | burn-proof only blocks envenom (commands_consumables.go:878, 913) and heat metal (spells.go:2616); fire never destroys items | |

## Fix — bugs

| name | type | classes:level | evidence |
|---|---|---|---|
| earthquake, call lightning, chain lightning, meteor swarm, holy word, unholy word | spell | various | hit everyone in the room (spells.go:1611, 1750, 2475, 2515, 2538); `combat.IsSafe` checks only self, safe room, immortal, protected (combat.go:110-132) — exclude group, pets, players |
| calm | spell | Cle16 Ran45 Dru23 Gho30 | registered offensive (spells.go:367; ROT is target-ignore, spells_data.go:122); `Cast`'s offensive tail calls SetFighting after calm's StopFighting (system.go:756-764) |
| call lightning | spell | Cle18 Ran15 Dru24 | `Sector == 1` is SectCity, not SectInside=0 (spells.go:1634, constants.go:114) |
| empower | spell | Mag35 Cle40 Dru45 | always returns false (spells.go:2790-2805) after 100 mana is spent |
| pick lock | skill | Thi7 | skill never read; `pick` uses dex only, any class can pick (commands_door.go:411-495; ROT act_move.c:1085) |
| plague | spell | none (unregistered) | function unreferenced (spells.go:1854); mob AI casts it (ai/specials.go:237,305,370) and fails; sold in maladictions group (commands_skills.go:838) |
| portal, nexus | spell | none (unregistered) | spellPortal/spellNexus (spells.go:2194, 2140) unreferenced; hooks system.go:697-708 dead |
| recharge | spell | none (unregistered) | spells.go:2280 unreferenced; charged wands/staves exist |
| protection evil/good | spell | Cle9 Ran13 Dru7 | flags never read in combat; only effect is -1 saves (spells.go:1211, 1228) — add ROM damage reduction |
| energy drain | spell | Mag35 Gho10 | exp drain and mana halving left TODO (spells.go:1709) — minor |
| quest pill | spell | items only | 3 items (limbo objects.toml:325, fanatics objects.toml:361 ×2) resolve to slot 0; ROT magic2.c:631 gives a quest point, port has none |
| login spell table | — | — | login.go:1209 prints ROT's 1–101 levels (acid blast 55), not learnable levels (mage 30) |

## Needs a missing system — decide

| name | type | classes:level | missing system | evidence |
|---|---|---|---|---|
| pass door | spell | Mag24 Cle32 Thi30 Dru23 Gho24 | door check | AffPassDoor never read; doMove blocks all closed doors (commands.go:2183) — small fix |
| farsight | spell | Mag14 Cle20 | scan | AffFarsight (spells.go:1539) never read |
| infravision | spell | Mag9 Cle13 Ran3 Dru6 Gho3 | darkness | `Room.IsDark` has no callers |
| create food/water/spring | spell | Cle3/4/17 Ran Dru | hunger/thirst | commented out (loop.go:283-291) |
| sneak | skill | Thi4 Ran10 Gho5 | move messages | AffSneak read only by `sneak` (commands_thief.go:19); doMove sends no leave/arrive messages |
| animate | spell | Gho23 | body parts | needs objects 12-17 (spells.go:2752); death cry only sends messages (combat/damage.go:396) |
| feed | skill | none | vampire power | all class levels 0 (defaults.go:272); DoFeed needs a learned value (combat/skills.go:596) |
| resurrect | spell | "vampire" only | vampire power | remort classes learn as their base class (character.go:230, tier.go:61) |

Low value, keep: detect poison (hunger off), fly (no terrain checks; only trip reads it), refresh (walking costs no move), word of recall (`recall` skill is L1 for all), change sex (6 items + spec_cast_mage).

## OK

**Spells (73):** acid blast, acid breath, armor, bless, blindness, burning hands, cancellation, cause critical, cause light, cause serious, charm person, chill touch, colour spray, conjure, cure blindness, cure critical, cure disease, cure light, cure poison, cure serious, curse, demonfire, detect evil, detect good, detect hidden, detect invis, detect magic, dispel evil, dispel good, dispel magic, enchant armor, enchant weapon, faerie fire, faerie fog, fire breath, fireball, fireshield, flamestrike, frenzy, frost breath, gas breath, gate, general purpose, giant strength, harm, haste, heal, heat metal, high explosive, iceshield, identify, invisibility, know alignment, lightning bolt, lightning breath, locate object, magic missile, mass healing, mass invis, poison, ray of truth, remove curse, restore mana, sanctuary, shield, shocking grasp, shockshield, sleep, slow, stone skin, summon, teleport, weaken.

**Skills (42):** second–fifth attack, dual wield, dodge, parry, shield block, enhanced damage, grip, kick, bash, trip, dirt kicking, disarm, gouge, stun, backstab, circle, berserk, rescue, hand to hand, hide, steal, peek, envenom, track, venom, assassinate, sword, dagger, spear, mace, axe, flail, whip, polearm, meditation, fast healing, haggle, lore, recall, scrolls, staves, wands.

## Other open items (balance calls)

Found during the 2026-10-29 restore work, not changed:

1. **Combat may run twice per tick** — violence update called from both `server.go` and `loop.go`; would double combat speed and assist checks.
2. **372 mob resets never fire** — Go allows one copy of a mob vnum per room; ROT resets carry a per-room limit often ≥2 (e.g. Moria room 4111 resets mob 4102 twice). 244 room/mob pairs. Fix = restore the limit to area data; adds a few hundred mobs.
3. **Multi-affect spells lose their second affect** — shared add-affect merges a spell's second affect into its first (frenzy loses damroll/AC). Bless fixed locally only.
4. **Breaths are mob/item-only** — ROT lets mages learn them; with ROT damage they topped the mage list and broke the sim.
5. **Charm duration** — ROT `number_fuzzy(level/4)`; Go `level/4 + 12`.
6. **Summon/gate** skip clan PK and can't-summon-shopkeeper checks; gate skips the can-see-room check.
7. **Breath item burning** (fire/frost damaging carried items) not ported.
