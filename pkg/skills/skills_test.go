package skills

import (
	"testing"

	"rotmud/pkg/types"
)

func TestNewSkill(t *testing.T) {
	skill := NewSkill("test skill")

	if skill.Name != "test skill" {
		t.Errorf("expected name 'test skill', got %q", skill.Name)
	}
	if skill.Type != TypeSkill {
		t.Errorf("expected TypeSkill, got %v", skill.Type)
	}
	if skill.MinPosition != types.PosStanding {
		t.Errorf("expected PosStanding, got %v", skill.MinPosition)
	}
	if skill.Beats != 12 {
		t.Errorf("expected Beats 12, got %d", skill.Beats)
	}
	if skill.Levels == nil {
		t.Error("expected Levels map to be initialized")
	}
	if skill.Rating == nil {
		t.Error("expected Rating map to be initialized")
	}
}

func TestSkillSetClassLevel(t *testing.T) {
	skill := NewSkill("dodge").
		SetClassLevel(Warrior, 13, 6).
		SetClassLevel(Thief, 1, 4)

	if skill.GetLevel(Warrior) != 13 {
		t.Errorf("expected warrior level 13, got %d", skill.GetLevel(Warrior))
	}
	if skill.GetRating(Warrior) != 6 {
		t.Errorf("expected warrior rating 6, got %d", skill.GetRating(Warrior))
	}
	if skill.GetLevel(Thief) != 1 {
		t.Errorf("expected thief level 1, got %d", skill.GetLevel(Thief))
	}
	if skill.GetRating(Thief) != 4 {
		t.Errorf("expected thief rating 4, got %d", skill.GetRating(Thief))
	}
	if skill.GetLevel(Mage) != 0 {
		t.Errorf("expected mage level 0 (not set), got %d", skill.GetLevel(Mage))
	}
	if skill.GetRating(Mage) != 0 {
		t.Errorf("expected mage rating 0 (not set), got %d", skill.GetRating(Mage))
	}
}

func TestSkillCanLearn(t *testing.T) {
	skill := NewSkill("backstab").
		SetClassLevel(Thief, 1, 5).
		SetClassLevel(Warrior, 0, 0)

	if !skill.CanLearn(Thief) {
		t.Error("thief should be able to learn backstab")
	}
	if skill.CanLearn(Warrior) {
		t.Error("warrior should not be able to learn backstab")
	}
	if skill.CanLearn(Mage) {
		t.Error("mage should not be able to learn backstab (not set)")
	}
}

func TestSkillRegistry(t *testing.T) {
	r := NewSkillRegistry()

	skill1 := NewSkill("dodge")
	skill2 := NewSkill("parry")
	skill3 := NewSkill("bash")

	idx1 := r.Register(skill1)
	idx2 := r.Register(skill2)
	idx3 := r.Register(skill3)

	if idx1 != 0 {
		t.Errorf("expected idx1 = 0, got %d", idx1)
	}
	if idx2 != 1 {
		t.Errorf("expected idx2 = 1, got %d", idx2)
	}
	if idx3 != 2 {
		t.Errorf("expected idx3 = 2, got %d", idx3)
	}
	if r.Count() != 3 {
		t.Errorf("expected count = 3, got %d", r.Count())
	}
}

func TestSkillRegistryFindByName(t *testing.T) {
	r := NewSkillRegistry()
	r.Register(NewSkill("dodge"))
	r.Register(NewSkill("parry"))
	r.Register(NewSkill("shield block"))

	// Exact match
	skill := r.FindByName("dodge")
	if skill == nil || skill.Name != "dodge" {
		t.Error("expected to find 'dodge' by exact match")
	}

	// Prefix match
	skill = r.FindByName("par")
	if skill == nil || skill.Name != "parry" {
		t.Error("expected to find 'parry' by prefix 'par'")
	}

	// Prefix match multi-word
	skill = r.FindByName("shield")
	if skill == nil || skill.Name != "shield block" {
		t.Error("expected to find 'shield block' by prefix 'shield'")
	}

	// Not found
	skill = r.FindByName("nonexistent")
	if skill != nil {
		t.Error("expected nil for nonexistent skill")
	}
}

func TestSkillRegistryFindByIndex(t *testing.T) {
	r := NewSkillRegistry()
	r.Register(NewSkill("dodge"))
	r.Register(NewSkill("parry"))

	skill := r.FindByIndex(0)
	if skill == nil || skill.Name != "dodge" {
		t.Error("expected to find 'dodge' at index 0")
	}

	skill = r.FindByIndex(1)
	if skill == nil || skill.Name != "parry" {
		t.Error("expected to find 'parry' at index 1")
	}

	skill = r.FindByIndex(999)
	if skill != nil {
		t.Error("expected nil for out-of-bounds index")
	}
}

