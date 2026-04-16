# Architecture

**Analysis Date:** 2026-04-16

## Pattern Overview

**Overall:** Layered monolithic architecture with pulse-based game loop

The ROT MUD server uses a classic MUD architecture consisting of discrete layers: network, game loop, command dispatch, game systems (combat, magic, skills), and data persistence. All components communicate through the central game loop via channels and callbacks.

**Key Characteristics:**
- Pulse-based timing (4 pulses per second, 250ms) synchronized across all systems
- Event-driven through goroutines and channels for network I/O
- Central game loop manages all characters and coordinates system updates
- Each major system (combat, magic, skills, AI) operates independently but coordinates through callbacks
- Data-oriented design with immutable TOML files for world data and JSON for player saves

## Layers

**Network Layer (`pkg/server`):**
- Purpose: Handle TCP/telnet and WebSocket connections
- Location: `go/pkg/server/`
- Contains: TCP listener, session management, login state machine, REST API, metrics
- Depends on: Game loop (for command queueing), persistence (for player auth), loader (for world data)
- Used by: Client connections via TCP/WebSocket

**Command Dispatcher (`pkg/game`):**
- Purpose: Parse commands, route to handlers, manage command execution
- Location: `go/pkg/game/commands.go`, `go/pkg/game/commands_*.go`
- Contains: Command registry, dispatch logic, 150+ command implementations
- Depends on: Combat, magic, skills, builder (OLC), help, persistence systems
- Used by: Game loop via OnCommand callback

**Game Loop (`pkg/game/loop.go`):**
- Purpose: Pulse-based timing hub, character/NPC management, system coordination
- Location: `go/pkg/game/loop.go`
- Contains: Pulse timing, command queue, character list, callback registration
- Depends on: Combat, magic, skills, AI, loader for world reference
- Used by: Server (to queue commands and coordinate game tick)

**Combat System (`pkg/combat`):**
- Purpose: Attack resolution, damage calculation, death handling
- Location: `go/pkg/combat/`
- Contains: MultiHit, OneHit, Damage, defense checking, THAC0 calculations
- Depends on: Types, skills (for defensive abilities)
- Used by: Game loop (OnViolence callback), AI, magic (for spell damage)

**Magic System (`pkg/magic`):**
- Purpose: Spell casting, affect management, target resolution
- Location: `go/pkg/magic/`
- Contains: 40+ spell definitions, affect system, target resolution
- Depends on: Types, skills (for improvement), combat (for damage spells)
- Used by: Command dispatcher, AI (for mob spells)

**Skills System (`pkg/skills`):**
- Purpose: Skill proficiency tracking, improvement on use, NPC skill levels
- Location: `go/pkg/skills/`
- Contains: Skill definitions, CheckImprove logic, NPC skill assignment
- Depends on: Types, constants
- Used by: Combat (for defensive skills), magic (for improvement), commands (for training)

**AI System (`pkg/ai`):**
- Purpose: NPC behavior and special functions
- Location: `go/pkg/ai/`
- Contains: 15+ special functions (combat spellcasting, thief behavior, etc.), default behaviors
- Depends on: Types, magic, combat, game handlers
- Used by: Game loop (OnMobile callback) to update NPC behavior every 1 second

**Data Loader (`pkg/loader`):**
- Purpose: Load world data from TOML files
- Location: `go/pkg/loader/`
- Contains: Area, room, mobile, object loaders; World struct for all loaded data
- Depends on: Types
- Used by: Server startup to initialize world

**Persistence (`pkg/persistence`):**
- Purpose: Player save/load in JSON format
- Location: `go/pkg/persistence/`
- Contains: Player serialization, bcrypt password hashing
- Depends on: Types
- Used by: Login handler, command dispatcher (on save/quit)

**Builder/OLC (`pkg/builder`):**
- Purpose: In-game content creation system
- Location: `go/pkg/builder/`
- Contains: Room, mobile, object editors; validation logic
- Depends on: Types, loader
- Used by: Command dispatcher (for build commands)

**Help System (`pkg/help`):**
- Purpose: In-game help topic lookup and display
- Location: `go/pkg/help/`
- Contains: TOML-based help file loader, topic matching
- Depends on: (None - standalone)
- Used by: Command dispatcher (for help commands)

**Type Definitions (`pkg/types`):**
- Purpose: Core data structures
- Location: `go/pkg/types/`
- Contains: Character, Object, Room, Descriptor, Affect, flag types
- Depends on: (None - foundational)
- Used by: All other packages

## Data Flow

**Command Execution Flow:**

1. Client sends text via TCP/WebSocket
2. `server.Session` reads from connection
3. Input queued to `GameLoop.commands` channel
4. Game loop pops command during pulse
5. `CommandDispatcher.Dispatch()` called with Command
6. Dispatch parses input, applies aliases, looks up handler
7. Handler executes (modifies character/world state)
8. Handler calls output callback to send result to player
9. Prompt sent after command completes

**Combat Round:**

1. Game loop pulse triggers `OnViolence` callback (every 3 pulses = 750ms)
2. `CombatSystem.ViolenceUpdate()` processes all fighting pairs
3. For each attacker, `MultiHit()` determines attacks per round
4. Each `OneHit()` calculates hit roll vs AC
5. On hit, `Damage()` applies damage with resistances/immunities
6. If target dies, `MakCorpse()` creates corpse, awards experience
7. Combat updates halt and survivor continues

