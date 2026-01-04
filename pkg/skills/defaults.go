package skills

import "rotmud/pkg/types"

// Class indices (matching types.Class*)
const (
	Mage    = types.ClassMage
	Cleric  = types.ClassCleric
	Thief   = types.ClassThief
	Warrior = types.ClassWarrior
)

// DefaultSkills returns a registry populated with default skills
func DefaultSkills() *SkillRegistry {
	r := NewSkillRegistry()

	// === Combat Skills ===

	// Basic attacks
	r.Register(NewSkill("second attack").
		SetClassLevel(Mage, 30, 5).
		SetClassLevel(Cleric, 24, 5).
		SetClassLevel(Thief, 12, 5).
		SetClassLevel(Warrior, 5, 3))

	r.Register(NewSkill("third attack").
		SetClassLevel(Mage, 0, 0). // Can't learn
		SetClassLevel(Cleric, 0, 0).
		SetClassLevel(Thief, 0, 0).
		SetClassLevel(Warrior, 25, 4))

	r.Register(NewSkill("fourth attack").
		SetClassLevel(Mage, 0, 0).
		SetClassLevel(Cleric, 0, 0).
		SetClassLevel(Thief, 0, 0).
		SetClassLevel(Warrior, 35, 5))

	r.Register(NewSkill("fifth attack").
		SetClassLevel(Mage, 0, 0).
		SetClassLevel(Cleric, 0, 0).
		SetClassLevel(Thief, 0, 0).
		SetClassLevel(Warrior, 45, 6))

	r.Register(NewSkill("dual wield").
		SetClassLevel(Mage, 0, 0).
		SetClassLevel(Cleric, 0, 0).
		SetClassLevel(Thief, 15, 4).
		SetClassLevel(Warrior, 20, 4))

	// Defensive skills
	r.Register(NewSkill("dodge").
		SetClassLevel(Mage, 20, 8).
		SetClassLevel(Cleric, 22, 6).
		SetClassLevel(Thief, 1, 4).
		SetClassLevel(Warrior, 13, 6))

	r.Register(NewSkill("parry").
		SetClassLevel(Mage, 22, 8).
		SetClassLevel(Cleric, 20, 6).
		SetClassLevel(Thief, 13, 6).
		SetClassLevel(Warrior, 1, 4))

	r.Register(NewSkill("shield block").
		SetClassLevel(Mage, 0, 0).
		SetClassLevel(Cleric, 10, 5).
		SetClassLevel(Thief, 0, 0).
		SetClassLevel(Warrior, 1, 3))

	r.Register(NewSkill("enhanced damage").
		SetClassLevel(Mage, 0, 0).
		SetClassLevel(Cleric, 30, 6).
		SetClassLevel(Thief, 25, 5).
		SetClassLevel(Warrior, 1, 3))

	r.Register(NewSkill("grip").
		SetClassLevel(Mage, 0, 0).
		SetClassLevel(Cleric, 0, 0).
		SetClassLevel(Thief, 0, 0).
		SetClassLevel(Warrior, 15, 4))

	// Offensive skills
	r.Register(NewSkill("kick").
		SetClassLevel(Mage, 0, 0).
		SetClassLevel(Cleric, 12, 4).
		SetClassLevel(Thief, 14, 6).
		SetClassLevel(Warrior, 8, 4))

	r.Register(NewSkill("bash").
		SetClassLevel(Mage, 0, 0).
		SetClassLevel(Cleric, 0, 0).
		SetClassLevel(Thief, 0, 0).
		SetClassLevel(Warrior, 1, 3))

	r.Register(NewSkill("trip").
		SetClassLevel(Mage, 0, 0).
		SetClassLevel(Cleric, 0, 0).
		SetClassLevel(Thief, 1, 4).
		SetClassLevel(Warrior, 15, 6))

	r.Register(NewSkill("dirt kicking").
		SetClassLevel(Mage, 0, 0).
		SetClassLevel(Cleric, 0, 0).
		SetClassLevel(Thief, 3, 4).
		SetClassLevel(Warrior, 3, 4))

	r.Register(NewSkill("disarm").
		SetClassLevel(Mage, 0, 0).
		SetClassLevel(Cleric, 0, 0).
		SetClassLevel(Thief, 12, 5).
		SetClassLevel(Warrior, 11, 4))

	r.Register(NewSkill("gouge").
		SetClassLevel(Mage, 0, 0).
		SetClassLevel(Cleric, 0, 0).
		SetClassLevel(Thief, 10, 4).
		SetClassLevel(Warrior, 0, 0))

	r.Register(NewSkill("stun").
		SetClassLevel(Mage, 0, 0).
		SetClassLevel(Cleric, 0, 0).
		SetClassLevel(Thief, 0, 0).
		SetClassLevel(Warrior, 20, 5))

	r.Register(NewSkill("backstab").
		SetClassLevel(Mage, 0, 0).
		SetClassLevel(Cleric, 0, 0).
		SetClassLevel(Thief, 1, 5).
		SetClassLevel(Warrior, 0, 0))

	r.Register(NewSkill("circle").
		SetClassLevel(Mage, 0, 0).
		SetClassLevel(Cleric, 0, 0).
		SetClassLevel(Thief, 15, 4).
		SetClassLevel(Warrior, 0, 0))

	r.Register(NewSkill("berserk").
		SetClassLevel(Mage, 0, 0).
		SetClassLevel(Cleric, 0, 0).
		SetClassLevel(Thief, 0, 0).
		SetClassLevel(Warrior, 18, 5))

	r.Register(NewSkill("rescue").
		SetClassLevel(Mage, 0, 0).
		SetClassLevel(Cleric, 12, 5).
		SetClassLevel(Thief, 0, 0).
		SetClassLevel(Warrior, 1, 4))

	// Hand to hand
	r.Register(NewSkill("hand to hand").
		SetClassLevel(Mage, 25, 8).
		SetClassLevel(Cleric, 10, 5).
		SetClassLevel(Thief, 15, 6).
		SetClassLevel(Warrior, 6, 4))

	// === Thief Skills ===

	r.Register(NewSkill("sneak").
		SetClassLevel(Mage, 0, 0).
		SetClassLevel(Cleric, 0, 0).
		SetClassLevel(Thief, 4, 4).
		SetClassLevel(Warrior, 0, 0))

	r.Register(NewSkill("hide").
		SetClassLevel(Mage, 0, 0).
		SetClassLevel(Cleric, 0, 0).
		SetClassLevel(Thief, 1, 4).
		SetClassLevel(Warrior, 0, 0))

	r.Register(NewSkill("steal").
		SetClassLevel(Mage, 0, 0).
		SetClassLevel(Cleric, 0, 0).
		SetClassLevel(Thief, 5, 4).
		SetClassLevel(Warrior, 0, 0))

	r.Register(NewSkill("pick lock").
		SetClassLevel(Mage, 0, 0).
		SetClassLevel(Cleric, 0, 0).
		SetClassLevel(Thief, 7, 4).
		SetClassLevel(Warrior, 0, 0))

	r.Register(NewSkill("peek").
		SetClassLevel(Mage, 0, 0).
		SetClassLevel(Cleric, 0, 0).
		SetClassLevel(Thief, 1, 4).
		SetClassLevel(Warrior, 0, 0))

	r.Register(NewSkill("envenom").
		SetClassLevel(Mage, 0, 0).
		SetClassLevel(Cleric, 0, 0).
		SetClassLevel(Thief, 10, 4).
		SetClassLevel(Warrior, 0, 0))

	r.Register(NewSkill("track").
		SetClassLevel(Mage, 0, 0).
		SetClassLevel(Cleric, 0, 0).
		SetClassLevel(Thief, 20, 5).
		SetClassLevel(Warrior, 0, 0))

	// === Vampire Skills ===

	r.Register(NewSkill("feed").
		SetClassLevel(Mage, 0, 0).
		SetClassLevel(Cleric, 0, 0).
		SetClassLevel(Thief, 0, 0).
		SetClassLevel(Warrior, 0, 0))
	// Note: Vampire class index 6 would need to be added

	// === Weapon Skills ===

	r.Register(NewSkill("sword").
		SetClassLevel(Mage, 20, 6).
		SetClassLevel(Cleric, 15, 5).
		SetClassLevel(Thief, 6, 4).
		SetClassLevel(Warrior, 1, 3))

	r.Register(NewSkill("dagger").
		SetClassLevel(Mage, 1, 4).
		SetClassLevel(Cleric, 10, 5).
		SetClassLevel(Thief, 1, 3).
		SetClassLevel(Warrior, 5, 4))

	r.Register(NewSkill("spear").
		SetClassLevel(Mage, 0, 0).
		SetClassLevel(Cleric, 15, 5).
		SetClassLevel(Thief, 0, 0).
		SetClassLevel(Warrior, 5, 4))

	r.Register(NewSkill("mace").
		SetClassLevel(Mage, 0, 0).
		SetClassLevel(Cleric, 1, 3).
		SetClassLevel(Thief, 15, 6).
		SetClassLevel(Warrior, 6, 4))

	r.Register(NewSkill("axe").
		SetClassLevel(Mage, 0, 0).
		SetClassLevel(Cleric, 0, 0).
		SetClassLevel(Thief, 0, 0).
		SetClassLevel(Warrior, 1, 3))

	r.Register(NewSkill("flail").
		SetClassLevel(Mage, 0, 0).
		SetClassLevel(Cleric, 1, 3).
		SetClassLevel(Thief, 0, 0).
		SetClassLevel(Warrior, 10, 4))

	r.Register(NewSkill("whip").
		SetClassLevel(Mage, 0, 0).
		SetClassLevel(Cleric, 0, 0).
		SetClassLevel(Thief, 10, 5).
		SetClassLevel(Warrior, 0, 0))

	r.Register(NewSkill("polearm").
		SetClassLevel(Mage, 0, 0).
		SetClassLevel(Cleric, 0, 0).
		SetClassLevel(Thief, 0, 0).
		SetClassLevel(Warrior, 10, 4))

	// === Utility Skills ===

	r.Register(NewSkill("meditation").
		SetClassLevel(Mage, 6, 5).
		SetClassLevel(Cleric, 6, 5).
		SetClassLevel(Thief, 0, 0).
		SetClassLevel(Warrior, 0, 0))

	r.Register(NewSkill("fast healing").
		SetClassLevel(Mage, 15, 8).
		SetClassLevel(Cleric, 9, 5).
		SetClassLevel(Thief, 16, 6).
		SetClassLevel(Warrior, 6, 4))

	r.Register(NewSkill("haggle").
		SetClassLevel(Mage, 15, 5).
		SetClassLevel(Cleric, 15, 5).
		SetClassLevel(Thief, 1, 3).
		SetClassLevel(Warrior, 18, 8))

	r.Register(NewSkill("lore").
		SetClassLevel(Mage, 10, 5).
		SetClassLevel(Cleric, 5, 4).
		SetClassLevel(Thief, 6, 6).
		SetClassLevel(Warrior, 15, 8))

	r.Register(NewSkill("recall").
		SetClassLevel(Mage, 1, 2).
		SetClassLevel(Cleric, 1, 2).
		SetClassLevel(Thief, 1, 2).
		SetClassLevel(Warrior, 1, 2))

	// === Magic Item Use ===

	r.Register(NewSkill("scrolls").
		SetClassLevel(Mage, 1, 2).
		SetClassLevel(Cleric, 1, 2).
		SetClassLevel(Thief, 10, 5).
		SetClassLevel(Warrior, 20, 8))

	r.Register(NewSkill("staves").
		SetClassLevel(Mage, 1, 2).
		SetClassLevel(Cleric, 1, 2).
		SetClassLevel(Thief, 15, 6).
		SetClassLevel(Warrior, 25, 8))

	r.Register(NewSkill("wands").
		SetClassLevel(Mage, 1, 2).
		SetClassLevel(Cleric, 1, 2).
		SetClassLevel(Thief, 12, 5).
		SetClassLevel(Warrior, 25, 8))

	return r
}
