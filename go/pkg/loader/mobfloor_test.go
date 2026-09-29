package loader

import (
	"testing"

	"rotmud/pkg/types"
)

func TestApplyMobFloor(t *testing.T) {
	weak := types.NewNPC(1, "weak", 100)
	weak.MaxHit, weak.Damage = 100, [3]int{1, 4, 0}
	applyMobFloor(weak)
	if hp := anchorAt(mobHPFloor, 100); weak.MaxHit != hp || weak.Hit != hp {
		t.Errorf("weak L100 mob hp %d, want floor %d", weak.MaxHit, hp)
	}
	if got, want := float64(mobAvgDamage(weak))*mobSwings(weak), float64(anchorAt(mobDamFloor, 100)); got < want-1 {
		t.Errorf("weak L100 mob deals %.0f a round, want floor %.0f", got, want)
	}
	if weak.HitRoll != anchorAt(mobHitrollFloor, 100) || weak.Armor[0] != anchorAt(mobACFloor, 100) {
		t.Errorf("weak L100 mob hitroll %d ac %d not at floor", weak.HitRoll, weak.Armor[0])
	}

	strong := types.NewNPC(2, "strong", 80)
	strong.MaxHit, strong.HitRoll = 50000, 99
	applyMobFloor(strong)
	if strong.MaxHit != 50000 || strong.HitRoll != 99 {
		t.Error("mobs above the floor must keep their own values")
	}

	low := types.NewNPC(3, "low", 20)
	low.Armor = [4]int{50, 50, 50, 50}
	applyMobFloor(low)
	if low.Armor[0] != 50 {
		t.Error("AC floor starts at level 60")
	}

	brute := types.NewNPC(4, "brute", 20)
	brute.Damage = [3]int{10, 10, 20} // 75 a hit
	applyMobFloor(brute)
	ceil := float64(max(anchorAt(mobDamFloor, 20), anchorAt(mobDamCeiling, 20)))
	if got := float64(mobAvgDamage(brute)) * mobSwings(brute); got > ceil {
		t.Errorf("L20 brute deals %.0f a round, over the ceiling %.0f", got, ceil)
	}
}
