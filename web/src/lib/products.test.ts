import { describe, expect, it } from 'vitest';
import type { Generic } from './catalogue';
import { kindOf, photoUrl, productLine, productName, type Product } from './products';

const boni = {
  id: 'boni',
  barcode: '5410000000001',
  name: 'Halfvolle melk',
  name_nl: '',
  name_fr: 'Lait demi-écrémé',
  brand: 'Boni',
  quantity: '1 L',
  categories: 'en:dairies,en:milks,en:semi-skimmed-milks',
  image_url: 'https://images.example/541.jpg',
} as Product;

const generic = (slug: string, off: string) =>
  ({ id: slug, slug, section: 'dairy', off_categories: off }) as Generic;

describe('products', () => {
  it('names a product in your language', () => {
    expect(productName(boni, 'fr')).toBe('Lait demi-écrémé');
    expect(productName(boni, 'nl')).toBe('Halfvolle melk');
    expect(productName(boni, 'en')).toBe('Halfvolle melk');
  });

  it('tells products apart by brand and size', () => {
    expect(productLine(boni)).toBe('Boni · 1 L');
    expect(productLine({ ...boni, brand: '' })).toBe('1 L');
  });

  it('shows the photo through the server', () => {
    expect(photoUrl(boni)).toBe('/api/app/products/boni/photo');
    expect(photoUrl({ ...boni, image_url: '' })).toBe('');
  });

  it('finds what kind of thing a product is, most specific first', () => {
    const generics = [
      generic('milk', 'en:milks'),
      generic('skimmed-milk', 'en:skimmed-milks'),
      generic('semi', 'en:semi-skimmed-milks'),
    ];
    expect(kindOf(boni, generics)?.slug).toBe('semi');
    expect(kindOf(boni, generics.slice(0, 2))?.slug).toBe('milk');
    expect(kindOf({ ...boni, categories: '' }, generics)).toBeUndefined();
  });
});
