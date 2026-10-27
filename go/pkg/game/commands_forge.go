package game

import (
	"fmt"
	"strings"
	"time"

	"rotmud/pkg/loader"
	"rotmud/pkg/types"
)

// forgeWeapons lists the weapon types a smith can be commissioned to forge,
// with the damage type each one deals.
var forgeWeapons = map[string]struct {
	class types.WeaponClass
	dam   types.DamageType
}{
	"dagger":  {types.WeaponDagger, types.DamPierce},
	"sword":   {types.WeaponSword, types.DamSlash},
	"axe":     {types.WeaponAxe, types.DamSlash},
	"mace":    {types.WeaponMace, types.DamBash},
	"spear":   {types.WeaponSpear, types.DamPierce},
	"flail":   {types.WeaponFlail, types.DamBash},
	"whip":    {types.WeaponWhip, types.DamSlash},
	"polearm": {types.WeaponPolearm, types.DamSlash},
}

var forgeWeaponNames = []string{"dagger", "sword", "axe", "mace", "spear", "flail", "whip", "polearm"}

// forgeNow is the clock for forge orders; tests replace it.
var forgeNow = time.Now

// gameDay is one game day of real time (24 ticks).
const gameDay = 24 * time.Duration(types.PulseTick/types.PulsePerSecond) * time.Second

// findSmith returns a mob in ch's room that takes weapon commissions.
func (d *CommandDispatcher) findSmith(ch *types.Character) (*types.Character, *loader.MobileData) {
	if ch.InRoom == nil || d.GameLoop == nil || d.GameLoop.World == nil {
		return nil, nil
	}
	for _, mob := range ch.InRoom.People {
		if !mob.IsNPC() {
			continue
		}
		if tmpl := d.GameLoop.World.GetMobTemplate(mob.MobVnum); tmpl != nil && len(tmpl.ForgeTiers) > 0 {
			return mob, tmpl
		}
	}
	return nil, nil
}

// cmdCommission orders a weapon from a smith: `commission <type> <tier>`.
// Without arguments it lists what the smith forges.
func (d *CommandDispatcher) cmdCommission(ch *types.Character, args string) {
	if ch.IsNPC() || ch.PCData == nil {
		return
	}
	smith, tmpl := d.findSmith(ch)
	forge := d.GameLoop.World.Forge
	if smith == nil || forge == nil {
		d.send(ch, "Nobody here forges weapons to order.\r\n")
		return
	}

	parts := strings.Fields(strings.ToLower(args))
	if len(parts) < 2 {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("%s says 'I forge weapons to order, ready in %d days:'\r\n", smith.ShortDesc, forge.ForgeDays))
		for _, name := range tmpl.ForgeTiers {
			if t := forge.Tier(name); t != nil {
				sb.WriteString(fmt.Sprintf("  %-11s level %3d  %dd%-3d %12s\r\n", t.Name, t.Level, t.DiceNumber, t.DiceSize, types.FormatCoin(t.Price)))
			}
		}
		sb.WriteString("Weapons: " + strings.Join(forgeWeaponNames, ", ") + "\r\n")
		sb.WriteString("Usage: commission <weapon> <tier>\r\n")
		d.send(ch, sb.String())
		return
	}

	if ch.PCData.Forge != nil {
		d.send(ch, fmt.Sprintf("%s says 'You already have a weapon on order. Collect it first.'\r\n", smith.ShortDesc))
		return
	}
	weapon, tierName := parts[0], parts[1]
	if _, ok := forgeWeapons[weapon]; !ok {
		d.send(ch, fmt.Sprintf("%s says 'I don't make those. I forge: %s.'\r\n", smith.ShortDesc, strings.Join(forgeWeaponNames, ", ")))
		return
	}
	tier := forge.Tier(tierName)
	offered := false
	for _, name := range tmpl.ForgeTiers {
		offered = offered || name == tierName
	}
	if tier == nil || !offered {
		d.send(ch, fmt.Sprintf("%s says 'That is not work I do. Type commission to see what I offer.'\r\n", smith.ShortDesc))
		return
	}
	if ch.Level < tier.Level {
		d.send(ch, fmt.Sprintf("%s says 'Come back when you are level %d. You couldn't wield it yet.'\r\n", smith.ShortDesc, tier.Level))
		return
	}
	if ch.Coin < tier.Price {
		d.send(ch, fmt.Sprintf("%s says 'That costs %s. Come back with the coin.'\r\n", smith.ShortDesc, types.FormatCoin(tier.Price)))
		return
	}

	ch.Coin -= tier.Price
	ch.PCData.Forge = &types.ForgeOrder{
		Smith:      smith.MobVnum,
		Tier:       tier.Name,
		WeaponType: weapon,
		ReadyAt:    forgeNow().Add(time.Duration(forge.ForgeDays) * gameDay).Unix(),
	}
	d.send(ch, fmt.Sprintf("You pay %s. %s says 'A %s %s. Come back in %d days.'\r\n",
		types.FormatCoin(tier.Price), smith.ShortDesc, tier.Name, weapon, forge.ForgeDays))
}

