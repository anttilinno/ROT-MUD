# Domain Pitfalls: Data-Driven Traits + Lua Scripting for ROT-MUD

**Domain:** Go MUD server — TOML-defined race/class traits + embedded Lua behavior hooks
**Researched:** 2026-04-16
**Scope:** Pitfalls specific to (a) embedding Lua in a Go game server, (b) migrating hardcoded
constants to data-driven loading, (c) trait-system rot over time, (d) additive stacking
exploits, (e) behavior-parity verification.

---

## Critical Pitfalls

These cause rewrites, data loss, server crashes, or unrecoverable save corruption.

### Pitfall 1: Sharing a single `lua.LState` across goroutines

**What goes wrong:** `LState` is explicitly *not* goroutine-safe (yuin/gopher-lua). Calling
Lua from the violence update goroutine while a player command handler also triggers a hook
produces map corruption panics, stack misalignment, or silent state corruption that only
manifests hours later.

**Why it happens:** Go makes concurrency easy, so a naive `globalLState := lua.NewState()`
in `init()` feels natural. Nothing in the Lua API signals the thread-affinity constraint.
In ROT-MUD the game loop runs on a single goroutine today, but `pkg/server/server.go`
connection handlers, the violence pulse, and MOBprog triggers are separate call sites —
any cross-goroutine dispatch into Lua will race.

**Consequences:** Nondeterministic crashes under load; players report random server
disconnects during combat; debugging is hellish because stack traces point into Lua VM
internals rather than Go code.

**Prevention:**
- Use an `LStatePool` (mutex-guarded `sync.Pool` of prewarmed states). Acquire on hook
  entry, release on exit. Documented pattern in gopher-lua README.
- Or: run *all* Lua calls on the single game-loop goroutine. Marshal hook-trigger requests
  through a channel so the VM is touched from exactly one goroutine.
- Forbid Lua access from anywhere except the game loop until this is enforced by the type
  system (e.g., a `LuaCtx` handle only constructable inside the game-loop closure).

**Warning signs:** `fatal error: concurrent map read and map write` pointing into
`gopher-lua/state.go`; intermittent panics in Lua hooks during peak load; "works in dev
with one player, crashes in prod."

**Detection:** `go test -race` with a stress test that fires hooks from multiple goroutines
concurrently.

**Phase:** Addressed in the Lua VM integration phase (corresponds to LUA-01). Must be
designed in from day one — retrofitting pool semantics after the fact means rewriting
every hook call site.

---

### Pitfall 2: Save-file incompatibility when race/class indices change

**What goes wrong:** Player saves serialize `ch.Race` and `ch.Class` as integer indices
(see `RaceHuman = 0`, `RaceElf = 1`, ... in `types/races.go`). Moving race definitions
to TOML means load order now determines indices — if `races/elf.toml` is removed or
alphabetical ordering shifts, every existing player's race silently points to a different
race on load. Dwarves become giants.

**Why it happens:** The existing system relies on compile-time ordinal stability. TOML
file discovery depends on filesystem ordering, author conventions, and git operations.
The MUD already uses `ch.Race == RaceVampire`-style comparisons which presuppose stable
indices, and `combat_sim_test.go:1080-1081` hardcodes `types.RaceGiant, types.RaceDwarf...`
as ints — these break silently.

**Consequences:** Catastrophic — every existing player character corrupted on first load
after migration. Unlike a crash, this *succeeds* and the player logs in with wrong stats,
wrong HP gain curve, wrong XP multipliers. By the time anyone notices, saves have been
overwritten with the corrupted state.

**Prevention:**
- **Serialize by name, not by index.** Player save format must write `"race": "elf"` and
  `"class": "mage"`, converting to/from index at load/save boundary only. This is the
  single most important decision in the whole project.
- Add a save-format migration step: on first load of a pre-migration save, look up
  historical index→name mapping and rewrite. Keep the compatibility table forever; never
  reuse a deleted race's old index.
- Snapshot-test the name→index mapping so a code reviewer must explicitly approve any
  shift.

**Warning signs:** Any code that does `binary.Write(w, ch.Race)` or JSON-serializes the
raw int. A loader that walks `filepath.Glob` to discover race files without a deterministic
sort.

