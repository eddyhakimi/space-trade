# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

**Space Trade** is a real-time multiplayer space-trading game for the browser, played with friends in one shared sector (an original game in the trading-sim genre). `PLAN.md` is the roadmap: architecture decision, MVP scope, milestones with verification steps, and the wire protocol summary. M0–M4 are done (players join, see the sector live, and fly anywhere on the map); next is M5 (playtest with friends), then trading. Work proceeds in small steps, each verified before moving on.

## Running the stack

Everything runs through Docker Compose; there is no local toolchain. Run Go commands (`go get`, `go test`, `go mod tidy`) **inside the backend container** so the `go 1.24` directive in `go.mod` isn't bumped by a newer host Go.

```bash
docker compose up -d --build      # start (first time, or after Dockerfile / go.mod / package.json / .air.toml changes)
docker compose ps                 # status (`docker compose status` does not exist) — check this first if the app is unreachable
docker compose logs -f backend    # watch Air rebuilds and Go compile errors
docker compose down               # stop
curl localhost:8080/api/v1/health # → {"status":"alive"}

# backend checks (no lint config; gofmt + vet are the lint step)
docker compose exec backend sh -c 'gofmt -l .; go vet ./... && go test ./...'
docker compose exec backend go test -count=3 ./...   # engine tests are timing-based: run several times after touching them
```

The app is at http://localhost:8080. Use `?slot=1`, `?slot=2`, … to play several players in one browser (separate sessions). nginx publishes port 8080; backend (8000) and frontend (5173) are only reachable on the compose network.

## Architecture

`browser → nginx :8080 → /api/* → backend :8000 (Go)  |  /* → frontend :5173 (Vite dev server)`

One WebSocket (`/api/v1/ws`) carries everything: the client sends intents (`join`, `travel`), the server sends a full `state` snapshot after every tick (1 s) and every change. **The server is authoritative**: clients never send positions, only targets; the server validates, snaps and times everything.

### Backend (`backend/`)

Go 1.24 stdlib `net/http` (method patterns like `GET /api/v1/health`) plus one dependency, `github.com/coder/websocket`. No database: state is in memory and lost on restart. Packages depend one way: **`api` → `engine` → `game`**. `main.go` only reads config (`PORT`, `TICK_MS` default 1000) and wires them.

