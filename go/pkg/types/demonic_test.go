package types

import "testing"

func TestDemonicSet(t *testing.T) {
	demon := NewCharacter("Azazel")
	demon.PCData = &PCData{}
	demon.Class = ClassDemon

	piece := func(tier int) *Object {
		obj := NewObject(VnumDemonicFirst, "a demonic ring", ItemTypeArmor)
		obj.Name, obj.LongDesc = "demonic ring", "A demonic ring smoulders here."
		ForgeDemonic(obj, tier, "Azazel")
		return obj
	}

	demon.Equip(piece(DemonBlack), WearLocFingerL)
	demon.Equip(piece(DemonBrass), WearLocFingerR)
	demon.Equip(piece(DemonPurple), WearLocNeck1)
	demon.Equip(NewObject(3000, "a plain ring", ItemTypeArmor), WearLocHead)

	if pieces, fizzle := demon.DemonicSet(); pieces != 3 || fizzle != 5+3 {
		t.Fatalf("DemonicSet = %d pieces, %d%% fizzle; want 3, 8", pieces, fizzle)
	}

	mortal := NewCharacter("Bob")
	mortal.PCData = &PCData{}
	mortal.Class = ClassWarrior
	mortal.Equip(piece(DemonBrass), WearLocFingerL)
	if pieces, fizzle := mortal.DemonicSet(); pieces != 0 || fizzle != 0 {
		t.Fatalf("non-demon DemonicSet = %d, %d; want 0, 0", pieces, fizzle)
	}

	got := piece(DemonRed)
	if got.ShortDesc != "a red demonic ring" || got.Name != "red demonic ring" || got.Owner != "Azazel" || got.Tier != DemonRed {
		t.Fatalf("forged = %q / %q / %q / tier %d", got.ShortDesc, got.Name, got.Owner, got.Tier)
	}
}
