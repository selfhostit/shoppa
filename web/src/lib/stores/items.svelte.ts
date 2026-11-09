import { exactly, genericName, type Generic } from '../catalogue';
import { parseEntry, upsert, type Item } from '../list';
import {
  enqueue,
  newId,
  pbNow,
  renamed,
  sameThing,
  unreachable,
  withPending,
  type DeviceCopy,
  type NewItem,
  type Op,
} from '../offline';
import { errorMessage, pb } from '../pb';
import { productName, type Product } from '../products';
import { session } from '../session.svelte';
import { catalogue } from './catalogue.svelte';
import { household } from './household.svelte';
import { run, status } from './status.svelte';

// The open list's items. Every change to an item goes through a queue that is
// shown at once and sent in order, waiting while the server can't be reached
// (lib/offline.ts).
class Items {
  /** the open list's items, with the changes still waiting laid over */
  items = $state<Item[]>([]);
  /** the open list's items as the server last sent them */
  base = $state.raw<Item[]>([]);
  /** changes to items not sent yet, oldest first */
  outbox = $state<Op[]>([]);

  private copy: DeviceCopy | null = null;
  private sending: Promise<void> | null = null;
  private inFlight: Op | null = null;

  /** Takes up the changes that waited on this phone. */
  attach(copy: DeviceCopy) {
    this.copy = copy;
    this.outbox = copy.outbox();
  }

  /** The open list's items as the server sent them. */
  setBase(items: Item[]) {
    this.base = items;
    this.show();
  }

  /** Keeps an item as the server sent it, or drops it, when it is on the open list. */
  apply(record: Item, deleted = false) {
    if (record.list !== household.list?.id) return;
    catalogue.keep([record]);
    this.setBase(upsert(this.base, record, deleted));
  }

  private show() {
    const list = household.list?.id ?? '';
    this.items = withPending(
      this.base,
      this.outbox,
      list,
      session.user?.id ?? '',
      catalogue.generics,
    );
  }

  private saveOutbox() {
    this.copy?.saveOutbox($state.snapshot(this.outbox) as Op[]);
  }

  /** Queues a change to an item, shows it at once, and sends it if it can. */
  private async change(op: Op) {
    // the change being sent can't take more in
    const [head, ...rest] = this.outbox;
    this.outbox =
      head && head === this.inFlight ? [head, ...enqueue(rest, op)] : enqueue(this.outbox, op);
    this.saveOutbox();
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
            status.offline = true;
            break;
          }
          // gone already (someone else removed it), or refused: drop it
          if ((err as { status?: number }).status !== 404) status.error = errorMessage(err);
        } finally {
          this.inFlight = null;
        }
        this.outbox = renamed(this.outbox.slice(1), op.id, id);
        this.saveOutbox();
        this.show();
      }
    })().finally(() => (this.sending = null));
    return this.sending;
  }

  /** Sends one change; returns the item's id on the server. */
  private async send(op: Op): Promise<string> {
    const here = op.list === household.list?.id;
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
      if (here) this.base = upsert(this.base, catalogue.keep([item])[0]);
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
    const list = household.list;
    const lang = catalogue.lang;
    if (!product) generic ??= exactly(catalogue.kindGenerics, name, lang);
    if (!(name || generic || product) || !list) return;
    if (product) catalogue.keepProduct(product);
    const body: NewItem = product
      ? { product: product.id }
      : generic
        ? { generic: generic.id }
        : { name };
    if (quantity) body.quantity = quantity;
    const shown = product
      ? productName(product, lang)
      : generic
        ? genericName(generic, lang)
        : name;
    body.name ??= shown;
    // what's on the list already comes back, as the server would do
    const there = this.items.find((i) => sameThing(i, body, catalogue.generics));
    await this.change({
      kind: 'add',
      id: there?.id ?? newId(),
      list: list.id,
      body,
      at: pbNow(),
      ...(there ? { existing: true } : {}),
    });
  }

  /** Puts several typed or pasted items on the open list, in their order. */
  async addMany(texts: string[]) {
    for (const text of texts) await this.add(text);
  }

  /** Puts a recently bought item back on the list. */
  async putBack(item: Item) {
    await this.save(item, { checked: false });
  }

  async tick(item: Item) {
    // the server keeps who did it, and when, even when that was offline
    await this.save(item, { checked: true, checked_at: pbNow() });
  }

  /** "I'll get this one": or, when it's yours already, lets it go. */
  async claim(item: Item) {
    const me = session.user?.id ?? '';
    await this.save(item, { claimed_by: item.claimed_by === me ? '' : me });
  }

  /** Changes an item: shown at once, sent when the server can be reached. */
  async save(item: Item, change: Partial<Item>) {
    await this.change({ kind: 'update', id: item.id, list: item.list, change, at: pbNow() });
  }

  async remove(item: Item) {
    await this.change({ kind: 'delete', id: item.id, list: item.list, at: pbNow() });
  }

  /** Keeps a typed item as one of the household's own, with an icon and a section. */
  async remember(item: Item, icon: string, section: string) {
    await this.flush();
    await run(async () => {
      const generic = await catalogue.addGeneric(item.name, icon, section);
      this.setBase(
        upsert(
          this.base,
          await pb.collection<Item>('items').update(item.id, { generic: generic.id }),
        ),
      );
    });
  }
}

export const items = new Items();
