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

	if err := e.Travel(context.Background(), pid, ship.ID, dest.X, dest.Y, false); err != nil {
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
	if err := e.Travel(context.Background(), pid, ship.ID, -5, 100, false); err != game.ErrOutOfBounds {
		t.Fatalf("out-of-bounds Travel: %v", err)
	}
	select {
	case <-sub.C:
		t.Fatal("rejected command caused a broadcast")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestTradeAppearsInNextState(t *testing.T) {
	e := startEngine(t, time.Hour) // no ticks: only command broadcasts
	pid, sub := joinAndSubscribe(t, e, "Alice")
	first := recv(t, sub)
	ship := first.Ships[0]
	var l game.Listing
	for _, st := range first.Stations {
		if st.ID != ship.At {
			continue
		}
		for _, sl := range st.Market {
			if sl.Buy > 0 {
				l = sl
			}
		}
	}
	if l.Buy == 0 {
		t.Fatalf("station %s sells nothing", ship.At)
	}

	if err := e.Buy(context.Background(), pid, ship.ID, l.Good, 2); err != nil {
		t.Fatal(err)
	}
	st := recv(t, sub)
	if c := st.Players[0].Credits; c != game.StartingCredits-l.BuyCost(2) {
		t.Fatalf("credits after buying 2 for %d: %d", l.BuyCost(2), c)
	}
	if got := st.Ships[0].Cargo[l.Good]; got != 2 {
		t.Fatalf("cargo after buy: %v", st.Ships[0].Cargo)
	}
	// The price rises with the pressure (cheap goods may need a few units to move a whole credit).
	for _, sl := range st.Stations {
		for _, after := range sl.Market {
			if sl.ID == ship.At && after.Good == l.Good && (after.Pressure != 2 || after.Buy < l.Buy) {
				t.Fatalf("market after buying 2: %+v, before %+v", after, l)
			}
		}
	}

	// A rejected trade returns its error and broadcasts nothing.
	if err := e.Sell(context.Background(), pid, ship.ID, l.Good, 5); err != game.ErrNotEnoughCargo {
		t.Fatalf("selling more than held: %v", err)
	}
	select {
	case <-sub.C:
		t.Fatal("rejected trade caused a broadcast")
	case <-time.After(50 * time.Millisecond):
	}
}
