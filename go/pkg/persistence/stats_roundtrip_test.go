package persistence

import (
	"testing"

	"rotmud/pkg/types"
)

// Worn items survive save/load with their bonuses and armour class applied
// exactly once; spell affects are dispelled; nothing drifts across reloads.
func TestStatsRoundTrip(t *testing.T) {
	p := NewPlayerPersistence(t.TempDir())

	ch := types.NewCharacter("Tester")
	ch.PCData = &types.PCData{Password: "x", Learned: map[string]int{}}
	ch.Level, ch.HitRoll, ch.DamRoll, ch.MaxHit, ch.Hit = 10, 5, 5, 100, 100
	for i := range ch.Armor {
		ch.Armor[i] = 100
	}

	plate := types.NewObject(3000, "a breastplate", types.ItemTypeArmor)
	plate.Values = [5]int{4, 4, 4, 4, 0}
	plate.Affects.Add(&types.Affect{Type: "object", Duration: -1, Location: types.ApplyHitroll, Modifier: 2})
	plate.Affects.Add(&types.Affect{Type: "object", Duration: -1, Location: types.ApplyStr, Modifier: 1})
	ch.Equip(plate, types.WearLocBody)
	ch.AddAffect(types.NewAffect("giant strength", 10, 10, types.ApplyDamroll, 3, 0))

	check := func(when string, c *types.Character, wantDam int) {
		t.Helper()
		if c.HitRoll != 7 || c.DamRoll != wantDam || c.ModStats[types.StatStr] != 1 || c.Armor[0] != 100-3*4 || c.MaxHit != 100 {
			t.Fatalf("%s: hit %d dam %d str+%d ac %d maxhit %d", when, c.HitRoll, c.DamRoll, c.ModStats[types.StatStr], c.Armor[0], c.MaxHit)
		}
	}
	check("live", ch, 8)

	for i := range 2 { // two round trips: no drift
		if err := p.SavePlayer(ch); err != nil {
			t.Fatal(err)
		}
		loaded, err := p.LoadPlayer("Tester")
		if err != nil {
			t.Fatal(err)
		}
		check("reload", loaded, 5) // giant strength dispelled on save
		if loaded.Affected.Len() != 0 {
			t.Fatalf("reload %d kept %d spell affects", i, loaded.Affected.Len())
		}
		ch = loaded
	}

	ch.Unequip(types.WearLocBody)
	if ch.HitRoll != 5 || ch.Armor[0] != 100 || ch.ModStats[types.StatStr] != 0 {
		t.Fatalf("after remove: hit %d ac %d str+%d", ch.HitRoll, ch.Armor[0], ch.ModStats[types.StatStr])
	}
}
