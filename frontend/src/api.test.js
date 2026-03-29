import { displayValue } from './api';

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
