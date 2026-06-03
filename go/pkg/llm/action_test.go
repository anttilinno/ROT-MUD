package llm

import "testing"

func TestValidateAcceptsNormalSpeech(t *testing.T) {
	good := []Action{
		{Tool: "say", Line: "Ah, young one. Mind the goblins to the north."},
		{Tool: "emote", Line: "leans on his staff and sighs wearily."},
		{Tool: "refuse", Line: "I'll not speak of such things, stranger."},
	}
	for _, a := range good {
		if err := a.Validate(); err != nil {
			t.Errorf("Validate(%q) = %v, want nil", a.Line, err)
		}
	}
}

func TestValidateRejectsGarbage(t *testing.T) {
	bad := []struct {
		name string
		line string
	}{
		{"json fence", "Charity work indeed. ```json`nicoslander"},
		{"email", "seek user@example.com for aid"},
		{"backtick", "the south `lurks`"},
		{"braces", "{\"tool\":\"say\"}"},
		{"http", "visit http://midgaard for more"},
		{"repetition", "Hassan lurks south. Hassan lurks south. Hassan lurks south. Hassan lurks south."},
		{"empty", "   "},
		{"single token", "1"},
		{"no words", "!!! ..."},
	}
	for _, c := range bad {
		a := Action{Tool: "say", Line: c.line}
		if err := a.Validate(); err == nil {
			t.Errorf("%s: Validate(%q) = nil, want error", c.name, c.line)
		}
	}
}
