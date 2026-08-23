import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Routes, Route, useLocation } from 'react-router-dom';
import Dashboard from './Dashboard';
import * as api from '../api';

jest.mock('../api', () => ({
  ...jest.requireActual('../api'),
  listCategories: jest.fn(),
  healthCheck: jest.fn(),
  listEntries: jest.fn(),
  getEntryByKey: jest.fn(),
}));

// The dashboard only reaches for entries when something is pinned or the
// picker is open; every test still gets a working default so a stray call
// never falls through to a real fetch.
beforeEach(() => {
  api.listEntries.mockResolvedValue({ entries: [], total: 0 });
  api.getEntryByKey.mockResolvedValue(null);
});

function LocationDisplay() {
  const location = useLocation();
  return <div data-testid="location">{location.pathname + location.search}</div>;
}

function renderDashboard() {
  return render(
    <MemoryRouter initialEntries={['/']}>
      <Routes>
        <Route path="*" element={<Dashboard />} />
      </Routes>
      <LocationDisplay />
    </MemoryRouter>
  );
}

test('shows total entry count', async () => {
  api.listCategories.mockResolvedValue([
    { name: 'watchlist', count: 5 },
    { name: 'config', count: 3 },
  ]);
  api.healthCheck.mockResolvedValue({ status: 'ok', version: '1.0.0' });
  renderDashboard();

  expect(await screen.findByText('8')).toBeInTheDocument(); // total entries
});

test('shows Healthy status when API is ok', async () => {
  api.listCategories.mockResolvedValue([]);
  api.healthCheck.mockResolvedValue({ status: 'ok', version: '1.0.0' });
  renderDashboard();

  expect(await screen.findByText('Healthy')).toBeInTheDocument();
});

test('shows category names as clickable links to filtered entries', async () => {
  api.listCategories.mockResolvedValue([
    { name: 'watchlist', count: 4 },
  ]);
  api.healthCheck.mockResolvedValue({ status: 'ok', version: '1.0.0' });
  renderDashboard();

  expect(await screen.findByRole('link', { name: 'watchlist' })).toHaveAttribute(
    'href', '/entries?category=watchlist'
  );
});

test('clicking a category row navigates to its filtered entries', async () => {
  api.listCategories.mockResolvedValue([
    { name: 'watchlist', count: 4 },
  ]);
  api.healthCheck.mockResolvedValue({ status: 'ok', version: '1.0.0' });
  renderDashboard();

  const categoryLink = await screen.findByRole('link', { name: 'watchlist' });
  await userEvent.click(categoryLink.closest('tr'));

  expect(screen.getByTestId('location')).toHaveTextContent(
    '/entries?category=watchlist'
  );
});

test('shows empty state when no entries exist', async () => {
  api.listCategories.mockResolvedValue([]);
  api.healthCheck.mockResolvedValue({ status: 'ok', version: '1.0.0' });
  renderDashboard();

  expect(await screen.findByText('No entries yet.')).toBeInTheDocument();
});

// ── Featured stats ───────────────────────────────────────────────────────────

// The labels of the cards currently on the dashboard. Scoped to .stat-card
// because the customize panel repeats every label as a checkbox, and the
// "Categories" card shares its name with the category table's heading.
function statLabels() {
  return Array.from(document.querySelectorAll('.stat-card__label')).map(el => el.textContent);
}

function statCard(label) {
  return Array.from(document.querySelectorAll('.stat-card'))
    .find(card => card.querySelector('.stat-card__label')?.textContent === label);
}

describe('featured stats', () => {
  beforeEach(() => {
    window.localStorage.clear();
    api.healthCheck.mockResolvedValue({ status: 'ok', version: 'v2026.8.17' });
  });

  test('shows the default three cards on a fresh browser', async () => {
    api.listCategories.mockResolvedValue([{ name: 'watchlist', count: 4 }]);
    renderDashboard();

    await screen.findByRole('button', { name: 'Customize stats' });
    expect(statLabels()).toEqual(['Total entries', 'Categories', 'Server']);
  });

  test('featuring a category adds its own card and persists the choice', async () => {
    api.listCategories.mockResolvedValue([
      { name: 'watchlist', count: 4 },
      { name: 'notes', count: 2 },
    ]);
    renderDashboard();

    await userEvent.click(await screen.findByRole('button', { name: 'Customize stats' }));
    await userEvent.click(screen.getByRole('checkbox', { name: 'notes (2)' }));

    expect(statLabels()).toEqual(['Total entries', 'Categories', 'Server', 'notes']);
    expect(statCard('notes').textContent).toContain('2');

    expect(JSON.parse(window.localStorage.getItem('homeapi.featuredStats')))
      .toEqual(['total', 'categories', 'server', 'category:notes']);
  });

  test('restores the saved selection on the next visit', async () => {
    window.localStorage.setItem('homeapi.featuredStats', JSON.stringify(['largest']));
    api.listCategories.mockResolvedValue([
      { name: 'watchlist', count: 9 },
      { name: 'notes', count: 2 },
    ]);
    renderDashboard();

    await screen.findByRole('button', { name: 'Customize stats' });
    expect(statLabels()).toEqual(['Largest category']);
    // Largest wins on count, not on ordering.
    expect(statCard('Largest category').textContent).toContain('watchlist');
  });

  test('unticking a stat removes its card', async () => {
    api.listCategories.mockResolvedValue([{ name: 'watchlist', count: 4 }]);
    renderDashboard();

    await userEvent.click(await screen.findByRole('button', { name: 'Customize stats' }));
    await userEvent.click(screen.getByRole('checkbox', { name: 'Total entries' }));

    expect(statLabels()).toEqual(['Categories', 'Server']);
  });

  test('reset restores the defaults', async () => {
    window.localStorage.setItem('homeapi.featuredStats', JSON.stringify(['largest']));
    api.listCategories.mockResolvedValue([{ name: 'watchlist', count: 4 }]);
    renderDashboard();

    await userEvent.click(await screen.findByRole('button', { name: 'Customize stats' }));
    await userEvent.click(screen.getByRole('button', { name: 'Reset to default' }));

    expect(statLabels()).toEqual(['Total entries', 'Categories', 'Server']);
    expect(window.localStorage.getItem('homeapi.featuredStats')).toBeNull();
  });

  test('a stat for a category that no longer exists is skipped, not crashed on', async () => {
    window.localStorage.setItem(
      'homeapi.featuredStats',
      JSON.stringify(['total', 'category:deleted']),
    );
    api.listCategories.mockResolvedValue([{ name: 'watchlist', count: 4 }]);
    renderDashboard();

    await screen.findByRole('button', { name: 'Customize stats' });
    expect(statLabels()).toEqual(['Total entries']);
  });

  test('a corrupt preference falls back to the defaults', async () => {
    window.localStorage.setItem('homeapi.featuredStats', 'not json');
    api.listCategories.mockResolvedValue([{ name: 'watchlist', count: 4 }]);
    renderDashboard();

    await screen.findByRole('button', { name: 'Customize stats' });
    expect(statLabels()).toEqual(['Total entries', 'Categories', 'Server']);
  });
});

