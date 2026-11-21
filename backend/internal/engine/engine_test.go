package engine

import (
	"context"
	"testing"
	"time"

	"github.com/eddyhakimi/space-trade/backend/internal/game"
)

// Helpers shared by all engine tests live in this file.

// startEngine runs a game loop for the duration of the test.
func startEngine(t *testing.T, tick time.Duration) *Engine {
	t.Helper()
	return startEngineWith(t, game.NewWorld(1), tick)
}

// startEngineWith runs a game loop over a prepared world.
func startEngineWith(t *testing.T, w *game.World, tick time.Duration) *Engine {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	e := New(w, tick)
	go e.Run(ctx)
	t.Cleanup(func() {
		cancel()
		<-e.done
	})
	return e
}

func recv(t *testing.T, sub *Subscription) game.Snapshot {
	t.Helper()
	select {
	case snap, ok := <-sub.C:
		if !ok {
			t.Fatal("subscription closed")
		}
		return snap
	case <-time.After(2 * time.Second):
		t.Fatal("no snapshot within 2s")
	}
	return game.Snapshot{}
}

// drainsToClosed reads queued messages and reports whether the channel is closed.
func drainsToClosed(sub *Subscription) bool {
	for {
		select {
		case _, ok := <-sub.C:
			if !ok {
				return true
			}
		case <-time.After(100 * time.Millisecond):
			return false
		}
	}
}

func joinAndSubscribe(t *testing.T, e *Engine, name string) (game.PlayerID, *Subscription) {
	t.Helper()
	ctx := context.Background()
	pid, _, err := e.Join(ctx, name, "")
	if err != nil {
		t.Fatal(err)
	}
	sub, err := e.Subscribe(ctx, pid)
	if err != nil {
		t.Fatal(err)
	}
	return pid, sub
}

func TestTicksIncrease(t *testing.T) {
	e := startEngine(t, 10*time.Millisecond)
	_, sub := joinAndSubscribe(t, e, "Alice")

	last := recv(t, sub).Tick
	for range 3 {
		s := recv(t, sub)
		if s.Tick <= last {
			t.Fatalf("tick went from %d to %d", last, s.Tick)
		}
		if s.Time < float64(s.Tick) || s.Time > float64(s.Tick)+1 {
			t.Fatalf("time %v not within tick %d", s.Time, s.Tick)
		}
		last = s.Tick
	}
}
