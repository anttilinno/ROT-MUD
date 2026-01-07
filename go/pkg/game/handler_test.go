package game

import (
	"testing"

	"rotmud/pkg/types"
)

func TestCharToRoom(t *testing.T) {
	room1 := types.NewRoom(3001, "Temple of Mota", "A grand temple.")
	room2 := types.NewRoom(3002, "Market Square", "A busy market.")
	ch := types.NewCharacter("TestPlayer")

	// Move to first room
	CharToRoom(ch, room1)
	if ch.InRoom != room1 {
		t.Error("Character not in room1")
	}
	if len(room1.People) != 1 || room1.People[0] != ch {
		t.Error("Character not in room1's people list")
	}

	// Move to second room
	CharToRoom(ch, room2)
	if ch.InRoom != room2 {
		t.Error("Character not in room2")
	}
	if len(room2.People) != 1 || room2.People[0] != ch {
		t.Error("Character not in room2's people list")
	}
	if len(room1.People) != 0 {
		t.Error("Character still in room1's people list")
	}
}

func TestCharFromRoom(t *testing.T) {
	room := types.NewRoom(3001, "Temple", "A grand temple.")
	ch := types.NewCharacter("TestPlayer")

	CharToRoom(ch, room)
	CharFromRoom(ch)

	if ch.InRoom != nil {
		t.Error("Character still has room reference")
	}
	if len(room.People) != 0 {
		t.Error("Character still in room's people list")
	}
}

func TestObjToRoom(t *testing.T) {
	room := types.NewRoom(3001, "Temple", "A temple.")
	obj := types.NewObject(3042, "a long sword", types.ItemTypeWeapon)

	ObjToRoom(obj, room)

	if obj.InRoom != room {
		t.Error("Object not in room")
	}
	if len(room.Objects) != 1 || room.Objects[0] != obj {
		t.Error("Object not in room's object list")
	}
}

func TestObjFromRoom(t *testing.T) {
	room := types.NewRoom(3001, "Temple", "A temple.")
	obj := types.NewObject(3042, "a long sword", types.ItemTypeWeapon)

	ObjToRoom(obj, room)
	ObjFromRoom(obj)

	if obj.InRoom != nil {
		t.Error("Object still has room reference")
	}
	if len(room.Objects) != 0 {
		t.Error("Object still in room's object list")
	}
}

func TestObjToChar(t *testing.T) {
	ch := types.NewCharacter("TestPlayer")
	obj := types.NewObject(3042, "a long sword", types.ItemTypeWeapon)

	ObjToChar(obj, ch)

	if obj.CarriedBy != ch {
		t.Error("Object not carried by character")
	}
	if len(ch.Inventory) != 1 || ch.Inventory[0] != obj {
		t.Error("Object not in character's inventory")
	}
}

func TestObjFromChar(t *testing.T) {
	ch := types.NewCharacter("TestPlayer")
	obj := types.NewObject(3042, "a long sword", types.ItemTypeWeapon)

	ObjToChar(obj, ch)
	ObjFromChar(obj)

	if obj.CarriedBy != nil {
		t.Error("Object still carried by character")
	}
	if len(ch.Inventory) != 0 {
		t.Error("Object still in character's inventory")
	}
}

func TestFindObjInInventory(t *testing.T) {
	ch := types.NewCharacter("TestPlayer")
	sword := types.NewObject(3042, "a long sword", types.ItemTypeWeapon)
	sword.Name = "sword long"
	shield := types.NewObject(3043, "a wooden shield", types.ItemTypeArmor)
	shield.Name = "shield wooden"

	ObjToChar(sword, ch)
	ObjToChar(shield, ch)

	// Test exact name match
	if FindObjInInventory(ch, "sword") != sword {
		t.Error("Failed to find sword by name")
	}

	// Test keyword in short desc
	if FindObjInInventory(ch, "wooden") != shield {
		t.Error("Failed to find shield by keyword")
	}

	// Test prefix match
	if FindObjInInventory(ch, "swo") != sword {
		t.Error("Failed to find sword by prefix")
	}

	// Test case insensitivity
	if FindObjInInventory(ch, "SWORD") != sword {
		t.Error("Failed to find sword with uppercase")
	}

	// Test not found
	if FindObjInInventory(ch, "axe") != nil {
		t.Error("Found non-existent object")
	}
}

func TestFindObjInRoom(t *testing.T) {
	room := types.NewRoom(3001, "Temple", "A temple.")
	ch := types.NewCharacter("TestPlayer")
	CharToRoom(ch, room)

	sword := types.NewObject(3042, "a long sword", types.ItemTypeWeapon)
	sword.Name = "sword long"
	ObjToRoom(sword, room)

	if FindObjInRoom(ch, "sword") != sword {
		t.Error("Failed to find sword in room")
	}

	if FindObjInRoom(ch, "axe") != nil {
		t.Error("Found non-existent object in room")
	}
}

