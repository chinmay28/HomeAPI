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

  if (loading) return <div className="card">Loading...</div>;

  return (
    <div>
      <h1 className="page-title">Dashboard</h1>

      <div className="stat-grid">
        <div className="card">
          <div style={{ fontSize: '0.875rem', color: '#6b7280' }}>Total Entries</div>
          <div style={{ fontSize: '2rem', fontWeight: '700' }}>{totalEntries}</div>
        </div>
        <div className="card">
          <div style={{ fontSize: '0.875rem', color: '#6b7280' }}>Status</div>
          <div style={{ fontSize: '2rem', fontWeight: '700', color: health?.status === 'ok' ? '#16a34a' : '#dc2626' }}>
            {health?.status === 'ok' ? 'Healthy' : 'Error'}
          </div>
          {health?.version && <div style={{ fontSize: '0.75rem', color: '#9ca3af' }}>v{health.version}</div>}
        </div>
      </div>

      <div className="card">
        <h2 style={{ fontSize: '1.125rem', fontWeight: '600', marginBottom: '1rem' }}>Categories</h2>
        {categories.length === 0 ? (
          <div className="empty-state">
            <p>No entries yet.</p>
            <Link to="/entries" className="btn btn-primary" style={{ marginTop: '1rem', display: 'inline-flex' }}>
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
                    <td data-label="Category" style={{ fontWeight: '500' }}>
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
    </div>
  );
}

export default Dashboard;
