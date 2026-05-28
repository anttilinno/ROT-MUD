# Economic Overhaul Plan

## Status

- **Monetary system:** Done. `pkg/types/currency.go` defines 4-tier denominations (cp/sp/gp/pp) stored internally as int64 copper. Shops, bank, deposit/withdraw, mob drops, corpse loot, and death penalty all converted.
- **Smith / crafting question (resolved):** Smiths craft equipment as a money sink, paired with hits-based durability + repair. Repair gives a universal recurring drain; crafted gear gives endgame goal and identity. The craft system is **race-aware, class-aware, 3-tier level-gated**, and the top tier is **best-in-slot** at level cap (inverts the earlier "always ≤ area drop" cap). Top-tier recipes require boss-drop materials, keeping area content relevant.

## Goals

- Net coin OUT ≥ net coin IN at steady state per level bucket.
- Distinct sinks per progression band:
  - L1–30: trivial drains (repair, identify, bank fees, T1 crafting tutorial)
  - L31–75: gear maintenance dominates; T2 crafting fills slots
  - L76–100: T3 crafting is the endgame goal; boss-material grind feeds it
- Reuse existing currency tiers — no new denominations.
- Crafting carries class + race identity, not just stats. Visual + lore differentiation matters.

## Baseline (as of 2026-05-28)

From `TestMobCoinDropDistribution` (`go/pkg/loader/mobcoin_sim_test.go`):

| Bucket  | Mobs | Median drop | P90 drop |
|---------|-----:|-------------|----------|
| L1-5    |   79 | 2s 8c       | 6s       |
| L6-10   |  115 | 7s 9c       | 1g 2s 6c |
| L11-20  |  228 | 1g 4s 5c    | 2g 3s 8c |
| L21-30  |  166 | 2g 4s       | 3g 8s 3c |
| L31-50  |  273 | 4g          | 6g 3s 8c |
| L51-75  |  232 | 6g 1s 5c    | 9g 2s 8c |
| L76-100 |  144 | 8g 3s 5c    | 1p 3g 8c |
| L101+   |   98 | 1p 3s 3c    | 1p 6g 4s 9c |

Single-pass world kill total: **~590p**. Zero mobs use explicit `Gold` field — drops are fully procedural from level.

Current sinks: 10% carried-coin death penalty (`pkg/combat/damage.go:307`), shop buys at `ProfitSell` margin (default 100% = base cost). No durability, no repair, no crafting, no identify fees, no bank fees.

## Phases

### E1 — Baseline + telemetry

**Goal:** Measure current inflation rate before tuning.

- Lock current `mobCoinDrop` numbers as a golden-master fixture (deterministic seeded RNG sweep across all mob templates).
- Add a coin ledger: every credit/debit on `ch.Coin` and `ch.PCData.BankCoin` logs `(txn_type, amount_copper, source_id, target_id, tick)` where `txn_type ∈ {mob_drop, corpse_loot, shop_buy, shop_sell, death_loss, bank_deposit, bank_withdraw, transfer, quest_reward, craft_fee, repair_fee, identify_fee, salvage_refund, …}`.
- Simulation test: walk N kills × M synthetic players per level bucket; dump source/sink ratio.
- **Exit criteria:** A reproducible report shows current inflation rate per level bucket. This becomes the regression baseline for E2–E6.

### E2 — Durability + repair (universal drain)

**Goal:** Every active player loses coin per session proportional to play time.

- Add `Object.DurabilityCurrent` and `Object.DurabilityMax` on weapons and armor only (not money, not consumables).
- Durability decrement model: **hits-based**, not time-based.
  - Worn armor: −1 per hit taken (only on hit, not on miss).
  - Wielded weapon: −1 per swing landed.
- At 0 durability: item "broken" — stats halved, wear-fail noise on combat round, item not destroyed.
- `repair <item>` command at smith NPCs (new `ActSmith` flag).
  - Cost formula: `((max - cur) / max) * item.Cost * repairMult` where `repairMult` is tunable per item tier (start 0.25).
  - Atomic: deduct coin, restore to max, or error.
- TOML: `durability_max` field on object templates; default by item tier (weapon=200, armor=150) when omitted.
- Crafted gear durability is inherent to its tier (see E3); broken crafted gear is repaired by the same smith who can craft its tier.
- **Exit criteria:** Ledger shows non-zero `repair` debits in every active session; sim of N hours of combat shows steady drain against drops.

### E3 — Race + class crafting (3-tier, level-gated, set bonuses)

**Goal:** Crafted gear carries race and class identity, fills 12 slots + 1 race-signature slot, and is the canonical endgame gear path. Crafting is the primary smith money sink; top tier is best-in-slot and demands boss-drop materials, keeping area content relevant.

#### Slot model

- **12 shared slots** (same on every race, class-restated): Ring, Collar, Plate, Helmet, Leggings, Boots, Gauntlets, Sleeves, Cape, Belt, Bracer, Visor.
- **1 race-signature slot** per race, unique to that race (vampire fang-amulet, dwarf beard-clasp, elf soul-gem, demon horn-circlet, etc.). 19 race-signature recipes total. Hard-locked: a vampire cannot wear a dwarf beard-clasp.

#### Three tiers, level-gated

| Tier | Wearable from | Power vs same-L area drop | Role |
|------|--------------:|---------------------------|------|
| T1   | L1            | Below area drop           | Tutorial / coin-sink intro. Mediocre filler. Coin-only craft cost. |
| T2   | L31           | ≈ area drop               | Useful, common path. Coin + craft-XP cost. |
| T3   | L76           | At or above best area drop at L100 | Endgame best-in-slot. Coin + craft-XP + boss-drop materials. |

Wearing below the level gate fails (`"You are not yet strong enough to wear this."`). Wearing above is always legal.

#### Race-class affinity (not lock)

- Each race has a canonical class affinity (data, in race TOML — `class_affinity = "warrior"`).
- A character whose race+class matches the affinity unlocks the **set bonus** path.
- Off-affinity characters can craft and wear the same gear, get base stats per piece, but never the set bonus. They also have a **T3 cap one tier below** — off-affinity caps at T2 endgame (still viable, not best).
- Encodes the "race-class fit" idea without locking players out of mix-and-match builds.

#### Set bonuses

A full crafted set is 13 pieces (12 shared + 1 signature). Tier of the set is the lowest-tier piece worn (`min(tiers)`). Bonuses scale with piece count:

| Pieces worn (same tier, same set) | Bonus |
|-----------------------------------|-------|
| 4                                  | small (+5% relevant stat — HitRoll for warriors, mana regen for mages, etc.) |
| 6                                  | medium (small bonus + 1 class capability flag — e.g. extra attack for warriors) |
| 8                                  | large (medium bonus + 1 race trait — e.g. fire resistance for demon, regen for troll) |
| 13 (full)                          | signature bonus: unique class+race ability tied to the recipe (e.g. warrior+dwarf full set: "Mountain Stance" — +20% damage when not moved this round) |

Set bonuses fire only for on-affinity characters. Off-affinity gets piece stats but no set bonuses.

Mixed-tier sets degrade: a player wearing T3+T2+T2 gear gets the **T2** set bonus (lowest tier present). Encourages full-set crafting rather than min-maxing one piece.

#### Recipes (data, not code)

