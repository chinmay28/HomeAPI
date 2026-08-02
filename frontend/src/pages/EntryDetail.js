import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { getEntry, updateEntry, deleteEntry, displayValue, prettyValue, isStructuredValue } from '../api';
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
        setForm({ category: e.category, key: e.key, value: displayValue(e.value) });
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
    <>
      <Notification notification={notification} onClear={() => setNotification(null)} />

      <div className="toolbar">
        <h1 className="page-title">Entry Detail</h1>
        <div className="toolbar-actions">
          <button className="btn" onClick={() => navigate('/entries')}>Back</button>
          {!editing && <button className="btn btn--primary" onClick={() => setEditing(true)}>Edit</button>}
          <button className="btn btn--danger" onClick={handleDelete}>Delete</button>
        </div>
      </div>

      <div className="card">
        {editing ? (
          <form onSubmit={handleUpdate}>
            <div className="form-group">
              <label htmlFor="edit-category">Category</label>
              <input id="edit-category" value={form.category} onChange={e => setForm({...form, category: e.target.value})} />
            </div>
            <div className="form-group">
              <label htmlFor="edit-key">Key</label>
              <input id="edit-key" value={form.key} onChange={e => setForm({...form, key: e.target.value})} required />
            </div>
            <div className="form-group">
              <label htmlFor="edit-value">Value</label>
              <textarea id="edit-value" value={form.value} onChange={e => setForm({...form, value: e.target.value})} rows={6} />
            </div>
            <div className="form-actions">
              <button type="submit" className="btn btn--primary">Save</button>
              <button type="button" className="btn" onClick={() => { setEditing(false); setForm({ category: entry.category, key: entry.key, value: displayValue(entry.value) }); }}>Cancel</button>
            </div>
          </form>
        ) : (
          <div>
            <div className="detail-field">
              <div className="detail-field__label">ID</div>
              <div className="detail-field__value">{entry.id}</div>
            </div>
            <div className="detail-field">
              <div className="detail-field__label">Category</div>
              <div className="detail-field__value"><span className="badge">{entry.category}</span></div>
            </div>
            <div className="detail-field">
              <div className="detail-field__label">Key</div>
              <div className="detail-field__value detail-field__value--key">{entry.key}</div>
            </div>
            <div className="detail-field">
              <div className="detail-field__label">
                Value{isStructuredValue(entry.value) && <span className="value-badge">JSON</span>}
              </div>
              <pre className={`value-display${isStructuredValue(entry.value) ? ' value-json' : ''}`}>{prettyValue(entry.value) || <em className="muted">empty</em>}</pre>
            </div>
            <div className="detail-field">
              <div className="detail-field__label">Created</div>
              <div className="detail-field__value detail-field__value--meta">{new Date(entry.created_at).toLocaleString()}</div>
            </div>
            <div className="detail-field">
              <div className="detail-field__label">Updated</div>
              <div className="detail-field__value detail-field__value--meta">{new Date(entry.updated_at).toLocaleString()}</div>
            </div>
          </div>
        )}
      </div>
    </>
  );
}

export default EntryDetail;
