package game

import (
	"testing"

	"rotmud/pkg/types"
)

func TestUseTeleportRune(t *testing.T) {
	d := NewCommandDispatcher()
	out := new(string)
	d.Output = func(ch *types.Character, msg string) { *out += msg }
	d.GameLoop = NewGameLoop()

	here := types.NewRoom(100, "Start", "Here.")
	dest := types.NewRoom(200, "New Thalos", "A market square.")
	d.GameLoop.Rooms[100] = here
	d.GameLoop.Rooms[200] = dest

	ch := types.NewCharacter("Aldor")
	ch.Position = types.PosStanding
	here.AddPerson(ch)
	ch.InRoom = here

	rune := types.NewObject(3180, "a teleport rune to New Thalos", types.ItemTypePortal)
	rune.Name = "rune teleport thalos"
	rune.Values[0] = 1   // single charge
	rune.Values[3] = 200 // destination
	ch.Inventory = append(ch.Inventory, rune)

	d.cmdUse(ch, "rune")

	if ch.InRoom != dest {
		t.Fatalf("expected to be in dest room, still in %v", ch.InRoom.Name)
	}
	if len(ch.Inventory) != 0 {
		t.Errorf("expected single-use rune consumed, inventory=%d", len(ch.Inventory))
	}
	if !contains(*out, "crumbles to dust") {
		t.Errorf("expected consume message, got %q", *out)
	}
}

func TestUseNonPortalRejected(t *testing.T) {
	d := NewCommandDispatcher()
	out := new(string)
	d.Output = func(ch *types.Character, msg string) { *out += msg }
	d.GameLoop = NewGameLoop()

	ch := types.NewCharacter("Aldor")
	ch.Position = types.PosStanding
	ch.InRoom = types.NewRoom(1, "Room", "x")

	sword := types.NewObject(50, "a sword", types.ItemTypeWeapon)
	sword.Name = "sword"
	ch.Inventory = append(ch.Inventory, sword)

	d.cmdUse(ch, "sword")
	if !contains(*out, "can't use that") {
		t.Errorf("expected rejection for non-portal, got %q", *out)
	}
}

func TestUseRuneEndsCombat(t *testing.T) {
	d := NewCommandDispatcher()
	d.Output = func(ch *types.Character, msg string) {}
	d.GameLoop = NewGameLoop()

	here := types.NewRoom(100, "Start", "Here.")
	dest := types.NewRoom(200, "Dest", "There.")
	d.GameLoop.Rooms[100] = here
	d.GameLoop.Rooms[200] = dest

	ch := types.NewCharacter("Aldor")
	mob := types.NewCharacter("goblin")
	CharToRoom(ch, here)
	CharToRoom(mob, here)
	ch.Fighting, ch.Position = mob, types.PosFighting
	mob.Fighting, mob.Position = ch, types.PosFighting

	rune := types.NewObject(3180, "a rune", types.ItemTypePortal)
	rune.Name = "rune"
	rune.Values[0] = 1
	rune.Values[3] = 200
	ch.Inventory = append(ch.Inventory, rune)

	d.cmdUse(ch, "rune")

	if ch.InRoom != dest {
		t.Fatalf("expected to be in dest room")
	}
	if mob.Fighting != nil || mob.Position != types.PosStanding {
		t.Errorf("opponent still fighting: fighting=%v pos=%v", mob.Fighting, mob.Position)
	}
	if ch.Fighting != nil || ch.Position != types.PosStanding {
		t.Errorf("user still fighting: pos=%v", ch.Position)
	}
}

func TestUseRuneByTwoWords(t *testing.T) {
	d := NewCommandDispatcher()
	d.Output = func(ch *types.Character, msg string) {}
	d.GameLoop = NewGameLoop()
	here := types.NewRoom(100, "Start", "x")
	shire := types.NewRoom(1103, "Shire", "x")
	d.GameLoop.Rooms[100], d.GameLoop.Rooms[1103] = here, shire
	ch := types.NewCharacter("Test")
	CharToRoom(ch, here)
	for _, r := range []struct {
		name string
		dest int
	}{{"rune teleport camelot", 3807}, {"rune teleport shire", 1103}} {
		o := types.NewObject(1, "a rune", types.ItemTypePortal)
		o.Name, o.Values[0], o.Values[3] = r.name, 1, r.dest
		ch.Inventory = append(ch.Inventory, o)
	}

	d.cmdUse(ch, "rune shire")
	if ch.InRoom != shire {
		t.Fatalf("used the wrong rune; in %v", ch.InRoom.Name)
	}
}
