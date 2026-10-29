package game

import (
	"slices"
	"testing"

	"rotmud/pkg/loader"
	"rotmud/pkg/types"
)

// ROT resets an area's rooms in vnum order. Valley mob 7803 resets in 7848
// (world max 4), 7853 (max 1), 7860 and 7866 (max 4): 7848 goes first, so
// 7853's max-1 reset never fires and three spawn, every boot.
func TestResetAreaRoomOrder(t *testing.T) {
	world, err := loader.NewAreaLoader("../../data/areas").LoadAll()
	if err != nil {
		t.Fatalf("load world: %v", err)
	}
	r := NewResetSystem(world)
	r.ResetArea(world.Rooms[7853].Area)

	if n := r.countMobInWorld(7803); n != 3 {
		t.Errorf("mob 7803 spawned %d times, want 3", n)
	}
	for _, ch := range world.Rooms[7853].People {
		if ch.MobVnum == 7803 {
			t.Error("max-1 reset in room 7853 fired")
		}
	}
}

// A mob killed and repopped is swapped in the game loop's character list:
// the corpse's owner goes, the fresh mob is registered so it fights back.
func TestRepopJoinsGameLoop(t *testing.T) {
	world, err := loader.NewAreaLoader("../../data/areas").LoadAll()
	if err != nil {
		t.Fatalf("load world: %v", err)
	}
	g := NewGameLoop()
	g.Resets = NewResetSystem(world)
	g.Resets.OnSpawn = g.AddCharacter
	area := world.Rooms[7853].Area
	g.Areas = []*types.Area{area}

	g.Resets.ResetArea(area)
	if len(g.Characters) == 0 {
		t.Fatal("reset registered no mobs")
	}
	dead := g.Characters[0]
	dead.InRoom.RemovePerson(dead)
	dead.InRoom, dead.Position = nil, types.PosDead
	before := len(g.Characters)

	area.Age = AreaResetAgeEmpty
	g.areaUpdate()
	if slices.Contains(g.Characters, dead) {
		t.Error("slain mob still in the character list")
	}
	if len(g.Characters) != before {
		t.Errorf("%d characters after repop, want %d", len(g.Characters), before)
	}
}
