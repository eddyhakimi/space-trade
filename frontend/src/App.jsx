import { useEffect, useState } from "react";
import "./styles.css";
import { clearSession, loadSession, useGameSocket } from "./net.js";
import JoinScreen from "./components/JoinScreen.jsx";
import StarMap from "./components/StarMap.jsx";
import ShipPanel from "./components/ShipPanel.jsx";
import PlayerList from "./components/PlayerList.jsx";
import MarketPanel from "./components/MarketPanel.jsx";
import PlunderPanel from "./components/PlunderPanel.jsx";

const TOAST_MS = 4000;

export default function App() {
  const [session, setSession] = useState(loadSession);
  const [joinError, setJoinError] = useState(null);

  if (!session) {
    return (
      <JoinScreen
        error={joinError}
        onJoin={(name) => {
          setJoinError(null);
          setSession({ name });
        }}
      />
    );
  }

  return (
    <Game
      session={session}
      onJoinFailed={(msg) => {
        clearSession();
        setJoinError(msg);
        setSession(null);
      }}
      onLeave={() => {
        clearSession();
        setSession(null);
      }}
    />
  );
}

function Game({ session, onJoinFailed, onLeave }) {
  const { status, game, me, error, travel, buy, sell, plunder } = useGameSocket(session, onJoinFailed);
  const myShip = game?.state.ships.find((s) => s.ownerId === me);
  const myPlayer = game?.state.players.find((p) => p.id === me);

  // Show command errors briefly.
  const [toast, setToast] = useState(null);
  useEffect(() => {
    if (!error) return;
    setToast(error.msg);
    const id = setTimeout(() => setToast(null), TOAST_MS);
    return () => clearTimeout(id);
  }, [error]);

  return (
    <div className="game">
      <header>
        <h1>Space Trade</h1>
        <span className={`conn ${status}`}>● {status}</span>
        {game && <span className="tick">Day {game.state.tick}</span>}
        <span className="spacer" />
        {myPlayer && <span className="credits">{myPlayer.credits.toLocaleString()} cr</span>}
        <span className="captain">Captain {myPlayer?.name ?? session.name}</span>
        <button onClick={onLeave} title="Forget this captain on this device">
          Leave
        </button>
      </header>

      {!game || !me ? (
        <p className="loading">Connecting to the sector…</p>
      ) : (
        <div className="layout">
          <StarMap game={game} me={me} onMapClick={(x, y, queue) => myShip && travel(myShip.id, x, y, queue)} />
          <aside>
            <PlunderPanel game={game} me={me} onPlunder={(target) => myShip && plunder(myShip.id, target)} />
            <ShipPanel game={game} me={me} />
            <MarketPanel
              game={game}
              me={me}
              onBuy={(good, qty) => myShip && buy(myShip.id, good, qty)}
              onSell={(good, qty) => myShip && sell(myShip.id, good, qty)}
            />
            <PlayerList players={game.state.players} me={me} />
          </aside>
        </div>
      )}

      {toast && (
        <div className="toast" role="alert">
          {toast}
        </div>
      )}
    </div>
  );
}
