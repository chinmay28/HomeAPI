import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import MinionCard, { MINION_FRAME_SRC, parseLastSeen } from './MinionCard';
import * as api from '../api';

jest.mock('../api', () => ({
  ...jest.requireActual('../api'),
  getEntryByKey: jest.fn(),
}));

function mockEntries(byKey) {
  api.getEntryByKey.mockImplementation(key => Promise.resolve(byKey[key] || null));
}

function renderCard() {
  return render(
    <MemoryRouter>
      <MinionCard />
    </MemoryRouter>
  );
}

test('stays hidden when nothing Minion-shaped exists', async () => {
  mockEntries({});
  const { container } = renderCard();
  await waitFor(() => expect(api.getEntryByKey).toHaveBeenCalledTimes(3));
  expect(container).toBeEmptyDOMElement();
  expect(screen.queryByText('Minion 3')).not.toBeInTheDocument();
});

test('shows the frame, the telemetry and a link to the params entry', async () => {
  mockEntries({
    'minion-params': { id: 7, key: 'minion-params', value_text: '{}' },
    'minion3-last-seen': {
      id: 8,
      key: 'minion3-last-seen',
      value_text: '{"wake":183,"at":"2026-11-03T15:15:00Z"}',
    },
  });
  renderCard();

  expect(await screen.findByText('Minion 3')).toBeInTheDocument();
  const img = screen.getByAltText('Minion 3 panel preview');
  expect(img).toHaveAttribute('src', MINION_FRAME_SRC);
  expect(screen.getByText('183')).toBeInTheDocument();
  expect(screen.queryByText('Never')).not.toBeInTheDocument();
  expect(screen.getByRole('link', { name: 'Edit params' })).toHaveAttribute('href', '/entries/7');
  expect(screen.queryByText(/Cfg\?/)).not.toBeInTheDocument();
});

test('with quotes but no params it offers to create them and reports no wake yet', async () => {
  mockEntries({
    'minion-quotes': { id: 3, key: 'minion-quotes', value_text: '{"VTI":"301.20"}' },
  });
  renderCard();

  expect(await screen.findByText('Minion 3')).toBeInTheDocument();
  expect(screen.getByText('Never')).toBeInTheDocument();
  expect(screen.getByText('—')).toBeInTheDocument();
  expect(screen.getByRole('link', { name: 'Create params' })).toHaveAttribute('href', '/entries?new=1');
  expect(screen.getByText(/Cfg\?/)).toBeInTheDocument();
});

test('survives a request failure as if the entry were absent', async () => {
  api.getEntryByKey.mockImplementation(key => (
    key === 'minion-params'
      ? Promise.resolve({ id: 1, key, value_text: '{}' })
      : Promise.reject(new Error('boom'))
  ));
  renderCard();
  expect(await screen.findByText('Minion 3')).toBeInTheDocument();
  expect(screen.getByText('Never')).toBeInTheDocument();
});

describe('parseLastSeen', () => {
  test('reads what the server writes', () => {
    const seen = parseLastSeen({ value_text: '{"wake":5,"at":"2026-11-03T15:15:00Z"}' });
    expect(seen.wake).toBe(5);
    expect(seen.at.toISOString()).toBe('2026-11-03T15:15:00.000Z');
  });

  test('falls back to the parsed value for an older server', () => {
    const seen = parseLastSeen({ value: { wake: 2, at: 'not a date' } });
    expect(seen).toEqual({ wake: 2, at: null });
  });

  test('rejects anything else', () => {
    expect(parseLastSeen(null)).toBeNull();
    expect(parseLastSeen({ value_text: 'plain text' })).toBeNull();
    expect(parseLastSeen({ value_text: '{"at":"2026-11-03T15:15:00Z"}' })).toBeNull();
  });
});
