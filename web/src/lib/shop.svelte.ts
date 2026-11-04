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
import {
  DeviceCopy,
  emptySnapshot,
  enqueue,
  newId,
  pbNow,
  renamed,
  sameThing,
  unreachable,
  withPending,
  type NewItem,
  type Op,
  type Snapshot,
} from './offline';
import { pinnedFor, type Pin } from './pins';
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

/** How often to try the server again while it can't be reached. */
const RETRY_MS = 30_000;

// The household's lists and the open list's items, kept live: a change made
// on any phone shows up on the others at once. A copy stays on the phone, and
// changes to items go through a queue that waits while the server can't be
// reached (lib/offline.ts).
class Shop {
  lists = $state<List[]>([]);
  items = $state<Item[]>([]);
  people = $state<User[]>([]);
  generics = $state<Generic[]>([]);
  sections = $state<Section[]>([]);
  /** the household's choice of product for a generic item */
  pins = $state<Pin[]>([]);
  /** the products on the list, by id, from the items' expand */
  products = $state<Record<string, Product>>({});
  listId = $state(remembered());
  loading = $state(true);
  error = $state('');
  /** true while the server can't be reached */
  offline = $state(false);
  /** changes to items not sent yet, oldest first */
  outbox = $state<Op[]>([]);

  private stops: UnsubscribeFunc[] = [];
  /** the open list's items as the server last sent them */
  private base: Item[] = [];
  private copy: DeviceCopy | null = null;
  private snap: Snapshot = emptySnapshot();
  private sending: Promise<void> | null = null;
  private connecting: Promise<void> | null = null;
  private inFlight: Op | null = null;
  private retry: ReturnType<typeof setInterval> | undefined;
  private persistTimer: ReturnType<typeof setTimeout> | undefined;
  private onOnline = () => this.connect();
  private onVisible = () => document.visibilityState === 'visible' && this.connect();

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

  /** Opens the copy on this phone, then loads from the server and listens for changes. */
  async start() {
    // a tapped notification opens its list
    const asked = new URLSearchParams(location.search).get('list');
    if (asked) {
      this.choose(asked);
      history.replaceState(null, '', location.pathname + location.hash);
    }
    const me = session.user?.id ?? '';
    this.copy = new DeviceCopy(me);
    this.snap = this.copy.snapshot();
    this.outbox = this.copy.outbox();
    const s = this.snap;
    this.lists = s.lists;
    this.people = s.people;
    this.generics = s.generics;
    this.sections = s.sections;
    this.pins = s.pins;
    this.products = s.products;
    this.setBase(s.items[this.list?.id ?? ''] ?? []);
    if (s.lists.length) this.loading = false;

    addEventListener('online', this.onOnline);
    document.addEventListener('visibilitychange', this.onVisible);
    this.retry = setInterval(
      () => (this.offline || this.outbox.length) && this.connect(),
      RETRY_MS,
    );
    await this.connect();
  }

  stop() {
    removeEventListener('online', this.onOnline);
    document.removeEventListener('visibilitychange', this.onVisible);
    clearInterval(this.retry);
    clearTimeout(this.persistTimer);
    this.stops.splice(0).forEach((stop) => stop());
  }

  /** Loads everything from the server, starts listening once, and sends what waited. */
  connect(): Promise<void> {
    this.connecting ??= this.load().finally(() => (this.connecting = null));
    return this.connecting;
  }

  private async load() {
    try {
      const [lists, people, generics, sections, pins] = await Promise.all([
        pb.collection<List>('lists').getFullList(),
        pb.collection<User>('users').getFullList({ sort: 'name' }),
        pb.collection<Generic>('generics').getFullList({ batch: 1000 }),
        pb.send<Section[]>('/api/app/sections', {}),
        pb.collection<Pin>('pins').getFullList({ expand: 'product' }),
      ]);
      this.pins = pins;
      this.lists = lists;
      this.people = people;
      this.generics = generics;
      this.sections = sections;
      await this.loadItems();
      if (!this.stops.length) await this.listen();
      this.offline = false;
      if (this.error === errorMessage({ status: 0 })) this.error = '';
      await this.flush();
    } catch (err) {
      if (unreachable(err)) this.offline = true;
      else this.error = errorMessage(err);
    } finally {
      this.loading = false;
      this.persist();
    }
  }

