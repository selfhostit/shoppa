import { upsert } from '../list';
import { pbNow } from '../offline';
import { pb } from '../pb';
import { session } from '../session.svelte';
import { currentTrip, TRIP_HOURS, type Trip } from '../trips';
import { household } from './household.svelte';
import { run } from './status.svelte';

// Who is shopping right now: "Anna is shopping at Colruyt". Starting a trip
// tells everyone else to add what they need now (backend/app/trips.go).
class Trips {
  trips = $state<Trip[]>([]);

  /** The trips that may still be going, for loading. */
  get filter() {
    return pb.filter('created >= {:since}', {
      since: pbNow(new Date(Date.now() - TRIP_HOURS * 3600_000)),
    });
  }

  apply(trip: Trip, deleted = false) {
    this.trips = upsert(this.trips, trip, deleted);
  }

  /** Who is shopping for the open list, if anyone. */
  get current(): Trip | undefined {
    return currentTrip(this.trips, household.list?.id ?? '');
  }

  /** Whether it's you shopping for the open list. */
  get mine(): boolean {
    return !!this.current && this.current.by === session.user?.id;
  }

  async start(store: string) {
    const list = household.list;
    if (!list) return;
    await run(async () =>
      this.apply(await pb.collection<Trip>('trips').create({ list: list.id, store: store.trim() })),
    );
  }

  async end() {
    const trip = this.current;
    if (!trip) return;
    await run(async () =>
      this.apply(await pb.collection<Trip>('trips').update(trip.id, { ended: pbNow(new Date()) })),
    );
  }
}

export const trips = new Trips();
