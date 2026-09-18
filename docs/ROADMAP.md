# ROT MUD Go Port - Roadmap

## Temporarily Disabled

Features implemented but disabled until prerequisites are met:

### Hunger/Thirst System
- Hunger and thirst tracking
- Condition effects on regeneration
- **Blocked by:** No food/drink shops implemented yet

## Not Yet Ported from C

Features from the original C codebase not yet implemented:

### Commands
- ooc - Out-of-character channel
- social (global) - Global social emotes
- gmote - Global emote
- cdonate - Clan-specific donation pit
- weddings - Wedding announcements board
- announce - Info/announcement channel
- sign - Sign posting system (IMM)
- squire/knight - Tier advancement commands (IMM)
- wedpost - Wedding post permission (IMM)
- randclan - Random clan assignment (IMM)

### Special Mob Behaviors
- spec_boaz - Unique NPC behavior
- spec_cast_judge - Law system judge casting
- spec_troll_member - Troll/Ogre faction rivalry
- spec_ogre_member - Troll/Ogre faction rivalry
- spec_dog_pee - Cosmetic dog behavior
- spec_cast_clan_adept - Clan hall healer NPC

### Systems
- Wedding Board - Wedding announcements
- Jukebox Lyrics Tick - Tick-based lyrics display
- Corner Room - Punishment room with restrictions

## Future Enhancements

Optional features for future implementation:

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

### Tier-2 follow-ups
Tier 2 (reroll + GodWars-style supernatural classes) is implemented — see
`.planning/TIER2-GODWARS.md`. Still open:
- Skill tables for tier-2 characters whose origin is ranger, druid or ghoul
- Deferred GodWars pieces: demon-lord/champion hierarchy, weaponform, head/tail mutations, imp/eyespy/firewall
- Tier-2 in the combat simulator (origin class + powers + demonic gear)

### Clan Halls
- Clan-owned rooms/areas
- Clan storage chests
- Clan board/messaging
- Customizable clan headquarters

### Additional Enhancements
- Weather effects on spells/combat
- Day/night cycle affecting gameplay
- Crafting system
- Achievement system
- Web admin dashboard
