# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

**Space Trade** is a real-time multiplayer (LAN) space-trading game for the browser, played with friends in one shared sector (an original game in the trading-sim genre).

## Tech Stack

- Backend:
  - Go 1.24
  - No database. All game state lives in memory.
  - Dev Loop: Air rebuilds and restarts the server on changes.
- Fronted:
  - React 18.3 served with Vite 6 dev server
  - The star map is plain SVG.
  - no state-management library
  - styling in plain CSS
- Networking:
  - A single WebSocket (/api/vi/ws) carries JSON messages.
  - Clients send intents, and the server sends a full state snapshot every tick.
  - Clients store their session in localStorage.
- Infrastructure:
  - dev setup only: Docker Compose with three services:
    - nginx:
      - 1.27-alpine on published port 8080.
      - Routes /api/\* to the backend (port 8000) and everything else to Vite (port 5173)routes /api/
      - Passes WebSocket upgrades through for both the game socket and Vite's hot reload.
    - backend: the Go container running Air, with the source folder mounted and a volume caching Go modules
    - fronted: the Vite container, with src/ mounted for hot reload.

## Commands

All tooling runs in Docker; run Go commands (`go get`, `go mod tidy`, tests) inside the backend container so the host Go doesn't bump `go.mod`.

```bash
docker compose up -d --build     # start; rebuild after Dockerfile / go.mod / package.json / .air.toml changes
docker compose ps                # first check if the app is unreachable
docker compose logs -f backend   # Air rebuilds + Go compile errors
docker compose exec backend sh -c 'gofmt -l .; go vet ./... && go test ./...'   # lint + test
docker compose exec backend go test -count=3 ./...   # after touching engine (timing-based tests)
docker compose restart nginx     # after editing nginx.conf (single-file bind mount)
./scripts/lan-url.sh             # LAN address for friends
```

App: http://localhost:8080 (`?slot=1`, `?slot=2`, … for several players in one browser).

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
