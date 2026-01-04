# ROT MUD - Go Rewrite

A modern Go implementation of the ROT (Rivers of Time) 1.4 MUD server,
originally derived from ROM 2.4 and Merc Diku Mud.

## Overview

This is a complete rewrite of the classic ROT MUD server in Go, featuring:

- Modern architecture with goroutines and channels
- TOML-based world data format
- JSON player saves
- WebSocket support for browser clients
- REST API for administration
- Prometheus metrics
- Comprehensive test coverage

## Project Structure

```
rotmud/
├── cmd/rotmud/          # Server entry point
├── pkg/
│   ├── types/           # Core data types (Character, Object, Room)
│   ├── server/          # TCP/WebSocket networking
│   ├── game/            # Game loop and commands
│   ├── combat/          # Combat system
│   ├── magic/           # Spell system
│   ├── skills/          # Skill system
│   ├── ai/              # NPC behaviors
│   ├── builder/         # OLC (Online Creation)
│   ├── loader/          # TOML data loading
│   ├── persistence/     # Player saves
│   ├── shops/           # Economy system
│   └── help/            # Help system
└── data/
    ├── config.toml      # Server configuration
    ├── areas/           # World data (TOML)
    ├── players/         # Player saves (JSON)
    └── help/            # Help files (TOML)
```

## Building

```bash
cd rotmud
go build ./cmd/rotmud
```

## Running

```bash
./rotmud
```

The server listens on:
- Port 4000: Telnet connections
- Port 4001: WebSocket connections
- Port 4002: REST API / Metrics

## Configuration

Edit `data/config.toml`:

```toml
[server]
telnet_port = 4000
websocket_port = 4001
pulse_ms = 250

[logging]
level = "info"
format = "json"
```

## Testing

```bash
go test ./...
```

## Package Documentation

Each package includes a `doc.go` file with detailed documentation.
View with:

```bash
go doc rotmud/pkg/types
go doc rotmud/pkg/game
go doc rotmud/pkg/combat
# etc.
```

## Architecture

### Game Loop

The server uses a pulse-based timing system (4 pulses/second):

| Update Type | Frequency | Purpose |
|-------------|-----------|---------|
| Violence | 750ms | Combat rounds |
| Mobile | 1s | NPC AI |
| Tick | 15s | Regeneration, affects |
| Area | 30s | Area resets |

### Command Flow

1. Player input arrives via TCP/WebSocket
2. Command queued to game loop channel
3. Game loop dispatches to command handler
4. Handler modifies game state
5. Output sent back to clients

### Data Flow

```
TOML Files ─→ Loader ─→ Game State ─→ Persistence ─→ JSON Files
                              ↑
                        Game Loop
                              ↓
                         Players
```

## Key Features

### Combat System
- THAC0-based attack resolution
- Defensive skills (parry, dodge, shield block)
- Damage types with immunity/resistance
- Experience and leveling

### Magic System
- 30+ spells across damage, healing, buffs, debuffs
- Mana-based casting with class requirements
- Affect system for temporary buffs

### Skill System
- Class-based skill availability
- Improvement through use
- Training with practice points

### NPC AI
- 19 special behavior functions
- Aggressive/scavenger/sentinel behaviors
- Shop keepers with haggle support

### OLC (Online Creation)
- Room editor with exits and flags
- Mobile editor with specials
- Object editor with properties

## Original Credits

ROT 1.4 is based on:
- ROM 2.4 by Russ Taylor
- Merc Diku Mud by Michael Chastain, Michael Quan, Mitchell Tse
- Original Diku Mud by Sebastian Hammer, Michael Seifert, Hans Henrik Stærfeldt, Tom Madsen, Katja Nyboe

## License

This code is subject to the original Diku, Merc, ROM, and ROT licenses.
See the `doc/` directory for license files.
