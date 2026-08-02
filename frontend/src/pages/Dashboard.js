import React, { useState, useEffect, useMemo, useCallback } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { getEntryByKey, listCategories, listEntries, healthCheck, previewValue } from '../api';
import {
  CORE_STATS,
  DEFAULT_FEATURED_STATS,
  categoryStatId,
  clearFeaturedStats,
  entryStatId,
  readFeaturedStats,
  statCategory,
  statEntryKey,
  writeFeaturedStats,
} from '../prefs';

/** How much of a pinned entry's value a card shows before it is cut off. */
const VALUE_MAX = 140;

/** Beyond this many characters a value is prose, not a figure, and is set in
 *  running text rather than at headline size. */
const VALUE_HEADLINE_MAX = 16;

/** The server's own ceiling on per_page — the most the picker can list at once. */
const PICKER_LIMIT = 200;

/** Below this many entries the picker's list is short enough to read whole, so
 *  the search box would only be in the way. */
const PICKER_SEARCH_MIN = 12;

function truncate(text) {
  return text.length > VALUE_MAX ? `${text.slice(0, VALUE_MAX)}…` : text;
}

/**
 * Build the card for a stat id, or null when it can't be rendered — a
 * `category:*` id whose category has since been emptied, an `entry:*` id whose
 * entry is gone, or an id from a newer build. Skipping beats crashing on a
 * stale preference.
 */
function buildStat(id, { totalEntries, categories, health, pinnedEntries }) {
  const entryKey = statEntryKey(id);
  if (entryKey !== null) {
    const entry = pinnedEntries[entryKey];
    // Still in flight. A placeholder holds the card's place so the row doesn't
    // reshuffle under the reader a moment after it paints.
    if (entry === undefined) return { id, label: entryKey, value: '…' };
    if (entry === null) return null;
    const preview = previewValue(entry);
    return {
      id,
      label: entry.key,
      value: truncate(preview) || '—',
      long: preview.length > VALUE_HEADLINE_MAX,
      sub: entry.category,
      to: `/entries/${entry.id}`,
    };
  }

  const category = statCategory(id);
  if (category !== null) {
    const match = categories.find(c => c.name === category);
    if (!match) return null;
    return { id, label: match.name, value: String(match.count), sub: 'entries' };
  }

  switch (id) {
    case 'total':
      return { id, label: 'Total entries', value: String(totalEntries) };
    case 'categories':
      return { id, label: 'Categories', value: String(categories.length) };
    case 'server': {
      const healthy = health?.status === 'ok';
      return {
        id,
        label: 'Server',
        value: healthy ? 'Healthy' : 'Error',
        tone: healthy ? 'ok' : 'bad',
        sub: health?.version,
      };
    }
    case 'largest': {
      if (categories.length === 0) return null;
      const biggest = categories.reduce((a, b) => (b.count > a.count ? b : a));
      return { id, label: 'Largest category', value: biggest.name, sub: `${biggest.count} entries` };
    }
    default:
      return null;
  }
}

