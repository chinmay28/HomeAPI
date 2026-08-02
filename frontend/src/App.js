import React, { useEffect, useState } from 'react';
import { Routes, Route, Link, NavLink, useLocation } from 'react-router-dom';
import Dashboard from './pages/Dashboard';
import Entries from './pages/Entries';
import EntryDetail from './pages/EntryDetail';
import Settings from './pages/Settings';
import useKeyboardOpen from './useKeyboardOpen';
import { APP_VERSION } from './version';
import './App.css';

/** Primary destinations, shown in the desktop header and the mobile tab bar. */
const NAV_ITEMS = [
  { to: '/', label: 'Dashboard', icon: <HomeIcon /> },
  { to: '/entries', label: 'Entries', icon: <ListIcon /> },
  { to: '/settings', label: 'Settings', icon: <GearIcon /> },
];

/** How long the developer badge stays on screen when the header mark is
 * tapped. Kept in sync with the `dev-flash*` animation durations in
 * App.css — the CSS fades out on its own clock, this unmounts it. */
const DEV_FLASH_MS = 3000;

function App() {
  const location = useLocation();
  // While the on-screen keyboard is up, drop the bottom chrome so it never
  // floats over the keyboard; it comes back the moment the keyboard closes.
  const keyboardOpen = useKeyboardOpen();
  // The FAB *is* the "new entry" action — it opens the create form on the
  // entries list by way of ?new=1 — so it steps aside once that form is up.
  const formOpen = location.pathname === '/entries'
    && new URLSearchParams(location.search).get('new') === '1';
  const showFab = !formOpen;
  // Tapping the developer mark throws the badge up full screen for a beat.
  const [devFlash, setDevFlash] = useState(false);

  useEffect(() => {
    if (!devFlash) return undefined;
    const timer = window.setTimeout(() => setDevFlash(false), DEV_FLASH_MS);
    // Nobody should be stuck waiting out an animation — Escape ends it early,
    // as does a tap anywhere on the overlay.
    const onKey = (event) => {
      if (event.key === 'Escape') setDevFlash(false);
    };
    window.addEventListener('keydown', onKey);
    return () => {
      window.clearTimeout(timer);
      window.removeEventListener('keydown', onKey);
    };
  }, [devFlash]);

  return (
    <div className={`app${keyboardOpen ? ' app--keyboard-open' : ''}`}>
      <header className="app__header">
        <Link to="/" className="app__brand">
          <img className="app__brand-logo" src="/icon.svg" alt="" aria-hidden="true" />
          {/* Name over version, as a lockup. */}
          <span className="app__brand-text">
            HomeAPI
            <span className="app__brand-version">{APP_VERSION}</span>
          </span>
        </Link>
        {/* Everything that hangs off the right edge. Grouping the nav with the
            developer mark keeps them together when the nav collapses on
            mobile — the mark then sits alone opposite the brand. */}
        <div className="app__header-end">
          {/* Desktop / wide-screen navigation. The mobile tab bar mirrors it. */}
          <nav className="app__nav" aria-label="Primary">
            {NAV_ITEMS.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                end={item.to === '/'}
                className={({ isActive }) => `btn btn--ghost${isActive ? ' btn--active' : ''}`}
              >
                {item.label}
              </NavLink>
            ))}
            <Link to="/entries?new=1" className="btn btn--primary">
              New entry
            </Link>
          </nav>
          {/* Developer credit. Deliberately quiet — a muted disk that only
              comes to full strength on hover, so it never competes with the
              primary action next to it. Tapping it shows the badge full
              screen, which is the only place its detail is readable. */}
          <button
            type="button"
            className="app__dev"
            title="Built by CM Hegday · 0x434d"
            aria-label="Show the developer badge"
            onClick={() => setDevFlash(true)}
          >
            {/* The button carries the label; the image would only repeat it. */}
            <img className="app__dev-logo" src="/dev-badge.png" alt="" aria-hidden="true" />
          </button>
        </div>
      </header>

      <main className="app__main">
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/entries" element={<Entries />} />
          <Route path="/entries/:id" element={<EntryDetail />} />
          <Route path="/settings" element={<Settings />} />
        </Routes>
      </main>

      <footer className="app__footer">
        <span>Self-hosted key-value store · your data stays on your machine.</span>
      </footer>

      {/* Floating action button — the primary create action on phones. */}
      {showFab && (
        <Link to="/entries?new=1" className="fab" aria-label="New entry" title="New entry">
          <PlusIcon />
        </Link>
      )}

      {/* Mobile bottom tab bar (hidden on wide screens via CSS). */}
      <nav className="tab-bar" aria-label="Primary">
        {NAV_ITEMS.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.to === '/'}
            className={({ isActive }) => `tab-bar__item${isActive ? ' tab-bar__item--active' : ''}`}
          >
            <span className="tab-bar__icon" aria-hidden="true">
              {item.icon}
            </span>
            <span className="tab-bar__label">{item.label}</span>
          </NavLink>
        ))}
      </nav>

      {/* Developer badge, full screen for three seconds. It lives out here
          rather than in the header because the header's backdrop-filter makes
          it a containing block — a fixed overlay inside it would be trapped
          in the header's strip instead of covering the viewport. */}
      {devFlash && (
        <div className="dev-flash" role="presentation" onClick={() => setDevFlash(false)}>
          <div className="dev-flash__lockup">
            <img
              className="dev-flash__logo"
              src="/dev-badge-full.png"
              alt="Built by CM Hegday — 0x434d"
            />
            <span className="dev-flash__handle">github.com/chinmay28</span>
          </div>
        </div>
      )}
    </div>
  );
}

/* Inline, dependency-free icons. They inherit `currentColor` and a 24px box. */

function HomeIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"
      strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M3 10.5 12 3l9 7.5" />
      <path d="M5 9.5V21h14V9.5" />
    </svg>
  );
}

function ListIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"
      strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M4 6h4M4 12h4M4 18h4" />
      <path d="M12 6h8M12 12h8M12 18h5" />
    </svg>
  );
}

function GearIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"
      strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <circle cx="12" cy="12" r="3.25" />
      <path d="M12 2.5v2.2M12 19.3v2.2M4.6 4.6l1.6 1.6M17.8 17.8l1.6 1.6M2.5 12h2.2M19.3 12h2.2M4.6 19.4l1.6-1.6M17.8 6.2l1.6-1.6" />
    </svg>
  );
}

function PlusIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5"
      strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M12 5v14M5 12h14" />
    </svg>
  );
}

export default App;
