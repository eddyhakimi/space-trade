package game

import (
	"testing"
)

func TestTravelSnapsToStation(t *testing.T) {
	w := NewWorld(1)
	p, s := newPlayer(t, w, "Alice")
	dest := otherStation(w, s.At)

	// Just inside the snap radius: docks at the exact station position.
	if err := w.Travel(p.ID, s.ID, dest.X+SnapRadius-1, dest.Y, 0, false); err != nil {
		t.Fatal(err)
	}
	if s.To != dest.ID || s.ToX != dest.X || s.ToY != dest.Y {
		t.Fatalf("target not snapped: %+v", *s)
	}
	advanceUntil(t, w, s)
	if s.Status != Docked || s.At != dest.ID || s.X != dest.X || s.Y != dest.Y {
		t.Fatalf("after arrival: %+v, want docked at %s", *s, dest.ID)
	}

	// Just outside the snap radius: stays in open space.
	if err := w.Travel(p.ID, s.ID, dest.X+SnapRadius+1, dest.Y, float64(w.Tick), false); err != nil {
		t.Fatal(err)
	}
	advanceUntil(t, w, s)
	if s.Status != Idle || s.At != "" {
		t.Fatalf("outside snap radius: %+v, want idle", *s)
	}
}
