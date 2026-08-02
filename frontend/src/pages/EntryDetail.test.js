import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import EntryDetail from './EntryDetail';
import * as api from '../api';

jest.mock('../api', () => ({
  ...jest.requireActual('../api'),
  getEntry: jest.fn(),
  updateEntry: jest.fn(),
  deleteEntry: jest.fn(),
}));

// The stored string the server would have produced for this parsed value —
// value_text is the exact text in the database, so tests that don't care about
// formatting get the obvious one rather than having to spell it out.
function storedTextFor(value) {
  if (value && typeof value === 'object' && !Array.isArray(value)
      && Object.keys(value).length === 1 && 'data' in value) {
    return String(value.data);
  }
  return JSON.stringify(value);
}

function makeEntry(overrides) {
  const entry = {
    id: 1,
    category: 'default',
    key: 'city',
    value: { data: 'San Jose' },
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
    ...overrides,
  };
  if (entry.value_text === undefined) entry.value_text = storedTextFor(entry.value);
  return entry;
}

function renderDetail(id = '1') {
  return render(
    <MemoryRouter initialEntries={[`/entries/${id}`]}>
      <Routes>
        <Route path="/entries/:id" element={<EntryDetail />} />
      </Routes>
    </MemoryRouter>
  );
}

// ── Value display ────────────────────────────────────────────────────────────

test('displays plain-text value from {data} envelope — does not show [object Object]', async () => {
  api.getEntry.mockResolvedValue(makeEntry({ value: { data: 'San Jose' } }));
  renderDetail();

  expect(await screen.findByText('San Jose')).toBeInTheDocument();
  expect(screen.queryByText('[object Object]')).not.toBeInTheDocument();
});

test('pretty-prints JSON object value across multiple lines', async () => {
  api.getEntry.mockResolvedValue(makeEntry({ key: 'loc', value: { lat: 37.3, lon: -121.9 } }));
  renderDetail();

  // Value is rendered pretty-printed; findByText normalizes whitespace, so the
  // multi-line JSON collapses to a single spaced string.
  expect(await screen.findByText('{ "lat": 37.3, "lon": -121.9 }')).toBeInTheDocument();
  expect(screen.queryByText('[object Object]')).not.toBeInTheDocument();
});

test('shows "empty" placeholder when value is empty string', async () => {
  api.getEntry.mockResolvedValue(makeEntry({ value: { data: '' } }));
  renderDetail();

  expect(await screen.findByText('empty')).toBeInTheDocument();
});

// ── Edit form ────────────────────────────────────────────────────────────────

test('edit form textarea shows plain string, not [object Object]', async () => {
  api.getEntry.mockResolvedValue(makeEntry({ value: { data: 'San Jose' } }));
  renderDetail();

  await screen.findByText('San Jose');
  fireEvent.click(screen.getByRole('button', { name: 'Edit' }));

  const textarea = screen.getByDisplayValue('San Jose');
  expect(textarea.value).toBe('San Jose');
  expect(textarea.value).not.toContain('[object Object]');
});

test('edit form textarea shows JSON string for object values', async () => {
  api.getEntry.mockResolvedValue(makeEntry({ key: 'loc', value: { lat: 37.3 } }));
  renderDetail();

  // Wait for load via the key, then edit — the textarea shows the stored text,
  // which for this entry is compact.
  await screen.findByText('loc');
  fireEvent.click(screen.getByRole('button', { name: 'Edit' }));

  const textarea = screen.getByDisplayValue('{"lat":37.3}');
  expect(textarea.value).toBe('{"lat":37.3}');
  expect(textarea.value).not.toContain('[object Object]');
});

test('cancel restores value as plain string, not [object Object]', async () => {
  api.getEntry.mockResolvedValue(makeEntry({ value: { data: 'San Jose' } }));
  renderDetail();

  await screen.findByText('San Jose');
  fireEvent.click(screen.getByRole('button', { name: 'Edit' }));
  fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));

  // Back to read view — value must display correctly
  expect(screen.getByText('San Jose')).toBeInTheDocument();
  expect(screen.queryByText('[object Object]')).not.toBeInTheDocument();
});

// ── Structure ────────────────────────────────────────────────────────────────