- `data/recipes/<class>/<tier>/<slot>.toml` for class-restated shared slots.
- `data/recipes/race-signature/<race>.toml` for race-unique signature pieces.
- Recipe schema:
  ```
  name = "Warrior's Battle Plate"
  base = "plate"            # slot
  class = "warrior"         # who can craft
  tier = 2
  level_req = 31
  coin_cost = 5000          # copper
  craft_xp_cost = 50
  
  [stats]
  ArmorPierce = 12
  Hp = 30
  HitRoll = 2
  
  [[materials]]
  vnum = 9201               # iron ingot
  qty = 4
  
  [[materials]]
  vnum = 9203               # leather strap
  qty = 2
  
  # T3-only recipes also declare boss_material entries:
  [[boss_material]]
  vnum = 9501               # "shard of the dragon", drops only from boss mob 12001
  qty = 1
  ```

- T3 recipes are **recipe-specific** for boss materials: warrior T3 plate needs `dragon_scale` (boss 12001); mage T3 robe needs `lich_phylactery` (boss 12042). No universal endgame material. Each boss drives a specific recipe demand, scattering the endgame raid pressure across the world.

#### Two-resource gating

- **Coin** — paid to smith on craft. Scales with tier.
- **Craft XP** — earned by repairing, salvaging, and crafting (not by buying or killing). Bounded pool, ticks up per action. Prevents pure-coin pay-to-craft; rewards smith-loop engagement.
- **Materials** — drop from regular mobs (T2 commons) or specific bosses (T3 boss materials).

Each axis blocks pure-grind in another dimension: coin alone can't get you T3; mob-kill grind alone can't either; nor can quest tokens. All three required.

#### Crafting commands

- `craft <recipe-name>` at master smith → quote (coin + craft-XP + materials) → confirm → spawn bespoke item instance, deducting all three.
- Crafted items are **bind-on-pickup** (`ItemFlagBound`) — non-tradeable, non-sellable except as salvage. Prevents alt-funneling.
- `salvage <item>` at smith → recovers ~80% materials, ~50% coin, full craft-XP. Item destroyed. Lets players experiment without lock-in (GW demon-armor sacrifice mechanic).
- `inspect <item>` shows tier, set name, set bonus pieces currently active.

#### Cosmetic identity

- Each tier has a visible appearance descriptor on the item short-desc: T1 base, T2 `gleaming`, T3 `ornate`. Race signature adds race tag (`a dwarf-forged ornate plate`).
- Visible in `look <player>` output for room status. Cheap social-flex dopamine; rewards completionism.

#### Power-cap rule (replaces earlier "≤ area drop forever")

- T1: caps at 80% of same-L area drop stats.
- T2: caps at 100% of same-L area drop stats.
- T3 (on-affinity, full set): caps at 130% of best L100 area drop. **This is now best-in-slot.**
- T3 off-affinity: effectively capped at T2 (no T3 tier unlock for off-affinity).
- Hard test fixture iterates every recipe + set bonus combo and asserts the cap against the area drop manifest.

#### Boss-material loot

- Boss mobs flagged `drops_craft_material = ["dragon_scale", ...]` in mob TOML.
- Material drops bypass autosac and are flagged `loot_priority = high` for autoloot.
- Crafted T3 material vnums are new item type `ItemTypeMaterial`.
- Material drop rate is **deterministic per kill** (not RNG) at start — 1 boss kill = 1 material. Re-evaluate based on E6 telemetry.

**Exit criteria:**

- All 14 classes have full T1/T2 recipe sets (12 slots × 3 tiers minus T3 for now); T3 lands as boss mobs are flagged.
- Race-signature recipe for each of the 19 races.
- Set bonus tests pass: 4/6/8/13 piece bonuses fire correctly, on-affinity only.
- Power-cap test fixture passes: no recipe + set combo exceeds the tier cap.
- Salvage round-trip recovers expected % of resources; ledger records `craft_fee` and `salvage_refund` correctly.

### E3.5 — Magical enchantments (high-coin sink + risk drama)

**Goal:** Add one magical enchant slot per qualifying item. Big-coin sink for endgame players; risk model gives crit-fail drama and a brick-risk pressure on master-tier attempts.

**New NPC type: enchanter.** Located in the mage quarter / arcane hubs. Distinct from smiths.

**Eligible items:**
- Crafted T2 and T3
- D2-style Magic / Rare / Set / Unique found items (see E7)
- **Not** eligible: crafted T1, shop commodity items, quest rewards (already unique), money, materials, reagents

**One enchant slot per item.** Re-enchanting first requires `scour <item>` at the enchanter — strips current enchant, costs coin (no refund). Second sink layer that punishes indecision.

**Three difficulty tiers, three risk profiles:**

| Tier      | Success | Coin cost (vs item.Cost) | Fail consequence                      |
|-----------|--------:|--------------------------|---------------------------------------|
| Simple    |  95%    | 50%                      | Coin lost; reagents kept              |
| Greater   |  75%    | 100%                     | Coin + reagents lost; −1 dur tier     |
| Master    |  40%    | 200%                     | Coin + reagents lost; 10% chance item destroyed |

`destroyed` on master-fail removes the item from inventory entirely. Surfaces odds explicitly in NPC dialog so players know the gamble (`"The enchanter eyes your blade. 'Master enchant: forty percent success, ten percent ruin. Pay 12g.'"`).

**Reagents (new item type `ItemTypeReagent`):**
- Drop from spell-using mobs (mages, demons, liches). Reagent type tied to mob's magic school.
- Examples: `wisdom_dust` (mage drops), `hellfire_essence` (demon drops), `void_shard` (lich drops), `seraph_feather` (paladin/divine drops).
- Reagents are tradeable on the player market (unlike crafted gear, which is BoP). Creates a player-driven reagent economy.
- Each enchant recipe requires 1-3 reagent types in specific qty.

**Enchant pool** (data in `data/enchants/*.toml`):
- **Stat boosts:** `HitRoll +N`, `DamRoll +N`, `Mana +N`, `Hp +N`, `Resist <school> +N%`
- **Capability flags:** `SlayUndead`, `ColdBrand`, `FireBrand`, `Silver`, `Holy`
- **Proc triggers:** `OnHit: cast sanctuary 5%`, `OnHit: drain 3hp`, `OnTakenDamage: shield 10%`

Enchant power scales with tier: Simple gives `+2 HitRoll`-class boosts, Greater gives `+5 HitRoll`-class, Master gives `+8` plus a capability flag or proc.

**Commands:**
- `enchant <item> <enchant-name>` at enchanter → quote (coin + reagents + tier difficulty + odds) → confirm → roll.
- `scour <item>` at enchanter → strip current enchant, no refund.
- `inspect <item>` (already in E3) shows the active enchant if any.

**Interaction with crafted set bonuses (E3):**
- Enchants are item-level, not set-level. Adding enchants does not break set membership.
- A full crafted set with 13 enchanted pieces is the absolute endgame BIS. Long, expensive grind to assemble.

**Exit criteria:**
- Enchanter NPC type wired; three tier commands work.
- Reagent drops from at least three mob archetypes (arcane / divine / chaotic).
- Brick rate observed in sim sits at ~10% on master-fail; full pipeline (coin sink + reagent sink + brick chance) verified via ledger.
- Power-cap fixture extended to include enchant deltas: enchanted T3 full-set still bounded within design ceiling.

### E4 — Identification + appraisal (info sink)

**Goal:** Compulsory mid-game drain on every meaningful drop.

