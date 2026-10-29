package game

import (
	"testing"

	"rotmud/pkg/loader"
	"rotmud/pkg/magic"
	"rotmud/pkg/types"
)

// Object values restored from ROT reach spawned objects in ROM's layout.
func TestSpawnedObjectValues(t *testing.T) {
	world, err := loader.NewAreaLoader("../../data/areas").LoadAll()
	if err != nil {
		t.Fatalf("load world: %v", err)
	}
	r := NewResetSystem(world)

	scroll := r.CreateObjFromTemplate(3042) // scroll of recall
	if scroll.Values[1] != magic.SlotWordOfRecall || scroll.Values[0] == 0 {
		t.Errorf("recall scroll values = %v, want level and word of recall slot", scroll.Values)
	}
	beer := r.CreateObjFromTemplate(3001)
	if beer.Values[1] == 0 || liquidTable[beer.Values[2]].Name != "beer" {
		t.Errorf("beer values = %v", beer.Values)
	}
	torch := r.CreateObjFromTemplate(3030)
	if torch.ItemType != types.ItemTypeLight || torch.Values[2] != 50 {
		t.Errorf("torch values = %v, want 50 hours", torch.Values)
	}
	throne := r.CreateObjFromTemplate(11615) // ROT divsouls.are: 2 2000 BEHKN 100 100
	if throne.ItemType != types.ItemTypeFurniture ||
		throne.Values != [5]int{2, 2000, types.StandOn | types.SitOn | types.RestOn | types.SleepOn | types.PutOn, 100, 100} {
		t.Errorf("furniture 11615 values = %v", throne.Values)
	}
}
