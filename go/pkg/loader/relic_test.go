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
