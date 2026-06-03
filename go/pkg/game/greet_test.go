package game

import (
	"testing"

	"rotmud/pkg/types"
)

func TestEntryHints(t *testing.T) {
	area := &types.Area{Name: "the Goblin Caves", LowRange: 40, HighRange: 50}

	here := types.NewRoom(100, "Cave Mouth", "A dark opening.")
	here.Area = area

	north := types.NewRoom(101, "Tunnel", "A tunnel.")
	here.Exits[types.DirNorth] = &types.Exit{ToRoom: north}

	// A dangerous mob lurking north.
	ogre := types.NewCharacter("a hulking ogre")
	ogre.Act.Set(types.ActNPC)
	ogre.ShortDesc = "a hulking ogre"
	ogre.Level = 60
	north.AddPerson(ogre)

	low := types.NewCharacter("Newbie")
	low.Level = 5

	hints := entryHints(low, here)
	for _, want := range []string{"Goblin Caves", "perilous", "exits lead", "north", "hulking ogre lurks"} {
		if !contains(hints, want) {
			t.Errorf("entryHints missing %q; got:\n%s", want, hints)
		}
	}
}

func TestEntryHintsHighLevelPlayer(t *testing.T) {
	area := &types.Area{Name: "the Newbie Yard", LowRange: 1, HighRange: 5}
	room := types.NewRoom(1, "Yard", "A safe yard.")
	room.Area = area

	hero := types.NewCharacter("Hero")
	hero.Level = 50

	hints := entryHints(hero, room)
	if !contains(hints, "little danger") {
		t.Errorf("expected trivial-area hint for high-level player; got: %s", hints)
	}
}

func TestEntryHintsNoArea(t *testing.T) {
	room := types.NewRoom(1, "Void", "Nothing.")
	ch := types.NewCharacter("X")
	if got := entryHints(ch, room); got != "" {
		t.Errorf("expected empty hints with no area/exits/threats, got %q", got)
	}
}
