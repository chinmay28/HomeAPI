import React from 'react';
import { Routes, Route, Link, useLocation } from 'react-router-dom';
import Dashboard from './pages/Dashboard';
import Entries from './pages/Entries';
import EntryDetail from './pages/EntryDetail';
import Settings from './pages/Settings';
import './App.css';

function App() {
  const location = useLocation();

  return (
    <div className="app">
      <header className="header">
        <div className="header-content">
          <Link to="/" className="logo">
            <svg className="logo-icon" viewBox="0 0 32 32" width="22" height="22" aria-hidden="true">
              <path d="M16 6 L26 15 L23 15 L23 25 L18.5 25 L18.5 18.5 L13.5 18.5 L13.5 25 L9 25 L9 15 L6 15 Z" fill="#16a34a" />
            </svg>
            <span>HomeAPI</span>
          </Link>
          <nav className="nav">
            <Link to="/" className={location.pathname === '/' ? 'active' : ''}>Dashboard</Link>
            <Link to="/entries" className={location.pathname.startsWith('/entries') ? 'active' : ''}>Entries</Link>
            <Link to="/settings" className={location.pathname === '/settings' ? 'active' : ''}>Settings</Link>
          </nav>
        </div>
      </header>
      <main className="main">
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/entries" element={<Entries />} />
          <Route path="/entries/:id" element={<EntryDetail />} />
          <Route path="/settings" element={<Settings />} />
        </Routes>
      </main>
    </div>
  );
}

export default App;
