package game

import (
	"errors"
	"maps"
	"testing"
)

// flyingTrader returns a trader carrying cargo on a leg between two stations
// that started at time 0, and the player's ship holding position on that leg
// where the trader will be at time at.
func flyingTrader(t *testing.T, w *World, cargo map[GoodID]int, at float64) (*Player, *Ship, *Ship) {
	t.Helper()
	p, s := newPlayer(t, w, "Alice")
	w.SpawnTraders(1)
	tr := w.Ships["t1"]
	dest := otherStation(w, tr.At)
	tr.Cargo = maps.Clone(cargo)
	tr.idleUntil = 0
	tr.startLeg(dest.X, dest.Y, dest.ID, 0, 0)
	if at >= tr.ArriveAt {
		t.Fatalf("leg too short: arrives at %v", tr.ArriveAt)
	}
	x, y := tr.PositionAt(at)
	s.Status, s.At, s.X, s.Y = Idle, "", x, y
	return p, s, tr
}

func TestPlunderTakesWhatFits(t *testing.T) {
	w := NewWorld(1)
	p, s, tr := flyingTrader(t, w, map[GoodID]int{"ore": 15, "food": 10}, 5)
	s.Cargo["water"] = 3 // 17 units of space left

	if err := w.Plunder(p.ID, s.ID, tr.ID, 5); err != nil {
		t.Fatal(err)
	}
	// Catalog order: food before ore.
	if s.Cargo["food"] != 10 || s.Cargo["ore"] != 7 || s.CargoUsed() != s.Capacity {
		t.Fatalf("player cargo %v", s.Cargo)
	}
	if !maps.Equal(tr.Cargo, map[GoodID]int{"ore": 8}) || tr.Status != Travelling {
		t.Fatalf("trader after plunder: %v, %s", tr.Cargo, tr.Status)
	}
	if p.Credits != StartingCredits {
		t.Fatalf("plunder changed credits: %d", p.Credits)
	}
}

func TestPlunderErrors(t *testing.T) {
	cases := []struct {
		name  string
		setup func(w *World, p *Player, s, tr *Ship) (PlayerID, ShipID, ShipID)
		want  error
	}{
		{"not your ship", func(w *World, p *Player, s, tr *Ship) (PlayerID, ShipID, ShipID) {
			return "p99", s.ID, tr.ID
		}, ErrNotYourShip},
		{"unknown target", func(w *World, p *Player, s, tr *Ship) (PlayerID, ShipID, ShipID) {
			return p.ID, s.ID, "t99"
		}, ErrUnknownShip},
		{"target is a player", func(w *World, p *Player, s, tr *Ship) (PlayerID, ShipID, ShipID) {
			_, other := newPlayer(t, w, "Bob")
			return p.ID, s.ID, other.ID
		}, ErrNotATrader},
		{"own ship docked", func(w *World, p *Player, s, tr *Ship) (PlayerID, ShipID, ShipID) {
			s.Status, s.At = Docked, w.Stations[0].ID
			return p.ID, s.ID, tr.ID
		}, ErrPlunderDocked},
		{"trader docked", func(w *World, p *Player, s, tr *Ship) (PlayerID, ShipID, ShipID) {
			tr.arrive(tr.X, tr.Y, w.Stations[0].ID)
			return p.ID, s.ID, tr.ID
		}, ErrTargetProtected},
		{"too far", func(w *World, p *Player, s, tr *Ship) (PlayerID, ShipID, ShipID) {
			s.X += PlunderRadius + 1
			return p.ID, s.ID, tr.ID
		}, ErrTooFar},
		{"empty trader", func(w *World, p *Player, s, tr *Ship) (PlayerID, ShipID, ShipID) {
			clear(tr.Cargo)
			return p.ID, s.ID, tr.ID
		}, ErrNothingToPlunder},
		{"hold full", func(w *World, p *Player, s, tr *Ship) (PlayerID, ShipID, ShipID) {
			s.Cargo["water"] = s.Capacity
			return p.ID, s.ID, tr.ID
		}, ErrCargoFull},
	}
	for _, c := range cases {
		w := NewWorld(1)
		p, s, tr := flyingTrader(t, w, map[GoodID]int{"ore": 15}, 5)
		pid, sid, tid := c.setup(w, p, s, tr)
		mine, theirs := maps.Clone(s.Cargo), maps.Clone(tr.Cargo)
		if err := w.Plunder(pid, sid, tid, 5); !errors.Is(err, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, err, c.want)
		}
		if !maps.Equal(s.Cargo, mine) || !maps.Equal(tr.Cargo, theirs) {
			t.Errorf("%s: failed plunder moved cargo", c.name)
		}
	}
}

func TestPlunderJustInRange(t *testing.T) {
	w := NewWorld(1)
	p, s, tr := flyingTrader(t, w, map[GoodID]int{"ore": 15}, 5)
	s.X += PlunderRadius - 1
	if err := w.Plunder(p.ID, s.ID, tr.ID, 5); err != nil {
		t.Fatalf("just inside the radius: %v", err)
	}
}

func TestPlunderAtArrival(t *testing.T) {
	w := NewWorld(1)
	p, s, tr := flyingTrader(t, w, map[GoodID]int{"ore": 15}, 5)
	s.X, s.Y = tr.ToX, tr.ToY // waiting at the trader's destination
	if err := w.Plunder(p.ID, s.ID, tr.ID, tr.ArriveAt+0.5); !errors.Is(err, ErrTargetProtected) {
		t.Fatalf("plunder after the trader docked: %v", err)
	}
}
