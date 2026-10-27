package game

import (
	"testing"
	"time"

	"rotmud/pkg/types"
)

func TestLevelPrice(t *testing.T) {
	for _, c := range []struct {
		level int
		want  int64
	}{{1, 50}, {10, 50}, {20, 200}, {50, 1250}, {100, 5000}} {
		if got := levelPrice(50, c.level); got != c.want {
			t.Errorf("levelPrice(50, %d) = %d, want %d", c.level, got, c.want)
		}
	}
}

// The healer's mana service restores 2d8 + level/3, not the whole pool.
func TestHealerManaIsPartial(t *testing.T) {
	d := NewCommandDispatcher()
	d.Output = func(ch *types.Character, msg string) {}
	room := types.NewRoom(1, "Temple", "x")
	healer := types.NewNPC(3707, "adept", 30)
	healer.Special = "spec_cast_adept"
	ch := types.NewCharacter("Test")
	ch.Level, ch.Coin, ch.Mana, ch.MaxMana = 5, 100, 0, 500
	CharToRoom(healer, room)
	CharToRoom(ch, room)

	d.cmdHeal(ch, "mana")
	if ch.Mana < 2+10 || ch.Mana > 16+10 {
		t.Errorf("mana = %d, want 2d8+10", ch.Mana)
	}
	if ch.Coin != 90 {
		t.Errorf("coin = %d, want 90 (10c at level 5)", ch.Coin)
	}
}

func TestTrainingFee(t *testing.T) {
	ch := types.NewCharacter("Test")
	ch.Level = 10
	if trainingFee(ch, 5) != 0 {
		t.Error("training should be free through level 10")
	}
	ch.Level = 20
	if trainingFee(ch, 1) != 400 || trainingFee(ch, 5) != 2000 {
		t.Errorf("fees at L20 = %d/%d, want 400/2000", trainingFee(ch, 1), trainingFee(ch, 5))
	}
}

// A rented room triples regen while resting there, until the rent runs out.
func TestInnRegenMultiplier(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	forgeNow = func() time.Time { return now }
	defer func() { forgeNow = time.Now }()

	room := types.NewRoom(3040, "Inn", "x")
	ch := types.NewCharacter("Test")
	ch.PCData = &types.PCData{InnRoom: 3040, InnUntil: now.Add(time.Minute).Unix()}
	CharToRoom(ch, room)

	ch.Position = types.PosStanding
	if innRegenMultiplier(ch) != 1 {
		t.Error("standing should not get the inn bonus")
	}
	ch.Position = types.PosSleeping
	if innRegenMultiplier(ch) != 3 {
		t.Error("sleeping in the rented room should triple regen")
	}
	now = now.Add(2 * time.Minute)
	if innRegenMultiplier(ch) != 1 {
		t.Error("expired rent should not give a bonus")
	}
}