- Items with vnum-level ≥ 20 drop with hidden affixes.
- Display shows base name + `(unidentified)`; stats hidden until identified.
- New sage NPC type (or reuse mage guild mobs). `identify <item>` command, flat fee = `item.Cost * identifyMult` (start 0.10).
- Items vnum-level < 20 unaffected.
- Crafted items are auto-identified (you made it; you know what's in it).
- Boss-drop materials are auto-identified (their purpose is recipe-specific; ID would be busywork).
- Edge cases:
  - Money objects (`ItemTypeMoney`) never need ID.
  - Quest reward items can flag `auto_identify = true` to skip the fee.
- **Exit criteria:** Every L20+ mob drop creates pressure to spend a fraction of the drop value on ID.

### E5 — Bank fees + transfer

**Goal:** Small constant trickle on coin movement; cannot dodge sinks by hoarding in bank.

- Each bank NPC has a `home_city` field.
- `deposit` at any banker: free.
- `withdraw` at non-home banker: `withdrawFeePct` (start 2%) charged off the top.
- `transfer <player> <amount>` at banker: small flat fee (start 1g) plus 1% of amount.
- Withdrawals at home city free (rewards travel cost).
- **Exit criteria:** Ledger shows `bank_fee` debits trickling steadily; coin movement no longer free.

### E6 — Faucet rebalance

**Goal:** Re-tune mob drops down where E2–E5/E3.5 sinks now compensate. End state: source/sink ratio ≈ 1.0 per bucket.

- Re-run E1 sim with E2/E3/E3.5/E4/E5 sinks active; measure new equilibrium per level bucket.
- Adjust `mobCoinDrop` curve coefficients (per-level multiplier).
- Death penalty: 10% → 5% of carried coin (sinks cover loss-pressure).
- Boss-material drop rate: revisit deterministic-per-kill default. If T3 crafting is too easy, drop to 50% per kill and add a guaranteed pity-timer (5th kill always drops).
- Reagent drop rate (E3.5): tune so enchanting a single T3 set member at master tier is 1-2 hours of focused arcane-mob farming.
- Golden master from E1 updates to new baseline.
- **Exit criteria:** Sim ratio sink/source within ±10% of 1.0 per bucket; golden master locked.

### E7 — Loot lottery + damaged drops (Diablo 2 style)

**Goal:** Mob-killed equipment drops at degraded durability (lore + repair sink amplifier). Parallel RNG-loot path to crafted gear: rarity tiers (Magic / Rare / Set / Unique) with affix rolls scoped to base item type and item-level.

#### Damaged drops

- When a mob with worn equipment dies, dropped gear inherits damage proportional to fight length:
  ```
  drop_durability = max(0.10, 1 - dmg_taken_pct) * baseDurabilityMax
  ```
- One-shot kills → near-pristine drops (fast farming preserves loot value).
- Long grindy kills → beat-up drops, immediate repair pressure.
- Floor at 10% so dropped gear is always wearable briefly.

**Exemptions (pristine):**
- Items inside containers / chests (hoarded, not worn).
- Quest-reward items.
- Shop inventory.
- Items withdrawn from bank.
- Boss-mob "shard" materials (already an `ItemTypeMaterial`, not worn gear).

#### Rarity tiers

| Rarity   | Color tag | Affix slots             | Drop weight | ilvl gate    |
|----------|-----------|-------------------------|-------------|--------------|
| Normal   | white     | 0                       | 70          | none         |
| Magic    | blue      | 1 prefix + 1 suffix     | 25          | mob L ≥ 10   |
| Rare     | yellow    | up to 3 prefix + 3 suffix | 4         | mob L ≥ 25   |
| Set      | green     | preset (member of a unique set) | 0.8 | mob L ≥ 40 |
| Unique   | gold      | preset, item-unique     | 0.2         | mob L ≥ 50   |

Rarities renamed to disambiguate from crafted set bonuses (E3):
- **Crafted Sets** = the 13-piece craft system from E3 with affinity-gated set bonuses.
- **Lottery Sets** = D2-style world-found set members (e.g. "Sigon's Set"). Used as the label in player-facing text to avoid confusion.

#### Magic Find

- Player stat `MagicFind` accumulates from gear, certain quests, and consumables.
- Roll-time formula: `rarity_weights[i] *= 1 + MagicFind/100` for non-Normal rarities; Normal weight unchanged.
- Soft cap at +300% MagicFind; diminishing returns above.
- Surfaces as a column in `score` output.

#### ilvl + affix pools

- `ilvl = max(mob.Level, area.Level + 5 if mob.is_boss)`. Boss kills always roll on at least area level + 5.
- Affix pools live in `data/affixes/<base_type>.toml`:
  - prefixes: `data/affixes/<base>/prefixes.toml`
  - suffixes: `data/affixes/<base>/suffixes.toml`
- Each affix declares `min_ilvl`, `max_ilvl`, value range, weight, and conflict tags (an item never rolls two affixes from the same conflict group).
- Affix scoping: certain affixes only spawn on certain bases (e.g. `of the Whale` only on body armor; `Sharp` only on weapons). Matches the D2 base-restriction model.

#### Set + Unique tables

- `data/loot/sets/*.toml` — each set TOML declares members (item base + preset affixes) and the set bonus stacking (2-piece / 4-piece / etc.).
- `data/loot/uniques/*.toml` — each unique declares a base item, preset affixes, lore text, and a `weight` in its drop bucket.
- A mob's drop spec can opt out of set/unique pool via `lottery_excludes = ["set", "unique"]` for low-quality vendors / target-dummy mobs.

#### Identify (E4 interaction)

- Magic / Rare / Set / Unique drops at `mob.Level ≥ 20` arrive unidentified (E4 already covers this).
- Identify cost scales with rarity: Normal = 0, Magic = base, Rare = 2×, Set = 4×, Unique = 8×. Surfaces a real gamble — vendor unidentified gold or pay the ID fee.

#### Interaction with crafted gear (E3)

- Single-slot Rare can exceed T2 crafted in raw stats; single-slot Unique can match T3 single-piece.
- Crafted T3 full 13-piece set with set bonuses + enchants is the absolute BIS — Rare/Unique drops cannot replicate the on-affinity set + signature bonus.
- Found gear is enchantable (E3.5) provided rarity ≥ Magic.
- Crafted gear is BoP; Rare/Unique drops are tradeable (drives a player market for found loot).

#### Auto-loot filters

- New config setting: `autoloot_rarity <threshold>` (`none`, `magic`, `rare`, `set`, `unique`). Only auto-pickup at or above the threshold.
- Players can also `autoloot_dur_min <pct>` to skip items below a durability floor — vendor trash trash is automatically left for the corpse decay.

#### Commands

- `loot` and `get <item> corpse` (existing) now display rarity color and durability percentage.
- `inspect <item>` shows rarity, affixes (if identified), durability, and (if applicable) set/lottery-set membership progress.

**Exit criteria:**

- Mob-killed gear drops at the documented durability formula; chest / quest / shop / bank items remain pristine.
- All four rarity tiers (Magic / Rare / Set / Unique) roll correctly with weighted affix pools scoped by item base and ilvl.
- Lottery Sets and Uniques load from TOML; at least 3 starter sets and 6 starter uniques covering the L40-100 band.
- Auto-loot filters honor rarity and durability thresholds.
- Identify cost scales by rarity; ledger records `identify_fee` correctly per rarity tier.
- Power-cap fixture extended: best in-game outcome (Rare/Unique gear in every non-craftable scenario, on-affinity T3 full crafted+enchanted in the best scenario) is documented and bounded.

### E8 — Gods + favor (worship as sink + progression)

**Goal:** Players choose 1 of N gods. Sacrifices, offerings, and tithes convert items / coin / mob corpses into per-god **favor**. Favor buys divine boons that shortcut progression or grant rare buffs. Recurring tithes plus passive favor decay create a durable sink that bites whether you craft, fight, or loot.

#### Pantheon

- **6–9 gods** at first pass, defined in `data/gods/*.toml`.
- Each god declares:
  - `domain` (war, magic, death, nature, trade, healing, chaos, light, dark)
  - `alignment` (existing `ch.Align` axis repurposed: good / neutral / evil)
  - `favored_offerings` (mob race tags, item types, alignment vectors)
  - `hated_offerings` (blasphemy table — sacrificing these costs favor)
  - `boon_table` (tier → cost in favor → effect)
- Player picks at L10+ via `pray <god>` at a temple. Stays unset for atheists (legal, mild penalty — see Risks).
- Switching gods later: `pray <new-god>` at that god's temple. **Apostasy penalty:** old favor reset to 0; new god starts at -50 favor; 7-day real-time cooldown between switches.

#### Existing `Align` axis repurposed

- `ch.Align` was already a -1000..+1000 int. Now drifts toward the chosen god's alignment based on play (aligned kills, hated kills, sacrifices).
- Drift far from god's alignment → favor accrual rate halves; `pray` rejected with a flavor message ("The god turns from you").
- Atonement quest restores alignment (see Atonement below).

#### Sacrifice + offering mechanics

Extend existing `sacrifice <corpse>` (currently grants 1cp via `server.go` adapter):

| Source                        | Favor delta            | Coin cost            | Notes |
|-------------------------------|------------------------|----------------------|-------|
| Corpse of favored-race mob    | +1 to +5 (level-scaled) | 0                    | aligned kill bonus, replaces today's flat 1cp |
| Corpse of hated-race mob      | -2 (blasphemy)         | 0                    | god is offended; can refuse to count if hated > favored ratio too high |
| `tithe <N>` at temple         | +1 per 100g            | N (consumed)         | flat coin sink, no item |
| `offer <item>` at temple      | +item.Cost / 100       | item destroyed       | universal item sink |
| Boss-material item            | +50                    | rare mat destroyed   | dramatic; competes with E3 T3 crafting |
| God-domain reagent (E3.5)     | +10                    | reagent destroyed    | links enchant economy |
| `consecrate` (PvE-only event) | +20                    | living mob destroyed | dark gods only; gated to specific quest mobs, not arbitrary player kills |

Favor is **per-god**, stored as `ch.PCData.Favor[god_name]`. Saved in player file.

#### Boons (favor spend)

Each god publishes a per-tier boon table. Costs in favor; many also have per-day or per-session cooldowns. Example schema:

| Cost  | Boon                                              | Cooldown      |
|-------|---------------------------------------------------|---------------|
| 10    | `pray` for full heal + mana refill                | 1× per game-day |
| 50    | Cast god-domain T1 spell as if known              | 1× per session |
| 100   | Bless next crafted item (free Simple enchant)     | per item, no cooldown |
| 200   | God-signature affect for 1 hour (regen / fire-brand / etc.) | 1× per session |
| 500   | Unlock god-domain quest chain                     | single-use lifetime |
| 1000  | Resurrect at temple (no XP loss next death this session) | 1× per session |
| 2000  | Divine weapon enchant (master-tier, **no brick risk**) | per item |

Boon table is **god-specific data**. War god offers damage boons; nature god offers regen; trade god offers shop discounts; death god offers undead-control. Power roughly equal across gods; flavor differs.

#### Tithe + corruption (recurring sink)

- Each god has a `tithe_target` per real-time week — typically a small percent (1-3%) of total coin earned.
- Players who hit the tithe target keep favor stable.
- Players under the target → favor decays at `decayRate` per real-time day until target met or favor hits 0.
- Favor at 0 + repeated decay → favor goes **negative**: temple guards aggro on entry; boons disabled; the god's domain mobs deal +10% damage to the player.
- Recovery: `atone <god>` quest. God-domain PvE objective + flat coin offering. Returns favor to 0 and restores boon access.

#### Atheism (no god picked)

- Legal at all levels.
- No boons available.
- ~10% flat penalty on `Hp_max` and `Mana_max` regen rates vs an on-favor worshipper (reasonable choice cost, not a wall).
- Cannot use any temple services (resurrection, identify-god-alignment, etc.).

#### Temple geography

Good and neutral gods keep temples inside good-aligned cities (Midgaard). Evil and chaotic gods are barred from those cities by guards; their temples live elsewhere. **Three forms — all first-pass content:**

- **Wilderness shrines**: small temples in dangerous outdoor zones (e.g. `Bloodmoor`, `Cursed Glade`, `Drow Forest`). Cheap to add via existing area loader. One shrine per evil god, reachable on foot from city gates. Serve as roadside checkpoints between hub cities.
- **Outlaw city: `Shadowport`** (evil-aligned hub): full alternate hub with bank, smiths, enchanters, sages, temple complex, housing market, and MUD school exit destination. Mirrors the Midgaard service set for evil-aligned players. Its guards refuse good-aligned characters above the threshold (mirror-image of Midgaard rules).
- **Outlaw city: `Skullhold`** (chaotic-aligned hub): full alternate hub with the same service set. Caters to chaotic-neutral and chaotic-evil play. Guards enforce a chaos-tolerant ruleset (no aggro on alignment alone; pickpocket / brawl is legal).

Neutral gods (nature, trade) may have shrines in all three hubs plus wilderness; players access whichever they reach first. Good gods refused entry to Shadowport / Skullhold in the same way evil gods are refused in Midgaard — symmetrical.

This makes evil + chaotic play a fully self-sufficient endgame loop. An evil character does not need to bribe their way into Midgaard for daily life; only for cross-faction quest content or specific items only Midgaard merchants stock.

#### City guard alignment enforcement

City guards consult `ch.Align` plus a per-area `respect_alignment` flag (already in E8 risks). Default Midgaard = good-lawful enforcement; outlaw cities flip the rules.

| Player state                                    | Guard reaction in a good-lawful city |
|-------------------------------------------------|--------------------------------------|
| L < 5 (any alignment)                           | Ignored (newbie grace)               |
| L ≥ 5, Align ≥ -350 (neutral or good)           | Welcome                              |
| L ≥ 5, Align -350 to -700 (evil)                | Suspicious — refused entry to temple quarter; normal elsewhere |
| L ≥ 5, Align < -700 (chaotic-evil)              | Aggro on sight; guards attack on city entry |

Alignment drift comes from gameplay (sacrifice/blasphemy, aligned kills). Reaching `Align < -700` is sustained evil play — guard hostility is consequence, not accident.

#### Newbie grace

- Below L5, `ch.Align` is clamped to a floor of -100 regardless of evil-aligned actions. Prevents griefing MUD school (killing good NPCs to spike Align before graduation) and protects the escort-exit choice from being foreclosed.
- Alignment drift unlocks fully at L5.

#### MUD school graduation: escorted exit

Players finish the tutorial inside good-aligned Midgaard. The MUD school's final quest is **destination selection** — the player picks the kind of life they intend to lead, and the guard sergeant routes them out the appropriate gate.

- NPC `Recruit Sergeant` runs a final dialog at MUD school's graduation room.
- Player picks one of: `good`, `neutral`, `evil`, `chaotic`.
- Sergeant flavor line varies by choice ("You've chosen a dark path. Walk it elsewhere — and don't return to my city unless you've cleansed your soul.").
- Destination teleport based on choice:
  - `good` / `neutral` → Midgaard market square
  - `evil` → Shadowport central plaza (full hub with services on arrival)
  - `chaotic` → Skullhold bazaar (full hub with services on arrival)
- **Choice is non-binding** — sets destination hub, not god-pick or alignment. Players can convert later via `pray` and atonement, and travel between hubs via the road network.
- Implementation: escort is a one-way `transfer` command issued by the sergeant. Cheap; no literal pathing.

This solves the trapped-evil-newbie problem without requiring evil players to fight through hostile guards on day one.

#### Cross-faction city access

Most daily play happens inside the player's home hub (Midgaard for good/neutral, Shadowport for evil, Skullhold for chaotic). Cross-faction visits are needed only for specific content: a Midgaard-only quest, a Shadowport-only enchant reagent, a unique merchant in Skullhold.

Each hub enforces alignment thresholds symmetrically (good/neutral aggro in Shadowport; evil aggro in Midgaard; Skullhold is chaos-tolerant but bans paladins / clerics of lawful-good gods). Three cross-faction entry paths apply at any hostile hub gate:

| Path        | Cost                       | Notes |
|-------------|----------------------------|-------|
| `atone <hub-aligned-god>` quest | time + aligned kills | Slow Align drift back toward the target hub's alignment. Permanent until next drift. |
| `disguise` (thief) / `polymorph` (mage) | reagents + skill check | Time-limited entry. Skill-gated. Links to E3.5 reagent economy. |
| `bribe <guard> <amount>` at the gate | coin scaling with Align distance | Time-limited gate pass. Per-real-day limit prevents spam. Big coin sink. |

Each path is intentional friction. Cross-faction play is a choice; hostile hubs remain reachable but never daily.

#### Temple shops (worshipper + cleric paths)

Each god's temple hosts a shop component (separate quartermaster NPC inside the temple room or a flagged temple-keeper mob). Stocks god-domain items: divine reagents (E3.5), domain-themed weapons (war god → maces / blades, nature god → staves / sickles, death god → daggers / cursed amulets), prayer-scroll books, holy symbols, potions, atonement supplies.

**Pricing + access tiers** (per shopper relative to this god):

| Shopper status                            | Buy multiplier | Sell multiplier | Notes |
|-------------------------------------------|---------------:|----------------:|-------|
| Cleric class + this god + favor > 100     | 0.50× (deep discount) | 1.20× | full inventory + cleric-only stock |
| Worshipper (any class) + favor > 0        | 0.75×          | 1.10×           | full standard inventory |
| Worshipper + favor ≤ 0 (lapsed)           | 1.00×          | 1.00×           | standard stock only; cleric-only refused |
| Atheist or worshipper of different but same-alignment god | 1.25×        | 0.90×           | standard stock only |
| Worshipper of **opposing-alignment** god  | refused        | refused         | temple guards may aggro on entry |

**Inventory tiers** declared per item in `data/temples/<god>/inventory.toml`:

- `tier = "open"` — sold to anyone whose status above is not refused.
- `tier = "worshipper"` — requires favor > 0 with this god.
- `tier = "cleric"` — requires `ch.Class == "cleric"` (or a `CapabilityFlag = ClericOf<god>` once trait system lands) AND favor > 100.
- `tier = "favored"` — requires favor > N (per item). Used for unique relics, master-tier divine weapons, prayer-books of high-tier domain spells.

**Currency: coin OR favor.**

- Default purchase deducts coin.
- `pay_with_favor <item>` deducts favor at the listed `favor_price` (e.g. 200 favor for a holy water flask, 5000 favor for a divine weapon).
- Favor-only items have no `coin_price` — they cannot be coin-purchased at any markup. Forces real worship engagement for the iconic items.
- Player can freely choose coin vs favor at point of sale for dual-currency items.

**Cleric-only stock examples:**
- Domain spell prayer-books (cast L+5 spells without learning cost — single-use)
- Divine weapons (master-tier enchanted, no brick risk, BoP-on-purchase)
- Holy symbols (passive +favor accrual rate while worn)
- Atonement supplies (bulk-discount on `atone` quest reagents)
- Resurrection-scroll for party member (consumes 1000 favor)

**Worshipper-tier stock examples:**
- Divine reagents at favorable price (cuts E3.5 enchant cost)
- Blessed potions (cheaper full-heal / mana refill than standard alchemists)
- Identify scrolls for god-domain items only (cheaper than sage)
- Temple recall scroll (faster return to home temple)

**Open-tier stock examples:**
- Basic candles, prayer-flags, journals (low-coin newbie items)
- Atonement starter (for lapsed worshippers)

**Sell-back:** temple buys back god-domain items (returning relics, donating gear) at the matching multiplier. Counts as a small `offer` — sell-back of a god-domain item grants +1 favor in addition to coin.

**Why this matters as a sink:**
- Cleric players get a dedicated economic loop tied to identity, not just generic shops.
- Worshippers have a discount path: tithe to maintain favor, get cheaper reagents/potions.
- Coin sink + favor sink + class differentiation all in one system.
- Opposing-alignment refusal makes god choice meaningful for shopping geography — chaotic-evil players cannot use lawful-good temples and vice versa, forcing travel + market diversity.

#### Integration with prior phases

- **E3 crafting:** god-domain T3 sets unlocked by god-domain boss materials (`seal_of_war`, `relic_of_nature`). On-favor characters with the god-aligned T3 set get an additional 5% bonus on top of class-affinity bonus.
- **E3.5 enchants:** 2000-favor boon **bypasses brick risk** at master tier — turns the god into endgame enchant insurance. Divine reagents (`seraph_feather`, `hellfire_essence`) drop more frequently from god-domain mobs of same alignment.
- **E4 identify:** temples ID god-domain items for free. Mini perk, encourages temple visits.
- **E5 banks:** trade god (if implemented) reduces bank withdrawal/transfer fees by half for worshippers.
- **E7 lottery:** Unique items can carry `god_aligned = "war"`. Worn by:
  - On-favor worshipper → +20% affix values
  - Atheist or different god → equips normally, no bonus
  - **Opposing alignment** worshipper → item refuses to equip (`"The blade burns your hand."`)

#### Commands

- `pray <god>` at temple: initial dedication, or after alignment drift to reaffirm.
- `pray` (no arg): consume small favor (10) for heal + mana refill (per-day cooldown).
- `tithe <amount>` at temple: convert coin to favor.
- `offer <item>` at temple: convert item to favor.
- `boon <name>` at temple: spend favor for a listed boon.
- `atone` at temple: start atonement quest (negative-favor recovery).
- `favor`: display current favor for active god + recent decay/gain history.
- `pay_with_favor <item>` at temple shop: purchase listed item using favor instead of coin (dual-currency items only).
- `bribe <guard> <amount>` at a hostile city gate: per-real-day-limited gate pass for evil characters returning to a good-lawful city.
- `disguise` (thief) / `polymorph self` (mage): time-limited evade of city guard alignment check.

#### Data layout

```
data/gods/<god_name>.toml
  name = "Tyr"
  domain = "war"
  alignment = 500           # good
  favored_races = ["evil_orc", "demon", "undead"]
  hated_races = ["human", "elf", "halfling"]
  favored_item_types = ["weapon"]
  
  [[boon]]
  cost = 10
  name = "battle_blessing"
  effect = "heal_full,mana_full"
  cooldown = "1 day"
  
  [[boon]]
  cost = 2000
  name = "divine_enchant"
  effect = "free_master_enchant"
  cooldown = "per_item"
```

**Exit criteria:**
- Pantheon TOML loads with 6–9 gods; loader batch-validates.
- `pray` / `tithe` / `offer` / `boon` / `atone` / `favor` / `pay_with_favor` commands implemented; favor + chosen god persisted in player save.
- Sacrifice grants per-god favor instead of (only) the current 1cp; flat 1cp fallback for atheists.
- Tithe and decay loop runs on real-time clock; ledger records `tithe_offering`, `favor_decay`, `boon_spend`, `temple_purchase`, `favor_purchase`.
- Each god has a populated temple shop with at least open / worshipper / cleric / favored stock tiers; cleric-only and favor-only items honor their gates.
- Opposing-alignment refusal verified by integration test: lawful character refused entry to chaotic-evil temple shop and vice versa.
- Phase 1 golden-master combat parity passes with the god system off (feature flag) and with a deterministic atheist character used as the canonical reference.
- Shadowport and Skullhold load as full hubs (bank, smiths, enchanters, sages, temple complex, MUD school exit destination, housing market) and pass the same service-set integration tests as Midgaard.

### E9 — Player housing (recurring rent sink + utility upgrades)

**Goal:** Long-running recurring coin sink tied to lifestyle, not combat. Hometown choice (already in E5) becomes meaningful: cheaper rent in your home hub, double rent elsewhere. Optional one-time upgrades convert housing into a personal services hub (forge, workbench, altar, safe) that converts upfront coin into ongoing rent multipliers.

#### Hub housing markets

Each first-pass hub (Midgaard, Shadowport, Skullhold) has its own innkeeper-run housing market with the same 4 tiers. Wilderness camps may rent rough lodging at room-tier prices for chaotic-evil characters who want zero city ties.

| Tier      | Rent / real-time week | Storage slots | Notes |
|-----------|----------------------:|--------------:|-------|
| Room      | 5g                    | 20            | newbie-affordable; inn quarter |
| Cottage   | 50g                   | 60            | mid-game; suburban district |
| Manor     | 500g                  | 200           | endgame; noble / outlaw quarter, social status |
| Fortress  | 2p                    | 500           | guild/clan tier; outside city walls; PvP-server defensible |

#### Hometown discount

- House in your registered `home_city` (same data field as the E5 bank hometown) → full listed rent.
- House in any other hub → **2× listed rent**, mirroring the E5 non-home withdrawal-fee structure. Same data field; no schema change.
- Atheist players still pick a hometown (purely civic, no god). Evil-aligned players can register Shadowport (or wilderness camp). Chaotic players can register Skullhold.
- `change_hometown <city>` at any banker: flat 1p fee + 14-day real-time cooldown. Prevents alt-cycling for arbitrage.

#### Mechanics

- `rent <tier>` at innkeeper → pay deposit (1 week rent) + first week → assigned a house instance with a private door keyed to the player.
- Rent auto-debits **weekly real-time** from bank balance first, then carried coin if bank insufficient.
- Missed rent → 7-day real-time grace; door locked, contents frozen.
- After grace + 30 days no payment → auto-downgrade to the next-lower tier if balance can sustain it (auto-evict gracefully). Otherwise eviction: storage contents auctioned by the innkeeper, proceeds returned to bank minus 20% innkeeper fee.
- `recall home` ports to your house entryway; cooldown shorter than temple recall, longer than mage `teleport`. Cross-hub legal (works from any zone if not no-recall).
- House interior is a small instanced area (entryway + storage room + optional upgrade rooms). Items inside the storage room have no carry-weight check; bank-grade no-decay.

#### Upgrades (one-time coin sink, ongoing service)

Each upgrade installs a fixture in the house. Coin paid once at construction; some upgrades raise weekly rent (maintenance).

| Upgrade        | Build cost | Rent mult | Effect |
|----------------|-----------:|----------:|--------|
| Smith forge    | 100g       | +10%      | in-house `repair` at 50% NPC price; no smith travel |
| Workbench      | 250g       | +15%      | craft T1/T2 in-house (no smith pilgrimage); T3 still requires master smith |
| Alchemy bench  | 250g       | +15%      | enchant from home at standard NPC odds; brick risk unchanged |
| Personal safe  | 500g       | +20%      | bank node — deposit/withdraw without traveling to a banker; subject to E5 hometown rules |
| Altar          | 1p         | +25%      | in-house `pray` for daily heal/mana boon; cleric-of-this-god gets discounted install (50% off) |
| Trophy hall    | 500g       | +10%      | display worn-out crafted gear, retired Uniques; pure cosmetic; social-flex |

Upgrades stack. A fully-loaded Manor with all six upgrades adds **+95% rent** on top of base 500g/week → 975g/week. Significant ongoing sink — only worthwhile for genuinely active endgame players.

#### Evil + chaotic housing

- Shadowport markets identical tier table; rent paid through Shadowport innkeeper. Good-aligned characters refused as renters above the suspicious threshold (mirror Midgaard rules).
- Skullhold markets identical tier table; chaotic-tolerant. Pickpocket risk in low-tier neighborhoods adds emergent flavor but does not affect rent system.
- Wilderness camps (Bloodmoor, Cursed Glade) offer Room-tier only (5g/week). Limited storage, no upgrades possible. Pure shelter for hardcore evil players who don't want any hub ties.

#### Integration with prior phases

- **E2 durability:** smith forge upgrade replaces smith travel for routine repairs. Upfront sink + per-repair savings.
- **E3 crafting:** workbench upgrade unlocks at-home T1/T2 crafting. T3 still requires master-smith pilgrimage (preserves the social hub).
- **E3.5 enchants:** alchemy bench replicates standard enchant NPC. Brick risk unchanged — no exploit.
- **E5 banks:** personal safe is a bank node, governed by the same E5 hometown rules (free withdrawal at home; surcharge if you set a different hometown after the safe is built).
- **E7 lottery:** house storage holds Lottery Set members during incomplete collections; trophy hall displays completed sets and retired Uniques.
- **E8 gods + temples:** altar upgrade is god-specific; once installed, it's locked to the player's current god. Switching gods (apostasy) requires rebuilding the altar. Cleric-of-this-god discount on install matches the temple-shop tier discount.

#### Commands

- `rent <tier>` at innkeeper: take a lease.
- `unrent` at innkeeper: voluntarily downgrade tier or move out (storage shipped to bank for safekeeping; 10% innkeeper fee).
- `upgrade <fixture>` at innkeeper: build an upgrade. Cost surfaced as quote; weekly rent recalculated.
- `recall home`: fast port to entryway.
- `safe` / `store <item>` / `take <item>` while in the storage room: like bank, no carry-weight.
- `house` (no arg): current tier, balance, upgrades, weeks paid, next debit time.

**Exit criteria:**
- Four tiers rent correctly at the documented prices; hometown discount honored; cross-hub 2× honored.
- Auto-debit, grace period, auto-downgrade, and eviction all fire correctly in sim and integration tests.
- Each upgrade installs and its effect verified end-to-end (e.g. workbench actually crafts T2 without smith proximity).
- Shadowport and Skullhold markets honor symmetrical alignment-refusal rules.
- Phase 1 golden-master combat parity passes with the housing feature flag off.

## Open Decisions

| Decision | Options | Lean |
|----------|---------|------|
| Durability decrement | hits-based / time-based | hits-based — fairer for AFK and rare-fight players |
| Repair cost scaling | linear vs `item.Cost²` | linear first pass; revisit after E2 sim |
| Crafted item binding | BoP / BoE / unbound | BoP — prevents alt-funnel |
| Material type for shop materials | new `ItemTypeMaterial` / repurpose junk | new — clearer separation |
| Identify cost mult | flat 10% / scale by affix count | flat 10% first |
| Home city for new chars | starting city / explicit choice | starting city — implicit |
| Boss material drop rate | deterministic per kill / RNG with pity | deterministic first (calibrate in E6) |
| Affinity bonus magnitude | small / medium / large | medium — meaningful but doesn't trivialize off-affinity play |
| T1 craft availability | every smith / select smiths only | every smith — tutorial reach |
| T3 craft availability | every smith / one master smith per class hub | one master per class — pilgrimage adds friction |
| Salvage refund rate (coin / mats / xp) | 50/80/100 / 30/60/100 / etc. | 50% coin / 80% mats / 100% craft-XP — encourages experimentation, still drains |
| Enchant brick rate (Master tier) | 5% / 10% / 15% | 10% — meaningful drama, not punishing |
| Enchant slot count per item | 1 / 2 / tier-based | 1 first pass; revisit after E3.5 telemetry |
| Reagent tradeability | tradeable / BoP | tradeable — drives player market |
| Damaged-drop durability floor | 5% / 10% / 25% | 10% — wearable briefly, immediate repair pressure |
| Rarity drop weights (Normal / Magic / Rare / Set / Unique) | 70/25/4/0.8/0.2 (D2-ish) / softer / harder | D2-ish first pass; tune in E6 |
| MagicFind soft cap | +200% / +300% / +500% | +300% — meaningful chase, prevents trivializing |
| Identify cost by rarity (Normal / Magic / Rare / Set / Unique) | 0/1/2/4/8 × base / linear | exponential 0/1/2/4/8 — punishes hoarding unidentified Uniques |
| Auto-loot default rarity threshold | none / magic / rare | none — players opt in |
| Pantheon size | 6 / 9 / 12 gods | 6 first pass; expand if engagement holds |
| God-pick gate | creation / L10 temple visit | L10 temple — players learn classes first |
| Apostasy cooldown | 3 / 7 / 14 days real-time | 7 days — meaningful commitment, not eternal |
| Atheist penalty (Hp/Mana regen) | 5% / 10% / 20% | 10% — choice cost, viable solo |
| Tithe target (% of weekly coin earned) | 1% / 3% / 5% | 1-3% per god, tuned in E6 |
| Favor decay rate (per real-time day under-tithed) | linear / accelerating | linear first pass; revisit if hoarding emerges |
| Cleric discount magnitude at temple shop | 25% / 50% / 75% off | 50% — strong cleric identity, doesn't trivialize |
| Worshipper discount magnitude | 10% / 25% / 50% off | 25% — meaningful, not min-max forcing |
| Atheist surcharge at temple shop | 0% / 25% / 50% | 25% — choice cost, not gate |
| Opposing-alignment temple access | refused entry / refused service / hostile guards | refused service first; hostile guards as later toggle |
| Favor-only items | a few iconic / many / none | a few iconic (resurrection-scroll, divine weapons) — forces real worship for identity items |
| Law axis (separate from Align) | add now / lump into Align / defer | defer — lump chaotic+evil onto current Align; revisit if nuance needed |
| Evil temple form | wilderness shrines / outlaw cities / both | wilderness shrines first; outlaw cities later content |
| City guard Align thresholds | -350/-700 / -500/-800 / -250/-600 | -350/-700 first pass; tune by playtest |
| Newbie grace Align floor (L<5) | -50 / -100 / -200 | -100 — generous, prevents grief, doesn't lock evil grad path |
| MUD school exit choice binding | binding god-pick / non-binding destination | non-binding — destination only, convert later via pray |
| Bribe per-real-day cap | 1 / 3 / unlimited (scales) | 1 per real-day — anti-spam, big coin sink per use |
| Disguise / polymorph entry duration | 5 / 15 / 60 in-game min | 15 in-game min — enough for a focused errand |
| Hub city set in first pass | 2 hubs (Midgaard + Shadowport) / 3 hubs (+ Skullhold) / 4+ | 3 hubs — covers good, evil, chaotic; viable solo loop per faction |
| Housing rent cadence | weekly real-time / monthly real-time / per-login | weekly real-time — predictable, mid-friction |
| Housing missed-rent action | full eviction / auto-downgrade / freeze indefinitely | grace + 30-day freeze + auto-downgrade — protects long-paused players |
| Cross-hub rent multiplier | 1.5× / 2× / 3× | 2× — matches E5 bank withdrawal-fee structure |
| Housing upgrade availability per tier | all tiers / Cottage+ / Manor+ | all tiers — newbie Room can install a forge if they pay |
| Trophy hall scope | crafted only / all worn-out / Uniques only | all worn-out — pure cosmetic, broad appeal |
| Change-hometown fee + cooldown | 0.5p / 1d / 1p / 14d / 2p / 30d | 1p / 14d real-time — prevents arbitrage, allows real relocation |

## Risks

- **Durability felt as tedium.** Mitigation: low L repair near-trivial; `repairMult` tunable per tier.
- **T3 crafting trivializes area content.** Mitigation: T3 recipes require boss-specific materials; bosses are area-content. Path to T3 IS area clears. Test: weekly raid logs show T3 owners still raiding for materials.
- **Off-affinity locked out of endgame.** Mitigation: off-affinity caps at T2 which is ≈ area drop power. Area drops fill the gap. Off-affinity is viable, not best.
- **Set bonus stacking with class spells unbalances PvE.** Mitigation: hard test fixture compares full-set on-affinity DPS vs scripted boss DPS; cap signature bonuses to ≤ 1 capability flag granted.
- **Affinity meta becomes mandatory; race choice becomes a stat decision.** Mitigation: race-signature slot grants stats even off-affinity, so race choice still matters; off-affinity still has full T2 access.
- **Custom craft + LLM-driven NPCs interact weirdly.** Mitigation: LLM smith dialog (Phase 14 Tier 1) describes the craft path, but the craft tool surface is the authoritative API; LLM cannot bypass tier or material gates.
- **Inflation overshoots after rebalance.** Mitigation: E6 ratio test gates rebalance; revert if drift >10%.
- **New player friction at L1–10.** Mitigation: durability slow at low item tiers; no ID fee below L20; no transfer fee below 1g; T1 crafting cheap and tutorial-paced.
- **(E3.5) Enchant brick rate frustrates players.** Mitigation: cap at 10% on Master tier; never on Simple/Greater. Surface odds explicitly in NPC dialog before confirmation. God boon (E8) provides a brick-free path for the truly committed.
- **(E3.5) Reagent grind becomes mandatory chore.** Mitigation: reagents are tradeable on player market — players can buy in via coin instead of farming.
- **(E7) Damaged-on-drop nerfs found gear too hard.** Mitigation: 10% durability floor; cheap repair at low L; one-shot kills preserve durability.
- **(E7) D2 vendor trash floods chat.** Mitigation: `autoloot_rarity` filter; `autoloot_dur_min` filter; corpse decay sweeps un-looted Normal items.
- **(E7) Set/Unique drops trivialize T3 crafted.** Mitigation: crafted T3 on-affinity full set + signature bonus + enchant unmatched by any single Rare/Unique drop. Best character mixes both paths.
- **(E7) RNG-loot chase invalidates planned crafting.** Mitigation: crafted gear is BoP and earned by deterministic progression; lottery loot is tradeable and chasable. Two playstyles, both valid endgame.
- **(E8) God-meta min-maxing — one god dominates.** Mitigation: boon power roughly equal across gods; differentiate by playstyle (war = burst, nature = sustain, trade = utility). A/B sim per god during E6.
- **(E8) Atheist players locked out of late game.** Mitigation: atheism legal with mild ~10% regen penalty; not a wall. Pure-atheist endgame still beatable with T3 crafted + Unique drops.
- **(E8) Alt god-hopping for boon-stacking.** Mitigation: apostasy resets favor + 7-day cooldown + starting penalty.
- **(E8) PvP imbalance from god-aligned gear refusing equip.** Mitigation: scoped to PvE; PvP servers configurable via `respect_god_alignment` flag.
- **(E8) Tithe and decay loop feels like real-life chore.** Mitigation: small tithe target (1-3% weekly); decay gentle (linear, ~5 favor / real-day); favor can be banked far above tithe target so casual play remains viable.
- **(E8) Favor + Phase 14 LLM NPCs interact unpredictably.** Mitigation: LLM temple priests describe boons + costs but the boon tool surface is authoritative; LLM cannot grant favor or boons outside the validated commands.
- **(E8) Temple-shop discount stacking with set bonuses + enchants trivializes cleric economy.** Mitigation: cleric discount applies only to listed coin price, not favor price; favor-only items unaffected by discounts. Sim check during E6.
- **(E8) Opposing-alignment refusal forces tedious travel for shopping.** Mitigation: most stock is duplicated across same-alignment temples; only god-iconic items are temple-exclusive. Atheists and same-alignment-different-god worshippers have surcharge but full access.
- **(E8) Cleric class becomes mandatory for endgame divine items.** Mitigation: a few iconic divine items are cleric-locked, but the bulk of god-themed stock is worshipper-accessible. Cleric advantage is discount + breadth, not exclusive endgame access.
- **(E8) Evil newbie traps post-MUD-school.** Mitigation: graduation escort routes evil-intending players to evil-aligned wilderness shrine roads. Newbie grace below L5 clamps Align so the destination choice isn't foreclosed by tutorial play. Non-binding choice — players can convert later.
- **(E8) Evil player permanently locked out of good cities.** Mitigation: three re-entry paths (atone quest, disguise/polymorph, bribe). All intentional friction, none permanent locks.
- **(E8) Bribe path turns alignment into pay-to-bypass.** Mitigation: bribe is per-real-day, scales with Align distance, time-limited entry only. Doesn't grant temple-quarter access or change Align. Pure logistics, not absolution.
- **(E8) Wilderness shrine raid griefing — high-level good players camp evil temple roads.** Mitigation: shrine grounds carry a `safezone` flag preventing PvP within a radius; approach roads not protected (free game). PvP server toggle.
- **(E9) Long pause locks players out / loses gear.** Mitigation: 7-day grace + 30-day freeze + auto-downgrade ladder. Gear is never destroyed; auctioned proceeds return to bank. Players returning after months are inconvenienced, not bankrupted.
- **(E9) Housing rent decouples from earned coin — chase players who don't earn at endgame.** Mitigation: rent auto-debits from bank first, carried coin second; tier curve is matched to E1 baseline + E2/E3/E5/E8 sinks so rent never exceeds 10-15% of typical weekly earn at any tier; auto-downgrade prevents catastrophic mismatch.
- **(E9) Housing upgrades trivialize the broader smith / enchanter / bank / temple economy.** Mitigation: upgrade costs are large (100g-1p one-time + rent multiplier); upgrades replicate **standard NPC** behavior, not better (enchant odds unchanged, brick risk unchanged, T3 craft still requires master smith pilgrimage); altar boons match temple-shop cleric tier, not god-of-this-altar exclusive.
- **(E9) Hometown lock-in feels punishing for players who relocate.** Mitigation: `change_hometown` legal at any banker; 1p fee + 14-day cooldown. Meaningful choice, not a wall.
- **(E9) Trophy hall encourages BoP-circumvention by displaying tradeable items.** Mitigation: trophy hall is display-only (read-only); displayed items cannot be retrieved without dismantling the display (10g fee). No new exchange surface.
- **(E9) Shadowport / Skullhold content cost balloons.** Mitigation: hub service set is templated. Same NPC interfaces as Midgaard (banker, smith, enchanter, sage, innkeeper, master smith, temple keeper, master enchanter) with reskins. Each hub is ~30% the area-content cost of Midgaard, not 100%, because the templates are shared.

## Dependencies

- Runs **independent** of the trait-system roadmap (Phases 1–12). No overlap with combat trait queries; durability, crafting, and set bonuses are item-data extensions.
- E3 set bonuses naturally piggyback on the trait query API (Phase 2) and the race/class TOML (Phase 8). Land trait system first if both run concurrently; otherwise E3 wires stat bonuses through a temporary hand-rolled apply step that the trait migration replaces.
- Race-class affinity data lives in the race TOML and is read by E3 craft validation; aligns with DATA-01 / DATA-02 schemas.
- Phase 14 LLM smith dialog references the crafting tool surface; LLM has no authority over tier/material gates.
- E3.5 enchant power values reuse the trait-system stat axes (TRAIT-01) and capability flags. Loading enchant TOML mirrors the homogeneous-section pattern from DATA-03.
- E7 rarity / affix pools reuse the same trait stat axes. Lottery Set and Unique TOML files load via the same batch-validated loader as crafted recipes; failures abort startup.
- E7 damaged-on-drop reads `Object.DurabilityMax` from E2; E2 lands first or E7 stubs durability with a defaulted constant.
- E7 MagicFind stat is a new `StatModifier` that flows through the trait query API (Phase 2).
- E8 pantheon TOML loader follows the same homogeneous-section pattern. Existing `ch.Align` is repurposed (no schema change required, but tests must verify save round-trip).
- E8 boons that grant temporary affects use the existing affect system (no new infrastructure).
- E8 atheism flag is implicit (no god picked); no save schema change beyond `ch.PCData.Favor map[string]int` and `ch.PCData.ChosenGod string`.
- E9 player housing reuses the E5 hometown field — no new schema for hometown discount. New schema: `ch.PCData.House struct { Hub, Tier, Upgrades, NextRentTick, ... }` plus a per-player instanced area registry.
- E9 upgrades replicate existing NPC behavior in-house; no new combat / magic mechanics introduced. Workbench reuses E3 craft tool surface; alchemy bench reuses E3.5 enchant tool surface; altar reuses E8 boon command set.
- Shadowport and Skullhold are area-content additions (~30% area cost of Midgaard each via shared NPC templates). New area files only; existing loader unchanged.

## Roadmap Integration

Phase 13 in `ROADMAP.md` references this document. The Phase 13 success criteria expand to cover E3 race+class crafting (criterion #3), E3.5 enchants (criterion #7), E7 lottery + damaged drops (criterion #8), E8 gods + favor + hubs (criterion #9), and E9 player housing + hub markets (criterion #10). E1 baseline + E2 durability + E4 identify + E5 bank fees + E6 rebalance remain numbered 1, 2, 4, 5, 6.
