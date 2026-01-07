// Package magic implements the spell and affect system for the ROT MUD.
//
// This package handles spell casting, target resolution, mana costs,
// and the affect system for buffs and debuffs. It is ported from
// magic.c and magic2.c.
//
// # Spell System
//
// Spells are defined with:
//
//   - Name and mana cost
//   - Target type (self, defensive, offensive, object)
//   - Minimum position to cast
//   - Class level requirements
//   - Spell function that applies the effect
//
// # Target Types
//
// Spells can target:
//
//   - [TargetIgnore]: No target needed (area spells)
//   - [TargetCharSelf]: Caster only
//   - [TargetCharDefense]: Friendly target (defaults to self)
//   - [TargetCharOffense]: Enemy target (defaults to fighting target)
//   - [TargetObjInv]: Object in inventory
//
// # Affect System
//
// Affects modify character stats and flags:
//
//   - Duration in game ticks
//   - Stat modifier (e.g., +2 strength)
//   - Bit vector for affect flags (e.g., AffInvisible)
//
// The [AffectTick] function processes affect decay each game tick,
// removing expired affects and restoring modified stats.
//
// # Spell Categories
//
// Default spells include:
//
// Damage spells:
//   - magic missile, chill touch, burning hands
//   - lightning bolt, fireball, acid blast
//
// Healing spells:
//   - cure light, cure serious, cure critical, heal
//
// Buff spells:
//   - armor, bless, giant strength, haste, sanctuary
//
// Debuff spells:
//   - blindness, curse, poison, weaken, plague
//
// # Usage Example
//
//	magic := magic.NewMagicSystem()
//	magic.Output = sendToPlayer
//
//	// Cast a spell
//	findTarget := func(ch *types.Character, name string, offensive bool) interface{} {
//	    return game.FindCharInRoom(ch, name)
//	}
//
//	success := magic.Cast(caster, "fireball", "goblin", findTarget)
//
//	// Process affect decay
//	magic.ProcessAffectTick(allCharacters)
package magic
