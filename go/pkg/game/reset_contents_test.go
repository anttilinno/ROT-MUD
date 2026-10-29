package game

import (
	"testing"

	"rotmud/pkg/loader"
)

// ROM 'P' resets: a room container is filled at reset, topped up (not
// duplicated) by later resets, and not refilled while players are around.
func TestResetFillsContainers(t *testing.T) {
	world, err := loader.NewAreaLoader("../../data/areas").LoadAll()
	if err != nil {
		t.Fatalf("load world: %v", err)
	}
	r := NewResetSystem(world)
	room := world.Rooms[3142]
	r.resetRoom(room)

	var chest = room.Objects[0]
	for _, obj := range room.Objects {
		if obj.Vnum == 3130 {
			chest = obj
		}
	}
	if chest.Vnum != 3130 || len(chest.Contents) != 1 || chest.Contents[0].Vnum != 3123 {
		t.Fatalf("container 3130 in room 3142 holds %v, want object 3123", chest.Contents)
	}

	r.resetRoom(room)
	if len(chest.Contents) != 1 {
		t.Errorf("second reset left %d items, want 1", len(chest.Contents))
	}

	chest.RemoveContent(chest.Contents[0])
	room.Area.NumPlayers = 1
	r.resetRoom(room)
	if len(chest.Contents) != 0 {
		t.Error("container refilled while players are in the area")
	}
	room.Area.NumPlayers = 0
	r.resetRoom(room)
	if len(chest.Contents) != 1 {
		t.Error("container not refilled once the area is empty")
	}
}
