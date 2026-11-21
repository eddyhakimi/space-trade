package api

// The JSON wire protocol of /api/v1/ws. Every message has a "type".
//
// Client → server:
//
//	{"type":"join","name":"Alice","token":"..."}       first message; token optional (resume)
//	{"type":"travel","shipId":"s1","x":512.5,"y":88}   fly to any map point; near a station = dock
//	                                                   optional "queue":true appends a stop to the route
//	{"type":"buy","shipId":"s1","good":"ore","qty":5}  while docked: buy at this station's price
//	{"type":"sell","shipId":"s1","good":"ore","qty":5} while docked: sell to this station
//
// Server → client:
//
//	{"type":"welcome","playerId":"p1","token":"..."}   after a successful join
//	{"type":"state","tickMs":1000,"tick":..,"time":..,"goods":[..],"stations":[..],"players":[..],"ships":[..]}
//	                                                   stations[].market lists the goods traded there
//	                                                   ({good, buy?, sell, baseBuy?, baseSell, pressure};
//	                                                   no buy = not for sale; prices move unit by unit
//	                                                   with trading and recover every tick),
//	                                                   players[].credits, ships[].capacity, ships[].cargo {good: qty}
//	{"type":"error","msg":"..."}                       before welcome: join failed, connection closes

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/coder/websocket"
	"github.com/eddyhakimi/space-trade/backend/internal/game"
)

// clientMsg is any message a client sends; Type selects which fields matter.
type clientMsg struct {
	Type   string      `json:"type"`
	Name   string      `json:"name"`
	Token  string      `json:"token"`
	ShipID game.ShipID `json:"shipId"`
	X      *float64    `json:"x"`
	Y      *float64    `json:"y"`
	Queue  bool        `json:"queue"`
	Good   game.GoodID `json:"good"`
	Qty    int         `json:"qty"`
}

type welcomeMsg struct {
	Type     string        `json:"type"`
	PlayerID game.PlayerID `json:"playerId"`
	Token    string        `json:"token"`
}

type stateMsg struct {
	Type   string `json:"type"`
	TickMs int64  `json:"tickMs"`
	game.Snapshot
}

type errorMsg struct {
	Type string `json:"type"`
	Msg  string `json:"msg"`
}

func newWelcomeMsg(pid game.PlayerID, token string) welcomeMsg {
	return welcomeMsg{Type: "welcome", PlayerID: pid, Token: token}
}

func newStateMsg(snap game.Snapshot, tick time.Duration) stateMsg {
	return stateMsg{Type: "state", TickMs: tick.Milliseconds(), Snapshot: snap}
}

var errBadJSON = errors.New("message is not valid JSON")

func readMsg(ctx context.Context, c *websocket.Conn) (clientMsg, error) {
	var m clientMsg
	_, data, err := c.Read(ctx)
	if err != nil {
		return m, err
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return m, errBadJSON
	}
	return m, nil
}

func writeJSON(ctx context.Context, c *websocket.Conn, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.Write(ctx, websocket.MessageText, b)
}

func writeError(ctx context.Context, c *websocket.Conn, msg string) error {
	return writeJSON(ctx, c, errorMsg{Type: "error", Msg: msg})
}
