package persistence

import (
	"os"
	"path/filepath"
	"testing"

	"rotmud/pkg/types"
)

func TestPlayerSave(t *testing.T) {
	t.Run("SavePlayer creates JSON file", func(t *testing.T) {
		tmpDir := t.TempDir()
		p := NewPlayerPersistence(tmpDir)

		ch := types.NewCharacter("TestPlayer")
		ch.Level = 10
		ch.Class = 1
		ch.Race = 2
		ch.Hit = 100
		ch.MaxHit = 150
		ch.Mana = 50
		ch.MaxMana = 100
		ch.Move = 80
		ch.MaxMove = 100
		ch.Gold = 500
		ch.Exp = 10000
		ch.Alignment = 350
		ch.PermStats[types.StatStr] = 18
		ch.PermStats[types.StatInt] = 15

		err := p.SavePlayer(ch)
		if err != nil {
			t.Fatalf("failed to save player: %v", err)
		}

		// Check file exists (name is capitalized)
		path := filepath.Join(tmpDir, "Testplayer.json")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected player file to exist at %s", path)
		}
	})
}

func TestPlayerLoad(t *testing.T) {
	t.Run("LoadPlayer restores character", func(t *testing.T) {
		tmpDir := t.TempDir()
		p := NewPlayerPersistence(tmpDir)

		// Create and save a character
		original := types.NewCharacter("TestLoad")
		original.Level = 25
		original.Class = 2
		original.Race = 3
		original.Hit = 200
		original.MaxHit = 250
		original.Mana = 100
		original.MaxMana = 150
		original.Move = 90
		original.MaxMove = 120
		original.Gold = 1000
		original.Silver = 500
		original.Platinum = 10
		original.Exp = 50000
		original.Alignment = -500
		original.PermStats[types.StatStr] = 20
		original.PermStats[types.StatInt] = 18
		original.PermStats[types.StatWis] = 16
		original.PermStats[types.StatDex] = 17
		original.PermStats[types.StatCon] = 19

		err := p.SavePlayer(original)
		if err != nil {
			t.Fatalf("failed to save player: %v", err)
		}

		// Load it back
		loaded, err := p.LoadPlayer("TestLoad")
		if err != nil {
			t.Fatalf("failed to load player: %v", err)
		}

		// Verify fields
		if loaded.Name != "TestLoad" {
			t.Errorf("expected name 'TestLoad', got '%s'", loaded.Name)
		}
		if loaded.Level != 25 {
			t.Errorf("expected level 25, got %d", loaded.Level)
		}
		if loaded.Hit != 200 {
			t.Errorf("expected hit 200, got %d", loaded.Hit)
		}
		if loaded.MaxHit != 250 {
			t.Errorf("expected max_hit 250, got %d", loaded.MaxHit)
		}
		if loaded.Gold != 1000 {
			t.Errorf("expected gold 1000, got %d", loaded.Gold)
		}
		if loaded.Alignment != -500 {
			t.Errorf("expected alignment -500, got %d", loaded.Alignment)
		}
		if loaded.PermStats[types.StatStr] != 20 {
			t.Errorf("expected str 20, got %d", loaded.PermStats[types.StatStr])
		}
	})
}

func TestPlayerExists(t *testing.T) {
	tmpDir := t.TempDir()
	p := NewPlayerPersistence(tmpDir)

	t.Run("Exists returns false for missing player", func(t *testing.T) {
		if p.Exists("NonExistent") {
			t.Error("expected Exists to return false")
		}
	})

	t.Run("Exists returns true after save", func(t *testing.T) {
		ch := types.NewCharacter("ExistsTest")
		p.SavePlayer(ch)

		if !p.Exists("ExistsTest") {
			t.Error("expected Exists to return true")
		}
	})
}

