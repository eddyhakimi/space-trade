package engine

import (
	"context"
	"testing"
	"time"

	"github.com/eddyhakimi/space-trade/backend/internal/game"
)

func TestTravelAppearsInNextState(t *testing.T) {
	e := startEngine(t, time.Hour) // no ticks: only command broadcasts
	pid, sub := joinAndSubscribe(t, e, "Alice")
	first := recv(t, sub)
	ship := first.Ships[0]
	if ship.Status != game.Docked {
		t.Fatalf("new ship not docked: %+v", ship)
	}
	dest := first.Stations[0]
	if dest.ID == ship.At {
		dest = first.Stations[1]
	}

	if err := e.Travel(context.Background(), pid, ship.ID, dest.X, dest.Y); err != nil {
		t.Fatal(err)
	}
	st := recv(t, sub)
	got := st.Ships[0]
	if got.Status != game.Travelling || got.To != dest.ID {
		t.Fatalf("after Travel: %+v", got)
	}
	// time is the tick plus the fraction of the current tick, so the
	// client can interpolate from the exact moment of the snapshot.
	if st.Time < float64(st.Tick) || st.Time > float64(st.Tick)+1 {
		t.Fatalf("time %v not within tick %d", st.Time, st.Tick)
	}
	if d := st.Time - got.DepartAt; d < 0 || d > 1e-3 {
		t.Fatalf("leg departs at %v, snapshot time %v", got.DepartAt, st.Time)
	}

	// A rejected command returns its error and broadcasts nothing.
	if err := e.Travel(context.Background(), pid, ship.ID, -5, 100); err != game.ErrOutOfBounds {
		t.Fatalf("out-of-bounds Travel: %v", err)
	}
	select {
	case <-sub.C:
		t.Fatal("rejected command caused a broadcast")
	case <-time.After(50 * time.Millisecond):
	}
}
