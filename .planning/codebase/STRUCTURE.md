# Codebase Structure

**Analysis Date:** 2026-04-16

## Directory Layout

```
ROT-MUD/
├── go/                          # Go source code
│   ├── cmd/
│   │   └── rotmud/
│   │       └── main.go          # Server entry point
│   ├── pkg/                     # Core packages
│   │   ├── ai/                  # NPC behavior and special functions
│   │   ├── builder/             # Online Creation (OLC) system
│   │   ├── combat/              # Combat system (damage, THAC0, death)
│   │   ├── game/                # Game loop and command dispatcher
│   │   ├── help/                # Help system
│   │   ├── loader/              # Data loading from TOML
│   │   ├── magic/               # Spell and affect system
│   │   ├── persistence/         # Player save/load (JSON)
│   │   ├── server/              # Network layer (TCP, WebSocket, API)
│   │   ├── shops/               # Shop system
│   │   ├── skills/              # Skill proficiency and training
│   │   └── types/               # Core data structures
│   └── data/                    # Game world data
│       ├── config.toml          # Server configuration
│       ├── areas/               # Area definitions (60+ areas)
│       │   ├── midgaard/
│       │   │   ├── area.toml    # Area metadata
│       │   │   ├── mobs/
│       │   │   │   └── mobs.toml
│       │   │   ├── objects/
│       │   │   │   └── objects.toml
│       │   │   ├── rooms/
│       │   │   │   └── rooms.toml
│       │   │   └── mobprogs/    # Mobile program triggers
│       │   └── [other areas]/
│       ├── help/                # Help file entries (TOML)
│       └── players/             # Player save files (JSON)
├── c_original/                  # Original C source code (reference)
├── docs/                        # Documentation
├── .planning/                   # Analysis and planning documents
│   └── codebase/                # Generated codebase maps
├── .gitignore
├── .mise.toml                   # Tool configuration
├── README.md
└── LICENSE
```

## Directory Purposes

**`go/cmd/rotmud/`:**
- Purpose: Application entry point
- Contains: `main.go` only
- Key files: `go/cmd/rotmud/main.go`

**`go/pkg/ai/`:**
- Purpose: NPC artificial intelligence and special behaviors
- Contains: Special functions, default behaviors, AI system
- Key files: `go/pkg/ai/specials.go` (15+ special functions), `go/pkg/ai/system.go`

**`go/pkg/builder/`:**
- Purpose: Online Creation system for building game content
- Contains: Room, mobile, object editors; validation
- Key files: `go/pkg/builder/olc.go`

**`go/pkg/combat/`:**
- Purpose: Combat mechanics and resolution
- Contains: Hit calculation, damage, death, experience
- Key files: `go/pkg/combat/combat.go`, `go/pkg/combat/hit.go`, `go/pkg/combat/skills.go`

**`go/pkg/game/`:**
- Purpose: Game loop, command dispatcher, and game logic
- Contains: Pulse timing, command registry, 150+ command implementations
- Key files: `go/pkg/game/loop.go` (main loop), `go/pkg/game/commands.go` (5500+ lines), `go/pkg/game/commands_*.go` (specialized command groups)

**`go/pkg/help/`:**
- Purpose: In-game help system
- Contains: Help file loader, topic matching
- Key files: `go/pkg/help/help.go`

**`go/pkg/loader/`:**
- Purpose: Load world data from TOML files at startup
- Contains: Area, room, mobile, object loaders; World struct
- Key files: `go/pkg/loader/loader.go`, `go/pkg/loader/schema.go`

**`go/pkg/magic/`:**
- Purpose: Spell system and affect management
- Contains: 40+ spell definitions, affect decay, target resolution
- Key files: `go/pkg/magic/spells.go` (87KB - all spell implementations), `go/pkg/magic/system.go`, `go/pkg/magic/spell.go`

**`go/pkg/persistence/`:**
- Purpose: Player character save and load
- Contains: JSON serialization, bcrypt password hashing
- Key files: `go/pkg/persistence/player.go`

**`go/pkg/server/`:**
- Purpose: Network layer - TCP, WebSocket, REST API
- Contains: Session management, login state machine, metrics, API handlers
- Key files: `go/pkg/server/server.go` (46KB - main server), `go/pkg/server/login.go` (57KB - login), `go/pkg/server/websocket.go`

**`go/pkg/shops/`:**
- Purpose: In-game shop system
- Contains: Shop definitions, buying/selling logic
- Key files: Not analyzed in detail

**`go/pkg/skills/`:**
- Purpose: Skill system and training
- Contains: Skill definitions, proficiency tracking, improvement on use
- Key files: `go/pkg/skills/defaults.go`, `go/pkg/skills/system.go`

**`go/pkg/types/`:**
- Purpose: Core game data structures
- Contains: Character, Object, Room, Descriptor, Affect types; flag types
- Key files: `go/pkg/types/character.go`, `go/pkg/types/object.go`, `go/pkg/types/room.go`, `go/pkg/types/flags.go` (17KB - all flag definitions)

**`go/data/`:**
- Purpose: Game world configuration and content
- Contains: Area definitions, help files, player saves

**`go/data/areas/`:**
- Purpose: Individual area definitions
- Organization: One directory per area (60+ areas)
- Structure: Each area has `area.toml`, `mobs/`, `objects/`, `rooms/`, optionally `mobprogs/`

