import { describe, expect, it } from 'vitest';
import type { Generic } from './catalogue';
import type { Item } from './list';
import { DeviceCopy, enqueue, newId, renamed, withPending, type NewItem, type Op } from './offline';

class MemoryStorage {
  items = new Map<string, string>();
  getItem = (k: string) => this.items.get(k) ?? null;
  setItem = (k: string, v: string) => void this.items.set(k, v);
  removeItem = (k: string) => void this.items.delete(k);
}

const item = (over: Partial<Item>): Item =>
  ({
    id: 'i1',
    collectionId: 'items',
    collectionName: 'items',
    list: 'l',
    name: 'x',
    generic: '',
    product: '',
    quantity: '',
    note: '',
    checked: false,
    checked_at: '',
    checked_by: '',
    added_by: 'bo',
    created: '2026-11-04 10:00:00.000Z',
    ...over,
  }) as Item;

const at = '2026-11-04 11:00:00.000Z';
const add = (id: string, body: NewItem, existing = false): Op => ({
  kind: 'add',
  id,
  list: 'l',
  body,
  at,
  ...(existing ? { existing } : {}),
});
const update = (id: string, change: Partial<Item>): Op => ({
  kind: 'update',
  id,
  list: 'l',
  change,
  at,
});
const remove = (id: string): Op => ({ kind: 'delete', id, list: 'l', at });

describe('newId', () => {
  it('makes ids the way PocketBase does', () => {
    expect(newId()).toMatch(/^[a-z0-9]{15}$/);
    expect(newId()).not.toBe(newId());
  });
});

describe('enqueue', () => {
  it('folds two changes to one item into one', () => {
    let ops = enqueue([], update('i1', { checked: true }));
    ops = enqueue(ops, update('i2', { note: 'ripe' }));
    ops = enqueue(ops, update('i1', { checked: false, quantity: '2×' }));
    expect(ops).toEqual([
      update('i1', { checked: false, quantity: '2×' }),
      update('i2', { note: 'ripe' }),
    ]);
  });

  it('drops an item added and removed before it synced', () => {
    let ops = enqueue([], add('new', { name: 'Brood' }));
    ops = enqueue(ops, update('new', { checked: true }));
    expect(enqueue(ops, remove('new'))).toEqual([]);
  });

  it('still removes an item the server has', () => {
    let ops = enqueue([], add('i1', { name: 'Brood' }, true));
    ops = enqueue(ops, update('i1', { note: 'x' }));
    expect(enqueue(ops, remove('i1'))).toEqual([remove('i1')]);
  });
});

describe('renamed', () => {
  it('points what waited at the id the server answered with', () => {
    expect(renamed([update('new', { checked: true })], 'new', 'i1')).toEqual([
      update('i1', { checked: true }),
    ]);
  });
});

describe('withPending', () => {
  const milk = { id: 'gmilk', slug: 'milk' } as Generic;

  it('shows ticks, changes and removals at once', () => {
    const items = [
      item({ id: 'a' }),
      item({ id: 'b' }),
      item({ id: 'c', checked: true, checked_by: 'bo' }),
    ];
    const shown = withPending(
      items,
      [
        update('a', { checked: true, checked_at: at }),
        update('c', { checked: false }),
        remove('b'),
        { ...update('a', { note: 'other list' }), list: 'other' },
      ],
      'l',
      'ann',
    );
    expect(shown.map((i) => [i.id, i.checked, i.checked_by, i.note])).toEqual([
      ['a', true, 'ann', ''],
      ['c', false, '', ''],
    ]);
  });

  it('shows an item added offline', () => {
    const [made] = withPending(
      [],
      [add('new', { generic: 'gmilk', name: 'Melk', quantity: '2×' })],
      'l',
      'ann',
    );
    expect(made).toMatchObject({
      id: 'new',
      generic: 'gmilk',
      name: 'Melk',
      quantity: '2×',
      added_by: 'ann',
      created: at,
    });
  });

  it('brings back what the list already has, as the server does', () => {
    const items = [
      item({ id: 'm', generic: 'gmilk', checked: true }),
      item({ id: 'k', name: 'Kaas' }),
    ];
    const shown = withPending(
      items,
      [add('x1', { generic: 'gmilk' }), add('x2', { name: 'kaas ', quantity: '3×' })],
      'l',
      'ann',
      [milk],
    );
    expect(shown.map((i) => [i.id, i.checked, i.quantity, i.added_by])).toEqual([
      ['m', false, '', 'ann'],
      ['k', false, '3×', 'bo'],
    ]);
  });
});

describe('DeviceCopy', () => {
  it('keeps each person apart and forgets on sign-out', () => {
    const storage = new MemoryStorage() as unknown as Storage;
    const ann = new DeviceCopy('ann', storage);
    ann.saveOutbox([remove('i1')]);
    ann.saveSnapshot({ ...ann.snapshot(), items: { l: [item({})] } });
    expect(new DeviceCopy('bo', storage).outbox()).toEqual([]);
    expect(new DeviceCopy('ann', storage).snapshot().items.l).toHaveLength(1);
    ann.forget();
    expect(ann.outbox()).toEqual([]);
    expect(ann.snapshot().lists).toEqual([]);
  });

  it('starts empty when what was kept is broken', () => {
    const storage = new MemoryStorage() as unknown as Storage;
    storage.setItem('shoppa:outbox:ann', '{nope');
    expect(new DeviceCopy('ann', storage).outbox()).toEqual([]);
  });
});
