// Package builder implements the Online Creation (OLC) system for the ROT MUD.
//
// OLC allows authorized builders to create and edit game content in-game,
// including rooms, mobiles (NPCs), and objects. This is ported from the
// original olc.c, olc_act.c, and olc_save.c files.
//
// # Editor Modes
//
// The system supports three editor modes:
//
//   - [EditorRoom]: Edit room properties, exits, flags
//   - [EditorMobile]: Edit NPC templates
//   - [EditorObject]: Edit item templates
//
// # Room Editor Commands
//
// When editing a room:
//
//   - show: Display all room properties
//   - name <text>: Set room name
//   - desc <text>: Set room description
//   - sector <type>: Set terrain type (inside, city, forest, etc.)
//   - north/south/etc <vnum>: Create exit to room
//   - north/south/etc delete: Remove exit
//   - flags [flag]: Show or toggle room flags
//   - done: Save and exit
//
// # Mobile Editor Commands
//
// When editing a mobile:
//
//   - show: Display all mobile properties
//   - name <keywords>: Set targeting keywords
//   - short <text>: Set short description
//   - long <text>: Set room description
//   - level <1-200>: Set level
//   - align <-1000 to 1000>: Set alignment
//   - special <name>: Set special function
//   - done: Save and exit
//
// # Object Editor Commands
//
// When editing an object:
//
//   - show: Display all object properties
//   - name <keywords>: Set targeting keywords
//   - short <text>: Set inventory description
//   - long <text>: Set ground description
//   - level <0-200>: Set minimum level
//   - cost <amount>: Set gold value
//   - weight <amount>: Set weight
//   - done: Save and exit
//
// # Builder Security
//
// Builders must have a security level >= 1 in their PCData to use OLC.
// Higher security levels may grant access to more areas or dangerous options.
//
// # Usage Example
//
//	olc := builder.NewOLCSystem()
//	olc.Output = sendToPlayer
//	olc.GetRoom = worldData.GetRoom
//	olc.SaveFunc = worldData.SaveEntity
//
//	// Start editing
//	state := &builder.EditorState{
//	    Mode:     builder.EditorRoom,
//	    EditVnum: 3001,
//	    Data:     room,
//	}
//
//	// Process builder commands
//	olc.ProcessCommand(builder, state, "name The Town Square")
//	olc.ProcessCommand(builder, state, "sector city")
//	olc.ProcessCommand(builder, state, "north 3002")
//	olc.ProcessCommand(builder, state, "done")
package builder