// ── Pinned entry values ──────────────────────────────────────────────────────

describe('featured entry values', () => {
  const minionSum = {
    id: 7,
    category: 'minion',
    key: 'minion-sum',
    value: { data: '42' },
    value_text: '42',
  };

  beforeEach(() => {
    window.localStorage.clear();
    api.healthCheck.mockResolvedValue({ status: 'ok', version: 'v2026.8.17' });
    api.listCategories.mockResolvedValue([{ name: 'minion', count: 3 }]);
  });

  test('pinning an entry shows its value as a card and persists the choice', async () => {
    api.listEntries.mockResolvedValue({ entries: [minionSum], total: 1 });
    api.getEntryByKey.mockResolvedValue(minionSum);
    renderDashboard();

    await userEvent.click(await screen.findByRole('button', { name: 'Customize stats' }));
    await userEvent.click(await screen.findByRole('checkbox', { name: 'minion-sum' }));

    const card = await screen.findByRole('link', { name: /minion-sum/ });
    expect(card).toHaveAttribute('href', '/entries/7');
    expect(card.textContent).toContain('42');

    expect(JSON.parse(window.localStorage.getItem('homeapi.featuredStats')))
      .toEqual(['total', 'categories', 'server', 'entry:minion-sum']);
  });

  test('a saved entry stat is restored and fetched by key', async () => {
    window.localStorage.setItem(
      'homeapi.featuredStats',
      JSON.stringify(['entry:minion-sum']),
    );
    api.getEntryByKey.mockResolvedValue(minionSum);
    renderDashboard();

    expect(await screen.findByText('42')).toBeInTheDocument();
    expect(api.getEntryByKey).toHaveBeenCalledWith('minion-sum');
    expect(statCard('minion-sum').textContent).toContain('minion');
  });

  test('a long value is shown as text rather than at headline size', async () => {
    window.localStorage.setItem('homeapi.featuredStats', JSON.stringify(['entry:note']));
    const note = {
      id: 9,
      category: 'notes',
      key: 'note',
      value: { data: 'the quick brown fox jumps over the lazy dog' },
      value_text: 'the quick brown fox jumps over the lazy dog',
    };
    api.getEntryByKey.mockResolvedValue(note);
    renderDashboard();

    const value = await screen.findByText(note.value_text);
    expect(value).toHaveClass('stat-card__value--text');
  });

  test('a pinned entry that no longer exists is skipped, not crashed on', async () => {
    window.localStorage.setItem(
      'homeapi.featuredStats',
      JSON.stringify(['total', 'entry:gone']),
    );
    api.getEntryByKey.mockResolvedValue(null);
    renderDashboard();

    await screen.findByRole('button', { name: 'Customize stats' });
    await waitFor(() => expect(statLabels()).toEqual(['Total entries']));
  });

  test('a pinned entry missing from the picker list is still untickable', async () => {
    window.localStorage.setItem('homeapi.featuredStats', JSON.stringify(['entry:minion-sum']));
    api.getEntryByKey.mockResolvedValue(minionSum);
    // The picker's page doesn't happen to include the pinned entry.
    api.listEntries.mockResolvedValue({ entries: [], total: 0 });
    renderDashboard();

    await userEvent.click(await screen.findByRole('button', { name: 'Customize stats' }));
    const checkbox = await screen.findByRole('checkbox', { name: 'minion-sum' });
    expect(checkbox).toBeChecked();

    await userEvent.click(checkbox);
    expect(statLabels()).toEqual([]);
  });

  test('searching the picker asks the server rather than filtering one page', async () => {
    api.listEntries.mockResolvedValue({ entries: [minionSum], total: 40 });
    renderDashboard();

    await userEvent.click(await screen.findByRole('button', { name: 'Customize stats' }));
    await userEvent.type(
      await screen.findByRole('searchbox', { name: 'Search entries to feature' }),
      'minion',
    );

    await waitFor(() => expect(api.listEntries).toHaveBeenCalledWith(
      expect.objectContaining({ search: 'minion' }),
    ));
  });
});
