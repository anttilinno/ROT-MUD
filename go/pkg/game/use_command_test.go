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
