package llm

import (
	"testing"
	"time"
)

// fakeClock lets tests advance time without sleeping.
type fakeClock struct{ t time.Time }

func (f *fakeClock) now() time.Time      { return f.t }
func (f *fakeClock) add(d time.Duration) { f.t = f.t.Add(d) }

func newTestBreaker(clk *fakeClock) *Breaker {
	b := NewBreaker()
	b.now = clk.now
	return b
}

func TestBreaker_StaysClosedBelowThreshold(t *testing.T) {
	b := newTestBreaker(&fakeClock{})
	// 4 failures in a full window of 10 → below threshold of 5.
	for i := 0; i < 4; i++ {
		b.Record(true)
	}
	for i := 0; i < 6; i++ {
		b.Record(false)
	}
	if b.IsOpen() {
		t.Fatal("breaker opened below threshold")
	}
	if !b.Allow() {
		t.Fatal("closed breaker should allow calls")
	}
}

func TestBreaker_OpensAtThreshold(t *testing.T) {
	b := newTestBreaker(&fakeClock{})
	for i := 0; i < 5; i++ {
		b.Record(true)
	}
	for i := 0; i < 5; i++ {
		b.Record(false)
	}
	if !b.IsOpen() {
		t.Fatal("breaker should open with 5/10 failures")
	}
	if b.Allow() {
		t.Fatal("open breaker should block calls")
	}
}

func TestBreaker_HalfOpenClosesOnSuccess(t *testing.T) {
	clk := &fakeClock{t: time.Unix(0, 0)}
	b := newTestBreaker(clk)
	for i := 0; i < 10; i++ {
		b.Record(true)
	}
	if !b.IsOpen() {
		t.Fatal("should be open")
	}
	// Before cooldown: blocked.
	if b.Allow() {
		t.Fatal("should block during cooldown")
	}
	// After cooldown: one trial admitted.
	clk.add(61 * time.Second)
	if !b.Allow() {
		t.Fatal("should admit trial after cooldown")
	}
	if b.Allow() {
		t.Fatal("should not admit a second concurrent trial")
	}
	// Trial succeeds → breaker closes.
	b.Record(false)
	if b.IsOpen() {
		t.Fatal("breaker should close after successful trial")
	}
	if !b.Allow() {
		t.Fatal("closed breaker should allow")
	}
}

func TestBreaker_HalfOpenReopensOnFailure(t *testing.T) {
	clk := &fakeClock{t: time.Unix(0, 0)}
	b := newTestBreaker(clk)
	for i := 0; i < 10; i++ {
		b.Record(true)
	}
	clk.add(61 * time.Second)
	if !b.Allow() {
		t.Fatal("should admit trial")
	}
	b.Record(true) // trial fails → re-open
	if !b.IsOpen() {
		t.Fatal("breaker should re-open after failed trial")
	}
	if b.Allow() {
		t.Fatal("should block again immediately after re-open")
	}
}
