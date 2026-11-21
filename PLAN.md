# Star Freight — Plan

A real-time multiplayer space-trading game for the browser, played with friends in one shared sector. Go backend, React frontend. Built iteratively: the MVP is the multiplayer "skeleton" (a shared sector where ships fly between stations in real time); trading and economy are layered on top afterwards.

## 1. Architecture options

### A. Authoritative in-memory game loop + WebSocket snapshots *(minimal / pragmatic)*
One goroutine owns the whole game state in memory. It ticks on a timer (1 tick ≈ 1 s), applies player commands received over a channel, and after each change broadcasts a full JSON snapshot to all connected clients over WebSockets. Persistence (e.g. re-add MongoDB or a JSON snapshot file) is added later.
- ➕ No locks or races in game logic (single owner); rules live in a pure, unit-testable Go package.
- ➕ Clients are "dumb": render the latest snapshot. Snapshots of a small sector are a few KB → no diffing.
- ➕ Least code to get something playable.
- ➖ A server restart wipes the game until persistence lands (fine for an MVP among friends).
- ➖ One process / a handful of rooms — fine for friends, not for thousands.

### B. Stateless REST + MongoDB as source of truth + polling
Every action is a REST call writing to Mongo; a background ticker updates documents; clients poll `GET /state` every 1–2 s.
- ➕ Survives restarts from day one; familiar CRUD; uses the Mongo already in compose.
- ➖ Ticker + concurrent player writes need transactions/optimistic locking.
- ➖ Polling feels laggy; ship movement looks jumpy.
- ➖ Game logic tangled with DB code → harder to test.

### C. Event-sourced simulation with delta sync and client-side prediction
Commands become events in a log; state is a fold over events; server sends deltas; client predicts/interpolates.
- ➕ Replays, durable by design, minimal bandwidth, smooth visuals.
- ➖ Far too much machinery for a friends-scale game; slows every iteration.

### Recommendation: **A**
It fits real-time ticks, keeps rules testable, and leaves a clean seam for persistence (serialize `World` to a store every N ticks — no rework). B's durability is cheap to add to A later; C's benefits don't matter at this scale.

## 2. MVP scope

**In:**
- One shared sector (single game, no lobby).
- Join with a name (no password; a random token in `localStorage` lets you reconnect as the same player).
- ~6 stations with fixed positions on a 2D map.
- Each player gets 1 ship, docked at a random station.
- Click anywhere on the map to fly there (re-routing mid-flight is allowed); a click near a station docks there. Shift+click queues more stops. Ships accelerate to a capped max speed and brake only at stations and the last stop.
- Everyone sees all players' ships move live (route lines only for your own ship); online/offline indicator per player.

**Out (later):** goods, prices, buying/selling, production, credits, leaderboard, persistence, multiple rooms, more ships, auth, events, chat, production deploy.

### Rules
- Ship state: `docked(at)`, `idle(x, y)` (holding position in open space) or `travelling(x, y → toX, toY [to station], v0, v1, departAt, arriveAt, route[])`.
- Any map point is a valid target; within `SnapRadius` (40) of a station the target snaps to it and the ship docks on arrival.
- Motion per leg (`game/motion.go`, mirrored in `geometry.js`): accelerate at `accel` (25 u/tick²) from `v0` up to `speed` (max, 50 u/tick), cruise, brake to `v1` at the target; short legs never reach max speed.
- Re-routing mid-flight starts the new leg from the ship's current position and keeps its speed (`v0`), dropping queued stops. Clicking on your flying ship brakes it to a stop straight ahead.
- `travel` with `queue` appends a stop (max 10) to a travelling ship's `route`; the ship stops at stations and at the last stop and flies through the others: `v1` is the fastest it can pass a waypoint and still brake for the next full stop. The client predicts these follow-up legs with the same math.
- Time is continuous: legs use float ticks (`departAt`, `arriveAt`) so commands between ticks don't make ships jump; arrivals resolve on the next tick boundary.
- Position while travelling is derived from time, so the client can interpolate smoothly between snapshots.

### Wire protocol (JSON over `/api/v1/ws`)
- Client → server: `join{name, token?}`, `travel{shipId, x, y, queue?}`
- Server → client: `welcome{playerId, token}`, `state{tick, time, tickMs, stations, ships, players}` (`time` = tick + fraction of the current tick) (full snapshot every tick and after each change), `error{msg}`
- `join` with an unknown token but a name takes over the player with that name if nobody is connected as them (after Leave, a lost welcome, or a stale tab), otherwise creates a new player (e.g. after a server restart). A name in use by a connected player is rejected. Errors before `welcome` close the connection; later errors keep it open.

