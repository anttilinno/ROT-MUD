# Scout Notes — Ideas from Classic MUDs

## Purpose

Idea inventory scouted from SMAUG (Realms of Despair), MUME (Multi-Users in Middle-earth), and GodWars II. **None of these are committed scope.** Each entry tags where it might fit existing phases (1–14 and ECONOMY sub-phases E1–E9) so we can promote items into actual plans later without re-doing the research.

Use as a menu when the next phase needs picking, or when an existing phase needs more design depth.

Status column:
- `pick?` — candidate, no decision
- `lean yes` — strong fit, low cost
- `lean defer` — interesting, big cost / out of current scope
- `lean drop` — researched, decided against

---

## From SMAUG / Realms of Despair

### Hunting mobs (mob memory + tracking)
- **Mechanic:** Killed-by-you mob's siblings remember your name, hunt you across zones, can `summon` you to their lair. Persistent grudge AI.
- **Fits:** Phase 14 Tier 2 scripted fallback. LLM post-mortem (already in plan) is the LLM version of this; SMAUG-style scripted tracking is the always-on fallback.
- **Cost:** Small. Per-mob-template "grudge list" map; ticker checks for line-of-sight movement.
- **Risk:** Griefy if applied broadly. Scope to boss/elite mobs only.
- **Status:** lean yes — gives Phase 14 a real fallback behavior, not just inert scripted attack.

### Orders + Councils as 3rd-tier orgs above clans
- **Mechanic:** Three player-org tiers. Clans = PvP/social guilds. Orders = religious/aligned brotherhoods (paladin order, vampire coven). Councils = cross-faction admin (mage council ratifying spells, merchant council setting tariffs).
- **Fits:** E8 — Orders are the clergy chapter for each god. Councils could drive world-event sinks.
- **Cost:** Medium. New schema in PlayerSave (`Order`, `Council`). New command surface.
- **Risk:** Empty-org problem on low-pop servers.
- **Status:** pick? — depends on player count; defer until population proves out.

### Languages
- **Mechanic:** Race + class tongues. Speakers understand; non-speakers see garbled text (`Bn'kkur grok!`). Players learn languages with skill investment.
- **Fits:** Trait system Phase 2/3 capability flag (`Speaks<lang>`). Cheap identity differentiator.
- **Cost:** Low. Garble function on `say` / `tell` if listener lacks the trait flag.
- **Risk:** Annoying to new players if overdone — scope to RP servers or per-zone toggle.
- **Status:** lean yes — fits trait system natively.

### Interactive room objects (levers, switches, pullchains, traps)
- **Mechanic:** Rooms contain non-monster interactive props. Pull lever → opens door elsewhere. Step on trap → take damage / teleport. Switches gate room access.
- **Fits:** Area content layer; cheap puzzle hooks in T3 dungeons (E3 boss-material areas).
- **Cost:** Medium. Loader extension for `[[rooms.interactives]]` section; command parser for `pull`, `push`.
- **Risk:** Bug-prone if state lives in room (lever stuck open). Mitigate with explicit reset semantics on zone repop.
- **Status:** pick? — high content value, moderate plumbing.

### Delayed teleport rooms / tunnels
- **Mechanic:** Stepping into a room teleports you after N rounds (warning shown). Tunnels = long teleport chains for travel.
- **Fits:** MUD school escort (E8) — replaces instant transfer with cinematic delay. Temple `recall home` (E9) flavor.
- **Cost:** Small. Room flag + tick handler.
- **Risk:** Players AFK during delay get displaced. Standard MUD pattern, accepted.
- **Status:** lean yes — direct fit for escort + recall.

### Planes / dimensions (parallel area systems)
- **Mechanic:** Multiple parallel worlds (astral, hell, ethereal). Spells/items move characters between planes. Planes have own zones, mobs, rules.
- **Fits:** Endgame raid tier — god-domain T3 raid zones live in domain planes (hell plane for death god, astral for magic god).
- **Cost:** High. New world coordinate axis; cross-plane visibility rules.
- **Risk:** Doubles area content burden if every plane needs full coverage.
- **Status:** lean defer — bookmark for post-Phase-13.

### Multiclass + class restacking
- **Mechanic:** Char takes class A to cap, then "remorts" to class B; some skills carry, some lose. Stat caps stack additively per remort.
- **Fits:** Already partially handled by Phase 8 remort-class trait stacking. SMAUG confirms additive stacking pattern.
- **Cost:** Already planned.
- **Status:** lean yes — already in roadmap, no new work.

### Currency tiers (gold / silver / copper)
- **Status:** Done — `pkg/types/currency.go` already implements this.

---

## From MUME

### Zoning + zone-wide repop + weather
- **Mechanic:** World partitioned into ~100-room zones. Repop fires per-zone, not per-room. Storms / time-of-day events also fire per-zone. Player can feel "the zone is alive."
- **Fits:** Existing world model; engine-level optimization. Weather adds atmosphere for free.
- **Cost:** Medium. Zone manager already exists implicitly; explicit zone-tick scheduler.
- **Risk:** Repop visibility — players in zone see mass spawn. Could be feature (lore moment) or bug (overwhelming).
- **Status:** lean yes — engine win + content feel.