**Detection:** Before the migration phase, write a test that loads a snapshot of every
existing character from a known save, migrates, saves, reloads, and asserts every single
field is byte-identical (modulo the race/class representation change). This is
**golden-master / characterization testing** — see Pitfall 14.

**Phase:** Must be addressed *before* MIGRATE-01/02 ship. Ideally the save-format change
lands as its own phase before any TOML loading goes live, so the name-keyed format is
already in production saves.

---

### Pitfall 3: Lua runaway loops halting the game loop

**What goes wrong:** A buggy or malicious hook script contains `while true do end` or an
accidental unbounded recursion. The game loop goroutine blocks indefinitely. Every player
freezes. No watchdog fires because from the OS's perspective the process is healthy and
busy.

**Why it happens:** Lua scripts are edited by admins/builders, not programmers. Typos
happen. A hook that recurses into itself (OnAttack triggers a spell that triggers OnAttack)
is easy to write and hard to see in review. gopher-lua has *no* default instruction limit.

**Consequences:** Entire MUD wedged. Kills player trust if it happens repeatedly. Requires
SIGKILL to recover (no graceful shutdown), losing any unsaved state.

**Prevention:**
- Install a `debug.sethook` instruction counter per-hook invocation. Budget: e.g., 100k
  instructions per hook. Exceed → `L.RaiseError`, hook aborts, warning logged, player-facing
  behavior falls back to "no hook fired."
- Use `LState.SetContext(ctx)` with a `context.WithTimeout` — gopher-lua honors context
  cancellation on calls to `L.PCallByContext` / `L.CallByParam`.
- Forbid recursive hook dispatch: hooks run with a per-goroutine "in hook" flag, and the
  dispatcher refuses to nest (or caps depth at e.g. 3).
- Route all hook calls through `pcall` equivalent (`L.PCall`) so panics in Lua bubble up as
  Go errors instead of propagating.

**Warning signs:** Any hook call site that uses `L.Call` instead of `L.PCall`. Any hook
design that allows "OnSpellCast fires → spell casts → OnSpellCast fires" without a
depth guard.

**Detection:** Fuzz-test the hook API with adversarial scripts (`while true`, deep
recursion, huge table allocations) and assert the server survives.

**Phase:** LUA-01. Instruction limiter and pcall wrapping ship with the VM, not later.

---

### Pitfall 4: Lua memory growth from uncleaned state / coroutine context leaks

**What goes wrong:** `LState` holds a registry, call stack, and globals table that grow as
scripts run. gopher-lua issue #56 and #437 document real production leaks — especially
around coroutines and context values not released.

**Why it happens:** Developers assume Go GC handles everything. But `LState` references
are rooted from Go-side pool objects; Lua-side allocations (tables attached to globals,
long-lived closures) stay alive as long as the LState does. Pooling without periodic
rotation keeps leaks compounding across hook invocations.

**Consequences:** Server RSS grows monotonically over days/weeks. Eventually OOM-killed.
Operators "solve" it by scheduled restarts, which masks the root cause and annoys players.

**Prevention:**
- Never store player/character data in Lua globals. All per-hook state is either (a) passed
  in as function arguments, or (b) stored in a sandbox environment discarded after the
  hook returns.
- Call `L.Close()` on pool retirement. Rotate pool members periodically (kill LState after
  N invocations, spin up replacement).
- Disable the coroutine library if hooks don't need it (`-LibCoroutine` in `lua.Options`).
- Budget an LState memory cap: track allocations via `L.SetMx(maxBytes)` if using
  mx-patched gopher-lua, otherwise sample `runtime.ReadMemStats` before/after hooks.

**Warning signs:** `RSS` graph with a steady linear slope. Pool that only grows and never
shrinks. Hooks that `lua.LState.SetGlobal` as a way to "pass data."

**Phase:** LUA-01 (pool design); ongoing monitoring in any phase that adds hook
surface area.

---

### Pitfall 5: Lua sandbox escape via Go callback injection

**What goes wrong:** Hooks get access to a `character` object. If that object exposes a
method like `char:setAdmin(true)` or `char:execGo("rm -rf /")` — or more subtly, exposes
a reference that *can be walked* to one — an admin who can write a race file has RCE on
the server.

