package game

import (
	"testing"

	"rotmud/pkg/loader"
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
