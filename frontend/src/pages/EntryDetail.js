import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { getEntry, updateEntry, deleteEntry } from '../api';
import Notification from '../components/Notification';

function EntryDetail() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [entry, setEntry] = useState(null);
  const [form, setForm] = useState({ category: '', key: '', value: '' });
  const [editing, setEditing] = useState(false);
  const [loading, setLoading] = useState(true);
  const [notification, setNotification] = useState(null);

  useEffect(() => {
    getEntry(id)
      .then(e => {
        setEntry(e);
        setForm({ category: e.category, key: e.key, value: e.value });
      })
      .catch(err => setNotification({ type: 'error', message: err.message }))
      .finally(() => setLoading(false));
  }, [id]);

  const handleUpdate = async (e) => {
    e.preventDefault();
    try {
      const updated = await updateEntry(id, form);
      setEntry(updated);
      setEditing(false);
      setNotification({ type: 'success', message: 'Entry updated' });
    } catch (err) {
      setNotification({ type: 'error', message: err.message });
    }
  };

  const handleDelete = async () => {
    if (!window.confirm('Delete this entry?')) return;
    try {
      await deleteEntry(id);
      navigate('/entries');
    } catch (err) {
      setNotification({ type: 'error', message: err.message });
    }
  };

  if (loading) return <div className="card">Loading...</div>;
  if (!entry) return <div className="card">Entry not found.</div>;

  return (
    <div>
      <Notification notification={notification} onClear={() => setNotification(null)} />

      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1rem' }}>
        <h1 className="page-title" style={{ marginBottom: 0 }}>Entry Detail</h1>
        <div style={{ display: 'flex', gap: '0.5rem' }}>
          <button className="btn btn-secondary" onClick={() => navigate('/entries')}>Back</button>
          {!editing && <button className="btn btn-primary" onClick={() => setEditing(true)}>Edit</button>}
          <button className="btn btn-danger" onClick={handleDelete}>Delete</button>
        </div>
      </div>

      <div className="card">
        {editing ? (
          <form onSubmit={handleUpdate}>
            <div className="form-group">
              <label>Category</label>
              <input value={form.category} onChange={e => setForm({...form, category: e.target.value})} />
            </div>
            <div className="form-group">
              <label>Key</label>
              <input value={form.key} onChange={e => setForm({...form, key: e.target.value})} required />
            </div>
            <div className="form-group">
              <label>Value</label>
              <textarea value={form.value} onChange={e => setForm({...form, value: e.target.value})} rows={5} />
            </div>
            <div style={{ display: 'flex', gap: '0.5rem' }}>
              <button type="submit" className="btn btn-primary">Save</button>
              <button type="button" className="btn btn-secondary" onClick={() => { setEditing(false); setForm({ category: entry.category, key: entry.key, value: entry.value }); }}>Cancel</button>
            </div>
          </form>
        ) : (
          <div>
            <div className="form-group">
              <label>ID</label>
              <div>{entry.id}</div>
            </div>
            <div className="form-group">
              <label>Category</label>
              <div><span style={{ background: '#e0e7ff', color: '#3730a3', padding: '0.125rem 0.5rem', borderRadius: '12px', fontSize: '0.875rem' }}>{entry.category}</span></div>
            </div>
            <div className="form-group">
              <label>Key</label>
              <div style={{ fontWeight: '500' }}>{entry.key}</div>
            </div>
            <div className="form-group">
              <label>Value</label>
              <div style={{ whiteSpace: 'pre-wrap', background: '#f9fafb', padding: '0.75rem', borderRadius: '6px' }}>{entry.value || <em style={{ color: '#9ca3af' }}>empty</em>}</div>
            </div>
            <div className="form-group">
              <label>Created</label>
              <div style={{ color: '#6b7280', fontSize: '0.875rem' }}>{new Date(entry.created_at).toLocaleString()}</div>
            </div>
            <div className="form-group">
              <label>Updated</label>
              <div style={{ color: '#6b7280', fontSize: '0.875rem' }}>{new Date(entry.updated_at).toLocaleString()}</div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

export default EntryDetail;
