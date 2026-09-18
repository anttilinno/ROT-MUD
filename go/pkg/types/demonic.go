package types

import "strings"

// Demonic armour: GodWars' 12-slot demon armour set (.planning/TIER2-GODWARS.md T1).
// One template per slot lives in data/areas/relic; the tier is per instance.
const (
	VnumDemonicFirst = 29650 // ring
	VnumDemonicLast  = 29661 // visor
)

// Demonic armour tiers, cheapest first.
const (
	DemonBlack = iota
	DemonGrey
	DemonPurple
	DemonRed
	DemonBrass
)

// DemonTierNames maps tier to the colour used in item names.
var DemonTierNames = [...]string{"black", "grey", "purple", "red", "brass"}

// demonFizzlePerPiece is each worn piece's chance (%) that a hostile spell
// fizzles on the wearer. GodWars: purple 3, red 4, brass 5; full brass = 60%.
var demonFizzlePerPiece = [...]int{0, 0, 3, 4, 5}

// DemonDamrollPerPiece is the damroll each worn piece gives a demon.
// ponytail: flat per piece like GodWars' damcap bonus; tune once the E1 sim exists.
const DemonDamrollPerPiece = 2

// IsDemonic reports whether obj is a piece of demonic armour.
func IsDemonic(obj *Object) bool {
	return obj != nil && obj.Vnum >= VnumDemonicFirst && obj.Vnum <= VnumDemonicLast
}

// DemonicSet returns how many demonic pieces a demon is wearing and the
// resulting hostile-spell fizzle chance. Computed from worn equipment on
// every call, so it cannot drift the way GodWars' running counter did.
// Anyone who is not a demon gets nothing.
func (ch *Character) DemonicSet() (pieces, fizzle int) {
	if ch.IsNPC() || ch.Class != ClassDemon {
		return 0, 0
	}
	for _, obj := range ch.Equipment {
		if !IsDemonic(obj) {
			continue
		}
		pieces++
		if obj.Tier >= 0 && obj.Tier < len(demonFizzlePerPiece) {
			fizzle += demonFizzlePerPiece[obj.Tier]
		}
	}
	return pieces, fizzle
}

// ForgeDemonic turns a freshly created demonic template into a tiered piece
// bound to owner: "a demonic ring" becomes "a purple demonic ring".
func ForgeDemonic(obj *Object, tier int, owner string) {
	colour := DemonTierNames[tier]
	obj.Tier = tier
	obj.Owner = owner
	obj.Name = colour + " " + obj.Name
	obj.ShortDesc = strings.Replace(obj.ShortDesc, "demonic", colour+" demonic", 1)
	obj.LongDesc = strings.Replace(obj.LongDesc, "demonic", colour+" demonic", 1)
}

// Forging costs at the demon lord (T3). GodWars charged 5000 power plus
// quest points per tier; ROT has no quest points, so higher tiers cost coin.
// Sacrificing a piece refunds the power, never the coin.
const DemonArmourPower = 2000

// DemonArmourCoin is the coin (copper) each tier costs to forge.
var DemonArmourCoin = [...]int64{0, 10 * CopperPerGold, 30 * CopperPerGold, 60 * CopperPerGold, 100 * CopperPerGold}

// DemonicSlots maps the slot names used by `demonarmour` to template vnums.
var DemonicSlots = map[string]int{
	"ring": 29650, "collar": 29651, "plate": 29652, "helmet": 29653,
	"leggings": 29654, "boots": 29655, "gauntlets": 29656, "sleeves": 29657,
	"cape": 29658, "belt": 29659, "bracer": 29660, "visor": 29661,
}
