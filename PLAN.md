# Star Freight — Plan

A real-time multiplayer space-trading game for the browser, played with friends in one shared sector. Go backend, React frontend. Built iteratively: the MVP is the multiplayer "skeleton" (a shared sector where ships fly between stations in real time); trading and economy are layered on top afterwards.

## 1. Architecture
Authoritative in-memory game loop + WebSocket snapshots. One goroutine (`engine.Run`) owns the whole `World`, advances it every tick (1 tick ≈ 1 s) and runs player commands received over a channel; after every tick and every change it broadcasts a full JSON snapshot to all clients over one WebSocket. Rules live in a pure, unit-testable Go package (`internal/game`), so they need no locks. Clients only send intents and render the latest snapshot, interpolating motion between snapshots. State is in memory and lost on restart; persistence (serialize `World` every N ticks) can be added later without rework.

## 2. MVP scope

**In:**
- One shared sector (single game, no lobby).
- Join with a name (no password; a random token in `localStorage` lets you reconnect as the same player).
- ~6 stations with fixed positions on a 2D map.
- Each player gets 1 ship, docked at a random station.
- Click anywhere on the map to fly there (re-routing mid-flight is allowed); a click near a station docks there. Shift+click queues more stops. Ships accelerate to a capped max speed and brake only at stations and the last stop.
- Everyone sees all players' ships move live (route lines only for your own ship); online/offline indicator per player.
- Trading: 10 goods, buy and sell for credits while docked; prices differ per station (M6).

**Out (later):** production, leaderboard, persistence, multiple rooms, more ships, auth, events, chat, production deploy.

### Rules
- Ship state: `docked(at)`, `idle(x, y)` (holding position in open space) or `travelling(x, y → toX, toY [to station], v0, v1, departAt, arriveAt, route[])`.
- Any map point is a valid target; within `SnapRadius` (40) of a station the target snaps to it and the ship docks on arrival.
- Motion per leg (`game/motion.go`, mirrored in `geometry.js`): accelerate at `accel` (25 u/tick²) from `v0` up to `speed` (max, 50 u/tick), cruise, brake to `v1` at the target; short legs never reach max speed.
- Re-routing mid-flight starts the new leg from the ship's current position and keeps its speed (`v0`), dropping queued stops. Clicking on your flying ship brakes it to a stop straight ahead.
- `travel` with `queue` appends a stop (max 10) to a travelling ship's `route`; the ship stops at stations and at the last stop and flies through the others: `v1` is the fastest it can pass a waypoint and still brake for the next full stop. The client predicts these follow-up legs with the same math.
- Time is continuous: legs use float ticks (`departAt`, `arriveAt`) so commands between ticks don't make ships jump; arrivals resolve on the next tick boundary.
- Position while travelling is derived from time, so the client can interpolate smoothly between snapshots.
- Trading happens at stations, while docked. Prices are fixed and stock is unlimited. A station trades only the goods it lists: a good it *produces* can be bought cheaply and sold back at 80 % of that price; a good it *needs* can only be sold to it, at a high price.
- Credits are whole numbers and belong to the player (1000 at start); cargo belongs to the ship (capacity 20 units, every good takes 1 unit).
- `buy`/`sell` resolve arrivals up to `now` first (a ship that just arrived can trade before the next tick), then need your docked ship, a good listed at that station and `qty ≥ 1`; buy also needs free space and credits, sell needs the units in the hold. All-or-nothing.
- No spread exploits: at any station `sell < buy`, and at normal prices buying at one producer and selling at another loses money.
- **Market pressure** (M7): every listing has a pressure, the net units taken from that station (+1 per unit bought, −1 per unit sold, clamped to ±100). Price = `round(base × e^(0.01 × pressure))`, minimum 1 (≈1 % per unit); producers' buyback stays at 80 % of the current price. Trades are priced unit by unit, so one big order costs the same as many small ones. Pressure decays every tick with a half-life of 60 ticks and snaps to 0 below 0.05, so a milked route recovers in a few minutes. Shared by all players.

