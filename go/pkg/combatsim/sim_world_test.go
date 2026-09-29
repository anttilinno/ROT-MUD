package combatsim

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"testing"

	"rotmud/pkg/loader"
	"rotmud/pkg/types"
)

// World-data combat sim: the toughest real mob at each level against a player
// wearing the best real equipment reachable at that level (items that appear
// in some reset, level <= player level). The sim's own weapon is kept so only
// armour/jewellery vary; gear is applied through Character.Equip exactly as
// in play (item affects + ROM apply_ac).

var worldSimLevels = []int{1, 10, 20, 30, 40, 50, 60, 75, 100}

var worldSimClasses = []int{
	types.ClassWarrior, types.ClassRanger, types.ClassThief, types.ClassCleric,
	types.ClassDruid, types.ClassGhoul, types.ClassMage,
}

// gearSlots maps a wear flag name to the wear locations it can fill.
var gearSlots = map[string][]types.WearLocation{
	"finger": {types.WearLocFingerL, types.WearLocFingerR},
	"neck":   {types.WearLocNeck1, types.WearLocNeck2},
	"wrist":  {types.WearLocWristL, types.WearLocWristR},
	"body":   {types.WearLocBody}, "head": {types.WearLocHead}, "legs": {types.WearLocLegs},
	"feet": {types.WearLocFeet}, "hands": {types.WearLocHands}, "arms": {types.WearLocArms},
	"shield": {types.WearLocShield}, "about": {types.WearLocAbout}, "waist": {types.WearLocWaist},
	"hold": {types.WearLocHold}, "float": {types.WearLocFloat}, "face": {types.WearLocFace},
}

var applyByName = map[string]types.ApplyType{
	"str": types.ApplyStr, "dex": types.ApplyDex, "int": types.ApplyInt, "wis": types.ApplyWis,
	"con": types.ApplyCon, "hit": types.ApplyHit, "mana": types.ApplyMana, "move": types.ApplyMove,
	"ac": types.ApplyAC, "hitroll": types.ApplyHitroll, "damroll": types.ApplyDamroll,
}

func gearObject(t *loader.ObjectData) *types.Object {
	itemType := types.ItemTypeTrash
	if t.ItemType == "armor" {
		itemType = types.ItemTypeArmor
	}
	obj := types.NewObject(t.Vnum, t.ShortDesc, itemType)
	obj.Level = t.Level
	if t.Armor != nil {
		obj.Values = [5]int{t.Armor.ACPierce, t.Armor.ACBash, t.Armor.ACSlash, t.Armor.ACExotic, 0}
	}
	for _, a := range t.Affects {
		if loc, ok := applyByName[a.Location]; ok {
			obj.Affects.Add(&types.Affect{Type: "object", Duration: -1, Location: loc, Modifier: a.Modifier})
		}
	}
	return obj
}

// gearScore ranks an item worn at loc. ponytail: fixed weights (1 AC point,
// 10 per hitroll, 15 per damroll, 1/2 per hp, 5 per str/dex/con); fine for
// picking best-in-slot, not a balance model.
func gearScore(obj *types.Object, loc types.WearLocation) int {
	s := 0
	for i := 0; i < 4; i++ {
		s += types.ArmorAC(obj, loc, i)
	}
	s /= 4
	for _, af := range obj.Affects.All() {
		switch af.Location {
		case types.ApplyAC:
			s -= af.Modifier
		case types.ApplyHitroll:
			s += 10 * af.Modifier
		case types.ApplyDamroll:
			s += 15 * af.Modifier
		case types.ApplyHit:
			s += af.Modifier / 2
		case types.ApplyStr, types.ApplyDex, types.ApplyCon:
			s += 5 * af.Modifier
		}
	}
	return s
}

type worldSim struct {
	world     *loader.World
	objs      []*loader.ObjectData // reachable wearables
	mobs      []*loader.MobileData // reachable, fightable mobs
	gearCache map[int][]gearPiece
}

