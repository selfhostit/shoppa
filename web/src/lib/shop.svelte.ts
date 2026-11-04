import type { UnsubscribeFunc } from 'pocketbase';
import {
  exactly,
  genericName,
  guessLang,
  OTHER,
  type Generic,
  type Lang,
  type Section,
} from './catalogue';
import { errorMessage, pb, type User } from './pb';
import { listsInOrder, parseEntry, upsert, type Item, type List } from './list';
import { kindOf, productLine, productName, type Product } from './products';
import { session } from './session.svelte';

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
  generics = $state<Generic[]>([]);
  sections = $state<Section[]>([]);
  /** the products on the list, by id, from the items' expand */
  products = $state<Record<string, Product>>({});
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

  /** The language item names show in for whoever is signed in. */
  get lang(): Lang {
    return session.user?.lang || guessLang(navigator.languages ?? [navigator.language]);
  }

  genericOf(item: Item): Generic | undefined {
    return item.generic ? this.generics.find((g) => g.id === item.generic) : undefined;
  }

  productOf(item: Item): Product | undefined {
    return item.product ? this.products[item.product] : undefined;
  }

  /** The catalogue item an item is, or its product is a kind of. */
  private kind(item: Item): Generic | undefined {
    const p = this.productOf(item);
    return p ? kindOf(p, this.generics) : this.genericOf(item);
  }

  /**
   * An item's name: its product's or generic's in your language, or what was
   * typed.
   */
  nameOf(item: Item): string {
    const p = this.productOf(item);
    if (p) return productName(p, this.lang);
    const g = this.genericOf(item);
    return g ? genericName(g, this.lang) : item.name;
  }

  /** "Boni · 1 L" for an exact product, nothing otherwise. */
  lineOf(item: Item): string {
    const p = this.productOf(item);
    return p ? productLine(p) : '';
  }

  nameOfGeneric(g: Generic): string {
    return genericName(g, this.lang);
  }

  emojiOf(item: Item): string {
    return this.kind(item)?.emoji ?? '';
  }

  sectionOf(item: Item): string {
    return this.kind(item)?.section || OTHER;
  }

  /** Keeps the products that come along with items. */
  private keep(items: Item[]) {
    const found = items.map((i) => i.expand?.product as Product | undefined).filter((p) => p);
    if (found.length)
      this.products = { ...this.products, ...Object.fromEntries(found.map((p) => [p!.id, p!])) };
    return items;
  }

  /** A person's first name, for "by Anna". */
  personName(id: string): string {
    const p = this.people.find((u) => u.id === id);
    return (p?.name || p?.email || '').split(/[\s@]/)[0];
  }

  /** Loads everything and starts listening for changes; returns a stop function. */
  async start() {
    this.error = '';
    try {
      const [lists, people, generics, sections] = await Promise.all([
        pb.collection<List>('lists').getFullList(),
        pb.collection<User>('users').getFullList({ sort: 'name' }),
        pb.collection<Generic>('generics').getFullList({ batch: 1000 }),
        pb.send<Section[]>('/api/app/sections', {}),
      ]);
      this.lists = lists;
      this.people = people;
      this.generics = generics;
      this.sections = sections;
      await this.loadItems();
      this.stops.push(
        await pb.collection<List>('lists').subscribe('*', ({ action, record }) => {
          this.lists = upsert(this.lists, record, action === 'delete');
          if (action === 'delete' && record.id === this.list?.id) this.loadItems();
        }),
        await pb.collection<Generic>('generics').subscribe('*', ({ action, record }) => {
          this.generics = upsert(this.generics, record, action === 'delete');
        }),
        await pb.collection<Item>('items').subscribe(
          '*',
          ({ action, record }) => {
            if (record.list !== this.list?.id) return;
            this.keep([record]);
            this.items = upsert(this.items, record, action === 'delete');
          },
          { expand: 'product' },
        ),
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
    this.items = this.keep(
      await pb
        .collection<Item>('items')
        .getFullList({ filter: pb.filter('list = {:id}', { id }), expand: 'product' }),
    );
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

  /**
   * Puts what was typed on the open list, as the generic it names if there is
   * one, or brings back the one already there.
   */
  async add(text: string, generic?: Generic, product?: Product) {
    const { name, quantity } = parseEntry(text);
    const list = this.list;
    if (!product) generic ??= exactly(this.generics, name, this.lang);
    if (!(name || generic || product) || !list) return;
    if (product) this.products = { ...this.products, [product.id]: product };
    await this.run(async () => {
      const item = await pb.send<Item>('/api/app/items', {
        method: 'POST',
        body: {
          list: list.id,
          ...(product ? { product: product.id } : generic ? { generic: generic.id } : { name }),
          ...(quantity ? { quantity } : {}),
        },
      });
      this.items = upsert(this.items, item);
    });
  }

  /** Products for what is being typed, best first. */
  async searchProducts(q: string): Promise<Product[]> {
    try {
      return await pb.send<Product[]>('/api/app/search', {
        query: { q, limit: 8 },
        requestKey: 'search',
      });
    } catch {
      // a newer search took its place, or the server can't be reached
      return [];
    }
  }

  /** Keeps a typed item as one of the household's own, with an emoji and a section. */
  async remember(item: Item, emoji: string, section: string) {
    await this.run(async () => {
      const generic = await pb.collection<Generic>('generics').create({
        [`name_${this.lang}`]: item.name,
        emoji: emoji.trim(),
        section,
      });
      this.generics = upsert(this.generics, generic);
      this.items = upsert(
        this.items,
        await pb.collection<Item>('items').update(item.id, { generic: generic.id }),
      );
    });
  }

  async setLang(lang: Lang) {
    const user = session.user;
    if (!user) return;
    await this.run(() => pb.collection('users').update(user.id, { lang }));
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
          this.keep([
            await pb.collection<Item>('items').update(item.id, change, { expand: 'product' }),
          ])[0],
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
