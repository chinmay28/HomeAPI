import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Routes, Route, useLocation } from 'react-router-dom';
import Dashboard from './Dashboard';
import * as api from '../api';

jest.mock('../api', () => ({
  ...jest.requireActual('../api'),
  listCategories: jest.fn(),
  healthCheck: jest.fn(),
}));

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
    api.healthCheck.mockResolvedValue({ status: 'ok', version: 'v1.0.17' });
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
