import type { RecordModel } from 'pocketbase';
import type { Item } from './list';

/** Someone shopping for a list: "Anna is shopping at Colruyt". */
export interface Trip extends RecordModel {
  list: string;
  /** who is shopping */
  by: string;
  store: string;
  /** set when the shopper is done */
  ended: string;
  /** who got a notification about it: everyone else with notifications on */
  notified: string[];
  created: string;
}

/** How long a trip lasts when the shopper forgets to end it. */
export const TRIP_HOURS = 3;

/** PocketBase dates are "2026-11-09 12:00:00.000Z". */
const time = (date: string) => Date.parse(date.replace(' ', 'T'));

/** Whether a trip is still going. */
export function going(trip: Trip, now = Date.now()): boolean {
  return !trip.ended && time(trip.created) + TRIP_HOURS * 3600_000 > now;
}

/** The latest trip still going for a list, if anyone is shopping for it. */
export function currentTrip(trips: Trip[], list: string, now = Date.now()): Trip | undefined {
  return trips
    .filter((t) => t.list === list && going(t, now))
    .sort((a, b) => b.created.localeCompare(a.created))[0];
}

/** Whether someone else put an item on the list while the shopper was out. */
export function addedDuring(item: Item, trip: Trip | undefined): boolean {
  return !!trip && item.added_by !== trip.by && item.created >= trip.created;
}

const names = new Intl.ListFormat('en', { type: 'conjunction' });

/** "Ben and Cleo": who a trip told, or "" when nobody got a notification. */
export function told(trip: Trip, name: (person: string) => string): string {
  return names.format(trip.notified.map(name).filter(Boolean));
}
