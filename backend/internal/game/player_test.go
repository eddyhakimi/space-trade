package game

import (
	"errors"
	"strings"
	"testing"
)

func TestAddPlayerDocksShip(t *testing.T) {
	w := NewWorld(1)
	p, s := newPlayer(t, w, "  Alice ")
	if p.Name != "Alice" {
		t.Errorf("name not trimmed: %q", p.Name)
	}
	st, ok := w.Station(s.At)
	if s.Status != Docked || !ok {
		t.Fatalf("new ship not docked at a station: %+v", *s)
	}
	if s.X != st.X || s.Y != st.Y {
		t.Errorf("ship at (%v,%v), station at (%v,%v)", s.X, s.Y, st.X, st.Y)
	}
}

func TestAddPlayerNameValidation(t *testing.T) {
	w := NewWorld(1)
	newPlayer(t, w, "Alice")

	cases := map[string]error{
		"":                      ErrInvalidName,
		"   ":                   ErrInvalidName,
		strings.Repeat("x", 21): ErrInvalidName,
		"alice":                 ErrNameTaken,
		strings.Repeat("ü", 20): nil, // 20 runes is fine even though > 20 bytes
	}
	for name, want := range cases {
		_, err := w.AddPlayer(name)
		if !errors.Is(err, want) {
			t.Errorf("AddPlayer(%q) = %v, want %v", name, err, want)
		}
	}
}

func TestSetOnline(t *testing.T) {
	w := NewWorld(1)
	p, _ := newPlayer(t, w, "Alice")
	if err := w.SetOnline(p.ID, true); err != nil || !p.Online {
		t.Fatalf("SetOnline: err=%v online=%v", err, p.Online)
	}
	if err := w.SetOnline("ghost", true); !errors.Is(err, ErrUnknownPlayer) {
		t.Errorf("unknown player: got %v", err)
	}
}

func TestPlayerByName(t *testing.T) {
	w := NewWorld(1)
	p, _ := newPlayer(t, w, "Alice")
	for _, name := range []string{"Alice", "alice", "  ALICE "} {
		if got, ok := w.PlayerByName(name); !ok || got != p {
			t.Errorf("PlayerByName(%q) = %v, %v", name, got, ok)
		}
	}
	if _, ok := w.PlayerByName("Bob"); ok {
		t.Error("found a player that doesn't exist")
	}
}
