# ROT MUD - Go Port

A modern Go implementation of [ROT (Rivers of Time) MUD 1.4](https://github.com/anttilinno/ROT-MUD), originally written in C. ROT is derived from ROM 2.4, which traces its lineage back through Merc to the original DikuMUD.

This is a **modified fork** of the original port, intended for continued development and enhancements.

## Features

### Core Game Systems
- Full combat system with THAC0-based resolution, multi-hit attacks, dual wielding
- 60+ spells across damage, healing, buffs, debuffs, and utility
- Skill system with class-based availability and improvement through use
- 4 classes (Mage, Cleric, Thief, Warrior) and 5 races
- Clan system with ranks, PK rules, and clan channels
- Quest system with kill/collect/explore objectives
- Pet and follower system (charm, animate, summon)
- Shop system with haggle skill
- Note/board system for player communication

### World Data
- **61 areas** converted from original ROM format
- **4,072 rooms** to explore
- **1,341 mobiles** with AI behaviors
- **1,677 objects** including weapons, armor, and magical items
- MOBprog scripting for dynamic NPC behavior
- OLC (Online Level Creation) for building

### Modern Architecture
- Written in idiomatic Go with goroutines and channels
- TOML-based world data (human-readable and editable)
- JSON player saves
- WebSocket support for browser-based clients
- REST API for administration
- Prometheus metrics for monitoring
- Structured logging with `slog`

## Quick Start

### Prerequisites
- Go 1.23+
- [mise](https://mise.jdx.dev/) (optional, for task running)

### Build and Run

```bash
# Using mise
mise run build
./rotmud

# Or directly with Go
go build -o rotmud ./cmd/rotmud
./rotmud
```

### Connect

| Protocol | Port | Client |
|----------|------|--------|
| Telnet | 4000 | `telnet localhost 4000` |
| WebSocket | 4001 | `ws://localhost:4001` |
| REST API | 4002 | `http://localhost:4002/api/stats` |

## Project Structure

```
.
├── cmd/
│   ├── rotmud/          # Server entry point
│   └── areconv/         # ROM .are to TOML converter
├── pkg/
│   ├── types/           # Core data types (Character, Object, Room)
│   ├── server/          # TCP/WebSocket/REST networking
│   ├── game/            # Game loop, commands, socials
│   ├── combat/          # Combat system
│   ├── magic/           # Spell system
│   ├── skills/          # Skill system
│   ├── ai/              # NPC AI and special behaviors
│   ├── builder/         # OLC editors
│   ├── loader/          # TOML area loading
│   ├── persistence/     # Player save/load
│   ├── shops/           # Economy system
│   └── help/            # Help system
└── data/
    ├── config.toml      # Server configuration
    ├── areas/           # World data (61 areas)
    └── help/            # Help files
```

## Configuration

Edit `data/config.toml`:

```toml
[server]
telnet_port = 4000
websocket_port = 4001
api_port = 4002
pulse_ms = 250

[game]
start_room = 3001
recall_room = 3001

[logging]
level = "info"
```

## Development

```bash
# Run tests
mise run test
# or: go test ./...

# Build area converter
mise run areconv

# Convert ROM areas to TOML
mise run convert-areas
```

## Game Architecture

### Pulse System

The game loop runs on a pulse-based timing system (250ms per pulse):

| Update | Interval | Purpose |
|--------|----------|---------|
| Violence | 3 pulses (750ms) | Combat rounds |
| Mobile | 4 pulses (1s) | NPC AI decisions |
| Tick | 60 pulses (15s) | Regeneration, affect decay |
| Area | 120 pulses (30s) | Area resets, mob respawns |

### Command Flow

```
Player Input → TCP/WebSocket → Command Queue → Game Loop
                                                  ↓
                                           Command Handler
                                                  ↓
                                           Game State Update
                                                  ↓
                                           Output to Clients
```

## History

| Year | Project | Description |
|------|---------|-------------|
| 1990 | DikuMUD | Original codebase from University of Copenhagen |
| 1991 | Merc | Major rewrite of Diku |
| 1993 | ROM | "Rivers of Mud" - Enhanced Merc |
| 1996 | ROT | "Rivers of Time" - Enhanced ROM with OLC, clans, quests |
| 2025 | Go Port | Modern reimplementation in Go |

## Related

- [Original ROT-MUD Repository](https://github.com/anttilinno/ROT-MUD) - Contains both C source and initial Go port
- [ROM 2.4 QuickMUD](http://www.rom.org/) - ROM resources
- [MUD Listings Codebases](https://mudlistings.com/) - Various MUD codebases

## License

The Go code is released under the BSD Zero Clause License (0BSD).

The game content and design are subject to the original Diku, Merc, ROM, and ROT licenses which require attribution in derivative works.

## Credits

### Original ROT 1.4
- Jason Dinkel, Gary McNickle, and the ROT development team

### ROM 2.4
- Russ Taylor (rtaylor@hypercube.org)

### Merc DikuMud
- Michael Chastain, Michael Quan, Mitchell Tse

### Original DikuMUD
- Sebastian Hammer, Michael Seifert, Hans Henrik Staerfeldt, Tom Madsen, Katja Nyboe
