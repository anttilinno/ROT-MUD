package types

import "testing"

func TestObject(t *testing.T) {
	t.Run("NewObject creates object with correct values", func(t *testing.T) {
		obj := NewObject(3042, "a long sword", ItemTypeWeapon)
		if obj.Vnum != 3042 {
			t.Errorf("expected vnum 3042, got %d", obj.Vnum)
		}
		if obj.ShortDesc != "a long sword" {
			t.Errorf("expected short desc 'a long sword', got '%s'", obj.ShortDesc)
		}
		if obj.ItemType != ItemTypeWeapon {
			t.Errorf("expected item type weapon, got %v", obj.ItemType)
		}
	})

	t.Run("Object flags work correctly", func(t *testing.T) {
		obj := NewObject(3042, "a glowing sword", ItemTypeWeapon)
		obj.ExtraFlags.Set(ItemGlow)
		obj.ExtraFlags.Set(ItemMagic)

		if !obj.ExtraFlags.Has(ItemGlow) {
			t.Error("expected ItemGlow flag")
		}
		if !obj.ExtraFlags.Has(ItemMagic) {
			t.Error("expected ItemMagic flag")
		}
	})

	t.Run("Object wear flags work correctly", func(t *testing.T) {
		obj := NewObject(3042, "a long sword", ItemTypeWeapon)
		obj.WearFlags.Set(WearTake)
		obj.WearFlags.Set(WearWield)

		if !obj.CanTake() {
			t.Error("expected object to be takeable")
		}
		if !obj.CanWield() {
			t.Error("expected object to be wieldable")
		}
	})

	t.Run("Weapon object has correct values", func(t *testing.T) {
		obj := NewObject(3042, "a long sword", ItemTypeWeapon)
		obj.Values[0] = int(WeaponSword) // weapon type
		obj.Values[1] = 2                // dice number
		obj.Values[2] = 6                // dice size
		obj.Values[3] = int(DamSlash)    // damage type

		if obj.WeaponType() != WeaponSword {
			t.Errorf("expected weapon type sword, got %v", obj.WeaponType())
		}
		if obj.DiceNumber() != 2 {
			t.Errorf("expected 2 dice, got %d", obj.DiceNumber())
		}
		if obj.DiceSize() != 6 {
			t.Errorf("expected d6, got d%d", obj.DiceSize())
		}
	})

	t.Run("Container object properties", func(t *testing.T) {
		obj := NewObject(3050, "a leather bag", ItemTypeContainer)
		obj.Values[0] = 100 // capacity
		obj.Values[3] = 50  // max weight per item

		if obj.Capacity() != 100 {
			t.Errorf("expected capacity 100, got %d", obj.Capacity())
		}
	})

	t.Run("Object can contain other objects", func(t *testing.T) {
		bag := NewObject(3050, "a bag", ItemTypeContainer)
		sword := NewObject(3042, "a sword", ItemTypeWeapon)

		bag.AddContent(sword)
		if len(bag.Contents) != 1 {
			t.Errorf("expected 1 item in bag, got %d", len(bag.Contents))
		}
		if sword.InObject != bag {
			t.Error("expected sword's InObject to be the bag")
		}

		bag.RemoveContent(sword)
		if len(bag.Contents) != 0 {
			t.Errorf("expected 0 items in bag after removal, got %d", len(bag.Contents))
		}
		if sword.InObject != nil {
			t.Error("expected sword's InObject to be nil after removal")
		}
	})

	t.Run("Object timer works", func(t *testing.T) {
		obj := NewObject(3042, "a sword", ItemTypeWeapon)
		obj.Timer = 5

		if obj.IsExpired() {
			t.Error("object with timer 5 should not be expired")
		}

		obj.Timer = 0
		if !obj.IsExpired() {
			t.Error("object with timer 0 should be expired")
		}

		obj.Timer = -1
		if obj.IsExpired() {
			t.Error("object with timer -1 (no timer) should not be expired")
		}
	})
}

func TestObjectCondition(t *testing.T) {
	t.Run("Condition descriptions", func(t *testing.T) {
		obj := NewObject(3042, "a sword", ItemTypeWeapon)

		obj.Condition = 100
		if obj.ConditionString() != "perfect" {
			t.Errorf("expected 'perfect', got '%s'", obj.ConditionString())
		}

		obj.Condition = 75
		if obj.ConditionString() != "good" {
			t.Errorf("expected 'good', got '%s'", obj.ConditionString())
		}

		obj.Condition = 25
		if obj.ConditionString() != "poor" {
			t.Errorf("expected 'poor', got '%s'", obj.ConditionString())
		}
	})
}
