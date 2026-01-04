// Package ai implements NPC behavior and special functions for the ROT MUD.
//
// This package provides the AI system for non-player characters, including
// special behavior functions ported from special.c and default behaviors
// like wandering and scavenging.
//
// # Special Functions
//
// Special functions are assigned to NPCs to give them unique behaviors.
// They are called each mobile update tick (every ~1 second) and can
// perform actions like casting spells, attacking players, or picking up items.
//
// Available specials include:
//
// Dragon breath weapons:
//   - spec_breath_any, spec_breath_fire, spec_breath_frost
//   - spec_breath_acid, spec_breath_gas, spec_breath_lightning
//
// Casting mobs:
//   - spec_cast_adept: Helpful healer, buffs low-level players
//   - spec_cast_cleric: Offensive cleric spells in combat
//   - spec_cast_mage: Offensive mage spells in combat
//   - spec_cast_undead: Undead-themed offensive spells
//
// Law enforcement:
//   - spec_guard: Attacks evil players and protects innocents
//   - spec_executioner: Hunts down troublemakers
//   - spec_patrolman: Breaks up fights
//
// Hostile mobs:
//   - spec_thief: Steals gold from players
//   - spec_nasty: Backstabs and flees
//   - spec_poison: Bites and poisons in combat
//
// Utility mobs:
//   - spec_janitor: Picks up trash and cheap items
//   - spec_fido: Eats corpses
//   - spec_mayor: Follows a scripted patrol path
//
// # Default Behaviors
//
// NPCs without specials (or when specials don't fire) use default behaviors
// based on their act flags:
//
//   - ActScavenger: Pick up valuable items
//   - ActAggressive: Attack players on sight
//   - ActSentinel: Stay in one room (disables wandering)
//
// # AI System
//
// The [AISystem] manages all NPC behavior:
//
//   - Maintains a registry of special functions
//   - Provides context with game callbacks (combat, magic, movement)
//   - Processes all NPCs each mobile update
//
// # Usage Example
//
//	ai := ai.NewAISystem()
//	ai.Magic = magicSystem
//	ai.Output = sendToPlayer
//	ai.StartCombat = func(ch, victim *types.Character) {
//	    combat.SetFighting(ch, victim)
//	}
//
//	// In the game loop's mobile update:
//	ai.ProcessAllMobiles(gameLoop.Characters)
//
//	// Or process a single mob:
//	if ai.ProcessMobile(goblin) {
//	    // Mob took an action
//	}
package ai
