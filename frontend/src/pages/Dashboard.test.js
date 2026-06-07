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
