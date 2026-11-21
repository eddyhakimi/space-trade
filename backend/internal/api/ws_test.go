package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/eddyhakimi/space-trade/backend/internal/engine"
	"github.com/eddyhakimi/space-trade/backend/internal/game"
)

type client struct {
	t *testing.T
	c *websocket.Conn
}

// serverMsg is the union of everything the server sends.
type serverMsg struct {
	Type     string        `json:"type"`
	PlayerID game.PlayerID `json:"playerId"`
	Token    string        `json:"token"`
	Msg      string        `json:"msg"`
	stateMsg
}

// startEngine runs an engine for the duration of the test.
func startEngine(t *testing.T, tick time.Duration) *engine.Engine {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	eng := engine.New(game.NewWorld(1), tick)
	go eng.Run(ctx)
	return eng
}

// startWS serves only the WebSocket handler and returns its ws:// URL.
func startWS(t *testing.T) string {
	t.Helper()
	eng := startEngine(t, time.Hour) // state messages only on changes
	srv := httptest.NewServer(wsHandler{eng: eng})
	t.Cleanup(srv.Close)
	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

func dial(t *testing.T, url string) *client {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	c, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.CloseNow() })
	return &client{t, c}
}

func (cl *client) send(v any) {
	cl.t.Helper()
	b, _ := json.Marshal(v)
	if err := cl.c.Write(context.Background(), websocket.MessageText, b); err != nil {
		cl.t.Fatal(err)
	}
}

func (cl *client) read() serverMsg {
	cl.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, data, err := cl.c.Read(ctx)
	if err != nil {
		cl.t.Fatalf("read: %v", err)
	}
	var m serverMsg
	if err := json.Unmarshal(data, &m); err != nil {
		cl.t.Fatalf("bad message %s: %v", data, err)
	}
	m.stateMsg.Type = m.Type
	return m
}

// readUntil skips messages until one matches.
func (cl *client) readUntil(desc string, ok func(serverMsg) bool) serverMsg {
	cl.t.Helper()
	for range 20 {
		if m := cl.read(); ok(m) {
			return m
		}
	}
	cl.t.Fatalf("never got %s", desc)
	return serverMsg{}
}

func (cl *client) join(name, token string) serverMsg {
	cl.t.Helper()
	cl.send(map[string]string{"type": "join", "name": name, "token": token})
	w := cl.read()
	if w.Type != "welcome" || w.PlayerID == "" || w.Token == "" {
		cl.t.Fatalf("expected welcome, got %+v", w)
	}
	return w
}

func isState(m serverMsg) bool { return m.Type == "state" }

func TestWSJoinGetsWelcomeAndState(t *testing.T) {
	url := startWS(t)
	a := dial(t, url)
	w := a.join("Alice", "")
	s := a.readUntil("state", isState)

	if s.TickMs != time.Hour.Milliseconds() {
		t.Fatalf("tickMs = %d, want %d", s.TickMs, time.Hour.Milliseconds())
	}
	if len(s.Stations) != 6 || len(s.Players) != 1 || len(s.Ships) != 1 {
		t.Fatalf("state: %d stations, %d players, %d ships", len(s.Stations), len(s.Players), len(s.Ships))
	}
	if p := s.Players[0]; p.ID != w.PlayerID || p.Name != "Alice" || !p.Online {
		t.Fatalf("player: %+v", p)
	}
	if s.Ships[0].OwnerID != w.PlayerID || s.Ships[0].Status != game.Docked {
		t.Fatalf("ship: %+v", s.Ships[0])
	}
}

func TestWSReconnectWithToken(t *testing.T) {
	url := startWS(t)
	a := dial(t, url)
	first := a.join("Alice", "")
	a.c.Close(websocket.StatusNormalClosure, "")

	b := dial(t, url)
	again := b.join("", first.Token)
	if again.PlayerID != first.PlayerID || again.Token != first.Token {
		t.Fatalf("resume gave %+v, want player %s", again, first.PlayerID)
	}
	s := b.readUntil("state with Alice online", func(m serverMsg) bool {
		return isState(m) && m.Players[0].Online
	})
	if len(s.Players) != 1 || len(s.Ships) != 1 {
		t.Fatalf("reconnect created extra entities: %d players, %d ships", len(s.Players), len(s.Ships))
	}
}

