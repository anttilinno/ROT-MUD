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

**Goal:** Re-tune mob drops down where E2–E5 sinks now compensate. End state: source/sink ratio ≈ 1.0 per bucket.

- Re-run E1 sim with E2–E5 sinks active; measure new equilibrium per level bucket.
- Adjust `mobCoinDrop` curve coefficients (per-level multiplier).
- Death penalty: 10% → 5% of carried coin (sinks cover loss-pressure).
- Boss-material drop rate: revisit deterministic-per-kill default. If T3 crafting is too easy, drop to 50% per kill and add a guaranteed pity-timer (5th kill always drops).
- Golden master from E1 updates to new baseline.
- **Exit criteria:** Sim ratio sink/source within ±10% of 1.0 per bucket; golden master locked.

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

## Risks

- **Durability felt as tedium.** Mitigation: low L repair near-trivial; `repairMult` tunable per tier.
- **T3 crafting trivializes area content.** Mitigation: T3 recipes require boss-specific materials; bosses are area-content. Path to T3 IS area clears. Test: weekly raid logs show T3 owners still raiding for materials.
- **Off-affinity locked out of endgame.** Mitigation: off-affinity caps at T2 which is ≈ area drop power. Area drops fill the gap. Off-affinity is viable, not best.
- **Set bonus stacking with class spells unbalances PvE.** Mitigation: hard test fixture compares full-set on-affinity DPS vs scripted boss DPS; cap signature bonuses to ≤ 1 capability flag granted.
- **Affinity meta becomes mandatory; race choice becomes a stat decision.** Mitigation: race-signature slot grants stats even off-affinity, so race choice still matters; off-affinity still has full T2 access.
- **Custom craft + LLM-driven NPCs interact weirdly.** Mitigation: LLM smith dialog (Phase 14 Tier 1) describes the craft path, but the craft tool surface is the authoritative API; LLM cannot bypass tier or material gates.
- **Inflation overshoots after rebalance.** Mitigation: E6 ratio test gates rebalance; revert if drift >10%.
- **New player friction at L1–10.** Mitigation: durability slow at low item tiers; no ID fee below L20; no transfer fee below 1g; T1 crafting cheap and tutorial-paced.

## Dependencies

- Runs **independent** of the trait-system roadmap (Phases 1–12). No overlap with combat trait queries; durability, crafting, and set bonuses are item-data extensions.
- E3 set bonuses naturally piggyback on the trait query API (Phase 2) and the race/class TOML (Phase 8). Land trait system first if both run concurrently; otherwise E3 wires stat bonuses through a temporary hand-rolled apply step that the trait migration replaces.
- Race-class affinity data lives in the race TOML and is read by E3 craft validation; aligns with DATA-01 / DATA-02 schemas.
- Phase 14 LLM smith dialog references the crafting tool surface; LLM has no authority over tier/material gates.

## Roadmap Integration

Phase 13 in `ROADMAP.md` already references this document. Success criteria ECON-01..06 update once E3 is rewritten (the criteria currently say "bound on pickup, capped at best comparable area drop"; criterion #3 needs to change to reflect the 3-tier + set-bonus + best-in-slot-at-T3 model).