func TestSkillRegistryGetIndex(t *testing.T) {
	r := NewSkillRegistry()
	r.Register(NewSkill("dodge"))
	r.Register(NewSkill("parry"))

	if r.GetIndex("dodge") != 0 {
		t.Error("expected index 0 for 'dodge'")
	}
	if r.GetIndex("parry") != 1 {
		t.Error("expected index 1 for 'parry'")
	}
	if r.GetIndex("nonexistent") != -1 {
		t.Error("expected index -1 for nonexistent skill")
	}
}

func TestSkillRegistryAll(t *testing.T) {
	r := NewSkillRegistry()
	r.Register(NewSkill("dodge"))
	r.Register(NewSkill("parry"))

	all := r.All()
	if len(all) != 2 {
		t.Errorf("expected 2 skills, got %d", len(all))
	}
	if all[0].Name != "dodge" {
		t.Error("expected first skill to be 'dodge'")
	}
	if all[1].Name != "parry" {
		t.Error("expected second skill to be 'parry'")
	}
}

func TestDefaultSkills(t *testing.T) {
	r := DefaultSkills()

	// Should have a good number of skills
	if r.Count() < 30 {
		t.Errorf("expected at least 30 skills, got %d", r.Count())
	}

	// Verify some specific skills exist
	dodge := r.FindByName("dodge")
	if dodge == nil {
		t.Fatal("expected 'dodge' skill to exist")
	}
	if !dodge.CanLearn(Thief) {
		t.Error("thief should be able to learn dodge")
	}
	if dodge.GetLevel(Thief) != 1 {
		t.Errorf("expected thief dodge level 1, got %d", dodge.GetLevel(Thief))
	}

	backstab := r.FindByName("backstab")
	if backstab == nil {
		t.Fatal("expected 'backstab' skill to exist")
	}
	if !backstab.CanLearn(Thief) {
		t.Error("thief should be able to learn backstab")
	}
	if backstab.CanLearn(Warrior) {
		t.Error("warrior should not be able to learn backstab")
	}

	bash := r.FindByName("bash")
	if bash == nil {
		t.Fatal("expected 'bash' skill to exist")
	}
	if !bash.CanLearn(Warrior) {
		t.Error("warrior should be able to learn bash")
	}
	if bash.CanLearn(Mage) {
		t.Error("mage should not be able to learn bash")
	}
}

func TestSkillSystemNewSkillSystem(t *testing.T) {
	sys := NewSkillSystem()

	if sys.Registry == nil {
		t.Error("expected Registry to be initialized")
	}
	if sys.Registry.Count() < 30 {
		t.Errorf("expected at least 30 default skills, got %d", sys.Registry.Count())
	}
}

func TestSkillSystemGetSkillPlayer(t *testing.T) {
	sys := NewSkillSystem()

	// Create a player character
	ch := &types.Character{
		Name:  "TestPlayer",
		Level: 10,
		Class: types.ClassThief,
		PCData: &types.PCData{
			Learned: map[string]int{
				"dodge":    75,
				"backstab": 50,
			},
		},
	}

	// Should return learned percentage
	if sys.GetSkill(ch, "dodge") != 75 {
		t.Errorf("expected dodge skill 75, got %d", sys.GetSkill(ch, "dodge"))
	}
	if sys.GetSkill(ch, "backstab") != 50 {
		t.Errorf("expected backstab skill 50, got %d", sys.GetSkill(ch, "backstab"))
	}

	// Skill not learned
	if sys.GetSkill(ch, "parry") != 0 {
		t.Error("expected 0 for unlearned skill")
	}

	// Skill doesn't exist
	if sys.GetSkill(ch, "nonexistent") != 0 {
		t.Error("expected 0 for nonexistent skill")
	}
}

func TestSkillSystemGetSkillPlayerLevelRequirement(t *testing.T) {
	sys := NewSkillSystem()

	// Create a low-level thief
	ch := &types.Character{
		Name:  "LowLevel",
		Level: 3, // Below steal requirement (level 5)
		Class: types.ClassThief,
		PCData: &types.PCData{
			Learned: map[string]int{
				"steal": 50, // Has learned but can't use due to level
			},
		},
	}

	// Should return 0 because level requirement not met
	if sys.GetSkill(ch, "steal") != 0 {
		t.Error("expected 0 for skill below level requirement")
	}

	// Level up and should now work
	ch.Level = 5
	if sys.GetSkill(ch, "steal") != 50 {
		t.Errorf("expected steal skill 50 after leveling, got %d", sys.GetSkill(ch, "steal"))
	}
}

