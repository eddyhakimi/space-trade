package game

import (
	"fmt"
	"maps"
	"slices"
)

// AI traders: ships without an owner that carry goods from a station that
// produces them to one that needs them; about 1 in traderEmptyChance trips is
// flown empty, so plundering one is a gamble. Their loading and unloading
// neither costs credits nor moves market prices; they exist to fill the
// sector and to be plundered (see plunder.go).
const (
	TraderSpeed   = 30 // max speed in map units per tick (players: DefaultShipSpeed)
	TraderAccel   = 10
	traderMinLoad = 10
	traderMaxLoad = 25
	traderMinWait = 4 // ticks docked between trips
	traderMaxWait = 10
	// traderEmptyChance: 1 in this many trips carries nothing.
	traderEmptyChance = 4
)

var traderNames = []string{"Halley", "Corvid", "Marlin", "Juno", "Kestrel", "Osprey", "Talon", "Vireo"}

// SpawnTraders adds n AI traders docked at random stations, with staggered
// departures. NewWorld spawns none (so tests stay simple); main.go calls this.
func (w *World) SpawnTraders(n int) {
	for range n {
		w.nextTrader++
		st := w.Stations[w.rng.IntN(len(w.Stations))]
		name := traderNames[(w.nextTrader-1)%len(traderNames)]
		if w.nextTrader > len(traderNames) {
			name = fmt.Sprintf("%s %d", name, (w.nextTrader-1)/len(traderNames)+1)
		}
		s := &Ship{
			ID:        ShipID(fmt.Sprintf("t%d", w.nextTrader)),
			Name:      name + " (trader)",
			NPC:       true,
			Speed:     TraderSpeed,
			Accel:     TraderAccel,
			Capacity:  traderMaxLoad,
			Cargo:     map[GoodID]int{},
			Status:    Docked,
			X:         st.X,
			Y:         st.Y,
			At:        st.ID,
			idleUntil: float64(w.Tick) + 1 + float64(w.rng.IntN(traderMaxWait)),
		}
		w.Ships[s.ID] = s
	}
}

// updateTraders runs every docked trader's loop: unload and wait after
// arriving, then load a good this station produces and fly to a station that
// needs it. Called from Advance after ships have moved.
func (w *World) updateTraders() {
	now := float64(w.Tick)
	// Sorted, so the random choices are deterministic for a given seed.
	for _, id := range slices.Sorted(maps.Keys(w.Ships)) {
		s := w.Ships[id]
		if !s.NPC || s.Status != Docked {
			continue
		}
		if s.idleUntil == 0 { // just arrived: unload, then rest a little
			clear(s.Cargo)
			s.idleUntil = now + float64(traderMinWait+w.rng.IntN(traderMaxWait-traderMinWait+1))
			continue
		}
		if now < s.idleUntil {
			continue
		}
		good, dest, ok := w.traderRoute(s.At)
		if !ok {
			s.idleUntil = now + traderMaxWait // nothing to do here: try again later
			continue
		}
		if w.rng.IntN(traderEmptyChance) != 0 {
			s.Cargo[good] = traderMinLoad + w.rng.IntN(traderMaxLoad-traderMinLoad+1)
		}
		s.idleUntil = 0
		s.Route = nil
		s.startLeg(dest.X, dest.Y, dest.ID, 0, now)
	}
}

// traderRoute picks a random good the station at produces and a random other
// station that only needs it.
func (w *World) traderRoute(at StationID) (GoodID, Station, bool) {
	from, ok := w.Station(at)
	if !ok {
		return "", Station{}, false
	}
	var goods []GoodID
	for _, l := range from.Market {
		if l.BaseBuy > 0 {
			goods = append(goods, l.Good)
		}
	}
	w.rng.Shuffle(len(goods), func(i, j int) { goods[i], goods[j] = goods[j], goods[i] })
	for _, g := range goods {
		var dests []Station
		for _, st := range w.Stations {
			if l, ok := st.listing(g); ok && st.ID != at && l.BaseBuy == 0 {
				dests = append(dests, st)
			}
		}
		if len(dests) > 0 {
			return g, dests[w.rng.IntN(len(dests))], true
		}
	}
	return "", Station{}, false
}
