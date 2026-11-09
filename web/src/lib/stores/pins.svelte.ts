import { upsert, type Item } from '../list';
import { pb } from '../pb';
import { pinnedFor, type Pin } from '../pins';
import type { Product } from '../products';
import { catalogue } from './catalogue.svelte';
import { items } from './items.svelte';
import { run } from './status.svelte';

// The household's choice of exact product for a generic item: "milk" is the
// Boni one, here.
class Pins {
  pins = $state<Pin[]>([]);

  apply(pin: Pin, deleted = false) {
    this.pins = upsert(this.pins, pin, deleted);
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
    catalogue.keepProduct(product);
    await items.flush();
    await run(async () => {
      const kept = await pb.send<Item>(`/api/app/items/${item.id}/pin`, {
        method: 'POST',
        body: { product: product.id },
      });
      const base = kept.id === item.id ? items.base : upsert(items.base, item, true);
      items.setBase(upsert(base, kept));
    });
  }

  async unpin(generic: string, product: Product) {
    const pin = this.pins.find((p) => p.generic === generic && p.product === product.id);
    if (!pin) return;
    const before = this.pins;
    this.apply(pin, true);
    await run(
      () => pb.collection('pins').delete(pin.id),
      () => (this.pins = before),
    );
  }
}

export const pins = new Pins();
