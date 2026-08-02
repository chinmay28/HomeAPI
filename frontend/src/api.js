const BASE = '/api';

// Extract a human-readable string from an API entry value.
// The API always returns value as JSON:
//   {"data": "San Jose"}  → plain-text values are wrapped in a data envelope
//   {"lat": 37.3, ...}    → JSON objects/arrays are embedded directly
export function displayValue(v) {
  if (v === null || v === undefined) return '';
  if (typeof v !== 'object') return String(v);
  if (!Array.isArray(v) && Object.prototype.hasOwnProperty.call(v, 'data') && Object.keys(v).length === 1) {
    return String(v.data);
  }
  return JSON.stringify(v);
}

// True when the value is a structured JSON object/array (i.e. not a plain-text
// {data} envelope or a scalar). Used to decide whether to pretty-print.
export function isStructuredValue(v) {
  if (v === null || typeof v !== 'object') return false;
  if (!Array.isArray(v) && Object.prototype.hasOwnProperty.call(v, 'data') && Object.keys(v).length === 1) {
    return false;
  }
  return true;
}

// Like displayValue, but pretty-prints structured JSON across multiple lines.
// Plain-text and scalar values are returned unchanged.
export function prettyValue(v) {
  if (isStructuredValue(v)) return JSON.stringify(v, null, 2);
  return displayValue(v);
}

// ── Stored text ─────────────────────────────────────────────────────────────
// `value` is parsed JSON by the time it reaches us, so the indentation and key
// order someone typed are already gone. `value_text` carries the stored string
// byte-for-byte; everything the GUI shows or edits comes from there so a save
// never reflows JSON the author formatted by hand.

// True when this text is a JSON object or array — matching the server's rule
// for what gets embedded as-is rather than wrapped in a {data} envelope.
export function isJSONText(text) {
  const trimmed = (text || '').trim();
  if (!trimmed || (trimmed[0] !== '{' && trimmed[0] !== '[')) return false;
  try {
    JSON.parse(trimmed);
    return true;
  } catch {
    return false;
  }
}

// Pretty-print JSON text with two-space indents, or null if it isn't JSON.
export function formatJSONText(text) {
  if (!isJSONText(text)) return null;
  return JSON.stringify(JSON.parse(text), null, 2);
}

// The entry's value exactly as stored. Falls back to the parsed value for a
// server too old to send value_text.
export function valueText(entry) {
  if (typeof entry?.value_text === 'string') return entry.value_text;
  return prettyValue(entry?.value);
}

// The value as it should be read on screen: text the author already formatted
// is shown as they wrote it; JSON that arrived on one line (from a curl script,
// say) is pretty-printed, since nobody wants to read that as a single line.
export function readableValue(entry) {
  const text = valueText(entry);
  if (isJSONText(text) && !text.includes('\n')) return formatJSONText(text);
  return text;
}

// A one-line preview for table cells, where a multi-line value would blow up
// the row height.
export function previewValue(entry) {
  if (typeof entry?.value_text === 'string') {
    return entry.value_text.replace(/\s+/g, ' ').trim();
  }
  return displayValue(entry?.value);
}

async function request(path, options = {}) {
  const res = await fetch(`${BASE}${path}`, {
    headers: { 'Content-Type': 'application/json', ...options.headers },
    ...options,
  });
  if (res.status === 204) return null;
  const data = await res.json();
  if (!res.ok) {
    throw new Error(data.error || 'Request failed');
  }
  return data;
}

export function listEntries(params = {}) {
  const query = new URLSearchParams();
  if (params.category) query.set('category', params.category);
  if (params.search) query.set('search', params.search);
  if (params.page) query.set('page', String(params.page));
  if (params.per_page) query.set('per_page', String(params.per_page));
  const qs = query.toString();
  return request(`/entries${qs ? '?' + qs : ''}`);
}

export function getEntry(id) {
  return request(`/entries/${id}`);
}

// Fetch one entry by its key, or null if no entry has that key.
//
// `/api/entries/:id_or_key` resolves a numeric segment as an id, so a key like
// "42" — or one carrying a slash — can't be looked up by path alone. The search
// fallback covers those: it matches on key or value, so the exact key still has
// to be picked out of the results.
export async function getEntryByKey(key) {
  try {
    const entry = await getEntry(encodeURIComponent(key));
    if (entry?.key === key) return entry;
  } catch {
    // Not addressable by path — fall through to the search.
  }
  const found = await listEntries({ search: key, per_page: 200 });
  return found.entries.find(e => e.key === key) || null;
}

export function createEntry(entry) {
  return request('/entries', {
    method: 'POST',
    body: JSON.stringify(entry),
  });
}

export function updateEntry(id, fields) {
  return request(`/entries/${id}`, {
    method: 'PUT',
    body: JSON.stringify(fields),
  });
}

export function deleteEntry(id) {
  return request(`/entries/${id}`, { method: 'DELETE' });
}

// Delete every entry matching the given selectors.
// Accepts { keys, ids, category, search, all, dryRun }; at least one of
// keys/ids/category/search is required unless all is true.
// Resolves to { deleted, matched, dry_run, entries }.
export function bulkDeleteEntries(params = {}) {
  const body = {};
  if (params.keys) body.keys = params.keys;
  if (params.ids) body.ids = params.ids;
  if (params.category) body.category = params.category;
  if (params.search) body.search = params.search;
  if (params.all) body.all = true;
  if (params.dryRun) body.dry_run = true;
  return request('/entries', {
    method: 'DELETE',
    body: JSON.stringify(body),
  });
}

export function listCategories() {
  return request('/categories');
}

export function exportData() {
  return request('/export');
}

export function importData(data) {
  return request('/import', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export function healthCheck() {
  return request('/health');
}
