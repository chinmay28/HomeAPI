import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import EntryDetail from './EntryDetail';
import * as api from '../api';

jest.mock('../api', () => ({
  ...jest.requireActual('../api'),
  getEntry: jest.fn(),
  updateEntry: jest.fn(),
  deleteEntry: jest.fn(),
}));

function makeEntry(overrides) {
  return {
    id: 1,
    category: 'default',
    key: 'city',
    value: { data: 'San Jose' },
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
    ...overrides,
  };
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

test('displays JSON object value as JSON string', async () => {
  api.getEntry.mockResolvedValue(makeEntry({ key: 'loc', value: { lat: 37.3, lon: -121.9 } }));
  renderDetail();

  expect(await screen.findByText('{"lat":37.3,"lon":-121.9}')).toBeInTheDocument();
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
  api.getEntry.mockResolvedValue(makeEntry({ value: { lat: 37.3 } }));
  renderDetail();

  await screen.findByText('{"lat":37.3}');
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
