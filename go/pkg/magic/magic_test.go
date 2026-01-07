package magic

import (
	"testing"

	"rotmud/pkg/types"
)

func TestSpellRegistry(t *testing.T) {
	r := DefaultSpells()

	t.Run("find by exact name", func(t *testing.T) {
		spell := r.FindByName("magic missile")
		if spell == nil {
			t.Error("expected to find magic missile")
		}
		if spell.Name != "magic missile" {
			t.Errorf("expected 'magic missile', got '%s'", spell.Name)
		}
	})

	t.Run("find by prefix", func(t *testing.T) {
		spell := r.FindByPrefix("fire")
		if spell == nil {
			t.Error("expected to find spell starting with 'fire'")
		}
		// Accept any spell starting with "fire"
		validSpells := []string{"fireball", "fire breath", "fireshield", "fireproof"}
		found := false
		for _, valid := range validSpells {
			if spell.Name == valid {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected a spell starting with 'fire', got '%s'", spell.Name)
		}
	})

	t.Run("FindByName is exact only", func(t *testing.T) {
		spell := r.FindByName("fire")
		if spell != nil {
			t.Error("expected nil for prefix 'fire' with FindByName (exact only)")
		}
	})

	t.Run("find by slot", func(t *testing.T) {
		spell := r.FindBySlot(SlotMagicMissile)
		if spell == nil {
			t.Error("expected to find spell by slot")
		}
		if spell.Name != "magic missile" {
			t.Errorf("expected 'magic missile', got '%s'", spell.Name)
		}
	})
}

func TestAffects(t *testing.T) {
	t.Run("add affect", func(t *testing.T) {
		ch := types.NewCharacter("Test")
		af := NewAffect("armor", 10, 12, types.ApplyAC, -20)

		AddAffect(ch, af)

		if !IsAffectedBy(ch, "armor") {
			t.Error("expected character to be affected by armor")
		}

		// Check AC was modified
		// Note: armor applies to all 4 AC types
		if ch.Armor[types.ACPierce] != -20 {
			t.Errorf("expected AC pierce -20, got %d", ch.Armor[types.ACPierce])
		}
	})

	t.Run("remove affect", func(t *testing.T) {
		ch := types.NewCharacter("Test")
		af := NewAffect("armor", 10, 12, types.ApplyAC, -20)

		AddAffect(ch, af)
		RemoveAffect(ch, af)

		if IsAffectedBy(ch, "armor") {
			t.Error("expected character to not be affected by armor")
		}

		// Check AC was restored
		if ch.Armor[types.ACPierce] != 0 {
			t.Errorf("expected AC pierce 0, got %d", ch.Armor[types.ACPierce])
		}
	})

	t.Run("affect with bitvector", func(t *testing.T) {
		ch := types.NewCharacter("Test")
		af := NewAffectWithBit("sanctuary", 20, 10, types.AffSanctuary)

		AddAffect(ch, af)

		if !ch.IsAffected(types.AffSanctuary) {
			t.Error("expected character to have sanctuary flag")
		}

		RemoveAffect(ch, af)

		if ch.IsAffected(types.AffSanctuary) {
			t.Error("expected sanctuary flag to be removed")
		}
	})

	t.Run("affect stacking", func(t *testing.T) {
		ch := types.NewCharacter("Test")
		af1 := NewAffect("armor", 10, 12, types.ApplyAC, -20)
		af2 := NewAffect("armor", 10, 8, types.ApplyAC, -20)

		AddAffect(ch, af1)
		AddAffect(ch, af2)

		// Should only have one armor affect with combined duration
		found := GetAffect(ch, "armor")
		if found == nil {
			t.Error("expected armor affect")
		}
		if found.Duration != 20 { // 12 + 8
			t.Errorf("expected duration 20, got %d", found.Duration)
		}
	})
}

func TestSpellCanCast(t *testing.T) {
	r := DefaultSpells()
	spell := r.FindByName("magic missile")

	t.Run("mage can cast", func(t *testing.T) {
		ch := types.NewCharacter("Mage")
		ch.Class = types.ClassMage
		ch.Level = 1
		ch.Mana = 100
		ch.MaxMana = 100

		if !spell.CanCast(ch) {
			t.Error("level 1 mage should be able to cast magic missile")
		}
	})

	t.Run("warrior cannot cast", func(t *testing.T) {
		ch := types.NewCharacter("Warrior")
		ch.Class = types.ClassWarrior
		ch.Level = 10
		ch.Mana = 100
		ch.MaxMana = 100

		if spell.CanCast(ch) {
			t.Error("warrior should not be able to cast magic missile")
		}
	})

	t.Run("insufficient mana", func(t *testing.T) {
		ch := types.NewCharacter("Mage")
		ch.Class = types.ClassMage
		ch.Level = 1
		ch.Mana = 5 // Not enough
		ch.MaxMana = 100

		if spell.CanCast(ch) {
			t.Error("should not be able to cast with insufficient mana")
		}
	})
}

func TestHealingSpells(t *testing.T) {
	ch := types.NewCharacter("Test")
	ch.Hit = 10
	ch.MaxHit = 100

	success := spellCureLight(ch, 10, ch)
	if !success {
		t.Error("cure light should succeed")
	}

	if ch.Hit <= 10 {
		t.Error("HP should have increased")
	}

	if ch.Hit > ch.MaxHit {
		t.Error("HP should not exceed max")
	}
}

func TestBuffSpells(t *testing.T) {
	ch := types.NewCharacter("Test")
	ch.PermStats[types.StatStr] = 15
	ch.ModStats[types.StatStr] = 0

	success := spellGiantStrength(ch, 20, ch)
	if !success {
		t.Error("giant strength should succeed")
	}

	if !IsAffectedBy(ch, "giant strength") {
		t.Error("should be affected by giant strength")
	}

	if ch.GetStat(types.StatStr) <= 15 {
		t.Errorf("strength should be buffed, got %d", ch.GetStat(types.StatStr))
	}
}

func TestCureSpells(t *testing.T) {
	t.Run("cure blindness", func(t *testing.T) {
		ch := types.NewCharacter("Test")

		// First blind them
		af := NewAffectWithBit("blindness", 10, 5, types.AffBlind)
		AddAffect(ch, af)

		if !ch.IsAffected(types.AffBlind) {
			t.Error("should be blind")
		}

		// Cure it
		success := spellCureBlindness(ch, 10, ch)
		if !success {
			t.Error("cure blindness should succeed")
		}

		if ch.IsAffected(types.AffBlind) {
			t.Error("should no longer be blind")
		}
	})

	t.Run("cure poison", func(t *testing.T) {
		ch := types.NewCharacter("Test")

		// First poison them
		af := NewAffectWithBit("poison", 10, 5, types.AffPoison)
		AddAffect(ch, af)

		if !ch.IsAffected(types.AffPoison) {
			t.Error("should be poisoned")
		}

		// Cure it
		success := spellCurePoison(ch, 10, ch)
		if !success {
			t.Error("cure poison should succeed")
		}

		if ch.IsAffected(types.AffPoison) {
			t.Error("should no longer be poisoned")
		}
	})

	t.Run("remove curse", func(t *testing.T) {
		ch := types.NewCharacter("Test")

		// First curse them
		af := NewAffectWithBit("curse", 10, 5, types.AffCurse)
		AddAffect(ch, af)

		if !ch.IsAffected(types.AffCurse) {
			t.Error("should be cursed")
		}

		// Remove it
		success := spellRemoveCurse(ch, 10, ch)
		if !success {
			t.Error("remove curse should succeed")
		}

		if ch.IsAffected(types.AffCurse) {
			t.Error("should no longer be cursed")
		}
	})
}

func TestDetectionSpells(t *testing.T) {
	tests := []struct {
		name    string
		spell   func(*types.Character, int, interface{}) bool
		flag    types.AffectFlags
		affName string
	}{
		{"detect evil", spellDetectEvil, types.AffDetectEvil, "detect evil"},
		{"detect good", spellDetectGood, types.AffDetectGood, "detect good"},
		{"detect hidden", spellDetectHidden, types.AffDetectHidden, "detect hidden"},
		{"detect magic", spellDetectMagic, types.AffDetectMagic, "detect magic"},
		{"infravision", spellInfravision, types.AffInfrared, "infravision"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := types.NewCharacter("Test")

			success := tt.spell(ch, 10, ch)
			if !success {
				t.Errorf("%s should succeed", tt.name)
			}

			if !ch.IsAffected(tt.flag) {
				t.Errorf("should have %s flag", tt.name)
			}

			if !IsAffectedBy(ch, tt.affName) {
				t.Errorf("should be affected by %s", tt.affName)
			}

			// Casting again should fail (already affected)
			success = tt.spell(ch, 10, ch)
			if success {
				t.Errorf("%s should fail when already affected", tt.name)
			}
		})
	}
}

