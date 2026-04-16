# Codebase Concerns

**Analysis Date:** 2026-04-16

## Security Concerns

**Weak Password Hashing:**
- Issue: Passwords are hashed using SHA256 only, without salt. This allows rainbow table attacks and GPU-based cracking.
- Files: `go/pkg/server/login.go:1942-1950`
- Impact: Player accounts vulnerable to brute-force and rainbow table attacks if player database is compromised
- Fix approach: Replace SHA256 with bcrypt, scrypt, or Argon2 hashing with proper salt generation per password

**Hardcoded API Key:**
- Issue: API key is hardcoded as "changeme" in the server initialization with a TODO comment indicating it should be loaded from config
- Files: `go/pkg/server/server.go:89`
- Impact: If API endpoints are exposed (WebSocket or REST), they are protected by a default credential
- Fix approach: Load API key from environment variable or config file at startup; validate it exists before starting server

**JSON Unmarshaling Without Validation:**
- Issue: Player files, config files, and note data are unmarshaled from JSON/TOML without schema validation
- Files: `go/pkg/persistence/player.go:177`, `go/pkg/loader/loader.go` (multiple), `go/pkg/game/notes.go:86`
- Impact: Malformed player saves or corrupted data files could cause panics or data loss
- Fix approach: Add schema validation after unmarshal operations; implement recovery handlers

**Plaintext Network Communication:**
- Issue: All TCP connections use plaintext protocol without TLS encryption
- Files: `go/pkg/server/server.go` (all connection handling)
- Impact: Passwords are transmitted in plaintext over the network; vulnerable to sniffing
- Fix approach: Implement optional TLS support (at minimum for password transmission)

## Tech Debt

**Monolithic Command Handler:**
- Issue: `go/pkg/game/commands.go` is 5566 lines - largest file in codebase. Contains dozens of command implementations in single file.
- Files: `go/pkg/game/commands.go`
- Impact: Difficult to navigate, test individual commands, or refactor; merge conflicts likely
- Fix approach: Split into command-specific files (e.g., `commands_examine.go`, `commands_help.go`, etc.); use command factory pattern

**Large Spell System File:**
- Issue: `go/pkg/magic/spells.go` is 3192 lines with many spell implementations inline
- Files: `go/pkg/magic/spells.go`
- Impact: Hard to modify individual spells; code duplication between similar spells
- Fix approach: Implement spell registry pattern; move spell implementations to separate files or generated from data

**Combat Simulation Test as Tuning Tool:**
- Issue: `go/pkg/combat/combat_sim_test.go` is a 100+ line combat balancing simulation, not a traditional test
- Files: `go/pkg/combat/combat_sim_test.go:1-35` (comments indicate manual tuning state)
- Impact: Not integrated into CI pipeline; manual testing burden; combat balance is not validated automatically
- Fix approach: Move to separate benchmarking/balance validation tool; integrate with performance testing suite

**Incomplete Furniture Bonus System:**
- Issue: Two TODOs in game loop indicate furniture bonuses are not implemented but referenced
- Files: `go/pkg/game/loop.go:418, 490`
- Impact: Furniture items cannot provide stat/regen bonuses as intended in original MUD
- Fix approach: Implement furniture bonus calculation and apply in regen functions

**Incomplete MOBprog Trigger System:**
- Issue: MOBprog surround trigger (TRIG_SURR) is not implemented
- Files: `go/pkg/game/commands_combat.go:1158`
- Impact: NPC scripts cannot detect when surrounded; reduces NPC behavior variety
- Fix approach: Implement surround detection and trigger MOBprog system

## Incomplete Features

**NPC Thievery System:**
- Issue: Special ability to steal objects from players is marked TODO and only gold stealing is implemented
- Files: `go/pkg/ai/specials.go:610`
- Impact: Thief mobs can only steal gold, not equipment; reduces combat encounter variety
- Fix approach: Implement object theft with weight/carried limits; add steal success/failure messages

**NPC Flee Movement:**
- Issue: NPC flee action doesn't actually move to a random exit - just stops fighting
- Files: `go/pkg/ai/specials.go:656`
- Impact: Mobs appear to flee but stay in room, reducing believability of NPC AI
- Fix approach: Select random exit and move mob using existing CharMover callback

