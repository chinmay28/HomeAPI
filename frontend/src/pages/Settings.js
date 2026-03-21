import React, { useState } from 'react';
import { exportData, importData } from '../api';
import Notification from '../components/Notification';

function Settings() {
  const [notification, setNotification] = useState(null);
  const [importMode, setImportMode] = useState('merge');
  const [importFile, setImportFile] = useState(null);
  const [importing, setImporting] = useState(false);

  const handleExport = async () => {
    try {
      const data = await exportData();
      const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `homeapi-export-${new Date().toISOString().slice(0, 10)}.json`;
      a.click();
      URL.revokeObjectURL(url);
      setNotification({ type: 'success', message: `Exported ${data.entries.length} entries` });
    } catch (err) {
      setNotification({ type: 'error', message: err.message });
    }
  };

  const handleImport = async (e) => {
    e.preventDefault();
    if (!importFile) return;
    setImporting(true);
    try {
      const text = await importFile.text();
      const parsed = JSON.parse(text);
      const payload = {
        entries: parsed.entries || parsed,
        mode: importMode,
      };
      const result = await importData(payload);
      setNotification({ type: 'success', message: `Imported: ${result.imported}, Skipped: ${result.skipped}, Errors: ${result.errors}` });
      setImportFile(null);
    } catch (err) {
      setNotification({ type: 'error', message: err.message });
    } finally {
      setImporting(false);
    }
  };

  return (
    <div>
      <Notification notification={notification} onClear={() => setNotification(null)} />
      <h1 className="page-title">Settings</h1>

      <div className="card">
        <h2 style={{ fontSize: '1.125rem', fontWeight: '600', marginBottom: '1rem' }}>Export Data</h2>
        <p style={{ color: '#6b7280', marginBottom: '1rem', fontSize: '0.875rem' }}>
          Download all entries as a JSON file for backup or migration.
        </p>
        <button className="btn btn-primary" onClick={handleExport}>Export Data</button>
      </div>

      <div className="card">
        <h2 style={{ fontSize: '1.125rem', fontWeight: '600', marginBottom: '1rem' }}>Import Data</h2>
        <p style={{ color: '#6b7280', marginBottom: '1rem', fontSize: '0.875rem' }}>
          Import entries from a previously exported JSON file.
        </p>
        <form onSubmit={handleImport}>
          <div className="form-group">
            <label>File</label>
            <input type="file" accept=".json" onChange={e => setImportFile(e.target.files[0])} />
          </div>
          <div className="form-group">
            <label>Import Mode</label>
            <select value={importMode} onChange={e => setImportMode(e.target.value)} style={{ width: 'auto' }}>
              <option value="merge">Merge (skip existing)</option>
              <option value="replace">Replace (overwrite existing)</option>
            </select>
          </div>
          <button type="submit" className="btn btn-primary" disabled={!importFile || importing}>
            {importing ? 'Importing...' : 'Import'}
          </button>
        </form>
      </div>
    </div>
  );
}

export default Settings;
