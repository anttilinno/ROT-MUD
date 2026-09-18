package types

import (
	"errors"
	"testing"
)

func TestRerollAscend(t *testing.T) {
	newHero := func(race int) *Character {
		ch := NewCharacter("Hero")
		ch.PCData = &PCData{}
		ch.Race, ch.Class, ch.Level, ch.Exp = race, ClassCleric, LevelHero, 100000
		return ch
	}

	ch := newHero(RaceHuman)
	if err := ch.Ascend(ClassDemon); !errors.Is(err, ErrNotRerolled) {
		t.Fatalf("ascend before reroll: got %v", err)
	}
	ch.Level = LevelHero - 1
	if err := ch.Reroll(); !errors.Is(err, ErrNotHero) {
		t.Fatalf("reroll below hero: got %v", err)
	}
	ch.Level = LevelHero
	if err := ch.Reroll(); err != nil {
		t.Fatal(err)
	}
	if err := ch.Ascend(ClassWarrior); !errors.Is(err, ErrNotTier2) {
		t.Fatalf("ascend to tier 1: got %v", err)
	}
	if err := ch.Ascend(ClassDemon); err != nil {
		t.Fatal(err)
	}
	if ch.Class != ClassDemon || ch.Level != 1 || ch.Exp != 0 || ch.PCData.Tier != TierAscended {
		t.Fatalf("after ascend: class=%d level=%d exp=%d tier=%d", ch.Class, ch.Level, ch.Exp, ch.PCData.Tier)
	}
	if ch.SkillClass() != ClassCleric {
		t.Fatalf("SkillClass = %d, want origin cleric", ch.SkillClass())
	}
	if err := ch.Reroll(); !errors.Is(err, ErrAlreadyTier2) {
		t.Fatalf("reroll a demon: got %v", err)
	}

	undead := newHero(RaceHeucuva)
	if err := undead.Reroll(); err != nil {
		t.Fatal(err)
	}
	if err := undead.Ascend(ClassDemon); !errors.Is(err, ErrRaceBarred) {
		t.Fatalf("heucuva demon pact: got %v", err)
	}
}
