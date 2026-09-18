package loader

import (
	"strings"
	"testing"

	"rotmud/pkg/types"
)

// Every demonic slot template must exist, name itself "demonic" (ForgeDemonic
// inserts the tier colour there), and occupy its own wear slot.
func TestRelicDemonicTemplates(t *testing.T) {
	world, err := NewAreaLoader("../../data/areas").LoadAll()
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]int{}
	for vnum := types.VnumDemonicFirst; vnum <= types.VnumDemonicLast; vnum++ {
		tmpl := world.GetObjTemplate(vnum)
		if tmpl == nil {
			t.Fatalf("missing demonic template %d", vnum)
		}
		if !strings.Contains(tmpl.ShortDesc, "demonic") || !strings.Contains(tmpl.LongDesc, "demonic") {
			t.Errorf("%d: descriptions must contain \"demonic\": %q / %q", vnum, tmpl.ShortDesc, tmpl.LongDesc)
		}
		for _, w := range tmpl.WearFlags {
			if w != "take" {
				if prev, dup := seen[w]; dup {
					t.Errorf("%d and %d share wear slot %q", prev, vnum, w)
				}
				seen[w] = vnum
			}
		}
	}
	if len(seen) != 12 {
		t.Errorf("demonic set covers %d slots, want 12", len(seen))
	}
}

// Every tier-2 rite master must exist and be reset somewhere in the Hall of
// Rites, and the hall must connect to the Temple of Thoth both ways.
func TestRiteMasters(t *testing.T) {
	world, err := NewAreaLoader("../../data/areas").LoadAll()
	if err != nil {
		t.Fatal(err)
	}
	reset := map[int]bool{}
	for vnum := 29600; vnum <= 29699; vnum++ {
		if room := world.GetRoom(vnum); room != nil {
			for _, mr := range room.MobResets {
				reset[mr.Vnum] = true
			}
		}
	}
	for c := types.ClassDemon; c < types.MaxClass; c++ {
		v := types.ClassTable[c].Tier2.MasterVnum
		if world.GetMobTemplate(v) == nil || !reset[v] {
			t.Errorf("%s rite master %d missing or never reset", types.ClassName(c), v)
		}
	}
	down := world.GetRoom(3001).GetExit(types.DirDown)
	up := world.GetRoom(29600).GetExit(types.DirUp)
	if down == nil || down.ToVnum != 29600 || up == nil || up.ToVnum != 3001 {
		t.Error("temple 3001 <-> hall 29600 link broken")
	}
}
