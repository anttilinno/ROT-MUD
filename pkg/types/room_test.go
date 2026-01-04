package types

import "testing"

func TestExit(t *testing.T) {
	t.Run("NewExit creates exit with correct values", func(t *testing.T) {
		exit := NewExit(DirNorth, 3001)
		if exit.Direction != DirNorth {
			t.Errorf("expected direction DirNorth, got %v", exit.Direction)
		}
		if exit.ToVnum != 3001 {
			t.Errorf("expected ToVnum 3001, got %d", exit.ToVnum)
		}
	})

	t.Run("Exit flags work correctly", func(t *testing.T) {
		exit := NewExit(DirNorth, 3001)
		exit.Flags.Set(ExitIsDoor)
		exit.Flags.Set(ExitClosed)

		if !exit.IsDoor() {
			t.Error("expected exit to be a door")
		}
		if !exit.IsClosed() {
			t.Error("expected exit to be closed")
		}
		if exit.IsLocked() {
			t.Error("expected exit to not be locked")
		}
	})

	t.Run("Exit can be opened and closed", func(t *testing.T) {
		exit := NewExit(DirNorth, 3001)
		exit.Flags.Set(ExitIsDoor)
		exit.Flags.Set(ExitClosed)

		exit.Open()
		if exit.IsClosed() {
			t.Error("expected exit to be open after Open()")
		}

		exit.Close()
		if !exit.IsClosed() {
			t.Error("expected exit to be closed after Close()")
		}
	})
}

func TestRoom(t *testing.T) {
	t.Run("NewRoom creates room with correct values", func(t *testing.T) {
		room := NewRoom(3001, "The Temple", "A grand temple.")
		if room.Vnum != 3001 {
			t.Errorf("expected vnum 3001, got %d", room.Vnum)
		}
		if room.Name != "The Temple" {
			t.Errorf("expected name 'The Temple', got '%s'", room.Name)
		}
		if room.Description != "A grand temple." {
			t.Errorf("expected description 'A grand temple.', got '%s'", room.Description)
		}
	})

	t.Run("Room flags work correctly", func(t *testing.T) {
		room := NewRoom(3001, "Test", "Test room")
		room.Flags.Set(RoomSafe)
		room.Flags.Set(RoomIndoors)

		if !room.IsSafe() {
			t.Error("expected room to be safe")
		}
		if !room.Flags.Has(RoomIndoors) {
			t.Error("expected room to be indoors")
		}
		if room.IsDark() {
			t.Error("expected room to not be dark")
		}
	})

	t.Run("SetExit and GetExit work correctly", func(t *testing.T) {
		room := NewRoom(3001, "Test", "Test room")
		exit := NewExit(DirNorth, 3002)
		room.SetExit(DirNorth, exit)

		retrieved := room.GetExit(DirNorth)
		if retrieved == nil {
			t.Fatal("expected to retrieve exit")
		}
		if retrieved.ToVnum != 3002 {
			t.Errorf("expected ToVnum 3002, got %d", retrieved.ToVnum)
		}
	})

	t.Run("GetExit returns nil for no exit", func(t *testing.T) {
		room := NewRoom(3001, "Test", "Test room")
		exit := room.GetExit(DirNorth)
		if exit != nil {
			t.Error("expected nil for non-existent exit")
		}
	})

	t.Run("ExitDirections returns directions with exits", func(t *testing.T) {
		room := NewRoom(3001, "Test", "Test room")
		room.SetExit(DirNorth, NewExit(DirNorth, 3002))
		room.SetExit(DirSouth, NewExit(DirSouth, 3000))

		dirs := room.ExitDirections()
		if len(dirs) != 2 {
			t.Errorf("expected 2 exit directions, got %d", len(dirs))
		}
	})

	t.Run("Sector type defaults to inside", func(t *testing.T) {
		room := NewRoom(3001, "Test", "Test room")
		if room.Sector != SectInside {
			t.Errorf("expected sector SectInside, got %v", room.Sector)
		}
	})

	t.Run("Room can track people", func(t *testing.T) {
		room := NewRoom(3001, "Test", "Test room")
		if room.PeopleCount() != 0 {
			t.Error("expected 0 people in new room")
		}
	})
}

func TestExtraDescription(t *testing.T) {
	t.Run("ExtraDescription stores keyword and text", func(t *testing.T) {
		ed := &ExtraDescription{
			Keywords:    []string{"statue", "marble"},
			Description: "A beautiful marble statue.",
		}

		if len(ed.Keywords) != 2 {
			t.Errorf("expected 2 keywords, got %d", len(ed.Keywords))
		}
		if ed.Description != "A beautiful marble statue." {
			t.Errorf("unexpected description: %s", ed.Description)
		}
	})

	t.Run("Room extra descriptions", func(t *testing.T) {
		room := NewRoom(3001, "Test", "Test room")
		room.ExtraDescriptions = append(room.ExtraDescriptions, &ExtraDescription{
			Keywords:    []string{"painting"},
			Description: "A portrait of a king.",
		})

		if len(room.ExtraDescriptions) != 1 {
			t.Errorf("expected 1 extra description, got %d", len(room.ExtraDescriptions))
		}
	})
}
