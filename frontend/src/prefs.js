/**
 * Per-device UI preferences, kept in localStorage.
 *
 * Deliberately not stored through the API: these are view settings for one
 * browser, not data. Putting them in the entries table would leak them into
 * `/api/entries`, the category list, and every export — a config row nobody
 * asked for showing up in someone's watchlist.
 */

const FEATURED_STATS_KEY = 'homeapi.featuredStats';

/** The stats offered on the dashboard that don't depend on the user's data. */
export const CORE_STATS = [
  { id: 'total', label: 'Total entries' },
  { id: 'categories', label: 'Categories' },
  { id: 'server', label: 'Server status' },
  { id: 'largest', label: 'Largest category' },
];

/** What a fresh install shows: the three cards the dashboard has always had. */
export const DEFAULT_FEATURED_STATS = ['total', 'categories', 'server'];

/** A per-category count card is identified as `category:<name>`. */
export function categoryStatId(name) {
  return `category:${name}`;
}

/** The category a `category:<name>` id refers to, or null for a core stat. */
export function statCategory(id) {
  return id.startsWith('category:') ? id.slice('category:'.length) : null;
}

function read(key, fallback) {
  try {
    const raw = window.localStorage.getItem(key);
    if (raw === null) return fallback;
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed) ? parsed : fallback;
  } catch {
    // Unreadable or corrupt (hand-edited, quota-cleared, private mode) — the
    // default is always a working answer.
    return fallback;
  }
}

function write(key, value) {
  try {
    window.localStorage.setItem(key, JSON.stringify(value));
  } catch {
    // Preference is lost on reload; the current page still honours it.
  }
}

/**
 * The stat ids featured on the dashboard, in display order.
 *
 * Ids of categories that have since been deleted are kept, not pruned: a
 * category comes back the moment an entry is filed under it again, and
 * silently forgetting the choice would be worse than a card that waits.
 * The dashboard skips ids it can't render.
 */
export function readFeaturedStats() {
  const saved = read(FEATURED_STATS_KEY, null);
  if (!saved) return DEFAULT_FEATURED_STATS;
  return saved.filter((id) => typeof id === 'string');
}

export function writeFeaturedStats(ids) {
  write(FEATURED_STATS_KEY, ids);
}

/** Forget the choice, so the dashboard falls back to the defaults. */
export function clearFeaturedStats() {
  try {
    window.localStorage.removeItem(FEATURED_STATS_KEY);
  } catch {
    // Nothing to undo.
  }
}
