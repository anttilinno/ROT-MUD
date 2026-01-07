// Package combat implements the combat system for the ROT MUD.
//
// This package handles all aspects of combat including attack resolution,
// damage calculation, death handling, and experience rewards. It is ported
// from the original fight.c.
//
// # Combat System
//
// The [CombatSystem] manages combat operations:
//
//   - [MultiHit]: Determines number of attacks and executes them
//   - [OneHit]: Single attack roll and damage calculation
//   - [Damage]: Applies damage with immunity/resistance checks
//
// # Attack Resolution
//
// Attack rolls use the classic THAC0 (To Hit Armor Class 0) system:
//
//  1. Calculate attacker's THAC0 based on level and class
//  2. Apply hitroll bonus from stats and equipment
//  3. Roll d20 and compare to (THAC0 - victim's AC)
//  4. If hit, calculate damage from weapon dice + damroll
//
// # Defensive Skills
//
// The [CheckDefenses] function checks for:
//
//   - Parry (requires wielded weapon)
//   - Dodge (dexterity-based)
//   - Shield block (requires equipped shield)
//
// # Damage Modifiers
//
// Damage is modified by:
//
//   - Immunity: No damage
//   - Resistance: 2/3 damage
//   - Vulnerability: 3/2 damage
//   - Sanctuary: Half damage
//   - High damage caps at 35/80
//
// # Death Handling
//
// When a character dies:
//
//   - Combat is stopped for all participants
//   - Corpse is created with victim's inventory
//   - Experience is awarded to killer (for NPC kills)
//   - NPCs are removed; players respawn
//
// # Utility Functions
//
// Helper functions include:
//
//   - [Dice]: Roll NdS dice (e.g., 2d6)
//   - [NumberRange]: Random number in range
//   - [NumberPercent]: Random 1-100
//   - [SetFighting], [StopFighting]: Combat state management
//   - [IsSafe]: Check if combat is allowed
//
// # Usage Example
//
//	cs := combat.NewCombatSystem()
//	cs.Output = sendToPlayer
//
//	// Start combat
//	combat.SetFighting(attacker, victim)
//
//	// Process an attack round
//	cs.MultiHit(attacker, victim)
//
//	// Check if victim died
//	if victim.Position == types.PosDead {
//	    // Handle death
//	}
package combat
