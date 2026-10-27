package server

import (
	"testing"

	"rotmud/pkg/types"
)

// TestGroupAddRefundsBoughtSkill: buying circle (5) then the stealth group
// (6, includes circle) must cost 6, not 11.
func TestGroupAddRefundsBoughtSkill(t *testing.T) {
	h := &LoginHandler{}
	ch := types.NewCharacter("Test")
	ch.Class = types.ClassThief
	ch.PCData = &types.PCData{Learned: map[string]int{}}
	bought := map[string]bool{}

	if r := h.addIndividualSkill(ch, "circle", bought); r != 0 {
		t.Fatalf("add circle: result %d", r)
	}
	if r := h.addGroup(ch, "stealth", bought); r != 0 {
		t.Fatalf("add stealth: result %d", r)
	}
	if ch.PCData.Points != 6 {
		t.Errorf("points = %d, want 6", ch.PCData.Points)
	}

	// circle now comes from the group: dropping it alone must not refund.
	if h.dropIndividualSkill(ch, "circle", bought) {
		t.Error("dropped group skill circle on its own")
	}
	if !h.dropGroup(ch, "stealth", bought) || ch.PCData.Points != 0 {
		t.Errorf("after dropping stealth points = %d, want 0", ch.PCData.Points)
	}
}

func TestGroupHint(t *testing.T) {
	h := &LoginHandler{}
	ch := types.NewCharacter("Test")
	ch.Class = types.ClassThief
	if got := h.groupHint(ch, "circle"); got != "circle is also in the stealth group (6): sneak, hide, backstab, circle.\r\n" {
		t.Errorf("hint = %q", got)
	}
	if got := h.groupHint(ch, "steal"); got != "" {
		t.Errorf("steal is in no group, got %q", got)
	}
}
