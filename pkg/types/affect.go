package types

// Affect represents a temporary effect on a character
// Based on AFFECT_DATA from merc.h:566-577
type Affect struct {
	Type         string      // Spell/skill name
	Level        int         // Caster level
	Duration     int         // Ticks remaining (-1 for permanent)
	Location     ApplyType   // What stat to modify
	Modifier     int         // How much to modify
	BitVector    AffectFlags // Flags to set (e.g., AffSanctuary)
	ShieldVector ShieldFlags // Shield flags to set (e.g., ShdFire, ShdIce)
}

// NewAffect creates a new affect
func NewAffect(spellType string, level, duration int, location ApplyType, modifier int, bits AffectFlags) *Affect {
	return &Affect{
		Type:      spellType,
		Level:     level,
		Duration:  duration,
		Location:  location,
		Modifier:  modifier,
		BitVector: bits,
	}
}

// IsExpired returns true if the affect has expired
// Permanent affects (duration -1) never expire
func (a *Affect) IsExpired() bool {
	return a.Duration == 0
}

// IsPermanent returns true if this is a permanent affect
func (a *Affect) IsPermanent() bool {
	return a.Duration == -1
}

// Tick decrements the duration by one tick
// Does nothing for permanent affects (duration -1) or already expired affects
func (a *Affect) Tick() {
	if a.Duration > 0 {
		a.Duration--
	}
}

// AffectList manages a list of affects
type AffectList struct {
	affects []*Affect
}

// Add adds an affect to the list
func (l *AffectList) Add(aff *Affect) {
	l.affects = append(l.affects, aff)
}

// Remove removes a specific affect from the list
func (l *AffectList) Remove(aff *Affect) {
	for i, a := range l.affects {
		if a == aff {
			l.affects = append(l.affects[:i], l.affects[i+1:]...)
			return
		}
	}
}

// RemoveByType removes all affects of a given type
func (l *AffectList) RemoveByType(spellType string) {
	filtered := l.affects[:0]
	for _, a := range l.affects {
		if a.Type != spellType {
			filtered = append(filtered, a)
		}
	}
	l.affects = filtered
}

// FindByType returns the first affect of a given type, or nil
func (l *AffectList) FindByType(spellType string) *Affect {
	for _, a := range l.affects {
		if a.Type == spellType {
			return a
		}
	}
	return nil
}

// HasType returns true if an affect of the given type exists
func (l *AffectList) HasType(spellType string) bool {
	return l.FindByType(spellType) != nil
}

// Len returns the number of affects
func (l *AffectList) Len() int {
	return len(l.affects)
}

// All returns all affects (for iteration)
func (l *AffectList) All() []*Affect {
	return l.affects
}

// TickAll decrements duration on all affects and removes expired ones
// Returns a slice of affects that just expired (for messaging)
func (l *AffectList) TickAll() []*Affect {
	var expired []*Affect
	remaining := l.affects[:0]

	for _, a := range l.affects {
		a.Tick()
		if a.IsExpired() {
			expired = append(expired, a)
		} else {
			remaining = append(remaining, a)
		}
	}

	l.affects = remaining
	return expired
}

// GetModifier returns the total modifier for a given apply location
func (l *AffectList) GetModifier(location ApplyType) int {
	total := 0
	for _, a := range l.affects {
		if a.Location == location {
			total += a.Modifier
		}
	}
	return total
}

// GetBitVector returns all affect flags combined
func (l *AffectList) GetBitVector() AffectFlags {
	var bits AffectFlags
	for _, a := range l.affects {
		bits |= a.BitVector
	}
	return bits
}

// GetShieldVector returns all shield flags combined
func (l *AffectList) GetShieldVector() ShieldFlags {
	var bits ShieldFlags
	for _, a := range l.affects {
		bits |= a.ShieldVector
	}
	return bits
}

// Clear removes all affects
func (l *AffectList) Clear() {
	l.affects = nil
}
