import React, { useState, useEffect, useCallback } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { listEntries, createEntry, deleteEntry, listCategories, previewValue } from '../api';
import Notification from '../components/Notification';
import ValueField from '../components/ValueField';

function Entries() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [data, setData] = useState({ entries: [], total: 0, page: 1, per_page: 50, total_pages: 0 });
  const [categories, setCategories] = useState([]);
  const [loading, setLoading] = useState(true);
  const [form, setForm] = useState({ category: '', key: '', value: '' });
  const [notification, setNotification] = useState(null);

  const category = searchParams.get('category') || '';
  const search = searchParams.get('search') || '';
  const page = parseInt(searchParams.get('page') || '1', 10);
  // The create form's open state lives in the URL so the app-shell's floating
  // action button ("New entry", which is just a link to ?new=1) can open it
  // from any page.
  const showForm = searchParams.get('new') === '1';

  const setShowForm = useCallback((open) => {
    setSearchParams(prev => {
      const params = new URLSearchParams(prev);
      if (open) params.set('new', '1');
      else params.delete('new');
      return params;
    }, { replace: true });
  }, [setSearchParams]);

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
    <>
      <Notification notification={notification} onClear={() => setNotification(null)} />

      <div className="toolbar">
        <h1 className="page-title">Entries</h1>
        {/* Opening the form is the header's "New entry" button (and the
            floating action button on phones); this only closes it again. */}
        {showForm && (
          <div className="toolbar-actions">
            <button className="btn" onClick={() => setShowForm(false)}>Cancel</button>
          </div>
        )}
      </div>

      {showForm && (
        <div className="card">
          <form onSubmit={handleCreate}>
            <div className="entry-form-grid">
              <div className="form-group">
                <label htmlFor="new-category">Category</label>
                <input id="new-category" value={form.category} onChange={e => setForm({...form, category: e.target.value})} placeholder="default" />
              </div>
              <div className="form-group">
                <label htmlFor="new-key">Key *</label>
                <input id="new-key" value={form.key} onChange={e => setForm({...form, key: e.target.value})} placeholder="e.g. AAPL" required />
              </div>
              <ValueField
                id="new-value"
                value={form.value}
                onChange={value => setForm({ ...form, value })}
                minRows={3}
                placeholder={'e.g. Apple Inc.\nor {"ticker": "AAPL"}'}
              />
            </div>
            <div className="form-actions">
              <button type="submit" className="btn btn--primary">Save</button>
            </div>
          </form>
        </div>
      )}

      <div className="filters">
        <select aria-label="Filter by category" value={category} onChange={e => setFilter('category', e.target.value)}>
          <option value="">All Categories</option>
          {categories.map(c => (
            <option key={c.name} value={c.name}>{c.name} ({c.count})</option>
          ))}
        </select>
        <input
          aria-label="Search entries"
          placeholder="Search entries..."
          value={search}
          onChange={e => setFilter('search', e.target.value)}
        />
      </div>

      <div className="card">
        {loading ? (
          <div>Loading...</div>
        ) : data.entries.length === 0 ? (
          <div className="empty-state">No entries found.</div>
        ) : (
          <>
            <div className="table-wrap">
              <table className="responsive-table">
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
                      <td data-label="Category"><span className="badge">{entry.category}</span></td>
                      <td data-label="Key" className="cell-key"><Link to={`/entries/${entry.id}`}>{entry.key}</Link></td>
                      <td data-label="Value" className="cell-value">{previewValue(entry)}</td>
                      <td data-label="Updated" className="cell-meta">{new Date(entry.updated_at).toLocaleString()}</td>
                      <td className="cell-actions">
                        <button className="btn btn--danger btn--small" onClick={() => handleDelete(entry.id)}>Delete</button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            {data.total_pages > 1 && (
              <div className="pager">
                {Array.from({ length: data.total_pages }, (_, i) => i + 1).map(p => (
                  <button
                    key={p}
                    className={`btn btn--small ${p === data.page ? 'btn--active' : ''}`}
                    aria-current={p === data.page ? 'page' : undefined}
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
    </>
  );
}

export default Entries;
