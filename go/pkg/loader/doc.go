// Package loader handles loading game data from TOML files.
//
// This package reads area definitions, mobile templates, object templates,
// and room data from the TOML-based data format used by the Go rewrite.
//
// # Directory Structure
//
// Game data is organized as:
//
//	data/
//	├── config.toml              # Server configuration
//	├── areas/
//	│   └── midgaard/
//	│       ├── area.toml        # Area metadata
//	│       ├── mobs/
//	│       │   └── temple.toml  # Mobile definitions
//	│       ├── objects/
//	│       │   └── weapons.toml # Object definitions
//	│       └── rooms/
//	│           └── temple.toml  # Room definitions
//	└── players/
//	    └── Gandalf.json         # Player saves
//
// # TOML Schemas
//
// Area metadata (area.toml):
//
//	id = "midgaard"
//	name = "The City of Midgaard"
//	reset_interval = 120
//	[vnum_range]
//	min = 3000
//	max = 3399
//
// Mobile definition:
//
//	[[mobiles]]
//	vnum = 3001
//	keywords = ["wizard", "old", "man"]
//	short_desc = "the old wizard"
//	level = 50
//	act_flags = ["npc", "sentinel"]
//	special = "spec_cast_mage"
//
// Room definition:
//
//	[[rooms]]
//	vnum = 3001
//	name = "The Temple"
//	sector = "inside"
//	room_flags = ["safe", "no_mob"]
//	description = """
//	A peaceful temple..."""
//	[[rooms.exits]]
//	direction = "north"
//	to_vnum = 3002
//
// Object definition:
//
//	[[objects]]
//	vnum = 3042
//	keywords = ["sword", "long"]
//	short_desc = "a long sword"
//	item_type = "weapon"
//	level = 10
//	[objects.weapon]
//	dice_number = 2
//	dice_size = 6
//	damage_type = "slash"
//
// # Loading Data
//
// Use [LoadArea] to load an entire area directory, which recursively
// loads all rooms, mobiles, and objects within it.
//
// # Usage Example
//
//	loader := loader.NewAreaLoader()
//	area, err := loader.LoadArea("data/areas/midgaard")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Access loaded data
//	room := area.Rooms[3001]
//	mob := loader.GetMobile(3001)
package loader
