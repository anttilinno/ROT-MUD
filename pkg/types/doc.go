// Package types defines the core data structures for the ROT MUD server.
//
// This package contains all fundamental types used throughout the game,
// ported from the original C codebase's merc.h header file.
//
// # Core Types
//
// The main entity types are:
//
//   - [Character]: Represents players and NPCs (from CHAR_DATA)
//   - [Object]: Represents items and equipment (from OBJ_DATA)
//   - [Room]: Represents locations in the world (from ROOM_INDEX_DATA)
//   - [Descriptor]: Represents network connections (from DESCRIPTOR_DATA)
//   - [Affect]: Represents temporary effects on characters (from AFFECT_DATA)
//
// # Flag Types
//
// The package provides bitfield flag types with Has/Set/Remove/Toggle methods:
//
//   - [ActFlags]: NPC behavior flags (sentinel, aggressive, etc.)
//   - [AffectFlags]: Character affect flags (invisible, sanctuary, etc.)
//   - [RoomFlags]: Room property flags (dark, safe, no-mob, etc.)
//   - [ItemFlags]: Object property flags (glow, magic, no-drop, etc.)
//   - [WearFlags]: Where objects can be equipped
//   - [ExitFlags]: Door/exit states (closed, locked, etc.)
//
// # Constants
//
// Game constants include:
//
//   - Directions (north, south, east, west, up, down)
//   - Positions (dead, sleeping, standing, fighting, etc.)
//   - Item types (weapon, armor, potion, etc.)
//   - Damage types (slash, bash, fire, cold, etc.)
//   - Stat indices (strength, intelligence, etc.)
//   - Class indices (mage, cleric, thief, warrior)
//
// # Usage Example
//
//	// Create a new character
//	player := types.NewCharacter("Gandalf")
//	player.Level = 50
//	player.Class = types.ClassMage
//
//	// Create an NPC
//	goblin := types.NewNPC(3001, "a goblin", 5)
//	goblin.Act.Set(types.ActAggressive)
//
//	// Check flags
//	if goblin.Act.Has(types.ActAggressive) {
//	    // Goblin will attack players
//	}
//
//	// Create a room
//	room := types.NewRoom(3001, "Town Square", "You are in the town square.")
//	room.Sector = types.SectCity
//	room.Flags.Set(types.RoomSafe)
package types