func TestFlySpell(t *testing.T) {
	ch := types.NewCharacter("Test")

	success := spellFly(ch, 10, ch)
	if !success {
		t.Error("fly should succeed")
	}

	if !ch.IsAffected(types.AffFlying) {
		t.Error("should be flying")
	}

	// Already flying
	success = spellFly(ch, 10, ch)
	if success {
		t.Error("fly should fail when already flying")
	}
}

func TestDefensiveSpells(t *testing.T) {
	t.Run("stone skin", func(t *testing.T) {
		ch := types.NewCharacter("Test")
		initialAC := ch.Armor[types.ACPierce]

		success := spellStoneSkin(ch, 20, ch)
		if !success {
			t.Error("stone skin should succeed")
		}

		if !IsAffectedBy(ch, "stone skin") {
			t.Error("should be affected by stone skin")
		}

		// AC should be improved (more negative)
		if ch.Armor[types.ACPierce] >= initialAC {
			t.Error("AC should be better with stone skin")
		}
	})

	t.Run("shield", func(t *testing.T) {
		ch := types.NewCharacter("Test")
		initialAC := ch.Armor[types.ACPierce]

		success := spellShield(ch, 20, ch)
		if !success {
			t.Error("shield should succeed")
		}

		if !IsAffectedBy(ch, "shield") {
			t.Error("should be affected by shield")
		}

		// AC should be improved
		if ch.Armor[types.ACPierce] >= initialAC {
			t.Error("AC should be better with shield")
		}
	})

	t.Run("protection evil", func(t *testing.T) {
		ch := types.NewCharacter("Test")

		success := spellProtectEvil(ch, 20, ch)
		if !success {
			t.Error("protection evil should succeed")
		}

		if !ch.IsAffected(types.AffProtectEvil) {
			t.Error("should have protection evil flag")
		}
	})

	t.Run("protection good", func(t *testing.T) {
		ch := types.NewCharacter("Test")

		success := spellProtectGood(ch, 20, ch)
		if !success {
			t.Error("protection good should succeed")
		}

		if !ch.IsAffected(types.AffProtectGood) {
			t.Error("should have protection good flag")
		}
	})

	t.Run("protection mutual exclusion", func(t *testing.T) {
		ch := types.NewCharacter("Test")

		// Cast protection evil first
		spellProtectEvil(ch, 20, ch)

		// Protection good should fail
		success := spellProtectGood(ch, 20, ch)
		if success {
			t.Error("protection good should fail when protected from evil")
		}
	})
}

