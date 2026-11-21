package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/coder/websocket"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
}

// TEMP (M0): echoes every message back. Replaced by the game protocol in M3.
func wsEchoHandler(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, nil) // default: same-origin only
	if err != nil {
		log.Printf("ws accept: %v", err)
		return
	}
	defer c.CloseNow()

	ctx := r.Context()
	for {
		typ, data, err := c.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) == -1 {
				log.Printf("ws read: %v", err)
			}
			return
		}
		if err := c.Write(ctx, typ, data); err != nil {
			log.Printf("ws write: %v", err)
			return
		}
	}
}

func newMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", healthHandler)
	mux.HandleFunc("GET /api/v1/ws", wsEchoHandler)
	return mux
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	log.Printf("backend listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, newMux()))
}
