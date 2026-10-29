package loader

import (
	"testing"

	"rotmud/pkg/types"
)

// Fields restored from ROT (tools/rotrestore/mobs.py) reach the spawned mob.
func TestCreateMobRestoredFields(t *testing.T) {
	w := NewWorld()
	w.MobTemplates[1] = &MobileData{
		Vnum: 1, Keywords: []string{"guard"}, Level: 30, Alignment: -500,
		ActFlags:    []string{"npc", "undead", "healer"},
		AffectedBy:  []string{"sanctuary", "invisible", "protect_good"},
		ShieldFlags: []string{"fire"},
		ManaDice:    DiceData{Number: 2, Size: 10, Bonus: 100},
		StartPos:    "sleeping", DefaultPos: "resting",
	}
	mob := w.CreateMobFromTemplate(1)

	if mob.Position != types.PosSleeping || mob.DefaultPos != types.PosResting {
		t.Errorf("position %v default %v, want sleeping/resting", mob.Position, mob.DefaultPos)
	}
	if mob.MaxMana != 110 || mob.Mana != 110 {
		t.Errorf("mana %d/%d, want 110", mob.Mana, mob.MaxMana)
	}
	if mob.Alignment != -500 || !mob.Act.Has(types.ActUndead) || !mob.Act.Has(types.ActIsHealer) {
		t.Error("alignment or act flags not applied")
	}
	if !mob.IsAffected(types.AffInvisible) || !mob.IsAffected(types.AffProtectGood) || !mob.IsShielded(types.ShdFire) {
		t.Error("affect or shield flags not applied")
	}
	// Sanctuary halves damage taken, so the HP floor is halved too.
	if want := anchorAt(mobHPFloor, 30) / 2; mob.MaxHit != want {
		t.Errorf("sanctuary mob hp %d, want half floor %d", mob.MaxHit, want)
	}

	w.MobTemplates[2] = &MobileData{Vnum: 2, Level: 5, StartPos: "fighting"}
	if mob := w.CreateMobFromTemplate(2); mob.Position != types.PosStanding {
		t.Errorf("mob spawned %v, want standing", mob.Position)
	}
}