### Trading
Normal prices (at pressure 0). Catalog (order = display order): Food, Water, Ore, Metals, Fuel, Textiles, Medicine, Electronics, Machinery, Luxuries (ids lowercase: `food`, `water`, …).

`P n` = station sells at n, buys back at n·8/10. `C n` = station only buys, at n. `–` = not traded.

| good | kepler | ostrava | halden | vesta | iolo | marrow |
|---|---|---|---|---|---|---|
| food | C 26 | C 30 | C 28 | P 14 | – | P 16 |
| water | – | C 16 | P 8 | C 14 | C 18 | – |
| ore | C 38 | P 18 | P 22 | – | C 42 | – |
| metals | P 45 | P 40 | – | C 72 | C 80 | – |
| fuel | C 50 | – | P 26 | C 48 | – | C 55 |
| textiles | – | C 44 | C 46 | P 24 | – | C 42 |
| medicine | – | C 150 | C 160 | – | P 85 | C 140 |
| electronics | C 185 | – | – | C 175 | P 105 | C 190 |
| machinery | P 78 | C 135 | C 140 | – | C 130 | – |
| luxuries | C 300 | – | C 320 | – | C 290 | P 170 |

Every station sells 2–3 goods and buys 6–8 (including buybacks); every good is sold at ≥ 1 station and bought at ≥ 2 others. The table lives in `game/market.go`; `market_test.go` guards these invariants.

### Wire protocol (JSON over `/api/v1/ws`)
- Client → server: `join{name, token?}`, `travel{shipId, x, y, queue?}`, `buy{shipId, good, qty}`, `sell{shipId, good, qty}`
- Server → client: `welcome{playerId, token}`, `state{tick, time, tickMs, goods, stations, ships, players}` (`time` = tick + fraction of the current tick; `stations[].market` = `[{good, buy?, sell, baseBuy?, baseSell, pressure}]`, no `buy` = not for sale, `buy`/`sell` are current prices; `players[].credits`; `ships[].capacity`, `ships[].cargo` = `{good: qty}`) (full snapshot every tick and after each change), `error{msg}`
- `join` with an unknown token but a name takes over the player with that name if nobody is connected as them (after Leave, a lost welcome, or a stale tab), otherwise creates a new player (e.g. after a server restart). A name in use by a connected player is rejected. Errors before `welcome` close the connection; later errors keep it open.

### Code layout
```
backend/
  main.go                 config + wiring only
  internal/game/          pure rules, no I/O — world.go, station.go, player.go, ship.go, motion.go,
                          goods.go, market.go (price table), trade.go (buy/sell), snapshot.go, errors.go
  internal/engine/        loop goroutine owning the World — engine.go, commands.go, session.go, broadcast.go
  internal/api/           HTTP + WebSocket — router.go, health.go, protocol.go (wire messages), ws.go
frontend/src/
  net.js                  useGameSocket hook (connect, join, reconnect, latest state) + session storage
  geometry.js             interpolation helpers (tickNow, progress, ETA), player colors
  market.js               pricing mirror of game/market.go (unit-by-unit costs, Max, ▲/▼ trend)
  App.jsx                 join screen vs. game screen
  components/             JoinScreen.jsx, StarMap.jsx (SVG), ShipPanel.jsx, MarketPanel.jsx, PlayerList.jsx
```

## 3. Milestones
Each milestone ends in a working, verifiable state. Backend tests: `docker compose exec backend go test ./...` (`-race` once the dev image has cgo). M0–M4 (WebSocket plumbing, game rules, game loop, protocol, playable frontend) and M6 (trading), M7 (market pressure) are done.

## 4. After the MVP (ordered by likely value)
1. **Economy:** per-station stock limits, production/consumption, events that shift prices; net-worth leaderboard; timed sessions with a winner.
2. Persistence: snapshot `World` every N ticks and on shutdown (e.g. re-add MongoDB or a JSON file); load on start.
3. Fleet: buy more/bigger ships, upgrades.
4. Game rooms / lobby with invite codes.
5. Events (pirates, shortages), own production, chat, auth, production build.
