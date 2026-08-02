import {
  displayValue,
  formatJSONText,
  getEntryByKey,
  isJSONText,
  previewValue,
  readableValue,
  valueText,
} from './api';

describe('displayValue', () => {
  test('unwraps plain text from {data: "..."} envelope', () => {
    expect(displayValue({ data: 'San Jose' })).toBe('San Jose');
  });

  test('unwraps empty string from {data: ""} envelope', () => {
    expect(displayValue({ data: '' })).toBe('');
  });

  test('unwraps numeric string from envelope', () => {
    expect(displayValue({ data: '72' })).toBe('72');
  });

  test('JSON.stringifies objects that are not the data envelope', () => {
    expect(displayValue({ lat: 37.3, lon: -121.9 })).toBe('{"lat":37.3,"lon":-121.9}');
  });

  test('JSON.stringifies arrays', () => {
    expect(displayValue(['a', 'b', 'c'])).toBe('["a","b","c"]');
  });

  test('does not treat multi-key objects as the data envelope', () => {
    const v = { data: 'x', other: 'y' };
    expect(displayValue(v)).toBe(JSON.stringify(v));
  });

  test('returns empty string for null', () => {
    expect(displayValue(null)).toBe('');
  });

  test('returns empty string for undefined', () => {
    expect(displayValue(undefined)).toBe('');
  });

  test('handles plain string defensively (pre-API-change responses)', () => {
    expect(displayValue('hello')).toBe('hello');
  });
});

describe('isJSONText', () => {
  test('recognises objects and arrays', () => {
    expect(isJSONText('{"a":1}')).toBe(true);
    expect(isJSONText('[1,2]')).toBe(true);
  });

  test('recognises hand-formatted JSON', () => {
    expect(isJSONText('{\n  "a": 1\n}')).toBe(true);
  });

  test('rejects plain text, scalars and broken JSON', () => {
    expect(isJSONText('San Jose')).toBe(false);
    expect(isJSONText('42')).toBe(false); // a bare number is stored as text
    expect(isJSONText('{"a":')).toBe(false);
    expect(isJSONText('')).toBe(false);
    expect(isJSONText(undefined)).toBe(false);
  });
});

describe('formatJSONText', () => {
  test('pretty-prints with two-space indents', () => {
    expect(formatJSONText('{"a":1}')).toBe('{\n  "a": 1\n}');
  });

  test('returns null for text that is not JSON', () => {
    expect(formatJSONText('San Jose')).toBeNull();
  });
});

describe('valueText', () => {
  test('returns the stored text byte-for-byte', () => {
    const stored = '{\n    "lat": 37.3\n}';
    expect(valueText({ value: { lat: 37.3 }, value_text: stored })).toBe(stored);
  });

  test('preserves plain text unchanged', () => {
    expect(valueText({ value: { data: 'San Jose' }, value_text: 'San Jose' })).toBe('San Jose');
  });

  test('falls back to the parsed value when the server sends no value_text', () => {
    expect(valueText({ value: { data: 'San Jose' } })).toBe('San Jose');
    expect(valueText({ value: { lat: 37.3 } })).toBe('{\n  "lat": 37.3\n}');
  });
});

describe('readableValue', () => {
  test('shows already-formatted JSON exactly as it was written', () => {
    const stored = '{\n    "lat": 37.3,\n    "lon": -121.9\n}';
    expect(readableValue({ value: {}, value_text: stored })).toBe(stored);
  });

  test('pretty-prints JSON that arrived on a single line', () => {
    expect(readableValue({ value: {}, value_text: '{"lat":37.3}' })).toBe('{\n  "lat": 37.3\n}');
  });

  test('leaves plain text alone, newlines included', () => {
    expect(readableValue({ value: {}, value_text: 'milk\neggs' })).toBe('milk\neggs');
  });
});

describe('previewValue', () => {
  test('collapses a multi-line value onto one line for table cells', () => {
    expect(previewValue({ value: {}, value_text: '{\n  "a": 1\n}' })).toBe('{ "a": 1 }');
  });

  test('falls back to the parsed value without value_text', () => {
    expect(previewValue({ value: { data: 'San Jose' } })).toBe('San Jose');
  });
});

describe('getEntryByKey', () => {
  const entry = (over) => ({ id: 1, category: 'default', key: 'city', value: { data: 'x' }, ...over });

  // A response body keyed by the path the request asked for.
  function mockFetch(routes) {
    global.fetch = jest.fn((url) => {
      const body = routes[url];
      if (body === undefined) {
        return Promise.resolve({
          ok: false, status: 404, json: () => Promise.resolve({ error: 'Entry not found' }),
        });
      }
      return Promise.resolve({ ok: true, status: 200, json: () => Promise.resolve(body) });
    });
  }

  afterEach(() => { delete global.fetch; });

  test('returns the entry the path lookup resolves', async () => {
    mockFetch({ '/api/entries/city': entry() });
    await expect(getEntryByKey('city')).resolves.toEqual(entry());
  });

  test('searches when a numeric key resolves some other entry by id', async () => {
    // /api/entries/42 is the entry with *id* 42, not the one keyed "42".
    mockFetch({
      '/api/entries/42': entry({ id: 42, key: 'city' }),
      '/api/entries?search=42&per_page=200': {
        entries: [entry({ id: 7, key: '42' }), entry({ id: 8, key: '420' })],
        total: 2,
      },
    });
    await expect(getEntryByKey('42')).resolves.toEqual(entry({ id: 7, key: '42' }));
  });

  test('searches when the key is not addressable as a path', async () => {
    mockFetch({
      '/api/entries?search=a%2Fb&per_page=200': { entries: [entry({ id: 3, key: 'a/b' })], total: 1 },
    });
    await expect(getEntryByKey('a/b')).resolves.toEqual(entry({ id: 3, key: 'a/b' }));
  });

  test('resolves to null when no entry has that key', async () => {
    mockFetch({ '/api/entries?search=gone&per_page=200': { entries: [], total: 0 } });
    await expect(getEntryByKey('gone')).resolves.toBeNull();
  });
});
