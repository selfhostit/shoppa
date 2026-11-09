import type { RecordModel } from 'pocketbase';
import { upsert, type Item } from '../list';
import { pb } from '../pb';
import { run } from './status.svelte';

/** An item that comes back on its list by itself, every so many days. */
export interface Staple extends RecordModel {
  item: string;
  every: number;
  /** when it next comes back, set by the server */
  next: string;
}

/** The periods offered, in days. */
export const STAPLE_DAYS = [7, 14, 28, 56] as const;

// Staples: "toilet paper, every 2 weeks". The server puts them back on the
// list each morning when they're due (backend/app/staples.go).
class Staples {
  staples = $state<Staple[]>([]);

  apply(staple: Staple, deleted = false) {
    this.staples = upsert(this.staples, staple, deleted);
  }

  of(item: Item): Staple | undefined {
    return this.staples.find((s) => s.item === item.id);
  }

  /** Makes an item come back every so many days, or, with 0, stops it. */
  async set(item: Item, every: number) {
    const was = this.of(item);
    await run(async () => {
      if (was && !every) {
        await pb.collection('staples').delete(was.id);
        this.apply(was, true);
      } else if (was) {
        this.apply(await pb.collection<Staple>('staples').update(was.id, { every }));
      } else if (every) {
        this.apply(await pb.collection<Staple>('staples').create({ item: item.id, every }));
      }
    });
  }
}

export const staples = new Staples();
