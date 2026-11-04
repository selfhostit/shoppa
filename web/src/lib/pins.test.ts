import { describe, expect, it } from 'vitest';
import type { Generic } from './catalogue';
import { pinnedFor, usualFor, type Pin } from './pins';
import type { Product } from './products';

const product = (id: string) => ({ id, name: id }) as Product;
const pin = (generic: string, p: string, uses: number, last = '2026-11-01') =>
  ({
    id: generic + p,
    generic,
    product: p,
    uses,
    last_used: last,
    expand: { product: product(p) },
  }) as Pin;
const generic = (id: string) => ({ id }) as Generic;

const pins = [
  pin('milk', 'boni', 3),
  pin('milk', 'campina', 5),
  pin('milk', 'alpro', 3, '2026-11-03'),
  pin('cheese', 'gouda', 1),
  pin('chocolate-milk', 'campina', 1),
];

describe('pins', () => {
  it('puts the product bought most first, then the latest', () => {
    expect(pinnedFor(pins, 'milk').map((p) => p.id)).toEqual(['campina', 'alpro', 'boni']);
    expect(pinnedFor(pins, 'bread')).toEqual([]);
  });

  it('gives the usual products for the suggestions, without repeats', () => {
    const usual = usualFor(pins, [generic('milk'), generic('chocolate-milk'), generic('cheese')]);
    expect(usual.map((p) => p.id)).toEqual(['campina', 'alpro', 'gouda']);
  });
});
