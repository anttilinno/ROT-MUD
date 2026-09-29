package types

// Modify applies (sign +1) or reverses (sign -1) an affect's stat modifier.
func (ch *Character) Modify(af *Affect, sign int) {
	mod := af.Modifier * sign
	switch af.Location {
	case ApplyStr:
		ch.ModStats[StatStr] += mod
	case ApplyDex:
		ch.ModStats[StatDex] += mod
	case ApplyInt:
		ch.ModStats[StatInt] += mod
	case ApplyWis:
		ch.ModStats[StatWis] += mod
	case ApplyCon:
		ch.ModStats[StatCon] += mod
	case ApplyHit:
		ch.MaxHit += mod
	case ApplyMana:
		ch.MaxMana += mod
	case ApplyMove:
		ch.MaxMove += mod
	case ApplyAC:
		for i := range ch.Armor {
			ch.Armor[i] += mod
		}
	case ApplyHitroll:
		ch.HitRoll += mod
	case ApplyDamroll:
		ch.DamRoll += mod
	case ApplySaves:
		ch.Saves += mod
	}
}

// ArmorAC is how much armour obj worn at loc lowers each AC type (ROM
// apply_ac): body counts triple, head/legs/about double, rings and lights
// nothing.
func ArmorAC(obj *Object, loc WearLocation, acType int) int {
	if obj.ItemType != ItemTypeArmor || acType < 0 || acType > 3 {
		return 0
	}
	switch loc {
	case WearLocBody:
		return 3 * obj.Values[acType]
	case WearLocHead, WearLocLegs, WearLocAbout:
		return 2 * obj.Values[acType]
	case WearLocFeet, WearLocHands, WearLocArms, WearLocShield, WearLocNeck1, WearLocNeck2,
		WearLocWaist, WearLocWristL, WearLocWristR, WearLocHold, WearLocFace:
		return obj.Values[acType]
	}
	return 0
}

// itemEffects applies (sign +1) or removes (sign -1) what a worn item does
// to its wearer: its affects' modifiers and its armour class.
func (ch *Character) itemEffects(obj *Object, loc WearLocation, sign int) {
	for _, af := range obj.Affects.All() {
		ch.Modify(af, sign)
	}
	for i := range ch.Armor {
		ch.Armor[i] -= sign * ArmorAC(obj, loc, i)
	}
}

// ModifierSum totals the modifiers at loc from worn items and active affects,
// i.e. how far a live value sits above its base.
func (ch *Character) ModifierSum(loc ApplyType) (sum int) {
	for _, obj := range ch.Equipment {
		if obj == nil {
			continue
		}
		for _, af := range obj.Affects.All() {
			if af.Location == loc {
				sum += af.Modifier
			}
		}
	}
	for _, af := range ch.Affected.All() {
		if af.Location == loc {
			sum += af.Modifier
		}
	}
	return sum
}
