package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// newTestServer serves the real routes backed by a running engine.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(NewRouter(startEngine(t, time.Hour)))
	t.Cleanup(srv.Close)
	return srv
}

func TestHealth(t *testing.T) {
	srv := newTestServer(t)

	res, err := http.Get(srv.URL + "/api/v1/health")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	var body map[string]string
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK || body["status"] != "alive" {
		t.Fatalf("got %d %v, want 200 status=alive", res.StatusCode, body)
	}
}

// TestWSRoute checks that /api/v1/ws is wired to the game protocol.
// The protocol itself is tested in ws_test.go.
func TestWSRoute(t *testing.T) {
	srv := newTestServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/v1/ws"
	c, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()

	if err := c.Write(ctx, websocket.MessageText, []byte(`{"type":"join","name":"Alice"}`)); err != nil {
		t.Fatal(err)
	}
	_, data, err := c.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var m struct{ Type, PlayerID string }
	if err := json.Unmarshal(data, &m); err != nil || m.Type != "welcome" || m.PlayerID == "" {
		t.Fatalf("got %s, want welcome", data)
	}
	c.Close(websocket.StatusNormalClosure, "")
}
