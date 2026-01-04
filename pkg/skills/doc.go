// Package skills implements the skill and training system for the ROT MUD.
//
// This package handles skill definitions, proficiency tracking, skill
// improvement through use, and the training system.
//
// # Skill Types
//
// Skills are categorized as:
//
//   - [TypeSkill]: Active abilities (dodge, parry, backstab)
//   - [TypeSpell]: Magical abilities (handled by magic package)
//   - [TypeWeapon]: Weapon proficiencies
//
// # Skill Definition
//
// Each skill has:
//
//   - Name and type
//   - Class level requirements (when each class can learn)
//   - Class ratings (how hard to improve, affects practice gains)
//
// # Learning and Improvement
//
// Skills are learned through:
//
//   - Practice sessions with trainers (up to 75%)
//   - Using the skill in gameplay (75% to 100%)
//
// The [CheckImprove] function handles skill improvement on use:
//
//  1. Check if character knows the skill
//  2. Calculate improvement chance based on intelligence and skill rating
//  3. On success, increase proficiency and award XP
//
// # NPC Skills
//
// NPCs receive automatic skill levels based on:
//
//   - Level (higher level = better skills)
//   - Act flags (ActWarrior gets parry, ActThief gets dodge)
//
// # Default Skills
//
// The package provides default combat skills:
//
// Defensive:
//   - dodge, parry, shield block
//
// Offensive:
//   - second attack, third attack
//   - kick, bash, backstab
//   - hand to hand
//
// Utility:
//   - sneak, hide
//   - haggle
//
// # Usage Example
//
//	skills := skills.NewSkillSystem()
//	skills.Output = sendToPlayer
//
//	// Get effective skill level
//	dodgeLevel := skills.GetSkill(player, "dodge")
//
//	// Check for improvement after using a skill
//	skills.CheckImprove(player, "dodge", true, 2)  // success, multiplier 2
//
//	// Learn from a trainer
//	skills.LearnSkill(player, "parry", 15)  // +15% proficiency
//
//	// Get learned percentage
//	pct := skills.GetLearnedPercent(player, "parry")
package skills
