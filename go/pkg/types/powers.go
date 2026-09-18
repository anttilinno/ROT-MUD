package types

import (
	"errors"
	"fmt"
)

// Power is a GodWars-style power a tier-2 class buys with its currency
// (.planning/TIER2-GODWARS.md T2/T4). Passive effects are derived from
// PCData.Powers at query time — IsAffected, GetStat, PowerBonus, InnateRIS —
// so nothing is stored as an affect and nothing can drift across save/load.
type Power struct {
	Key      string // name used by `powers <key>`
	Class    int    // tier-2 class that can buy it
	Cost     int    // in the class's currency
	Requires string // power that must be owned first ("" = none)
	Desc     string

	Affects AffectFlags // permanent affect flags
	Res     ImmFlags    // resistances
	Stat    int         // stat raised by StatMod (ignored when StatMod == 0)
	StatMod int
	Hitroll int
	Damroll int
	Command string // active power: the command it unlocks (travel, inferno, demonform, ...)
	Form    bool   // Command toggles a timed battle form (see FormHitroll/FormDamroll)
}

// Battle-form bonus while a form's timer affect (Type == Command) is active.
const (
	FormHitroll  = 5
	FormDamroll  = 10
	FormDuration = 8 // ticks
)

const weapons = ImmBash | ImmPierce | ImmSlash

