import type { UnsubscribeFunc } from 'pocketbase';
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
import { run, status } from './stores/status.svelte';

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
      const [lists, people, generics, sections, pinned, picked, sectionIcons] = await Promise.all([
        pb.collection<List>('lists').getFullList(),
        pb.collection<User>('users').getFullList({ sort: 'name' }),
        pb.collection<Generic>('generics').getFullList({ batch: 1000 }),
        pb.send<Section[]>('/api/app/sections', {}),
        pb.collection<Pin>('pins').getFullList({ expand: 'product' }),
        pb.collection<Choice>('item_choices').getFullList({ batch: 1000 }),
        pb.collection<SectionIcon>('section_icons').getFullList(),
      ]);
      pins.pins = pinned;
      choices.choices = picked;
      choices.sectionIcons = sectionIcons;
      household.lists = lists;
      household.people = people;
      catalogue.generics = generics;
      catalogue.sections = sections;
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
      await pb.collection<List>('lists').subscribe('*', ({ action, record }) => {
        household.apply(record, action === 'delete');
        if (action === 'delete' && record.id === household.list?.id) this.loadItems();
      }),
      await pb
        .collection<Pin>('pins')
        .subscribe('*', ({ action, record }) => pins.apply(record, action === 'delete'), {
          expand: 'product',
        }),
      await pb
        .collection<Choice>('item_choices')
        .subscribe('*', ({ action, record }) => choices.applyChoice(record, action === 'delete')),
      await pb
        .collection<SectionIcon>('section_icons')
        .subscribe('*', ({ action, record }) =>
          choices.applySectionIcon(record, action === 'delete'),
        ),
      await pb
        .collection<Generic>('generics')
        .subscribe('*', ({ action, record }) =>
          catalogue.applyGeneric(record, action === 'delete'),
        ),
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

  async createList(name: string, kind: Kind) {
    await run(async () => {
      const list = await household.create(name, kind);
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