func TestSkillSystemGetSkillNPC(t *testing.T) {
	sys := NewSkillSystem()

	// Create an NPC warrior
	npc := &types.Character{
		Name:  "Guard",
		Level: 20,
		Act:   types.ActNPC | types.ActWarrior,
	}

	// NPCs get level-based skills
	dodge := sys.GetSkill(npc, "dodge")
	if dodge != npc.Level {
		t.Errorf("expected NPC dodge = level (%d), got %d", npc.Level, dodge)
	}

	parry := sys.GetSkill(npc, "parry")
	if parry != npc.Level {
		t.Errorf("expected NPC parry = level (%d), got %d", npc.Level, parry)
	}

	// Second attack for warrior NPC
	secondAttack := sys.GetSkill(npc, "second attack")
	if secondAttack <= 0 {
		t.Error("expected NPC warrior to have second attack > 0")
	}

	// Hand to hand is based on level
	handToHand := sys.GetSkill(npc, "hand to hand")
	expected := 40 + npc.Level
	if handToHand != expected {
		t.Errorf("expected NPC hand to hand = %d, got %d", expected, handToHand)
	}
}

func TestSkillSystemGetSkillNPCThief(t *testing.T) {
	sys := NewSkillSystem()

	npc := &types.Character{
		Name:  "Thief",
		Level: 15,
		Act:   types.ActNPC | types.ActThief,
	}

	// Thief NPC should have backstab
	backstab := sys.GetSkill(npc, "backstab")
	if backstab <= 0 {
		t.Error("expected NPC thief to have backstab > 0")
	}

	// And sneak/hide
	sneak := sys.GetSkill(npc, "sneak")
	expected := npc.Level + 20
	if sneak != expected {
		t.Errorf("expected NPC sneak = %d, got %d", expected, sneak)
	}
}

func TestSkillSystemLearnSkill(t *testing.T) {
	sys := NewSkillSystem()

	ch := &types.Character{
		Name:  "TestPlayer",
		Level: 10,
		Class: types.ClassThief,
		PCData: &types.PCData{
			Learned: make(map[string]int),
		},
	}

	// Learn a skill
	ok := sys.LearnSkill(ch, "dodge", 25)
	if !ok {
		t.Error("expected LearnSkill to succeed")
	}
	if ch.PCData.Learned["dodge"] != 25 {
		t.Errorf("expected dodge = 25, got %d", ch.PCData.Learned["dodge"])
	}

	// Learn more of the same skill
	ok = sys.LearnSkill(ch, "dodge", 30)
	if !ok {
		t.Error("expected LearnSkill to succeed")
	}
	if ch.PCData.Learned["dodge"] != 55 {
		t.Errorf("expected dodge = 55, got %d", ch.PCData.Learned["dodge"])
	}

	// Cap at 100
	ok = sys.LearnSkill(ch, "dodge", 100)
	if !ok {
		t.Error("expected LearnSkill to succeed")
	}
	if ch.PCData.Learned["dodge"] != 100 {
		t.Errorf("expected dodge = 100 (capped), got %d", ch.PCData.Learned["dodge"])
	}
}

func TestSkillSystemLearnSkillClassRestriction(t *testing.T) {
	sys := NewSkillSystem()

	// Mage trying to learn bash (warrior only)
	ch := &types.Character{
		Name:  "TestMage",
		Level: 10,
		Class: types.ClassMage,
		PCData: &types.PCData{
			Learned: make(map[string]int),
		},
	}

	ok := sys.LearnSkill(ch, "bash", 50)
	if ok {
		t.Error("expected LearnSkill to fail for mage learning bash")
	}
	if ch.PCData.Learned["bash"] != 0 {
		t.Error("expected bash = 0 after failed learn")
	}
}

func TestSkillSystemLearnSkillLevelRestriction(t *testing.T) {
	sys := NewSkillSystem()

	// Low-level thief trying to learn steal (requires level 5)
	ch := &types.Character{
		Name:  "LowLevel",
		Level: 3,
		Class: types.ClassThief,
		PCData: &types.PCData{
			Learned: make(map[string]int),
		},
	}

	ok := sys.LearnSkill(ch, "steal", 50)
	if ok {
		t.Error("expected LearnSkill to fail due to level requirement")
	}
}