// PowerTable lists every tier-2 power. Costs follow GodWars' inpart prices;
// the currency comes from kills (victim level each), see game/powers.go.
var PowerTable = []Power{
	// Demon — GodWars inpart.
	{Key: "fangs", Class: ClassDemon, Cost: 2500, Hitroll: 2, Desc: "Fangs that find the throat."},
	{Key: "claws", Class: ClassDemon, Cost: 2500, Damroll: 3, Desc: "Talons that rend flesh."},
	{Key: "wings", Class: ClassDemon, Cost: 1000, Affects: AffFlying, Desc: "Leathery wings; you fly."},
	{Key: "nightsight", Class: ClassDemon, Cost: 3000, Affects: AffInfrared | AffDarkVision, Desc: "See through any darkness."},
	{Key: "shadowsight", Class: ClassDemon, Cost: 7500, Affects: AffDetectHidden | AffDetectInvis, Desc: "Nothing hides from you."},
	{Key: "might", Class: ClassDemon, Cost: 7500, Requires: "claws", Stat: StatStr, StatMod: 2, Damroll: 2, Desc: "Infernal strength."},
	{Key: "toughness", Class: ClassDemon, Cost: 7500, Res: weapons, Desc: "Hide that turns blades and blows."},
	{Key: "speed", Class: ClassDemon, Cost: 7500, Affects: AffHaste, Desc: "Unholy speed."},
	{Key: "travel", Class: ClassDemon, Cost: 1500, Command: "travel", Desc: "Step through hell to any mortal (travel <player>)."},
	{Key: "inferno", Class: ClassDemon, Cost: 10000, Command: "inferno", Desc: "Engulf your foes in hellfire."},
	{Key: "demonform", Class: ClassDemon, Cost: 25000, Requires: "might", Command: "demonform", Form: true, Desc: "Become a great demonic beast for a time."},

	// Vampire — disciplines.
	{Key: "celerity", Class: ClassVampire, Cost: 7500, Affects: AffHaste, Desc: "Preternatural speed."},
	{Key: "fortitude", Class: ClassVampire, Cost: 7500, Res: weapons, Desc: "Dead flesh shrugs off blows."},
	{Key: "potence", Class: ClassVampire, Cost: 7500, Stat: StatStr, StatMod: 2, Damroll: 2, Desc: "Unnatural strength."},
	{Key: "auspex", Class: ClassVampire, Cost: 5000, Affects: AffDetectHidden | AffDetectInvis, Desc: "Heightened senses."},
	{Key: "protean", Class: ClassVampire, Cost: 3000, Affects: AffInfrared | AffDarkVision, Damroll: 2, Desc: "Beast's eyes and claws."},
	{Key: "obfuscate", Class: ClassVampire, Cost: 5000, Affects: AffSneak, Desc: "Move unseen and unheard."},
	{Key: "mist", Class: ClassVampire, Cost: 1500, Command: "travel", Desc: "Travel as mist to any mortal (travel <player>)."},
	{Key: "bloodrage", Class: ClassVampire, Cost: 20000, Requires: "potence", Command: "bloodrage", Form: true, Desc: "Let the Beast rise for a time."},

	// Werewolf — gifts.
	{Key: "claws", Class: ClassWerewolf, Cost: 2500, Damroll: 3, Desc: "Razor claws."},
	{Key: "nightsight", Class: ClassWerewolf, Cost: 3000, Affects: AffInfrared | AffDarkVision, Desc: "Wolf's eyes."},
	{Key: "senses", Class: ClassWerewolf, Cost: 5000, Affects: AffDetectHidden | AffDetectInvis, Desc: "Scent and hearing betray the hidden."},
	{Key: "might", Class: ClassWerewolf, Cost: 7500, Requires: "claws", Stat: StatStr, StatMod: 2, Damroll: 2, Desc: "Strength of the pack."},
	{Key: "toughness", Class: ClassWerewolf, Cost: 7500, Res: weapons, Desc: "Hide like iron."},
	{Key: "speed", Class: ClassWerewolf, Cost: 7500, Affects: AffHaste, Desc: "Speed of the hunt."},
	{Key: "moonbridge", Class: ClassWerewolf, Cost: 1500, Command: "travel", Desc: "Walk the moonbridge to any mortal (travel <player>)."},
	{Key: "crinos", Class: ClassWerewolf, Cost: 20000, Requires: "might", Command: "crinos", Form: true, Desc: "Take the war form for a time."},

	// Magus — spheres.
	{Key: "forces", Class: ClassMagus, Cost: 7500, Res: ImmFire | ImmCold | ImmLightning, Desc: "Command over the elements."},
	{Key: "life", Class: ClassMagus, Cost: 7500, Affects: AffRegeneration, Desc: "Your flesh knits itself."},
	{Key: "mind", Class: ClassMagus, Cost: 5000, Affects: AffDetectHidden | AffDetectInvis, Desc: "Perceive every mind nearby."},
	{Key: "prime", Class: ClassMagus, Cost: 7500, Stat: StatInt, StatMod: 2, Hitroll: 2, Desc: "Raw quintessence sharpens you."},
	{Key: "correspondence", Class: ClassMagus, Cost: 1500, Command: "travel", Desc: "Fold space to any mortal (travel <player>)."},
	{Key: "forcebolt", Class: ClassMagus, Cost: 10000, Requires: "forces", Command: "forcebolt", Desc: "Hurl raw force at every foe."},

	// Highlander — katas.
	{Key: "bladesense", Class: ClassHighlander, Cost: 5000, Hitroll: 4, Desc: "Your blade finds its mark."},
	{Key: "speed", Class: ClassHighlander, Cost: 7500, Affects: AffHaste, Desc: "Swordmaster's speed."},
	{Key: "might", Class: ClassHighlander, Cost: 7500, Stat: StatStr, StatMod: 2, Damroll: 2, Desc: "Strength of a thousand duels."},
	{Key: "toughness", Class: ClassHighlander, Cost: 7500, Res: weapons, Desc: "Immortal flesh."},
	{Key: "quickening", Class: ClassHighlander, Cost: 20000, Requires: "bladesense", Command: "quickening", Form: true, Desc: "Unleash the quickening for a time."},

	// Angel — mirror of demon.
	{Key: "wings", Class: ClassAngel, Cost: 1000, Affects: AffFlying, Desc: "Feathered wings; you fly."},
	{Key: "nightsight", Class: ClassAngel, Cost: 3000, Affects: AffInfrared | AffDarkVision, Desc: "Heaven's light in your eyes."},
	{Key: "truesight", Class: ClassAngel, Cost: 7500, Affects: AffDetectHidden | AffDetectInvis, Desc: "Nothing hides from heaven."},
	{Key: "might", Class: ClassAngel, Cost: 7500, Stat: StatStr, StatMod: 2, Damroll: 2, Desc: "Strength of the host."},
	{Key: "toughness", Class: ClassAngel, Cost: 7500, Res: weapons, Desc: "Skin like hammered gold."},
	{Key: "speed", Class: ClassAngel, Cost: 7500, Affects: AffHaste, Desc: "Swift as a prayer."},
	{Key: "travel", Class: ClassAngel, Cost: 1500, Command: "travel", Desc: "Descend to any mortal (travel <player>)."},
	{Key: "smite", Class: ClassAngel, Cost: 10000, Command: "smite", Desc: "Call holy fire on every foe."},
	{Key: "angelform", Class: ClassAngel, Cost: 25000, Requires: "might", Command: "angelform", Form: true, Desc: "Reveal your true glory for a time."},
}