**Why it happens:** It's tempting to expose the whole `*types.Character` via
`luar`-style reflection (gopher-luar bridges Go structs into Lua). The moment you do,
every exported field and method is a potential attack surface, including ones added in
future commits without Lua in mind.

**Consequences:** Admin-grade compromise. Even if admins are trusted, a compromised admin
account = compromised server. In 2024, Redis had a widely-publicized Lua sandbox escape
with similar mechanics.

**Prevention:**
- **Do not use reflection bridges** like `gopher-luar` on Go domain objects. Instead,
  hand-write a minimal Lua API surface: `char_get_hp(char)`, `char_damage(char, n, type)`,
  `char_get_race_name(char)`. Each function is a `lua.LGFunction` that validates arguments
  and returns only primitive types or opaque userdata handles.
- Strip dangerous standard libraries. At a minimum disable: `io`, `os`, `debug`, `package`,
  `require`, `loadfile`, `dofile`. Use `lua.Options{SkipOpenLibs: true}` and open only
  `base`, `math`, `string`, `table`.
- Run hooks in a child environment (`setfenv` equivalent via `L.NewTable` +
  `L.SetFEnv`) so globals set by the script don't persist.
- Review the exposed API every time a new hook event is added. Treat the Lua API as a
  public contract with security implications.

**Warning signs:** Any `import "gopher-luar"` or `luar.New(L, characterObject)`.
`L.OpenLibs()` called without argument filtering. A hook that takes a handle and can
follow pointer chains into the world state.

**Phase:** LUA-01 (API design). Security posture must be baked into the very first hook
binding — retrofits leak.

---

### Pitfall 6: Behavior divergence during MIGRATE-01/02

**What goes wrong:** Migrating 19 races + 14 classes from hardcoded Go tables to TOML
results in subtle numeric differences. The `fightRaces` test passes at L100 (per memory
notes from `project_combat_sim_tuning.md`) but a Dwarf mage at L50 mana pool differs by
+3. Nobody notices for weeks. Then a player complains their saved character is "weaker
than before."

**Why it happens:** The existing tables have *commented* design intent
(`races.go:47-287`) but the actual game behavior is whatever the code computes. Some
modifiers are applied at character creation, some at load, some per-action. A TOML
migration captures the *tables* but may miss the *application sites* — any edge case
in `pkg/combat/` that reads a field differently than the TOML loader populates it.

**Consequences:** Player trust damage. Combat balance drift that invalidates the tuning
work captured in `combat_sim_test.go`. Possible save corruption if old saves encode
computed-not-raw values (e.g., starting stats applied at creation time).

**Prevention:**
- **Dual-run mode.** During migration, load both tables (hardcoded + TOML) and assert
  field-by-field equality at startup. Fail hard on mismatch. Remove hardcoded version
  only after N days of green dual-run.
- **Combat sim as regression gate.** `combat_sim_test.go` currently runs L100 combat
  parity checks. Extend it to run pre-migration vs post-migration and assert win% deltas
  are within 0.5 percentage points. No exceptions.
- **Golden-master per race/class.** For each race × class combo, snapshot the character
  struct produced at creation (all stats, all applied modifiers, resulting THAC0/HP/mana
  curves at levels 1, 10, 50, 100). Assert TOML version produces identical snapshots.

**Warning signs:** A migration PR that touches TOML but does not add characterization
tests. A claim that "the tables are simple, just straight transcription" — they never are.

**Detection:** Diff the output of a character-creation fixture before/after migration,
byte-identical or the migration fails.

**Phase:** Must be the acceptance gate for MIGRATE-01 and MIGRATE-02. Write the
golden-master test *first*, then migrate.

---

## Moderate Pitfalls

Cause real pain but recoverable without data loss or rewrites.

### Pitfall 7: Additive stacking creating degenerate combinations

**What goes wrong:** Project decision is "additive trait stacking — race + class both
apply." A future Titan (size huge, +Str, high HP) × Gladiator (T2 Str-primary, HPMax=25)
combination stacks all damage bonuses additively. The result is a character whose output
is 2× any balanced baseline, breaking PvP and trivializing mobs.

**Why it happens:** Additive modifiers are individually fine but compose poorly when both
sources push the same axis. In Stellaris, Path of Exile, and League of Legends this is
a well-known failure mode (see sources). The current ROT-MUD tuning work already caps
mob HP slope and acid blast damage — those caps exist because raw additive math blew up.
Every new trait axis is a new opportunity for this.

