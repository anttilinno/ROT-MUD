# ROT MUD Go Port - Features

## Core Systems

- Game loop with pulse-based timing (250ms pulses)
- Violence, mobile, character tick, and area reset updates
- Full combat system with multi-hit, dodge, parry, shield block
- Magic system with 60+ spells
- Skill system with training and improvement
- Affect system with duration tracking
- Death handling with corpse creation and respawn
- HP, mana, and movement regeneration with position modifiers

## Commands

### Movement
- north, south, east, west, up, down
- enter (walk through portals)
- recall

### Position
- sit, stand, rest, sleep, wake

### Information
- look, score, who, inventory, equipment
- affects, time, weather, report, examine
- scan, consider, areas, compare, count
- credits, whois, wizlist, worth
- skills, spells, peek, read, rules, story
- channels, motd/imotd, news, changes

### Communication
- say, tell, reply, gossip, music, grats
- ask, answer, cgossip (clan), emote, pmote
- shout, yell, gtell, note, pose, quote
- quiet, replay, qgossip, afk, deaf

### Object Manipulation
- get, drop, wear, wield, remove
- eat, drink, quaff, recite, zap, brandish
- fill, pour, envenom, give, put
- sacrifice, close, open, lock, unlock, pick
- buy, sell, list, value, second (dual wield)

### Combat
- kill, flee, cast, backstab, bash, kick
- trip, disarm, rescue, dirt, circle
- berserk, gouge, feed, murder, surrender

### Configuration
- alias/unalias, autolist, autoassist
- autoexit, autogold, autoloot, autosac
- autosplit, autostore, autopeek
- brief, color/colour, combine, compact
- description, long, nofollow, noloot
- nosummon, notran, outfit, password
- prompt, scroll, title, wimpy, prefix

### Miscellaneous
- save, quit, help, commands, socials
- quest, clan, deity, follow, group
- order, split, train, gain, practice
- sneak, hide, steal, track, visible
- play, voodoo, forge, donate, member
- forget/remember

### Immortal/Admin
- goto, stat, where, shutdown, advance
- restore, peace, echo, transfer, at
- load, purge, sockets, force, slay
- freeze, mstat, ostat, rstat
- mfind, ofind, mwhere, owhere
- invis, holylight, incognito
- aedit, redit, medit, oedit, resets, hedit
- mset, oset, rset, snoop, mload, oload
- switch, return, immkiss, violate, protect
- twit, pack, gset, ban/allow, disconnect
- pecho, wiznet, string, trust
- wizlock, newlock, log
- noshout, notell, noemote, nochannels
- vnum, clone, finger, zecho, gecho
- allpeace, recover, memory, poofin/poofout
- smote, immtalk, pardon, deny, penalty
- notitle, norestore, guild, noclan
- ghost, wecho, permban, flag

## Spells (60+)

### Damage
- magic missile, fireball, lightning bolt
- cause light/serious/critical
- chill touch, burning hands, shocking grasp
- acid blast, colour spray, demonfire
- energy drain, flamestrike, chain lightning
- meteor swarm, implode, disintegrate
- earthquake, call lightning (area)
- dispel evil, dispel good, harm
- ray of truth, heat metal

### Healing
- cure light/serious/critical, heal, refresh
- cure blindness, cure poison, cure disease
- remove curse, mass healing

### Buffs
- armor, bless, sanctuary, haste, giant strength
- detect invis/evil/good/hidden/magic
- invisibility, fly, pass door, infravision
- stone skin, shield, protection evil/good
- frenzy, fireshield, iceshield, shockshield

### Debuffs
- blindness, curse, poison, slow, sleep
- faerie fire, plague, weaken, charm person, calm

### Utility
- dispel magic, word of recall, teleport
- summon, locate object, farsight
- create food/water/spring, identify
- enchant armor/weapon, recharge
- floating disc, create rose, continual light
- cancellation, faerie fog, ventriloquate
- change sex, know alignment, nexus, portal
- transport, conjure, mass invis
- gate, control weather, fireproof
- restore mana, animate, resurrect
- protection voodoo, empower

### Breath Weapons
- acid breath, fire breath, frost breath
- gas breath, lightning breath

### Ultimate
- holy word, unholy word

## Skills

### Combat (Active)
- backstab, bash, kick, trip, disarm
- dirt kicking, gouge, stun, circle
- berserk, rescue

### Combat (Passive)
- second/third/fourth/fifth attack
- dual wield, dodge, parry, shield block
- enhanced damage, grip, hand to hand

### Thief
- sneak, hide, steal, pick lock
- peek, envenom, track

### Weapon Proficiencies
- sword, dagger, spear, mace
- axe, flail, whip, polearm

### Utility
- meditation, fast healing, haggle
- lore, recall, scrolls, staves, wands

### Vampire
- feed

## Game Systems

### Clan System
- Clan membership and ranks
- Clan commands (list, info, who, talk, leave)
- Leader commands (induct, outcast, promote, demote)
- Member invitations (accept, deny, invite, kick)
- Clan gossip channel (cgossip)
- PK rules enforcement

### Quest System
- Quest listing and information
- Quest acceptance and abandonment
- Progress tracking
- Kill, collect, and explore triggers
- XP and gold rewards

### Pet/Follower System
- Animate, resurrect, conjure spells
- Order command for charmed followers
- Follower limits

### Shop System
- Buy, sell, list, value commands
- Haggle skill integration
- Shop hours and item types

### Note/Board System
- Multiple boards (note, idea, news, changes)
- Note reading and writing
- Unread tracking

### MOBprog System
- 8 trigger types
- File-based program loading
- MOBprog commands (say, emote, echo, etc.)

### Special Mob Behaviors
- Dragon breath attacks
- Casting mob behaviors
- Guard behaviors
- Thief/nasty behaviors
- Utility behaviors (janitor, fido)
- Mayor patrol

### Object Stacking
- Quantity-based get/drop/give
- Combined inventory display

### Help System
- Help file loading and lookup
- Level-restricted entries

### OLC (Online Creation)
- Area editor (aedit)
- Room editor (redit)
- Mobile editor (medit)
- Object editor (oedit)
- Reset editor (resets)
- Help editor (hedit)
- TOML persistence

## Modern Features

### Networking
- TCP server (port 4000)
- WebSocket support (port 4001)
- Full login and character creation

### Admin API
- REST endpoints (/api/players, /api/stats, /api/shutdown)
- Prometheus metrics

### Data Management
- TOML format for world data
- JSON format for player saves
- 61 areas (4072 rooms, 1341 mobs, 1677 objects)

### Logging
- Structured logging (slog)
- Configuration file (TOML)
