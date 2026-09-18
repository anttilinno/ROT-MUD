package types

import "errors"

// Tier values stored in PCData.Tier.
const (
	TierMortal   = 0 // tier-1 class, never rerolled
	TierRerolled = 1 // hero who rerolled; waiting to complete a tier-2 rite
	TierAscended = 2 // tier-2 class
)

// Reroll errors.
var (
	ErrNotHero       = errors.New("you must be a hero to reroll")
	ErrAlreadyTier2  = errors.New("you have already left mortal classes behind")
	ErrAlreadyReroll = errors.New("you have already rerolled; complete a rite to ascend")
)

// Ascend errors.
var (
	ErrNotRerolled  = errors.New("they have not rerolled")
	ErrNotTier2     = errors.New("that is not a tier 2 class")
	ErrRaceBarred   = errors.New("their race may not take that rite")
	ErrOriginBarred = errors.New("their class may not take that rite")
	ErrNoPlayerData = errors.New("no player data")
)

// Reroll marks a tier-1 hero as eligible for a tier-2 rite. Nothing else
// changes until the rite calls Ascend.
func (ch *Character) Reroll() error {
	switch {
	case ch.PCData == nil:
		return ErrNoPlayerData
	case IsTier2Class(ch.Class):
		return ErrAlreadyTier2
	case ch.PCData.Tier == TierRerolled:
		return ErrAlreadyReroll
	case ch.Level < LevelHero:
		return ErrNotHero
	}
	ch.PCData.Tier = TierRerolled
	return nil
}

// Ascend completes a tier-2 rite: the origin class is remembered for skill
// lookups (see SkillClass), the class changes, and the character restarts at
// level 1 with starting vitals. Learned skills are kept.
func (ch *Character) Ascend(class int) error {
	switch {
	case ch.PCData == nil:
		return ErrNoPlayerData
	case ch.PCData.Tier != TierRerolled:
		return ErrNotRerolled
	case !IsTier2Class(class):
		return ErrNotTier2
	case !ClassTable[class].RaceAllowed(ch.Race):
		return ErrRaceBarred
	case !ClassTable[class].OriginAllowed(ch.Class):
		return ErrOriginBarred
	}
	ch.PCData.Classes = []int{ch.Class}
	ch.Class = class
	ch.PCData.Tier = TierAscended
	ch.PCData.Power, ch.PCData.PowerTotal, ch.PCData.Powers = 0, 0, nil
	ch.PCData.Rite, ch.PCData.RiteKills = 0, 0
	ch.Level = 1
	ch.Exp = 0
	ch.SetStartingVitals()
	return nil
}

// SetStartingVitals sets level-1 max hit/mana/move for the character's class
// and fills the current pools.
func (ch *Character) SetStartingVitals() {
	class := GetClass(ch.Class)
	if class == nil {
		class = GetClass(ClassWarrior)
	}
	ch.MaxHit = class.HPMax + ch.GetStat(StatCon)
	if class.FreesMana {
		ch.MaxMana = 100 + ch.GetStat(StatInt)*2
	} else {
		ch.MaxMana = 50
	}
	ch.MaxMove = 100 + ch.GetStat(StatCon) + ch.GetStat(StatDex)
	ch.Hit, ch.Mana, ch.Move = ch.MaxHit, ch.MaxMana, ch.MaxMove
}

// Matches reports whether killing victim counts toward the rite.
func (r Rite) Matches(victim *Character, night bool) bool {
	switch {
	case !victim.IsNPC(), victim.Level < r.MinLevel:
		return false
	case r.Night && !night:
		return false
	case r.ActFlag != 0 && !victim.Act.Has(r.ActFlag):
		return false
	case r.Align > 0 && victim.Alignment < 350: // ROM IS_GOOD threshold
		return false
	case r.Align < 0 && victim.Alignment > -350: // ROM IS_EVIL threshold
		return false
	}
	return true
}

// Saveable reports whether a player's progress is written on quit or
// disconnect. Fresh level-1 characters are dropped to avoid abandoned
// accounts; ascended tier-2 characters restart at level 1 and must persist.
func (ch *Character) Saveable() bool {
	return ch.Level > 1 || (ch.PCData != nil && ch.PCData.Tier > TierMortal)
}
