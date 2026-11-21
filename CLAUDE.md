# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Running the stack

Everything runs through Docker Compose; there is no local toolchain setup and no test/lint configuration yet.

```bash
docker compose up -d --build      # start (first time, or after Dockerfile / go.mod / package.json / .air.toml changes)
docker compose ps                 # status (`docker compose status` does not exist)
docker compose logs -f backend    # watch Air rebuilds and Go compile errors
docker compose down               # stop
curl localhost:8080/api/v1/health # → {"status":"alive"}
```

The app is at http://localhost:8080. nginx publishes port 8080; backend (8000) and frontend (5173) are reachable only on the compose network.

## Architecture

- **nginx** (`nginx/nginx.conf`) is the single entry point: `/api/` → `backend:8000`, everything else → `frontend:5173`. The `/` location forwards WebSocket upgrade headers so Vite HMR works through the proxy. New backend routes must live under `/api/` to be reachable.
- **frontend** (`frontend/`): React 18 served by the Vite dev server (not a production build). Only `frontend/src` is bind-mounted, so edits there hot-reload; changes to `index.html`, `vite.config.js` or `package.json` need a rebuild. `vite.config.js` sets `allowedHosts: true` and `hmr.clientPort: 8080` because the browser talks to Vite via nginx. The UI calls the backend with relative paths (`/api/v1/...`), relying on nginx for same-origin routing.
- **backend** (`backend/`): Go 1.24, standard library only (`net/http` with method patterns like `GET /api/v1/health`). Port comes from `PORT`. No database: game state is kept in memory (see `PLAN.md`).
  - Dockerfile has three stages: `dev` (used by compose), `build` + `prod` (static binary on alpine; `docker build --target prod ./backend`).
  - Hot reload: the `dev` stage runs Air with `backend/` bind-mounted at `/app`. Saving a `.go` file rebuilds to `tmp/server` and restarts within ~1s (nginx returns 502 during the rebuild). Go modules are cached in the `gomodcache` volume.
  - Air v1.61.7 ignores the `entrypoint` key in `.air.toml`; use `bin`. Changing `.air.toml` requires `docker compose restart backend`.
