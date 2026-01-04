// Package help implements the in-game help system for the ROT MUD.
//
// This package provides help topic lookup and display, with topics
// loaded from TOML files.
//
// # Help File Format
//
// Help entries are stored in TOML files in data/help/:
//
//	[[help]]
//	keywords = ["north", "south", "east", "west", "movement"]
//	level = 0
//	syntax = "north | n"
//	description = """
//	Movement commands allow you to travel between rooms.
//	Use the direction name or its first letter."""
//	see_also = ["look", "exits"]
//
// # Help Entry Properties
//
//   - keywords: Words that match this help topic
//   - level: Minimum level to view (0 = everyone)
//   - syntax: Command syntax (optional)
//   - description: The help text
//   - see_also: Related topics (optional)
//
// # Topic Lookup
//
// The system supports:
//
//   - Exact match: "help north"
//   - Prefix match: "help nor" matches "north"
//   - Multiple matches: Shows disambiguation list
//
// # Help Commands
//
// Players use:
//
//   - help: Shows the summary/index
//   - help <topic>: Shows specific help
//   - commands: Lists available commands by category
//
// # Loading Help
//
// Help files are loaded from a directory:
//
//	data/help/
//	├── commands.toml    # Command help
//	├── skills.toml      # Skill help
//	├── spells.toml      # Spell help
//	└── summary.toml     # Default help / index
//
// # Usage Example
//
//	help := help.NewSystem()
//	err := help.LoadDir("data/help")
//
//	// Find a topic
//	entry := help.Find("north")
//	if entry != nil {
//	    output := entry.Format()
//	    sendToPlayer(player, output)
//	}
//
//	// Programmatic registration
//	help.Register(&help.Entry{
//	    Keywords:    []string{"test"},
//	    Description: "This is a test.",
//	})
package help