**`go/data/config.toml`:**
- Purpose: Server configuration (port, pulse rate, logging)
- Key settings: telnet_port (4000), websocket_port (4001), pulse_ms (250)

## Key File Locations

**Entry Points:**
- `go/cmd/rotmud/main.go`: Program entry, creates logger and server, starts listening

**Configuration:**
- `go/data/config.toml`: Server configuration (port, pulse, logging)

**Core Logic:**
- `go/pkg/game/loop.go`: Game loop timing and pulse coordination
- `go/pkg/game/commands.go`: Main command dispatcher and 150+ command implementations
- `go/pkg/server/server.go`: Network listener, session management, game loop integration
- `go/pkg/combat/combat.go`: Combat system and violence update
- `go/pkg/magic/spells.go`: All spell implementations and effects
- `go/pkg/ai/specials.go`: NPC special functions and behaviors

**Testing:**
- `go/pkg/combat/combat_sim_test.go`: Combat system testing and balancing (44KB)
- `go/pkg/game/commands_test.go`: Command dispatcher tests
- Various `*_test.go` files throughout `pkg/`

**Data Loading:**
- `go/pkg/loader/loader.go`: Loads areas, rooms, mobiles, objects from TOML
- `go/data/areas/*/`: Area definitions in TOML format

## Naming Conventions

**Files:**
- `*.go`: Source code files
- `*_test.go`: Test files
- `*.toml`: Configuration and data files (areas, help)
- `*.json`: Player save files

**Directories:**
- Lowercase, descriptive names (e.g., `combat`, `magic`, `server`)
- Package per directory (Go convention)
- Area directories match area name (e.g., `midgaard`, `draconia`)

**Go Functions:**
- Exported functions: PascalCase (e.g., `NewCharacter`, `SetFighting`)
- Unexported functions: camelCase (e.g., `charToRoom`, `findCharInRoom`)
- Command handlers: `cmd` prefix + descriptive name (e.g., `cmdNorth`, `cmdCast`)
- Special functions: `spec_` prefix (e.g., `spec_cast_mage`, `spec_thief`)
- Spell functions: `spell` prefix (e.g., `spellFireball`, `spellHeal`)

**Types:**
- PascalCase (e.g., `Character`, `CommandDispatcher`, `GameLoop`)
- Flag types: Descriptive constant prefix (e.g., `ActAggressive`, `AffInvisible`)
- Methods on types: Receiver variable `ch`, `d`, `s` depending on type

**Package Names:**
- Single word, all lowercase (Go convention)
- Match directory name

## Where to Add New Code

**New Feature:**
- Primary code: Add handler function in `go/pkg/game/commands.go` or `go/pkg/game/commands_*.go`
  - Pattern: `func (d *CommandDispatcher) cmdXxx(ch *types.Character, args string)`
  - Register in `registerBasicCommands()` or appropriate function
- Tests: Create `go/pkg/game/commands_test.go` with test cases
- If feature requires new system: Create new package in `go/pkg/`

**New Component/Module:**
- Implementation: `go/pkg/newpackage/` directory
- Doc file: `go/pkg/newpackage/doc.go` (required for all packages)
- Tests: `go/pkg/newpackage/*_test.go`
- Entry point: `go/pkg/newpackage/system.go` or main file
- Integration: Wire up in `server.New()` or relevant system initialization

**Utilities:**
- Shared helpers: Add to existing package if related, or create utility package
- String formatting: Add to `go/pkg/game/act.go` or appropriate handler
- File I/O: Extend `go/pkg/loader/` or `go/pkg/persistence/`

**New Game Content:**
- Areas: Create directory `go/data/areas/newarea/` with structure:
  - `area.toml`: Area metadata (name, vnum_range, credits)
  - `mobs/mobs.toml`: Mobile templates
  - `objects/objects.toml`: Object templates
  - `rooms/rooms.toml`: Room definitions
  - `mobprogs/` (optional): Mobile program triggers
- Help files: Add entries to `go/data/help/` TOML files

## Special Directories

**`go/pkg/game/`:**
- Purpose: Largest package, contains game loop and commands
- Files: `loop.go` (game timing), `commands.go` (5500+ lines), `commands_*.go` (specialized command groups)
- Organization: Command files grouped by category (combat, skills, thief, objects, etc.)
- Note: Size driven by command dispatch centralization - each command is a method

**`go/pkg/magic/`:**
- Purpose: Spell system with spell data and implementations
- Files: `spells.go` (87KB - all 40+ spells), `system.go`, `spells_data.go`
- Organization: Spell functions named `spellXxx` and collected by type

**`go/pkg/server/`:**
- Purpose: Network layer, most complex networking code
- Files: `server.go` (46KB), `login.go` (57KB), `websocket.go`
- Generated: `player.json` files in `go/data/players/` (one per logged-in character)
- Committed: `config.toml` only

**`go/data/players/`:**
- Purpose: Player save data storage
- Generated: Yes - created/updated at login and save
- Committed: Typically no - usually in .gitignore
- Format: JSON files named `<CharacterName>.json`

**`go/data/areas/`:**
- Purpose: World definition
- Generated: No - hand-authored content
- Committed: Yes - core game world
- Organization: 60+ area directories, each with structured TOML files

---

*Structure analysis: 2026-04-16*