function Dashboard() {
  const navigate = useNavigate();
  const [categories, setCategories] = useState([]);
  const [health, setHealth] = useState(null);
  const [loading, setLoading] = useState(true);
  const [featured, setFeatured] = useState(readFeaturedStats);
  const [customizing, setCustomizing] = useState(false);
  // key → entry, or null once we know there is no such entry. A key that is
  // absent is still loading.
  const [pinnedEntries, setPinnedEntries] = useState({});
  // The entries offered in the picker. Only fetched while it is open.
  const [entryOptions, setEntryOptions] = useState(null);
  const [entryTotal, setEntryTotal] = useState(0);
  const [entryFilter, setEntryFilter] = useState('');

  useEffect(() => {
    Promise.all([listCategories(), healthCheck()])
      .then(([cats, h]) => {
        setCategories(cats);
        setHealth(h);
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  const pinnedKeys = useMemo(
    () => featured.map(statEntryKey).filter(key => key !== null),
    [featured],
  );

  // Pinned entries are fetched one by one rather than off the entries list:
  // there is no page size that is both small enough to be cheap and large
  // enough to be sure a given key is on it.
  useEffect(() => {
    if (pinnedKeys.length === 0) {
      setPinnedEntries({});
      return undefined;
    }
    let cancelled = false;
    Promise.all(pinnedKeys.map(key => getEntryByKey(key).catch(() => null)))
      .then(entries => {
        if (cancelled) return;
        setPinnedEntries(Object.fromEntries(pinnedKeys.map((key, i) => [key, entries[i]])));
      });
    return () => { cancelled = true; };
  }, [pinnedKeys]);

  // The picker's entry list. Searching runs on the server, so a store larger
  // than one page is still fully reachable.
  useEffect(() => {
    if (!customizing) return undefined;
    let cancelled = false;
    const timer = window.setTimeout(() => {
      listEntries({ search: entryFilter, per_page: PICKER_LIMIT })
        .then(data => {
          if (cancelled) return;
          setEntryOptions(data.entries);
          setEntryTotal(data.total);
        })
        .catch(() => { if (!cancelled) setEntryOptions([]); });
    }, entryFilter ? 200 : 0);
    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [customizing, entryFilter]);

  const totalEntries = categories.reduce((sum, c) => sum + c.count, 0);

  const cards = useMemo(
    () => featured
      .map(id => buildStat(id, { totalEntries, categories, health, pinnedEntries }))
      .filter(Boolean),
    [featured, totalEntries, categories, health, pinnedEntries],
  );

  const toggleStat = useCallback((id) => {
    setFeatured(prev => {
      // Appending keeps the row in the order the user built it, rather than
      // having cards jump around as they are ticked on and off.
      const next = prev.includes(id) ? prev.filter(s => s !== id) : [...prev, id];
      writeFeaturedStats(next);
      return next;
    });
  }, []);

  const resetStats = useCallback(() => {
    clearFeaturedStats();
    setFeatured(DEFAULT_FEATURED_STATS);
  }, []);

  if (loading) return <div className="card">Loading...</div>;

  // Pinned keys the current search or page doesn't cover are listed anyway:
  // whatever is featured must always be untickable from here.
  const listedKeys = new Set((entryOptions || []).map(e => e.key));
  const pinnedElsewhere = pinnedKeys.filter(key => !listedKeys.has(key));
  const entryChoices = [
    ...pinnedElsewhere.map(key => ({ key, category: pinnedEntries[key]?.category })),
    ...(entryOptions || []),
  ];
  const showEntrySearch = entryTotal >= PICKER_SEARCH_MIN || entryFilter !== '';

  return (
    <>
      <div className="toolbar">
        <h1 className="page-title">Dashboard</h1>
        <div className="toolbar-actions">
          <button
            className="btn btn--small"
            aria-expanded={customizing}
            onClick={() => setCustomizing(!customizing)}
          >
            {customizing ? 'Done' : 'Customize stats'}
          </button>
        </div>
      </div>

      {customizing && (
        <div className="card">
          <h2 className="card__title">Featured stats</h2>
          <p className="card__hint">
            Pick what this dashboard shows. Saved in this browser.
          </p>
          <fieldset className="stat-picker">
            <legend className="stat-picker__legend">Overview</legend>
            {CORE_STATS.map(stat => (
              <label key={stat.id} className="stat-picker__option">
                <input
                  type="checkbox"
                  checked={featured.includes(stat.id)}
                  onChange={() => toggleStat(stat.id)}
                />
                <span>{stat.label}</span>
              </label>
            ))}
          </fieldset>
          {categories.length > 0 && (
            <fieldset className="stat-picker">
              <legend className="stat-picker__legend">Category counts</legend>
              {categories.map(cat => {
                const id = categoryStatId(cat.name);
                return (
                  <label key={id} className="stat-picker__option">
                    <input
                      type="checkbox"
                      checked={featured.includes(id)}
                      onChange={() => toggleStat(id)}
                    />
                    <span>{cat.name} ({cat.count})</span>
                  </label>
                );
              })}
            </fieldset>
          )}
          <fieldset className="stat-picker">
            <legend className="stat-picker__legend">Entry values</legend>
            {showEntrySearch && (
              <input
                className="stat-picker__search"
                type="search"
                aria-label="Search entries to feature"
                placeholder="Search entries..."
                value={entryFilter}
                onChange={e => setEntryFilter(e.target.value)}
              />
            )}
            {entryOptions === null ? (
              <span className="card__hint">Loading entries...</span>
            ) : entryChoices.length === 0 ? (
              <span className="card__hint">
                {entryFilter ? 'No entries match that search.' : 'No entries yet.'}
              </span>
            ) : (
              entryChoices.map(entry => {
                const id = entryStatId(entry.key);
                return (
                  <label key={id} className="stat-picker__option">
                    <input
                      type="checkbox"
                      checked={featured.includes(id)}
                      onChange={() => toggleStat(id)}
                    />
                    <span>{entry.key}</span>
                  </label>
                );
              })
            )}
            {entryTotal > (entryOptions?.length || 0) && (
              <span className="card__hint">
                Showing {entryOptions.length} of {entryTotal} entries — search to narrow.
              </span>
            )}
          </fieldset>
          <div className="form-actions">
            <button className="btn btn--small" onClick={resetStats}>Reset to default</button>
          </div>
        </div>
      )}

      {cards.length > 0 ? (
        <div className="stat-grid">
          {cards.map(card => {
            const body = (
              <>
                <span className="stat-card__label">{card.label}</span>
                <span
                  className={
                    'stat-card__value'
                    + (card.tone ? ` stat-card__value--${card.tone}` : '')
                    + (card.long ? ' stat-card__value--text' : '')
                  }
                >
                  {card.value}
                </span>
                {card.sub && <span className="stat-card__sub">{card.sub}</span>}
              </>
            );
            // A pinned entry's card doubles as the way into that entry.
            return card.to ? (
              <Link key={card.id} to={card.to} className="stat-card stat-card--link">
                {body}
              </Link>
            ) : (
              <div key={card.id} className="stat-card">{body}</div>
            );
          })}
        </div>
      ) : (
        !customizing && (
          <div className="card">
            <div className="empty-state">
              <p>No stats featured.</p>
              <button className="btn btn--primary" onClick={() => setCustomizing(true)}>
                Choose stats
              </button>
            </div>
          </div>
        )
      )}

      <div className="card">
        <h2 className="card__title">Categories</h2>
        {categories.length === 0 ? (
          <div className="empty-state">
            <p>No entries yet.</p>
            <Link to="/entries?new=1" className="btn btn--primary">
              Create your first entry
            </Link>
          </div>
        ) : (
          <div className="table-wrap">
            <table className="responsive-table">
              <thead>
                <tr>
                  <th>Category</th>
                  <th>Entries</th>
                </tr>
              </thead>
              <tbody>
                {categories.map(cat => {
                  const to = `/entries?category=${encodeURIComponent(cat.name)}`;
                  return (
                    <tr
                      key={cat.name}
                      className="clickable-row"
                      onClick={() => navigate(to)}
                      onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); navigate(to); } }}
                      tabIndex={0}
                      role="link"
                    >
                      <td data-label="Category" className="cell-key">
                        <Link to={to} onClick={(e) => e.stopPropagation()}>{cat.name}</Link>
                      </td>
                      <td data-label="Entries">{cat.count}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </>
  );
}

export default Dashboard;
