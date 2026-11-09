import type { RecordModel, UnsubscribeFunc } from 'pocketbase';
import type { Choice, Generic, Section, SectionIcon } from './catalogue';
import type { Kind } from './kinds';
import type { Item, List } from './list';
import { DeviceCopy, emptySnapshot, unreachable, type Snapshot } from './offline';
import { errorMessage, pb, type User } from './pb';
import type { Pin } from './pins';
import { session } from './session.svelte';
import { catalogue } from './stores/catalogue.svelte';
import { choices } from './stores/choices.svelte';
import { household } from './stores/household.svelte';
import { items } from './stores/items.svelte';
import { pins } from './stores/pins.svelte';
import { staples, type Staple } from './stores/staples.svelte';
import { trips } from './stores/trips.svelte';
import type { Trip } from './trips';
import { run, status } from './stores/status.svelte';

/**
 * The household's collections each kept whole in a store: loaded on every
 * connect, then kept live record by record.
 */
type Synced = {
  name: string;
  options: () => { filter?: string; expand?: string };
  set: (records: RecordModel[]) => void;
  apply: (record: RecordModel, deleted: boolean) => void;
};

const synced: Synced[] = [
  {
    name: 'lists',
    options: () => ({}),
    set: (r) => (household.lists = r as List[]),
    apply: (r, d) => household.apply(r as List, d),
  },
  {
    name: 'generics',
    options: () => ({}),
    set: (r) => (catalogue.generics = r as Generic[]),
    apply: (r, d) => catalogue.applyGeneric(r as Generic, d),
  },
  {
    name: 'pins',
    options: () => ({ expand: 'product' }),
    set: (r) => (pins.pins = r as Pin[]),
    apply: (r, d) => pins.apply(r as Pin, d),
  },
  {
    name: 'item_choices',
    options: () => ({}),
    set: (r) => (choices.choices = r as Choice[]),
    apply: (r, d) => choices.applyChoice(r as Choice, d),
  },
  {
    name: 'section_icons',
    options: () => ({}),
    set: (r) => (choices.sectionIcons = r as SectionIcon[]),
    apply: (r, d) => choices.applySectionIcon(r as SectionIcon, d),
  },
  {
    name: 'staples',
    options: () => ({}),
    set: (r) => (staples.staples = r as Staple[]),
    apply: (r, d) => staples.apply(r as Staple, d),
  },
  {
    name: 'trips',
    options: () => ({ filter: trips.filter }),
    set: (r) => (trips.trips = r as Trip[]),
    apply: (r, d) => trips.apply(r as Trip, d),
  },
];

/** How often to try the server again while it can't be reached. */
const RETRY_MS = 30_000;

// Keeps the stores in lib/stores live: loads them from the copy on the phone,
// then from the server, listens for changes made on other phones, and writes
// the copy back as they change. The stores hold the state and the changes;
// this is only how they stay in sync.
class Shop {
  private stops: UnsubscribeFunc[] = [];
  private copy: DeviceCopy | null = null;
  private snap: Snapshot = emptySnapshot();
  private connecting: Promise<void> | null = null;
  private retry: ReturnType<typeof setInterval> | undefined;
  private persistTimer: ReturnType<typeof setTimeout> | undefined;
  private unwatch: (() => void) | undefined;
  private onOnline = () => this.connect();
  private onVisible = () => document.visibilityState === 'visible' && this.connect();

  /** Opens the copy on this phone, then loads from the server and listens for changes. */
  async start() {
    // a tapped notification opens its list
    const asked = new URLSearchParams(location.search).get('list');
    if (asked) {
      household.choose(asked);
      history.replaceState(null, '', location.pathname + location.hash);
    }
    this.copy = new DeviceCopy(session.user?.id ?? '');
    this.snap = this.copy.snapshot();
    items.attach(this.copy);
    const s = this.snap;
    household.lists = s.lists;
    household.people = s.people;
    catalogue.generics = s.generics;
    catalogue.sections = s.sections;
    catalogue.products = s.products;
    pins.pins = s.pins;
    choices.choices = s.choices;
    choices.sectionIcons = s.sectionIcons;
    items.setBase(s.items[household.list?.id ?? ''] ?? []);
    if (s.lists.length) status.loading = false;

    this.unwatch = $effect.root(() => {
      $effect(() => this.persist(this.current()));
    });
    addEventListener('online', this.onOnline);
    document.addEventListener('visibilitychange', this.onVisible);
    this.retry = setInterval(
      () => (status.offline || items.outbox.length) && this.connect(),
      RETRY_MS,
    );
    await this.connect();
  }

