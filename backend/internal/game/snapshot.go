package game

import (
	"cmp"
	"slices"
)

// Snapshot is a value copy of the world, safe to hand to other goroutines
// and to marshal as JSON.
type Snapshot struct {
	Tick int64 `json:"tick"`
	// Time is the moment of the snapshot in ticks, including the fraction of
	// the current tick; clients interpolate from it.
	Time     float64   `json:"time"`
	Stations []Station `json:"stations"`
	Players  []Player  `json:"players"`
	Ships    []Ship    `json:"ships"`
}

func (w *World) Snapshot(now float64) Snapshot {
	snap := Snapshot{
		Tick:     w.Tick,
		Time:     now,
		Stations: slices.Clone(w.Stations),
		Players:  make([]Player, 0, len(w.Players)),
		Ships:    make([]Ship, 0, len(w.Ships)),
	}
	for _, p := range w.Players {
		snap.Players = append(snap.Players, *p)
	}
	for _, s := range w.Ships {
		snap.Ships = append(snap.Ships, *s)
	}
	// Stable order so clients don't see lists reshuffle every tick.
	slices.SortFunc(snap.Players, func(a, b Player) int { return cmp.Compare(a.ID, b.ID) })
	slices.SortFunc(snap.Ships, func(a, b Ship) int { return cmp.Compare(a.ID, b.ID) })
	return snap
}