**Energy Drain Spell Incomplete:**
- Issue: Energy Drain spell drains HP and heals caster, but XP drain and mana halving are marked TODO
- Files: `go/pkg/magic/spells.go:1694`
- Impact: High-level spell is missing mechanics; mages have less threatening caster abilities
- Fix approach: Implement XP loss (percentage of victim's next level exp) and mana halving

**Hunger and Thirst System Disabled:**
- Issue: Hunger/thirst mechanics are completely disabled pending food/drink shop implementation
- Files: `go/pkg/game/loop.go:252, 402, 474, 511` (all regeneration functions reference disabled system)
- Impact: Players never starve/thirst; no survival gameplay element; unused condition tracking in saves
- Fix approach: Either implement food/drink shops or remove the disabled code entirely

**Furniture Command Parsing:**
- Issue: The command parser has a stub for parsing command execution that references unimplemented furniture
- Files: `go/pkg/game/loop.go:573`
- Impact: Furniture-related commands may not route correctly
- Fix approach: Complete furniture command routing and testing

## Known Bugs

**Command Dispatch Flow Issue:**
- Issue: The dispatch flow references parsing and furniture bonuses that are marked TODO or incomplete
- Files: `go/pkg/game/loop.go:410-573`
- Impact: Furniture bonuses and some command paths may not work as intended
- Fix approach: Complete command parsing loop; add integration tests for furniture interactions

## Performance Bottlenecks

**Linear Object Combining in Display:**
- Issue: `formatObjectList()` uses O(n²) algorithm to combine duplicate objects when display flag is set
- Files: `go/pkg/game/commands.go:53-100`
- Impact: Inventories with many duplicate objects (potions, ammo) will be slow to format
- Fix approach: Use map-based grouping instead of nested loop search

**NPC AI Updates Every Violence Pulse:**
- Issue: Special mob abilities run once per violence update (every 0.75 seconds) with no throttling
- Files: `go/pkg/ai/specials.go` (all mob special routines)
- Impact: Mobs like "nasty" can steal gold every 0.75s; no cooldown between special attempts
- Fix approach: Add per-mob ability cooldown tracking

**Regex-Based Matching in Command Recognition:**
- Issue: Command matching may use string operations without optimization hints
- Files: `go/pkg/game/commands.go` (command routing)
- Impact: High-traffic servers with hundreds of players will have command dispatch overhead
- Fix approach: Profile command dispatch; consider hash-based command tables

## Race Conditions

**Character Deletion During Command Execution:**
- Issue: Player character can be deleted/disconnected while command is still processing
- Files: `go/pkg/server/server.go:354-392` (OnDelete and DisconnectPlayer callbacks)
- Impact: Command code may reference deleted character after callback fires
- Fix approach: Implement reference counting or defer character cleanup until command completes

**Session Map Modification During Iteration:**
- Issue: In violence update, server iterates over game loop characters but character state can change
- Files: `go/pkg/server/server.go:134-149`
- Impact: Rare race condition if character dies/logs out during violence pulse update
- Fix approach: Use snapshot iteration pattern (copy character slice before iterating)

**Workspace Notes Concurrent Access:**
- Issue: Note editor state stored in global map with RWMutex protection, but editor content passed by value
- Files: `go/pkg/game/notes.go:253` (global noteEditorsMu)
- Impact: Note editor state could become inconsistent if edits happen simultaneously
- Fix approach: Use channel-based message passing for note editor operations

## Fragile Areas

**Combat Damage Calculation:**
- Files: `go/pkg/combat/hit.go:9-150+`, `go/pkg/combat/damage.go`, `go/pkg/magic/spells.go` (damage calculations)
- Why fragile: Multiple damage paths (weapon, spell, special attack) with overlapping responsibility; damage types and resistances scattered across codebase
- Safe modification: All damage changes must be tested with combat_sim_test.go across all class/race combinations; changes to spell damage require rebalancing
- Test coverage: Combat sim covers balance but not all spell interactions; no regression tests for specific damage formulas

**Player Persistence:**
- Files: `go/pkg/persistence/player.go`, `go/pkg/server/server.go` (save/load callbacks)
- Why fragile: Player state is complex (inventory, equipment, affects, quest progress, skills); partial saves could corrupt character
- Safe modification: Always add new fields with omitempty JSON tags; maintain backward compatibility with old save format; validate all loaded data
- Test coverage: Only basic save/load tested; no tests for corrupted/partial files

**World Loading and Room Management:**
- Files: `go/pkg/loader/loader.go`, `go/pkg/server/server.go:402-429`
- Why fragile: World state is loaded once at startup; no hot-reload capability; room vnums are hardcoded in multiple places
- Safe modification: Adding new world data requires server restart; room vnum references should use constants; test area loading with corrupted/incomplete files
- Test coverage: Integration test exists but doesn't cover error recovery

**Game Loop Architecture:**
- Files: `go/pkg/game/loop.go`, `go/pkg/server/server.go:99-114`
- Why fragile: Single event loop for all game updates (combat, regeneration, ambient events); no staging or rollback capability
- Safe modification: Changes to update order affect balance/gameplay; new update phases must be carefully placed; concurrent character modification risky
- Test coverage: Limited unit testing; mostly integration-tested through game flow

## Test Coverage Gaps

**Command Handler Missing Tests:**
- What's not tested: Individual command implementations (especially immortal commands, builder commands, skill system)
- Files: `go/pkg/game/commands.go:1-5566` (5566 lines but no corresponding test file)
- Risk: Commands could have bugs that don't manifest until player uses them; no regression prevention
- Priority: High - commands are critical player-facing code

**Combat Edge Cases:**
- What's not tested: Multiple simultaneous combat scenarios, special damage types, edge cases with low/high stats
- Files: `go/pkg/combat/` (multiple files with limited test coverage)
- Risk: Rare combat situations could crash server or corrupt character state
- Priority: High - combat is core gameplay

**Spell System Not Covered:**
- What's not tested: Individual spell implementations, mana costs, target validation, area-of-effect calculations
- Files: `go/pkg/magic/spells.go:1-3192` (only spell system test is combat sim)
- Risk: Spells could behave unexpectedly; balance issues won't be caught
- Priority: Medium - spells important but less frequently used than basic combat

**NPC AI Not Tested:**
- What's not tested: Special mob abilities, MOBprog trigger evaluation, NPC movement and pathfinding
- Files: `go/pkg/ai/specials.go` (no unit tests)
- Risk: Mob AI bugs only discovered when players encounter them
- Priority: Medium - affects gameplay feel but not critical systems

**Persistence Not Fully Tested:**
- What's not tested: Corrupted save files, partial saves, upgrade paths between save formats
- Files: `go/pkg/persistence/player.go` (basic test exists but not comprehensive)
- Risk: Player data loss or corruption on upgrade
- Priority: Medium-High - data loss is severe but rare

## Scaling Limits

**TCP Connection Handling:**
- Current: Single goroutine per connection with buffered channels and mutex-protected session map
- Limit: ~1000-5000 concurrent connections (Go goroutine overhead, mutex contention on session map)
- Scaling path: Implement connection pooling, consider epoll for high-connection scenarios, shard session map by connection hash

**Game Loop Single-Threaded:**
- Current: All character updates serialized in single game loop goroutine
- Limit: ~500-1000 concurrent players before command latency becomes noticeable
- Scaling path: Implement region-based parallelization (divide world into zones, update zones in parallel), use actor model for character updates

**Object Inventory:**
- Current: Linked list of objects in character inventory (common for MUDs)
- Limit: Performance degrades with >100 items in inventory
- Scaling path: Implement object stacking/grouping system, use map-based lookup for equipped items

**Combat Update Frequency:**
- Current: Violence updates every 0.75 seconds (3 pulses at 4 PulsePerSecond)
- Limit: 1000 players in combat = 1000+ actions per violence round
- Scaling path: Consider adaptive combat tick rate or region-based combat updates

## Dependencies at Risk

**No Version Pinning Constraints:**
- Risk: Go modules used without specific version constraints (if using latest tags)
- Impact: Dependency updates could introduce breaking changes
- Migration plan: Use go.mod with specific semantic versions; test against both old and new versions

**TOML Configuration Unmarshaling:**
- Risk: If TOML structure changes, loader will fail silently or panic
- Impact: Configuration updates could break world loading
- Migration plan: Add schema validation; implement config migration helpers

## Missing Critical Features

**Hot Reload:**
- Problem: Server must restart to load new areas, NPCs, objects, or config
- Blocks: Can't update content without downtime; players must disconnect during updates
- Approach: Implement hot-reload for area data with careful state migration

**Administrative Command Logging:**
- Problem: No audit trail of immortal commands (no creation/deletion/modification logs)
- Blocks: Can't investigate abuse or recover from accidental changes
- Approach: Implement command audit log with timestamp, player, target, changes

**Comprehensive Error Recovery:**
- Problem: Panics in command execution or combat aren't caught
- Blocks: Single malformed command from player can crash server
- Approach: Wrap command execution in defer recovery handler; implement graceful error handling throughout

**Persistent Transaction Support:**
- Problem: Character saves don't use atomic transactions; partial saves could corrupt data
- Blocks: Can't guarantee consistency during power loss or crash
- Approach: Implement atomic file operations (write to temp, rename) or database with transactions

---

*Concerns audit: 2026-04-16*