**Spell Casting Flow:**

1. Command handler calls `MagicSystem.Cast(caster, spellName, targetName, finder)`
2. System looks up spell definition
3. Validates mana cost, position, level requirements
4. Resolves target using provided finder function
5. Calls spell function to apply effect
6. Handles affect duration, stat modifiers, flag changes
7. Output sent to caster and affected characters

**NPC Behavior Tick:**

1. Game loop pulse triggers `OnMobile` callback (every 4 pulses = 1 second)
2. `AISystem.ProcessAllMobiles()` iterates all NPCs
3. For each NPC with special function, calls it with context (combat, magic, output)
4. Special functions perform actions (spell, attack, move, etc.)
5. If no special or special doesn't fire, default behaviors apply (wander, scavenge)
6. NPC position/actions update game state

**Area Reset Flow:**

1. Game loop pulse triggers `OnAreaReset` callback (every 120 pulses = 30 seconds)
2. `ResetSystem.ProcessReset()` iterates areas
3. For each reset object in area, execute reset (spawn mobs, objects, etc.)
4. Respawned entities added to rooms
5. Dead NPCs can be respawned if reset defined

**State Management:**

- **Character state** stored in `types.Character` struct, modified in-place
- **Room state** stored in `types.Room` struct with person/object lists
- **Game-wide state** stored in `GameLoop` struct (characters list, rooms map, etc.)
- **World data** stored in `loader.World` struct, loaded at startup (read-only)
- **Player data** stored in JSON files, loaded at login, saved on quit
- **Temporary effects** stored in `Character.Affects` slice, processed each tick

## Key Abstractions

**Command Handler:**
- Purpose: Encapsulate command logic
- Examples: `cmdNorth()`, `cmdCast()`, `cmdKill()` in `go/pkg/game/commands*.go`
- Pattern: `func (d *CommandDispatcher) cmdXxx(ch *types.Character, args string)`

**Special Function:**
- Purpose: Encapsulate NPC behavior
- Examples: `spec_cast_mage()`, `spec_thief()` in `go/pkg/ai/specials.go`
- Pattern: `func (s *AISystem) specXxx(ch *types.Character, ctx *SpecialContext) bool`

**Spell Function:**
- Purpose: Encapsulate spell effect
- Examples: `spellFireball()`, `spellHeal()` in `go/pkg/magic/spells.go`
- Pattern: `func (m *MagicSystem) spellXxx(caster, target *types.Character, level int) bool`

**Output Callback:**
- Purpose: Decouple game logic from network I/O
- Examples: `d.Output(ch, msg)` sends message to character
- Pattern: `func(ch *types.Character, msg string)` - allows server to route to session

**Act Function:**
- Purpose: Format messages with character pronoun substitution
- Location: `go/pkg/game/act.go`
- Pattern: Supports tokens like `$n` (name), `$e` (pronoun), `$m` (objective)

## Entry Points

**Server Entry:**
- Location: `go/cmd/rotmud/main.go`
- Triggers: Program start
- Responsibilities: Parse flags, create logger, initialize server, start listening

**Game Loop Start:**
- Location: `go/pkg/server/server.go` - `Start()` method
- Triggers: Server startup after data loading
- Responsibilities: Spawn game loop goroutine, start pulse timer, initialize systems

**Login Handler:**
- Location: `go/pkg/server/login.go`
- Triggers: New TCP/WebSocket connection
- Responsibilities: Greeting, password validation, character selection/creation, move to playing state

**Command Queue:**
- Location: `go/pkg/game/loop.go` - `QueueCommand()` method
- Triggers: Player input from network layer
- Responsibilities: Buffer command in channel, game loop will execute

## Error Handling

**Strategy:** Graceful degradation with logging

**Patterns:**
- **Critical errors** (loader, startup): Log and exit
- **Runtime errors** (command handler): Send user message, log error, continue
- **Validation errors** (spell requirements): Prevent action, send message to player
- **Network errors** (send fail): Log and disconnect session
- **Panic recovery**: Wrap goroutines with panic handlers to prevent server crash

Examples from codebase:
- `go/pkg/server/server.go` line 16: Server startup error exits immediately
- `go/pkg/game/commands.go` - Each handler checks preconditions and sends messages on failure
- `go/pkg/combat/combat.go` - Nil checks throughout to prevent panics

## Cross-Cutting Concerns

**Logging:** 
- Uses `log/slog` structured logging
- Initialized in `main()`, passed to server
- Errors and important events logged with context

**Validation:**
- Position requirements: Most commands check `ch.Position` before executing
- Level requirements: Immortal commands check `ch.Level`
- Resource requirements: Spells check mana, skills check proficiency
- Flag checks: Act flags determine NPC behaviors, affect flags track effects

**Authentication:**
- Passwords hashed with bcrypt in `persistence`
- Stored in player JSON files
- Checked during login in `server/login.go`
- Immortal/builder commands check `ch.PCData.Security`

**Thread Safety:**
- Game loop is single-threaded (command queue serializes access)
- Server uses RWMutex for session map (`mu sync.RWMutex`)
- Character state modified only during command execution or callbacks in game loop
- No concurrent access to character/room/object state during game execution

---

*Architecture analysis: 2026-04-16*
