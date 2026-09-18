package types

import (
	"errors"
	"testing"
)

func TestPowers(t *testing.T) {
	ch := NewCharacter("Azazel")
	ch.PCData = &PCData{}
	ch.Class = ClassDemon
	ch.PermStats[StatStr] = 15

	if _, err := ch.BuyPower("claws"); !errors.Is(err, ErrPowerCost) {
		t.Fatalf("buy with no power: %v", err)
	}
	ch.GainPower(20000)
	if _, err := ch.BuyPower("might"); !errors.Is(err, ErrPowerRequires) {
		t.Fatalf("might before claws: %v", err)
	}
	if _, err := ch.BuyPower("celerity"); !errors.Is(err, ErrUnknownPower) {
		t.Fatalf("vampire power on demon: %v", err)
	}
	for _, k := range []string{"claws", "might", "wings", "toughness"} {
		if _, err := ch.BuyPower(k); err != nil {
			t.Fatalf("buy %s: %v", k, err)
		}
	}
	if _, err := ch.BuyPower("wings"); !errors.Is(err, ErrPowerOwned) {
		t.Fatalf("rebuy: %v", err)
	}
	if want := 20000 - 2500 - 7500 - 1000 - 7500; ch.PCData.Power != want || ch.PCData.PowerTotal != 20000 {
		t.Fatalf("power = %d/%d, want %d/20000", ch.PCData.Power, ch.PCData.PowerTotal, want)
	}

	if !ch.IsAffected(AffFlying) || ch.IsAffected(AffHaste) {
		t.Error("wings should grant flying and nothing else")
	}
	if got := ch.GetStat(StatStr); got != 17 {
		t.Errorf("str = %d, want 15+2 from might", got)
	}
	if hit, dam := ch.PowerBonus(); hit != 0 || dam != 3+2 {
		t.Errorf("PowerBonus = %d/%d, want 0/5", hit, dam)
	}
	res, vuln := ch.InnateRIS()
	if !res.Has(ImmSlash) || !res.Has(ImmFire) || !vuln.Has(ImmHoly) {
		t.Errorf("RIS = %v/%v, want toughness + demon fire res, holy vuln", res, vuln)
	}

	// A battle form's bonus exists only while its timer affect does.
	ch.GainPower(25000)
	if _, err := ch.BuyPower("demonform"); err != nil {
		t.Fatal(err)
	}
	ch.AddAffect(NewAffect("demonform", 1, FormDuration, ApplyNone, 0, 0))
	if hit, dam := ch.PowerBonus(); hit != FormHitroll || dam != 5+FormDamroll {
		t.Errorf("in demonform PowerBonus = %d/%d", hit, dam)
	}

	mortal := NewCharacter("Bob")
	mortal.PCData = &PCData{}
	if mortal.GainPower(100) || mortal.PCData.Power != 0 {
		t.Error("tier-1 characters must not earn power")
	}
}

func TestRiteMatches(t *testing.T) {
	mob := NewNPC(1, "paladin", 60)
	mob.Alignment = 800
	demon := ClassTable[ClassDemon].Tier2.Rite
	if !demon.Matches(mob, false) {
		t.Error("good L60 mob should count for the demon pact")
	}
	mob.Alignment = 0
	if demon.Matches(mob, false) {
		t.Error("neutral mob counted for the demon pact")
	}
	vamp := ClassTable[ClassVampire].Tier2.Rite
	if vamp.Matches(mob, false) || !vamp.Matches(mob, true) {
		t.Error("vampire rite must count only at night")
	}
	mob.Level = 10
	if vamp.Matches(mob, true) {
		t.Error("low-level mob counted")
	}
}

func TestVampireNeedsGhoul(t *testing.T) {
	ch := NewCharacter("Hero")
	ch.PCData = &PCData{Tier: TierRerolled}
	ch.Race, ch.Class = RaceHuman, ClassWarrior
	if err := ch.Ascend(ClassVampire); !errors.Is(err, ErrOriginBarred) {
		t.Fatalf("warrior embraced: %v", err)
	}
	ch.Class = ClassGhoul
	if err := ch.Ascend(ClassVampire); err != nil {
		t.Fatal(err)
	}
}
