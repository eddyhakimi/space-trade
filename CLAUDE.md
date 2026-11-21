# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Running the stack

Everything runs through Docker Compose; there is no local toolchain setup and no lint configuration yet. Run Go commands (`go get`, `go test`, `go mod tidy`) inside the backend container so the `go 1.24` directive isn't bumped by a newer host Go.

```bash
docker compose up -d --build      # start (first time, or after Dockerfile / go.mod / package.json / .air.toml changes)
docker compose ps                 # status (`docker compose status` does not exist)
docker compose logs -f backend    # watch Air rebuilds and Go compile errors
docker compose down               # stop
curl localhost:8080/api/v1/health # → {"status":"alive"}
docker compose exec backend go test ./...   # backend tests (-race needs cgo; not available on the alpine image yet)
```

The app is at http://localhost:8080. nginx publishes port 8080; backend (8000) and frontend (5173) are reachable only on the compose network.

## Architecture

- **nginx** (`nginx/nginx.conf`) is the single entry point: `/api/` → `backend:8000`, everything else → `frontend:5173`. Both locations forward WebSocket upgrades (`map $http_upgrade $connection_upgrade`): `/` for Vite HMR, `/api/` for the game socket `/api/v1/ws` (with `proxy_read_timeout 1h`). `/api/` sends `Host $http_host` (keeps the port) so it matches the browser `Origin` for coder/websocket's same-origin check. Upstreams go through variables + `resolver 127.0.0.11` so recreated containers are re-resolved. New backend routes must live under `/api/` to be reachable.
  - `nginx.conf` is a single-file bind mount: editors that replace the file (new inode) leave the container on the old copy, so after editing run `docker compose restart nginx` (a `nginx -s reload` may not be enough).
- **frontend** (`frontend/`): React 18 served by the Vite dev server (not a production build). Only `frontend/src` is bind-mounted, so edits there hot-reload; changes to `index.html`, `vite.config.js` or `package.json` need a rebuild. `vite.config.js` sets `allowedHosts: true` and `hmr.clientPort: 8080` because the browser talks to Vite via nginx. The UI calls the backend with relative paths (`/api/v1/...`), relying on nginx for same-origin routing.
- **backend** (`backend/`): Go 1.24, `net/http` with method patterns like `GET /api/v1/health`; only dependency is `github.com/coder/websocket`. Routes are registered in `newMux()` (`main.go`) so tests (`main_test.go`, `httptest`) use the real routing. `GET /api/v1/ws` is currently a temporary echo endpoint (M0). Port comes from `PORT`. No database: game state is kept in memory (see `PLAN.md`).
  - Dockerfile has a single `dev` stage (used by compose). There is no production build for now (planned post-MVP in `PLAN.md`).
  - Hot reload: the `dev` stage runs Air with `backend/` bind-mounted at `/app`. Saving a `.go` file rebuilds to `tmp/server` and restarts within ~1s (nginx returns 502 during the rebuild). Go modules are cached in the `gomodcache` volume.
  - Air v1.61.7 ignores the `entrypoint` key in `.air.toml`; use `bin`. Changing `.air.toml` requires `docker compose restart backend`.
