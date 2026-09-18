package game

import (
	"strings"
	"testing"

	"rotmud/pkg/types"
)

func TestWearDemonicArmour(t *testing.T) {
	d := NewCommandDispatcher()
	var out string
	d.Output = func(_ *types.Character, msg string) { out += msg }

	wear := func(ch *types.Character, owner string) *types.Object {
		ring := types.NewObject(types.VnumDemonicFirst, "a demonic ring", types.ItemTypeArmor)
		ring.Name = "demonic ring"
		ring.WearFlags.Set(types.WearTake)
		ring.WearFlags.Set(types.WearFinger)
		types.ForgeDemonic(ring, types.DemonBrass, owner)
		ch.AddInventory(ring)
		out = ""
		d.Dispatch(Command{Character: ch, Input: "wear ring"})
		return ring
	}
	newPlayer := func(class int) *types.Character {
		ch := types.NewCharacter("Azazel")
		ch.PCData = &types.PCData{}
		ch.Class, ch.Position = class, types.PosStanding
		return ch
	}

	if ring := wear(newPlayer(types.ClassWarrior), "Azazel"); ring.WearLoc != types.WearLocNone || !strings.Contains(out, "only a demon") {
		t.Fatalf("warrior wore demonic ring: %q", out)
	}
	if ring := wear(newPlayer(types.ClassDemon), "Belial"); ring.WearLoc != types.WearLocNone || !strings.Contains(out, "bound to Belial") {
		t.Fatalf("demon wore another's ring: %q", out)
	}
	if ring := wear(newPlayer(types.ClassDemon), "Azazel"); ring.WearLoc == types.WearLocNone {
		t.Fatalf("demon could not wear own ring: %q", out)
	}
}
