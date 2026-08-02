import React, { useState, useEffect, useMemo, useCallback } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { listCategories, healthCheck } from '../api';
import {
  CORE_STATS,
  DEFAULT_FEATURED_STATS,
  categoryStatId,
  clearFeaturedStats,
  readFeaturedStats,
  statCategory,
  writeFeaturedStats,
} from '../prefs';

/**
 * Build the card for a stat id, or null when it can't be rendered — a
 * `category:*` id whose category has since been emptied, or an id from a
 * newer build. Skipping beats crashing on a stale preference.
 */
function buildStat(id, { totalEntries, categories, health }) {
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

  useEffect(() => {
    Promise.all([listCategories(), healthCheck()])
      .then(([cats, h]) => {
        setCategories(cats);
        setHealth(h);
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  const totalEntries = categories.reduce((sum, c) => sum + c.count, 0);

  const cards = useMemo(
    () => featured.map(id => buildStat(id, { totalEntries, categories, health })).filter(Boolean),
    [featured, totalEntries, categories, health],
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

  const options = [
    ...CORE_STATS,
    ...categories.map(c => ({ id: categoryStatId(c.name), label: `${c.name} (${c.count})` })),
  ];

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
          <div className="form-actions">
            <button className="btn btn--small" onClick={resetStats}>Reset to default</button>
          </div>
        </div>
      )}

      {cards.length > 0 ? (
        <div className="stat-grid">
          {cards.map(card => (
            <div key={card.id} className="stat-card">
              <span className="stat-card__label">{card.label}</span>
              <span className={`stat-card__value${card.tone ? ` stat-card__value--${card.tone}` : ''}`}>
                {card.value}
              </span>
              {card.sub && <span className="stat-card__sub">{card.sub}</span>}
            </div>
          ))}
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