- **`internal/game`** — pure rules: no I/O, no goroutines, no locks.
  - `world.go`: `World`, `NewWorld(seed)`, `Advance()` (one tick; not called `Tick` because that's the counter field)
  - `station.go`: fixed station layout, `SnapRadius` (40): travel targets this close to a station dock there
  - `player.go`: `AddPlayer` (name rules, gives a ship), `PlayerByName`, `SetOnline`
  - `ship.go`: `Travel` (any point, re-route keeps speed, `queue` appends a route stop, click on own ship = brake), `PositionAt`/`SpeedAt`, `advance` (ends legs, starts the next queued one), `arrive`
  - `motion.go`: per-leg speed profile (accelerate → cruise at max → brake). **Mirrored in `frontend/src/geometry.js`; change both together.**
  - `snapshot.go`: value copy of the world for clients; `errors.go`: all sentinel errors — **their messages are shown to players**
  - Time is continuous: commands take `now` in float ticks; legs store `DepartAt`/`ArriveAt`; arrivals resolve in `Advance`.
- **`internal/engine`** — runs the game. **Only `Engine.Run` touches the `World`**; everything else submits closures via `exec` and waits for the reply.
  - `engine.go`: loop (ticker + command channel), `now()` = tick + elapsed fraction of the current tick
  - `commands.go`: player actions (`Travel`); `session.go`: `Join` (session tokens; a known name with no open connection is taken over, so Leave / lost welcomes / stale tabs never lock a name), `Subscribe`/`Unsubscribe` (connection count → `Online`)
  - `broadcast.go`: sends a `game.Snapshot` to every `Subscription` after each tick/change; a subscriber with a full buffer (8) is dropped
  - The engine knows nothing about JSON or HTTP.
- **`internal/api`** — HTTP and the wire format.
  - `router.go`: `NewRouter(eng)` (health + ws); `health.go`
  - `protocol.go`: **every wire message**, with the full protocol documented at the top of the file
  - `ws.go`: one connection — `handshake` (join → welcome; failure → error + close), `writeLoop` (state + 20 s pings), `readLoop` → `command` dispatch (bad commands → `error`, connection stays open)
- Dockerfile has a single `dev` stage running Air with `backend/` bind-mounted at `/app`; saving a `.go` file rebuilds to `tmp/server` and restarts in ~1 s (nginx returns 502 meanwhile). Go modules are cached in the `gomodcache` volume. No production build yet (post-MVP).

### Frontend (`frontend/`)

React 18 on the Vite dev server, plain SVG, no other libraries. Only `frontend/src` is bind-mounted (hot reload); `index.html`, `vite.config.js` and `package.json` changes need a rebuild. `vite.config.js` sets `allowedHosts: true` and `hmr.clientPort: 8080` because the browser reaches Vite through nginx. All backend calls use relative paths (`/api/v1/...`).

- `net.js`: `useGameSocket` (join, reconnect after 2 s, latest state, `travel(shipId, x, y)`); stores `{name, token}` in `localStorage` under `spacetrade.session[.<slot>]`; reconnects re-read the token (another tab may have saved a newer one) and retry rejected joins up to 5 times before returning to the join screen
- `geometry.js`: `tickNow` (from `state.time` + time since the snapshot arrived), leg profile (mirror of `game/motion.go`), `shipLegs` (current leg + predicted legs for queued stops), `shipState`, route ETA, player colors
- `App.jsx`: join screen vs. game screen; `styles.css`: dark theme, color tokens on `:root`
- `components/`: `StarMap.jsx` (click → map coordinates via `getScreenCTM` → `travel`, shift+click queues; ships interpolated every animation frame on a smoothed clock, headings turn gradually; only your own route is drawn), `ShipPanel.jsx`, `PlayerList.jsx`, `JoinScreen.jsx`

### nginx (`nginx/nginx.conf`)

Both locations forward WebSocket upgrades via `map $http_upgrade $connection_upgrade` (`/` for Vite HMR, `/api/` for the game socket, with `proxy_read_timeout 1h`). `/api/` sends `Host $http_host` (keeps the port) so coder/websocket's same-origin check matches the browser `Origin`. Upstreams are variables + `resolver 127.0.0.11`, so recreated containers are re-resolved. New backend routes must live under `/api/`.

## Playing on the LAN (M5)

Friends on the same network type `http://<host LAN IP>:8080` into their browser; `./scripts/lan-url.sh` prints that address and checks it answers. It works because nginx publishes 8080 on all interfaces, the socket URL comes from `location.host`, `/api/` forwards `Host $http_host` (so coder/websocket's same-origin check passes for any address), and Vite has `allowedHosts: true`. Never hardcode `localhost` in the frontend.

- Browsers that force HTTPS ("Always use secure connections" in Chrome) show a "connection is not secure" page for LAN addresses (`localhost` is exempt); players click "Continue to site". The nginx log then shows `\x16\x03\x01…` 400s, which are TLS handshakes hitting the plain-HTTP port.
- Docker-published ports normally bypass ufw; if friends still can't connect, check the host firewall (`sudo ufw status`).
- The host must keep the stack running; every backend restart (including Air rebuilds after saving a `.go` file) resets the world for everyone.

## Adding a player action (e.g. buy/sell)

1. `game`: a `World` method (takes `now` if timing matters) + sentinel errors in `errors.go` + tests.
2. `engine/commands.go`: a wrapper running it via `exec`, returning `broadcast = err == nil`.
3. `api/protocol.go`: document the message at the top, add its fields to `clientMsg`; add a `case` in `ws.go` `command`.
4. Frontend: a sender in `net.js`, UI in a component.
5. `PLAN.md`: add it to the wire protocol section.

## Testing conventions

- `game` tests: `NewWorld(1)` (deterministic), pass `now` explicitly; shared helpers in `world_test.go`.
- `engine` and `api` tests: a `time.Hour` tick, so states are broadcast only after commands (deterministic), or 10 ms to exercise ticking; shared helpers in `engine_test.go` / `ws_test.go`. Tests sit next to the file they cover.
- Frontend has no automated tests: verify in the browser with two `?slot` tabs (join, fly, watch the other tab).

## Gotchas & known issues

- `nginx.conf` is a single-file bind mount: editors that replace the file (new inode) leave the container on the old copy. After editing, run `docker compose restart nginx` (`nginx -s reload` may not be enough).
- Air v1.61.7 ignores `entrypoint` in `.air.toml`; use `bin`. Changing `.air.toml` needs `docker compose restart backend`.
- Image builds currently fail at `go mod download` on this host: build containers get an IPv6 DNS server they can't reach. Workarounds: `docker build --network host …`, or set `"dns"` in `/etc/docker/daemon.json` and restart Docker. Running containers are unaffected (modules come from the `gomodcache` volume).
- `go test -race` needs cgo, which the alpine dev image lacks (needs `build-base` + `CGO_ENABLED=1`, blocked by the build issue above).
- Every backend restart, including Air rebuilds, resets the world; clients reconnect and rejoin under their stored name as new players.
- Dev setup only: Vite dev server with `allowedHosts: true`, port 8080 published on all interfaces, no production build.
