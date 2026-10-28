package game

import (
	"strings"
	"testing"

	"rotmud/pkg/types"
)

// TestCompareUsesSameSlot: comparing a shield must pick the worn shield, not
// the first worn armor (a neck cloak).
func TestCompareUsesSameSlot(t *testing.T) {
	d := NewCommandDispatcher()
	var out string
	d.Output = func(ch *types.Character, msg string) { out += msg }

	ch := types.NewCharacter("Test")
	armor := func(name string, slot types.WearFlags) *types.Object {
		o := types.NewObject(1, name, types.ItemTypeArmor)
		o.WearFlags.Set(types.WearTake)
		o.WearFlags.Set(slot)
		return o
	}
	ch.Equipment[types.WearLocNeck1] = armor("a sub issue cloak", types.WearNeck)
	ch.Equipment[types.WearLocShield] = armor("a sub issue shield", types.WearShield)
	newShield := armor("a small wooden shield", types.WearShield)
	newShield.Name = "shield wooden"
	ch.Inventory = append(ch.Inventory, newShield)

	d.cmdCompare(ch, "shield")
	if strings.Contains(out, "cloak") || !strings.Contains(out, "sub issue shield") {
		t.Errorf("compared against the wrong item: %q", out)
	}
}