func TestSkillSystemLearnSkillNPC(t *testing.T) {
	sys := NewSkillSystem()

	npc := &types.Character{
		Name: "Guard",
		Act:  types.ActNPC,
	}

	// NPCs can't learn skills through this method
	ok := sys.LearnSkill(npc, "dodge", 50)
	if ok {
		t.Error("expected LearnSkill to fail for NPC")
	}
}

func TestSkillSystemGetLearnedPercent(t *testing.T) {
	sys := NewSkillSystem()

	ch := &types.Character{
		Name:  "TestPlayer",
		Level: 10,
		Class: types.ClassThief,
		PCData: &types.PCData{
			Learned: map[string]int{
				"dodge": 65,
			},
		},
	}

	if sys.GetLearnedPercent(ch, "dodge") != 65 {
		t.Errorf("expected 65%%, got %d", sys.GetLearnedPercent(ch, "dodge"))
	}
	if sys.GetLearnedPercent(ch, "parry") != 0 {
		t.Error("expected 0 for unlearned skill")
	}
}

func TestSkillSystemCheckImproveNPC(t *testing.T) {
	sys := NewSkillSystem()

	var outputCalled bool
	sys.Output = func(ch *types.Character, msg string) {
		outputCalled = true
	}

	npc := &types.Character{
		Name: "Guard",
		Act:  types.ActNPC,
	}

	// NPCs don't improve
	sys.CheckImprove(npc, "dodge", true, 1)
	if outputCalled {
		t.Error("expected no output for NPC")
	}
}

func TestSkillSystemCheckImproveUnknownSkill(t *testing.T) {
	sys := NewSkillSystem()

	var outputCalled bool
	sys.Output = func(ch *types.Character, msg string) {
		outputCalled = true
	}

	ch := &types.Character{
		Name:  "TestPlayer",
		Level: 10,
		Class: types.ClassThief,
		PCData: &types.PCData{
			Learned: map[string]int{},
		},
	}

	// Can't improve a skill you don't know
	sys.CheckImprove(ch, "dodge", true, 1)
	if outputCalled {
		t.Error("expected no output for unknown skill")
	}
}

func TestSkillSystemCheckImproveMaxed(t *testing.T) {
	sys := NewSkillSystem()

	var outputCalled bool
	sys.Output = func(ch *types.Character, msg string) {
		outputCalled = true
	}

	ch := &types.Character{
		Name:  "TestPlayer",
		Level: 10,
		Class: types.ClassThief,
		PCData: &types.PCData{
			Learned: map[string]int{
				"dodge": 100,
			},
		},
	}

	// Can't improve a maxed skill
	sys.CheckImprove(ch, "dodge", true, 1)
	if outputCalled {
		t.Error("expected no output for maxed skill")
	}
}

func TestSkillSystemOutputCallback(t *testing.T) {
	sys := NewSkillSystem()

	var messages []string
	sys.Output = func(ch *types.Character, msg string) {
		messages = append(messages, msg)
	}

	ch := &types.Character{
		Name:  "TestPlayer",
		Level: 50,
		Class: types.ClassThief,
		PCData: &types.PCData{
			Learned: map[string]int{
				"dodge": 50,
			},
		},
	}
	ch.PermStats[types.StatInt] = 25 // High intelligence for better improvement chance

	// Try many times to get an improvement (it's random)
	for i := 0; i < 1000; i++ {
		sys.CheckImprove(ch, "dodge", true, 1)
	}

	// At high int and 1000 attempts, we should have gotten at least one improvement
	if len(messages) == 0 {
		t.Error("expected at least one improvement message in 1000 attempts")
	}
}

func TestMinHelper(t *testing.T) {
	if min(5, 10) != 5 {
		t.Error("min(5, 10) should be 5")
	}
	if min(10, 5) != 5 {
		t.Error("min(10, 5) should be 5")
	}
	if min(5, 5) != 5 {
		t.Error("min(5, 5) should be 5")
	}
}

// Benchmark tests
func BenchmarkSkillLookup(b *testing.B) {
	sys := NewSkillSystem()
	ch := &types.Character{
		Name:  "TestPlayer",
		Level: 50,
		Class: types.ClassThief,
		PCData: &types.PCData{
			Learned: map[string]int{
				"dodge":    75,
				"backstab": 50,
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sys.GetSkill(ch, "dodge")
	}
}

func BenchmarkSkillRegistryFindByName(b *testing.B) {
	r := DefaultSkills()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.FindByName("backstab")
	}
}

func BenchmarkSkillRegistryFindByIndex(b *testing.B) {
	r := DefaultSkills()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.FindByIndex(10)
	}
}