// FindPower returns the class's power with the given key, or nil.
func FindPower(class int, key string) *Power {
	for i := range PowerTable {
		if PowerTable[i].Class == class && PowerTable[i].Key == key {
			return &PowerTable[i]
		}
	}
	return nil
}

// HasPower reports whether the character owns the power with this key.
func (ch *Character) HasPower(key string) bool {
	return ch.PCData != nil && ch.PCData.Powers[key]
}

// CommandPower returns the owned power that unlocks command, or nil.
func (ch *Character) CommandPower(command string) *Power {
	for i := range PowerTable {
		p := &PowerTable[i]
		if p.Class == ch.Class && p.Command == command && ch.HasPower(p.Key) {
			return p
		}
	}
	return nil
}

// ownedPowers calls fn for each power the character owns in its current class.
// ponytail: linear scan of PowerTable (~45 rows) per query; mortals exit on the
// empty map. Cache a per-character summary if profiling ever shows it.
func (ch *Character) ownedPowers(fn func(*Power)) {
	if ch.PCData == nil || len(ch.PCData.Powers) == 0 {
		return
	}
	for i := range PowerTable {
		if p := &PowerTable[i]; p.Class == ch.Class && ch.PCData.Powers[p.Key] {
			fn(p)
		}
	}
}

// powerAffects is the OR of affect flags granted by owned powers.
func (ch *Character) powerAffects() (flags AffectFlags) {
	ch.ownedPowers(func(p *Power) { flags |= p.Affects })
	return flags
}

// powerStat is the total stat bonus from owned powers.
func (ch *Character) powerStat(stat int) (mod int) {
	ch.ownedPowers(func(p *Power) {
		if p.StatMod != 0 && p.Stat == stat {
			mod += p.StatMod
		}
	})
	return mod
}

// PowerBonus is the hitroll and damroll granted by owned powers.
func (ch *Character) PowerBonus() (hit, dam int) {
	ch.ownedPowers(func(p *Power) {
		hit += p.Hitroll
		dam += p.Damroll
		if p.Form && ch.Affected.HasType(p.Command) {
			hit += FormHitroll
			dam += FormDamroll
		}
	})
	return hit, dam
}

// InnateRIS returns resistances and vulnerabilities a player gets from their
// tier-2 class and owned powers, on top of Character.Res/Vuln.
func (ch *Character) InnateRIS() (res, vuln ImmFlags) {
	if c := GetClass(ch.Class); c != nil && c.Tier2 != nil && !ch.IsNPC() {
		res, vuln = c.Tier2.Res, c.Tier2.Vuln
	}
	ch.ownedPowers(func(p *Power) { res |= p.Res })
	return res, vuln
}

// Power purchase errors.
var (
	ErrNoPowers      = errors.New("you have no supernatural powers to develop")
	ErrUnknownPower  = errors.New("there is no such power")
	ErrPowerOwned    = errors.New("you already have that power")
	ErrPowerRequires = errors.New("you must first master")
	ErrPowerCost     = errors.New("you cannot afford that")
)

// BuyPower spends class currency on the class power named key.
func (ch *Character) BuyPower(key string) (*Power, error) {
	if ch.PCData == nil || !IsTier2Class(ch.Class) {
		return nil, ErrNoPowers
	}
	p := FindPower(ch.Class, key)
	switch {
	case p == nil:
		return nil, ErrUnknownPower
	case ch.HasPower(key):
		return p, ErrPowerOwned
	case p.Requires != "" && !ch.HasPower(p.Requires):
		return p, fmt.Errorf("%w %s", ErrPowerRequires, p.Requires)
	case ch.PCData.Power < p.Cost:
		return p, ErrPowerCost
	}
	ch.PCData.Power -= p.Cost
	if ch.PCData.Powers == nil {
		ch.PCData.Powers = make(map[string]bool)
	}
	ch.PCData.Powers[key] = true
	return p, nil
}

// GainPower credits class currency to a tier-2 player; others get nothing.
func (ch *Character) GainPower(amount int) bool {
	if amount <= 0 || ch.PCData == nil || !IsTier2Class(ch.Class) {
		return false
	}
	ch.PCData.Power += amount
	ch.PCData.PowerTotal += amount
	return true
}
