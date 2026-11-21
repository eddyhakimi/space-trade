import { useState } from "react";

export default function JoinScreen({ onJoin, error }) {
  const [name, setName] = useState("");
  const trimmed = name.trim();

  return (
    <main className="join">
      <h1>Space Trade</h1>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          if (trimmed) onJoin(trimmed);
        }}
      >
        <label htmlFor="name">Captain name</label>
        <input
          id="name"
          autoFocus
          maxLength={20}
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="e.g. Nova"
        />
        <button type="submit" disabled={!trimmed}>
          Join sector
        </button>
        {error && <p className="error">{error}</p>}
      </form>
    </main>
  );
}
