package help

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewSystem(t *testing.T) {
	sys := NewSystem()
	if sys.entries == nil {
		t.Error("expected entries map to be initialized")
	}
	if sys.Count() != 0 {
		t.Errorf("expected count 0, got %d", sys.Count())
	}
}

func TestSystemRegister(t *testing.T) {
	sys := NewSystem()

	entry := &Entry{
		Keywords:    []string{"look", "l"},
		Description: "Look around the room.",
	}
	sys.Register(entry)

	if sys.Count() != 1 {
		t.Errorf("expected count 1, got %d", sys.Count())
	}

	// Should find by both keywords
	found := sys.Find("look")
	if found != entry {
		t.Error("expected to find entry by 'look'")
	}

	found = sys.Find("l")
	if found != entry {
		t.Error("expected to find entry by 'l'")
	}
}

func TestSystemFind(t *testing.T) {
	sys := NewSystem()

	entry1 := &Entry{Keywords: []string{"north"}, Description: "Go north"}
	entry2 := &Entry{Keywords: []string{"northeast"}, Description: "Go northeast"}
	sys.Register(entry1)
	sys.Register(entry2)

	// Exact match
	found := sys.Find("north")
	if found != entry1 {
		t.Error("expected exact match for 'north'")
	}

	// Prefix match (should find north first if alphabetically)
	found = sys.Find("nor")
	if found == nil {
		t.Error("expected prefix match for 'nor'")
	}

	// Not found
	found = sys.Find("xyz")
	if found != nil {
		t.Error("expected nil for non-existent keyword")
	}
}

func TestSystemFindCaseInsensitive(t *testing.T) {
	sys := NewSystem()

	entry := &Entry{Keywords: []string{"look"}, Description: "Look around"}
	sys.Register(entry)

	if sys.Find("LOOK") != entry {
		t.Error("expected case-insensitive match for 'LOOK'")
	}
	if sys.Find("Look") != entry {
		t.Error("expected case-insensitive match for 'Look'")
	}
}

func TestSystemFindAll(t *testing.T) {
	sys := NewSystem()

	entry1 := &Entry{Keywords: []string{"north"}, Description: "Go north"}
	entry2 := &Entry{Keywords: []string{"northeast"}, Description: "Go northeast"}
	entry3 := &Entry{Keywords: []string{"south"}, Description: "Go south"}
	sys.Register(entry1)
	sys.Register(entry2)
	sys.Register(entry3)

	matches := sys.FindAll("nor")
	if len(matches) != 2 {
		t.Errorf("expected 2 matches for 'nor', got %d", len(matches))
	}

	matches = sys.FindAll("s")
	if len(matches) != 1 {
		t.Errorf("expected 1 match for 's', got %d", len(matches))
	}

	matches = sys.FindAll("xyz")
	if len(matches) != 0 {
		t.Errorf("expected 0 matches for 'xyz', got %d", len(matches))
	}
}

func TestSystemAllKeywords(t *testing.T) {
	sys := NewSystem()

	entry := &Entry{Keywords: []string{"look", "l"}, Description: "Look"}
	sys.Register(entry)

	keywords := sys.AllKeywords()
	if len(keywords) != 2 {
		t.Errorf("expected 2 keywords, got %d", len(keywords))
	}
}

func TestEntryFormat(t *testing.T) {
	entry := &Entry{
		Keywords:    []string{"look"},
		Syntax:      "look [target]",
		Description: "Look at your surroundings or a specific target.",
		SeeAlso:     []string{"examine", "scan"},
	}

	output := entry.Format()

	// Check contains expected elements
	if !containsString(output, "LOOK") {
		t.Error("expected output to contain 'LOOK'")
	}
	if !containsString(output, "Syntax: look [target]") {
		t.Error("expected output to contain syntax")
	}
	if !containsString(output, "surroundings") {
		t.Error("expected output to contain description")
	}
	if !containsString(output, "See also: examine, scan") {
		t.Error("expected output to contain see also")
	}
}

func TestEntryFormatMinimal(t *testing.T) {
	entry := &Entry{
		Keywords:    []string{"test"},
		Description: "A test entry.",
	}

	output := entry.Format()

	// Should not have empty sections
	if containsString(output, "Syntax:") {
		t.Error("should not have syntax section when empty")
	}
	if containsString(output, "See also:") {
		t.Error("should not have see also section when empty")
	}
}

func TestSystemLoadFile(t *testing.T) {
	// Create a temp file with TOML content
	tmpDir := t.TempDir()
	helpFile := filepath.Join(tmpDir, "commands.toml")

	content := `
[[help]]
keywords = ["north", "n"]
syntax = "north"
description = "Move north to the next room."

[[help]]
keywords = ["south", "s"]
syntax = "south"
description = "Move south to the next room."
see_also = ["north", "east", "west"]
`
	if err := os.WriteFile(helpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	sys := NewSystem()
	if err := sys.LoadFile(helpFile); err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}

	if sys.Count() != 2 {
		t.Errorf("expected 2 entries, got %d", sys.Count())
	}

	north := sys.Find("north")
	if north == nil {
		t.Error("expected to find 'north'")
	}
	if north.Syntax != "north" {
		t.Errorf("expected syntax 'north', got %q", north.Syntax)
	}

	// Alias lookup
	n := sys.Find("n")
	if n != north {
		t.Error("expected 'n' to be alias for 'north'")
	}

	south := sys.Find("south")
	if south == nil {
		t.Error("expected to find 'south'")
	}
	if len(south.SeeAlso) != 3 {
		t.Errorf("expected 3 see_also entries, got %d", len(south.SeeAlso))
	}
}

func TestSystemLoadDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple help files
	content1 := `
[[help]]
keywords = ["north"]
description = "Go north."
`
	content2 := `
[[help]]
keywords = ["south"]
description = "Go south."
`
	if err := os.WriteFile(filepath.Join(tmpDir, "movement.toml"), []byte(content1), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "movement2.toml"), []byte(content2), 0644); err != nil {
		t.Fatal(err)
	}
	// Non-toml file should be ignored
	if err := os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("ignored"), 0644); err != nil {
		t.Fatal(err)
	}

	sys := NewSystem()
	if err := sys.LoadDir(tmpDir); err != nil {
		t.Fatalf("LoadDir failed: %v", err)
	}

	if sys.Count() != 2 {
		t.Errorf("expected 2 entries, got %d", sys.Count())
	}
}

func TestSystemLoadFileInvalid(t *testing.T) {
	tmpDir := t.TempDir()
	badFile := filepath.Join(tmpDir, "bad.toml")

	// Invalid TOML
	if err := os.WriteFile(badFile, []byte("not valid { toml"), 0644); err != nil {
		t.Fatal(err)
	}

	sys := NewSystem()
	err := sys.LoadFile(badFile)
	if err == nil {
		t.Error("expected error loading invalid TOML")
	}
}

func TestSystemLoadFileNotFound(t *testing.T) {
	sys := NewSystem()
	err := sys.LoadFile("/nonexistent/path/help.toml")
	if err == nil {
		t.Error("expected error loading nonexistent file")
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStringHelper(s, substr))
}

func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