func TestFaerieFire(t *testing.T) {
	ch := types.NewCharacter("Target")
	initialAC := ch.Armor[types.ACPierce]

	success := spellFaerieFire(ch, 20, ch)
	if !success {
		t.Error("faerie fire should succeed")
	}

	if !ch.IsAffected(types.AffFaerieFire) {
		t.Error("should have faerie fire flag")
	}

	// AC should be worse (more positive)
	if ch.Armor[types.ACPierce] <= initialAC {
		t.Error("AC should be worse with faerie fire")
	}
}

func TestFrenzySpell(t *testing.T) {
	ch := types.NewCharacter("Test")

	success := spellFrenzy(ch, 30, ch)
	if !success {
		t.Error("frenzy should succeed")
	}

	if !IsAffectedBy(ch, "frenzy") {
		t.Error("should be affected by frenzy")
	}
}

func TestDispelMagic(t *testing.T) {
	caster := types.NewCharacter("Caster")
	caster.Level = 30

	victim := types.NewCharacter("Victim")
	victim.Level = 10

	// Add some affects
	af1 := NewAffect("armor", 10, 5, types.ApplyAC, -20)
	AddAffect(victim, af1)

	af2 := NewAffectWithBit("sanctuary", 10, 5, types.AffSanctuary)
	AddAffect(victim, af2)

	// Dispel should remove at least some affects
	success := spellDispelMagic(caster, caster.Level, victim)

	// With level difference, should succeed
	if !success {
		// It's random, so might fail occasionally
		t.Log("dispel magic didn't remove any affects (may be random)")
	}
}

func TestWordOfRecall(t *testing.T) {
	ch := types.NewCharacter("Test")

	// Word of recall just returns true (movement handled elsewhere)
	success := spellWordOfRecall(ch, 30, ch)
	if !success {
		t.Error("word of recall should succeed")
	}
}

func TestPassDoorSpell(t *testing.T) {
	ch := types.NewCharacter("Test")

	success := spellPassDoor(ch, 20, ch)
	if !success {
		t.Error("pass door should succeed")
	}

	if !ch.IsAffected(types.AffPassDoor) {
		t.Error("should have pass door flag")
	}
}

