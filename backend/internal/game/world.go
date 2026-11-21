// Package game holds the pure game rules: no I/O, no goroutines, no locks.
// A World must only be touched by one goroutine at a time (the engine loop).
//
// Files: world.go (the World and time), station.go, player.go, ship.go,
// goods.go (catalog), market.go (price table), trade.go (buy/sell),
// trader.go (AI traders), plunder.go,
// snapshot.go (read-only copies for clients), errors.go.
package game

import "math/rand/v2"

// MapSize is the width and height of the sector in map units.
const MapSize = 1000

type World struct {
	Tick     int64
	Stations []Station // fixed order, used for rendering
	Goods    []Good    // fixed catalog, in display order
	Players  map[PlayerID]*Player
	Ships    map[ShipID]*Ship

	stationIdx map[StationID]int
	goodIdx    map[GoodID]int
	rng        *rand.Rand
	nextPlayer int
	nextShip   int
	nextTrader int
}

// NewWorld creates the sector. The seed only affects random choices such as
// a new player's starting station, so tests are deterministic.
func NewWorld(seed uint64) *World {
	stations := defaultStations()
	idx := make(map[StationID]int, len(stations))
	for i, s := range stations {
		idx[s.ID] = i
	}
	goods := defaultGoods()
	goodIdx := make(map[GoodID]int, len(goods))
	for i, g := range goods {
		goodIdx[g.ID] = i
	}
	return &World{
		Stations:   stations,
		Goods:      goods,
		goodIdx:    goodIdx,
		Players:    map[PlayerID]*Player{},
		Ships:      map[ShipID]*Ship{},
		stationIdx: idx,
		rng:        rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15)),
	}
}

// Advance moves the world forward by one tick: ends legs that have arrived,
// runs the AI traders and lets market prices recover.
// (Named Advance because Tick is the tick counter.)
func (w *World) Advance() {
	w.Tick++
	for _, s := range w.Ships {
		s.advance(float64(w.Tick))
	}
	w.updateTraders()
	w.decayMarkets()
}