**Consequences:** Balance rewrites. Players who specced for the broken combo feel nerfed
when fixed; players who didn't feel cheated. The whole trait system gets a bad reputation
as "untrustable."

**Prevention:**
- **Per-axis caps.** Any trait that modifies damage output, HP, AC, hit chance, etc. must
  have an explicit post-stacking clamp defined alongside the trait type. Example:
  total damage bonus from all sources clamped to +100% base.
- **Categorize stacking mode per trait.** Not every trait should be additive. Resistances
  naturally stack multiplicatively ("takes 50% of 50% = 25%"). Decide per trait type:
  `StatModifier` is additive, `ResistanceMultiplier` is multiplicative, `Immunity` is
  boolean union.
- **Monte-Carlo sim as tripwire.** Every new race/class PR runs the combat sim across all
  race×class×level combinations and fails if any win% exceeds the target corridor.
- **"Diminishing returns" helper.** For sensitive axes (hit bonus, dex→AC), expose
  `applyDiminishing(value, softcap)` utility rather than raw addition.

**Warning signs:** Trait PR where the author can't name what the upper bound is on their
modifier's stacked contribution. Any trait described as "just adds X."

**Phase:** TRAIT-01/02. Stacking policy is part of the trait type system, not a later
polish pass.

---

### Pitfall 8: Trait sprawl — one-off traits for one-off behaviors

**What goes wrong:** `VulnerableToSilver`, `VulnerableToWoodenStake`, `VulnerableToSunlight`,
`ResistSunlight`, `VulnerableToHolyWater`... each new monster adds one-off trait flags.
After 50 races, there are 200 trait kinds and nobody knows which are still referenced.
Combat code becomes `if HasTrait(VulnerableToSilver) || HasTrait(VulnerableToMithril) ...`
— exactly the identity-check scatter the refactor was meant to eliminate, just with trait
names instead of race constants.

**Why it happens:** Easy to add a new trait. No pressure to generalize. Content authors
(builders) bypass code review.

**Consequences:** Maintainability collapse. The refactor's core value prop —
"add a race by writing a data file" — becomes "add a race by writing a data file AND
adding trait kinds AND updating combat code to honor them." Back to square one.

**Prevention:**
- **Parameterized traits.** Instead of `VulnerableToSilver` as a distinct kind, use
  `Vulnerability{DamageType: DamSilver, Multiplier: 2.0}`. The trait *kind* is
  `Vulnerability`; the damage type is data. Combat code handles one kind.
- **Closed trait kind set.** `TraitKind` is an enum with ~10-15 members defined in Go.
  Adding a new kind requires Go code + review. Adding a new *instance* of an existing
  kind requires only TOML. This preserves the "data-only for new races" promise while
  preventing trait-kind inflation.
- **Audit on every milestone.** List all trait kinds in use and all code sites that match
  on specific kinds. If the list grows faster than race count, something is wrong.

**Warning signs:** Two traits that only differ in one parameter. Combat code with a long
`switch trait.Kind` across similar cases. Proposals for a "generic `Custom` trait with
free-form data."

**Phase:** TRAIT-01. Nail the parameterization story before writing TOML files, or the
TOML files encode the wrong shape.

---

### Pitfall 9: Lua hooks silently replacing, not augmenting, Go behavior

**What goes wrong:** `OnDeath` hook fires — does it *replace* the default death handling
or *augment* it? If replace, a buggy Lua script prevents corpse creation and item drops.
If augment, the hook can't suppress default behavior when that's the whole point (vampire
"dies" but rises as a bat).

**Why it happens:** Hook semantics aren't specified up-front. Builders assume one thing,
Go code does another.

**Consequences:** Bug reports that are actually design ambiguities. Inconsistent behavior
across races. Rework of existing hook scripts when semantics clarify.

**Prevention:**
- Make the Go↔Lua contract explicit. Every hook has a documented type:
  - `Pre`-hook: runs before default, can cancel by returning `false`.
  - `Post`-hook: runs after default, return value ignored.
  - `Replace`-hook: hook *is* the behavior. Absence of hook means default runs.
