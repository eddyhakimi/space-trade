// Command backend runs the Space Trade game server. It only reads config and
// wires the pieces together:
//
//	internal/game    pure game rules (World, stations, players, ships)
//	internal/engine  the game loop goroutine that owns the World
//	internal/api     HTTP routes and the WebSocket protocol
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/eddyhakimi/space-trade/backend/internal/api"
	"github.com/eddyhakimi/space-trade/backend/internal/engine"
	"github.com/eddyhakimi/space-trade/backend/internal/game"
)

type config struct {
	port    string        // PORT, default 8000
	tick    time.Duration // TICK_MS (milliseconds per game tick), default 1000
	traders int           // TRADERS (AI trade ships), default 6; 0 disables
}

func loadConfig() config {
	cfg := config{port: "8000", tick: time.Second, traders: 6}
	if v := os.Getenv("PORT"); v != "" {
		cfg.port = v
	}
	if v := os.Getenv("TICK_MS"); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms > 0 {
			cfg.tick = time.Duration(ms) * time.Millisecond
		} else {
			log.Printf("ignoring invalid TICK_MS=%q", v)
		}
	}
	if v := os.Getenv("TRADERS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			cfg.traders = n
		} else {
			log.Printf("ignoring invalid TRADERS=%q", v)
		}
	}
	return cfg
}

func main() {
	cfg := loadConfig()

	world := game.NewWorld(uint64(time.Now().UnixNano()))
	world.SpawnTraders(cfg.traders)
	eng := engine.New(world, cfg.tick)
	go eng.Run(context.Background())

	log.Printf("backend listening on :%s (tick %s, %d traders)", cfg.port, cfg.tick, cfg.traders)
	log.Fatal(http.ListenAndServe(":"+cfg.port, api.NewRouter(eng)))
}
