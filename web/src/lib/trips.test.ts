import { describe, expect, it } from 'vitest';
import type { Item } from './list';
import { addedDuring, currentTrip, going, told, type Trip } from './trips';

const trip = (over: Partial<Trip>): Trip =>
  ({
    id: 't',
    list: 'l',
    by: 'anna',
    store: '',
    ended: '',
    notified: [],
    created: '2026-11-09 10:00:00.000Z',
    ...over,
  }) as Trip;
const at = (date: string) => Date.parse(date);

describe('trips', () => {
  it('go on until ended, or for three hours', () => {
    expect(going(trip({}), at('2026-11-09T12:59:00Z'))).toBe(true);
    expect(going(trip({}), at('2026-11-09T13:01:00Z'))).toBe(false);
    expect(going(trip({ ended: '2026-11-09 10:30:00.000Z' }), at('2026-11-09T11:00:00Z'))).toBe(
      false,
    );
  });

  it('the latest one going counts, for its own list', () => {
    const now = at('2026-11-09T11:00:00Z');
    const later = trip({ id: 'later', by: 'ben', created: '2026-11-09 10:30:00.000Z' });
    expect(currentTrip([trip({}), later], 'l', now)?.id).toBe('later');
    expect(currentTrip([trip({})], 'other', now)).toBeUndefined();
  });

  it('marks what others added while out', () => {
    const item = (added_by: string, created: string) => ({ added_by, created }) as Item;
    const t = trip({});
    expect(addedDuring(item('ben', '2026-11-09 10:05:00.000Z'), t)).toBe(true);
    expect(addedDuring(item('anna', '2026-11-09 10:05:00.000Z'), t)).toBe(false);
    expect(addedDuring(item('ben', '2026-11-09 09:55:00.000Z'), t)).toBe(false);
    expect(addedDuring(item('ben', '2026-11-09 10:05:00.000Z'), undefined)).toBe(false);
  });

  it('name who got a notification', () => {
    const name = (id: string) => ({ ben: 'Ben', cleo: 'Cleo' })[id] ?? '';
    expect(told(trip({}), name)).toBe('');
    expect(told(trip({ notified: ['ben'] }), name)).toBe('Ben');
    expect(told(trip({ notified: ['ben', 'cleo'] }), name)).toBe('Ben and Cleo');
    expect(told(trip({ notified: ['gone', 'ben'] }), name)).toBe('Ben');
  });
});
