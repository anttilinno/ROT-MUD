package llm

import (
	"context"
	"errors"
	"testing"
	"time"
)

// stubChatter returns a scripted reply/error and optionally blocks on a gate so
// tests can control in-flight timing.
type stubChatter struct {
	act  Action
	err  error
	gate chan struct{} // if non-nil, Chat blocks until it receives
}

func (s *stubChatter) Chat(ctx context.Context, persona, playerName, playerSay string) (Action, error) {
	if s.gate != nil {
		select {
		case <-s.gate:
		case <-ctx.Done():
			return Action{}, ctx.Err()
		}
	}
	return s.act, s.err
}

func testPool(t *testing.T, c chatter) *Pool {
	t.Helper()
	p := newPool(Config{Enabled: true, Workers: 2, Queue: 8, Timeout: time.Second}, c)
	p.Start()
	t.Cleanup(p.Stop)
	return p
}

func waitResult(t *testing.T, p *Pool) Result {
	t.Helper()
	select {
	case r := <-p.Results():
		return r
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for result")
		return Result{}
	}
}

func TestPool_SuccessPath(t *testing.T) {
	p := testPool(t, &stubChatter{act: Action{Tool: "say", Line: "Welcome."}})
	if !p.Submit(Request{Key: "mob1", PlayerSay: "hi"}) {
		t.Fatal("Submit dropped a valid request")
	}
	r := waitResult(t, p)
	if r.Err != nil {
		t.Fatalf("unexpected err: %v", r.Err)
	}
	if r.Key != "mob1" || r.Action.Line != "Welcome." {
		t.Fatalf("bad result: %+v", r)
	}
}

func TestPool_InvalidActionIsError(t *testing.T) {
	// Model returns an illegal tool → Pool surfaces an error so the caller
	// falls back to scripted behavior.
	p := testPool(t, &stubChatter{act: Action{Tool: "cast_fireball", Line: "boom"}})
	p.Submit(Request{Key: "mob1", PlayerSay: "hi"})
	r := waitResult(t, p)
	if r.Err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestPool_TransportErrorIsError(t *testing.T) {
	p := testPool(t, &stubChatter{err: errors.New("connection refused")})
	p.Submit(Request{Key: "mob1", PlayerSay: "hi"})
	if r := waitResult(t, p); r.Err == nil {
		t.Fatal("expected transport error, got nil")
	}
}

func TestPool_DropDuplicateInflight(t *testing.T) {
	gate := make(chan struct{})
	p := testPool(t, &stubChatter{act: Action{Tool: "say", Line: "ok"}, gate: gate})

	if !p.Submit(Request{Key: "mob1", PlayerSay: "first"}) {
		t.Fatal("first submit dropped")
	}
	// Worker is now blocked in Chat with mob1 in flight. A second submit for the
	// same key must be dropped.
	if p.Submit(Request{Key: "mob1", PlayerSay: "second"}) {
		t.Fatal("duplicate in-flight request was not dropped")
	}
	close(gate) // let the first finish
	if r := waitResult(t, p); r.Err != nil {
		t.Fatalf("unexpected err: %v", r.Err)
	}
	if got := p.Stats().Drops; got != 1 {
		t.Fatalf("expected 1 drop, got %d", got)
	}
}

func TestPool_DisabledDropsEverything(t *testing.T) {
	p := newPool(Config{Enabled: false, Workers: 1, Queue: 4, Timeout: time.Second}, &stubChatter{})
	p.Start()
	t.Cleanup(p.Stop)
	if p.Submit(Request{Key: "mob1"}) {
		t.Fatal("disabled pool accepted a request")
	}
}

func TestPool_BreakerOpensAndDrops(t *testing.T) {
	// Force the breaker open by feeding 10 failures, then confirm Submit drops.
	b := NewBreaker()
	for i := 0; i < 10; i++ {
		b.Record(true)
	}
	if !b.IsOpen() {
		t.Fatal("breaker should be open after 10 failures")
	}

	p := newPool(Config{Enabled: true, Workers: 1, Queue: 4, Timeout: time.Second}, &stubChatter{})
	p.breaker = b
	p.Start()
	t.Cleanup(p.Stop)

	if p.Submit(Request{Key: "mob1"}) {
		t.Fatal("breaker open but Submit accepted request")
	}
	if p.Stats().Drops == 0 {
		t.Fatal("expected a drop recorded")
	}
}