func TestFindCharInRoom(t *testing.T) {
	room := types.NewRoom(3001, "Temple", "A temple.")
	player := types.NewCharacter("TestPlayer")
	mob := types.NewNPC(3001, "guard", 10)
	mob.ShortDesc = "a burly guard"

	CharToRoom(player, room)
	CharToRoom(mob, room)

	// Find mob by name
	if FindCharInRoom(player, "guard") != mob {
		t.Error("Failed to find guard by name")
	}

	// Find by keyword in short desc
	if FindCharInRoom(player, "burly") != mob {
		t.Error("Failed to find guard by keyword")
	}

	// Self reference
	if FindCharInRoom(player, "self") != player {
		t.Error("Failed to find self")
	}

	// Not found
	if FindCharInRoom(player, "wizard") != nil {
		t.Error("Found non-existent character")
	}
}

func TestWearLocationName(t *testing.T) {
	tests := []struct {
		loc      types.WearLocation
		expected string
	}{
		{types.WearLocWield, "<wielded>"},
		{types.WearLocBody, "<worn on torso>"},
		{types.WearLocHead, "<worn on head>"},
		{types.WearLocHold, "<held>"},
	}

	for _, tc := range tests {
		result := WearLocationName(tc.loc)
		if result != tc.expected {
			t.Errorf("WearLocationName(%d) = %q, expected %q", tc.loc, result, tc.expected)
		}
	}
}

func TestParseNumberPrefix(t *testing.T) {
	tests := []struct {
		input string
		count int
		name  string
	}{
		{"corpse", 1, "corpse"},
		{"2.corpse", 2, "corpse"},
		{"3.sword", 3, "sword"},
		{"10.guard", 10, "guard"},
		{"1.item", 1, "item"},
		{"0.item", 1, "0.item"},     // 0 is invalid, treat as literal
		{"-1.item", 1, "-1.item"},   // negative is invalid
		{"abc.item", 1, "abc.item"}, // non-numeric prefix
		{".item", 1, ".item"},       // empty prefix
		{"2.", 2, ""},               // empty item name
	}

	for _, tc := range tests {
		count, name := parseNumberPrefix(tc.input)
		if count != tc.count || name != tc.name {
			t.Errorf("parseNumberPrefix(%q) = (%d, %q), expected (%d, %q)",
				tc.input, count, name, tc.count, tc.name)
		}
	}
}

func TestFindObjInRoomNumbered(t *testing.T) {
	room := types.NewRoom(3001, "Temple", "A temple.")
	ch := types.NewCharacter("TestPlayer")
	CharToRoom(ch, room)

	// Create multiple corpses
	corpse1 := types.NewObject(0, "the corpse of a goblin", types.ItemTypeCorpseNPC)
	corpse1.Name = "corpse"
	ObjToRoom(corpse1, room)

	corpse2 := types.NewObject(0, "the corpse of an orc", types.ItemTypeCorpseNPC)
	corpse2.Name = "corpse"
	ObjToRoom(corpse2, room)

	corpse3 := types.NewObject(0, "the corpse of a troll", types.ItemTypeCorpseNPC)
	corpse3.Name = "corpse"
	ObjToRoom(corpse3, room)

	// Test finding by index
	if FindObjInRoom(ch, "corpse") != corpse1 {
		t.Error("'corpse' should return first corpse")
	}
	if FindObjInRoom(ch, "1.corpse") != corpse1 {
		t.Error("'1.corpse' should return first corpse")
	}
	if FindObjInRoom(ch, "2.corpse") != corpse2 {
		t.Error("'2.corpse' should return second corpse")
	}
	if FindObjInRoom(ch, "3.corpse") != corpse3 {
		t.Error("'3.corpse' should return third corpse")
	}
	if FindObjInRoom(ch, "4.corpse") != nil {
		t.Error("'4.corpse' should return nil (only 3 corpses)")
	}
}

func TestCanWearAt(t *testing.T) {
	ch := types.NewCharacter("TestPlayer")

	// Create a sword that can be wielded
	sword := types.NewObject(3042, "a long sword", types.ItemTypeWeapon)
	sword.WearFlags.Set(types.WearWield)

	loc := CanWearAt(sword, ch)
	if loc != types.WearLocNone {
		t.Errorf("Expected WearLocNone for wield (needs WearWield check), got %d", loc)
	}

	// Create armor that can be worn on body
	armor := types.NewObject(3043, "leather armor", types.ItemTypeArmor)
	armor.WearFlags.Set(types.WearBody)

	loc = CanWearAt(armor, ch)
	if loc != types.WearLocBody {
		t.Errorf("Expected WearLocBody, got %d", loc)
	}

	// Equip the armor, should now return WearLocNone (slot occupied)
	ch.Equip(armor, types.WearLocBody)
	loc = CanWearAt(armor, ch)
	if loc != types.WearLocNone {
		t.Errorf("Expected WearLocNone for occupied slot, got %d", loc)
	}

	// Test double slots (finger)
	ring := types.NewObject(3044, "a gold ring", types.ItemTypeJewelry)
	ring.WearFlags.Set(types.WearFinger)

	loc = CanWearAt(ring, ch)
	if loc != types.WearLocFingerL {
		t.Errorf("Expected WearLocFingerL, got %d", loc)
	}

	// Fill first finger slot
	ch.Equip(ring, types.WearLocFingerL)

	// Create second ring
	ring2 := types.NewObject(3045, "a silver ring", types.ItemTypeJewelry)
	ring2.WearFlags.Set(types.WearFinger)

	loc = CanWearAt(ring2, ch)
	if loc != types.WearLocFingerR {
		t.Errorf("Expected WearLocFingerR, got %d", loc)
	}
}