### Code layout
```
backend/
  main.go                 config + wiring only
  internal/game/          pure rules, no I/O — world.go, station.go, player.go, ship.go, snapshot.go, errors.go
  internal/engine/        loop goroutine owning the World — engine.go, commands.go, session.go, broadcast.go
  internal/api/           HTTP + WebSocket — router.go, health.go, protocol.go (wire messages), ws.go
frontend/src/
  net.js                  useGameSocket hook (connect, join, reconnect, latest state) + session storage
  geometry.js             interpolation helpers (tickNow, progress, ETA), player colors
  App.jsx                 join screen vs. game screen
  components/             JoinScreen.jsx, StarMap.jsx (SVG), ShipPanel.jsx, PlayerList.jsx
```

## 3. Milestones
Each milestone ends in a working, verifiable state. Backend tests: `docker compose exec backend go test ./...` (`-race` once the dev image has cgo, see Next steps)

### M0 — Plumbing for WebSockets ✅ done
- Add `github.com/coder/websocket`; dev Dockerfile `COPY go.mod go.sum ./`. The prod build stages were removed for now (production build is post-MVP).
- nginx `/api/`: `proxy_http_version 1.1`, `Upgrade`/`Connection` headers, `proxy_read_timeout 1h`.
- Temporary echo endpoint `GET /api/v1/ws`; frontend `EchoPanel` sends a message every second and shows the echo.
- **Verify:** `docker compose up -d --build`; `docker compose exec backend go test ./...` passes; http://localhost:8080 shows the echo panel `open` with `seq` increasing every second. Health still `alive`.

### M1 — Game rules: world, players, travel (pure Go) ✅ done
- `internal/game`: `Station`, `Player`, `Ship`, `World`; `NewWorld()` with 6 stations; `AddPlayer(name)` (ship docked at a station); `Travel(shipId, to)`; `World.Tick()` advances ships and docks arrivals.
- **Verify (tests):** new player's ship is docked; ship arrives after exactly `ceil(dist/speed)` ticks; `Travel` fails while travelling, to the current station, to an unknown station, or for someone else's ship.

### M2 — Game loop & hub (server, no network yet) ✅ done
- `internal/server` (since renamed to `internal/engine`): loop goroutine owns `World`; `time.Ticker` (configurable, default 1 s); commands arrive with a reply channel; after each tick/command marshal one snapshot and fan out to subscribers via buffered channels (disconnect slow clients).
- **Verify (tests, `-race`):** with a 10 ms tick a subscriber receives snapshots with increasing `tick`; a submitted `travel` appears in the next snapshot; many concurrent commands produce no race reports.

### M3 — WebSocket protocol & player identity ✅ done
- Replace echo with the real protocol; `join` creates a player or resumes by token; invalid commands return `error`; online flag tracks connections.
- **Verify:** Go test with `httptest.Server` + websocket client: join → `welcome` + `state`; reconnect with token → same `playerId`; bad `travel` → `error`. Manual: two browser-console sockets see each other in `players`.

### M4 — Playable frontend ✅ done
- Join screen (name; token in `localStorage`); SVG star map with stations and all ships (own ship highlighted, travelling ships interpolated along their route); click a station to send your ship there; player list with online status; tick counter; error toast; auto-reconnect.
- **Verify (manual):** two browser windows (one private) join with different names; send a ship → it moves smoothly in both windows and docks on arrival; reload → still the same player and ship; stop the backend → "reconnecting" shown, recovers after restart (state reset is expected).

### M5 — Friends playtest on the LAN
- Friends on the same network open `http://<host LAN IP>:8080` and join; nothing to install. nginx is published on all interfaces, the WebSocket origin check and Vite accept any host, and the client builds its socket URL from `location.host`.
- `scripts/lan-url.sh` prints the address(es) to share and checks that the game answers on each.
- Playing over the internet (Tailscale or a Cloudflare tunnel) stays a later option.
- **Verify:** done so far: page, API, game socket and Vite HMR all answer through the LAN IP. Still to do: 2–4 people on other devices connect, fly around, and see no desyncs or dropped connections over ~15 min; collect feedback for the next iteration.

## 4. After the MVP (ordered by likely value)
1. **Trading:** goods (Ore, Fuel, Food, Electronics), per-station stock, dynamic prices, buy/sell while docked, credits.
2. **Economy:** station production/consumption each tick, prices drift; net-worth leaderboard; timed sessions with a winner.
3. Persistence: snapshot `World` every N ticks and on shutdown (e.g. re-add MongoDB or a JSON file); load on start.
4. Fleet: buy more/bigger ships, upgrades.
5. Game rooms / lobby with invite codes.
6. Events (pirates, shortages), own production, chat, auth, production build.
