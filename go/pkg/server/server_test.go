package server

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"rotmud/pkg/types"
)

func TestNewServer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	srv := New(logger)

	if srv == nil {
		t.Fatal("New should return a non-nil server")
	}

	if srv.GameLoop == nil {
		t.Error("GameLoop should be initialized")
	}

	if srv.Dispatcher == nil {
		t.Error("Dispatcher should be initialized")
	}

	if srv.sessions == nil {
		t.Error("sessions map should be initialized")
	}
}

func TestSession(t *testing.T) {
	sess := &Session{
		Character: types.NewCharacter("TestPlayer"),
	}

	if sess.Character.Name != "TestPlayer" {
		t.Errorf("expected character name 'TestPlayer', got %q", sess.Character.Name)
	}
}

func TestSendToCharacter(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	srv := New(logger)

	ch := types.NewCharacter("TestPlayer")

	// SendToCharacter should not panic when character has no session
	srv.SendToCharacter(ch, "Hello, world!\r\n")
}

func TestServerStatistics(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	srv := New(logger)

	// Check start time is set
	if srv.startTime.IsZero() {
		t.Error("startTime should be set")
	}

	// Start time should be recent
	if time.Since(srv.startTime) > time.Second {
		t.Error("startTime should be recent")
	}
}

func TestSessionCount(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	srv := New(logger)

	// Initially no sessions
	srv.mu.RLock()
	count := len(srv.sessions)
	srv.mu.RUnlock()

	if count != 0 {
		t.Errorf("expected 0 sessions, got %d", count)
	}
}

func TestDescriptor(t *testing.T) {
	desc := &types.Descriptor{
		State: types.ConPlaying,
		Host:  "127.0.0.1",
	}

	if desc.State != types.ConPlaying {
		t.Error("expected state to be ConPlaying")
	}

	if desc.Host != "127.0.0.1" {
		t.Errorf("expected host '127.0.0.1', got %q", desc.Host)
	}

	if !desc.IsPlaying() {
		t.Error("IsPlaying should return true")
	}
}

func TestDescriptorNewDescriptor(t *testing.T) {
	desc := types.NewDescriptor("localhost")

	if desc.Host != "localhost" {
		t.Errorf("expected host 'localhost', got %q", desc.Host)
	}

	if desc.State != types.ConGetName {
		t.Error("expected initial state to be ConGetName")
	}

	if desc.IsPlaying() {
		t.Error("IsPlaying should return false initially")
	}
}

func TestDescriptorEditorState(t *testing.T) {
	desc := types.NewDescriptor("localhost")

	if desc.InEditor() {
		t.Error("InEditor should return false initially")
	}

	desc.Editor = types.EditorRoom
	if !desc.InEditor() {
		t.Error("InEditor should return true when editor is set")
	}
}

func TestInitializeNewCharacter(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	srv := New(logger)

	// Test warrior class
	ch := types.NewCharacter("TestWarrior")
	ch.Class = types.ClassWarrior
	ch.PCData = &types.PCData{Learned: make(map[string]int)}
	ch.PermStats = [5]int{15, 10, 10, 12, 14} // Some base stats

	srv.Login.initializeNewCharacter(ch)

	if ch.HitRoll != 5 {
		t.Errorf("expected warrior hitroll 5, got %d", ch.HitRoll)
	}
	if ch.DamRoll != 5 {
		t.Errorf("expected warrior damroll 5, got %d", ch.DamRoll)
	}

	// Test thief class
	ch2 := types.NewCharacter("TestThief")
	ch2.Class = types.ClassThief
	ch2.PCData = &types.PCData{Learned: make(map[string]int)}
	ch2.PermStats = [5]int{10, 10, 10, 15, 12}

	srv.Login.initializeNewCharacter(ch2)

	if ch2.HitRoll != 3 {
		t.Errorf("expected thief hitroll 3, got %d", ch2.HitRoll)
	}
	if ch2.DamRoll != 3 {
		t.Errorf("expected thief damroll 3, got %d", ch2.DamRoll)
	}

	// Test mage class (default case)
	ch3 := types.NewCharacter("TestMage")
	ch3.Class = types.ClassMage
	ch3.PCData = &types.PCData{Learned: make(map[string]int)}
	ch3.PermStats = [5]int{10, 18, 12, 10, 10}

	srv.Login.initializeNewCharacter(ch3)

	if ch3.HitRoll != 2 {
		t.Errorf("expected mage hitroll 2, got %d", ch3.HitRoll)
	}
	if ch3.DamRoll != 2 {
		t.Errorf("expected mage damroll 2, got %d", ch3.DamRoll)
	}
}

func TestLoadWorldWithFountain(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	srv := New(logger)
	srv.DataPath = "../../data"

	err := srv.LoadWorld()
	if err != nil {
		t.Fatalf("failed to load world: %v", err)
	}

	// Check Temple Square room (3005) has obj_resets loaded
	templeSquare := srv.World.GetRoom(3005)
	if templeSquare == nil {
		t.Fatal("expected room 3005 (Temple Square)")
	}

	// Check obj_resets are loaded
	if len(templeSquare.ObjResets) == 0 {
		t.Fatal("expected obj_resets in Temple Square")
	}

	// Find the fountain in any room (it has max=1 globally, so only one spawns)
	// Both rooms 3005 and 3141 have the fountain reset, first one processed wins
	foundFountain := false
	var fountainRoom *types.Room
	for _, room := range srv.World.Rooms {
		for _, obj := range room.Objects {
			if obj.Vnum == 3135 {
				foundFountain = true
				fountainRoom = room
				if obj.ItemType != types.ItemTypeFountain {
					t.Errorf("expected fountain type, got %v", obj.ItemType)
				}
				break
			}
		}
		if foundFountain {
			break
		}
	}

	if !foundFountain {
		t.Error("expected fountain (vnum 3135) to be spawned somewhere in the world")
	} else {
		t.Logf("Fountain spawned in room %d (%s)", fountainRoom.Vnum, fountainRoom.Name)
	}
}
