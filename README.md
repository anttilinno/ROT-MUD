# ROT MUD - Go Port

A modern Go implementation of [ROT (Rivers of Time) MUD 1.4](https://github.com/anttilinno/ROT-MUD), originally written in C. ROT is derived from ROM 2.4, which traces its lineage back through Merc to the original DikuMUD.

This is a **modified fork** of the original port, intended for continued development and enhancements.

## Features

- Full combat system with multi-hit attacks, dual wielding, dodge, parry, shield block
- 60+ spells across damage, healing, buffs, debuffs, and utility
- Skill system with class-based availability and improvement
- 4 classes (Mage, Cleric, Thief, Warrior) and 5 races
- Clan, quest, pet/follower, shop, and note systems
- 61 areas with 4,072 rooms, 1,341 mobiles, 1,677 objects
- WebSocket support, REST API, Prometheus metrics

See [docs/FEATURES.md](docs/FEATURES.md) for the complete feature list.

See [docs/ROADMAP.md](docs/ROADMAP.md) for planned and unimplemented features.

## Quick Start

### Prerequisites
- Go 1.23+
- [mise](https://mise.jdx.dev/) (optional, for task running)

### Build and Run

```bash
# Using mise
mise run build
cd go && ./rotmud

# Or directly with Go
cd go && go build -o rotmud ./cmd/rotmud && ./rotmud
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
├── c_original/          # Original C source tarball
├── docs/
│   ├── FEATURES.md      # Complete feature list
│   └── ROADMAP.md       # Planned features
├── go/
│   ├── cmd/
│   │   ├── rotmud/      # Server entry point
│   │   └── areconv/     # ROM .are to TOML converter
│   ├── pkg/
│   │   ├── types/       # Core data types (Character, Object, Room)
│   │   ├── server/      # TCP/WebSocket/REST networking
│   │   ├── game/        # Game loop, commands, socials
│   │   ├── combat/      # Combat system
│   │   ├── magic/       # Spell system
│   │   ├── skills/      # Skill system
│   │   ├── ai/          # NPC AI and special behaviors
│   │   ├── builder/     # OLC editors
│   │   ├── loader/      # TOML area loading
│   │   ├── persistence/ # Player save/load
│   │   ├── shops/       # Economy system
│   │   └── help/        # Help system
│   └── data/
│       ├── config.toml  # Server configuration
│       ├── areas/       # World data (61 areas)
│       └── help/        # Help files
└── .mise.toml           # Task runner configuration
```

## Configuration

Edit `go/data/config.toml`:

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

# Build server
mise run build

# Build and run
mise run dev

# Build area converter
mise run areconv

# Convert ROM areas to TOML
mise run convert-areas

# Clean build artifacts
mise run clean
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

## Original C Source

The original C source tarball in `c_original/` was downloaded from:

**Primary source:**
- https://mudlistings.com/resources/files/Codebases/Rot1.4OLCGCC4.tar.gz

**Alternative sources:**
- https://github.com/prool/ROT-MUD.git
- https://itbacon.com/2023/09/11/rot-mud-v1-4-with-olc-2023

Extract with `tar -xzf c_original/Rot1.4OLCGCC4.tar.gz` to browse the original source. See the `doc/` directory inside for original license files.

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
