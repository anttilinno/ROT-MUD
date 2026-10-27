package game

import (
	"strings"
	"testing"
	"time"

	"rotmud/pkg/loader"
	"rotmud/pkg/types"
)

func TestCommissionAndCollect(t *testing.T) {
	d := NewCommandDispatcher()
	var out string
	d.Output = func(ch *types.Character, msg string) { out += msg }
	d.GameLoop = NewGameLoop()
	d.GameLoop.World = loader.NewWorld()
	d.GameLoop.World.Forge = &loader.ForgeConfig{ForgeDays: 14, Tiers: []loader.ForgeTier{
		{Name: "fine", Level: 5, DiceNumber: 2, DiceSize: 4, Price: 20},
		{Name: "legendary", Level: 75, DiceNumber: 6, DiceSize: 8, Hitroll: 5, Damroll: 5, Price: 15000},
	}}
	d.GameLoop.World.MobTemplates[3003] = &loader.MobileData{Vnum: 3003, ForgeTiers: []string{"fine"}}

	room := types.NewRoom(3020, "Weapon Shop", "x")
	smith := types.NewNPC(3003, "weaponsmith", 45)
	smith.ShortDesc = "the weaponsmith"
	ch := types.NewCharacter("Test")
	ch.PCData = &types.PCData{Learned: map[string]int{}}
	ch.Level, ch.Coin, ch.Position = 6, 100, types.PosStanding
	CharToRoom(smith, room)
	CharToRoom(ch, room)

	now := time.Unix(1_000_000, 0)
	forgeNow = func() time.Time { return now }
	defer func() { forgeNow = time.Now }()

	d.cmdCommission(ch, "dagger legendary")
	if ch.PCData.Forge != nil || !strings.Contains(out, "not work I do") {
		t.Fatalf("smith took a tier it doesn't offer: %q", out)
	}

	d.cmdCommission(ch, "dagger fine")
	if ch.PCData.Forge == nil || ch.Coin != 80 {
		t.Fatalf("order not placed: forge=%v coin=%d", ch.PCData.Forge, ch.Coin)
	}

	out = ""
	now = now.Add(13 * gameDay)
	d.cmdCollect(ch, "")
	if len(ch.Inventory) != 0 || !strings.Contains(out, "Come back in 1 day") {
		t.Fatalf("collected too early: %q", out)
	}

	now = now.Add(gameDay)
	d.cmdCollect(ch, "")
	if len(ch.Inventory) != 1 || ch.PCData.Forge != nil {
		t.Fatalf("collect failed: inv=%d order=%v", len(ch.Inventory), ch.PCData.Forge)
	}
	w := ch.Inventory[0]
	if w.WeaponType() != types.WeaponDagger || w.DamageType() != types.DamPierce || w.Values[1] != 2 || w.Values[2] != 4 {
		t.Errorf("wrong weapon: %s values=%v", w.ShortDesc, w.Values)
	}
}
