# Economic Overhaul Plan

## Status

- **Monetary system:** Done. `pkg/types/currency.go` defines 4-tier denominations (cp/sp/gp/pp) stored internally as int64 copper. Shops, bank, deposit/withdraw, mob drops, corpse loot, and death penalty all converted.
- **Open question (resolved here):** Should smiths craft custom equipment as a money sink? — **Yes**, but paired with durability/repair. Custom craft alone is a one-shot drain and risks power creep; repair gives a recurring, universally-felt drag. Together they cover both endgame luxury spend and mid-game maintenance.

## Goals

- Net coin OUT ≥ net coin IN at steady state per level bucket.
- Distinct sinks per progression band:
  - L1–30: trivial drains (repair, identify, bank fees)
  - L31–75: gear maintenance dominates
  - L76+: luxury custom craft + bespoke materials
- Reuse existing currency tiers — no new denominations.
- No pay-to-win: custom-crafted gear must never exceed best comparable area drop in raw power; it sells fit / appearance / convenience.

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
- Add a coin ledger: every credit/debit on `ch.Coin` and `ch.PCData.BankCoin` logs `(txn_type, amount_copper, source_id, target_id, tick)` where `txn_type ∈ {mob_drop, corpse_loot, shop_buy, shop_sell, death_loss, bank_deposit, bank_withdraw, transfer, quest_reward, …}`.
- Simulation test: walk N kills × M synthetic players per level bucket; dump source/sink ratio.
- **Exit criteria:** A reproducible report shows current inflation rate (coin in − coin out) per level bucket. This becomes the regression baseline for E2–E6.

### E2 — Durability + repair (universal drain)

**Goal:** Every active player loses coin per session proportional to play time.

- Add `Object.DurabilityCurrent` and `Object.DurabilityMax` to weapons and armor only (not money, not consumables).
- Durability decrement model: **hits-based**, not time-based.
  - Worn armor: −1 per hit taken (only on hit, not on miss).
  - Wielded weapon: −1 per swing landed.
- At 0 durability: item is "broken" — stats halved, wear-fail noise on combat round, but item not destroyed.
- Add `repair <item>` command at smith NPCs (new `ActSmith` flag on banker-like mobs).
  - Cost formula: `((max - cur) / max) * item.Cost * repairMult` where `repairMult` is a tunable per item tier (start 0.25).
  - Atomic: deduct coin, restore durability to max, or error.
- TOML: `durability_max` field on object templates; default by item tier (weapon=200, armor=150) when omitted.
- **Exit criteria:** Ledger from E1 shows non-zero `repair` debits in every active session; sim of N hours of combat shows steady drain matched against drops.

### E3 — Smith custom crafting (luxury sink)

**Goal:** Rich endgame players have a meaningful coin sink without unbalancing power.

- New NPC type: master smith. Flag on existing smith mobs that completed E2.
- Recipe data: TOML in `data/recipes/*.toml` with:
  - `base` (object vnum or item type slot — e.g. "longsword")
  - `[[affix]]` entries with `name`, `cost_mult`, `level_req`, `stat_delta` (HitRoll, DamRoll, Hp, etc.), `material_required` (vnum)
- Player command: `craft <base> <affix>...` → quote price → confirm → spawn bespoke object instance.
- Pricing: `base.Cost * Π(affix.cost_mult)` plus consumed materials.
- **Power cap (hard constraint):** sum of stat deltas on a crafted item ≤ best in-game area drop of the same slot at the player's level. Enforced by a test fixture iterating all recipes vs the area drop manifest.
- Materials: new item type `ItemTypeMaterial`. Drops from L20+ mobs (replacing some low-value junk loot). Consumed by craft.
- **BoP (bind on pickup):** crafted items flagged `ItemFlagBound` and are not sellable / not transferable to defeat alt-funneling.
- **Exit criteria:** Crafted item economy ledger shows material→coin sink path; power-cap fixture passes against full area drop set.

### E4 — Identification + appraisal (info sink)

**Goal:** Compulsory mid-game drain on every meaningful drop.

- Items with vnum-level ≥ 20 drop with hidden affixes.
- Display shows base name + `(unidentified)`; stats hidden until identified.
- New sage NPC type (or reuse mage guild mobs). `identify <item>` command, flat fee = `item.Cost * identifyMult` (start 0.10).
- Items vnum-level < 20 unaffected.
- Edge cases:
  - Money objects (`ItemTypeMoney`) never need ID.
  - Quest reward items can flag `auto_identify = true` to skip the fee.
- **Exit criteria:** Every L20+ kill creates pressure to spend a fraction of the drop value on ID.

### E5 — Bank fees + transfer

**Goal:** Small constant trickle on coin movement; cannot dodge sinks by hoarding in bank.

- Each bank NPC has a `home_city` field.
- `deposit` at any banker: free.
- `withdraw` at non-home banker: `withdrawFeePct` (start 2%) charged off the top.
- `transfer <player> <amount>` command at banker: small flat fee (start 1g) plus 1% of amount.
- Withdrawals at home city free (rewards travel cost).
- **Exit criteria:** Ledger shows `bank_fee` debits trickling steadily; player coin movement no longer free.

### E6 — Faucet rebalance

**Goal:** Re-tune mob drops down where E2–E5 sinks now compensate. End state: source/sink ratio ≈ 1.0 per bucket.

- Re-run E1 sim with E2–E5 sinks active; measure new equilibrium.
- Adjust `mobCoinDrop` curve coefficients (per-level multiplier) to land near 1.0.
- Death penalty: reduce from 10% → 5% of carried coin (sinks already cover loss-pressure).
- Golden master from E1 updates to new baseline.
- **Exit criteria:** Sim ratio sink/source within ±10% of 1.0 per bucket; golden master locked to new numbers.

## Open Decisions

| Decision | Options | Lean |
|----------|---------|------|
| Durability decrement model | hits-based / time-based | hits-based — fairer for AFK and rare-fight players |
| Repair cost scaling | linear vs `item.Cost²` | linear first pass; revisit after E2 sim |
| Crafted item binding | BoP / BoE / unbound | BoP — prevents alt-funnel |
| Material drop source | new junk replacement / new item type | new `ItemTypeMaterial` — clearer separation |
| Identify cost mult | flat 10% / scale by affix count | flat 10% first; revisit if too easy |
| Home city for new chars | starting city / explicit choice | starting city — implicit, no extra step |

## Risks

- **Durability felt as tedium.** Mitigation: low L repair near-trivial; repairMult tunable per tier.
- **Custom craft power creep.** Mitigation: hard test fixture caps craft stats vs area drops.
- **Bank hoarding dodges sinks.** Mitigation: withdrawal fee on non-home banks, transfer fee.
- **Inflation overshoots after rebalance.** Mitigation: E6 ratio test gates the rebalance; revert if ratio drifts >10%.
- **New player friction at L1–10.** Mitigation: durability ticks are slow at low item tiers; no ID fee below L20; no transfer fee below 1g.

## Dependencies

- This work runs **independent** of the trait-system roadmap (Phases 1–12). No overlap with combat trait queries; durability and crafting are item-data extensions, not combat-rule changes.
- Can begin anytime after the currency commit lands.
- Loader work in E3 (recipe TOML) follows the same homogeneous-section pattern as the trait system, but does not block on it.

## Roadmap Integration

Either:
- **Standalone milestone** after the trait system completes (recommended — keeps phase boundaries clean), or
- **Phase 13+** appended to `ROADMAP.md` if the trait system completes faster than expected and resources free up.

Decision deferred until E1 baseline measurement clarifies urgency.
