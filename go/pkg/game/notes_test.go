package game

import (
	"os"
	"testing"
	"time"

	"rotmud/pkg/types"
)

func TestNoteSystem(t *testing.T) {
	// Create temp directory for test data
	tmpDir, err := os.MkdirTemp("", "notes_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	ns := NewNoteSystem(tmpDir)

	t.Run("add note", func(t *testing.T) {
		note := &Note{
			Type:    NoteNote,
			Sender:  "TestPlayer",
			To:      "all",
			Subject: "Test Subject",
			Text:    "Test content\nLine 2",
		}
		ns.Add(note)

		if note.ID != 1 {
			t.Errorf("expected ID 1, got %d", note.ID)
		}
		if note.Date.IsZero() {
			t.Error("expected date to be set")
		}
		if note.DateStamp == "" {
			t.Error("expected date stamp to be set")
		}
	})

	t.Run("get notes for player", func(t *testing.T) {
		ch := types.NewCharacter("Reader")

		notes := ns.GetForPlayer(NoteNote, ch)
		if len(notes) != 1 {
			t.Errorf("expected 1 note, got %d", len(notes))
		}

		if notes[0].Subject != "Test Subject" {
			t.Errorf("expected 'Test Subject', got '%s'", notes[0].Subject)
		}
	})

	t.Run("private note visibility", func(t *testing.T) {
		// Add a private note
		privateNote := &Note{
			Type:    NoteNote,
			Sender:  "Sender",
			To:      "PrivateRecipient",
			Subject: "Private Message",
			Text:    "Secret content",
		}
		ns.Add(privateNote)

		// Wrong player shouldn't see it
		wrongPlayer := types.NewCharacter("WrongPerson")
		notes := ns.GetForPlayer(NoteNote, wrongPlayer)
		for _, n := range notes {
			if n.Subject == "Private Message" {
				t.Error("private note should not be visible to wrong player")
			}
		}

		// Right player should see it
		rightPlayer := types.NewCharacter("PrivateRecipient")
		notes = ns.GetForPlayer(NoteNote, rightPlayer)
		found := false
		for _, n := range notes {
			if n.Subject == "Private Message" {
				found = true
				break
			}
		}
		if !found {
			t.Error("private note should be visible to recipient")
		}

		// Sender should also see it
		sender := types.NewCharacter("Sender")
		notes = ns.GetForPlayer(NoteNote, sender)
		found = false
		for _, n := range notes {
			if n.Subject == "Private Message" {
				found = true
				break
			}
		}
		if !found {
			t.Error("private note should be visible to sender")
		}
	})

	t.Run("immortal visibility", func(t *testing.T) {
		immNote := &Note{
			Type:    NoteNote,
			Sender:  "Immortal",
			To:      "imm",
			Subject: "Immortal Only",
			Text:    "Staff message",
		}
		ns.Add(immNote)

		// Regular player shouldn't see it
		mortal := types.NewCharacter("Mortal")
		mortal.Level = 50
		notes := ns.GetForPlayer(NoteNote, mortal)
		for _, n := range notes {
			if n.Subject == "Immortal Only" {
				t.Error("imm note should not be visible to mortal")
			}
		}

		// Immortal should see it
		imm := types.NewCharacter("Immortal")
		imm.Level = 102 // Above LevelHero (101)
		notes = ns.GetForPlayer(NoteNote, imm)
		found := false
		for _, n := range notes {
			if n.Subject == "Immortal Only" {
				found = true
				break
			}
		}
		if !found {
			t.Error("imm note should be visible to immortal")
		}
	})

	t.Run("remove note", func(t *testing.T) {
		// Add note to remove
		noteToRemove := &Note{
			Type:    NoteNote,
			Sender:  "Admin",
			To:      "all",
			Subject: "To Be Removed",
			Text:    "This will be removed",
		}
		ns.Add(noteToRemove)
		id := noteToRemove.ID

		admin := types.NewCharacter("Admin")
		admin.Level = 110
		initialCount := len(ns.GetForPlayer(NoteNote, admin))

		// Remove it
		if !ns.Remove(NoteNote, id) {
			t.Error("expected remove to succeed")
		}

		// Verify count decreased by 1
		finalCount := len(ns.GetForPlayer(NoteNote, admin))
		if finalCount != initialCount-1 {
			t.Errorf("expected count %d after remove, got %d", initialCount-1, finalCount)
		}
	})

	t.Run("save and load", func(t *testing.T) {
		// Add a note to the second system
		newNote := &Note{
			Type:    NoteIdea,
			Sender:  "Tester",
			To:      "all",
			Subject: "Persistence Test",
			Text:    "This should persist",
		}
		ns.Add(newNote)

		// Save
		if err := ns.Save(); err != nil {
			t.Fatalf("save failed: %v", err)
		}

		// Create new system and load
		ns2 := NewNoteSystem(tmpDir)
		if err := ns2.Load(); err != nil {
			t.Fatalf("load failed: %v", err)
		}

		// Verify note exists
		ch := types.NewCharacter("Reader")
		notes := ns2.GetForPlayer(NoteIdea, ch)
		found := false
		for _, n := range notes {
			if n.Subject == "Persistence Test" {
				found = true
				break
			}
		}
		if !found {
			t.Error("note should persist after save/load")
		}
	})

	t.Run("unread count", func(t *testing.T) {
		// Get current count
		ch := types.NewCharacter("Counter")
		lastRead := time.Now()

		// Add new note after lastRead
		time.Sleep(10 * time.Millisecond)
		newNote := &Note{
			Type:    NoteNews,
			Sender:  "Admin",
			To:      "all",
			Subject: "New News",
			Text:    "Breaking news",
		}
		ns.Add(newNote)

		count := ns.GetUnreadCount(NoteNews, ch, lastRead)
		if count != 1 {
			t.Errorf("expected 1 unread, got %d", count)
		}
	})
}

func TestNoteEditor(t *testing.T) {
	ch := types.NewCharacter("Writer")

	t.Run("create editor", func(t *testing.T) {
		if HasNoteEditor(ch) {
			t.Error("should not have editor initially")
		}

		ed := GetNoteEditor(ch)
		if ed == nil {
			t.Fatal("expected editor")
		}

		if !HasNoteEditor(ch) {
			t.Error("should have editor after Get")
		}
	})

	t.Run("edit note", func(t *testing.T) {
		ed := GetNoteEditor(ch)
		ed.To = "all"
		ed.Subject = "Test Subject"
		ed.AddLine("Line 1")
		ed.AddLine("Line 2")

		if ed.To != "all" {
			t.Errorf("expected To 'all', got '%s'", ed.To)
		}
		if ed.Subject != "Test Subject" {
			t.Errorf("expected Subject 'Test Subject', got '%s'", ed.Subject)
		}

		text := ed.GetText()
		if text != "Line 1\r\nLine 2" {
			t.Errorf("unexpected text: %s", text)
		}
	})

	t.Run("clear editor", func(t *testing.T) {
		ClearNoteEditor(ch)

		if HasNoteEditor(ch) {
			t.Error("should not have editor after clear")
		}
	})
}

func TestNoteType(t *testing.T) {
	tests := []struct {
		t    NoteType
		want string
	}{
		{NoteNote, "note"},
		{NoteIdea, "idea"},
		{NoteNews, "news"},
		{NoteChanges, "changes"},
		{NotePenalty, "penalty"},
	}

	for _, tt := range tests {
		if got := tt.t.String(); got != tt.want {
			t.Errorf("NoteType(%d).String() = %s, want %s", tt.t, got, tt.want)
		}
	}
}
