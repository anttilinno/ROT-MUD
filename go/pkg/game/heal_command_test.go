package game

import (
	"testing"

	"rotmud/pkg/types"
)

// setupHealerRoom builds a player standing in a room with a healer NPC
// (spec_cast_adept) and returns the dispatcher, player, and an output sink.
func setupHealerRoom() (*CommandDispatcher, *types.Character, *string) {
	d := NewCommandDispatcher()
	out := new(string)
	d.Output = func(ch *types.Character, msg string) { *out += msg }

	room := types.NewRoom(3001, "Temple", "The temple of Midgaard.")

	healer := types.NewCharacter("the healer")
	healer.Act.Set(types.ActNPC)
	healer.Special = "spec_cast_adept"
	healer.Level = 103

	player := types.NewCharacter("Test")
	player.Position = types.PosStanding

	room.AddPerson(healer)
	room.AddPerson(player)
	healer.InRoom = room
	player.InRoom = room

	return d, player, out
}

func TestObservableState(t *testing.T) {
	ch := types.NewCharacter("Test")
	ch.MaxHit = 100
	ch.Hit = 40 // "has some big nasty wounds"
	ch.Coin = 0 // destitute
	ch.AffectedBy.Set(types.AffPoison)

	s := observableState(ch)
	for _, want := range []string{"wounds", "poisoned", "destitute", "no visible equipment"} {
		if !contains(s, want) {
			t.Errorf("observableState missing %q; got: %s", want, s)
		}
	}
}

func TestWealthImpression(t *testing.T) {
	cases := map[int64]string{0: "destitute", 10: "poor", 200: "modest", 1000: "well-off", 99999: "wealthy"}
	for copper, want := range cases {
		if got := wealthImpression(copper); !contains(got, want) {
			t.Errorf("wealthImpression(%d)=%q, want substring %q", copper, got, want)
		}
	}
}

func TestHealCommandRegistered(t *testing.T) {
	d := NewCommandDispatcher()
	if d.Registry.Find("heal") == nil {
		t.Fatal("heal command not registered")
	}
}

func TestHealMenuNoHealer(t *testing.T) {
	d := NewCommandDispatcher()
	out := new(string)
	d.Output = func(ch *types.Character, msg string) { *out += msg }
	ch := types.NewCharacter("Test")
	ch.Position = types.PosStanding
	ch.InRoom = types.NewRoom(1, "Void", "Nothing here.")

	d.Dispatch(Command{Character: ch, Input: "heal"})
	if !contains(*out, "can't do that here") {
		t.Errorf("expected refusal with no healer present, got %q", *out)
	}
}

func TestHealMenuListsServices(t *testing.T) {
	d, player, out := setupHealerRoom()
	d.Dispatch(Command{Character: player, Input: "heal"})
	for _, want := range []string{"light", "heal", "mana", "remove curse"} {
		if !contains(*out, want) {
			t.Errorf("menu missing %q; got:\n%s", want, *out)
		}
	}
}

func TestHealBuyDeductsGoldAndHeals(t *testing.T) {
	d, player, out := setupHealerRoom()
	player.Coin = 5000 // 50 gold
	player.MaxHit = 100
	player.Hit = 10

	d.Dispatch(Command{Character: player, Input: "heal light"})

	// cure light costs 10 copper
	if player.Coin != 4990 {
		t.Errorf("expected 4990 copper left (5000-10), got %d", player.Coin)
	}
	if player.Hit <= 10 {
		t.Errorf("expected hp to rise after cure light, got %d", player.Hit)
	}
	if !contains(*out, "utters the words") {
		t.Errorf("expected healer to utter the magic words; got %q", *out)
	}
}

func TestHealManaRestore(t *testing.T) {
	d, player, _ := setupHealerRoom()
	player.Coin = 5000
	player.MaxMana = 100
	player.Mana = 5

	d.Dispatch(Command{Character: player, Input: "heal mana"})

	if player.Mana != 100 {
		t.Errorf("expected mana restored to 100, got %d", player.Mana)
	}
	if player.Coin != 4990 { // mana costs 10 copper
		t.Errorf("expected 4990 copper left, got %d", player.Coin)
	}
}

func TestHealNotEnoughGold(t *testing.T) {
	d, player, out := setupHealerRoom()
	player.Coin = 10 // too little for 'heal' (50 copper)

	d.Dispatch(Command{Character: player, Input: "heal heal"})

	if player.Coin != 10 {
		t.Errorf("expected gold unchanged when too poor, got %d", player.Coin)
	}
	if !contains(*out, "beyond your means") {
		t.Errorf("expected donation-too-high message, got %q", *out)
	}
}
