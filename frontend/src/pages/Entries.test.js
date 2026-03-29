import React from 'react';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import Entries from './Entries';
import * as api from '../api';

jest.mock('../api', () => ({
  ...jest.requireActual('../api'),
  listEntries: jest.fn(),
  listCategories: jest.fn(),
  createEntry: jest.fn(),
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

function mockList(entries = []) {
  api.listEntries.mockResolvedValue({
    entries,
    total: entries.length,
    page: 1,
    per_page: 50,
    total_pages: entries.length > 0 ? 1 : 0,
  });
  api.listCategories.mockResolvedValue([]);
}

function renderEntries() {
  return render(
    <MemoryRouter initialEntries={['/entries']}>
      <Entries />
    </MemoryRouter>
  );
}

// ── Value rendering ─────────────────────────────────────────────────────────

test('renders plain-text value from {data} envelope — does not show [object Object]', async () => {
  mockList([makeEntry({ value: { data: 'San Jose' } })]);
  renderEntries();

  expect(await screen.findByText('San Jose')).toBeInTheDocument();
  expect(screen.queryByText('[object Object]')).not.toBeInTheDocument();
});

test('renders JSON object value as JSON string — does not crash', async () => {
  mockList([makeEntry({ key: 'location', value: { lat: 37.3, lon: -121.9 } })]);
  renderEntries();

  expect(await screen.findByText('{"lat":37.3,"lon":-121.9}')).toBeInTheDocument();
  expect(screen.queryByText('[object Object]')).not.toBeInTheDocument();
});

test('renders JSON array value as JSON string', async () => {
  mockList([makeEntry({ key: 'tags', value: ['a', 'b', 'c'] })]);
  renderEntries();

  expect(await screen.findByText('["a","b","c"]')).toBeInTheDocument();
});

test('renders empty string value without crashing', async () => {
  mockList([makeEntry({ key: 'blank', value: { data: '' } })]);
  renderEntries();

  // Entry row must appear (key is visible); no crash
  expect(await screen.findByText('blank')).toBeInTheDocument();
  expect(screen.queryByText('[object Object]')).not.toBeInTheDocument();
});

test('renders multiple entries with mixed value types', async () => {
  mockList([
    makeEntry({ id: 1, key: 'city', value: { data: 'San Jose' } }),
    makeEntry({ id: 2, key: 'loc', value: { lat: 37.3 } }),
    makeEntry({ id: 3, key: 'tags', value: ['x', 'y'] }),
  ]);
  renderEntries();

  expect(await screen.findByText('San Jose')).toBeInTheDocument();
  expect(screen.getByText('{"lat":37.3}')).toBeInTheDocument();
  expect(screen.getByText('["x","y"]')).toBeInTheDocument();
  expect(screen.queryByText('[object Object]')).not.toBeInTheDocument();
});

// ── Structure ────────────────────────────────────────────────────────────────

test('shows empty state when no entries', async () => {
  mockList([]);
  renderEntries();

  expect(await screen.findByText('No entries found.')).toBeInTheDocument();
});

test('renders key as a link to the entry detail page', async () => {
  mockList([makeEntry({ id: 42, key: 'AAPL', value: { data: 'Apple Inc.' } })]);
  renderEntries();

  const link = await screen.findByRole('link', { name: 'AAPL' });
  expect(link).toHaveAttribute('href', '/entries/42');
});

test('renders category badge', async () => {
  mockList([makeEntry({ category: 'watchlist', value: { data: 'v' } })]);
  renderEntries();

  expect(await screen.findByText('watchlist')).toBeInTheDocument();
});
