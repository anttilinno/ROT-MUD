package game

import (
	"testing"

	"rotmud/pkg/types"
)

// TestResetRelocksDoor: an opened door goes back to its reset state (closed
// and locked) on both sides when the area resets.
func TestResetRelocksDoor(t *testing.T) {
	a := types.NewRoom(3719, "A", "x")
	b := types.NewRoom(3721, "B", "x")
	east := types.NewExit(types.DirEast, 3721)
	east.ToRoom = b
	east.ResetFlags = types.ExitIsDoor | types.ExitClosed | types.ExitLocked
	west := types.NewExit(types.DirWest, 3719)
	west.ToRoom = a
	west.ResetFlags = types.ExitIsDoor | types.ExitClosed | types.ExitLocked
	a.SetExit(types.DirEast, east)
	b.SetExit(types.DirWest, west)
	east.Flags, west.Flags = types.ExitIsDoor, types.ExitIsDoor // unlocked and opened

	NewResetSystem(nil).resetRoom(a)

	if !east.Flags.Has(types.ExitLocked) || !west.Flags.Has(types.ExitLocked) {
		t.Errorf("door not relocked: east=%v west=%v", east.Flags, west.Flags)
	}
}
