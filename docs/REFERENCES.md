# Reference Sources

External codebases and original data this port draws on, and how to get them
on a fresh machine. Nothing here is needed to build or run the server.

## GodWars Deluxe (tier-2 classes)

The tier-2 supernatural classes (demon, vampire, werewolf, magus, highlander,
angel), demonic armour and powers are modelled on GodWars. Design and mapping:
[`.planning/TIER2-GODWARS.md`](../.planning/TIER2-GODWARS.md).

| | |
|---|---|
| Repository | https://github.com/benjamin-small/GodWars (maintenance fork of GodWars Deluxe) |
| Commit studied | `9f54659a171815cd457eb3535dca5306a45d549d` (2026-08-28) |
| Local path used | `~/Repos/Misc/GodWars` (sibling of this repo; not a submodule) |

```bash
cd ~/Repos/Misc
git clone https://github.com/benjamin-small/GodWars.git
git -C GodWars checkout 9f54659a171815cd457eb3535dca5306a45d549d   # optional: exact version studied
```

Where to look:

| Topic | File |
|---|---|
| Class bits and hybrids | `src/merc.h` (`CLASS_*`, `DEM_*`, `ARM_*`) |
| Demon powers (`inpart`), demon/angel armour, demonform, weaponform, pact | `src/demon.c` |
| Demonic armour set bonus on equip/unequip | `src/handler.c` (`SITEM_DEMONIC`) |
| Armour templates (vnums 29650–29661) | `area/relic.are` |
| Power gain from kills | `src/fight.c` |
| Vampire embrace, werewolf, clans/tribes | `src/clan.c`, `src/werewolf.c` |

**License:** GodWars file headers still carry Richard Woolcock's "not to be
copied without permission" notice. Port mechanics and ideas only — no code.

## ROT 1.4 original C source and areas

The Go port was converted from ROT 1.4. The original is in this repo:

```bash
mkdir -p /tmp/rotc && tar xzf c_original/Rot1.4OLCGCC4.tar.gz -C /tmp/rotc
# sources: /tmp/rotc/rot/src    areas: /tmp/rotc/rot/area
```

Useful when checking what the conversion kept: `src/db2.c` (ROM mob/object
file format), `src/db.c` (`create_mobile`), `src/const.c` (`attack_table`,
`race_table`), `src/act_comm.c` (`do_reroll`).

## Mob data re-import (`tools/rot-import`)

The original TOML conversion dropped mob hitroll, AC, damage type,
off/imm/res/vuln flags, size, race and the ranger/druid/vampire act flags.
They were restored (commit `c70b10a`) with these one-off scripts, kept so the
import can be re-run or audited:

```bash
python3 tools/rot-import/import_mobs.py /tmp/rotc/rot/area go/data/areas           # dry run: match report
python3 tools/rot-import/import_mobs.py /tmp/rotc/rot/area go/data/areas --write   # write fields
python3 tools/rot-import/import_act.py  /tmp/rotc/rot/area go/data/areas           # act class flags
python3 tools/rot-import/import_race.py /tmp/rotc/rot/src  go/data/areas           # race_table flags (idempotent)
```

Mobs are matched by vnum and short description; `chess1.are`, `guard.are` and
`mystica.are` do not parse and are not used by the Go world. Running
`import_mobs.py --write` twice duplicates lines — run it on unimported data only.