func TestPlayerInventory(t *testing.T) {
	t.Run("Inventory is saved and loaded", func(t *testing.T) {
		tmpDir := t.TempDir()
		p := NewPlayerPersistence(tmpDir)

		ch := types.NewCharacter("InvTest")
		sword := types.NewObject(3042, "a sword", types.ItemTypeWeapon)
		sword.Level = 10
		shield := types.NewObject(3100, "a shield", types.ItemTypeArmor)
		shield.Level = 15

		ch.AddInventory(sword)
		ch.AddInventory(shield)

		err := p.SavePlayer(ch)
		if err != nil {
			t.Fatalf("save failed: %v", err)
		}

		loaded, err := p.LoadPlayer("InvTest")
		if err != nil {
			t.Fatalf("load failed: %v", err)
		}

		if len(loaded.Inventory) != 2 {
			t.Errorf("expected 2 items in inventory, got %d", len(loaded.Inventory))
		}
	})
}

func TestPlayerEquipment(t *testing.T) {
	t.Run("Equipment is saved and loaded", func(t *testing.T) {
		tmpDir := t.TempDir()
		p := NewPlayerPersistence(tmpDir)

		ch := types.NewCharacter("EquipTest")
		sword := types.NewObject(3042, "a sword", types.ItemTypeWeapon)
		ch.Equip(sword, types.WearLocWield)

		err := p.SavePlayer(ch)
		if err != nil {
			t.Fatalf("save failed: %v", err)
		}

		loaded, err := p.LoadPlayer("EquipTest")
		if err != nil {
			t.Fatalf("load failed: %v", err)
		}

		wielded := loaded.GetEquipment(types.WearLocWield)
		if wielded == nil {
			t.Fatal("expected wielded item")
		}
		if wielded.Vnum != 3042 {
			t.Errorf("expected vnum 3042, got %d", wielded.Vnum)
		}
	})
}

func TestPlayerSkills(t *testing.T) {
	t.Run("Skills are saved and loaded", func(t *testing.T) {
		tmpDir := t.TempDir()
		p := NewPlayerPersistence(tmpDir)

		ch := types.NewCharacter("SkillTest")
		ch.PCData = &types.PCData{
			Learned: map[string]int{
				"sword":    85,
				"parry":    70,
				"fireball": 95,
			},
		}

		err := p.SavePlayer(ch)
		if err != nil {
			t.Fatalf("save failed: %v", err)
		}

		loaded, err := p.LoadPlayer("SkillTest")
		if err != nil {
			t.Fatalf("load failed: %v", err)
		}

		if loaded.PCData == nil {
			t.Fatal("expected PCData")
		}
		if loaded.PCData.Learned["sword"] != 85 {
			t.Errorf("expected sword skill 85, got %d", loaded.PCData.Learned["sword"])
		}
		if loaded.PCData.Learned["fireball"] != 95 {
			t.Errorf("expected fireball skill 95, got %d", loaded.PCData.Learned["fireball"])
		}
	})
}

