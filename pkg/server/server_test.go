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
