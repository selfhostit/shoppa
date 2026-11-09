import { describe, expect, it } from 'vitest';
import { t, tn } from './i18n';
import {
  itemKey,
  listsInOrder,
  parseEntry,
  splitEntries,
  recentlyBought,
  toBuy,
  upsert,
  type Item,
  type List,
} from './list';

const item = (over: Partial<Item>): Item =>
  ({
    id: over.name ?? 'x',
    collectionId: 'items',
    collectionName: 'items',
    list: 'l',
    name: 'x',
    quantity: '',
    note: '',
    checked: false,
    checked_at: '',
    checked_by: '',
    added_by: '',
    created: '2026-11-04 10:00:00.000Z',
    ...over,
  }) as Item;

describe('parseEntry', () => {
  it('finds a quantity before or after the name', () => {
    for (const text of [
      '2 melk',
      '2x melk',
      '2× melk',
      '2 x melk',
      'melk x2',
      'melk 2x',
      'melk 2×',
      '2xmelk',
    ]) {
      expect(parseEntry(text), text).toEqual({ name: 'melk', quantity: '2×' });
    }
  });

  it('leaves names with numbers alone', () => {
    expect(parseEntry('  7up  ')).toEqual({ name: '7up', quantity: '' });
    expect(parseEntry('Brood')).toEqual({ name: 'Brood', quantity: '' });
    expect(parseEntry('Cola 1,5 L')).toEqual({ name: 'Cola 1,5 L', quantity: '' });
  });
});

describe('splitEntries', () => {
  it('splits lines, commas and semicolons', () => {
    expect(splitEntries('melk, 2 brood; eieren\nkaas')).toEqual([
      'melk',
      '2 brood',
      'eieren',
      'kaas',
    ]);
  });
  it('drops bullets, numbering, checkboxes and a leading "and"', () => {
    expect(
      splitEntries('- melk\n* brood\n• kaas\n1. eieren\n2) boter\n[ ] thee\n[x] koffie'),
    ).toEqual(['melk', 'brood', 'kaas', 'eieren', 'boter', 'thee', 'koffie']);
    expect(splitEntries('milk, bread, and eggs')).toEqual(['milk', 'bread', 'eggs']);
  });
  it('keeps a decimal comma and a quantity in front', () => {
    expect(splitEntries('1,5 kg tomaten, 2 melk')).toEqual(['1,5 kg tomaten', '2 melk']);
  });
  it('leaves one item alone, and empty lines out', () => {
    expect(splitEntries('melk')).toEqual(['melk']);
    expect(splitEntries('\n , melk,\n\n')).toEqual(['melk']);
  });
});

describe('itemKey', () => {
  it('matches the way the server tells items apart', () => {
    expect(itemKey('  Halfvolle   MÉLK!')).toBe('halfvolle melk');
    expect(itemKey('Crème brûlée')).toBe('creme brulee');
    expect(itemKey('Œufs')).toBe('oeufs');
  });
});

describe('the list', () => {
  const items = [
    item({ name: 'Kaas', created: '2026-11-04 10:02:00.000Z' }),
    item({ name: 'Brood', created: '2026-11-04 10:01:00.000Z' }),
    item({ name: 'Melk', checked: true, checked_at: '2026-11-03 18:00:00.000Z' }),
    item({ name: 'Eieren', checked: true, checked_at: '2026-11-04 09:00:00.000Z' }),
  ];

  it('shows what to buy in the order it was added', () => {
    expect(toBuy(items).map((i) => i.name)).toEqual(['Brood', 'Kaas']);
  });

  it('shows the latest purchases first', () => {
    expect(recentlyBought(items).map((i) => i.name)).toEqual(['Eieren', 'Melk']);
    expect(recentlyBought(items, 1).map((i) => i.name)).toEqual(['Eieren']);
  });

  it('keeps realtime changes in step', () => {
    const changed = upsert(items, { ...items[0], checked: true });
    expect(changed.filter((i) => i.name === 'Kaas')).toHaveLength(1);
    expect(upsert(items, items[0], true)).toHaveLength(3);
  });

  it('orders the lists and leaves archived ones out', () => {
    const list = (name: string, sort: number, archived = false) =>
      ({ id: name, name, sort, archived, created: '' }) as unknown as List;
    expect(
      listsInOrder([list('B', 2), list('A', 1), list('Old', 0, true)]).map((l) => l.name),
    ).toEqual(['A', 'B']);
  });
});

describe('texts', () => {
  it('fills in values and counts', () => {
    expect(t('list.tick', { name: 'Melk' })).toBe('Tick off Melk');
    expect(tn('list.items', 1)).toBe('1 item');
    expect(tn('list.items', 3)).toBe('3 items');
  });
});
