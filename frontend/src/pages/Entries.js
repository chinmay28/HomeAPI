import React, { useState, useEffect, useCallback } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { listEntries, createEntry, deleteEntry, listCategories } from '../api';
import Notification from '../components/Notification';

function Entries() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [data, setData] = useState({ entries: [], total: 0, page: 1, per_page: 50, total_pages: 0 });
  const [categories, setCategories] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({ category: '', key: '', value: '' });
  const [notification, setNotification] = useState(null);

  const category = searchParams.get('category') || '';
  const search = searchParams.get('search') || '';
  const page = parseInt(searchParams.get('page') || '1', 10);

  const fetchEntries = useCallback(() => {
    setLoading(true);
    listEntries({ category, search, page, per_page: 50 })
      .then(setData)
      .catch(err => setNotification({ type: 'error', message: err.message }))
      .finally(() => setLoading(false));
  }, [category, search, page]);

  useEffect(() => { fetchEntries(); }, [fetchEntries]);
  useEffect(() => { listCategories().then(setCategories).catch(() => {}); }, []);

  const handleCreate = async (e) => {
    e.preventDefault();
    try {
      await createEntry({
        category: form.category || 'default',
        key: form.key,
        value: form.value,
      });
      setNotification({ type: 'success', message: 'Entry created' });
      setForm({ category: '', key: '', value: '' });
      setShowForm(false);
      fetchEntries();
      listCategories().then(setCategories).catch(() => {});
    } catch (err) {
      setNotification({ type: 'error', message: err.message });
    }
  };

  const handleDelete = async (id) => {
    if (!window.confirm('Delete this entry?')) return;
    try {
      await deleteEntry(id);
      setNotification({ type: 'success', message: 'Entry deleted' });
      fetchEntries();
      listCategories().then(setCategories).catch(() => {});
    } catch (err) {
      setNotification({ type: 'error', message: err.message });
    }
  };

  const setFilter = (key, value) => {
    const params = new URLSearchParams(searchParams);
    if (value) params.set(key, value);
    else params.delete(key);
    params.delete('page');
    setSearchParams(params);
  };

  return (
    <div>
      <Notification notification={notification} onClear={() => setNotification(null)} />

      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1rem' }}>
        <h1 className="page-title" style={{ marginBottom: 0 }}>Entries</h1>
        <button className="btn btn-primary" onClick={() => setShowForm(!showForm)}>
          {showForm ? 'Cancel' : 'New Entry'}
        </button>
      </div>

      {showForm && (
        <div className="card">
          <form onSubmit={handleCreate}>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 2fr', gap: '1rem', alignItems: 'end' }}>
              <div className="form-group">
                <label>Category</label>
                <input value={form.category} onChange={e => setForm({...form, category: e.target.value})} placeholder="default" />
              </div>
              <div className="form-group">
                <label>Key *</label>
                <input value={form.key} onChange={e => setForm({...form, key: e.target.value})} placeholder="e.g. AAPL" required />
              </div>
              <div className="form-group">
                <label>Value</label>
                <input value={form.value} onChange={e => setForm({...form, value: e.target.value})} placeholder="e.g. Apple Inc." />
              </div>
            </div>
            <button type="submit" className="btn btn-primary" style={{ marginTop: '0.5rem' }}>Save</button>
          </form>
        </div>
      )}

      <div style={{ display: 'flex', gap: '1rem', marginBottom: '1rem' }}>
        <select value={category} onChange={e => setFilter('category', e.target.value)} style={{ width: 'auto', minWidth: '150px' }}>
          <option value="">All Categories</option>
          {categories.map(c => (
            <option key={c.name} value={c.name}>{c.name} ({c.count})</option>
          ))}
        </select>
        <input
          placeholder="Search entries..."
          value={search}
          onChange={e => setFilter('search', e.target.value)}
          style={{ maxWidth: '300px' }}
        />
      </div>

      <div className="card">
        {loading ? (
          <div>Loading...</div>
        ) : data.entries.length === 0 ? (
          <div className="empty-state">No entries found.</div>
        ) : (
          <>
            <table>
              <thead>
                <tr>
                  <th>Category</th>
                  <th>Key</th>
                  <th>Value</th>
                  <th>Updated</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                {data.entries.map(entry => (
                  <tr key={entry.id}>
                    <td><span style={{ background: '#e0e7ff', color: '#3730a3', padding: '0.125rem 0.5rem', borderRadius: '12px', fontSize: '0.8rem' }}>{entry.category}</span></td>
                    <td><Link to={`/entries/${entry.id}`} style={{ fontWeight: '500' }}>{entry.key}</Link></td>
                    <td style={{ maxWidth: '300px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{entry.value}</td>
                    <td style={{ fontSize: '0.8rem', color: '#6b7280' }}>{new Date(entry.updated_at).toLocaleString()}</td>
                    <td>
                      <button className="btn btn-danger" style={{ padding: '0.2rem 0.5rem', fontSize: '0.75rem' }} onClick={() => handleDelete(entry.id)}>Delete</button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            {data.total_pages > 1 && (
              <div style={{ display: 'flex', justifyContent: 'center', gap: '0.5rem', marginTop: '1rem' }}>
                {Array.from({ length: data.total_pages }, (_, i) => i + 1).map(p => (
                  <button
                    key={p}
                    className={`btn ${p === data.page ? 'btn-primary' : 'btn-secondary'}`}
                    style={{ padding: '0.25rem 0.75rem', minWidth: '2rem' }}
                    onClick={() => { const params = new URLSearchParams(searchParams); params.set('page', String(p)); setSearchParams(params); }}
                  >
                    {p}
                  </button>
                ))}
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}

export default Entries;
