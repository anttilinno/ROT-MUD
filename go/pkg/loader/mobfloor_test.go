package loader

import (
	"testing"

	"rotmud/pkg/types"
)

func TestApplyMobFloor(t *testing.T) {
	weak := types.NewNPC(1, "weak", 100)
	weak.MaxHit, weak.Damage = 100, [3]int{1, 4, 0}
	applyMobFloor(weak)
	f := mobFloorAt(100)
	avg := weak.Damage[0]*(weak.Damage[1]+1)/2 + weak.Damage[2]
	if weak.MaxHit != f.hp || weak.Hit != f.hp || avg != f.dam || weak.HitRoll != f.hitroll || weak.Armor[0] != f.ac {
		t.Fatalf("weak L100 mob not raised to floor: hp %d dam %d hit %d ac %d", weak.MaxHit, avg, weak.HitRoll, weak.Armor[0])
	}

	strong := types.NewNPC(2, "strong", 80)
	strong.MaxHit, strong.HitRoll = 50000, 99
	applyMobFloor(strong)
	if strong.MaxHit != 50000 || strong.HitRoll != 99 {
		t.Fatal("mobs above the floor must keep their own values")
	}

	low := types.NewNPC(3, "low", 59)
	low.MaxHit = 10
	applyMobFloor(low)
	if low.MaxHit != 10 {
		t.Fatal("mobs below level 60 must not be floored")
	}
}