// cmdCollect picks up a finished commission from the smith who took it.
func (d *CommandDispatcher) cmdCollect(ch *types.Character, args string) {
	if ch.IsNPC() || ch.PCData == nil {
		return
	}
	order := ch.PCData.Forge
	if order == nil {
		d.send(ch, "You have nothing on order.\r\n")
		return
	}
	smith, _ := d.findSmith(ch)
	if smith == nil || smith.MobVnum != order.Smith {
		d.send(ch, "The smith who took your order isn't here.\r\n")
		return
	}
	if left := time.Unix(order.ReadyAt, 0).Sub(forgeNow()); left > 0 {
		days := int((left + gameDay - 1) / gameDay)
		unit := "days"
		if days == 1 {
			unit = "day"
		}
		d.send(ch, fmt.Sprintf("%s says 'Your %s isn't done. Come back in %d %s.'\r\n", smith.ShortDesc, order.WeaponType, days, unit))
		return
	}

	tier := d.GameLoop.World.Forge.Tier(order.Tier)
	if tier == nil {
		d.send(ch, fmt.Sprintf("%s says 'I can no longer make that. Here is your money back.'\r\n", smith.ShortDesc))
		ch.PCData.Forge = nil
		return
	}
	obj := forgeWeapon(tier, order.WeaponType, smith)
	ch.PCData.Forge = nil
	ObjToChar(obj, ch)
	d.send(ch, fmt.Sprintf("%s hands you %s.\r\n", smith.ShortDesc, obj.ShortDesc))
}

// forgeWeapon builds a commissioned weapon of the given tier and type.
func forgeWeapon(tier *loader.ForgeTier, weapon string, smith *types.Character) *types.Object {
	w := forgeWeapons[weapon]
	obj := types.NewObject(0, fmt.Sprintf("a %s %s forged by %s", tier.Name, weapon, smith.ShortDesc), types.ItemTypeWeapon)
	obj.Name = fmt.Sprintf("%s %s forged", weapon, tier.Name)
	obj.LongDesc = fmt.Sprintf("A %s %s lies here.", tier.Name, weapon)
	obj.Level = tier.Level
	obj.Weight = 10
	obj.Cost = int(tier.Price / 10) // shops pay little back for custom work
	obj.Material = "steel"
	obj.WearFlags.Set(types.WearTake)
	obj.WearFlags.Set(types.WearWield)
	obj.Values[0] = int(w.class)
	obj.Values[1] = tier.DiceNumber
	obj.Values[2] = tier.DiceSize
	obj.Values[3] = int(w.dam)
	if tier.Hitroll != 0 {
		obj.Affects.Add(types.NewAffect("forged", tier.Level, -1, types.ApplyHitroll, tier.Hitroll, 0))
	}
	if tier.Damroll != 0 {
		obj.Affects.Add(types.NewAffect("forged", tier.Level, -1, types.ApplyDamroll, tier.Damroll, 0))
	}
	return obj
}
