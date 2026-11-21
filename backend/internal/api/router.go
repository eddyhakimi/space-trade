// Package api is the HTTP side of the backend: routes, the health check and
// the WebSocket game protocol. It translates between JSON and engine calls;
// game rules live in internal/game and the loop in internal/engine.
//
// Files: router.go (routes), health.go, protocol.go (wire messages), ws.go
// (one WebSocket connection).
package api

import (
	"net/http"

	"github.com/eddyhakimi/space-trade/backend/internal/engine"
)

// NewRouter returns all backend routes. nginx forwards /api/ here.
func NewRouter(eng *engine.Engine) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", healthHandler)
	mux.Handle("GET /api/v1/ws", wsHandler{eng: eng})
	return mux
}
