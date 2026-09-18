package loader

// Gear bonus caps for level 60+ items. World items above level 60 carried
// bonuses (median +5 hit/dam, up to +500) that made best-in-slot gear decide
// every high-level fight (combat_sim_world_test). Each positive bonus is
// clamped to a budget that continues the level 40-59 curve; penalties are
// left alone. Applied when templates load, so every created item obeys it.
const gearCapLevel = 60

// gearCap returns the largest bonus a level-lv item may carry at location
// loc, or 0 for "no cap".
func gearCap(loc string, lv int) int {
	step := (lv - gearCapLevel) / 20 // +1 per 20 levels above 60
	switch loc {
	case "hitroll", "damroll":
		return 2 + step
	case "hit", "mana", "move":
		return 25 + 5*step
	case "str", "dex", "int", "wis", "con":
		return 2
	case "ac": // negative is better
		return 3 + step
	}
	return 0
}

// capGear clamps a level 60+ item template's bonuses and armour values.
func capGear(o *ObjectData) {
	if o.Level < gearCapLevel {
		return
	}
	for i := range o.Affects {
		a := &o.Affects[i]
		c := gearCap(a.Location, o.Level)
		switch {
		case c == 0:
		case a.Location == "ac" && a.Modifier < -c:
			a.Modifier = -c
		case a.Location != "ac" && a.Modifier > c:
			a.Modifier = c
		}
	}
	if o.Armor != nil {
		maxAC := 9 + (o.Level-gearCapLevel)/20
		for _, v := range []*int{&o.Armor.ACPierce, &o.Armor.ACBash, &o.Armor.ACSlash, &o.Armor.ACExotic} {
			*v = min(*v, maxAC)
		}
	}
}
