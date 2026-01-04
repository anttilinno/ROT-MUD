# ROT MUD: Go Port - Status & Future

## Overview

The ROT MUD server has been ported from C to Go with modern architecture, TOML data formats, WebSocket support, and REST API.

- **Status:** ~99% Complete
- **Data Format:** TOML for world data, JSON for player saves
- **See:** `MISSING_FEATURES.md` for detailed porting status

---

## Project Structure

```
go_port/
├── cmd/
│   ├── rotmud/main.go       # Server entry point
│   └── areconv/             # ROM .are to TOML converter
├── pkg/
│   ├── types/               # Core data types (Character, Object, Room, etc.)
│   ├── server/              # TCP/WebSocket networking
│   ├── game/                # Game loop, commands, clans, quests, pets
│   ├── combat/              # Combat system
│   ├── magic/               # Spell system
│   ├── skills/              # Skill system
│   ├── loader/              # TOML area loaders, MOBprog loading
│   ├── persistence/         # Player save/load (JSON)
│   ├── builder/             # OLC system
│   ├── shops/               # Shop system
│   ├── help/                # Help system
│   └── ai/                  # NPC AI and special behaviors
└── data/
    ├── config.toml          # Server configuration
    ├── areas/               # 61 areas in TOML format
    │   └── <area>/
    │       ├── area.toml
    │       ├── mobs/
    │       ├── objects/
    │       ├── rooms/
    │       └── mobprogs/    # MOBprog files
    └── players/             # Player saves (JSON)

c_original/
└── src/                     # Original C source code
```

---

## Completed Features

### Core Systems
- Game loop with pulse-based timing (250ms pulses)
- Violence, mobile, character tick, and area reset updates
- Full combat system with multi-hit, dodge, parry, shield block
- Magic system with 60+ spells
- Skill system with training and improvement
- Affect system with duration tracking
- Death handling with corpse creation and respawn

### Commands
- All movement commands (6 directions, enter, recall)
- All position commands (sit, stand, rest, sleep, wake)
- All information commands (look, score, who, inventory, equipment, etc.)
- All communication commands (say, tell, gossip, channels, notes)
- All object commands (get, drop, wear, wield, give, put, etc.)
- All combat commands (kill, flee, backstab, bash, kick, etc.)
- All configuration commands (alias, autolist, prompt, etc.)
- All immortal commands (goto, stat, force, slay, OLC editors, etc.)
- Special commands: play, voodoo, quest, clan, member

### Systems
- Clan system with ranks, induction, PK rules
- Quest system with kill/collect/explore triggers
- Pet/follower system (animate, resurrect, conjure)
- Shop system with haggle skill
- Note/board system (note, idea, news, changes)
- MOBprog system with file-based loading
- Object stacking with quantity commands
- Help system with level-restricted entries
- NPC AI with special behaviors

### Modern Features
- WebSocket support with full login/character creation
- REST admin API (/api/players, /api/stats, /api/shutdown)
- Prometheus metrics
- Structured logging (slog)
- Configuration file (TOML)

### Data
- 61 areas converted (4072 rooms, 1341 mobs, 1677 objects)
- Full player persistence (equipment, inventory, affects, quests)
- OLC editors save to TOML files

---

## Not Yet Ported from C

See `MISSING_FEATURES.md` Section 7 for full details. Key items:

- **Commands:** delete, reroll, ooc, gmote, cdonate, weddings, announce
- **Specials:** spec_boaz, spec_cast_judge, spec_troll_member, spec_ogre_member, spec_cast_clan_adept
- **Systems:** Tier/Remort, Wedding board, Jukebox lyrics tick

---

## Future Implementations (Optional)

These features were not fully implemented in the original ROT MUD or are optional enhancements:

### Bank System
- Gold deposit/withdraw at banker NPCs
- Interest accumulation over time
- Secure storage between sessions

### Auction System
- Player-to-player item auctions
- Bid/buyout mechanics
- Auction channel broadcasting
- Auction house NPC

### Wedding System
- Marriage ceremonies between players
- Wedding rings/items
- Spouse commands
- Divorce mechanics

### Tier/Remort System
- Multi-tier advancement beyond max level
- Remort to restart with bonuses
- Tier-specific skills/spells
- Legacy stat bonuses

### Clan Halls
- Clan-owned rooms/areas
- Clan storage chests
- Clan board/messaging
- Customizable clan headquarters

### Additional Enhancements
- Weather system effects on spells/combat
- Day/night cycle affecting gameplay
- Crafting system
- Achievement system
- Web-based admin dashboard

---

## Running the Server

```bash
# Build
cd go_port && go build ./...

# Run tests
go test ./... -count=1

# Start server
./cmd/rotmud/rotmud

# Connect
telnet localhost 4000
# or WebSocket at ws://localhost:4001
```

---

## Key Go Packages Used

| Package | Purpose |
|---------|---------|
| `log/slog` | Structured logging |
| `encoding/json` | Player save serialization |
| `github.com/pelletier/go-toml/v2` | TOML parsing |
| `github.com/gorilla/websocket` | WebSocket support |
| `github.com/prometheus/client_golang` | Metrics |

---

## License

Subject to Diku, Merc, ROM, and ROT licenses. See `doc/` for details.
