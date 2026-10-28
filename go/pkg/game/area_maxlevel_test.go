package game

import (
	"testing"

	"rotmud/pkg/types"
)

func TestAreaMaxLevelBlocksEntry(t *testing.T) {
	d := NewCommandDispatcher()
	d.Output = func(*types.Character, string) {}

	school := types.NewArea("Mud School", 3700, 3799)
	school.MaxLevel = 10
	outside := types.NewRoom(3001, "Temple", "Outside.")
	inside := types.NewRoom(3700, "School", "Inside.")
	inside.Area = school
	exit := types.NewExit(types.DirNorth, 3700)
	exit.ToRoom = inside
	outside.SetExit(types.DirNorth, exit)

	for _, tc := range []struct {
		level int
		in    bool
	}{{10, true}, {11, false}} {
		ch := types.NewCharacter("Walker")
		ch.Level, ch.Position = tc.level, types.PosStanding
		CharToRoom(ch, outside)
		d.doMove(ch, types.DirNorth)
		if got := ch.InRoom == inside; got != tc.in {
			t.Errorf("level %d: entered=%v, want %v", tc.level, got, tc.in)
		}
	}
}