func TestWSErrors(t *testing.T) {
	url := startWS(t)

	// Not starting with a join closes the connection after an error.
	x := dial(t, url)
	x.send(map[string]string{"type": "travel"})
	if m := x.read(); m.Type != "error" {
		t.Fatalf("expected error, got %+v", m)
	}

	// Duplicate name is rejected.
	dial(t, url).join("Alice", "")
	y := dial(t, url)
	y.send(map[string]string{"type": "join", "name": "alice"})
	if m := y.read(); m.Type != "error" || m.Msg != game.ErrNameTaken.Error() {
		t.Fatalf("expected name-taken error, got %+v", m)
	}

	// Bad commands get an error but the connection stays usable.
	b := dial(t, url)
	b.join("Bob", "")
	s := b.readUntil("state", isState)
	b.send(map[string]any{"type": "travel", "shipId": "s1", "x": 500, "y": 500}) // Alice's ship
	if m := b.readUntil("error", isError); m.Msg != game.ErrNotYourShip.Error() {
		t.Fatalf("got %+v", m)
	}
	b.send(map[string]any{"type": "travel", "shipId": "s2"}) // no coordinates
	b.readUntil("error", isError)
	b.send(map[string]any{"type": "travel", "shipId": "s2", "x": 5000, "y": 10})
	if m := b.readUntil("error", isError); m.Msg != game.ErrOutOfBounds.Error() {
		t.Fatalf("got %+v", m)
	}
	b.send(map[string]string{"type": "dance"})
	b.readUntil("error", isError)

	// The connection still works: fly to open space, then re-route mid-flight to a station.
	var mine game.Ship
	for _, sh := range s.Ships {
		if sh.OwnerID == "p2" {
			mine = sh
		}
	}
	shipState := func(m serverMsg) (game.Ship, bool) {
		if !isState(m) {
			return game.Ship{}, false
		}
		for _, sh := range m.Ships {
			if sh.ID == mine.ID {
				return sh, true
			}
		}
		return game.Ship{}, false
	}
	b.send(map[string]any{"type": "travel", "shipId": mine.ID, "x": 500, "y": 400})
	b.readUntil("Bob travelling to open space", func(m serverMsg) bool {
		sh, ok := shipState(m)
		return ok && sh.Status == game.Travelling && sh.To == "" && sh.ToX == 500 && sh.ToY == 400
	})
	dest := s.Stations[0]
	if dest.ID == mine.At { // the ship has barely left its station (no ticks pass in this test)
		dest = s.Stations[1]
	}
	b.send(map[string]any{"type": "travel", "shipId": mine.ID, "x": dest.X + 10, "y": dest.Y})
	b.readUntil("Bob re-routed to a station", func(m serverMsg) bool {
		sh, ok := shipState(m)
		return ok && sh.Status == game.Travelling && sh.To == dest.ID && sh.ToX == dest.X
	})
	b.send(map[string]any{"type": "travel", "shipId": mine.ID, "x": 500, "y": 400, "queue": true})
	b.readUntil("Bob queued a stop", func(m serverMsg) bool {
		sh, ok := shipState(m)
		return ok && sh.To == dest.ID && len(sh.Route) == 1 && sh.Route[0].X == 500 && sh.Route[0].Y == 400
	})
}

func isError(m serverMsg) bool { return m.Type == "error" }

func TestWSPlayersSeeEachOther(t *testing.T) {
	url := startWS(t)
	a := dial(t, url)
	a.join("Alice", "")
	b := dial(t, url)
	b.join("Bob", "")

	a.readUntil("Bob online", func(m serverMsg) bool {
		return isState(m) && len(m.Players) == 2 && m.Players[1].Online
	})
	b.c.Close(websocket.StatusNormalClosure, "")
	a.readUntil("Bob offline", func(m serverMsg) bool {
		return isState(m) && len(m.Players) == 2 && !m.Players[1].Online
	})
}

