package loader

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadMidgaard(t *testing.T) {
	// Get path to data directory relative to this test file
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not get test file path")
	}
	dataPath := filepath.Join(filepath.Dir(filename), "..", "..", "data", "areas")

	t.Run("Load midgaard area from disk", func(t *testing.T) {
		loader := NewAreaLoader(dataPath)
		world, err := loader.LoadAll()
		if err != nil {
			t.Fatalf("failed to load areas: %v", err)
		}

		// Check we loaded multiple areas (converted from ROM format)
		if len(world.Areas) < 10 {
			t.Errorf("expected at least 10 areas, got %d", len(world.Areas))
		}

		// Check room count - should have thousands of rooms
		if len(world.Rooms) < 1000 {
			t.Errorf("expected at least 1000 rooms, got %d", len(world.Rooms))
		}

		// Check specific rooms - Temple of Thoth (ROM midgaard)
		temple := world.GetRoom(3001)
		if temple == nil {
			t.Fatal("expected room 3001 (Temple of Thoth)")
		}
		if temple.Name != "Temple of Thoth" {
			t.Errorf("expected 'Temple of Thoth', got '%s'", temple.Name)
		}

		// Check exits are resolved - north leads to 3054
		northExit := temple.GetExit(0) // DirNorth
		if northExit == nil {
			t.Fatal("expected north exit from temple")
		}
		if northExit.ToRoom == nil {
			t.Fatal("expected north exit to resolve to room")
		}
		if northExit.ToRoom.Vnum != 3054 {
			t.Errorf("expected north exit to lead to 3054, got %d", northExit.ToRoom.Vnum)
		}

		// Check MUD School room exists (3700)
		school := world.GetRoom(3700)
		if school == nil {
			t.Fatal("expected room 3700 (MUD School entrance)")
		}
		if school.Name != "Entrance to Mud School" {
			t.Errorf("expected 'Entrance to Mud School', got '%s'", school.Name)
		}

		// Check mob templates
		wizard := world.GetMobTemplate(3000)
		if wizard == nil {
			t.Fatal("expected mob template 3000 (wizard)")
		}
		if wizard.Level != 46 {
			t.Errorf("expected wizard level 46, got %d", wizard.Level)
		}

		// Check object templates - long sword (vnum 3022)
		sword := world.GetObjTemplate(3022)
		if sword == nil {
			t.Fatal("expected object template 3022 (long sword)")
		}
		if sword.ItemType != "weapon" {
			t.Errorf("expected item_type 'weapon', got '%s'", sword.ItemType)
		}
		if sword.Weapon == nil {
			t.Fatal("expected weapon data")
		}
		if sword.Weapon.DiceNumber != 2 {
			t.Errorf("expected dice_number 2, got %d", sword.Weapon.DiceNumber)
		}

		// Check fountain object template (vnum 3135)
		fountain := world.GetObjTemplate(3135)
		if fountain == nil {
			t.Fatal("expected object template 3135 (fountain)")
		}
		if fountain.ItemType != "fountain" {
			t.Errorf("expected item_type 'fountain', got '%s'", fountain.ItemType)
		}

		// Check adept mob template (vnum 3707) has special function
		adeptTmpl := world.GetMobTemplate(3707)
		if adeptTmpl == nil {
			t.Fatal("expected mob template 3707 (adept)")
		}
		if adeptTmpl.Special != "spec_cast_adept" {
			t.Errorf("expected special 'spec_cast_adept', got '%s'", adeptTmpl.Special)
		}

		// Verify special is copied to Character when created from template
		adeptChar := world.CreateMobFromTemplate(3707)
		if adeptChar == nil {
			t.Fatal("failed to create adept from template")
		}
		if adeptChar.Special != "spec_cast_adept" {
			t.Errorf("expected character special 'spec_cast_adept', got '%s'", adeptChar.Special)
		}

		// Check Temple Square room (3005) has obj_resets for fountain
		templeSquare := world.GetRoom(3005)
		if templeSquare == nil {
			t.Fatal("expected room 3005 (Temple Square)")
		}
		if templeSquare.Name != "The Temple Square" {
			t.Errorf("expected 'The Temple Square', got '%s'", templeSquare.Name)
		}

		// Check obj_resets are loaded
		foundFountainReset := false
		for _, or := range templeSquare.ObjResets {
			if or.Vnum == 3135 {
				foundFountainReset = true
				t.Logf("Found fountain reset: vnum=%d, max=%d, count=%d", or.Vnum, or.Max, or.Count)
			}
		}
		if !foundFountainReset {
			t.Errorf("expected fountain reset (vnum 3135) in Temple Square obj_resets, found %d resets", len(templeSquare.ObjResets))
			for _, or := range templeSquare.ObjResets {
				t.Logf("  reset: vnum=%d", or.Vnum)
			}
		}
	})
}
