package game

import (
	"errors"
	"testing"
)

func TestSpawnTraders(t *testing.T) {
	w := NewWorld(1)
	w.SpawnTraders(3)
	if len(w.Players) != 0 || len(w.Ships) != 3 {
		t.Fatalf("got %d players, %d ships; want 0 and 3", len(w.Players), len(w.Ships))
	}
	for _, id := range []ShipID{"t1", "t2", "t3"} {
		s, ok := w.Ships[id]
		if !ok || !s.NPC || s.OwnerID != "" || s.Status != Docked || s.Speed != TraderSpeed {
			t.Errorf("trader %s: %+v", id, s)
		}
	}
}

func TestTradersLoopBetweenStations(t *testing.T) {
	w := NewWorld(1)
	p, _ := newPlayer(t, w, "Alice")
	w.SpawnTraders(3)

	type trip struct {
		from StationID
		good GoodID
	}
	last := map[ShipID]ShipStatus{}
	at := map[ShipID]StationID{}
	current := map[ShipID]trip{}
	departures := map[ShipID]int{}
	loaded, empty := 0, 0
	for range 600 {
		w.Advance()
		for id, s := range w.Ships {
			if !s.NPC {
				continue
			}
			if s.SpeedAt(float64(w.Tick)) > TraderSpeed+1e-9 {
				t.Fatalf("%s faster than TraderSpeed: %v", id, s.SpeedAt(float64(w.Tick)))
			}
			switch {
			case last[id] == Docked && s.Status == Travelling: // departed
				from, _ := w.Station(at[id])
				dest, _ := w.Station(s.To)
				if dest.ID == "" || dest.ID == from.ID {
					t.Fatalf("%s departed %s for %q", id, from.ID, s.To)
				}
				departures[id]++
				if len(s.Cargo) == 0 { // an empty run
					empty++
					break
				}
				if len(s.Cargo) != 1 {
					t.Fatalf("%s departed with cargo %v, want one good or none", id, s.Cargo)
				}
				loaded++
				var g GoodID
				for g = range s.Cargo {
				}
				if l, ok := from.listing(g); !ok || l.BaseBuy == 0 {
					t.Errorf("%s loaded %s at %s, which doesn't produce it", id, g, from.ID)
				}
				if l, ok := dest.listing(g); !ok || l.BaseBuy != 0 {
					t.Errorf("%s carries %s to %s, which doesn't only need it", id, g, dest.ID)
				}
				current[id] = trip{from.ID, g}
			case last[id] == Travelling && s.Status == Docked: // arrived; unloads on the same tick
				if s.CargoUsed() != 0 {
					t.Errorf("%s kept cargo %v after arriving", id, s.Cargo)
				}
			}
			last[id], at[id] = s.Status, s.At
		}
	}
	for _, id := range []ShipID{"t1", "t2", "t3"} {
		if departures[id] < 4 {
			t.Errorf("%s made %d trips in 600 ticks, want at least 4", id, departures[id])
		}
	}
	if loaded == 0 || empty == 0 {
		t.Errorf("%d loaded and %d empty trips, want some of each", loaded, empty)
	}
	// Traders never touch the player market.
	for _, st := range w.Stations {
		for _, l := range st.Market {
			if l.Pressure != 0 {
				t.Errorf("%s %s: pressure %v after trader trips", st.ID, l.Good, l.Pressure)
			}
		}
	}
	if p.Credits != StartingCredits {
		t.Errorf("player credits changed to %d", p.Credits)
	}
}

func TestPlayersCantCommandTraders(t *testing.T) {
	w := NewWorld(1)
	p, _ := newPlayer(t, w, "Alice")
	w.SpawnTraders(1)
	tr := w.Ships["t1"]
	if err := w.Travel(p.ID, tr.ID, 500, 500, 0, false); !errors.Is(err, ErrNotYourShip) {
		t.Errorf("Travel with a trader: %v", err)
	}
	if err := w.Buy(p.ID, tr.ID, "food", 1, 0); !errors.Is(err, ErrNotYourShip) {
		t.Errorf("Buy with a trader: %v", err)
	}
}
