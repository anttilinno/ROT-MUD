// Package persistence handles player data saving and loading.
//
// This package manages the serialization and deserialization of player
// characters to/from JSON files. It is the Go replacement for save.c.
//
// # Player Save Format
//
// Player data is stored as JSON in data/players/<name>.json:
//
//	{
//	    "name": "Gandalf",
//	    "password": "$2a$...",
//	    "level": 50,
//	    "class": 0,
//	    "race": 0,
//	    "hp": 500,
//	    "max_hp": 500,
//	    "mana": 1000,
//	    "max_mana": 1000,
//	    "exp": 1500000,
//	    "gold": 5000,
//	    "stats": [18, 20, 18, 15, 16],
//	    "skills": {
//	        "fireball": 95,
//	        "meditation": 80
//	    },
//	    "equipment": {
//	        "wield": {"vnum": 3042, "enchant": 2}
//	    },
//	    "inventory": [
//	        {"vnum": 3050, "count": 5}
//	    ],
//	    "affects": [
//	        {"type": "sanctuary", "duration": 10, "modifier": 0}
//	    ]
//	}
//
// # Password Security
//
// Passwords are hashed using bcrypt before storage. The original
// plaintext password is never saved.
//
// # Automatic Saving
//
// The system supports:
//
//   - Save on quit
//   - Periodic auto-save
//   - Save on level gain
//   - Save on shutdown
//
// # Object Persistence
//
// Player equipment and inventory are saved by vnum reference.
// On load, objects are recreated from their templates with any
// enchantments or modifications preserved.
//
// # Usage Example
//
//	persist := persistence.NewPlayerPersistence("data/players")
//
//	// Save a player
//	err := persist.Save(player)
//
//	// Load a player
//	player, err := persist.Load("Gandalf")
//
//	// Check if player exists
//	exists := persist.Exists("Gandalf")
//
//	// Delete a player
//	err := persist.Delete("Gandalf")
package persistence