func TestWSTrade(t *testing.T) {
	url := startWS(t)
	a := dial(t, url)
	w := a.join("Alice", "")
	s := a.readUntil("state", isState)
	ship := s.Ships[0]
	var l game.Listing
	for _, st := range s.Stations {
		for _, sl := range st.Market {
			if st.ID == ship.At && sl.Buy > 0 {
				l = sl
			}
		}
	}
	if l.Buy == 0 {
		t.Fatalf("station %s sells nothing", ship.At)
	}

	a.send(map[string]any{"type": "buy", "shipId": ship.ID, "good": l.Good, "qty": 2})
	a.readUntil("cargo bought", func(m serverMsg) bool {
		return isState(m) && m.Ships[0].Cargo[l.Good] == 2 && m.Players[0].Credits == game.StartingCredits-l.BuyCost(2)
	})
	a.send(map[string]any{"type": "sell", "shipId": ship.ID, "good": l.Good, "qty": 2})
	a.readUntil("cargo sold", func(m serverMsg) bool {
		_, held := m.Ships[0].Cargo[l.Good]
		return isState(m) && !held && m.Players[0].ID == w.PlayerID
	})

	a.send(map[string]any{"type": "buy", "shipId": ship.ID, "good": l.Good, "qty": 0})
	if m := a.readUntil("error", isError); m.Msg != game.ErrBadQuantity.Error() {
		t.Fatalf("got %+v", m)
	}
	a.send(map[string]any{"type": "buy", "shipId": ship.ID, "good": "gold", "qty": 1})
	if m := a.readUntil("error", isError); m.Msg != game.ErrUnknownGood.Error() {
		t.Fatalf("got %+v", m)
	}
	// The connection still works after errors.
	a.send(map[string]any{"type": "sell", "shipId": ship.ID, "good": l.Good, "qty": 1})
	if m := a.readUntil("error", isError); m.Msg != game.ErrNotEnoughCargo.Error() {
		t.Fatalf("got %+v", m)
	}
}

func TestWSPlunderErrors(t *testing.T) {
	url := startWS(t)
	a := dial(t, url)
	a.join("Alice", "")
	s := a.readUntil("state", isState)
	ship := s.Ships[0]

	a.send(map[string]any{"type": "plunder", "shipId": ship.ID, "targetId": "t99"})
	if m := a.readUntil("error", isError); m.Msg != game.ErrUnknownShip.Error() {
		t.Fatalf("got %+v", m)
	}
	a.send(map[string]any{"type": "plunder", "shipId": ship.ID, "targetId": ship.ID})
	if m := a.readUntil("error", isError); m.Msg != game.ErrNotATrader.Error() {
		t.Fatalf("got %+v", m)
	}
}

func TestWSOthersCargoHidden(t *testing.T) {
	url := startWS(t)
	a := dial(t, url)
	alice := a.join("Alice", "")
	s := a.readUntil("state", isState)
	ship := s.Ships[0]
	var l game.Listing
	for _, st := range s.Stations {
		for _, sl := range st.Market {
			if st.ID == ship.At && sl.Buy > 0 {
				l = sl
			}
		}
	}
	a.send(map[string]any{"type": "buy", "shipId": ship.ID, "good": l.Good, "qty": 2})
	a.readUntil("own cargo visible", func(m serverMsg) bool {
		return isState(m) && m.Ships[0].Cargo[l.Good] == 2
	})

	b := dial(t, url)
	b.join("Bob", "")
	m := b.readUntil("state with both ships", func(m serverMsg) bool { return isState(m) && len(m.Ships) == 2 })
	for _, sh := range m.Ships {
		if sh.OwnerID == alice.PlayerID && sh.Cargo != nil {
			t.Fatalf("Bob sees Alice's cargo: %v", sh.Cargo)
		}
		if sh.OwnerID != alice.PlayerID && sh.Cargo == nil {
			t.Fatalf("Bob's own cargo missing: %+v", sh)
		}
	}
}

func TestWSStateHasHeat(t *testing.T) {
	url := startWS(t)
	a := dial(t, url)
	a.join("Alice", "")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for {
		_, data, err := a.c.Read(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), `"type":"state"`) {
			if !strings.Contains(string(data), `"heat":0`) {
				t.Fatalf("state without players[].heat: %s", data)
			}
			return
		}
	}
}
