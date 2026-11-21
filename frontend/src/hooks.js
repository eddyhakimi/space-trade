import { useEffect, useState } from "react";

// Re-render every ms milliseconds (e.g. so ETAs and ranges update between snapshots).
export function useClock(ms) {
  const [now, setNow] = useState(() => performance.now());
  useEffect(() => {
    const id = setInterval(() => setNow(performance.now()), ms);
    return () => clearInterval(id);
  }, [ms]);
  return now;
}