  private async listen() {
    let first = true;
    this.stops.push(
      // live updates reconnect by themselves; catch up on what was missed meanwhile
      await pb.realtime.subscribe('PB_CONNECT', () => {
        if (first) first = false;
        else this.connect();
      }),
      await pb.collection<List>('lists').subscribe('*', ({ action, record }) => {
        this.lists = upsert(this.lists, record, action === 'delete');
        if (action === 'delete' && record.id === this.list?.id) this.loadItems();
        this.persist();
      }),
      await pb.collection<Pin>('pins').subscribe(
        '*',
        ({ action, record }) => {
          this.pins = upsert(this.pins, record, action === 'delete');
          this.persist();
        },
        { expand: 'product' },
      ),
      await pb.collection<Generic>('generics').subscribe('*', ({ action, record }) => {
        this.generics = upsert(this.generics, record, action === 'delete');
        this.persist();
      }),
      await pb.collection<Item>('items').subscribe(
        '*',
        ({ action, record }) => {
          if (record.list !== this.list?.id) return;
          this.keep([record]);
          this.setBase(upsert(this.base, record, action === 'delete'));
        },
        { expand: 'product' },
      ),
    );
  }

  /** The open list's items: the server's, with the changes still waiting laid over. */
  private setBase(items: Item[]) {
    this.base = items;
    this.show();
    this.persist();
  }

  private show() {
    const list = this.list?.id ?? '';
    this.items = withPending(this.base, this.outbox, list, session.user?.id ?? '', this.generics);
  }

  /** Writes the copy on the phone, a moment after the last change. */
  private persist() {
    clearTimeout(this.persistTimer);
    this.persistTimer = setTimeout(() => {
      const list = this.list?.id;
      this.snap = {
        lists: this.lists,
        people: this.people,
        generics: this.generics,
        sections: this.sections,
        pins: this.pins,
        products: this.products,
        items: { ...this.snap.items, ...(list ? { [list]: this.base } : {}) },
      };
      this.copy?.saveSnapshot($state.snapshot(this.snap) as Snapshot);
    }, 300);
  }

  async loadItems() {
    const id = this.list?.id;
    if (!id) {
      this.setBase([]);
      return;
    }
    try {
      this.setBase(
        this.keep(
          await pb
            .collection<Item>('items')
            .getFullList({ filter: pb.filter('list = {:id}', { id }), expand: 'product' }),
        ),
      );
    } catch (err) {
      if (!unreachable(err)) throw err;
      this.offline = true;
      this.setBase(this.snap.items[id] ?? []);
    }
  }

  async open(listId: string) {
    this.choose(listId);
    await this.run(() => this.loadItems());
  }

  /** Makes a list the open one, here and next time the app opens. */
  private choose(listId: string) {
    this.listId = listId;
    try {
      localStorage.setItem(currentKey, listId);
    } catch {
      // private browsing: it's only a convenience
    }
  }

  /** Queues a change to an item, shows it at once, and sends it if it can. */
  private async change(op: Op) {
    // the change being sent can't take more in
    const [head, ...rest] = this.outbox;
    this.outbox =
      head && head === this.inFlight ? [head, ...enqueue(rest, op)] : enqueue(this.outbox, op);
    this.copy?.saveOutbox($state.snapshot(this.outbox) as Op[]);
    this.show();
    await this.flush();
  }

  /** Sends the waiting changes, oldest first, until one can't get through. */
  flush(): Promise<void> {
    this.sending ??= (async () => {
      while (this.outbox.length) {
        const op = this.outbox[0];
        this.inFlight = op;
        let id = op.id;
        try {
          id = await this.send(op);
        } catch (err) {
          if (unreachable(err)) {
            this.offline = true;
            break;
          }
          // gone already (someone else removed it), or refused: drop it
          if ((err as { status?: number }).status !== 404) this.error = errorMessage(err);
        } finally {
          this.inFlight = null;
        }
        this.outbox = renamed(this.outbox.slice(1), op.id, id);
        this.copy?.saveOutbox($state.snapshot(this.outbox) as Op[]);
        this.show();
      }
    })().finally(() => (this.sending = null));
    return this.sending;
  }

