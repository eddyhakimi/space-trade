package game

import (
	"errors"
	"maps"
	"testing"
)

func TestPlunderAddsHeat(t *testing.T) {
	w := NewWorld(1)
	p, s, tr := flyingTrader(t, w, map[GoodID]int{"ore": 15}, 5)
	if p.Wanted() {
		t.Fatal("new player is wanted")
	}

	// A failed plunder adds no heat.
	s.X += PlunderRadius + 1
	if err := w.Plunder(p.ID, s.ID, tr.ID, 5); !errors.Is(err, ErrTooFar) {
		t.Fatalf("far plunder: %v", err)
	}
	if p.Heat != 0 {
		t.Fatalf("failed plunder added heat: %d", p.Heat)
	}

	s.X -= PlunderRadius + 1
	if err := w.Plunder(p.ID, s.ID, tr.ID, 5); err != nil {
		t.Fatal(err)
	}
	if p.Heat != HeatPerPlunder || !p.Wanted() {
		t.Fatalf("heat after plunder: %d", p.Heat)
	}
}

func TestHeatCapsAndCools(t *testing.T) {
	w := NewWorld(1)
	p, _ := newPlayer(t, w, "Alice")
	for range 10 {
		p.addHeat()
	}
	if p.Heat != MaxHeat {
		t.Fatalf("heat %d, want capped at %d", p.Heat, MaxHeat)
	}
	p.Heat = 3
	for range 5 {
		w.Advance()
	}
	if p.Heat != 0 || p.Wanted() {
		t.Fatalf("heat %d after cooling, want 0", p.Heat)
	}
}

func TestTooHotToTrade(t *testing.T) {
	w := NewWorld(1)
	p, s, l := dockedTrade(t, w, "Alice")

	p.Heat = RefuseTradeHeat // wanted, but stations still trade
	if err := w.Buy(p.ID, s.ID, l.Good, 1, 0); err != nil {
		t.Fatalf("buy at heat %d: %v", p.Heat, err)
	}

	p.Heat = RefuseTradeHeat + 1
	credits, cargo := p.Credits, maps.Clone(s.Cargo)
	if err := w.Buy(p.ID, s.ID, l.Good, 1, 0); !errors.Is(err, ErrWanted) {
		t.Fatalf("buy at heat %d: %v", p.Heat, err)
	}
	if err := w.Sell(p.ID, s.ID, l.Good, 1, 0); !errors.Is(err, ErrWanted) {
		t.Fatalf("sell at heat %d: %v", p.Heat, err)
	}
	if p.Credits != credits || !maps.Equal(s.Cargo, cargo) {
		t.Fatal("refused trade changed credits or cargo")
	}

	w.Advance() // cools to RefuseTradeHeat
	if err := w.Sell(p.ID, s.ID, l.Good, 1, float64(w.Tick)); err != nil {
		t.Fatalf("sell after cooling to %d: %v", p.Heat, err)
	}
}
