import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { getEntryByKey, valueText } from '../api';

/** The entries the Minion 3 display reads and writes (see docs/MINION3_DISPLAY.md). */
export const MINION_PARAMS_KEY = 'minion-params';
export const MINION_LAST_SEEN_KEY = 'minion3-last-seen';
export const MINION_QUOTES_KEY = 'minion-quotes';

/** The server-rendered preview: the exact frame the panel shows, at 2×. */
export const MINION_FRAME_SRC = '/api/displays/minion3/frame.png?scale=2';

/**
 * Parse the `minion3-last-seen` telemetry the device's wake leaves behind,
 * or null when the entry is missing or not what the server writes.
 */
export function parseLastSeen(entry) {
  if (!entry) return null;
  try {
    const seen = JSON.parse(valueText(entry));
    if (!seen || typeof seen.wake !== 'number') return null;
    const at = seen.at ? new Date(seen.at) : null;
    return { wake: seen.wake, at: at && !Number.isNaN(at.getTime()) ? at : null };
  } catch {
    return null;
  }
}

function formatSeen(at) {
  return at.toLocaleString(undefined, {
    weekday: 'short', month: 'numeric', day: 'numeric', hour: '2-digit', minute: '2-digit',
  });
}

/**
 * The Minion 3 e-paper panel: the frame the device will fetch next, when it
 * last did, and the way to its configuration. Nothing here is state of its
 * own — one image and two fields from `minion3-last-seen` — and the card
 * only appears once something Minion-shaped exists in the store, so a fresh
 * install's dashboard is not haunted by a panel it does not have.
 */
function MinionCard() {
  // undefined while loading; null when absent.
  const [params, setParams] = useState(undefined);
  const [lastSeen, setLastSeen] = useState(undefined);
  const [quotes, setQuotes] = useState(undefined);

  useEffect(() => {
    let cancelled = false;
    const fetchKey = (key) => getEntryByKey(key).catch(() => null);
    Promise.all([fetchKey(MINION_PARAMS_KEY), fetchKey(MINION_LAST_SEEN_KEY), fetchKey(MINION_QUOTES_KEY)])
      .then(([p, seen, q]) => {
        if (cancelled) return;
        setParams(p || null);
        setLastSeen(seen || null);
        setQuotes(q || null);
      });
    return () => { cancelled = true; };
  }, []);

  if (params === undefined || lastSeen === undefined || quotes === undefined) return null;
  if (!params && !lastSeen && !quotes) return null;

  const seen = parseLastSeen(lastSeen);

  return (
    <div className="card minion-card">
      <div className="minion-card__head">
        <h2 className="card__title">Minion 3</h2>
        {params ? (
          <Link to={`/entries/${params.id}`} className="btn btn--small">Edit params</Link>
        ) : (
          <Link to="/entries?new=1" className="btn btn--small">Create params</Link>
        )}
      </div>
      <img
        className="minion-card__frame"
        src={MINION_FRAME_SRC}
        alt="Minion 3 panel preview"
        width={500}
        height={244}
      />
      <dl className="minion-card__meta">
        <div>
          <dt>Last seen</dt>
          <dd>{seen?.at ? formatSeen(seen.at) : 'Never'}</dd>
        </div>
        <div>
          <dt>Wake</dt>
          <dd>{seen ? String(seen.wake) : '—'}</dd>
        </div>
      </dl>
      {!params && (
        <p className="card__hint minion-card__hint">
          No <code>{MINION_PARAMS_KEY}</code> entry yet — the panel is showing the built-in layout and says
          “Cfg?” in the corner until you create one.
        </p>
      )}
    </div>
  );
}

export default MinionCard;
