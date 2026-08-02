import React from 'react';
import ReactDOM from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import App from './App';
import { startThemeSync } from './theme';
import './index.css';

// Apply the saved theme preference and follow the OS while it stays "system".
// The inline script in index.html has already painted it; this wires the live
// listener so the app changes with the machine without a reload.
startThemeSync();

const root = ReactDOM.createRoot(document.getElementById('root'));
root.render(
  <React.StrictMode>
    <BrowserRouter>
      <App />
    </BrowserRouter>
  </React.StrictMode>
);
