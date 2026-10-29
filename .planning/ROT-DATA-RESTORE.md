# ROT data restore and play-test fixes — roadmap

Status as of 2026-10-27. Branch `feat/teleport-runes` (main checkout), worked in
the worktree `../ROT-MUD-runes` on `feat/teleport-runes-finish` and
fast-forwarded. 33 commits ahead of `master`, **not pushed**.

## Why

Play-testing a new character through mud school kept hitting bugs whose root
cause was the original `.are` → TOML conversion: it silently dropped whole
sections and fields. The original ROT 1.4 source is in
`c_original/Rot1.4OLCGCC4.tar.gz` (area files in `rot/area/`, C in
`rot/src/`), so lost data can be restored from it instead of guessed.

## Restore scripts

`tools/rotrestore/*.py` — one per restored field. Each takes the ROT area dir
and `go/data/areas`, dry-runs by default, writes with `--write`. Run with
`python3 -I` against an extracted copy of the tarball:

```
mkdir -p /tmp/rot && tar xzf c_original/Rot1.4OLCGCC4.tar.gz -C /tmp/rot ./rot
python3 -I tools/rotrestore/objvalues.py /tmp/rot/rot/area go/data/areas
```

| Script | Restores | Re-runnable |
|---|---|---|
| `weapons.py` | weapon class, damage type, flags | **no** — appends lines |
| `doors.py` | exit door/pickproof/nopass + D-reset closed/locked | yes (rewrites flags) |
| `wealth.py` | mob `gold` (ROT wealth, read as copper) | yes |
| `extradescs.py` | object + room extra descriptions | yes (skips done files) |
| `shops.py` | `[mobiles.shop]` from #SHOPS | yes (skips done files) |
| `roomflags.py` | room flags, heal/mana rates | yes |
| `objvalues.py` | magic/drink/food/money/container/light/portal values | yes (skips existing sections) |
| `mobs.py` | mob alignment, start/default position, mana dice, act/affect/shield/off flags | yes |
| `audit_mobs.py` | **audit only**: per-field mob comparison (flag-name mismatches like `nopurge`/`no_purge` and two-letter flags show as false differences) | — |

ROT conventions to keep: wealth/money is silver in ROT, read as copper here;
ROM shop close hour is inclusive, written +1 (Shop.IsOpen is exclusive); mob AC
in area files is ×10 in game units; flags are letters or decimal bitmasks.

## Done

Data restored from ROT: weapon types (320), door state (734 resets), mob wealth
(1341), object + room extra descriptions (462 + 954), shops (81, guessing
removed), room flags (2660) + rates, object values (507 items).

Code fixes from play-testing (all with tests): teleport-rune WIP, brief mode,
train only at trainers, adept spells target the player, duplicate server regen
+ affect tick, mob AC default, default prompt, creation group refund + hint,
coins to purse (get/autoloot), combine default + comm flags saved, shop output
wiring, shop multi-keyword match, shop carry limits, one object-creation path,
weapon damage types on spawn, compare by wear slot, lore value in coins,
parry/dodge only on melee hits, ROT sacrifice payout, extra descriptions on
look/read/examine, room flags parsed in full, container key slot, everlasting
lights, debug logging (`rotmud -debug`).

Features: `bind` + working word of recall (recall command always Midgaard),
weapon commissions (`commission`/`collect`, `data/forge.toml`, three smiths),
economy sinks (level-scaled healer, training fees above L10, inn `rent`),
Midgaard shops ≤ L10, shopkeepers unattackable, thief spell groups
(shadowcraft, beguiling), charm person registered.

## Next

### 1. Mob fields — done 2026-10-29

`mobs.py` restored alignment (1013 mobs), start/default positions (95),
mana dice, and the ROT act/affect/shield/off flags (sanctuary on 246 mobs,
invisible, protect evil/good, elemental shields, assist_*). Race, AC,
hitroll, damage type and imm/res/vuln were already restored in Phase 16
(`tools/rot-import`). Spawn now uses positions, mana and every flag; mobs
go back to their default position after a fight (ROM `stop_fighting`) and
only standing mobs wander.

Balance: the mob HP floor is on effective HP, so a sanctuary mob gets half
the floor. L30-L60 sim results are back in range; L100 sanctuary mobs whose
own HP is above the floor (Rachal, the Alchemist) are now much harder for
melee classes, as in ROT.

Left over: `assist_*` off flags are in the data but nothing reads them (no
mob assist AI yet; ROM `check_assist`); fire/ice/shock shields set
`ShieldedBy` but deal no damage to attackers; iron/wood imm/vuln have no Go
flag; `CanSee` hides a hidden mob even while it fights (ROM shows it).

### 2. Resets

- 86 `P` resets (objects put in containers) are missing — containers spawn
  empty. Needs a TOML shape for container contents + reset support.
- ~183 `M` resets missing (3009 in ROT vs 2826) — find which and why.
- 97 `R` resets (randomised exits) — check whether the Go reset system has
  any support.

### 3. Spells referenced by items but not registered

cancellation, faerie fog, weaken, change sex, mass invis, enchant weapon,
enchant armor, detect poison, know alignment, gas/lightning/frost breath,
quest pill. Several have spell functions already (like charm person did) —
register those; the rest are stubs or missing.

### 4. Smaller follow-ups

- Furniture values (sit/rest bonuses) — not restored; `hitGain` has a TODO.
- Teleport/summon/gate spells: the magic `CharacterToRoom` hook was never
  called before word of recall — check they actually move characters.
- Ventriloquate is a stub; excluded from thief spells.
- Bless gives +13 hitroll / 98 hours from the L60 adept; ROM is level/8 and
  6+level hours.
- d20 to-hit roll uses `NumberBits(5) % 20` (biased low); ROM rerolls ≥ 20.
  Fixing it shifts all hit rates — decide with the combat sim.
- Mob count at boot varies 2451/2452 — unexplained.
- City watch reacting to attacks on shopkeepers — skipped since keepers are
  unattackable; revisit if wanted.
- Midgaard guildmasters can't `train` (ROT original); trainer is the sailor
  (Abandoned Warehouse, `s s s e e e s` from the temple).

### 5. Land the branch — done

`feat/teleport-runes` is merged into `master`.
