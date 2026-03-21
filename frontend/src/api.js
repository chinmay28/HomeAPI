const BASE = '/api';

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
