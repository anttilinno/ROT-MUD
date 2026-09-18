package loader

import "testing"

func TestCapGear(t *testing.T) {
	o := &ObjectData{Level: 100, Armor: &ArmorData{ACPierce: 50, ACBash: 5}, Affects: []AffectData{
		{"damroll", 500}, {"hitroll", -10}, {"hit", 300}, {"ac", -100}, {"str", 6}, {"saves", -20},
	}}
	capGear(o)
	want := []int{4, -10, 35, -5, 2, -20} // damroll/hp/ac/str capped; penalty and uncapped saves kept
	for i, a := range o.Affects {
		if a.Modifier != want[i] {
			t.Errorf("%s = %d, want %d", a.Location, a.Modifier, want[i])
		}
	}
	if o.Armor.ACPierce != 11 || o.Armor.ACBash != 5 {
		t.Errorf("armor = %d/%d, want 11/5", o.Armor.ACPierce, o.Armor.ACBash)
	}

	low := &ObjectData{Level: 59, Affects: []AffectData{{"damroll", 10}}}
	capGear(low)
	if low.Affects[0].Modifier != 10 {
		t.Error("items below level 60 must not be capped")
	}
}
