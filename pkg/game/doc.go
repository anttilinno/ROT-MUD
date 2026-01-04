// Package game implements the core game loop and command processing.
//
// This package is the heart of the MUD server, handling the main game loop,
// command dispatch, and character/object manipulation.
//
// # Game Loop
//
// The [GameLoop] manages pulse-based timing matching the original ROT timing:
//
//   - 4 pulses per second (250ms per pulse)
//   - Violence update every 3 pulses (combat rounds)
//   - Mobile AI update every 4 pulses
//   - Character tick every 60 pulses (regeneration, affect decay)
//   - Area reset every 120 pulses
//
// # Command System
//
// The [CommandRegistry] provides command lookup and execution:
//
//   - Prefix matching (e.g., "n" matches "north")
//   - Position requirements (can't walk while sleeping)
//   - Level requirements (immortal commands)
//   - Aliases (e.g., "'" for "say")
//
// Available commands include movement, information, object manipulation,
// communication, combat, magic, skills, groups, shops, and help.
//
// # Handler Functions
//
// Character and object manipulation functions:
//
//   - [CharToRoom], [CharFromRoom]: Move characters between rooms
//   - [ObjToChar], [ObjFromChar]: Transfer objects to/from inventory
//   - [ObjToRoom], [ObjFromRoom]: Place/remove objects in rooms
//   - [FindCharInRoom], [FindObjInRoom]: Target resolution
//
// # Act Function
//
// The [Act] function formats messages with token substitution:
//
//   - $n: actor's name
//   - $N: victim's name
//   - $e/$E: he/she/it pronouns
//   - $m/$M: him/her/it pronouns
//   - $s/$S: his/her/its pronouns
//
// # Usage Example
//
//	// Create and start the game loop
//	loop := game.NewGameLoop()
//	dispatcher := game.NewCommandDispatcher()
//	dispatcher.Output = sendToPlayer
//	dispatcher.GameLoop = loop
//
//	loop.OnCommand = dispatcher.Dispatch
//	loop.Start()
//
//	// Queue a player command
//	loop.QueueCommand(player, "north")
package game
