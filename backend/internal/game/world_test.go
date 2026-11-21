package game

import (
	"math"
	"testing"
)

// Helpers shared by all game tests live in this file.

// newPlayer adds a player and returns it with its (only) ship.
func newPlayer(t *testing.T, w *World, name string) (*Player, *Ship) {
	t.Helper()
	p, err := w.AddPlayer(name)
	if err != nil {
		t.Fatalf("AddPlayer(%q): %v", name, err)
	}
	for _, s := range w.Ships {
		if s.OwnerID == p.ID {
			return p, s
		}
	}
	t.Fatalf("player %s has no ship", p.ID)
	return nil, nil
}

// otherStation returns any station that is not id.
func otherStation(w *World, id StationID) Station {
	for _, s := range w.Stations {
		if s.ID != id {
			return s
		}
	}
	panic("only one station")
}

// openPoint is a point far from every station (no snapping).
func openPoint(t *testing.T, w *World) (float64, float64) {
	t.Helper()
	for x := 50.0; x < MapSize; x += 50 {
		for y := 50.0; y < MapSize; y += 50 {
			if _, near := w.nearestStation(x, y); !near {
				return x, y
			}
		}
	}
	t.Fatal("no open point on the map")
	return 0, 0
}

// dockedTrade adds a player and returns it with its docked ship and a good
// the ship's station produces (sells and buys back).
func dockedTrade(t *testing.T, w *World, name string) (*Player, *Ship, Listing) {
	t.Helper()
	p, s := newPlayer(t, w, name)
	st, _ := w.Station(s.At)
	for _, l := range st.Market {
		if l.Buy > 0 {
			return p, s, l
		}
	}
	t.Fatalf("station %s produces nothing", st.ID)
	return nil, nil, Listing{}
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// advanceUntil advances whole ticks until the ship is no longer travelling
// and returns the tick it stopped at.
func advanceUntil(t *testing.T, w *World, s *Ship) int64 {
	t.Helper()
	for range 1000 {
		w.Advance()
		if s.Status != Travelling {
			return w.Tick
		}
	}
	t.Fatal("ship never arrived")
	return 0
}

func TestNewWorldHasStations(t *testing.T) {
	w := NewWorld(1)
	if len(w.Stations) != 6 {
		t.Fatalf("got %d stations, want 6", len(w.Stations))
	}
	for _, s := range w.Stations {
		if s.X < 0 || s.X > MapSize || s.Y < 0 || s.Y > MapSize {
			t.Errorf("station %s outside the map: (%v,%v)", s.ID, s.X, s.Y)
		}
		if got, ok := w.Station(s.ID); !ok || got.ID != s.ID || got.Name != s.Name {
			t.Errorf("Station(%s) lookup failed", s.ID)
		}
	}
	if len(w.Goods) != 10 {
		t.Fatalf("got %d goods, want 10", len(w.Goods))
	}
	for _, g := range w.Goods {
		if got, ok := w.Good(g.ID); !ok || got != g {
			t.Errorf("Good(%s) lookup failed", g.ID)
		}
	}
	if _, ok := w.Good("gold"); ok {
		t.Error("found a good that doesn't exist")
	}
}

func TestSnapshotIsCopy(t *testing.T) {
	w := NewWorld(1)
	_, s := newPlayer(t, w, "Alice")
	newPlayer(t, w, "Bob")

	snap := w.Snapshot(0.25)
	if snap.Time != 0.25 {
		t.Errorf("Time = %v, want 0.25", snap.Time)
	}
	if len(snap.Players) != 2 || len(snap.Ships) != 2 || len(snap.Stations) != 6 {
		t.Fatalf("snapshot sizes: %d players, %d ships, %d stations", len(snap.Players), len(snap.Ships), len(snap.Stations))
	}
	if snap.Players[0].ID != "p1" || snap.Players[1].ID != "p2" {
		t.Errorf("players not sorted by ID: %v", snap.Players)
	}

	snap.Ships[0].Status = Travelling
	snap.Ships[0].Cargo["food"] = 5
	snap.Players[0].Name = "Mallory"
	snap.Stations[0].Name = "Renamed"
	snap.Stations[0].Market[0].Sell = 9999
	snap.Goods[0].Name = "Renamed"
	if s.Status != Docked || w.Players["p1"].Name != "Alice" || w.Stations[0].Name == "Renamed" {
		t.Error("changing the snapshot changed the world")
	}
	if s.CargoUsed() != 0 || w.Stations[0].Market[0].Sell == 9999 || w.Goods[0].Name == "Renamed" {
		t.Error("changing the snapshot's cargo, market or goods changed the world")
	}
}
