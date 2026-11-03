import type { UnsubscribeFunc } from 'pocketbase';
import { errorMessage, pb, type User } from './pb';
import { listsInOrder, parseEntry, upsert, type Item, type List } from './list';

const currentKey = 'shoppa:list';

function remembered(): string {
  try {
    return localStorage.getItem(currentKey) ?? '';
  } catch {
    return '';
  }
}

// The household's lists and the open list's items, kept live: a change made
// on any phone shows up on the others at once.
class Shop {
  lists = $state<List[]>([]);
  items = $state<Item[]>([]);
  people = $state<User[]>([]);
  listId = $state(remembered());
  loading = $state(true);
  error = $state('');

  private stops: UnsubscribeFunc[] = [];

  get ordered() {
    return listsInOrder(this.lists);
  }

  get list(): List | undefined {
    return this.ordered.find((l) => l.id === this.listId) ?? this.ordered[0];
  }

  /** A person's first name, for "by Anna". */
  nameOf(id: string): string {
    const p = this.people.find((u) => u.id === id);
    return (p?.name || p?.email || '').split(/[\s@]/)[0];
  }

  /** Loads everything and starts listening for changes; returns a stop function. */
  async start() {
    this.error = '';
    try {
      const [lists, people] = await Promise.all([
        pb.collection<List>('lists').getFullList(),
        pb.collection<User>('users').getFullList({ sort: 'name' }),
      ]);
      this.lists = lists;
      this.people = people;
      await this.loadItems();
      this.stops.push(
        await pb.collection<List>('lists').subscribe('*', ({ action, record }) => {
          this.lists = upsert(this.lists, record, action === 'delete');
          if (action === 'delete' && record.id === this.list?.id) this.loadItems();
        }),
        await pb.collection<Item>('items').subscribe('*', ({ action, record }) => {
          if (record.list !== this.list?.id) return;
          this.items = upsert(this.items, record, action === 'delete');
        }),
      );
    } catch (err) {
      this.error = errorMessage(err);
    } finally {
      this.loading = false;
    }
  }

  stop() {
    this.stops.splice(0).forEach((stop) => stop());
  }

  async loadItems() {
    const id = this.list?.id;
    if (!id) {
      this.items = [];
      return;
    }
    this.items = await pb
      .collection<Item>('items')
      .getFullList({ filter: pb.filter('list = {:id}', { id }) });
  }

  async open(listId: string) {
    this.listId = listId;
    try {
      localStorage.setItem(currentKey, listId);
    } catch {
      // private browsing: it's only a convenience
    }
    await this.run(() => this.loadItems());
  }

  /** Puts what was typed on the open list, or brings back the one already there. */
  async add(text: string) {
    const { name, quantity } = parseEntry(text);
    const list = this.list;
    if (!name || !list) return;
    await this.run(async () => {
      const item = await pb.send<Item>('/api/app/items', {
        method: 'POST',
        body: { list: list.id, name, ...(quantity ? { quantity } : {}) },
      });
      this.items = upsert(this.items, item);
    });
  }

  /** Puts a recently bought item back on the list. */
  async putBack(item: Item) {
    await this.save(item, { checked: false });
  }

  async tick(item: Item) {
    // the server sets when and by whom; the phone shows it at once
    await this.save(
      item,
      { checked: true },
      { checked_at: new Date().toISOString().replace('T', ' ') },
    );
  }

  /** Changes an item, showing the change (and `shown`) before the server answers. */
  async save(item: Item, change: Partial<Item>, shown: Partial<Item> = {}) {
    const before = this.items;
    this.items = upsert(this.items, { ...item, ...change, ...shown });
    await this.run(
      async () =>
        (this.items = upsert(
          this.items,
          await pb.collection<Item>('items').update(item.id, change),
        )),
      () => (this.items = before),
    );
  }

  async remove(item: Item) {
    const before = this.items;
    this.items = upsert(this.items, item, true);
    await this.run(
      () => pb.collection('items').delete(item.id),
      () => (this.items = before),
    );
  }

  async createList(name: string) {
    await this.run(async () => {
      const list = await pb.collection<List>('lists').create({
        name: name.trim(),
        emoji: '🛒',
        sort: Math.max(0, ...this.lists.map((l) => l.sort)) + 1,
      });
      this.lists = upsert(this.lists, list);
      await this.open(list.id);
    });
  }

  async renameList(list: List, name: string) {
    await this.run(async () => {
      this.lists = upsert(
        this.lists,
        await pb.collection<List>('lists').update(list.id, { name: name.trim() }),
      );
    });
  }

  async deleteList(list: List) {
    await this.run(async () => {
      await pb.collection('lists').delete(list.id);
      this.lists = upsert(this.lists, list, true);
      await this.open(this.ordered[0]?.id ?? '');
    });
  }

  private async run(fn: () => Promise<unknown>, undo?: () => void) {
    this.error = '';
    try {
      await fn();
    } catch (err) {
      undo?.();
      this.error = errorMessage(err);
    }
  }
}

export const shop = new Shop();
