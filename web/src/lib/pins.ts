import type { RecordModel } from 'pocketbase';
import type { Generic } from './catalogue';
import type { Product } from './products';

export interface Pin extends RecordModel {
  generic: string;
  product: string;
  uses: number;
  last_used: string;
  expand?: { product?: Product };
}

/** The products pinned for a generic item, the one bought most first. */
export function pinnedFor(pins: Pin[], generic: string): Product[] {
  return pins
    .filter((p) => p.generic === generic && p.expand?.product)
    .sort((a, b) => b.uses - a.uses || b.last_used.localeCompare(a.last_used))
    .map((p) => p.expand!.product!);
}

/**
 * The household's usual products for what is being suggested: for each
 * suggested generic, its pinned products (at most `each`), without repeats.
 */
export function usualFor(pins: Pin[], generics: Generic[], each = 2): Product[] {
  const seen = new Set<string>();
  const out: Product[] = [];
  for (const g of generics) {
    for (const p of pinnedFor(pins, g.id).slice(0, each)) {
      if (!seen.has(p.id)) {
        seen.add(p.id);
        out.push(p);
      }
    }
  }
  return out;
}
