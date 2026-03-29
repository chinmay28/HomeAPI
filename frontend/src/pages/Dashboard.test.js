import React from 'react';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import Dashboard from './Dashboard';
import * as api from '../api';

jest.mock('../api', () => ({
  ...jest.requireActual('../api'),
  listCategories: jest.fn(),
  healthCheck: jest.fn(),
}));

function renderDashboard() {
  return render(
    <MemoryRouter>
      <Dashboard />
    </MemoryRouter>
  );
}

test('shows total entry count and category count', async () => {
  api.listCategories.mockResolvedValue([
    { name: 'watchlist', count: 5 },
    { name: 'config', count: 3 },
  ]);
  api.healthCheck.mockResolvedValue({ status: 'ok', version: '1.0.0' });
  renderDashboard();

  expect(await screen.findByText('8')).toBeInTheDocument(); // total entries
  expect(screen.getByText('2')).toBeInTheDocument();        // category count
});

test('shows Healthy status when API is ok', async () => {
  api.listCategories.mockResolvedValue([]);
  api.healthCheck.mockResolvedValue({ status: 'ok', version: '1.0.0' });
  renderDashboard();

  expect(await screen.findByText('Healthy')).toBeInTheDocument();
});

test('shows category names with View links', async () => {
  api.listCategories.mockResolvedValue([
    { name: 'watchlist', count: 4 },
  ]);
  api.healthCheck.mockResolvedValue({ status: 'ok', version: '1.0.0' });
  renderDashboard();

  expect(await screen.findByText('watchlist')).toBeInTheDocument();
  expect(screen.getByRole('link', { name: 'View' })).toHaveAttribute(
    'href', '/entries?category=watchlist'
  );
});

test('shows empty state when no entries exist', async () => {
  api.listCategories.mockResolvedValue([]);
  api.healthCheck.mockResolvedValue({ status: 'ok', version: '1.0.0' });
  renderDashboard();

  expect(await screen.findByText('No entries yet.')).toBeInTheDocument();
});