type gearPiece struct {
	tmpl *loader.ObjectData
	loc  types.WearLocation
}

func newWorldSim(t *testing.T) *worldSim {
	world, err := loader.NewAreaLoader("../../data/areas").LoadAll()
	if err != nil {
		t.Fatal(err)
	}
	objSeen, mobSeen := map[int]bool{}, map[int]bool{}
	for _, room := range world.Rooms {
		for _, mr := range room.MobResets {
			mobSeen[mr.Vnum] = true
			for _, e := range mr.Equips {
				objSeen[e.Vnum] = true
			}
		}
		for _, or := range room.ObjResets {
			objSeen[or.Vnum] = true
		}
	}
	ws := &worldSim{world: world, gearCache: map[int][]gearPiece{}}
	for v := range objSeen {
		if o := world.GetObjTemplate(v); o != nil {
			ws.objs = append(ws.objs, o)
		}
	}
	for v := range mobSeen {
		m := world.GetMobTemplate(v)
		// Skip shops, rite masters, and weapon-immune NPCs (ROT's deliberately
		// unkillable mobs, e.g. Mr. Miyagi).
		if m == nil || m.Shop != nil || v >= 29600 && v <= 29699 || slices.Contains(m.ImmFlags, "weapon") {
			continue
		}
		skip := false
		for _, f := range m.ActFlags {
			switch f {
			case "train", "practice", "gain", "healer", "is_healer", "pet":
				skip = true
			}
		}
		if !skip {
			ws.mobs = append(ws.mobs, m)
		}
	}
	sort.Slice(ws.objs, func(i, j int) bool { return ws.objs[i].Vnum < ws.objs[j].Vnum })
	return ws
}

// bestGear picks the highest-scoring reachable item per wear location.
func (ws *worldSim) bestGear(level int) []gearPiece {
	if g, ok := ws.gearCache[level]; ok {
		return g
	}
	type cand struct {
		gearPiece
		score int
	}
	byLoc := map[types.WearLocation][]cand{}
	for _, o := range ws.objs {
		if o.Level > level {
			continue
		}
		for _, wf := range o.WearFlags {
			for _, loc := range gearSlots[wf] {
				obj := gearObject(o)
				if s := gearScore(obj, loc); s > 0 {
					byLoc[loc] = append(byLoc[loc], cand{gearPiece{o, loc}, s})
				}
			}
		}
	}
	used := map[int]bool{} // one copy of each item per paired slot
	var gear []gearPiece
	for loc := types.WearLocation(0); loc < types.WearLocMax; loc++ {
		cs := byLoc[loc]
		sort.Slice(cs, func(i, j int) bool { return cs[i].score > cs[j].score })
		for _, c := range cs {
			if !used[c.tmpl.Vnum] {
				used[c.tmpl.Vnum] = true
				gear = append(gear, c.gearPiece)
				break
			}
		}
	}
	ws.gearCache[level] = gear
	return gear
}

// geared builds the sim player, strips the sim's assumed armour, and wears real gear.
func (ws *worldSim) geared(classIdx, raceIdx, level int) *types.Character {
	p := makePlayer(classIdx, raceIdx, level)
	for i := range p.Armor {
		p.Armor[i] = 100
	}
	for _, g := range ws.bestGear(level) {
		p.Equip(gearObject(g.tmpl), g.loc)
	}
	p.Hit = p.MaxHit
	return p
}

func mobThreat(m *loader.MobileData) int {
	hp := m.HitDice.Number*(m.HitDice.Size/2) + m.HitDice.Bonus
	dam := m.DamageDice.Number*(m.DamageDice.Size+1)/2 + m.DamageDice.Bonus
	for _, a := range m.AffectedBy {
		if a == "sanctuary" {
			hp *= 2
		}
	}
	return hp * max(dam, 1)
}

