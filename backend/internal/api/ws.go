package api

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/eddyhakimi/space-trade/backend/internal/engine"
	"github.com/eddyhakimi/space-trade/backend/internal/game"
)

const (
	joinTimeout  = 10 * time.Second
	pingInterval = 20 * time.Second
	pingTimeout  = 10 * time.Second
)

// wsHandler serves the game protocol (see protocol.go) on one WebSocket
// connection per request: handshake (join), then a writer streaming state
// and a reader executing commands until either side fails.
type wsHandler struct {
	eng *engine.Engine
}

func (h wsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, nil) // same-origin only
	if err != nil {
		log.Printf("ws accept: %v", err)
		return
	}
	defer c.CloseNow()

	// r.Context() is not cancelled for hijacked connections; we cancel ourselves.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pid, ok := h.handshake(ctx, c)
	if !ok {
		return
	}
	sub, err := h.eng.Subscribe(ctx, pid)
	if err != nil {
		return
	}
	defer h.eng.Unsubscribe(sub)

	go func() {
		defer cancel() // writer failed: also end the reader
		h.writeLoop(ctx, c, pid, sub)
	}()
	h.readLoop(ctx, c, pid)
}

// handshake expects a join as the first message and answers with welcome.
// On failure the client gets an error and the connection is closed.
func (h wsHandler) handshake(ctx context.Context, c *websocket.Conn) (game.PlayerID, bool) {
	jctx, jcancel := context.WithTimeout(ctx, joinTimeout)
	m, err := readMsg(jctx, c)
	jcancel()
	if err != nil || m.Type != "join" {
		writeError(ctx, c, "first message must be a join")
		c.Close(websocket.StatusPolicyViolation, "expected join")
		return "", false
	}
	pid, token, err := h.eng.Join(ctx, m.Name, m.Token)
	if err != nil {
		writeError(ctx, c, err.Error())
		c.Close(websocket.StatusPolicyViolation, "join failed")
		return "", false
	}
	if err := writeJSON(ctx, c, newWelcomeMsg(pid, token)); err != nil {
		return "", false
	}
	return pid, true
}

// writeLoop sends every snapshot as a state message and pings the client
// regularly so dead connections are noticed. Returns on any failure.
func (h wsHandler) writeLoop(ctx context.Context, c *websocket.Conn, pid game.PlayerID, sub *engine.Subscription) {
	ping := time.NewTicker(pingInterval)
	defer ping.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case snap, ok := <-sub.C:
			if !ok {
				c.Close(websocket.StatusTryAgainLater, "too slow or server stopping")
				return
			}
			if err := writeJSON(ctx, c, newStateMsg(snap, h.eng.TickInterval())); err != nil {
				return
			}
		case <-ping.C:
			pctx, pcancel := context.WithTimeout(ctx, pingTimeout)
			err := c.Ping(pctx)
			pcancel()
			if err != nil {
				log.Printf("ws ping %s: %v", pid, err)
				return
			}
		}
	}
}

// readLoop executes commands until the connection ends. Invalid commands
// get an error message back; the connection stays open.
func (h wsHandler) readLoop(ctx context.Context, c *websocket.Conn, pid game.PlayerID) {
	for {
		m, err := readMsg(ctx, c)
		if errors.Is(err, errBadJSON) {
			writeError(ctx, c, err.Error())
			continue
		}
		if err != nil {
			if websocket.CloseStatus(err) == -1 && ctx.Err() == nil {
				log.Printf("ws read %s: %v", pid, err)
			}
			return
		}
		if err := h.command(ctx, pid, m); err != nil {
			writeError(ctx, c, err.Error())
		}
	}
}

// command dispatches one client message to the engine.
func (h wsHandler) command(ctx context.Context, pid game.PlayerID, m clientMsg) error {
	switch m.Type {
	case "travel":
		if m.X == nil || m.Y == nil {
			return errors.New("travel needs x and y")
		}
		return h.eng.Travel(ctx, pid, m.ShipID, *m.X, *m.Y, m.Queue)
	default:
		return errors.New("unknown message type " + m.Type)
	}
}
