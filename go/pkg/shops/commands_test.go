package shops

import (
	"strings"
	"testing"

	"rotmud/pkg/types"
)

type fixedHourWorld struct{ hour int }

func (w fixedHourWorld) GetCurrentHour() int                          { return w.hour }
func (w fixedHourWorld) GetObjectTemplate(vnum int) *types.Object     { return nil }
func (w fixedHourWorld) CloneObject(tmpl *types.Object) *types.Object { return tmpl }

// TestClosedShopSaysClosed checks that buy/sell/list in a room whose only
// keeper is closed get the closed message, not "nothing for sale".
func TestClosedShopSaysClosed(t *testing.T) {
	registry := NewShopRegistry()
	shop := NewShop(3001)
	shop.OpenHour, shop.CloseHour = 9, 17
	registry.Register(shop)

	room := types.NewRoom(1, "Shop", "x")
	keeper := &types.Character{Name: "keeper", ShortDesc: "the keeper", MobVnum: 3001, Act: types.ActNPC}
	ch := types.NewCharacter("Aldor")
	room.AddPerson(keeper)
	room.AddPerson(ch)
	ch.InRoom = room

	h := NewShopHandler(registry, fixedHourWorld{hour: 3})
	var out string
	h.Output = func(_ *types.Character, msg string) { out += msg }

	for name, cmd := range map[string]func(*types.Character, string){
		"buy": h.DoBuy, "sell": h.DoSell, "list": h.DoList,
	} {
		out = ""
		cmd(ch, "rune")
		if !strings.Contains(out, "we're closed") {
			t.Errorf("%s: expected closed message, got %q", name, out)
		}
	}
}