// mobsAt returns reachable mobs at the level nearest to level, toughest first.
func (ws *worldSim) mobsAt(level int) []*loader.MobileData {
	for d := 0; d <= 10; d++ {
		var out []*loader.MobileData
		for _, m := range ws.mobs {
			if m.Level == level-d || m.Level == level+d {
				out = append(out, m)
			}
		}
		if len(out) > 0 {
			sort.Slice(out, func(i, j int) bool {
				if ti, tj := mobThreat(out[i]), mobThreat(out[j]); ti != tj {
					return ti > tj
				}
				return out[i].Vnum < out[j].Vnum // deterministic among equal threats
			})
			return out
		}
	}
	return nil
}

func (ws *worldSim) mobFn(vnum int) func(int) *types.Character {
	return func(int) *types.Character { return ws.world.CreateMobFromTemplate(vnum) }
}

func TestCombatSimWorldGear(t *testing.T) {
	const n = 500
	ws := newWorldSim(t)

	t.Log("")
	t.Log("=== REAL MOBS vs REAL GEAR (human, N=500; sim weapon kept) ===")
	t.Log("")
	t.Log("Toughest / median reachable mob per level (HP, avg damage/hit, sanctuary):")
	toughest, median := map[int]int{}, map[int]int{}
	for _, lv := range worldSimLevels {
		ms := ws.mobsAt(lv)
		top, mid := ms[0], ms[len(ms)/2]
		toughest[lv], median[lv] = top.Vnum, mid.Vnum
		desc := func(m *loader.MobileData) string {
			mob := ws.world.CreateMobFromTemplate(m.Vnum)
			return fmt.Sprintf("%-28.28s L%-3d hp %-5d dmg %-4d sanc %-5v",
				m.ShortDesc, m.Level, mob.MaxHit,
				mob.Damage[0]*(mob.Damage[1]+1)/2+mob.Damage[2], mob.IsAffected(types.AffSanctuary))
		}
		t.Logf("Lv%-3d top: %s | median: %s", lv, desc(top), desc(mid))
	}

	t.Log("")
	t.Log("Best reachable gear vs the sim's assumed gear (warrior shown for sim AC):")
	t.Log("Level  pieces  AC(real)  AC(sim)  +hit  +dam  +hp  +mana  +str  +dex  +con")
	for _, lv := range worldSimLevels {
		base := makePlayer(types.ClassWarrior, types.RaceHuman, lv)
		p := ws.geared(types.ClassWarrior, types.RaceHuman, lv)
		t.Logf("Lv%-4d %-7d %-9d %-8d %-5d %-5d %-4d %-6d %-5d %-5d %-4d", lv, len(ws.bestGear(lv)),
			p.Armor[types.ACSlash], base.Armor[types.ACSlash],
			p.HitRoll-base.HitRoll, p.DamRoll-base.DamRoll, p.MaxHit-base.MaxHit, p.MaxMana-base.MaxMana,
			p.ModStats[types.StatStr], p.ModStats[types.StatDex], p.ModStats[types.StatCon])
	}
	t.Log("")

	table := func(title string, playerFn func(c, r, l int) *types.Character, mobs map[int]int) {
		hdr := fmt.Sprintf("%-10s", title)
		for _, lv := range worldSimLevels {
			hdr += fmt.Sprintf("  Lv%-6d", lv)
		}
		t.Log(hdr)
		t.Log(strings.Repeat("-", len(hdr)))
		for _, ci := range worldSimClasses {
			row := fmt.Sprintf("%-10s", types.ClassTable[ci].Name)
			for _, lv := range worldSimLevels {
				r := runSimFull(ci, types.RaceHuman, lv, n, playerFn, ws.mobFn(mobs[lv]))
				row += fmt.Sprintf("  %3.0f%%/%2.0fs", r.winPct(), r.avgSeconds())
			}
			t.Log(row)
		}
		t.Log("")
	}
	table("SimGear vs toughest", makePlayer, toughest)
	table("RealGear vs toughest", ws.geared, toughest)
	table("SimGear vs median", makePlayer, median)
	table("RealGear vs median", ws.geared, median)
}
