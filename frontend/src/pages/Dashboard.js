import React, { useState, useEffect } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { listCategories, healthCheck } from '../api';

function Dashboard() {
  const navigate = useNavigate();
  const [categories, setCategories] = useState([]);
  const [health, setHealth] = useState(null);
  const [loading, setLoading] = useState(true);

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
  const healthy = health?.status === 'ok';

  if (loading) return <div className="card">Loading...</div>;

  return (
    <>
      <h1 className="page-title">Dashboard</h1>

      <div className="stat-grid">
        <div className="stat-card">
          <span className="stat-card__label">Total entries</span>
          <span className="stat-card__value">{totalEntries}</span>
        </div>
        <div className="stat-card">
          <span className="stat-card__label">Categories</span>
          <span className="stat-card__value">{categories.length}</span>
        </div>
        <div className="stat-card">
          <span className="stat-card__label">Server</span>
          <span className={`stat-card__value ${healthy ? 'stat-card__value--ok' : 'stat-card__value--bad'}`}>
            {healthy ? 'Healthy' : 'Error'}
          </span>
          {/* The version the *server* reports, next to the client build in the
              header — a mismatch means the binary is serving a stale bundle. */}
          {health?.version && <span className="stat-card__sub">{health.version}</span>}
        </div>
      </div>

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
