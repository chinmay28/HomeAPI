import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import Settings from './Settings';
import { resetThemeForTests } from '../theme';

jest.mock('../api', () => ({
  ...jest.requireActual('../api'),
  exportData: jest.fn(),
  importData: jest.fn(),
}));

/** Stand in for the OS setting, which jsdom has no notion of. */
function mockSystemDark(dark) {
  window.matchMedia = jest.fn().mockImplementation(query => ({
    matches: query === '(prefers-color-scheme: dark)' ? dark : false,
    media: query,
    addEventListener: jest.fn(),
    removeEventListener: jest.fn(),
    addListener: jest.fn(),
    removeListener: jest.fn(),
    dispatchEvent: jest.fn(),
  }));
}

beforeEach(() => {
  window.localStorage.clear();
  document.documentElement.removeAttribute('data-theme');
  resetThemeForTests();
  mockSystemDark(false);
});

const theme = () => document.documentElement.getAttribute('data-theme');

test('defaults to following the system', async () => {
  render(<Settings />);

  expect(screen.getByRole('button', { name: 'System' })).toHaveAttribute('aria-pressed', 'true');
  expect(screen.getByText(/Following your system/)).toBeInTheDocument();
});

test('reports the resolved theme while following a dark system', () => {
  mockSystemDark(true);
  render(<Settings />);

  expect(screen.getByText(/currently dark/)).toBeInTheDocument();
});

test('pinning dark applies it and persists the choice', async () => {
  render(<Settings />);

  await userEvent.click(screen.getByRole('button', { name: 'Dark' }));

  expect(theme()).toBe('dark');
  expect(window.localStorage.getItem('homeapi.theme')).toBe('dark');
  expect(screen.getByRole('button', { name: 'Dark' })).toHaveAttribute('aria-pressed', 'true');
  expect(screen.getByText(/Pinned to dark/)).toBeInTheDocument();
});

test('pinning light overrides a dark system', async () => {
  mockSystemDark(true);
  render(<Settings />);

  await userEvent.click(screen.getByRole('button', { name: 'Light' }));

  expect(theme()).toBe('light');
});

test('going back to system re-adopts the system theme', async () => {
  mockSystemDark(true);
  render(<Settings />);

  await userEvent.click(screen.getByRole('button', { name: 'Light' }));
  expect(theme()).toBe('light');

  await userEvent.click(screen.getByRole('button', { name: 'System' }));
  expect(theme()).toBe('dark');
  expect(window.localStorage.getItem('homeapi.theme')).toBe('system');
});

test('a saved preference is read back on the next visit', () => {
  window.localStorage.setItem('homeapi.theme', 'dark');
  resetThemeForTests();
  render(<Settings />);

  expect(screen.getByRole('button', { name: 'Dark' })).toHaveAttribute('aria-pressed', 'true');
});

test('the browser chrome colour tracks the theme', async () => {
  render(<Settings />);

  await userEvent.click(screen.getByRole('button', { name: 'Dark' }));

  const meta = document.querySelector('meta[name="theme-color"]:not([media])');
  expect(meta.getAttribute('content')).toBe('#18181b');
});
