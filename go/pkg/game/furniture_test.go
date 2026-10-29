package game

import (
	"strings"
	"testing"

	"rotmud/pkg/types"
)

func TestFurniture(t *testing.T) {
	d := NewCommandDispatcher()
	var out string
	d.Output = func(ch *types.Character, msg string) { out += msg }
	d.GameLoop = NewGameLoop()

	room := types.NewRoom(100, "Inn", "A cosy inn.")
	bench := types.NewObject(1, "a wooden bench", types.ItemTypeFurniture)
	bench.Name = "bench"
	bench.Values = [5]int{1, 0, types.SitOn | types.RestOn, 100, 100}
	ObjToRoom(bench, room)

	a, b := types.NewCharacter("Aldor"), types.NewCharacter("Bria")
	for _, ch := range []*types.Character{a, b} {
		ch.Position = types.PosStanding
		CharToRoom(ch, room)
	}

	d.cmdSit(a, "bench")
	if a.On != bench || a.Position != types.PosSitting {
		t.Fatalf("sit bench: on=%v pos=%v out=%q", a.On, a.Position, out)
	}

	out = ""
	d.cmdSit(b, "bench")
	if b.On != nil || !strings.Contains(out, "There's no more room on a wooden bench.") {
		t.Errorf("bench holds one: on=%v out=%q", b.On, out)
	}

	out = ""
	d.cmdSleep(a, "")
	if a.Position != types.PosSitting || !strings.Contains(out, "You can't sleep on that!") {
		t.Errorf("sleep on a bench without sleep flags: pos=%v out=%q", a.Position, out)
	}

	out = ""
	d.doLook(b, "")
	if !strings.Contains(out, "Aldor is sitting on a wooden bench.") {
		t.Errorf("look: %q", out)
	}

	d.cmdStand(a, "")
	if a.On != nil {
		t.Error("standing up should leave the bench")
	}

	d.cmdRest(a, "bench")
	CharToRoom(a, types.NewRoom(101, "Street", ""))
	if a.On != nil {
		t.Error("leaving the room should leave the bench")
	}

	d.cmdRest(b, "bench")
	ObjFromRoom(bench)
	if b.On != nil {
		t.Error("removing the bench should drop its user")
	}
}