func TestEquipmentFullState(t *testing.T) {
	t.Run("Equipment preserves full state", func(t *testing.T) {
		tmpDir := t.TempDir()
		p := NewPlayerPersistence(tmpDir)

		ch := types.NewCharacter("EquipFullTest")

		// Create a wand with charges (uses Values)
		wand := types.NewObject(5001, "a magic wand", types.ItemTypeWand)
		wand.Level = 25
		wand.Values[0] = 20 // Spell level
		wand.Values[1] = 10 // Max charges
		wand.Values[2] = 7  // Current charges
		wand.Values[3] = 42 // Spell number
		wand.Condition = 85
		wand.Weight = 2
		wand.Cost = 5000
		wand.Material = "oak"
		wand.Enchanted = true

		// Add an affect to the wand
		wandAff := types.NewAffect("bless", 20, -1, types.ApplyHitroll, 2, 0)
		wand.Affects.Add(wandAff)

		ch.Equip(wand, types.WearLocHold)

		// Create armor with AC
		armor := types.NewObject(5002, "a set of dragon scale", types.ItemTypeArmor)
		armor.Level = 30
		armor.Values[0] = -25 // AC pierce
		armor.Values[1] = -25 // AC bash
		armor.Values[2] = -25 // AC slash
		armor.Values[3] = -20 // AC exotic
		armor.Condition = 100
		armor.Weight = 15
		armor.Cost = 10000

		ch.Equip(armor, types.WearLocBody)

		err := p.SavePlayer(ch)
		if err != nil {
			t.Fatalf("save failed: %v", err)
		}

		loaded, err := p.LoadPlayer("EquipFullTest")
		if err != nil {
			t.Fatalf("load failed: %v", err)
		}

		// Verify wand
		loadedWand := loaded.GetEquipment(types.WearLocHold)
		if loadedWand == nil {
			t.Fatal("expected held wand")
		}
		if loadedWand.Vnum != 5001 {
			t.Errorf("expected wand vnum 5001, got %d", loadedWand.Vnum)
		}
		if loadedWand.Values[2] != 7 {
			t.Errorf("expected 7 charges, got %d", loadedWand.Values[2])
		}
		if loadedWand.Material != "oak" {
			t.Errorf("expected material 'oak', got '%s'", loadedWand.Material)
		}
		if !loadedWand.Enchanted {
			t.Error("expected wand to be enchanted")
		}
		if loadedWand.Affects.Len() != 1 {
			t.Errorf("expected 1 affect on wand, got %d", loadedWand.Affects.Len())
		}
		if aff := loadedWand.Affects.FindByType("bless"); aff == nil {
			t.Error("expected bless affect on wand")
		} else if aff.Modifier != 2 {
			t.Errorf("expected hitroll modifier 2, got %d", aff.Modifier)
		}

		// Verify armor
		loadedArmor := loaded.GetEquipment(types.WearLocBody)
		if loadedArmor == nil {
			t.Fatal("expected body armor")
		}
		if loadedArmor.Values[0] != -25 {
			t.Errorf("expected AC pierce -25, got %d", loadedArmor.Values[0])
		}
	})
}

func TestInventoryNestedContainers(t *testing.T) {
	t.Run("Nested containers are saved and loaded", func(t *testing.T) {
		tmpDir := t.TempDir()
		p := NewPlayerPersistence(tmpDir)

		ch := types.NewCharacter("ContainerTest")

		// Create a backpack (outer container)
		backpack := types.NewObject(6001, "a leather backpack", types.ItemTypeContainer)
		backpack.Level = 5
		backpack.Values[0] = 100 // Capacity

		// Create a pouch inside the backpack (inner container)
		pouch := types.NewObject(6002, "a small pouch", types.ItemTypeContainer)
		pouch.Level = 1
		pouch.Values[0] = 20 // Capacity

		// Put items in the pouch
		gem := types.NewObject(6003, "a ruby", types.ItemTypeGem)
		gem.Cost = 500
		pouch.AddContent(gem)

		coin := types.NewObject(6004, "a gold coin", types.ItemTypeMoney)
		coin.Cost = 1
		pouch.AddContent(coin)

		// Put the pouch in the backpack
		backpack.AddContent(pouch)

		// Put a sword directly in the backpack
		sword := types.NewObject(6005, "a short sword", types.ItemTypeWeapon)
		sword.Level = 10
		sword.Values[1] = 1 // Dice number
		sword.Values[2] = 8 // Dice size
		backpack.AddContent(sword)

		ch.AddInventory(backpack)

		err := p.SavePlayer(ch)
		if err != nil {
			t.Fatalf("save failed: %v", err)
		}

		loaded, err := p.LoadPlayer("ContainerTest")
		if err != nil {
			t.Fatalf("load failed: %v", err)
		}

		if len(loaded.Inventory) != 1 {
			t.Fatalf("expected 1 item in inventory, got %d", len(loaded.Inventory))
		}

		loadedBackpack := loaded.Inventory[0]
		if loadedBackpack.Vnum != 6001 {
			t.Errorf("expected backpack vnum 6001, got %d", loadedBackpack.Vnum)
		}
		if len(loadedBackpack.Contents) != 2 {
			t.Fatalf("expected 2 items in backpack, got %d", len(loadedBackpack.Contents))
		}

		// Find the pouch
		var loadedPouch *types.Object
		var loadedSword *types.Object
		for _, item := range loadedBackpack.Contents {
			if item.Vnum == 6002 {
				loadedPouch = item
			}
			if item.Vnum == 6005 {
				loadedSword = item
			}
		}

		if loadedPouch == nil {
			t.Fatal("expected pouch in backpack")
		}
		if len(loadedPouch.Contents) != 2 {
			t.Errorf("expected 2 items in pouch, got %d", len(loadedPouch.Contents))
		}

		// Verify gem is in pouch
		foundGem := false
		for _, item := range loadedPouch.Contents {
			if item.Vnum == 6003 {
				foundGem = true
				if item.Cost != 500 {
					t.Errorf("expected gem cost 500, got %d", item.Cost)
				}
			}
		}
		if !foundGem {
			t.Error("expected ruby in pouch")
		}

		if loadedSword == nil {
			t.Fatal("expected sword in backpack")
		}
		if loadedSword.Values[2] != 8 {
			t.Errorf("expected sword dice size 8, got %d", loadedSword.Values[2])
		}
	})
}

