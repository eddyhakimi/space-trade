## Architecture

`browser → nginx :8080 → /api/* → backend :8000 (Go)  |  /* → frontend :5173 (Vite dev server)`

One WebSocket (`/api/v1/ws`) carries everything: the client sends intents (`join`, `travel`, `buy`, `sell`, `plunder`), the server sends a full `state` snapshot after every tick (1 s) and every change. **The server is authoritative**: clients never send positions, only targets; the server validates, snaps and times everything.

### Backend (`backend/`)

Go 1.24 stdlib `net/http` (method patterns like `GET /api/v1/health`) plus one dependency, `github.com/coder/websocket`. No database: state is in memory and lost on restart. Packages depend one way: **`api` → `engine` → `game`**. `main.go` only reads config (`PORT`, `TICK_MS` default 1000, `TRADERS` default 6) and wires them.

- **`internal/game`** — pure rules: no I/O, no goroutines, no locks.
  - `world.go`: `World`, `NewWorld(seed)`, `Advance()` (one tick; not called `Tick` because that's the counter field)
  - `station.go`: fixed station layout, `SnapRadius` (40): travel targets this close to a station dock there
  - `player.go`: `AddPlayer` (name rules, gives a ship and `StartingCredits`), `PlayerByName`, `SetOnline`
  - `ship.go`: `Travel` (any point, re-route keeps speed, `queue` appends a route stop, click on own ship = brake), `PositionAt`/`SpeedAt`, `advance` (ends legs, starts the next queued one), `arrive`
  - `goods.go`: the 10-good catalog, `StartingCredits`, `DefaultCargoCapacity`; `market.go`: the normal price table (`Station.Market` of `Listing{Good, Buy, Sell, BaseBuy, BaseSell, Pressure}`; `produces`/`needs` helpers; edit prices here, `market_test.go` checks the invariants) and market pressure (`push` per unit traded, `BuyCost`/`SellValue` price orders unit by unit, `decayMarkets` runs in `Advance`). **Pricing is mirrored in `frontend/src/market.js`; change both together.** `trade.go`: `Buy`/`Sell` (docked only, resolve arrivals first, all-or-nothing). Credits live on `Player`, cargo on `Ship`.
  - `trader.go`: AI traders (`Ship.NPC`, no owner, ids `t1…`): `SpawnTraders` (called only from `main.go`, so tests have none unless they spawn them), `updateTraders` from `Advance` (load where produced, fly to a station that only needs it, unload; about 1 in 4 trips flies empty; no credits, no pressure). `plunder.go`: `Plunder` (`PlunderRadius` 40, mirrored as `PLUNDER_RADIUS` in `geometry.js`; a success adds heat). `wanted.go`: `Player.Heat` (remaining wanted ticks, cooled in `Advance`), `HeatPerPlunder`, `MaxHeat`, `RefuseTradeHeat` (checked in `tradeAt` → `ErrWanted`; mirrored as `REFUSE_TRADE_HEAT` in `market.js`).
  - `motion.go`: per-leg speed profile (accelerate → cruise at max → brake). **Mirrored in `frontend/src/geometry.js`; change both together.**
  - `snapshot.go`: value copy of the world for clients; `errors.go`: all sentinel errors — **their messages are shown to players**
  - Time is continuous: commands take `now` in float ticks; legs store `DepartAt`/`ArriveAt`; arrivals resolve in `Advance`.
- **`internal/engine`** — runs the game. **Only `Engine.Run` touches the `World`**; everything else submits closures via `exec` and waits for the reply.
  - `engine.go`: loop (ticker + command channel), `now()` = tick + elapsed fraction of the current tick
  - `commands.go`: player actions (`Travel`, `Buy`, `Sell`, `Plunder`); `session.go`: `Join` (session tokens; a known name with no open connection is taken over, so Leave / lost welcomes / stale tabs never lock a name), `Subscribe`/`Unsubscribe` (connection count → `Online`)
  - `broadcast.go`: sends a `game.Snapshot` to every `Subscription` after each tick/change; a subscriber with a full buffer (8) is dropped
  - The engine knows nothing about JSON or HTTP.
- **`internal/api`** — HTTP and the wire format.
  - `router.go`: `NewRouter(eng)` (health + ws); `health.go`
  - `protocol.go`: **every wire message**, with the full protocol documented at the top of the file
  - `ws.go`: one connection — `handshake` (join → welcome; failure → error + close), `writeLoop` (state per viewer via `newStateMsg`, which hides other ships' cargo; + 20 s pings), `readLoop` → `command` dispatch (bad commands → `error`, connection stays open)
- Dockerfile has a single `dev` stage running Air with `backend/` bind-mounted at `/app`; saving a `.go` file rebuilds to `tmp/server` and restarts in ~1 s (nginx returns 502 meanwhile). Go modules are cached in the `gomodcache` volume. No production build yet (post-MVP).

### Frontend (`frontend/`)

React 18 on the Vite dev server, plain SVG, no other libraries. Only `frontend/src` is bind-mounted (hot reload); `index.html`, `vite.config.js` and `package.json` changes need a rebuild. `vite.config.js` sets `allowedHosts: true` and `hmr.clientPort: 8080` because the browser reaches Vite through nginx. All backend calls use relative paths (`/api/v1/...`).

- `net.js`: `useGameSocket` (join, reconnect after 2 s, latest state, senders `travel`, `buy`, `sell`, `plunder` via one `send`); stores `{name, token}` in `localStorage` under `spacetrade.session[.<slot>]`; reconnects re-read the token (another tab may have saved a newer one) and retry rejected joins up to 5 times before returning to the join screen
- `geometry.js`: `tickNow` (from `state.time` + time since the snapshot arrived), leg profile (mirror of `game/motion.go`), `shipLegs` (current leg + predicted legs for queued stops), `shipState`, route ETA, player colors, `PLUNDER_RADIUS` (= `game.PlunderRadius`), `tradersInRange`
- `market.js`: mirror of the pricing in `game/market.go` (`buyCost`, `sellValue`, `maxBuy`, `trend`), plus `REFUSE_TRADE_HEAT` and `formatHeat`
- `App.jsx`: join screen vs. game screen; `styles.css`: dark theme, color tokens on `:root`
- `components/`: `StarMap.jsx` (click → map coordinates via `getScreenCTM` → `travel`, shift+click queues; ships interpolated every animation frame on a smoothed clock, headings turn gradually; only your own route is drawn), `ShipPanel.jsx` (position, hold), `MarketPanel.jsx` (docked station's market; Max/All limits only disable buttons, the server decides), `PlunderPanel.jsx` (travelling traders within range while you're in open space), `PlayerList.jsx` (leaderboard by credits), `JoinScreen.jsx`; `hooks.js`: `useClock`. Traders are drawn grey (`--npc`); a ring shows your plunder range in open space; wanted players' ships get a red outline, ☠ on the leaderboard and a header badge, and the market shows a refusal notice when too hot

### nginx (`nginx/nginx.conf`)

Both locations forward WebSocket upgrades via `map $http_upgrade $connection_upgrade` (`/` for Vite HMR, `/api/` for the game socket, with `proxy_read_timeout 1h`). `/api/` sends `Host $http_host` (keeps the port) so coder/websocket's same-origin check matches the browser `Origin`. Upstreams are variables + `resolver 127.0.0.11`, so recreated containers are re-resolved. New backend routes must live under `/api/`.
