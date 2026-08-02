import { useSyncExternalStore } from 'react';

/**
 * Theme preference: "system" (the default) follows the OS, "light" and "dark"
 * pin it. The resolved value — what is actually on screen — is stamped onto
 * <html data-theme> which is what index.css keys off.
 *
 * The preference is a single module-level store rather than component state:
 * the control lives on the Settings page but the header, tab bar and every
 * other surface have to change with it, and a stale second copy of the value
 * would quietly clobber the user's choice the next time the OS scheme flipped.
 */

export const THEMES = ['system', 'light', 'dark'];

export const THEME_STORAGE_KEY = 'homeapi.theme';

/** Background colour per resolved theme, mirroring --bg in index.css. Used for
 * the theme-color meta tag, which tints the mobile browser's own chrome. */
const CHROME_COLOR = { light: '#fafafa', dark: '#18181b' };

function darkQuery() {
  if (typeof window === 'undefined' || !window.matchMedia) return null;
  return window.matchMedia('(prefers-color-scheme: dark)');
}

/** The stored preference, defaulting to "system". Storage can throw (Safari in
 * private mode, cookies blocked) — a broken preference must not break the app. */
export function readTheme() {
  try {
    const saved = window.localStorage.getItem(THEME_STORAGE_KEY);
    return THEMES.includes(saved) ? saved : 'system';
  } catch {
    return 'system';
  }
}

/** Turn a preference into the theme actually being shown. */
export function resolveTheme(theme) {
  if (theme === 'light' || theme === 'dark') return theme;
  return darkQuery()?.matches ? 'dark' : 'light';
}

/** Stamp the resolved theme onto <html> and match the browser chrome to it. */
function paint(resolved) {
  document.documentElement.setAttribute('data-theme', resolved);
  // The static per-scheme meta tags in index.html follow the OS, which is wrong
  // the moment someone pins a theme against it — one managed tag wins.
  let meta = document.querySelector('meta[name="theme-color"]:not([media])');
  if (!meta) {
    meta = document.createElement('meta');
    meta.setAttribute('name', 'theme-color');
    document.head.appendChild(meta);
  }
  meta.setAttribute('content', CHROME_COLOR[resolved]);
}

let pref = null; // read lazily so importing this module touches nothing
let resolved = null;
const listeners = new Set();

function ensureLoaded() {
  if (pref === null) {
    pref = readTheme();
    resolved = resolveTheme(pref);
  }
}

function getPref() {
  ensureLoaded();
  return pref;
}

function getResolved() {
  ensureLoaded();
  return resolved;
}

function subscribe(listener) {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

function notify() {
  listeners.forEach((listener) => listener());
}

/** Set and persist the preference, repainting immediately. */
export function setTheme(next) {
  if (!THEMES.includes(next)) return;
  ensureLoaded();
  try {
    window.localStorage.setItem(THEME_STORAGE_KEY, next);
  } catch {
    // Preference is lost on reload; the current page still honours it.
  }
  pref = next;
  resolved = resolveTheme(next);
  paint(resolved);
  notify();
}

/**
 * Apply the saved preference and keep it in step with the OS. Called once at
 * startup (src/index.js); the inline script in index.html has already painted
 * the same value, so this is about wiring the listener, not about first paint.
 */
export function startThemeSync() {
  ensureLoaded();
  paint(resolved);

  const query = darkQuery();
  if (!query) return () => {};
  const onChange = () => {
    // A pinned theme ignores the OS; only "system" tracks it.
    if (pref !== 'system') return;
    resolved = resolveTheme('system');
    paint(resolved);
    notify();
  };
  // addEventListener is unsupported on MediaQueryList in older Safari.
  if (query.addEventListener) {
    query.addEventListener('change', onChange);
    return () => query.removeEventListener('change', onChange);
  }
  query.addListener(onChange);
  return () => query.removeListener(onChange);
}

/** The theme preference, a setter, and the theme actually on screen. */
export function useTheme() {
  const theme = useSyncExternalStore(subscribe, getPref, getPref);
  const active = useSyncExternalStore(subscribe, getResolved, getResolved);
  return { theme, setTheme, resolved: active };
}

/** Test seam: drop the cached preference so the next read hits storage again. */
export function resetThemeForTests() {
  pref = null;
  resolved = null;
  listeners.clear();
}
