import type { Choice, Generic, Section } from './catalogue';
import { itemKey, type Item, type List } from './list';
import type { User } from './pb';
import type { Pin } from './pins';
import type { Product } from './products';

// Shoppa keeps a copy of the household's lists on the phone, so the list opens
// in a shop with no signal. Changes made then wait here, in the order they were
// made, and are sent once the server can be reached. New items get a real
// PocketBase id on the phone, so nothing needs renaming after the sync.

/** What is put on a list: what /api/app/items takes. */
export type NewItem = {
  name?: string;
  generic?: string;
  product?: string;
  quantity?: string;
};

export type Op =
  | {
      kind: 'add';
      id: string;
      list: string;
      body: NewItem;
      at: string;
      /** the list already had it (maybe ticked off), under this id */
      existing?: boolean;
    }
  | { kind: 'update'; id: string; list: string; change: Partial<Item>; at: string }
  | { kind: 'delete'; id: string; list: string; at: string };

/** Everything the list screen needs, as last seen from the server. */
export type Snapshot = {
  lists: List[];
  people: User[];
  generics: Generic[];
  sections: Section[];
  pins: Pin[];
  /** the household's picks of section and icon */
  choices: Choice[];
  products: Record<string, Product>;
  /** each list's items, by list id */
  items: Record<string, Item[]>;
};

export const emptySnapshot = (): Snapshot => ({
  lists: [],
  people: [],
  generics: [],
  sections: [],
  pins: [],
  choices: [],
  products: {},
  items: {},
});

const snapshotKey = (user: string) => `shoppa:copy:${user}`;
const outboxKey = (user: string) => `shoppa:outbox:${user}`;

function read<T>(storage: Storage, key: string, fallback: T): T {
  try {
    return (JSON.parse(storage.getItem(key) ?? 'null') as T) ?? fallback;
  } catch {
    return fallback;
  }
}

function write(storage: Storage, key: string, value: unknown) {
  try {
    storage.setItem(key, JSON.stringify(value));
  } catch {
    // private browsing or a full disk: carry on online only
  }
}

/** The copy kept on this phone for one person. */
export class DeviceCopy {
  constructor(
    private user: string,
    private storage: Storage = localStorage,
  ) {}

  snapshot(): Snapshot {
    return {
      ...emptySnapshot(),
      ...read<Partial<Snapshot>>(this.storage, snapshotKey(this.user), {}),
    };
  }

  outbox(): Op[] {
    const ops = read<Op[]>(this.storage, outboxKey(this.user), []);
    return Array.isArray(ops) ? ops : [];
  }

  saveSnapshot(s: Snapshot) {
    write(this.storage, snapshotKey(this.user), s);
  }

  saveOutbox(ops: Op[]) {
    if (ops.length) write(this.storage, outboxKey(this.user), ops);
    else this.storage.removeItem(outboxKey(this.user));
  }

  forget() {
    this.storage.removeItem(snapshotKey(this.user));
    this.storage.removeItem(outboxKey(this.user));
  }
}

/** A new record id, the way PocketBase makes them: 15 of a-z and 0-9. */
export function newId(random: () => number = Math.random): string {
  const chars = 'abcdefghijklmnopqrstuvwxyz0123456789';
  let id = '';
  for (let i = 0; i < 15; i++) id += chars[Math.floor(random() * chars.length)];
  return id;
}

/** Now, in PocketBase's date format. */
export function pbNow(now = new Date()): string {
  return now.toISOString().replace('T', ' ');
}

/** True when the request never reached the server (offline, NAS down). */
export function unreachable(err: unknown): boolean {
  const e = err as { status?: number; isAbort?: boolean };
  return e?.status === 0 && !e.isAbort;
}

/**
 * Adds a change to the queue. Two changes to one item become one, and
 * removing an item that never reached the server drops it altogether.
 */
export function enqueue(ops: Op[], op: Op): Op[] {
  if (op.kind === 'update') {
    const at = ops.findIndex((o) => o.kind === 'update' && o.id === op.id);
    if (at >= 0) {
      const was = ops[at] as Extract<Op, { kind: 'update' }>;
      return ops.map((o, i) =>
        i === at ? { ...was, change: { ...was.change, ...op.change } } : o,
      );
    }
  }
  if (op.kind === 'delete') {
    const added = ops.some((o) => o.kind === 'add' && o.id === op.id && !o.existing);
    ops = ops.filter((o) => o.id !== op.id);
    if (added) return ops;
  }
  return [...ops, op];
}

/** After a queued add, the server may answer with the item already on the list. */
export function renamed(ops: Op[], from: string, to: string): Op[] {
  return from === to ? ops : ops.map((o) => (o.id === from ? { ...o, id: to } : o));
}

/** What makes two items one, as the server sees it (backend/app/lists.go). */
export function sameThing(i: Item, body: NewItem, generics: Generic[]): boolean {
  if (body.product) return i.product === body.product;
  if (body.generic) return !i.product && i.generic === body.generic;
  const g = generics.find((g) => g.id === i.generic);
  return !i.product && !g && itemKey(i.name) === itemKey(body.name ?? '');
}

/**
 * One list's items with the changes still waiting to be sent laid over them,
 * so the list looks as it will once they are.
 */
export function withPending(
  items: Item[],
  ops: Op[],
  list: string,
  me: string,
  generics: Generic[] = [],
): Item[] {
  let out = items;
  for (const op of ops) {
    if (op.list !== list) continue;
    if (op.kind === 'add') {
      const there =
        out.find((i) => i.id === op.id) ?? out.find((i) => sameThing(i, op.body, generics));
      if (there) {
        out = out.map((i) =>
          i === there
            ? {
                ...i,
                ...(i.checked ? { checked: false, checked_by: '', added_by: me } : {}),
                ...(op.body.quantity ? { quantity: op.body.quantity } : {}),
              }
            : i,
        );
        continue;
      }
      out = [
        ...out,
        {
          id: op.id,
          collectionId: '',
          collectionName: 'items',
          list,
          name: op.body.name ?? '',
          generic: op.body.generic ?? '',
          product: op.body.product ?? '',
          quantity: op.body.quantity ?? '',
          note: '',
          checked: false,
          checked_at: '',
          checked_by: '',
          added_by: me,
          created: op.at,
        } as Item,
      ];
    } else if (op.kind === 'update') {
      out = out.map((i) =>
        i.id === op.id
          ? {
              ...i,
              ...op.change,
              ...(op.change.checked === true && !i.checked ? { checked_by: me } : {}),
              ...(op.change.checked === false ? { checked_by: '' } : {}),
            }
          : i,
      );
    } else {
      out = out.filter((i) => i.id !== op.id);
    }
  }
  return out;
}
