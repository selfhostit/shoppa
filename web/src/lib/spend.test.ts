import { describe, expect, it } from 'vitest';
import { money, monthName, parsePrice } from './spend';

describe('spend', () => {
  it('reads prices typed either way', () => {
    expect(parsePrice('3,20')).toBe(3.2);
    expect(parsePrice(' € 1.555 ')).toBe(1.56);
    expect(parsePrice('')).toBe(0);
    expect(parsePrice('abc')).toBe(0);
    expect(parsePrice('-2')).toBe(0);
  });

  it('writes money and months in your language', () => {
    expect(money(12.5, 'en')).toBe('€12.50');
    expect(money(12.5, 'nl')).toMatch(/12,50/);
    expect(monthName('2026-11', 'en')).toBe('November 2026');
  });
});
