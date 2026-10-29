package game

import (
	"testing"

	"rotmud/pkg/types"
)

// ROM 'R' resets shuffle a maze room's first N exits; the rest stay put.
func TestShuffleExits(t *testing.T) {
	room := types.NewRoom(1, "maze", "")
	room.RandomExits = 4
	for d := range 5 {
		room.Exits[d] = types.NewExit(types.Direction(d), 100+d)
	}
	moved := false
	for range 50 {
		shuffleExits(room)
		seen := map[int]bool{}
		for d, exit := range room.Exits[:4] {
			seen[exit.ToVnum] = true
			if exit.Direction != types.Direction(d) {
				t.Fatalf("exit in slot %d says direction %d", d, exit.Direction)
			}
			moved = moved || exit.ToVnum != 100+d
		}
		if len(seen) != 4 || room.Exits[4].ToVnum != 104 {
			t.Fatalf("shuffle lost or moved the wrong exits: %v", seen)
		}
	}
	if !moved {
		t.Error("exits never moved in 50 resets")
	}
}