### Realistic equipment naming
- **Mechanic:** Equipment named like real medieval gear (`a hauberk of mail`, `a Westernesse-forged longsword`). Almost no glowing/humming/faintly-throbbing spam. Magic items are rare and feel special.
- **Fits:** E7 affix naming rule. Constrains visual descriptor spam. Lottery Rares stand out because Normals are mundane.
- **Cost:** Naming convention only — codify in `data/affixes/*.toml` style guide.
- **Risk:** None.
- **Status:** lean yes — codify as design rule.

### Tradeoff equipment (no strictly-best items)
- **Mechanic:** Every piece has pros AND cons. Heavier plate = better defense but more encumbrance and movement cost. Lighter chain = worse defense but faster regen. Weapon damage vs weapon speed vs weapon parry rating.
- **Fits:** E3 craft design constraint, E7 affix pool constraint. Every craftable + every lottery drop must carry a downside dimension.
- **Cost:** Design discipline. Test fixture: assert no recipe + no affix combo is strictly dominant in all stat axes.
- **Risk:** Players hate it if downsides feel arbitrary. Pair stat with lore (heavy = slow because heavy, not because RNG).
- **Status:** lean yes — adopt as E3/E7 design rule.

### Faction war hard-locked at character creation
- **Mechanic:** Pick Good (Free Peoples) or Evil (Sauron's armies) at creation. Hard PvP enforcement. Cannot switch sides ever.
- **Fits:** E8 hub system — alternative to non-binding MUD school destination choice. Harder edge.
- **Cost:** Schema-level char-create change.
- **Risk:** Forecloses player choice; punishes new players who pick wrong before understanding.
- **Status:** lean defer — E8 non-binding choice is gentler first pass. Bookmark for hardcore-server flag.

### Legend system (post-cap progression)
- **Mechanic:** After reaching legend status (post-cap), characters gain +1 practice + +1 HP per level. Slow but eternal endgame progression.
- **Fits:** Endgame after L100. Matches T3 endgame timeline.
- **Cost:** Small. Level cap extension + slower XP curve at legend tier.
- **Risk:** Power creep if uncapped. Soft cap legend progression.
- **Status:** pick? — strong fit but needs a phase to belong to. Add as future Phase 15?

### Permadeath / racial death triggers
- **Mechanic:** Troll caught in sunlight = character deletion. Race-specific permadeath conditions. Vulnerabilities have real consequences, not just stat penalties.
- **Fits:** Trait system Phase 2 vulnerabilities — make them LETHAL, not soft. Identity stake.
- **Cost:** Low. Vulnerability trait can declare `lethal = true` + condition.
- **Risk:** Players quit if they lose chars to surprise sun. Mitigate with warnings (`The sun rises... your skin begins to smoke`) and grace periods.
- **Status:** lean yes — opt-in per race; bold racial identity.

### Hunger / thirst / food
- **Mechanic:** Players must eat and drink. Hunger debuffs at low levels; starvation damages at zero. Food and drink are an actual economy.
- **Fits:** New recurring sink. Tavern/inn economy. Pairs with E9 housing kitchen upgrade.
- **Cost:** Medium. Hunger/thirst ticks; food consumable items; tavern shop population.
- **Risk:** Tedium if intervals too short. Set hunger interval generous (real-time hours, not minutes).
- **Status:** pick? — strong sink candidate; could be E10.

### Equipment durability with breakage consequences
- **Mechanic:** Equipment wears down; broken items become unusable or actively dangerous.
- **Fits:** Already in E2.
- **Status:** Done — covered by E2.

### Identification systems
- **Mechanic:** Items don't reveal stats until identified.
- **Status:** Already in E4. MUME confirms pattern.

### Light/dark mechanics
- **Mechanic:** Rooms can be dark; torches/spells needed. Dark areas hide mob attacks; affect combat.
- **Status:** Already in ROT-MUD core. Worth promoting to trait-system room flags (Phase 11).

---

## From GodWars II

### Callings (meta-classes above classes)
- **Mechanic:** 6 classes grouped into 4 Callings (Revenant = Vampire; Lycanthrope = Werewolf; Illuminati = Mage; Nephilim = Demon + Dragon + Titan). Callings carry meta-traits all member classes share.
- **Fits:** E3 race-class affinity hierarchy. Callings as meta-group, classes as leaves, set bonuses align by Calling not class. Cleaner hierarchy than flat class list.
- **Cost:** Low. Callings file in TOML; classes reference parent Calling.
- **Risk:** Calling design needs cohesion — easy to write incoherent meta-groups.
- **Status:** lean yes — adopt for E3 set-bonus structure.

### Stat training boosts random class-secondary
- **Mechanic:** Each `train <stat>` always boosts the trained stat, AND has a chance to boost a class-specific secondary stat. Mage train of int has chance to also boost mana regen; warrior train of str has chance to also boost HP.
- **Fits:** Existing train system; small extension.
- **Cost:** Small. Per-class secondary table.
- **Risk:** Stacking on a min-maxer over many trains. Cap secondary at half of trained level.
- **Status:** lean yes — cheap class identity at every progression beat.

### Subclasses at L50 with stat-buy picks
- **Mechanic:** At a level threshold, pick a subclass. Subclass costs 5 build points distributed across themed sub-abilities. Mage → Flameborn Adept: +2 Fire Magic, +2 Form of Fire, +1 Fire Elemental.
- **Fits:** Mid-game choice between class pick and T3 endgame. Build diversity inside class.
- **Cost:** Medium. Subclass data, build-point UI/command, downstream stat plumbing.
- **Risk:** Balance — easy to write strictly-best subclass.
- **Status:** pick? — strong design candidate; could be E11 or a separate phase.

### Form changes (vampire bat, werewolf wolf, demon true, mage elemental)
- **Mechanic:** Race/class command `morph <form>` swaps stats, slot layout, available skills, and visual. Form has its own duration / drain / cooldown.
- **Fits:** New mechanic. Major identity lever for race classes. Could pair with GW demon body-part craft.
- **Cost:** High. Form data type; skill / stat swap engine; equipped-gear handling during form (slots disappear?).
- **Risk:** Form-during-combat complexity (equipped items, ongoing affects).
- **Status:** lean defer — strong feature but big plumbing. Bookmark for post-Phase-13.

### Cross-class material crafting (GW2 moonsilver)
- **Mechanic:** Werewolves craft moonsilver — material no other class can produce, but other classes can use in their own crafting. Class-to-class trade dependency.
- **Fits:** E3 — adds craft trade economy between classes. Materials feed each other's recipes.
- **Cost:** Small. Recipe declarations + class-locked craft action.
- **Risk:** Forces multi-class server population to maintain economy. On a small server, missing classes break recipes.
- **Status:** pick? — strong economy hook; gated by player count.

### Summons / familiars
- **Mechanic:** Class command `summon <creature>` brings a pet into combat. Demons summon imps/hellhounds, mages summon elementals. Pet has stats, scaling, commands (`order <pet> attack`).
- **Fits:** New mechanic. Combat depth. Fits demon/mage/druid flavor.
- **Cost:** High. Pet entity, AI, command surface, leash/loyalty system.
- **Risk:** Pet-AI complexity; aggro management; PvE balance.
- **Status:** lean defer — bookmark; big mechanic, deserves own phase.

### Demon body-part craft (already documented elsewhere)
- **Status:** Already absorbed into E3 race-signature slot design. GodWars demon armor inspired the 12-slot + 1-signature model.

---

## Cross-cutting patterns

These show up in multiple sources and deserve direct adoption:

- **Mob memory** (SMAUG hunting + MUME repop persistence) — boss/elite mobs remember player and act on it.
- **Tradeoff design** (MUME equipment + GW2 subclass build points) — no strictly-best option; every choice has a downside.
- **Hierarchical groups** (SMAUG clan/order/council + GW2 callings/classes) — 2–3 nested tiers of identity above individual choices.
- **Form / state changes** (GW2 morph + MUME troll sun) — entity has multiple states with different mechanics.
- **Persistent world consequences** (MUME perma-death + SMAUG hunting grudge) — player actions stick.

---

## Suggested next steps

1. Read this doc as a menu. Mark each entry's status with concrete decisions.
2. Promote `lean yes` items into existing phases (E3 callings, E3 tradeoff rule, E7 naming rule, E8 orders, etc.) at their next plan-phase pass.
3. Spin a `Phase 15` shell for legend system + form changes + summons if those graduate from `lean defer`.
4. Re-scout periodically as new MUDs / patterns come up (e.g. Achaea, Aardwolf, Discworld, Lusternia — none scouted yet).

---

## Sources

- [SMAUG | MUD Wiki | Fandom](https://mud.fandom.com/wiki/SMAUG)
- [SMAUG | Muds Wiki | Fandom](https://muds.fandom.com/wiki/SMAUG)
- [Realms of Despair — SMAUG features](https://realmsofdespair.com/index.php/smaug/)
- [SMAUG GitHub mirror](https://github.com/smaugmuds/_smaug_)
- [MUME — Multi-Users in Middle-earth](https://mume.org/play/)
- [MUME Help Index](https://mume.org/help/)
- [MUME Help: Skills](https://mume.org/help/skills)
- [MUME Help: Subjects](https://mume.org/help/subjects)
- [MUME Help: Legend Home Instructions](https://mume.org/help/legend_home_instructions)
- [MUME | MUD Wiki | Fandom](https://mud.fandom.com/wiki/MUME)
- [MUME on MUDverse](https://www.mudverse.com/game/484)
- [General MUME Guide](https://mumeresource.wordpress.com/guides/general-mume-guide/)
- [God Wars 2 Classes (godwars2.github.io)](https://godwars2.github.io/classes/)
- [God Wars II official site](http://www.godwars2.org/)
- [God Wars II | WikiMU* | Fandom](https://mu.fandom.com/wiki/God_Wars_II)
- [GodWars | Muds Wiki | Fandom](https://muds.fandom.com/wiki/GodWars)
- [GodWars: Apocalypse — Demon class](https://www.godwars.net/~apoc/demon.html)