  /** Sends one change; returns the item's id on the server. */
  private async send(op: Op): Promise<string> {
    const here = op.list === this.list?.id;
    if (op.kind === 'add') {
      const item = await pb.send<Item>('/api/app/items', {
        method: 'POST',
        body: { id: op.id, list: op.list, ...op.body },
      });
      // the list may have had it already, under its own id
      if (here) this.base = upsert(upsert(this.base, { id: op.id } as Item, true), item);
      return item.id;
    }
    if (op.kind === 'update') {
      const item = await pb
        .collection<Item>('items')
        .update(op.id, op.change, { expand: 'product' });
      if (here) this.base = upsert(this.base, this.keep([item])[0]);
      return op.id;
    }
    await pb.collection('items').delete(op.id);
    if (here) this.base = upsert(this.base, { id: op.id } as Item, true);
    return op.id;
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
    const body: NewItem = product
      ? { product: product.id }
      : generic
        ? { generic: generic.id }
        : { name };
    if (quantity) body.quantity = quantity;
    const shown = product
      ? productName(product, this.lang)
      : generic
        ? genericName(generic, this.lang)
        : name;
    body.name ??= shown;
    // what's on the list already comes back, as the server would do
    const there = this.items.find((i) => sameThing(i, body, this.generics));
    await this.change({
      kind: 'add',
      id: there?.id ?? newId(),
      list: list.id,
      body,
      at: pbNow(),
      ...(there ? { existing: true } : {}),
    });
  }

  /** The products pinned for a generic item, the one bought most first. */
  pinnedFor(generic: string): Product[] {
    return pinnedFor(this.pins, generic);
  }

  /**
   * Makes a generic item the exact product picked for it, and remembers the
   * choice. When that product is already on the list, the two become one.
   */
  async pin(item: Item, product: Product) {
    this.products = { ...this.products, [product.id]: product };
    await this.flush();
    await this.run(async () => {
      const kept = await pb.send<Item>(`/api/app/items/${item.id}/pin`, {
        method: 'POST',
        body: { product: product.id },
      });
      this.setBase(upsert(kept.id === item.id ? this.base : upsert(this.base, item, true), kept));
    });
  }

  async unpin(generic: string, product: Product) {
    const pin = this.pins.find((p) => p.generic === generic && p.product === product.id);
    if (!pin) return;
    const before = this.pins;
    this.pins = upsert(this.pins, pin, true);
    await this.run(
      () => pb.collection('pins').delete(pin.id),
      () => (this.pins = before),
    );
  }

  /** The product with this barcode, or null when nobody knows it. */
  async lookup(code: string): Promise<Product | null> {
    try {
      return await pb.send<Product>(`/api/app/barcode/${encodeURIComponent(code)}`, {});
    } catch (err) {
      if ((err as { status?: number }).status === 404) return null;
      throw err;
    }
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
    await this.flush();
    await this.run(async () => {
      const generic = await pb.collection<Generic>('generics').create({
        [`name_${this.lang}`]: item.name,
        emoji: emoji.trim(),
        section,
      });
      this.generics = upsert(this.generics, generic);
      this.setBase(
        upsert(
          this.base,
          await pb.collection<Item>('items').update(item.id, { generic: generic.id }),
        ),
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
    // the server keeps who did it, and when, even when that was offline
    await this.save(item, { checked: true, checked_at: pbNow() });
  }

  /** Changes an item: shown at once, sent when the server can be reached. */
  async save(item: Item, change: Partial<Item>) {
    await this.change({ kind: 'update', id: item.id, list: item.list, change, at: pbNow() });
  }

  async remove(item: Item) {
    await this.change({ kind: 'delete', id: item.id, list: item.list, at: pbNow() });
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
