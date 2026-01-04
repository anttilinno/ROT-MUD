package game

import (
	"testing"
)

func TestParseTriggerType(t *testing.T) {
	tests := []struct {
		input    string
		expected MOBprogTrigger
	}{
		{"speech", TriggerSpeech},
		{"SPEECH", TriggerSpeech},
		{"Speech", TriggerSpeech},
		{"act", TriggerAct},
		{"fight", TriggerFight},
		{"death", TriggerDeath},
		{"entry", TriggerEntry},
		{"greet", TriggerGreet},
		{"give", TriggerGive},
		{"bribe", TriggerBribe},
		{"unknown", TriggerSpeech}, // default
		{"", TriggerSpeech},        // default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseTriggerType(tt.input)
			if result != tt.expected {
				t.Errorf("ParseTriggerType(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTriggerTypeString(t *testing.T) {
	tests := []struct {
		input    MOBprogTrigger
		expected string
	}{
		{TriggerSpeech, "speech"},
		{TriggerAct, "act"},
		{TriggerFight, "fight"},
		{TriggerDeath, "death"},
		{TriggerEntry, "entry"},
		{TriggerGreet, "greet"},
		{TriggerGive, "give"},
		{TriggerBribe, "bribe"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := TriggerTypeString(tt.input)
			if result != tt.expected {
				t.Errorf("TriggerTypeString(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestAddProgramFromData(t *testing.T) {
	mps := NewMOBprogSystem()
	mps.ClearPrograms() // Clear default programs

	// Add a program using the data-based method
	mps.AddProgramFromData(
		1000,
		"speech",
		"hello",
		[]string{"say Greetings!", "emote waves."},
	)

	// Verify it was added
	progs := mps.GetPrograms(1000)
	if len(progs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(progs))
	}

	prog := progs[0]
	if prog.MobVnum != 1000 {
		t.Errorf("expected MobVnum 1000, got %d", prog.MobVnum)
	}
	if prog.Trigger != TriggerSpeech {
		t.Errorf("expected TriggerSpeech, got %v", prog.Trigger)
	}
	if prog.Phrase != "hello" {
		t.Errorf("expected phrase 'hello', got %q", prog.Phrase)
	}
	if len(prog.Commands) != 2 {
		t.Errorf("expected 2 commands, got %d", len(prog.Commands))
	}
}

func TestClearPrograms(t *testing.T) {
	mps := NewMOBprogSystem()

	// Should have default programs
	initialCount := 0
	for _, progs := range mps.programs {
		initialCount += len(progs)
	}
	if initialCount == 0 {
		t.Fatal("expected default programs")
	}

	// Clear all programs
	mps.ClearPrograms()

	// Should have no programs
	afterCount := 0
	for _, progs := range mps.programs {
		afterCount += len(progs)
	}
	if afterCount != 0 {
		t.Errorf("expected 0 programs after clear, got %d", afterCount)
	}
}

func TestMultipleProgramsForSameMob(t *testing.T) {
	mps := NewMOBprogSystem()
	mps.ClearPrograms()

	// Add multiple programs for the same mob
	mps.AddProgramFromData(2000, "greet", "50", []string{"say Welcome!"})
	mps.AddProgramFromData(2000, "speech", "buy", []string{"say What would you like to buy?"})
	mps.AddProgramFromData(2000, "death", "100", []string{"mpecho The merchant falls!"})

	progs := mps.GetPrograms(2000)
	if len(progs) != 3 {
		t.Fatalf("expected 3 programs, got %d", len(progs))
	}

	// Verify each trigger type is present
	triggers := make(map[MOBprogTrigger]bool)
	for _, prog := range progs {
		triggers[prog.Trigger] = true
	}
	if !triggers[TriggerGreet] {
		t.Error("expected greet trigger")
	}
	if !triggers[TriggerSpeech] {
		t.Error("expected speech trigger")
	}
	if !triggers[TriggerDeath] {
		t.Error("expected death trigger")
	}
}
