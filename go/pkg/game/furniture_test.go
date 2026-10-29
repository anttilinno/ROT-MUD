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

// Furniture v3/v4 scale hit, move and mana regen by value/90 (ROT update.c).
func TestFurnitureRegenBonus(t *testing.T) {
	g := NewGameLoop()
	room := types.NewRoom(100, "Inn", "")
	room.HealRate, room.ManaRate = 90, 90
	mob := types.NewNPC(1, "mob", 85) // NPC resting gain: 5+level = 90
	mob.Position = types.PosResting
	mob.MaxHit, mob.MaxMana, mob.MaxMove = 1000, 1000, 1000
	CharToRoom(mob, room)

	bed := types.NewObject(1, "a bed", types.ItemTypeFurniture)
	bed.Values = [5]int{1, 0, types.RestOn, 180, 45}
	ObjToRoom(bed, room)
	mob.On = bed

	if got := g.hitGain(mob); got != 180 {
		t.Errorf("hit gain = %d, want 180", got)
	}
	if got := g.manaGain(mob); got != 45 {
		t.Errorf("mana gain = %d, want 45", got)
	}
	if got := g.moveGain(mob); got != 170 { // level 85 * 180/90
		t.Errorf("move gain = %d, want 170", got)
	}
}