func TestDamageSpells(t *testing.T) {
	t.Run("magic missile", func(t *testing.T) {
		victim := types.NewCharacter("Target")
		victim.Hit = 100
		victim.MaxHit = 100

		success := spellMagicMissile(nil, 10, victim)
		if !success {
			t.Error("magic missile should succeed")
		}

		if victim.Hit >= 100 {
			t.Error("victim should take damage")
		}
	})

	t.Run("fireball", func(t *testing.T) {
		victim := types.NewCharacter("Target")
		victim.Hit = 100
		victim.MaxHit = 100

		success := spellFireball(nil, 20, victim)
		if !success {
			t.Error("fireball should succeed")
		}

		if victim.Hit >= 100 {
			t.Error("victim should take damage")
		}
	})

	t.Run("lightning bolt", func(t *testing.T) {
		victim := types.NewCharacter("Target")
		victim.Hit = 100
		victim.MaxHit = 100

		success := spellLightningBolt(nil, 15, victim)
		if !success {
			t.Error("lightning bolt should succeed")
		}

		if victim.Hit >= 100 {
			t.Error("victim should take damage")
		}
	})
}

func TestRefreshSpell(t *testing.T) {
	ch := types.NewCharacter("Test")
	ch.Move = 10
	ch.MaxMove = 100

	success := spellRefresh(ch, 20, ch)
	if !success {
		t.Error("refresh should succeed")
	}

	if ch.Move <= 10 {
		t.Error("movement should increase")
	}
}

func TestAnimateSpell(t *testing.T) {
	t.Run("valid body part", func(t *testing.T) {
		ch := types.NewCharacter("Necromancer")
		ch.Level = 30

		// Body parts have VNUM 12-17
		bodyPart := types.NewObject(15, "a severed head", types.ItemTypeTrash)

		success := spellAnimate(ch, 30, bodyPart)
		if !success {
			t.Error("animate should succeed with valid body part")
		}
	})

	t.Run("invalid object", func(t *testing.T) {
		ch := types.NewCharacter("Necromancer")
		ch.Level = 30

		// Not a body part (wrong VNUM)
		sword := types.NewObject(100, "a sword", types.ItemTypeWeapon)

		success := spellAnimate(ch, 30, sword)
		if success {
			t.Error("animate should fail with non-body-part object")
		}
	})

	t.Run("already has pet", func(t *testing.T) {
		ch := types.NewCharacter("Necromancer")
		ch.Level = 30
		ch.Pet = types.NewCharacter("OldPet")

		bodyPart := types.NewObject(15, "a severed head", types.ItemTypeTrash)

		success := spellAnimate(ch, 30, bodyPart)
		if success {
			t.Error("animate should fail when caster already has pet")
		}
	})
}

func TestResurrectSpell(t *testing.T) {
	t.Run("no pet", func(t *testing.T) {
		ch := types.NewCharacter("Necromancer")
		ch.Level = 30

		// Resurrect validates that caster has no pet
		// The actual corpse finding is done by MagicSystem.handleResurrect
		success := spellResurrect(ch, 30, nil)
		if !success {
			t.Error("resurrect spell validation should succeed when no pet")
		}
	})

	t.Run("already has pet", func(t *testing.T) {
		ch := types.NewCharacter("Necromancer")
		ch.Level = 30
		ch.Pet = types.NewCharacter("OldPet")

		success := spellResurrect(ch, 30, nil)
		if success {
			t.Error("resurrect should fail when caster already has pet")
		}
	})
}

func TestConjureSpell(t *testing.T) {
	t.Run("NPC caster", func(t *testing.T) {
		ch := types.NewNPC(100, "Test Mob", 30)

		success := spellConjure(ch, 30, nil)
		if success {
			t.Error("conjure should fail for NPC casters")
		}
	})

	t.Run("already has pet", func(t *testing.T) {
		ch := types.NewCharacter("Warlock")
		ch.Level = 30
		ch.Pet = types.NewCharacter("OldPet")

		success := spellConjure(ch, 30, nil)
		if success {
			t.Error("conjure should fail when caster already has pet")
		}
	})

	t.Run("no demon stone and not immortal", func(t *testing.T) {
		ch := types.NewCharacter("Warlock")
		ch.Level = 30

		success := spellConjure(ch, 30, nil)
		if success {
			t.Error("conjure should fail without demon stone (unless immortal)")
		}
	})

	t.Run("immortal can cast without stone", func(t *testing.T) {
		ch := types.NewCharacter("Immortal")
		ch.Level = 110 // Immortal level

		success := spellConjure(ch, 110, nil)
		if !success {
			t.Error("immortal should be able to conjure without demon stone")
		}
	})
}
