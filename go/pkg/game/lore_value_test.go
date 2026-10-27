package game

import (
	"strings"
	"testing"

	"rotmud/pkg/types"
)

// Item values are copper; lore shows them in coin denominations.
func TestLoreShowsCoinValue(t *testing.T) {
	d := NewCommandDispatcher()
	var out string
	d.Output = func(ch *types.Character, msg string) { out += msg }
	ch := types.NewCharacter("Test")
	ch.PCData = &types.PCData{Learned: map[string]int{"lore": 100}}
	sword := types.NewObject(3021, "a small sword", types.ItemTypeWeapon)
	sword.Name = "sword small"
	sword.Cost = 250
	ch.Inventory = append(ch.Inventory, sword)

	d.cmdLore(ch, "sword")
	if !strings.Contains(out, "Value 2g 5s.") {
		t.Errorf("lore value not in coins: %q", out)
	}
}