- Encode this in the trait definition itself: `BehaviorHook{Event: OnDeath, Mode: Post}`.
- Type-check hook return values on the Go side; log + skip if script returns wrong shape.

**Warning signs:** A hook TOML file that doesn't specify when it runs relative to defaults.
A hook script that reads succeed/fail conditions from magic globals.

**Phase:** LUA-02. Lock the semantics the day hook events are added.

---

### Pitfall 10: TOML load failure on startup with no migration path

**What goes wrong:** Admin edits `races/elf.toml`, introduces a typo, restarts server.
Server fails to start. All players offline. The fault is at the file they just edited but
the error message says `cannot unmarshal string into int field at line 47` — not helpful.

**Why it happens:** Data-driven systems shift errors from compile-time to load-time.
Schemaless loaders (which is what Go's default TOML parser provides) surface errors as
unhelpful reflect messages.

**Consequences:** Outages from edits. Loss of trust in the data-driven approach. Admins
start preferring "just let me edit Go."

**Prevention:**
- Validate every TOML file against a declared schema. Options:
  - Hand-written `Validate()` method on each trait struct.
  - JSON Schema + TOML→JSON validation via `json-schema-everywhere`.
- On startup, load *all* files, collect *all* errors, present a readable summary, refuse
  to start only if a validated-required file is invalid. Don't bail on the first error.
- Provide a standalone CLI validator (`rot-mud validate races/`) that admins can run
  before deploying.
- Version the schema. `races/elf.toml` has a `schema_version = 1` header. Loader
  supports old versions with warnings, refuses unknown future versions. (See Nano Node's
  TOML migration pattern.)

**Warning signs:** A loader that calls `toml.Unmarshal` into a struct and nothing else.
"Just fix the typo and restart" as the operator workflow.

**Phase:** DATA-01/02/03. Schema validation is part of the loader, not a future
enhancement.

---

### Pitfall 11: Trait query API performance cliff

**What goes wrong:** `ch.HasTrait(TraitKind)` is called in the inner hit loop for every
swing. If implemented as `for _, t := range ch.Traits { if t.Kind == kind ... }`, a
character with 30 traits costs 30 comparisons per query. Combat sim at L100 runs millions
of queries per simulated fight; the test suite slows from 10s to 5min.

**Why it happens:** "It's just a slice, how bad could it be?"

**Consequences:** Combat sim becomes unusable as a tuning tool — exactly the tool the
tuning notes say is load-bearing. Production player experience also degrades at
combat-heavy peak loads.

**Prevention:**
- Compose traits at character load/creation time into a denormalized struct:
  `ResolvedTraits { Immunities ImmFlagsBitset; Resistances ImmFlagsBitset; Vulnerabilities
  ImmFlagsBitset; StatModifiers [MaxStats]int; Capabilities CapFlagsBitset; ... }`.
- `HasImmunity(damType)` is a single bitmask test. `GetStatModifier(Stat)` is an array
  index.
- Re-resolve only when the trait set changes (race/class change — rare; equipment affects
  — existing affect system handles this).
- Benchmark the API surface before migrating combat call sites. `go test -bench` should
  show sub-nanosecond `HasTrait`.

**Warning signs:** `HasTrait` implementation that iterates on every call. Trait query used
inside a tight loop without caching.

**Phase:** TRAIT-03. Measure before migrating combat code (COMBAT-01) — if the API is
slow, the migration amplifies the slowness across every damage site.

---

### Pitfall 12: Hook script errors crashing the combat step

**What goes wrong:** A Lua script references `nil.hp` and throws. If not caught, the
error propagates up through the hook dispatcher, through the combat step, and panics the
game-loop goroutine. Without a `recover()` in the game loop, the whole server dies from
one bad script.

**Why it happens:** Defensive programming against user input is easy to forget when the
user is a trusted admin. Lua errors feel "far away" from Go panics.

**Consequences:** Server crash. All unsaved player state lost. If the bad script runs on
every combat tick, the server crashes on restart too — recovery requires SSH access to
disable the hook file.

**Prevention:**
- Wrap every hook invocation in `L.PCall` (Lua's protected call). Script errors become Go
  `error` values, logged, hook skipped.
- Wrap the game loop in `defer recover()` with structured logging; a panic aborts the
  current tick, not the whole server. (This is already called out in CONCERNS.md as a
  "missing critical feature.")
- Circuit-breaker per hook: if a hook errors N times in M minutes, auto-disable it and
  alert. Prevents flapping-error amplification.
- Admin command to disable hooks at runtime without file edits, as emergency recovery.

**Warning signs:** Any hook dispatcher that uses `L.Call` (unprotected). No `recover()`
in the game loop. No runtime hook-disable command.

**Phase:** LUA-01 (pcall), cross-cutting with game-loop resilience (addresses
CONCERNS.md `Comprehensive Error Recovery`).

---

### Pitfall 13: Race/class identity checks persist after "migration"

**What goes wrong:** Audit of the codebase: `ch.Class == types.ClassVampire || ch.Class ==
types.ClassLich` in `combat/combat.go:321`, `combat_sim_test.go:1080` hardcodes race
constants, `magic/magic_test.go` references them 7 times per the grep. Migration claims
"done" but these checks remain, so adding the promised Vampire data-file race (PROOF-01)
still doesn't work without Go changes.

**Why it happens:** Grep-and-replace is incomplete. Identity checks hide in tests, helper
functions, switch statements. The refactor target is "no `== RaceX`" but validators only
check production code.

**Consequences:** PROOF-01 fails — the headline value prop doesn't work. Months of
migration work appear not to have solved the problem.

**Prevention:**
- **Forbidden-pattern lint.** CI rule: no file outside `pkg/types/` and
  `pkg/loader/` may reference `types.Race<Name>` or `types.Class<Name>` constants.
  Violations fail the build.
- Audit pass as migration acceptance: `rg 'types\.Race[A-Z]|types\.Class[A-Z]'` outside
  the allowed list must return zero results.
- `isVampire()` helper (combat.go:321) becomes `ch.HasTrait(TraitUndead)` or
  `ch.HasCapability(CapBloodSucker)`. The helper goes away or becomes a thin trait
  wrapper.
- Tests get the same treatment — `combat_sim_test.go` loads races by name from TOML, not
  by integer constant.

**Warning signs:** "We'll clean up the tests later." Any code path where the concept of
"vampire" leaks into non-data files.

**Phase:** PROOF-01 gating check. No migration is complete until the audit is zero.

---

### Pitfall 14: No characterization test suite → migration proceeds blind

**What goes wrong:** The project migrates 19 races and 14 classes with no automated
parity suite. Existing tests (`combat_sim_test.go`) cover L100 balance, not field-by-field
equivalence. Migration lands, subtle bug introduced, no one catches it for weeks.

**Why it happens:** Golden-master testing is tedious to set up. Tempting to "just review
the diff carefully."

**Consequences:** The one thing the project must get right (behavior parity, per PROJECT.md
constraints) is unverified.

**Prevention:**
- **Before touching any TOML, build the golden-master fixture.** Script that:
  - For each race × class combination,
  - For levels 1, 10, 25, 50, 75, 100,
  - Creates a character, runs a fixed 100-turn solo combat script against a fixed dummy,
  - Records final HP, mana, move, AC, THAC0, XP gained, damage dealt/taken, affects applied,
  - Stores as JSON snapshot.
- Migration PR runs the script again, diffs against snapshot, must be identical.
- Any intentional behavior change is a separate PR that updates snapshots with review.
- Source: characterization test pattern from Codurance/Fabrizio Duroni (see Sources).

**Warning signs:** PR description says "behavior should be identical, reviewed carefully."
No snapshot fixtures in the migration PR diff.

**Phase:** Prerequisite for MIGRATE-01. The snapshot script is phase 0 work.

---

## Minor Pitfalls

### Pitfall 15: Hot-reload temptation despite being "out of scope"

**What goes wrong:** PROJECT.md says hot-reload is out of scope, but six months in,
"reload races" becomes an admin convenience request. Someone implements it without
thinking through running combat (mid-fight a vampire stops being a vampire; resistances
shift mid-swing).

**Prevention:** Keep it out of scope. If demand arises, design it properly: reload into
a shadow table, swap atomically at a safe point (not during combat), discard characters
in the middle of trait-dependent actions.

**Phase:** Watch-list for scope creep in any future milestone.

---

### Pitfall 16: Builder-authored Lua with no review process

**What goes wrong:** Builders write hook scripts. Nobody reviews them. Subtle balance
issues ship to production.

**Prevention:** Version-control hook files. Require PR review for race/class data
changes (builders commit to a dev branch). Even without security concerns (see Pitfall 5),
balance review is needed.

**Phase:** Operational, not code. Document in contribution guide.

---

### Pitfall 17: Lua version pinning drift

**What goes wrong:** `go.mod` uses `yuin/gopher-lua` without a minimum version. A future
`go get -u` pulls a new version with subtly different semantics (e.g. stricter number
coercion). All hook scripts break at once.

**Prevention:** Pin exact version. Upgrades are deliberate PRs with hook script regression
runs. Matches CONCERNS.md `No Version Pinning Constraints`.

**Phase:** LUA-01.

---

### Pitfall 18: Deprecation path for retired races/classes

**What goes wrong:** A future milestone removes a race ("Heucuva retired for lore reasons").
Existing players of that race log in → nil dereference because `RaceByName("heucuva")`
returns nil.

**Prevention:** Never delete race/class entries; mark `deprecated = true` and either
(a) refuse new character creation but allow loads, or (b) offer a one-time migration to a
similar race. Save-format compatibility table (Pitfall 2) lists retired races with
migration targets.

**Phase:** Operational policy. Document alongside DATA-01/02.

---

## Phase-Specific Warnings

| Phase topic | Likely pitfall(s) | Mitigation |
|-------------|-------------------|------------|
| TRAIT-01 (typed trait structs) | #7 (stacking), #8 (sprawl) | Closed kind set; stacking policy per kind from day 1 |
| TRAIT-02 (additive composition) | #7 (stacking), #11 (perf) | Per-axis caps; resolved-trait cache at composition time |
| TRAIT-03 (query API) | #11 (perf cliff) | Bitmask-based; benchmark before combat migration |
| DATA-01/02/03 (TOML loaders) | #2 (save-file index shift), #10 (startup fail) | Name-keyed saves; schema validation; versioned headers |
| LUA-01 (VM embed) | #1 (goroutine), #3 (runaway), #4 (memory), #5 (sandbox), #12 (errors), #17 (pinning) | Pool pattern; pcall+ctx; hand-written API; restricted libs; pinned version |
| LUA-02 (hook events) | #9 (semantics) | Pre/Post/Replace modes declared per hook |
| COMBAT-01 (trait-based combat) | #11 (perf), #13 (identity leakage) | Benchmark; CI forbidden-pattern lint |
| MIGRATE-01/02 (race/class migration) | #2 (saves), #6 (divergence), #13 (leakage), #14 (no parity suite) | Golden-master fixture first; dual-run mode; audit lint |
| PROOF-01 (new race via data only) | #13 (identity leakage) | Audit must return zero before marking done |

---

## What Might I Have Missed?

Two additional areas worth flagging for phase-specific research when those phases come up:

- **Lua script debugging UX.** When a hook misbehaves, how does the builder diagnose it?
  gopher-lua's error traces are OK but not great. A phase that adds more hooks should
  invest in `__tostring` conventions, a `/script-trace` admin command, or similar.
- **Composite class interactions (tier 2 remort).** `ClassLich` is tier 2 undead. If
  remorting preserves traits from tier 1, a Lich character might carry both Vampire
  traits and Lich traits stacked — potentially the worst case for Pitfall 7. Needs
  explicit handling in composition rules.

---

## Sources

### Lua-in-Go embedding
- [yuin/gopher-lua README — LStatePool pattern and goroutine constraints](https://github.com/yuin/gopher-lua) — HIGH confidence (official repo)
- [gopher-lua Go Integration (DeepWiki)](https://deepwiki.com/yuin/gopher-lua/5-go-integration) — MEDIUM confidence (third-party docs)
- [gopher-lua issue #5 — concurrency discussion](https://github.com/yuin/gopher-lua/issues/5) — HIGH confidence (maintainer responses)
- [gopher-lua issue #56 — memory leak report](https://github.com/yuin/gopher-lua/issues/56) — MEDIUM confidence
- [gopher-lua issue #89 — terminating a running state](https://github.com/yuin/gopher-lua/issues/89) — MEDIUM confidence
- [gopher-lua issue #197 — memory optimizations](https://github.com/yuin/gopher-lua/issues/197) — MEDIUM confidence
- [gopher-lua issue #335 — LState pooling](https://github.com/yuin/gopher-lua/issues/335) — MEDIUM confidence
- [gopher-lua discussion #437 — coroutine context leaks](https://github.com/yuin/gopher-lua/discussions/437) — MEDIUM confidence

### Lua sandboxing and runaway prevention
- [kikito/lua-sandbox — instruction-count hook pattern](https://github.com/kikito/sandbox.lua) — HIGH confidence
- [lua-users wiki: Sand Boxes](http://lua-users.org/wiki/SandBoxes) — HIGH confidence (community reference)
- [Luau sandboxing docs (Roblox)](https://luau.org/sandbox/) — HIGH confidence (production-grade game use)
- [Redis, Lua, and the Dangers In-Between (Upwind)](https://www.upwind.io/feed/redis-lua-and-the-dangers-in-between) — MEDIUM confidence (real CVE post-mortem)
- [Embedding Lua in the Source Engine (Valve)](https://developer.valvesoftware.com/wiki/Embedding_Lua_in_the_Source_Engine) — MEDIUM confidence

### Additive stacking / balance pitfalls
- [Paradox Forums: additive vs multiplicative bonuses](https://forum.paradoxplaza.com/forum/threads/additive-bonuses-vs-multiplicative-bonuses.1144836/) — MEDIUM confidence (designer discussion)
- [NeoGAF: multiplicative vs additive](https://www.neogaf.com/threads/gaf-i-need-your-help-understanding-multiplicative-vs-additive.1262391/) — LOW confidence
- [Path of Exile: Additive and Multiplicative modifiers](https://www.pathofexile.com/forum/view-thread/892570) — MEDIUM confidence

### Data-driven / ECS design
- [Wikipedia: Entity component system](https://en.wikipedia.org/wiki/Entity_component_system) — MEDIUM confidence
- [Data-oriented design: Component-Based Objects](https://www.dataorienteddesign.com/dodmain/node5.html) — MEDIUM confidence
- [Seba's Lab: ECS and maintainability](https://www.sebaslab.com/the-quest-for-maintainable-code-and-the-path-to-ecs/) — MEDIUM confidence

### Migration verification
- [Wikipedia: Characterization test](https://en.wikipedia.org/wiki/Characterization_test) — HIGH confidence
- [Codurance: Testing legacy code with Golden Master](https://www.codurance.com/publications/2012/11/11/testing-legacy-code-with-golden-master) — MEDIUM confidence
- [The Code Whisperer: Surviving Legacy Code with Golden Master](https://blog.thecodewhisperer.com/permalink/surviving-legacy-code-with-golden-master-and-sampling) — MEDIUM confidence
- [Fabrizio Duroni: Golden Master / Characterization testing](https://www.fabrizioduroni.it/blog/post/2018/03/20/golden-master-test-characterization-test-legacy-code) — MEDIUM confidence

### TOML schema / config migration
- [JSON Schema Everywhere: Schema Validation for TOML](https://json-schema-everywhere.github.io/toml) — HIGH confidence
- [Nano Node PR #2221: TOML config migration on startup](https://github.com/nanocurrency/nano-node/pull/2221) — MEDIUM confidence (real-world migration example)
- [toml-lang/toml issue #76: adding schema validation RFC](https://github.com/toml-lang/toml/issues/76) — MEDIUM confidence

### Codebase-specific (read during research)
- `/home/antti/Repos/Misc/ROT-MUD/.planning/PROJECT.md` — HIGH (source of truth)
- `/home/antti/Repos/Misc/ROT-MUD/.planning/codebase/CONCERNS.md` — HIGH (existing audit)
- `/home/antti/Repos/Misc/ROT-MUD/go/pkg/types/races.go` — HIGH
- `/home/antti/Repos/Misc/ROT-MUD/go/pkg/types/classes.go` — HIGH
- `/home/antti/Repos/Misc/ROT-MUD/go/pkg/combat/combat.go` (line 321 `isVampire`) — HIGH
- `/home/antti/Repos/Misc/ROT-MUD/go/pkg/combat/combat_sim_test.go` (lines 1080-1081 race constants in tests) — HIGH
