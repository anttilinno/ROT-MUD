package types

import "testing"

func TestCharacter(t *testing.T) {
	t.Run("NewCharacter creates character with correct values", func(t *testing.T) {
		ch := NewCharacter("Gandalf")
		if ch.Name != "Gandalf" {
			t.Errorf("expected name 'Gandalf', got '%s'", ch.Name)
		}
		if ch.Position != PosStanding {
			t.Errorf("expected position Standing, got %v", ch.Position)
		}
	})

	t.Run("Character stats work correctly", func(t *testing.T) {
		ch := NewCharacter("Test")
		ch.PermStats[StatStr] = 18
		ch.PermStats[StatInt] = 15
		ch.ModStats[StatStr] = 2 // Bonus from equipment/spell

		if ch.GetStat(StatStr) != 20 {
			t.Errorf("expected str 20 (18+2), got %d", ch.GetStat(StatStr))
		}
		if ch.GetStat(StatInt) != 15 {
			t.Errorf("expected int 15, got %d", ch.GetStat(StatInt))
		}
	})

	t.Run("Character HP/Mana/Move work correctly", func(t *testing.T) {
		ch := NewCharacter("Test")
		ch.MaxHit = 100
		ch.Hit = 100
		ch.MaxMana = 50
		ch.Mana = 50
		ch.MaxMove = 100
		ch.Move = 100

		if ch.HitPercent() != 100 {
			t.Errorf("expected 100%% HP, got %d%%", ch.HitPercent())
		}

		ch.Hit = 50
		if ch.HitPercent() != 50 {
			t.Errorf("expected 50%% HP, got %d%%", ch.HitPercent())
		}
	})

	t.Run("Character is NPC", func(t *testing.T) {
		ch := NewCharacter("Mob")
		ch.Act.Set(ActNPC)

		if !ch.IsNPC() {
			t.Error("expected character with ActNPC to be NPC")
		}
	})

	t.Run("Character is player", func(t *testing.T) {
		ch := NewCharacter("Player")
		if ch.IsNPC() {
			t.Error("expected character without ActNPC to be player")
		}
	})

	t.Run("Character affects work", func(t *testing.T) {
		ch := NewCharacter("Test")
		aff := NewAffect("sanctuary", 50, 10, ApplyNone, 0, AffSanctuary)
		ch.AddAffect(aff)

		if !ch.IsAffected(AffSanctuary) {
			t.Error("expected character to be affected by sanctuary")
		}
		if ch.IsAffected(AffBlind) {
			t.Error("expected character to not be affected by blind")
		}
	})

	t.Run("Character equipment slots", func(t *testing.T) {
		ch := NewCharacter("Test")
		sword := NewObject(3042, "a sword", ItemTypeWeapon)

		ch.Equip(sword, WearLocWield)
		if ch.GetEquipment(WearLocWield) != sword {
			t.Error("expected sword to be equipped in wield slot")
		}

		ch.Unequip(WearLocWield)
		if ch.GetEquipment(WearLocWield) != nil {
			t.Error("expected wield slot to be empty after unequip")
		}
	})

	t.Run("Character inventory", func(t *testing.T) {
		ch := NewCharacter("Test")
		sword := NewObject(3042, "a sword", ItemTypeWeapon)

		ch.AddInventory(sword)
		if len(ch.Inventory) != 1 {
			t.Errorf("expected 1 item in inventory, got %d", len(ch.Inventory))
		}
		if sword.CarriedBy != ch {
			t.Error("expected sword's CarriedBy to be the character")
		}

		ch.RemoveInventory(sword)
		if len(ch.Inventory) != 0 {
			t.Errorf("expected 0 items after removal, got %d", len(ch.Inventory))
		}
	})

	t.Run("Character carry weight", func(t *testing.T) {
		ch := NewCharacter("Test")
		sword := NewObject(3042, "a sword", ItemTypeWeapon)
		sword.Weight = 5
		armor := NewObject(3043, "armor", ItemTypeArmor)
		armor.Weight = 20

		ch.AddInventory(sword)
		ch.AddInventory(armor)

		if ch.CarryWeight() != 25 {
			t.Errorf("expected carry weight 25, got %d", ch.CarryWeight())
		}
	})
}

func TestCharacterCombat(t *testing.T) {
	t.Run("Character can enter combat", func(t *testing.T) {
		ch := NewCharacter("Player")
		mob := NewCharacter("Mob")
		mob.Act.Set(ActNPC)

		ch.Fighting = mob
		if !ch.InCombat() {
			t.Error("expected character to be in combat")
		}
		if ch.Fighting != mob {
			t.Error("expected Fighting to be the mob")
		}
	})

	t.Run("Character position affects combat", func(t *testing.T) {
		ch := NewCharacter("Test")
		ch.Position = PosSleeping

		if ch.CanAct() {
			t.Error("sleeping character should not be able to act")
		}

		ch.Position = PosResting
		if !ch.CanAct() {
			t.Error("resting character should be able to act")
		}

		ch.Position = PosStanding
		if !ch.CanAct() {
			t.Error("standing character should be able to act")
		}
	})
}

func TestCharacterAlignment(t *testing.T) {
	t.Run("Alignment affects description", func(t *testing.T) {
		ch := NewCharacter("Test")

		ch.Alignment = 1000
		if ch.AlignmentString() != "angelic" {
			t.Errorf("expected 'angelic', got '%s'", ch.AlignmentString())
		}

		ch.Alignment = 0
		if ch.AlignmentString() != "neutral" {
			t.Errorf("expected 'neutral', got '%s'", ch.AlignmentString())
		}

		ch.Alignment = -1000
		if ch.AlignmentString() != "satanic" {
			t.Errorf("expected 'satanic', got '%s'", ch.AlignmentString())
		}
	})

	t.Run("Alignment check functions", func(t *testing.T) {
		ch := NewCharacter("Test")

		ch.Alignment = 500
		if !ch.IsGood() {
			t.Error("expected alignment 500 to be good")
		}

		ch.Alignment = -500
		if !ch.IsEvil() {
			t.Error("expected alignment -500 to be evil")
		}

		ch.Alignment = 0
		if !ch.IsNeutral() {
			t.Error("expected alignment 0 to be neutral")
		}
	})
}