  stop() {
    removeEventListener('online', this.onOnline);
    document.removeEventListener('visibilitychange', this.onVisible);
    clearInterval(this.retry);
    clearTimeout(this.persistTimer);
    this.unwatch?.();
    this.stops.splice(0).forEach((stop) => stop());
  }

  /** Loads everything from the server, starts listening once, and sends what waited. */
  connect(): Promise<void> {
    this.connecting ??= this.load().finally(() => (this.connecting = null));
    return this.connecting;
  }

  private async load() {
    try {
      const [people, sections] = await Promise.all([
        pb.collection<User>('users').getFullList({ sort: 'name' }),
        pb.send<Section[]>('/api/app/sections', {}),
        ...synced.map(async (c) =>
          c.set(await pb.collection(c.name).getFullList({ batch: 1000, ...c.options() })),
        ),
      ]);
      household.people = people as User[];
      catalogue.sections = sections as Section[];
      await this.loadItems();
      if (!this.stops.length) await this.listen();
      status.offline = false;
      if (status.error === errorMessage({ status: 0 })) status.error = '';
      await items.flush();
    } catch (err) {
      if (unreachable(err)) status.offline = true;
      else status.error = errorMessage(err);
    } finally {
      status.loading = false;
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
      ...(await Promise.all(
        synced.map((c) =>
          pb
            .collection(c.name)
            .subscribe('*', ({ action, record }) => c.apply(record, action === 'delete'), {
              expand: c.options().expand,
            }),
        ),
      )),
      await pb.collection<List>('lists').subscribe('*', ({ action, record }) => {
        if (action === 'delete' && record.id === household.list?.id) this.loadItems();
      }),
      await pb
        .collection<Item>('items')
        .subscribe('*', ({ action, record }) => items.apply(record, action === 'delete'), {
          expand: 'product',
        }),
    );
  }

  /** Everything the copy on the phone keeps, as it is now. */
  private current(): Snapshot {
    const list = household.list?.id;
    return {
      lists: household.lists,
      people: household.people,
      generics: catalogue.generics,
      sections: catalogue.sections,
      pins: pins.pins,
      choices: choices.choices,
      sectionIcons: choices.sectionIcons,
      products: catalogue.products,
      items: { ...this.snap.items, ...(list ? { [list]: items.base } : {}) },
    };
  }

  /** Writes the copy on the phone, a moment after the last change. */
  private persist(next: Snapshot) {
    clearTimeout(this.persistTimer);
    this.persistTimer = setTimeout(() => {
      this.snap = $state.snapshot(next) as Snapshot;
      this.copy?.saveSnapshot(this.snap);
    }, 300);
  }

  /** Loads the open list's items, or shows the copy on the phone when offline. */
  async loadItems() {
    const id = household.list?.id;
    if (!id) {
      items.setBase([]);
      return;
    }
    try {
      items.setBase(
        catalogue.keep(
          await pb
            .collection<Item>('items')
            .getFullList({ filter: pb.filter('list = {:id}', { id }), expand: 'product' }),
        ),
      );
    } catch (err) {
      if (!unreachable(err)) throw err;
      status.offline = true;
      items.setBase(this.snap.items[id] ?? []);
    }
  }

  async open(listId: string) {
    household.choose(listId);
    await run(() => this.loadItems());
  }

  /** Makes a new list, empty or with an old one's items, and opens it. */
  async createList(name: string, kind: Kind, from?: List) {
    await run(async () => {
      const list = from ? await household.copy(from, name) : await household.create(name, kind);
      await this.open(list.id);
    });
  }

  async deleteList(list: List) {
    await run(async () => {
      await household.delete(list);
      await this.open(household.ordered[0]?.id ?? '');
    });
  }
}

export const shop = new Shop();
