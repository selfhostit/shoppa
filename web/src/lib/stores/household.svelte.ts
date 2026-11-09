import { kindOfList, type Kind } from '../kinds';
import { listsInOrder, upsert, type List } from '../list';
import { pb, type User } from '../pb';
import { run } from './status.svelte';

const currentKey = 'shoppa:list';

function remembered(): string {
  try {
    return localStorage.getItem(currentKey) ?? '';
  } catch {
    return '';
  }
}

// The household's lists and people, and which list is open.
class Household {
  lists = $state<List[]>([]);
  people = $state<User[]>([]);
  listId = $state(remembered());

  get ordered() {
    return listsInOrder(this.lists);
  }

  get list(): List | undefined {
    return this.ordered.find((l) => l.id === this.listId) ?? this.ordered[0];
  }

  /** What the open list is for. */
  get kind(): Kind {
    return kindOfList(this.list);
  }

  /** A person's first name, for "by Anna". */
  personName(id: string): string {
    const p = this.people.find((u) => u.id === id);
    return (p?.name || p?.email || '').split(/[\s@]/)[0];
  }

  /** Makes a list the open one, here and next time the app opens. */
  choose(listId: string) {
    this.listId = listId;
    try {
      localStorage.setItem(currentKey, listId);
    } catch {
      // private browsing: it's only a convenience
    }
  }

  /** Keeps a list as the server sent it, or drops it. */
  apply(list: List, deleted = false) {
    this.lists = upsert(this.lists, list, deleted);
  }

  /** Makes a new list, last in the order, and returns it. */
  async create(name: string, kind: Kind): Promise<List> {
    const list = await pb.collection<List>('lists').create({
      name: name.trim(),
      kind,
      sort: Math.max(0, ...this.lists.map((l) => l.sort)) + 1,
    });
    this.apply(list);
    return list;
  }

  /** Starts a new list from an old one: its kind and every item, all still to buy. */
  async copy(from: List, name: string): Promise<List> {
    const list = await pb.send<List>(`/api/app/lists/${from.id}/copy`, {
      method: 'POST',
      body: { name: name.trim() },
    });
    this.apply(list);
    return list;
  }

  async rename(list: List, name: string) {
    await run(async () =>
      this.apply(await pb.collection<List>('lists').update(list.id, { name: name.trim() })),
    );
  }

  /** Changes what a list is for, and so its sections. */
  async setKind(list: List, kind: Kind) {
    await run(async () => this.apply(await pb.collection<List>('lists').update(list.id, { kind })));
  }

  async delete(list: List) {
    await pb.collection('lists').delete(list.id);
    this.apply(list, true);
  }
}

export const household = new Household();