test('renders key, category, and id fields', async () => {
  api.getEntry.mockResolvedValue(makeEntry({ id: 7, category: 'watchlist', key: 'AAPL', value: { data: 'Apple' } }));
  renderDetail('7');

  expect(await screen.findByText('AAPL')).toBeInTheDocument();
  expect(screen.getByText('watchlist')).toBeInTheDocument();
  expect(screen.getByText('7')).toBeInTheDocument();
});

test('shows Entry not found when API returns null', async () => {
  api.getEntry.mockRejectedValue(new Error('Not found'));
  renderDetail('999');

  expect(await screen.findByText('Entry not found.')).toBeInTheDocument();
});

// ── Formatted JSON is retained ───────────────────────────────────────────────

const HAND_FORMATTED = '{\n    "lat": 37.3,\n    "lon": -121.9\n}';

test('shows hand-formatted JSON exactly as it was written', async () => {
  api.getEntry.mockResolvedValue(makeEntry({ key: 'loc', value_text: HAND_FORMATTED }));
  renderDetail();

  const block = await screen.findByText(/"lat"/);
  expect(block.textContent).toBe(HAND_FORMATTED);
});

test('edit textarea keeps the stored formatting instead of collapsing it', async () => {
  api.getEntry.mockResolvedValue(makeEntry({ key: 'loc', value_text: HAND_FORMATTED }));
  renderDetail();

  await screen.findByText('loc');
  fireEvent.click(screen.getByRole('button', { name: 'Edit' }));

  expect(screen.getByLabelText('Value').value).toBe(HAND_FORMATTED);
});

test('saving an untouched entry sends the formatting back unchanged', async () => {
  api.getEntry.mockResolvedValue(makeEntry({ key: 'loc', value_text: HAND_FORMATTED }));
  api.updateEntry.mockResolvedValue(makeEntry({ key: 'loc', value_text: HAND_FORMATTED }));
  renderDetail();

  await screen.findByText('loc');
  fireEvent.click(screen.getByRole('button', { name: 'Edit' }));
  fireEvent.click(screen.getByRole('button', { name: 'Save' }));

  await waitFor(() => expect(api.updateEntry).toHaveBeenCalled());
  expect(api.updateEntry.mock.calls[0][1].value).toBe(HAND_FORMATTED);
});

test('Format JSON pretty-prints a one-line value', async () => {
  api.getEntry.mockResolvedValue(makeEntry({ key: 'loc', value_text: '{"lat":37.3}' }));
  renderDetail();

  await screen.findByText('loc');
  fireEvent.click(screen.getByRole('button', { name: 'Edit' }));
  fireEvent.click(screen.getByRole('button', { name: 'Format JSON' }));

  expect(screen.getByLabelText('Value').value).toBe('{\n  "lat": 37.3\n}');
});

test('Format JSON is not offered for plain text', async () => {
  api.getEntry.mockResolvedValue(makeEntry({ value_text: 'San Jose' }));
  renderDetail();

  await screen.findByText('San Jose');
  fireEvent.click(screen.getByRole('button', { name: 'Edit' }));

  expect(screen.queryByRole('button', { name: 'Format JSON' })).not.toBeInTheDocument();
});

test('the value box grows to fit multi-line content', async () => {
  const long = '{\n' + Array.from({ length: 9 }, (_, i) => `  "k${i}": ${i}`).join(',\n') + '\n}';
  api.getEntry.mockResolvedValue(makeEntry({ key: 'big', value_text: long }));
  renderDetail();

  await screen.findByText('big');
  fireEvent.click(screen.getByRole('button', { name: 'Edit' }));

  const textarea = screen.getByLabelText('Value');
  expect(Number(textarea.getAttribute('rows'))).toBe(long.split('\n').length);
});

test('the value box stops growing for very large values', async () => {
  const huge = Array.from({ length: 200 }, (_, i) => `line ${i}`).join('\n');
  api.getEntry.mockResolvedValue(makeEntry({ key: 'huge', value_text: huge }));
  renderDetail();

  await screen.findByText('huge');
  fireEvent.click(screen.getByRole('button', { name: 'Edit' }));

  expect(Number(screen.getByLabelText('Value').getAttribute('rows'))).toBe(20);
});