func TestInventoryItemStates(t *testing.T) {
	t.Run("Item states are preserved", func(t *testing.T) {
		tmpDir := t.TempDir()
		p := NewPlayerPersistence(tmpDir)

		ch := types.NewCharacter("ItemStateTest")

		// Create a staff with charges
		staff := types.NewObject(7001, "a staff of power", types.ItemTypeStaff)
		staff.Level = 40
		staff.Values[0] = 30 // Spell level
		staff.Values[1] = 5  // Max charges
		staff.Values[2] = 3  // Current charges
		staff.Values[3] = 99 // Spell number
		staff.Condition = 75
		staff.Timer = 100 // Decay timer
		staff.Owner = "ItemStateTest"

		ch.AddInventory(staff)

		// Create a worn item
		ring := types.NewObject(7002, "a ring of protection", types.ItemTypeArmor)
		ring.Level = 15
		ring.Enchanted = true
		ringAff := types.NewAffect("armor", 15, -1, types.ApplyAC, -10, 0)
		ring.Affects.Add(ringAff)

		ch.AddInventory(ring)

		err := p.SavePlayer(ch)
		if err != nil {
			t.Fatalf("save failed: %v", err)
		}

		loaded, err := p.LoadPlayer("ItemStateTest")
		if err != nil {
			t.Fatalf("load failed: %v", err)
		}

		if len(loaded.Inventory) != 2 {
			t.Fatalf("expected 2 items, got %d", len(loaded.Inventory))
		}

		// Find and verify staff
		var loadedStaff *types.Object
		var loadedRing *types.Object
		for _, item := range loaded.Inventory {
			if item.Vnum == 7001 {
				loadedStaff = item
			}
			if item.Vnum == 7002 {
				loadedRing = item
			}
		}

		if loadedStaff == nil {
			t.Fatal("expected staff in inventory")
		}
		if loadedStaff.Values[2] != 3 {
			t.Errorf("expected 3 charges, got %d", loadedStaff.Values[2])
		}
		if loadedStaff.Condition != 75 {
			t.Errorf("expected condition 75, got %d", loadedStaff.Condition)
		}
		if loadedStaff.Timer != 100 {
			t.Errorf("expected timer 100, got %d", loadedStaff.Timer)
		}
		if loadedStaff.Owner != "ItemStateTest" {
			t.Errorf("expected owner 'ItemStateTest', got '%s'", loadedStaff.Owner)
		}

		if loadedRing == nil {
			t.Fatal("expected ring in inventory")
		}
		if !loadedRing.Enchanted {
			t.Error("expected ring to be enchanted")
		}
		if loadedRing.Affects.Len() != 1 {
			t.Errorf("expected 1 affect on ring, got %d", loadedRing.Affects.Len())
		}
	})
}
