package game

import (
	"strings"
	"testing"

	"rotmud/pkg/loader"
	"rotmud/pkg/types"
)

// A hero rerolls, takes the demon pact, earns and spends power, forges and
// sacrifices demonic armour — the whole tier-2 loop through real commands.
func TestDemonLifecycle(t *testing.T) {
	world, err := loader.NewAreaLoader("../../data/areas").LoadAll()
	if err != nil {
		t.Fatal(err)
	}
	d := NewCommandDispatcher()
	d.GameLoop = NewGameLoop()
	d.GameLoop.World = world
	var out string
	d.Output = func(_ *types.Character, msg string) { out += msg }
	run := func(ch *types.Character, input string) string {
		out = ""
		d.Dispatch(Command{Character: ch, Input: input})
		return out
	}

	throne := types.NewRoom(29601, "The Brass Throne", "")
	lord := types.NewNPC(29601, "demon lord", 100)
	lord.ShortDesc = "the demon lord"
	CharToRoom(lord, throne)

	hero := types.NewCharacter("Faust")
	hero.PCData = &types.PCData{Learned: map[string]int{"sword": 80}}
	hero.Race, hero.Class, hero.Level = types.RaceHuman, types.ClassWarrior, types.LevelHero
	hero.Position = types.PosStanding
	hero.Coin = 1000 * types.CopperPerGold
	CharToRoom(hero, throne)

	if got := run(hero, "rite"); !strings.Contains(got, "turned from mortal paths") {
		t.Fatalf("rite before reroll: %q", got)
	}
	run(hero, "reroll confirm")
	run(hero, "rite begin")
	if hero.PCData.Rite != types.ClassDemon {
		t.Fatalf("rite not begun: %q", out)
	}

	paladin := types.NewNPC(1, "paladin", 60)
	paladin.Alignment = 1000
	for range 10 {
		d.onKill(hero, paladin)
	}
	if got := run(hero, "rite complete"); !strings.Contains(got, "reborn as a demon") {
		t.Fatalf("rite complete: %q", got)
	}
	if hero.Class != types.ClassDemon || hero.Level != 1 || hero.SkillClass() != types.ClassWarrior {
		t.Fatalf("after pact: class %d level %d skillclass %d", hero.Class, hero.Level, hero.SkillClass())
	}

	for range 100 {
		d.onKill(hero, paladin) // 60 power each
	}
	if got := run(hero, "powers wings"); !strings.Contains(got, "power of wings") || !hero.IsAffected(types.AffFlying) {
		t.Fatalf("powers wings: %q", got)
	}
	if got := run(hero, "powers"); !strings.Contains(got, "demonic power") || !strings.Contains(got, "* wings") {
		t.Fatalf("powers list: %q", got)
	}

	before := hero.PCData.Power
	if got := run(hero, "demonarmour purple ring"); !strings.Contains(got, "blast of flames") {
		t.Fatalf("demonarmour: %q", got)
	}
	ring := hero.Inventory[len(hero.Inventory)-1]
	if ring.ShortDesc != "a purple demonic ring" || ring.Owner != "Faust" || hero.PCData.Power != before-types.DemonArmourPower {
		t.Fatalf("forged %q owner %q power %d", ring.ShortDesc, ring.Owner, hero.PCData.Power)
	}

	hero.RemoveInventory(ring)
	ObjToRoom(ring, throne)
	if got := run(hero, "sacrifice ring"); !strings.Contains(got, "drain 2000") || hero.PCData.Power != before {
		t.Fatalf("sacrifice demonic: %q power %d want %d", got, hero.PCData.Power, before)
	}
}
